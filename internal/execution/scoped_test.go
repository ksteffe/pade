package execution_test

import (
	"bytes"
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/ksteffe/pade/internal/binding"
	envprovider "github.com/ksteffe/pade/internal/binding/env"
	"github.com/ksteffe/pade/internal/execution"
)

func TestScopedRunInjectsOnlyChildEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell quoting differs on windows")
	}
	t.Setenv("PADE_SCOPED_SECRET", "child-only-value")
	cfg, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  demo.read:
    provider: env
    env:
      - PADE_SCOPED_SECRET
`), "bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	runner := &execution.Runner{Registry: binding.NewRegistry(envprovider.New())}

	// Base env without the secret — injection must add it for the child only.
	base := filterEnv(os.Environ(), "PADE_SCOPED_SECRET")
	res, err := runner.Run(context.Background(), cfg, []string{"demo.read"}, execution.Options{
		Command: []string{"/bin/sh", "-c", `printf '%s' "$PADE_SCOPED_SECRET"`},
		Env:     base,
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("run: %v stderr=%s", err, stderr.String())
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit=%d", res.ExitCode)
	}
	if stdout.String() != "child-only-value" {
		t.Fatalf("stdout=%q", stdout.String())
	}
	if strings.Contains(stderr.String(), "child-only-value") {
		t.Fatalf("stderr leaked secret: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "demo.read (env)") {
		t.Fatalf("expected injection notice, got %q", stderr.String())
	}
	if _, ok := lookupEnv(base, "PADE_SCOPED_SECRET"); ok {
		t.Fatal("base env should not include secret")
	}
}

func TestScopedRunPropagatesExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell quoting differs on windows")
	}
	t.Setenv("PADE_SCOPED_OK", "1")
	cfg, err := binding.Parse([]byte(`
version: "0.1"
capabilities:
  demo.read:
    provider: env
    env:
      - PADE_SCOPED_OK
`), "bindings.yaml")
	if err != nil {
		t.Fatal(err)
	}
	runner := &execution.Runner{Registry: binding.NewRegistry(envprovider.New())}
	var stderr bytes.Buffer
	_, err = runner.Run(context.Background(), cfg, []string{"demo.read"}, execution.Options{
		Command: []string{"/bin/sh", "-c", "exit 7"},
		Stderr:  &stderr,
		Quiet:   true,
	})
	ee, ok := err.(*execution.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T %v", err, err)
	}
	if ee.Code != 7 {
		t.Fatalf("code=%d", ee.Code)
	}
}

func TestScopedRunMissingCapability(t *testing.T) {
	t.Parallel()
	runner := &execution.Runner{Registry: binding.NewRegistry(envprovider.New())}
	_, err := runner.Run(context.Background(), &binding.Config{
		Capabilities: map[string]binding.CapabilityBinding{},
	}, []string{"nope"}, execution.Options{
		Command: []string{"true"},
		Quiet:   true,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func filterEnv(env []string, dropKey string) []string {
	out := make([]string, 0, len(env))
	prefix := dropKey + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}

func lookupEnv(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return strings.TrimPrefix(e, prefix), true
		}
	}
	return "", false
}
