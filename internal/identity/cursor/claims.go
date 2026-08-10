package cursor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SafeClaims are decoded JWT payload fields safe to display.
// The raw token is never included.
type SafeClaims struct {
	Provider      string   `json:"provider"`
	Subject       string   `json:"subject,omitempty"`
	CloudAgentID  string   `json:"cloudAgentId,omitempty"`
	AgentRuntime  string   `json:"agentRuntime,omitempty"`
	OwnerUserID   string   `json:"ownerUserId,omitempty"`
	TeamID        string   `json:"teamId,omitempty"`
	RepoURL       string   `json:"repoUrl,omitempty"`
	RepoURLs      []string `json:"repoUrls,omitempty"`
	RepoCount     int      `json:"repoCount,omitempty"`
	EnvironmentID string   `json:"environmentId,omitempty"`
	Source        string   `json:"source,omitempty"`
	ExpiresAt     string   `json:"expiresAt,omitempty"`
	Audience      string   `json:"audience,omitempty"`
}

// DecodeSafeClaims parses the JWT payload without verifying the signature.
// Verification belongs to relying parties (for example pade-broker).
// The raw token must not be returned or logged by callers of this helper.
func DecodeSafeClaims(token string) (SafeClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return SafeClaims{}, fmt.Errorf("cursor identity: token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some issuers use padded encoding.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return SafeClaims{}, fmt.Errorf("cursor identity: invalid JWT payload encoding")
		}
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return SafeClaims{}, fmt.Errorf("cursor identity: invalid JWT payload JSON")
	}
	out := SafeClaims{Provider: "cursor"}
	out.Subject = stringClaim(raw, "sub")
	out.CloudAgentID = stringClaim(raw, "cloud_agent_id")
	out.AgentRuntime = stringClaim(raw, "agent_runtime")
	out.OwnerUserID = stringClaim(raw, "owner_user_id")
	out.TeamID = stringClaim(raw, "team_id")
	out.RepoURL = stringClaim(raw, "repo_url")
	out.EnvironmentID = stringClaim(raw, "environment_id")
	out.Source = stringClaim(raw, "source")
	out.Audience = stringClaim(raw, "aud")
	if urls, ok := raw["repo_urls"].([]interface{}); ok {
		for _, u := range urls {
			if s, ok := u.(string); ok && s != "" {
				out.RepoURLs = append(out.RepoURLs, s)
			}
		}
	}
	if n, ok := raw["repo_count"].(float64); ok {
		out.RepoCount = int(n)
	}
	if exp, ok := raw["exp"].(float64); ok {
		out.ExpiresAt = time.Unix(int64(exp), 0).UTC().Format(time.RFC3339)
	}
	return out, nil
}

func stringClaim(raw map[string]interface{}, key string) string {
	v, ok := raw[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// Numeric ids sometimes appear as JSON numbers; prefer string claims from Cursor.
		return fmt.Sprintf("%.0f", t)
	default:
		return fmt.Sprint(t)
	}
}
