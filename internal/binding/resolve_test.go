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
