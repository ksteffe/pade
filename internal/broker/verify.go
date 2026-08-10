package broker

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const defaultSkew = 30 * time.Second

// Verifier validates Cursor (or test) OIDC JWTs against JWKS.
type Verifier struct {
	Issuer   string
	Audience string
	JWKSURL  string
	HTTPDo   func(*http.Request) (*http.Response, error)
	Skew     time.Duration
	Now      func() time.Time

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

type jwksDoc struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type cursorClaims struct {
	jwt.RegisteredClaims
	CloudAgentID string   `json:"cloud_agent_id"`
	AgentRuntime string   `json:"agent_runtime"`
	RepoURL      string   `json:"repo_url"`
	RepoURLs     []string `json:"repo_urls"`
	RepoCount    int      `json:"repo_count"`
	OwnerUserID  string   `json:"owner_user_id"`
	TeamID       string   `json:"team_id"`
}

// Verify parses and validates a bearer JWT. It never logs the token.
func (v *Verifier) Verify(ctx context.Context, rawToken string) (Claims, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Claims{}, fmt.Errorf("missing bearer token")
	}
	if err := v.ensureKeys(ctx); err != nil {
		return Claims{}, err
	}

	skew := v.Skew
	if skew == 0 {
		skew = defaultSkew
	}
	nowFn := v.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	parsed, err := jwt.ParseWithClaims(rawToken, &cursorClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		kid, _ := t.Header["kid"].(string)
		v.mu.Lock()
		key := v.keys[kid]
		v.mu.Unlock()
		if key == nil {
			return nil, fmt.Errorf("unknown signing key")
		}
		return key, nil
	}, jwt.WithIssuer(v.Issuer), jwt.WithAudience(v.Audience), jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithTimeFunc(nowFn), jwt.WithLeeway(skew))
	if err != nil {
		// Do not wrap with token material.
		return Claims{}, fmt.Errorf("token verification failed: %w", err)
	}
	cc, ok := parsed.Claims.(*cursorClaims)
	if !ok || !parsed.Valid {
		return Claims{}, fmt.Errorf("token verification failed")
	}
	sub, err := cc.GetSubject()
	if err != nil || sub == "" {
		return Claims{}, fmt.Errorf("token subject missing")
	}
	return Claims{
		Subject:      sub,
		Audience:     v.Audience,
		Issuer:       v.Issuer,
		CloudAgentID: cc.CloudAgentID,
		AgentRuntime: cc.AgentRuntime,
		RepoURL:      cc.RepoURL,
		RepoURLs:     append([]string(nil), cc.RepoURLs...),
		RepoCount:    cc.RepoCount,
		OwnerUserID:  cc.OwnerUserID,
		TeamID:       cc.TeamID,
		JTI:          cc.ID,
	}, nil
}

func (v *Verifier) ensureKeys(ctx context.Context) error {
	v.mu.Lock()
	if len(v.keys) > 0 && time.Since(v.fetchedAt) < 5*time.Minute {
		v.mu.Unlock()
		return nil
	}
	v.mu.Unlock()
	return v.refreshKeys(ctx)
}

func (v *Verifier) refreshKeys(ctx context.Context) error {
	url := strings.TrimSpace(v.JWKSURL)
	if url == "" {
		return fmt.Errorf("jwks URL is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("jwks request failed")
	}
	do := v.HTTPDo
	if do == nil {
		do = http.DefaultClient.Do
	}
	resp, err := do(req)
	if err != nil {
		return fmt.Errorf("jwks fetch failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks fetch http %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("jwks read failed")
	}
	var doc jwksDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("jwks parse failed")
	}
	keys := map[string]*rsa.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || k.Kid == "" || k.N == "" || k.E == "" {
			continue
		}
		pub, err := rsaPublicKeyFromJWK(k)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return fmt.Errorf("jwks contained no usable RSA keys")
	}
	v.mu.Lock()
	v.keys = keys
	v.fetchedAt = time.Now()
	v.mu.Unlock()
	return nil
}

func rsaPublicKeyFromJWK(k jwkKey) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		nb, err = base64.URLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, err
		}
	}
	eb, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		eb, err = base64.URLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, err
		}
	}
	var eInt int
	for _, b := range eb {
		eInt = eInt<<8 + int(b)
	}
	if eInt == 0 {
		return nil, fmt.Errorf("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: eInt}, nil
}
