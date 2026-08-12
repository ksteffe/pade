package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultFileName = "pade.yaml"

// LegacyMigrationHint is returned when a pre-v1alpha1 Intent document is detected.
const LegacyMigrationHint = `legacy PADE v0.1 manifest detected; migrate to
apiVersion: pade.local/v1alpha1
kind: DevelopmentSession`

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
	if err := detectLegacyManifest(data); err != nil {
		return nil, err
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest YAML: %w", err)
	}
	m.SourcePath = sourcePath
	m.rawYAML = append([]byte(nil), data...)
	if m.Spec.Capabilities == nil {
		m.Spec.Capabilities = map[string]CapabilityRequest{}
	}
	return &m, nil
}

// detectLegacyManifest rejects the historical flat version: "0.1" Intent shape.
func detectLegacyManifest(data []byte) error {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		// Let the typed unmarshal report YAML errors.
		return nil
	}
	if raw == nil {
		return nil
	}
	_, hasVersion := raw["version"]
	_, hasAPIVersion := raw["apiVersion"]
	if hasVersion && !hasAPIVersion {
		return fmt.Errorf("%s", LegacyMigrationHint)
	}
	return nil
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
