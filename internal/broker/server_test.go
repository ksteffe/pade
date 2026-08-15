package broker_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/ksteffe/pade/internal/binding"
	keepersm "github.com/ksteffe/pade/internal/binding/keepersm"
	"github.com/ksteffe/pade/internal/broker"
)

const (
	testIssuer   = "https://api.cursor.com"
	testAudience = "https://pade-broker.local"
	testSubject  = "user:42"
	testRepo     = "github.com/ksteffe/pade"
)

func TestPolicyAuthorize(t *testing.T) {
	t.Parallel()
	p, err := broker.ParsePolicy([]byte(`
version: "0.1"
oidc:
  issuer: https://api.cursor.com
  audience: https://pade-broker.local
policies:
  - subject: "user:42"
    requireRepoURLs: true
    repositories: ["github.com/ksteffe/pade"]
    capabilities: ["github.user.read"]
`))
	if err != nil {
		t.Fatal(err)
	}
	ok := p.Authorize(broker.Claims{
		Subject:  testSubject,
		RepoURLs: []string{testRepo},
	}, "github.user.read")
	if !ok.Allowed {
		t.Fatalf("%+v", ok)
	}
	if d := p.Authorize(broker.Claims{Subject: "user:99", RepoURLs: []string{testRepo}}, "github.user.read"); d.Allowed {
		t.Fatal("unauthorized subject")
	}
	if d := p.Authorize(broker.Claims{Subject: testSubject, RepoURLs: []string{testRepo}}, "prod.admin"); d.Allowed {
		t.Fatal("unauthorized capability")
	}
	if d := p.Authorize(broker.Claims{Subject: testSubject}, "github.user.read"); d.Allowed {
		t.Fatal("missing repo_urls must fail")
	}
	if d := p.Authorize(broker.Claims{
		Subject:  testSubject,
		RepoURLs: []string{testRepo, "github.com/other/repo"},
	}, "github.user.read"); d.Allowed {
		t.Fatal("multi-repo must fail singleton policy")
	}
}

func TestResolveHappyPathAndDenies(t *testing.T) {
	t.Setenv("PADE_KSM_FAKE", "1")
	key := mustKey(t)
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(jwksFor(key, "test-kid"))
	}))
	t.Cleanup(jwks.Close)

	policy, err := broker.ParsePolicy([]byte(`
version: "0.1"
oidc:
  issuer: https://api.cursor.com
  audience: https://pade-broker.local
  jwksURL: ` + jwks.URL + `
policies:
  - subject: "user:42"
    requireRepoURLs: true
    repositories: ["github.com/ksteffe/pade"]
    capabilities: ["github.user.read"]
`))
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  github.user.read:
    provider: keeper-secrets-manager
    keeperSecretsManager:
      refs:
        GITHUB_TOKEN: "keeper://pade-demo-github/field/password"
`), "broker-bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg := binding.NewRegistry(keepersm.New())
	srv := &broker.Server{
		Policy: policy,
		Verifier: &broker.Verifier{
			Issuer:   testIssuer,
			Audience: testAudience,
			JWKSURL:  jwks.URL,
			HTTPDo:   jwks.Client().Do,
		},
		Registry: reg,
		Bindings: bindings,
	}
	hs := httptest.NewServer(srv.Handler())
	t.Cleanup(hs.Close)

	tok := mustSign(t, key, "test-kid", jwt.MapClaims{
		"iss":            testIssuer,
		"sub":            testSubject,
		"aud":            testAudience,
		"iat":            time.Now().Unix(),
		"nbf":            time.Now().Add(-5 * time.Second).Unix(),
		"exp":            time.Now().Add(2 * time.Minute).Unix(),
		"jti":            "jti-1",
		"cloud_agent_id": "bc-test",
		"agent_runtime":  "managed",
		"repo_urls":      []string{testRepo},
		"repo_count":     1,
	})

	// Happy path
	resp := postResolve(t, hs.URL, tok, "github.user.read")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, b)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control=%q want no-store", cc)
	}
	var out struct {
		Env map[string]string `json:"env"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Env["GITHUB_TOKEN"] != "pade-demo-ksm-token" {
		t.Fatalf("env=%v", out.Env)
	}

	cases := []struct {
		name string
		tok  string
		cap  string
		code int
	}{
		{"missing bearer", "", "github.user.read", 401},
		{"wrong audience", mustSign(t, key, "test-kid", jwt.MapClaims{
			"iss": testIssuer, "sub": testSubject, "aud": "other",
			"iat": time.Now().Unix(), "nbf": time.Now().Unix() - 5, "exp": time.Now().Unix() + 60,
			"repo_urls": []string{testRepo},
		}), "github.user.read", 401},
		{"wrong issuer", mustSign(t, key, "test-kid", jwt.MapClaims{
			"iss": "https://evil.example", "sub": testSubject, "aud": testAudience,
			"iat": time.Now().Unix(), "nbf": time.Now().Unix() - 5, "exp": time.Now().Unix() + 60,
			"repo_urls": []string{testRepo},
		}), "github.user.read", 401},
		{"expired", mustSign(t, key, "test-kid", jwt.MapClaims{
			"iss": testIssuer, "sub": testSubject, "aud": testAudience,
			"iat": time.Now().Unix() - 400, "nbf": time.Now().Unix() - 400, "exp": time.Now().Unix() - 60,
			"repo_urls": []string{testRepo},
		}), "github.user.read", 401},
		{"not yet valid", mustSign(t, key, "test-kid", jwt.MapClaims{
			"iss": testIssuer, "sub": testSubject, "aud": testAudience,
			"iat": time.Now().Unix() + 120, "nbf": time.Now().Unix() + 120, "exp": time.Now().Unix() + 400,
			"repo_urls": []string{testRepo},
		}), "github.user.read", 401},
		{"unknown kid", mustSign(t, key, "other-kid", jwt.MapClaims{
			"iss": testIssuer, "sub": testSubject, "aud": testAudience,
			"iat": time.Now().Unix(), "nbf": time.Now().Unix() - 5, "exp": time.Now().Unix() + 60,
			"repo_urls": []string{testRepo},
		}), "github.user.read", 401},
		{"unauthorized subject", mustSign(t, key, "test-kid", jwt.MapClaims{
			"iss": testIssuer, "sub": "user:999", "aud": testAudience,
			"iat": time.Now().Unix(), "nbf": time.Now().Unix() - 5, "exp": time.Now().Unix() + 60,
			"repo_urls": []string{testRepo},
		}), "github.user.read", 403},
		{"unauthorized capability", tok, "prod.admin", 403},
		{"missing repo_urls", mustSign(t, key, "test-kid", jwt.MapClaims{
			"iss": testIssuer, "sub": testSubject, "aud": testAudience,
			"iat": time.Now().Unix(), "nbf": time.Now().Unix() - 5, "exp": time.Now().Unix() + 60,
			"repo_url": testRepo,
		}), "github.user.read", 403},
		{"multi repo", mustSign(t, key, "test-kid", jwt.MapClaims{
			"iss": testIssuer, "sub": testSubject, "aud": testAudience,
			"iat": time.Now().Unix(), "nbf": time.Now().Unix() - 5, "exp": time.Now().Unix() + 60,
			"repo_urls": []string{testRepo, "github.com/acme/other"}, "repo_count": 2,
		}), "github.user.read", 403},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := postResolve(t, hs.URL, tc.tok, tc.cap)
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			if r.StatusCode != tc.code {
				t.Fatalf("status=%d want=%d body=%s", r.StatusCode, tc.code, body)
			}
			if bytes.Contains(body, []byte(tok)) || bytes.Contains(body, []byte("pade-demo-ksm-token")) {
				t.Fatalf("response leaked secret/token: %s", body)
			}
			if strings.Contains(string(body), "eyJ") {
				t.Fatalf("response looks like JWT leak: %s", body)
			}
		})
	}
}

func postResolve(t *testing.T, base, token, capability string) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"capability": capability})
	req, err := http.NewRequest(http.MethodPost, base+"/v1/resolve", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func jwksFor(key *rsa.PrivateKey, kid string) map[string]interface{} {
	pub := &key.PublicKey
	return map[string]interface{}{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": kid,
			"alg": "RS256",
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	}
}

func mustSign(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
