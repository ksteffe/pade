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

func TestExecFailureSurfacesStderr(t *testing.T) {
	stub := buildStub(t)
	b := binding.CapabilityBinding{
		Provider: "exec",
		Exec: &binding.ExecBinding{
			Command: []string{stub},
			Config: map[string]interface{}{
				"fail": true,
			},
		},
	}
	_, err := New().Resolve(context.Background(), "demo.derived", b)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "derived-secret") {
		t.Fatalf("error must not leak secrets: %v", err)
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
