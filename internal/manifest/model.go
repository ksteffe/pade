package manifest

import (
	"fmt"
)

// Manifest is the parsed portable workspace declaration.
// It never contains secret values.
type Manifest struct {
	Version      string                          `yaml:"version" json:"version"`
	Environment  *Environment                   `yaml:"environment,omitempty" json:"environment,omitempty"`
	Services     map[string]Service             `yaml:"services,omitempty" json:"services,omitempty"`
	Capabilities map[string]CapabilityRequest   `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Lifecycle    *Lifecycle                     `yaml:"lifecycle,omitempty" json:"lifecycle,omitempty"`

	// SourcePath is the filesystem path the manifest was loaded from.
	SourcePath string `yaml:"-" json:"-"`
}

// Environment declares optional environment pointers (Dev Containers).
// Workspace lifecycle remains owned by DevPod or an equivalent runtime.
type Environment struct {
	DevContainer string `yaml:"devcontainer,omitempty" json:"devcontainer,omitempty"`
}

// Service is an optional named process declaration from earlier design drafts.
type Service struct {
	Command string `yaml:"command" json:"command"`
	Port    int    `yaml:"port" json:"port"`
	Ingress string `yaml:"ingress,omitempty" json:"ingress,omitempty"`
}

// CapabilityRequest declares required authority without credentials.
type CapabilityRequest struct {
	Access   string   `yaml:"access,omitempty" json:"access,omitempty"`
	Provider string   `yaml:"provider,omitempty" json:"provider,omitempty"`
	Env      []string `yaml:"env,omitempty" json:"env,omitempty"`
	Required *bool    `yaml:"required,omitempty" json:"required,omitempty"`
}

// IsRequired reports whether the capability must be satisfied.
// Defaults to true when omitted.
func (c CapabilityRequest) IsRequired() bool {
	if c.Required == nil {
		return true
	}
	return *c.Required
}

// Lifecycle holds optional lifecycle hints (may be unsupported by providers).
type Lifecycle struct {
	IdleTimeout      string `yaml:"idleTimeout,omitempty" json:"idleTimeout,omitempty"`
	MaximumLifetime string `yaml:"maximumLifetime,omitempty" json:"maximumLifetime,omitempty"`
}

// Check reports a single validation finding.
type Check struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (c Check) String() string {
	mark := "✗"
	if c.OK {
		mark = "✓"
	}
	return fmt.Sprintf("%s %s", mark, c.Message)
}
