package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ksteffe/pade/spec"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Result is the outcome of validating a manifest.
type Result struct {
	Valid  bool    `json:"valid"`
	Checks []Check `json:"checks"`
	Errors []string `json:"errors,omitempty"`
}

// Validate checks the manifest against the embedded JSON Schema and performs
// lightweight semantic checks (referenced files, ports). It never inspects
// secret values.
func Validate(m *Manifest) (*Result, error) {
	res := &Result{Valid: true}

	if err := validateSchema(m, res); err != nil {
		return nil, err
	}

	if m.Version == "0.1" {
		res.Checks = append(res.Checks, Check{OK: true, Message: fmt.Sprintf("%s is valid", displayName(m.SourcePath))})
	}

	baseDir := filepath.Dir(m.SourcePath)
	if m.Environment != nil && m.Environment.DevContainer != "" {
		dcPath := m.Environment.DevContainer
		if !filepath.IsAbs(dcPath) {
			dcPath = filepath.Join(baseDir, dcPath)
		}
		if _, err := os.Stat(dcPath); err != nil {
			res.Valid = false
			msg := fmt.Sprintf("%s does not exist", m.Environment.DevContainer)
			res.Checks = append(res.Checks, Check{OK: false, Message: msg})
			res.Errors = append(res.Errors, msg)
		} else {
			res.Checks = append(res.Checks, Check{OK: true, Message: fmt.Sprintf("%s exists", m.Environment.DevContainer)})
		}
	}

	for name, svc := range m.Services {
		if svc.Port < 1 || svc.Port > 65535 {
			res.Valid = false
			msg := fmt.Sprintf("service %q has invalid port %d", name, svc.Port)
			res.Checks = append(res.Checks, Check{OK: false, Message: msg})
			res.Errors = append(res.Errors, msg)
			continue
		}
		if strings.TrimSpace(svc.Command) == "" {
			res.Valid = false
			msg := fmt.Sprintf("service %q has empty command", name)
			res.Checks = append(res.Checks, Check{OK: false, Message: msg})
			res.Errors = append(res.Errors, msg)
			continue
		}
		res.Checks = append(res.Checks, Check{
			OK:      true,
			Message: fmt.Sprintf("service %q uses valid port %d", name, svc.Port),
		})
	}

	for name, cap := range m.Capabilities {
		if err := validateCapability(name, cap); err != nil {
			res.Valid = false
			res.Checks = append(res.Checks, Check{OK: false, Message: err.Error()})
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		res.Checks = append(res.Checks, Check{
			OK:      true,
			Message: fmt.Sprintf("capability %q is well formed", name),
		})
	}

	return res, nil
}

func validateCapability(name string, cap CapabilityRequest) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("capability name is empty")
	}
	if cap.Provider == "env" && len(cap.Env) == 0 {
		return fmt.Errorf("capability %q uses provider env but declares no env keys", name)
	}
	for _, key := range cap.Env {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("capability %q has an empty env key name", name)
		}
		// Ensure we only ever deal with names, never values.
		if strings.Contains(key, "=") {
			return fmt.Errorf("capability %q env entry %q looks like an assignment; declare names only", name, key)
		}
	}
	return nil
}

func validateSchema(m *Manifest, res *Result) error {
	compiler := jsonschema.NewCompiler()
	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(spec.JSON)))
	if err != nil {
		return fmt.Errorf("parse embedded schema: %w", err)
	}
	if err := compiler.AddResource("https://pade.dev/schema/v0.1/pade.schema.json", schemaDoc); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	sch, err := compiler.Compile("https://pade.dev/schema/v0.1/pade.schema.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	raw, err := yamlToJSONDocument(m)
	if err != nil {
		return err
	}
	if err := sch.Validate(raw); err != nil {
		res.Valid = false
		msg := fmt.Sprintf("schema validation failed: %s", err.Error())
		res.Checks = append(res.Checks, Check{OK: false, Message: msg})
		res.Errors = append(res.Errors, msg)
	}
	return nil
}

// yamlToJSONDocument converts the typed manifest into a generic JSON value for schema validation.
func yamlToJSONDocument(m *Manifest) (any, error) {
	// Re-marshal through JSON so numbers/bools match JSON Schema expectations.
	b, err := json.Marshal(struct {
		Version      string                        `json:"version"`
		Environment  *Environment                 `json:"environment,omitempty"`
		Services     map[string]Service           `json:"services,omitempty"`
		Capabilities map[string]CapabilityRequest `json:"capabilities,omitempty"`
		Lifecycle    *Lifecycle                   `json:"lifecycle,omitempty"`
	}{
		Version:      m.Version,
		Environment:  m.Environment,
		Services:     omitEmptyServices(m.Services),
		Capabilities: omitEmptyCapabilities(m.Capabilities),
		Lifecycle:    m.Lifecycle,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal manifest for schema: %w", err)
	}
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal manifest JSON: %w", err)
	}
	return doc, nil
}

func omitEmptyServices(in map[string]Service) map[string]Service {
	if len(in) == 0 {
		return nil
	}
	return in
}

func omitEmptyCapabilities(in map[string]CapabilityRequest) map[string]CapabilityRequest {
	if len(in) == 0 {
		return nil
	}
	return in
}

func displayName(path string) string {
	if path == "" {
		return DefaultFileName
	}
	return filepath.Base(path)
}
