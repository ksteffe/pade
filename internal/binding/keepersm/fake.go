package keepersm

import (
	"fmt"
	"strings"
)

// Demo stub tokens for CI (PADE_KSM_FAKE=1). github-whoami treats pade-demo-* as offline stubs.
var fakeTokens = map[string]string{
	"keeper://pade-demo-github/field/password":       "pade-demo-ksm-token",
	"keeper://pade-demo-github":                      "pade-demo-ksm-token",
	"keeper://pade-demo-alice-github/field/password": "pade-demo-alice-ksm-token",
	"keeper://pade-demo-alice-github":                "pade-demo-alice-ksm-token",
	"keeper://pade-demo-bob-github/field/password":   "pade-demo-bob-ksm-token",
	"keeper://pade-demo-bob-github":                  "pade-demo-bob-ksm-token",
}

type fakeClient struct{}

func (fakeClient) GetNotationResults(notation string) ([]string, error) {
	n := strings.TrimSpace(notation)
	if val, ok := fakeTokens[n]; ok {
		return []string{val}, nil
	}
	// Also accept normalized forms already expanded by NormalizeNotation.
	for ref, val := range fakeTokens {
		norm, err := NormalizeNotation(ref)
		if err == nil && norm == n {
			return []string{val}, nil
		}
	}
	return nil, fmt.Errorf("keeper-secrets-manager fake: unknown notation")
}
