// Command pade-provider-github is the in-tree GitHub App reference provider skeleton.
//
// Non-normative: not part of the PADE standard. PADE core must not grow GitHub-specific fields.
//
// Fake mode (PADE_PROVIDER_FAKE=1) returns a derived-looking installation token for dogfood
// of the exec contract. Real GitHub App JWT → installation token exchange is Milestone D/E.
package main

import (
	"encoding/json"
	"fmt"
	"io"
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

	if os.Getenv("PADE_PROVIDER_FAKE") != "1" {
		fmt.Fprintln(os.Stderr, "real GitHub App derivation is not implemented yet; set PADE_PROVIDER_FAKE=1 for contract dogfood")
		os.Exit(3)
	}

	tokenEnv := stringFrom(req.Config["tokenEnv"], "GITHUB_TOKEN")
	// Fake installation-token-shaped value (not a real secret).
	value := stringFrom(req.Config["value"], "ghs_pade_fake_installation_token")

	switch req.Operation {
	case "probe":
		write(map[string]interface{}{
			"status":  "available",
			"message": "github reference provider (fake mode)",
			"meta": map[string]string{
				"capability": req.Capability,
				"mode":       "fake",
			},
		})
	case "resolve":
		write(map[string]interface{}{
			"env": map[string]string{
				tokenEnv: value,
			},
			"expiresAt": time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
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

func stringFrom(v interface{}, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}
