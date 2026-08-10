package keeper_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/binding"
	keeper "github.com/ksteffe/pade/internal/binding/keeper"
)

func TestKeeperProbeAndResolve(t *testing.T) {
	t.Parallel()
	fake := writeFakeKeeper(t, map[string]string{
		"pade-demo-github": "keeper-secret",
	})

	p := &keeper.Provider{
		KeeperBin: fake,
		LookPath:  func(file string) (string, error) { return file, nil },
		CommandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			return exec.CommandContext(ctx, name, arg...)
		},
	}
	b := binding.CapabilityBinding{
		Provider: "keeper",
		Keeper: &binding.KeeperBinding{
			Refs: map[string]string{
				"GITHUB_TOKEN": "keeper://pade-demo-github",
			},
		},
	}

	probe, err := p.Probe(context.Background(), "github.user.read", b)
	if err != nil {
		t.Fatal(err)
	}
	if probe.Status != "available" {
		t.Fatalf("probe=%+v", probe)
	}
	if probe.Meta["resolvedValues"] != "[hidden]" {
		t.Fatalf("meta=%v", probe.Meta)
	}
	if strings.Contains(probe.Message, "keeper-secret") || strings.Contains(probe.Meta["refs"], "keeper-secret") {
		t.Fatalf("probe leaked secret: %+v", probe)
	}

	mat, err := p.Resolve(context.Background(), "github.user.read", b)
	if err != nil {
		t.Fatal(err)
	}
	if mat.Env["GITHUB_TOKEN"] != "keeper-secret" {
		t.Fatalf("material=%v", mat.Env)
	}
}

func TestKeeperMissingCLI(t *testing.T) {
	t.Parallel()
	p := &keeper.Provider{
		KeeperBin: "keeper-does-not-exist-for-pade-test",
		LookPath:  func(file string) (string, error) { return "", os.ErrNotExist },
	}
	b := binding.CapabilityBinding{
		Provider: "keeper",
		Keeper: &binding.KeeperBinding{
			Refs: map[string]string{"GITHUB_TOKEN": "keeper://some-uid"},
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

func TestParseKeeperBinding(t *testing.T) {
	t.Parallel()
	cfg, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  github.user.read:
    provider: keeper
    keeper:
      refs:
        GITHUB_TOKEN: "keeper://abc123UID"
`), "bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	b := cfg.Capabilities["github.user.read"]
	if b.Provider != "keeper" || b.Keeper == nil {
		t.Fatalf("unexpected: %+v", b)
	}
}

func TestRejectNonKeeperRef(t *testing.T) {
	t.Parallel()
	_, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  demo:
    provider: keeper
    keeper:
      refs:
        GITHUB_TOKEN: "not-a-keeper-ref"
`), "bindings.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
}

func writeFakeKeeper(t *testing.T, values map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-keeper")
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	body.WriteString("if [ \"$1\" != \"get\" ] || [ \"$2\" != \"--format=password\" ]; then echo 'usage' >&2; exit 2; fi\n")
	body.WriteString("case \"$3\" in\n")
	for uid, val := range values {
		body.WriteString("  '" + uid + "') printf '%s\\n' '" + val + "' ;;\n")
	}
	body.WriteString("  *) echo 'unknown uid' >&2; exit 1 ;;\n")
	body.WriteString("esac\n")
	if err := os.WriteFile(path, []byte(body.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
