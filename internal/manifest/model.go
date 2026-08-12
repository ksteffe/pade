package manifest

import (
	"fmt"
)

// APIVersionV1Alpha1 is the provisional Intent apiVersion.
const APIVersionV1Alpha1 = "pade.local/v1alpha1"

// KindDevelopmentSession is the only Intent kind in v1alpha1.
const KindDevelopmentSession = "DevelopmentSession"

// Manifest is the parsed portable Intent document (DevelopmentSession).
// It never contains secret values.
type Manifest struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Kind       string   `yaml:"kind" json:"kind"`
	Metadata   Metadata `yaml:"metadata" json:"metadata"`
	Spec       Spec     `yaml:"spec" json:"spec"`

	// SourcePath is the filesystem path the manifest was loaded from.
	SourcePath string `yaml:"-" json:"-"`

	// rawYAML retains the original document so schema validation can reject
	// unknown fields that typed unmarshal would otherwise drop.
	rawYAML []byte `yaml:"-" json:"-"`
}

// Metadata identifies the portable object without Kubernetes ObjectMeta fields.
type Metadata struct {
	Name        string            `yaml:"name" json:"name"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// Spec holds portable desired intent for a development session.
type Spec struct {
	Capabilities map[string]CapabilityRequest `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
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
