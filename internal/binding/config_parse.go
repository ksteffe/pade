package binding

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a bindings file from path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bindings: %w", err)
	}
	return Parse(data, path)
}

// Parse unmarshals bindings YAML with unknown fields rejected.
func Parse(data []byte, sourcePath string) (*Config, error) {
	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
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
