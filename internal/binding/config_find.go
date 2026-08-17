package binding

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Find locates a bindings file. Explicit path wins, then PADE_BINDINGS,
// then ~/.config/pade/bindings.yaml, then (only with PADE_TRUST_WORKSPACE_BINDINGS)
// <manifestDir>/.pade/bindings.yaml.
// Missing optional files return ("", nil).
func Find(manifestDir, explicit string) (path string, err error) {
	path, _, err = FindWithNotice(manifestDir, explicit)
	return path, err
}

// FindWithNotice is like Find but also returns a non-secret notice when a
// workspace-local bindings file exists but was skipped for lack of trust opt-in.
func FindWithNotice(manifestDir, explicit string) (path string, notice string, err error) {
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return "", "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", "", fmt.Errorf("bindings file not found: %w", err)
		}
		return abs, "", nil
	}
	if v := strings.TrimSpace(os.Getenv("PADE_BINDINGS")); v != "" {
		abs, err := filepath.Abs(v)
		if err != nil {
			return "", "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", "", fmt.Errorf("PADE_BINDINGS not found: %w", err)
		}
		return abs, "", nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".config", UserConfigSubdir, DefaultFileName)
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, "", nil
			}
		}
	}
	if manifestDir != "" {
		candidate := filepath.Join(manifestDir, WorkspaceConfigDir, DefaultFileName)
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				if trustWorkspaceBindings() {
					return abs, "", nil
				}
				return "", fmt.Sprintf(
					"ignoring untrusted workspace bindings %s (set %s=1 to load, or use --bindings / PADE_BINDINGS / ~/.config/pade/bindings.yaml)",
					abs, TrustWorkspaceBindingsEnv,
				), nil
			}
		}
	}
	return "", "", nil
}

func trustWorkspaceBindings() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(TrustWorkspaceBindingsEnv)))
	switch v {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// LoadOptional finds and loads development-side bindings, or returns an empty
// config when none exist. When a workspace-local bindings file is skipped, a
// notice is written to stderr.
//
// Development-side bindings always reject provider: exec — even when the file
// is explicitly trusted via --bindings, PADE_BINDINGS, user config, or
// PADE_TRUST_WORKSPACE_BINDINGS. Executable providers are broker-side only.
func LoadOptional(manifestDir, explicit string) (*Config, error) {
	path, notice, err := FindWithNotice(manifestDir, explicit)
	if err != nil {
		return nil, err
	}
	if notice != "" {
		fmt.Fprintln(os.Stderr, notice)
	}
	if path == "" {
		return &Config{
			Version:      "0.1",
			Capabilities: map[string]CapabilityBinding{},
		}, nil
	}
	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}
	if err := cfg.RejectDevelopmentSideExec(); err != nil {
		return nil, err
	}
	return cfg, nil
}
