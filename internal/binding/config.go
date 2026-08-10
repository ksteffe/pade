package binding

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultFileName    = "bindings.yaml"
	WorkspaceConfigDir = ".pade"
	UserConfigSubdir   = "pade"
)

// Config is local, developer-specific capability binding configuration.
// It must not be committed when it contains environment-specific security details.
type Config struct {
	Version      string                       `yaml:"version" json:"version"`
	Capabilities map[string]CapabilityBinding `yaml:"capabilities" json:"capabilities"`
	SourcePath   string                       `yaml:"-" json:"-"`
}

// CapabilityBinding maps a portable capability name to a credential provider.
type CapabilityBinding struct {
	Provider string        `yaml:"provider" json:"provider"`
	Env      []string      `yaml:"env,omitempty" json:"env,omitempty"`
	Vault    *VaultBinding `yaml:"vault,omitempty" json:"vault,omitempty"`
}

// VaultBinding configures a Vault KV lookup. Field values are Vault secret keys
// mapped to process environment variable names — never secret material.
type VaultBinding struct {
	Path   string            `yaml:"path" json:"path"`
	Fields map[string]string `yaml:"fields" json:"fields"` // vaultField -> ENV_NAME
}

// Load reads a bindings file from path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bindings: %w", err)
	}
	return Parse(data, path)
}

// Parse unmarshals bindings YAML.
func Parse(data []byte, sourcePath string) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse bindings YAML: %w", err)
	}
	c.SourcePath = sourcePath
	if c.Capabilities == nil {
		c.Capabilities = map[string]CapabilityBinding{}
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate performs lightweight structural checks (no secret inspection).
func (c *Config) Validate() error {
	if c.Version != "" && c.Version != "0.1" {
		return fmt.Errorf("unsupported bindings version %q (want 0.1)", c.Version)
	}
	for name, b := range c.Capabilities {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("binding capability name is empty")
		}
		if strings.TrimSpace(b.Provider) == "" {
			return fmt.Errorf("binding %q: provider is required", name)
		}
		switch b.Provider {
		case "env":
			if len(b.Env) == 0 {
				return fmt.Errorf("binding %q: env provider requires env key names", name)
			}
			for _, key := range b.Env {
				if strings.TrimSpace(key) == "" {
					return fmt.Errorf("binding %q: empty env key name", name)
				}
				if strings.Contains(key, "=") {
					return fmt.Errorf("binding %q: env entry %q looks like an assignment; declare names only", name, key)
				}
			}
		case "vault":
			if b.Vault == nil {
				return fmt.Errorf("binding %q: vault provider requires vault config", name)
			}
			if strings.TrimSpace(b.Vault.Path) == "" {
				return fmt.Errorf("binding %q: vault.path is required", name)
			}
			if len(b.Vault.Fields) == 0 {
				return fmt.Errorf("binding %q: vault.fields is required", name)
			}
			for field, envName := range b.Vault.Fields {
				if strings.TrimSpace(field) == "" || strings.TrimSpace(envName) == "" {
					return fmt.Errorf("binding %q: vault.fields entries must be non-empty", name)
				}
				if strings.Contains(envName, "=") {
					return fmt.Errorf("binding %q: vault field env name %q looks like an assignment", name, envName)
				}
			}
		default:
			return fmt.Errorf("binding %q: unsupported provider %q", name, b.Provider)
		}
	}
	return nil
}

// Find locates a bindings file. Explicit path wins, then PADE_BINDINGS,
// then <manifestDir>/.pade/bindings.yaml, then ~/.config/pade/bindings.yaml.
// Missing optional files return (nil, "", nil).
func Find(manifestDir, explicit string) (path string, err error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("bindings file not found: %w", err)
		}
		return abs, nil
	}
	if v := strings.TrimSpace(os.Getenv("PADE_BINDINGS")); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("PADE_BINDINGS not found: %w", err)
		}
		return abs, nil
	}
	if manifestDir != "" {
		candidate := filepath.Join(manifestDir, WorkspaceConfigDir, DefaultFileName)
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".config", UserConfigSubdir, DefaultFileName)
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			}
		}
	}
	return "", nil
}

// LoadOptional finds and loads bindings, or returns an empty config when none exist.
func LoadOptional(manifestDir, explicit string) (*Config, error) {
	path, err := Find(manifestDir, explicit)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return &Config{
			Version:      "0.1",
			Capabilities: map[string]CapabilityBinding{},
		}, nil
	}
	return Load(path)
}
