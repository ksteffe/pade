package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ksteffe/pade/internal/manifest"
)

func TestParseAndValidateCapabilityFirst(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "examples", "web-app.yaml")
	m, err := manifest.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res, err := manifest.Validate(m)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid, got errors: %v", res.Errors)
	}
	if len(m.Capabilities) != 2 {
		t.Fatalf("expected 2 capabilities, got %d", len(m.Capabilities))
	}
}

func TestParseAndValidateOrchestrated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dcDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dcDir, "devcontainer.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join("..", "..", "spec", "examples", "web-app-orchestrated.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "pade.yaml")
	if err := os.WriteFile(manifestPath, src, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := manifest.Load(manifestPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res, err := manifest.Validate(m)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid, got errors: %v", res.Errors)
	}
}

func TestRejectInvalidVersion(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte("version: \"9.9\"\n"), "pade.yaml")
	if err != nil {
		t.Fatal(err)
	}
	res, err := manifest.Validate(m)
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid {
		t.Fatal("expected invalid version to fail")
	}
}

func TestRejectEnvAssignment(t *testing.T) {
	t.Parallel()
	raw := []byte(`
version: "0.1"
capabilities:
  demo:
    provider: env
    env:
      - SECRET=value
`)
	m, err := manifest.Parse(raw, "pade.yaml")
	if err != nil {
		t.Fatal(err)
	}
	res, err := manifest.Validate(m)
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid {
		t.Fatal("expected env assignment to fail")
	}
}

func TestRejectMissingDevcontainer(t *testing.T) {
	t.Parallel()
	raw := []byte(`
version: "0.1"
environment:
  devcontainer: ".devcontainer/missing.json"
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "pade.yaml")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := manifest.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	res, err := manifest.Validate(m)
	if err != nil {
		t.Fatal(err)
	}
	if res.Valid {
		t.Fatal("expected missing devcontainer to fail")
	}
}
