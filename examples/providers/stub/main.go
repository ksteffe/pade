// Command pade-provider-stub exercises the draft exec provider contract.
// Non-normative dogfood only — not part of the PADE standard.
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

	if truthy(req.Config["fail"]) {
		fmt.Fprintln(os.Stderr, "stub provider refused request")
		os.Exit(2)
	}

	tokenEnv := stringFrom(req.Config["tokenEnv"], "DEMO_TOKEN")
	value := stringFrom(req.Config["value"], "stub-derived-token")

	switch req.Operation {
	case "probe":
		write(map[string]interface{}{
			"status":  "available",
			"message": "stub provider ready",
			"meta": map[string]string{
				"capability": req.Capability,
				"mode":       "stub",
			},
		})
	case "resolve":
		write(map[string]interface{}{
			"env": map[string]string{
				tokenEnv: value,
			},
			"expiresAt": time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
		})
	default:
		fail("unsupported operation %q", req.Operation)
	}
}

func write(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(v); err != nil {
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

func truthy(v interface{}) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1" || t == "yes"
	default:
		return false
	}
}
