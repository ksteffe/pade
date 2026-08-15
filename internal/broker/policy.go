package broker

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// PolicyFile is server-owned authorization configuration.
// It must not be part of portable pade.yaml.
type PolicyFile struct {
	Version  string       `yaml:"version"`
	OIDC     OIDCConfig   `yaml:"oidc"`
	Policies []PolicyRule `yaml:"policies"`
}

// OIDCConfig configures token verification.
type OIDCConfig struct {
	Issuer   string `yaml:"issuer"`
	Audience string `yaml:"audience"`
	JWKSURL  string `yaml:"jwksURL,omitempty"`
}

// PolicyRule authorizes one subject (optionally confined to repos) for capabilities.
// RequireRepoURLs is a pointer so omission is distinguishable from explicit false;
// Validate requires the field to be set on every rule (fail closed).
type PolicyRule struct {
	Subject         string   `yaml:"subject"`
	RequireRepoURLs *bool    `yaml:"requireRepoURLs"`
	Repositories    []string `yaml:"repositories"`
	Capabilities    []string `yaml:"capabilities"`
}

// LoadPolicy reads and validates a broker policy file.
func LoadPolicy(path string) (*PolicyFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read broker policy: %w", err)
	}
	return ParsePolicy(data)
}

// ParsePolicy unmarshals policy YAML with unknown fields rejected.
func ParsePolicy(data []byte) (*PolicyFile, error) {
	var p PolicyFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("parse broker policy: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate checks structural policy requirements.
func (p *PolicyFile) Validate() error {
	if p.Version != "" && p.Version != "0.1" {
		return fmt.Errorf("unsupported broker policy version %q", p.Version)
	}
	if strings.TrimSpace(p.OIDC.Issuer) == "" {
		return fmt.Errorf("oidc.issuer is required")
	}
	if strings.TrimSpace(p.OIDC.Audience) == "" {
		return fmt.Errorf("oidc.audience is required")
	}
	if len(p.Policies) == 0 {
		return fmt.Errorf("at least one policy rule is required")
	}
	for i, rule := range p.Policies {
		if strings.TrimSpace(rule.Subject) == "" {
			return fmt.Errorf("policies[%d]: subject is required", i)
		}
		if len(rule.Capabilities) == 0 {
			return fmt.Errorf("policies[%d]: capabilities are required", i)
		}
		if rule.RequireRepoURLs == nil {
			return fmt.Errorf("policies[%d]: requireRepoURLs must be explicitly set (true or false)", i)
		}
		if *rule.RequireRepoURLs && len(rule.Repositories) == 0 {
			return fmt.Errorf("policies[%d]: repositories required when requireRepoURLs is true", i)
		}
	}
	return nil
}

// AuthzDecision is a safe authorization outcome (no secrets).
type AuthzDecision struct {
	Allowed    bool
	Reason     string
	Subject    string
	Capability string
}

// Authorize checks whether claims may resolve capability. Fail closed.
func (p *PolicyFile) Authorize(claims Claims, capability string) AuthzDecision {
	capability = strings.TrimSpace(capability)
	dec := AuthzDecision{Subject: claims.Subject, Capability: capability}
	if capability == "" {
		dec.Reason = "capability is required"
		return dec
	}
	if claims.Subject == "" {
		dec.Reason = "token subject is empty"
		return dec
	}

	var matched *PolicyRule
	for i := range p.Policies {
		if p.Policies[i].Subject == claims.Subject {
			matched = &p.Policies[i]
			break
		}
	}
	if matched == nil {
		dec.Reason = "subject not authorized"
		return dec
	}
	if !containsFold(matched.Capabilities, capability) {
		dec.Reason = "capability not authorized for subject"
		return dec
	}
	if matched.RequireRepoURLs != nil && *matched.RequireRepoURLs {
		if len(claims.RepoURLs) == 0 {
			dec.Reason = "repo_urls required but absent (complete repo set unknown)"
			return dec
		}
		if !sameStringSet(normalizeRepos(matched.Repositories), normalizeRepos(claims.RepoURLs)) {
			dec.Reason = "repository set not authorized"
			return dec
		}
	}
	dec.Allowed = true
	dec.Reason = "allowed"
	return dec
}

func containsFold(list []string, want string) bool {
	for _, v := range list {
		if strings.EqualFold(strings.TrimSpace(v), want) {
			return true
		}
	}
	return false
}

func normalizeRepos(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, r := range in {
		r = strings.ToLower(strings.TrimSpace(r))
		r = strings.TrimSuffix(r, ".git")
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	return out
}

func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]struct{}{}
	for _, v := range a {
		m[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := m[v]; !ok {
			return false
		}
	}
	return true
}
