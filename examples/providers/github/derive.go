package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// opaqueConfig holds GitHub App settings. These keys are provider-local only.
type opaqueConfig struct {
	AppID          string
	InstallationID string
	PrivateKeyPath string
	PrivateKeyPEM  string
	APIURL         string
	TokenEnv       string
	Repositories   []string
	Permissions    map[string]string
}

func configFromMap(m map[string]interface{}) opaqueConfig {
	if m == nil {
		m = map[string]interface{}{}
	}
	cfg := opaqueConfig{
		AppID:          firstNonEmpty(stringFrom(m["appId"], ""), os.Getenv("GITHUB_APP_ID")),
		InstallationID: firstNonEmpty(stringFrom(m["installationId"], ""), os.Getenv("GITHUB_APP_INSTALLATION_ID")),
		PrivateKeyPath: firstNonEmpty(stringFrom(m["privateKeyPath"], ""), os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH")),
		PrivateKeyPEM:  firstNonEmpty(stringFrom(m["privateKey"], ""), os.Getenv("GITHUB_APP_PRIVATE_KEY")),
		APIURL:         firstNonEmpty(stringFrom(m["apiURL"], ""), os.Getenv("GITHUB_API_URL"), "https://api.github.com"),
		TokenEnv:       stringFrom(m["tokenEnv"], "GITHUB_TOKEN"),
		Permissions:    stringMapFrom(m["permissions"]),
		Repositories:   stringSliceFrom(m["repositories"]),
	}
	return cfg
}

func (c opaqueConfig) validate() error {
	if strings.TrimSpace(c.AppID) == "" {
		return fmt.Errorf("appId is required (config.appId or GITHUB_APP_ID)")
	}
	if strings.TrimSpace(c.InstallationID) == "" {
		return fmt.Errorf("installationId is required (config.installationId or GITHUB_APP_INSTALLATION_ID)")
	}
	if strings.TrimSpace(c.PrivateKeyPath) == "" && strings.TrimSpace(c.PrivateKeyPEM) == "" {
		return fmt.Errorf("private key is required (config.privateKeyPath, GITHUB_APP_PRIVATE_KEY_PATH, config.privateKey, or GITHUB_APP_PRIVATE_KEY)")
	}
	return nil
}

type installationToken struct {
	Token     string
	ExpiresAt time.Time
}

// deriveInstallationToken mints a short-lived GitHub App installation token.
// The durable private key never appears in the returned Material.
func deriveInstallationToken(client *http.Client, cfg opaqueConfig) (*installationToken, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if client == nil {
		client = http.DefaultClient
	}
	pemBytes, err := loadPrivateKeyPEM(cfg)
	if err != nil {
		return nil, err
	}
	key, err := parseRSAPrivateKey(pemBytes)
	if err != nil {
		return nil, err
	}
	jwtStr, err := mintAppJWT(cfg.AppID, key, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return createInstallationAccessToken(client, cfg, jwtStr)
}

func loadPrivateKeyPEM(cfg opaqueConfig) ([]byte, error) {
	// Prefer path when set (broker-side file mount); inline PEM is a fallback.
	if strings.TrimSpace(cfg.PrivateKeyPath) != "" {
		b, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key file: %w", err)
		}
		return b, nil
	}
	if strings.TrimSpace(cfg.PrivateKeyPEM) != "" {
		return []byte(cfg.PrivateKeyPEM), nil
	}
	return nil, fmt.Errorf("private key is required")
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

func mintAppJWT(appID string, key *rsa.PrivateKey, now time.Time) (string, error) {
	// Clock skew cushion per GitHub guidance.
	iat := now.Add(-60 * time.Second)
	exp := iat.Add(9 * time.Minute)
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(iat),
		ExpiresAt: jwt.NewNumericDate(exp),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign app JWT: %w", err)
	}
	return signed, nil
}

func createInstallationAccessToken(client *http.Client, cfg opaqueConfig, appJWT string) (*installationToken, error) {
	url := strings.TrimRight(cfg.APIURL, "/") + "/app/installations/" + cfg.InstallationID + "/access_tokens"
	body := map[string]interface{}{}
	if names := repositoryNamesForAPI(cfg.Repositories); len(names) > 0 {
		body["repositories"] = names
	}
	if len(cfg.Permissions) > 0 {
		body["permissions"] = cfg.Permissions
	}
	var bodyReader io.Reader
	if len(body) > 0 {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "pade-provider-github")
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("installation token request failed")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read installation token response failed")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Do not include response body (may echo sensitive context).
		return nil, fmt.Errorf("installation token request returned HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode installation token response failed")
	}
	if strings.TrimSpace(parsed.Token) == "" {
		return nil, fmt.Errorf("installation token response missing token")
	}
	exp, err := time.Parse(time.RFC3339, parsed.ExpiresAt)
	if err != nil {
		// GitHub returns RFC3339; if absent, default to ~1h.
		exp = time.Now().UTC().Add(55 * time.Minute)
	}
	return &installationToken{Token: parsed.Token, ExpiresAt: exp.UTC()}, nil
}

func stringFrom(v interface{}, def string) string {
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) != "" {
			return t
		}
	case float64:
		// YAML/JSON numbers often decode as float64.
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case json.Number:
		return t.String()
	}
	return def
}

func stringSliceFrom(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s := stringFrom(item, "")
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func stringMapFrom(v interface{}) map[string]string {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		s := stringFrom(val, "")
		if s != "" {
			out[k] = s
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// repositoryNamesForAPI converts config values to GitHub installation-token
// repository names (not owner/name). Accepts either "repo" or "owner/repo".
func repositoryNamesForAPI(repos []string) []string {
	if len(repos) == 0 {
		return nil
	}
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if i := strings.LastIndex(r, "/"); i >= 0 && i+1 < len(r) {
			r = r[i+1:]
		}
		out = append(out, r)
	}
	return out
}
