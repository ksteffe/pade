package binding_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/binding"
	envprovider "github.com/ksteffe/pade/internal/binding/env"
)

func TestResolveMaterialsEnv(t *testing.T) {
	t.Setenv("PADE_EXEC_TEST_KEY", "secret-value-should-not-leak")
	cfg, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  demo.cap:
    provider: env
    env:
      - PADE_EXEC_TEST_KEY
`), "bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg := binding.NewRegistry(envprovider.New())
	results, err := binding.ResolveMaterials(context.Background(), reg, cfg, []string{"demo.cap"})
	if err != nil {
		t.Fatal(err)
	}
	defer binding.ClearMaterials(results)

	merged := binding.MergeEnv([]string{"PATH=/bin", "PADE_EXEC_TEST_KEY=old"}, results)
	found := false
	for _, e := range merged {
		if strings.HasPrefix(e, "PADE_EXEC_TEST_KEY=") {
			found = true
			if e != "PADE_EXEC_TEST_KEY=secret-value-should-not-leak" {
				t.Fatalf("merged=%q", e)
			}
		}
	}
	if !found {
		t.Fatal("missing injected key")
	}
}

func TestResolveMaterialsMissingBinding(t *testing.T) {
	t.Parallel()
	reg := binding.NewRegistry(envprovider.New())
	_, err := binding.ResolveMaterials(context.Background(), reg, &binding.Config{
		Capabilities: map[string]binding.CapabilityBinding{},
	}, []string{"missing.cap"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// countingProvider records Resolve/Probe calls to prove exec materialization
// does not Probe after a successful Resolve.
type countingProvider struct {
	resolves int
	probes   int
}

func (p *countingProvider) Name() string { return "counting" }

func (p *countingProvider) Probe(context.Context, string, binding.CapabilityBinding) (binding.ProbeResult, error) {
	p.probes++
	return binding.ProbeResult{
		Provider: p.Name(),
		Status:   "available",
		Meta:     map[string]string{"fromProbe": "yes"},
	}, nil
}

func (p *countingProvider) Resolve(context.Context, string, binding.CapabilityBinding) (*binding.Material, error) {
	p.resolves++
	return &binding.Material{
		Provider: p.Name(),
		Env:      map[string]string{"COUNTING_TOKEN": "counting-secret"},
	}, nil
}

func TestResolveMaterialsDoesNotProbeAfterResolve(t *testing.T) {
	t.Parallel()
	cp := &countingProvider{}
	reg := binding.NewRegistry(cp)
	cfg := &binding.Config{
		Capabilities: map[string]binding.CapabilityBinding{
			"demo.cap": {Provider: "counting"},
		},
	}
	results, err := binding.ResolveMaterials(context.Background(), reg, cfg, []string{"demo.cap"})
	if err != nil {
		t.Fatal(err)
	}
	defer binding.ClearMaterials(results)
	if cp.resolves != 1 {
		t.Fatalf("resolves=%d want 1", cp.resolves)
	}
	if cp.probes != 0 {
		t.Fatalf("probes=%d want 0 (Probe must not run after Resolve)", cp.probes)
	}
	if results[0].Meta["resolvedValues"] != "[hidden]" {
		t.Fatalf("meta=%v", results[0].Meta)
	}
	if _, ok := results[0].Meta["fromProbe"]; ok {
		t.Fatal("Meta must not come from Probe on the exec path")
	}
}
