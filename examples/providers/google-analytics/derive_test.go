package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestDeriveAccessToken(t *testing.T) {
	key, pemBytes := mustTestKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		vals, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if vals.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Fatalf("grant_type=%q", vals.Get("grant_type"))
		}
		assertion := vals.Get("assertion")
		parsed, err := jwt.Parse(assertion, func(token *jwt.Token) (interface{}, error) {
			return &key.PublicKey, nil
		})
		if err != nil || !parsed.Valid {
			t.Fatalf("jwt invalid: %v", err)
		}
		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok || claims["iss"] != "sa@example.iam.gserviceaccount.com" {
			t.Fatalf("claims=%v", claims)
		}
		if claims["scope"] != defaultScope {
			t.Fatalf("scope=%v", claims["scope"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya29.test_access_token","expires_in":3600,"token_type":"Bearer"}`))
	}))
	defer srv.Close()

	tok, err := deriveAccessToken(srv.Client(), opaqueConfig{
		ClientEmail:   "sa@example.iam.gserviceaccount.com",
		PrivateKeyPEM: string(pemBytes),
		TokenURL:      srv.URL,
		Scope:         defaultScope,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tok.Token != "ya29.test_access_token" {
		t.Fatalf("token=%q", tok.Token)
	}
	if tok.ExpiresAt.Before(time.Now().UTC().Add(30 * time.Minute)) {
		t.Fatalf("expiresAt=%s", tok.ExpiresAt)
	}
}

func TestDeriveReadsServiceAccountFile(t *testing.T) {
	_, pemBytes := mustTestKey(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sa.json")
	sa := map[string]string{
		"type":         "service_account",
		"client_email": "file-sa@example.iam.gserviceaccount.com",
		"private_key":  string(pemBytes),
		"token_uri":    "", // filled after server start
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"ya29.from_file","expires_in":1800,"token_type":"Bearer"}`))
	}))
	defer srv.Close()
	sa["token_uri"] = srv.URL
	raw, err := json.Marshal(sa)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	tok, err := deriveAccessToken(srv.Client(), opaqueConfig{
		ServiceAccountFile: path,
		TokenURL:           defaultTokenURL, // file token_uri should win
	})
	if err != nil {
		t.Fatal(err)
	}
	if tok.Token != "ya29.from_file" {
		t.Fatalf("token=%q", tok.Token)
	}
}

func TestConfigFromMapAndEnv(t *testing.T) {
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/tmp/sa.json")
	t.Setenv("GA_PROPERTY_ID", "properties/123")
	cfg := configFromMap(map[string]interface{}{
		"tokenEnv": "GA_ACCESS_TOKEN",
	})
	if cfg.ServiceAccountFile != "/tmp/sa.json" || cfg.PropertyID != "properties/123" {
		t.Fatalf("cfg=%+v", cfg)
	}
	if cfg.Scope != defaultScope || cfg.TokenURL != defaultTokenURL {
		t.Fatalf("defaults cfg=%+v", cfg)
	}
}

func TestValidateRequiresFields(t *testing.T) {
	if err := (opaqueConfig{}).validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestTokenExchangeHTTPErrorDoesNotLeakBody(t *testing.T) {
	_, pemBytes := mustTestKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","access_token":"ya29.should_not_leak"}`))
	}))
	defer srv.Close()

	_, err := deriveAccessToken(srv.Client(), opaqueConfig{
		ClientEmail:   "sa@example.iam.gserviceaccount.com",
		PrivateKeyPEM: string(pemBytes),
		TokenURL:      srv.URL,
		Scope:         defaultScope,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "ya29.should_not_leak") || strings.Contains(msg, "invalid_grant") {
		t.Fatalf("error leaked response body: %s", msg)
	}
	if !strings.Contains(msg, "HTTP 401") {
		t.Fatalf("error=%q", msg)
	}
}

func mustTestKey(t *testing.T) (*rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return key, pemBytes
}
