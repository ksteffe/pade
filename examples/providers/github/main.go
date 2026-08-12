// Command pade-provider-github is the in-tree GitHub App reference provider.
//
// Non-normative: not part of the PADE standard. PADE core must not grow GitHub-specific fields.
//
// Modes:
//   - PADE_PROVIDER_FAKE=1: returns a fake installation-token-shaped value (CI/contract dogfood)
//   - real: App private key (broker-side) → short-lived installation access token
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
				"message": "github reference provider (fake mode)",
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
					"mode":       "github-app",
				},
			})
			return
		}
		write(map[string]interface{}{
			"status":  "available",
			"message": "github reference provider (app installation token)",
			"meta": map[string]string{
				"capability": req.Capability,
				"mode":       "github-app",
				// Safe metadata only — never private key paths with secrets.
				"apiURL": cfg.APIURL,
			},
		})
	case "resolve":
		tokenEnv := cfg.TokenEnv
		if tokenEnv == "" {
			tokenEnv = "GITHUB_TOKEN"
		}
		if fake {
			value := stringFrom(req.Config["value"], "ghs_pade_fake_installation_token")
			write(map[string]interface{}{
				"env": map[string]string{
					tokenEnv: value,
				},
				"expiresAt": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			})
			return
		}
		tok, err := deriveInstallationToken(http.DefaultClient, cfg)
		if err != nil {
			fail("%v", err)
		}
		write(map[string]interface{}{
			"env": map[string]string{
				tokenEnv: tok.Token,
			},
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
