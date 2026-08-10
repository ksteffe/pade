package binding_test

import (
	"context"
	"testing"

	"github.com/ksteffe/pade/internal/binding"
	envprovider "github.com/ksteffe/pade/internal/binding/env"
)

// Milestone 5: same capability name, two binding configs / ambient identities,
// different resolved material — without embedding secrets in the portable manifest.
func TestResolveMaterialsSeparateIdentities(t *testing.T) {
	aliceYAML := []byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    provider: env
    env:
      - GA_PROPERTY_ID
`)
	bobYAML := []byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    provider: env
    env:
      - GA_PROPERTY_ID
`)

	aliceCfg, err := binding.Parse(aliceYAML, "alice.bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	bobCfg, err := binding.Parse(bobYAML, "bob.bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}

	reg := binding.NewRegistry(envprovider.New())

	t.Setenv("GA_PROPERTY_ID", "alice-material")
	aliceResults, err := binding.ResolveMaterials(context.Background(), reg, aliceCfg, []string{"google-analytics.read"})
	if err != nil {
		t.Fatal(err)
	}
	defer binding.ClearMaterials(aliceResults)
	aliceVal := aliceResults[0].Material.Env["GA_PROPERTY_ID"]
	if aliceVal != "alice-material" {
		t.Fatalf("alice GA_PROPERTY_ID=%q", aliceVal)
	}

	t.Setenv("GA_PROPERTY_ID", "bob-material")
	bobResults, err := binding.ResolveMaterials(context.Background(), reg, bobCfg, []string{"google-analytics.read"})
	if err != nil {
		t.Fatal(err)
	}
	defer binding.ClearMaterials(bobResults)
	bobVal := bobResults[0].Material.Env["GA_PROPERTY_ID"]
	if bobVal != "bob-material" {
		t.Fatalf("bob GA_PROPERTY_ID=%q", bobVal)
	}

	if aliceVal == bobVal {
		t.Fatal("expected distinct credential material per identity")
	}
}
