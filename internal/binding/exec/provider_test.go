package exec

import (
	"context"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ksteffe/pade/internal/binding"
)

func TestExecResolveAndProbe(t *testing.T) {
	stub := buildStub(t)
	cfg := map[string]interface{}{
		"tokenEnv": "DEMO_TOKEN",
		"value":    "derived-secret-value",
	}
	b := binding.CapabilityBinding{
		Provider: "exec",
		Exec: &binding.ExecBinding{
			Command: []string{stub},
			Config:  cfg,
		},
	}
	p := New()
	ctx := context.Background()

	probe, err := p.Probe(ctx, "demo.derived", b)
	if err != nil {
		t.Fatal(err)
	}
	if probe.Status != "available" {
		t.Fatalf("probe status=%q message=%q", probe.Status, probe.Message)
	}

	mat, err := p.Resolve(ctx, "demo.derived", b)
	if err != nil {
		t.Fatal(err)
	}
	if mat.Env["DEMO_TOKEN"] != "derived-secret-value" {
		t.Fatalf("env=%v", mat.Env)
	}
	if mat.ExpiresAt == nil {
		t.Fatal("expected expiresAt")
	}
	if mat.ExpiresAt.Before(time.Now()) {
		t.Fatalf("expiresAt in the past: %s", mat.ExpiresAt)
	}
}

func TestExecFailureDoesNotLeakStderr(t *testing.T) {
	script := filepath.Join(t.TempDir(), "fail.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'BOOTSTRAP_SECRET=super-secret' >&2\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	b := binding.CapabilityBinding{
		Provider: "exec",
		Exec: &binding.ExecBinding{
			Command: []string{script},
		},
	}
	_, err := New().Resolve(context.Background(), "demo.derived", b)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "BOOTSTRAP_SECRET") || strings.Contains(msg, "super-secret") {
		t.Fatalf("error must not leak stderr: %v", err)
	}
	if !strings.Contains(msg, "failed") {
		t.Fatalf("expected failure message: %v", err)
	}
}

func TestExecOversizedStdout(t *testing.T) {
	script := filepath.Join(t.TempDir(), "big.sh")
	// Write more than 1 MiB to stdout.
	if err := os.WriteFile(script, []byte("#!/bin/sh\ndd if=/dev/zero bs=1024 count=1100 2>/dev/null\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := New().Resolve(context.Background(), "demo", binding.CapabilityBinding{
		Provider: "exec",
		Exec:     &binding.ExecBinding{Command: []string{script}},
	})
	if err == nil || !strings.Contains(err.Error(), "stdout exceeded") {
		t.Fatalf("expected stdout limit error, got %v", err)
	}
}

func TestExecOversizedStderr(t *testing.T) {
	script := filepath.Join(t.TempDir(), "bigerr.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ndd if=/dev/zero bs=1024 count=1100 >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := New().Resolve(context.Background(), "demo", binding.CapabilityBinding{
		Provider: "exec",
		Exec:     &binding.ExecBinding{Command: []string{script}},
	})
	if err == nil || !strings.Contains(err.Error(), "stderr exceeded") {
		t.Fatalf("expected stderr limit error, got %v", err)
	}
}

func TestProviderEnvironOmitsAmbientSecrets(t *testing.T) {
	t.Setenv("UNRELATED_SECRET", "should-not-pass")
	t.Setenv("PADE_TEST_OK", "1")
	env := providerEnviron()
	for _, e := range env {
		if strings.HasPrefix(e, "UNRELATED_SECRET=") {
			t.Fatal("unrelated ambient secret leaked to provider env")
		}
	}
	found := false
	for _, e := range env {
		if e == "PADE_TEST_OK=1" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected PADE_ prefix to be allowlisted")
	}
}

func TestBindingsValidateExec(t *testing.T) {
	cfg := &binding.Config{
		Version: "0.1",
		Capabilities: map[string]binding.CapabilityBinding{
			"demo.derived": {
				Provider: "exec",
				Exec: &binding.ExecBinding{
					Command: []string{"./bin/pade-provider-stub"},
				},
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.Capabilities["demo.derived"].Exec.Command = nil
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func buildStub(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	out := filepath.Join(t.TempDir(), "pade-provider-stub")
	cmd := osexec.Command("go", "build", "-o", out, "./examples/providers/stub")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build stub: %v\n%s", err, outBytes)
	}
	return out
}
