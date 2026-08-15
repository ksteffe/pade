package broker_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ksteffe/pade/internal/broker"
)

func TestVerifyRequiresExpiration(t *testing.T) {
	key := mustKey(t)
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jwksFor(key, "test-kid"))
	}))
	t.Cleanup(jwks.Close)

	v := &broker.Verifier{
		Issuer:   testIssuer,
		Audience: testAudience,
		JWKSURL:  jwks.URL,
		HTTPDo:   jwks.Client().Do,
	}
	ctx := context.Background()

	valid := mustSign(t, key, "test-kid", jwt.MapClaims{
		"iss": testIssuer, "sub": testSubject, "aud": testAudience,
		"iat": time.Now().Unix(), "exp": time.Now().Add(2 * time.Minute).Unix(),
	})
	if _, err := v.Verify(ctx, valid); err != nil {
		t.Fatalf("valid token: %v", err)
	}

	noExp := mustSign(t, key, "test-kid", jwt.MapClaims{
		"iss": testIssuer, "sub": testSubject, "aud": testAudience,
		"iat": time.Now().Unix(),
	})
	if _, err := v.Verify(ctx, noExp); err == nil {
		t.Fatal("expected missing exp to fail")
	}

	expired := mustSign(t, key, "test-kid", jwt.MapClaims{
		"iss": testIssuer, "sub": testSubject, "aud": testAudience,
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-time.Hour).Unix(),
	})
	if _, err := v.Verify(ctx, expired); err == nil {
		t.Fatal("expected expired token to fail")
	}

	tooLong := mustSign(t, key, "test-kid", jwt.MapClaims{
		"iss": testIssuer, "sub": testSubject, "aud": testAudience,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(48 * time.Hour).Unix(),
	})
	if _, err := v.Verify(ctx, tooLong); err == nil {
		t.Fatal("expected max lifetime rejection")
	}

	wrongIss := mustSign(t, key, "test-kid", jwt.MapClaims{
		"iss": "https://evil.example", "sub": testSubject, "aud": testAudience,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
	})
	if _, err := v.Verify(ctx, wrongIss); err == nil {
		t.Fatal("expected wrong issuer rejection")
	}

	wrongAud := mustSign(t, key, "test-kid", jwt.MapClaims{
		"iss": testIssuer, "sub": testSubject, "aud": "https://other.example",
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
	})
	if _, err := v.Verify(ctx, wrongAud); err == nil {
		t.Fatal("expected wrong audience rejection")
	}

	hs := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": testIssuer, "sub": testSubject, "aud": testAudience,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
	})
	hs.Header["kid"] = "test-kid"
	hsTok, err := hs.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(ctx, hsTok); err == nil {
		t.Fatal("expected unexpected algorithm rejection")
	}
}

func TestJWKSRejectsOversized(t *testing.T) {
	key := mustKey(t)
	pub := &key.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	base, _ := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{
			{"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig", "n": n, "e": e},
		},
	})
	// Pad past 1 MiB while remaining syntactically closer to JWKS (trailing garbage is fine; size check is first).
	oversized := append(bytes.Repeat([]byte(" "), (1<<20)+16), base...)
	v := &broker.Verifier{
		Issuer:   testIssuer,
		Audience: testAudience,
		JWKSURL:  "http://127.0.0.1/keys",
		HTTPDo: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(oversized)),
				Header:     make(http.Header),
			}, nil
		},
	}
	_, err := v.Verify(context.Background(), "x.y.z")
	if err == nil || !strings.Contains(err.Error(), "exceeds size limit") {
		t.Fatalf("expected size limit error, got %v", err)
	}
}

func TestJWKSRejectsDuplicateKid(t *testing.T) {
	key := mustKey(t)
	pub := &key.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	raw, _ := json.Marshal(map[string]interface{}{
		"keys": []map[string]string{
			{"kty": "RSA", "kid": "dup", "alg": "RS256", "use": "sig", "n": n, "e": e},
			{"kty": "RSA", "kid": "dup", "alg": "RS256", "use": "sig", "n": n, "e": e},
		},
	})
	v := &broker.Verifier{
		Issuer:   testIssuer,
		Audience: testAudience,
		JWKSURL:  "http://127.0.0.1/keys",
		HTTPDo: func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(string(raw))),
				Header:     make(http.Header),
			}, nil
		},
	}
	_, err := v.Verify(context.Background(), "x.y.z")
	if err == nil || !strings.Contains(err.Error(), "duplicate kid") {
		t.Fatalf("expected duplicate kid error, got %v", err)
	}
}

func TestJWKSRejectsRemoteHTTP(t *testing.T) {
	v := &broker.Verifier{
		Issuer:   testIssuer,
		Audience: testAudience,
		JWKSURL:  "http://evil.example/keys",
	}
	_, err := v.Verify(context.Background(), "token")
	if err == nil || !strings.Contains(err.Error(), "insecure http") {
		t.Fatalf("expected insecure http rejection, got %v", err)
	}
}

func TestJWKSRejectsBadAlgAndUse(t *testing.T) {
	key := mustKey(t)
	pub := &key.PublicKey
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())

	for _, tc := range []struct {
		name string
		key  map[string]string
		want string
	}{
		{"bad alg", map[string]string{"kty": "RSA", "kid": "k1", "alg": "HS256", "use": "sig", "n": n, "e": e}, "unsupported alg"},
		{"bad use", map[string]string{"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "enc", "n": n, "e": e}, "unsupported use"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]interface{}{"keys": []map[string]string{tc.key}})
			v := &broker.Verifier{
				Issuer: testIssuer, Audience: testAudience, JWKSURL: "http://127.0.0.1/keys",
				HTTPDo: func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(string(raw))),
						Header:     make(http.Header),
					}, nil
				},
			}
			_, err := v.Verify(context.Background(), "x.y.z")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %q in error, got %v", tc.want, err)
			}
		})
	}
}
