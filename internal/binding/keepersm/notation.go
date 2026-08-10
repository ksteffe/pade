package keepersm

import (
	"fmt"
	"strings"
)

const refPrefix = "keeper://"

// NormalizeNotation accepts Keeper Notation refs and expands bare
// keeper://UID to keeper://UID/field/password for Secrets Manager.
// The returned string is a handle only — never secret material.
func NormalizeNotation(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, refPrefix) {
		return "", fmt.Errorf("keeper-secrets-manager ref must start with %s", refPrefix)
	}
	rest := strings.TrimSpace(strings.TrimPrefix(ref, refPrefix))
	if rest == "" {
		return "", fmt.Errorf("keeper-secrets-manager ref is missing a record UID")
	}
	// Bare UID (no path) → password field shorthand.
	if !strings.Contains(rest, "/") {
		if strings.ContainsAny(rest, "[]#") {
			return "", fmt.Errorf("keeper-secrets-manager ref has invalid bare UID")
		}
		return refPrefix + rest + "/field/password", nil
	}
	return refPrefix + rest, nil
}
