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

func TestPasswordFromNoisyStdout(t *testing.T) {
	t.Parallel()
	fake := writeFakeKeeperNoisy(t, "pade-demo-github", "keeper-secret")
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
			Refs: map[string]string{"GITHUB_TOKEN": "keeper://pade-demo-github"},
		},
	}
	mat, err := p.Resolve(context.Background(), "github.user.read", b)
	if err != nil {
		t.Fatal(err)
	}
	if mat.Env["GITHUB_TOKEN"] != "keeper-secret" {
		t.Fatalf("material=%v", mat.Env)
	}
}

func writeFakeKeeperNoisy(t *testing.T, uid, val string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-keeper-noisy")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"get\" ] && [ \"$2\" = \"--format=password\" ]; then\n" +
		"  printf '%s\\n' 'Logging in to Keeper Commander'\n" +
		"  printf '%s\\n' 'Syncing...'\n" +
		"  printf '%s\\n' 'Decrypted [1] record(s)'\n" +
		"  printf '%s\\n' '" + val + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo usage >&2; exit 2\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = uid
	return path
}

func writeFakeKeeper(t *testing.T, values map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-keeper")
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	body.WriteString("uid=\"\"\n")
	body.WriteString("mode=password\n")
	body.WriteString("if [ \"$1\" = \"get\" ] && [ \"$2\" = \"--format=password\" ]; then\n")
	body.WriteString("  if [ \"$3\" = \"--unmask\" ]; then uid=\"$4\"; else uid=\"$3\"; fi\n")
	body.WriteString("elif [ \"$1\" = \"get\" ] && [ \"$2\" = \"--format=json\" ]; then\n")
	body.WriteString("  mode=json\n")
	body.WriteString("  if [ \"$3\" = \"--unmask\" ]; then uid=\"$4\"; else uid=\"$3\"; fi\n")
	body.WriteString("elif [ \"$1\" = \"find-password\" ]; then uid=\"$2\"\n")
	body.WriteString("elif [ \"$1\" = \"clipboard-copy\" ] && [ \"$2\" = \"--output\" ] && [ \"$3\" = \"stdout\" ]; then uid=\"$4\"\n")
	body.WriteString("else echo 'usage' >&2; exit 2; fi\n")
	body.WriteString("val=\"\"\n")
	body.WriteString("case \"$uid\" in\n")
	for uid, val := range values {
		body.WriteString("  '" + uid + "') val='" + val + "' ;;\n")
	}
	body.WriteString("  *) echo 'unknown uid' >&2; exit 1 ;;\n")
	body.WriteString("esac\n")
	body.WriteString("if [ \"$mode\" = \"json\" ]; then\n")
	body.WriteString("  printf '{\"uid\":\"%s\",\"fields\":[{\"type\":\"password\",\"value\":[\"%s\"]}]}\\n' \"$uid\" \"$val\"\n")
	body.WriteString("else\n")
	body.WriteString("  printf '%s\\n' \"$val\"\n")
	body.WriteString("fi\n")
	if err := os.WriteFile(path, []byte(body.String()), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
