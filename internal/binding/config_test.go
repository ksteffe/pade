package binding_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/binding"
	envprovider "github.com/ksteffe/pade/internal/binding/env"
)

func TestParseEnvBinding(t *testing.T) {
	t.Parallel()
	cfg, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    provider: env
    env:
      - GA_PROPERTY_ID
      - GOOGLE_APPLICATION_CREDENTIALS
`), "bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	b := cfg.Capabilities["google-analytics.read"]
	if b.Provider != "env" || len(b.Env) != 2 {
		t.Fatalf("unexpected binding: %+v", b)
	}
}

func TestRejectEnvAssignmentInBindings(t *testing.T) {
	t.Parallel()
	_, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  demo:
    provider: env
    env:
      - SECRET=value
`), "bindings.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnvProbeAvailability(t *testing.T) {
	t.Setenv("PADE_TEST_CAP_A", "1")
	t.Setenv("PADE_TEST_CAP_B", "2")
	p := envprovider.New()
	b := binding.CapabilityBinding{Provider: "env", Env: []string{"PADE_TEST_CAP_A", "PADE_TEST_CAP_B"}}
	probe, err := p.Probe(context.Background(), "demo", b)
	if err != nil {
		t.Fatal(err)
	}
	if probe.Status != "available" {
		t.Fatalf("status=%q message=%q", probe.Status, probe.Message)
	}
	mat, err := p.Resolve(context.Background(), "demo", b)
	if err != nil {
		t.Fatal(err)
	}
	if mat.Env["PADE_TEST_CAP_A"] != "1" {
		t.Fatalf("resolve failed: %#v", mat.Env)
	}
}

func TestEnvProbeMissing(t *testing.T) {
	os.Unsetenv("PADE_TEST_MISSING_KEY")
	p := envprovider.New()
	b := binding.CapabilityBinding{Provider: "env", Env: []string{"PADE_TEST_MISSING_KEY"}}
	probe, err := p.Probe(context.Background(), "demo", b)
	if err != nil {
		t.Fatal(err)
	}
	if probe.Status != "unavailable" {
		t.Fatalf("status=%q", probe.Status)
	}
	if strings.Contains(probe.Message, "=") {
		t.Fatalf("message must not look like secret assignment: %q", probe.Message)
	}
}

func TestResolveAllUnbound(t *testing.T) {
	t.Parallel()
	reg := binding.NewRegistry(envprovider.New())
	statuses, err := binding.ResolveAll(context.Background(), reg, map[string]binding.CapabilityRequestView{
		"google-analytics.read": {Access: "read", Required: true},
	}, &binding.Config{Capabilities: map[string]binding.CapabilityBinding{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Status != "unbound" || statuses[0].Bound {
		t.Fatalf("unexpected: %+v", statuses)
	}
}

func TestLoadWorkspaceBindings(t *testing.T) {
	dir := t.TempDir()
	padeDir := filepath.Join(dir, ".pade")
	if err := os.MkdirAll(padeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    provider: env
    env:
      - GA_PROPERTY_ID
`)
	if err := os.WriteFile(filepath.Join(padeDir, "bindings.yaml"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := binding.LoadOptional(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SourcePath == "" || cfg.Capabilities["google-analytics.read"].Provider != "env" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
