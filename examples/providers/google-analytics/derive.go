package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultTokenURL = "https://oauth2.googleapis.com/token"
	defaultScope    = "https://www.googleapis.com/auth/analytics.readonly"
	defaultTokenEnv = "GA_ACCESS_TOKEN"
	defaultPropEnv  = "GA_PROPERTY_ID"
)

// opaqueConfig holds Google Analytics / Google auth settings. Provider-local only.
type opaqueConfig struct {
	ServiceAccountFile string
	ServiceAccountJSON string
	ClientEmail        string
	PrivateKeyPEM      string
	TokenURL           string
	Scope              string
	Subject            string // optional domain-wide delegation
	TokenEnv           string
	PropertyID         string
	PropertyEnv        string
}

type serviceAccountFile struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

func configFromMap(m map[string]interface{}) opaqueConfig {
	if m == nil {
		m = map[string]interface{}{}
	}
	cfg := opaqueConfig{
		ServiceAccountFile: firstNonEmpty(stringFrom(m["serviceAccountFile"], ""), os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")),
		ServiceAccountJSON: firstNonEmpty(stringFrom(m["serviceAccountJSON"], ""), os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON")),
		ClientEmail:        firstNonEmpty(stringFrom(m["clientEmail"], ""), os.Getenv("GOOGLE_CLIENT_EMAIL")),
		PrivateKeyPEM:      firstNonEmpty(stringFrom(m["privateKey"], ""), os.Getenv("GOOGLE_PRIVATE_KEY")),
		TokenURL:           firstNonEmpty(stringFrom(m["tokenURL"], ""), os.Getenv("GOOGLE_TOKEN_URL"), defaultTokenURL),
		Scope:              firstNonEmpty(stringFrom(m["scope"], ""), os.Getenv("GOOGLE_OAUTH_SCOPE"), defaultScope),
		Subject:            stringFrom(m["subject"], ""),
		TokenEnv:           stringFrom(m["tokenEnv"], defaultTokenEnv),
		PropertyID:         firstNonEmpty(stringFrom(m["propertyId"], ""), os.Getenv("GA_PROPERTY_ID")),
		PropertyEnv:        stringFrom(m["propertyEnv"], defaultPropEnv),
	}
	return cfg
}

func (c opaqueConfig) validate() error {
	if strings.TrimSpace(c.ServiceAccountFile) == "" &&
		strings.TrimSpace(c.ServiceAccountJSON) == "" &&
		(strings.TrimSpace(c.ClientEmail) == "" || strings.TrimSpace(c.PrivateKeyPEM) == "") {
		return fmt.Errorf("service account is required (serviceAccountFile / GOOGLE_APPLICATION_CREDENTIALS, serviceAccountJSON, or clientEmail+privateKey)")
	}
	return nil
}

type accessToken struct {
	Token     string
	ExpiresAt time.Time
}

func deriveAccessToken(client *http.Client, cfg opaqueConfig) (*accessToken, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	if strings.TrimSpace(cfg.Scope) == "" {
		cfg.Scope = defaultScope
	}
	if strings.TrimSpace(cfg.TokenURL) == "" {
		cfg.TokenURL = defaultTokenURL
	}
	email, pemBytes, tokenURL, err := loadServiceAccount(cfg)
	if err != nil {
		return nil, err
	}
	if tokenURL == "" {
		tokenURL = cfg.TokenURL
	}
	key, err := parseRSAPrivateKey(pemBytes)
	if err != nil {
		return nil, err
	}
	assertion, err := mintServiceAccountJWT(email, cfg.Scope, tokenURL, cfg.Subject, key, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return exchangeJWTForAccessToken(client, tokenURL, assertion)
}

func loadServiceAccount(cfg opaqueConfig) (clientEmail string, privateKeyPEM []byte, tokenURL string, err error) {
	// Prefer file path when set (broker-side mount).
	if strings.TrimSpace(cfg.ServiceAccountFile) != "" {
		raw, readErr := os.ReadFile(cfg.ServiceAccountFile)
		if readErr != nil {
			return "", nil, "", fmt.Errorf("read service account file: %w", readErr)
		}
		return parseServiceAccountJSON(raw, cfg)
	}
	if strings.TrimSpace(cfg.ServiceAccountJSON) != "" {
		return parseServiceAccountJSON([]byte(cfg.ServiceAccountJSON), cfg)
	}
	if strings.TrimSpace(cfg.ClientEmail) == "" || strings.TrimSpace(cfg.PrivateKeyPEM) == "" {
		return "", nil, "", fmt.Errorf("service account is required")
	}
	return cfg.ClientEmail, []byte(cfg.PrivateKeyPEM), cfg.TokenURL, nil
}

func parseServiceAccountJSON(raw []byte, cfg opaqueConfig) (string, []byte, string, error) {
	var sa serviceAccountFile
	if err := json.Unmarshal(raw, &sa); err != nil {
		return "", nil, "", fmt.Errorf("decode service account JSON failed")
	}
	email := firstNonEmpty(sa.ClientEmail, cfg.ClientEmail)
	pemStr := firstNonEmpty(sa.PrivateKey, cfg.PrivateKeyPEM)
	if email == "" || pemStr == "" {
		return "", nil, "", fmt.Errorf("service account JSON missing client_email or private_key")
	}
	tokenURL := firstNonEmpty(sa.TokenURI, cfg.TokenURL, defaultTokenURL)
	return email, []byte(pemStr), tokenURL, nil
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("private key PEM decode failed")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS1 private key: %w", err)
		}
		return key, nil
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
		}
		key, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type %q", block.Type)
	}
}

func mintServiceAccountJWT(email, scope, audience, subject string, key *rsa.PrivateKey, now time.Time) (string, error) {
	iat := now
	exp := iat.Add(time.Hour)
	claims := jwt.MapClaims{
		"iss":   email,
		"scope": scope,
		"aud":   audience,
		"iat":   iat.Unix(),
		"exp":   exp.Unix(),
	}
	if strings.TrimSpace(subject) != "" {
		claims["sub"] = subject
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign service account JWT: %w", err)
	}
	return signed, nil
}

func exchangeJWTForAccessToken(client *http.Client, tokenURL, assertion string) (*accessToken, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", assertion)
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "pade-provider-google-analytics")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read token exchange response failed")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Do not include response body (may contain sensitive context).
		return nil, fmt.Errorf("token exchange returned HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode token exchange response failed")
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return nil, fmt.Errorf("token exchange response missing access_token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	return &accessToken{
		Token:     parsed.AccessToken,
		ExpiresAt: time.Now().UTC().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func stringFrom(v interface{}, def string) string {
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) != "" {
			return t
		}
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case json.Number:
		return t.String()
	}
	return def
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
