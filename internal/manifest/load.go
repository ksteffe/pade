package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultFileName = "pade.yaml"

// Load reads and parses a PADE manifest from path.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return Parse(data, path)
}

// Parse unmarshals YAML bytes into a Manifest.
func Parse(data []byte, sourcePath string) (*Manifest, error) {
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest YAML: %w", err)
	}
	m.SourcePath = sourcePath
	if m.Services == nil {
		m.Services = map[string]Service{}
	}
	if m.Capabilities == nil {
		m.Capabilities = map[string]CapabilityRequest{}
	}
	return &m, nil
}

// Find looks for pade.yaml starting at dir (or DefaultFileName override via path).
// If path is non-empty, it is used directly.
func Find(dir, path string) (string, error) {
	if path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("manifest not found: %w", err)
		}
		return abs, nil
	}
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	candidate := filepath.Join(dir, DefaultFileName)
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("manifest not found at %s: %w", abs, err)
	}
	return abs, nil
}
