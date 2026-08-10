package planner_test

import (
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/binding"
	"github.com/ksteffe/pade/internal/manifest"
	"github.com/ksteffe/pade/internal/planner"
)

func TestBuildPlanCapabilityFirst(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    access: read
  datadog.logs.read:
    access: read
`), "pade.yaml")
	if err != nil {
		t.Fatal(err)
	}
	plan := planner.Build(m, planner.BuildOptions{})
	if plan.Workspace.Runtime != "devpod" {
		t.Fatalf("runtime: got %q", plan.Workspace.Runtime)
	}
	if len(plan.Capabilities) != 2 {
		t.Fatalf("capabilities: got %d", len(plan.Capabilities))
	}
	if plan.Capabilities[0].Name != "datadog.logs.read" {
		t.Fatalf("expected sorted capabilities, first=%q", plan.Capabilities[0].Name)
	}
	for _, c := range plan.Capabilities {
		if c.Status == "" {
			t.Fatalf("capability %q missing status", c.Name)
		}
		if strings.Contains(strings.ToLower(c.Status), "secret") {
			t.Fatalf("status must not mention secrets: %q", c.Status)
		}
	}
}

func TestBuildPlanWithBindings(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    access: read
`), "pade.yaml")
	if err != nil {
		t.Fatal(err)
	}
	plan := planner.Build(m, planner.BuildOptions{
		Bindings: &binding.Config{SourcePath: "/tmp/bindings.yaml"},
		Statuses: []binding.Status{{
			Name:     "google-analytics.read",
			Bound:    true,
			Provider: "env",
			Status:   "available",
			Message:  "required environment variables are set",
			Meta:     map[string]string{"env": "GA_PROPERTY_ID,GOOGLE_APPLICATION_CREDENTIALS"},
		}},
	})
	if plan.BindingsPath != "/tmp/bindings.yaml" {
		t.Fatalf("bindings path=%q", plan.BindingsPath)
	}
	c := plan.Capabilities[0]
	if !c.Bound || c.Provider != "env" || c.Status != "available" {
		t.Fatalf("cap=%+v", c)
	}
	if len(c.Requires) != 2 {
		t.Fatalf("requires=%v", c.Requires)
	}
}

func TestBuildPlanDoesNotEchoSecrets(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    provider: env
    env:
      - GA_PROPERTY_ID
      - GOOGLE_APPLICATION_CREDENTIALS
`), "pade.yaml")
	if err != nil {
		t.Fatal(err)
	}
	plan := planner.Build(m, planner.BuildOptions{})
	if len(plan.Capabilities) != 1 {
		t.Fatalf("got %d caps", len(plan.Capabilities))
	}
	c := plan.Capabilities[0]
	for _, r := range c.Requires {
		if strings.Contains(r, "=") {
			t.Fatalf("requires must be names only: %q", r)
		}
	}
}
