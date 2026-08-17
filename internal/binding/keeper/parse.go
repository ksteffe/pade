package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func parseUID(ref string) (string, error) {
	if !strings.HasPrefix(ref, refPrefix) {
		return "", fmt.Errorf("keeper ref %q must start with %s", ref, refPrefix)
	}
	uid := strings.TrimSpace(strings.TrimPrefix(ref, refPrefix))
	if uid == "" {
		return "", fmt.Errorf("keeper ref %q is missing a record UID", ref)
	}
	if strings.Contains(uid, "/") || strings.Contains(uid, "#") {
		return "", fmt.Errorf("keeper ref %q: only keeper://<recordUID> is supported (password field)", ref)
	}
	return uid, nil
}

// passwordFromCLIOutput picks the secret line from Commander stdout, skipping
// common login/sync chatter that can appear ahead of --format=password output.
func passwordFromCLIOutput(s string) string {
	s = stripANSI(s)
	var last string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "logging in"),
			strings.HasPrefix(lower, "successfully authenticated"),
			strings.HasPrefix(lower, "syncing"),
			strings.HasPrefix(lower, "decrypted "),
			strings.HasPrefix(lower, "not logged in"),
			strings.HasPrefix(lower, "password:"),
			strings.HasPrefix(lower, "enter password"),
			strings.HasPrefix(lower, "user("):
			continue
		}
		last = line
	}
	return last
}

func secretFromKeeperJSON(data []byte) (string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("empty json")
	}
	// Commander may print login chatter before the JSON object.
	if idx := bytes.IndexByte(trimmed, '{'); idx > 0 {
		trimmed = trimmed[idx:]
	}
	var doc map[string]any
	if err := json.Unmarshal(trimmed, &doc); err != nil {
		return "", fmt.Errorf("parse json")
	}
	if v := stringFromAny(doc["password"]); v != "" {
		return v, nil
	}
	preferTypes := []string{"password", "secret", "oneTimeCode"}
	preferLabels := []string{"password", "credential", "token", "secret", "pat", "api key", "apikey"}
	for _, typ := range preferTypes {
		if v := firstFieldValue(doc, typ, ""); v != "" {
			return v, nil
		}
	}
	for _, label := range preferLabels {
		if v := firstFieldValue(doc, "", label); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("no password/secret field found")
}

func firstFieldValue(doc map[string]any, wantType, wantLabel string) string {
	for _, key := range []string{"fields", "custom", "custom_fields"} {
		arr, ok := doc[key].([]any)
		if !ok {
			continue
		}
		for _, raw := range arr {
			field, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			typ := strings.ToLower(strings.TrimSpace(stringFromAny(field["type"])))
			label := strings.ToLower(strings.TrimSpace(firstNonEmpty(
				stringFromAny(field["label"]),
				stringFromAny(field["name"]),
			)))
			if wantType != "" && typ != strings.ToLower(wantType) {
				continue
			}
			if wantLabel != "" && label != strings.ToLower(wantLabel) {
				continue
			}
			if v := fieldValue(field); v != "" {
				return v
			}
		}
	}
	return ""
}

func fieldValue(field map[string]any) string {
	if v := stringFromAny(field["value"]); v != "" {
		return v
	}
	if arr, ok := field["value"].([]any); ok {
		for _, item := range arr {
			if v := stringFromAny(item); v != "" {
				return v
			}
		}
	}
	return ""
}

func stringFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case float64:
		// JSON numbers are uncommon for secrets; ignore.
		return ""
	case []any:
		for _, item := range t {
			if s := stringFromAny(item); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func stripANSI(s string) string {
	// Minimal CSI sequence stripper for Commander colorized banners.
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					break
				}
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
