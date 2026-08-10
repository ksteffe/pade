package onepassword_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/binding"
	onepassword "github.com/ksteffe/pade/internal/binding/onepassword"
)

func TestOnePasswordProbeAndResolve(t *testing.T) {
	t.Parallel()
	fake := writeFakeOp(t, map[string]string{
		"op://Employee/GA Demo/property_id":      "op-property",
		"op://Employee/GA Demo/credentials_path": "/tmp/op-ga.json",
	})

	p := &onepassword.Provider{
		OpBin:    fake,
		LookPath: func(file string) (string, error) { return file, nil },
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, name, arg...)
		},
	}
	b := binding.CapabilityBinding{
		Provider: "onepassword",
		OnePassword: &binding.OnePasswordBinding{
			Refs: map[string]string{
				"GA_PROPERTY_ID":                 "op://Employee/GA Demo/property_id",
				"GOOGLE_APPLICATION_CREDENTIALS": "op://Employee/GA Demo/credentials_path",
			},
		},
	}

	probe, err := p.Probe(context.Background(), "google-analytics.read", b)
	if err != nil {
		t.Fatal(err)
	}
	if probe.Status != "available" {
		t.Fatalf("probe=%+v", probe)
	}
	if probe.Meta["resolvedValues"] != "[hidden]" {
		t.Fatalf("meta=%v", probe.Meta)
	}
	if strings.Contains(probe.Message, "op-property") || strings.Contains(probe.Meta["refs"], "op-property") {
		t.Fatalf("probe leaked secret: %+v", probe)
	}

	mat, err := p.Resolve(context.Background(), "google-analytics.read", b)
	if err != nil {
		t.Fatal(err)
	}
	if mat.Env["GA_PROPERTY_ID"] != "op-property" || mat.Env["GOOGLE_APPLICATION_CREDENTIALS"] != "/tmp/op-ga.json" {
		t.Fatalf("material=%v", mat.Env)
	}
}

func TestOnePasswordMissingCLI(t *testing.T) {
	t.Parallel()
	p := &onepassword.Provider{
		OpBin:    "op-does-not-exist-for-pade-test",
		LookPath: func(file string) (string, error) { return "", os.ErrNotExist },
	}
	b := binding.CapabilityBinding{
		Provider: "onepassword",
		OnePassword: &binding.OnePasswordBinding{
			Refs: map[string]string{"GA_PROPERTY_ID": "op://v/i/f"},
		},
	}
	probe, err := p.Probe(context.Background(), "cap", b)
	if err != nil {
		t.Fatal(err)
	}
	if probe.Status != "unavailable" {
		t.Fatalf("status=%q", probe.Status)
	}
}

func TestParseOnePasswordBinding(t *testing.T) {
	t.Parallel()
	cfg, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  google-analytics.read:
    provider: onepassword
    onepassword:
      refs:
        GA_PROPERTY_ID: "op://Employee/GA Demo/property_id"
`), "bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	b := cfg.Capabilities["google-analytics.read"]
	if b.Provider != "onepassword" || b.OnePassword == nil {
		t.Fatalf("unexpected: %+v", b)
	}
}

func TestRejectNonOpRef(t *testing.T) {
	t.Parallel()
	_, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  demo:
    provider: onepassword
    onepassword:
      refs:
        GA_PROPERTY_ID: "not-an-op-ref"
`), "bindings.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
}

func writeFakeOp(t *testing.T, values map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-op")
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	body.WriteString("if [ \"$1\" != \"read\" ]; then echo 'usage' >&2; exit 2; fi\n")
	body.WriteString("case \"$2\" in\n")
	for ref, val := range values {
		body.WriteString("  '" + ref + "') printf '%s\\n' '" + val + "' ;;\n")
	}
	body.WriteString("  *) echo 'unknown ref' >&2; exit 1 ;;\n")
	body.WriteString("esac\n")
	if err := os.WriteFile(path, []byte(body.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
