// Command pade-provider-google-analytics is the in-tree GA reference provider.
//
// Non-normative: not part of the PADE standard. PADE core must not grow Google-specific fields.
//
// Modes:
//   - PADE_PROVIDER_FAKE=1: returns a fake access-token-shaped value (CI/contract dogfood)
//   - real: service account (broker-side) → short-lived OAuth2 access token
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type request struct {
	Capability string                 `json:"capability"`
	Operation  string                 `json:"operation"`
	Config     map[string]interface{} `json:"config"`
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fail("read stdin: %v", err)
	}
	var req request
	if err := json.Unmarshal(data, &req); err != nil {
		fail("invalid request JSON")
	}

	cfg := configFromMap(req.Config)
	fake := os.Getenv("PADE_PROVIDER_FAKE") == "1"

	switch req.Operation {
	case "probe":
		if fake {
			write(map[string]interface{}{
				"status":  "available",
				"message": "google-analytics reference provider (fake mode)",
				"meta": map[string]string{
					"capability": req.Capability,
					"mode":       "fake",
				},
			})
			return
		}
		if err := cfg.validate(); err != nil {
			write(map[string]interface{}{
				"status":  "unavailable",
				"message": err.Error(),
				"meta": map[string]string{
					"capability": req.Capability,
					"mode":       "google-sa",
				},
			})
			return
		}
		write(map[string]interface{}{
			"status":  "available",
			"message": "google-analytics reference provider (service account access token)",
			"meta": map[string]string{
				"capability": req.Capability,
				"mode":       "google-sa",
				// Safe metadata only — never private key material.
				"tokenURL": cfg.TokenURL,
				"scope":    cfg.Scope,
			},
		})
	case "resolve":
		tokenEnv := cfg.TokenEnv
		if tokenEnv == "" {
			tokenEnv = defaultTokenEnv
		}
		propEnv := cfg.PropertyEnv
		if propEnv == "" {
			propEnv = defaultPropEnv
		}
		if fake {
			value := stringFrom(req.Config["value"], "ya29.pade_fake_access_token")
			env := map[string]string{tokenEnv: value}
			if cfg.PropertyID != "" {
				env[propEnv] = cfg.PropertyID
			} else {
				env[propEnv] = stringFrom(req.Config["propertyId"], "properties/000000000")
			}
			write(map[string]interface{}{
				"env":       env,
				"expiresAt": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			})
			return
		}
		tok, err := deriveAccessToken(http.DefaultClient, cfg)
		if err != nil {
			fail("%v", err)
		}
		env := map[string]string{tokenEnv: tok.Token}
		if cfg.PropertyID != "" {
			env[propEnv] = cfg.PropertyID
		}
		write(map[string]interface{}{
			"env":       env,
			"expiresAt": tok.ExpiresAt.Format(time.RFC3339),
		})
	default:
		fail("unsupported operation %q", req.Operation)
	}
}

func write(v interface{}) {
	if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
		fail("encode response: %v", err)
	}
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
