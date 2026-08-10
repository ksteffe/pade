package keeper

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/ksteffe/pade/internal/binding"
)

const refPrefix = "keeper://"

// Provider resolves capabilities via Keeper Commander (`keeper get --format=password`).
// PADE_KEEPER_BIN may override the binary path (used by the dogfood fake-keeper shim).
type Provider struct {
	// KeeperBin defaults to "keeper", or PADE_KEEPER_BIN when set at New() time.
	KeeperBin string
	// LookPath finds KeeperBin on PATH; defaults to exec.LookPath.
	LookPath func(file string) (string, error)
	// CommandContext builds an *exec.Cmd; defaults to exec.CommandContext.
	CommandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

func New() *Provider {
	bin := strings.TrimSpace(os.Getenv("PADE_KEEPER_BIN"))
	if bin == "" {
		bin = "keeper"
	}
	return &Provider{
		KeeperBin:      bin,
		LookPath:       exec.LookPath,
		CommandContext: exec.CommandContext,
	}
}

func (p *Provider) Name() string { return "keeper" }

func (p *Provider) Probe(ctx context.Context, name string, b binding.CapabilityBinding) (binding.ProbeResult, error) {
	meta := keeperMeta(b)
	if err := requireConfig(b); err != nil {
		return binding.ProbeResult{
			Provider: p.Name(),
			Status:   "unavailable",
			Message:  err.Error(),
			Meta:     meta,
		}, nil
	}
	if _, err := p.resolveBin(); err != nil {
		return binding.ProbeResult{
			Provider: p.Name(),
			Status:   "unavailable",
			Message:  err.Error(),
			Meta:     meta,
		}, nil
	}
	envNames := sortedKeys(b.Keeper.Refs)
	ref := b.Keeper.Refs[envNames[0]]
	if _, err := p.readRef(ctx, ref); err != nil {
		return binding.ProbeResult{
			Provider: p.Name(),
			Status:   "unavailable",
			Message:  err.Error(),
			Meta:     meta,
		}, nil
	}
	_ = name
	return binding.ProbeResult{
		Provider: p.Name(),
		Status:   "available",
		Message:  "keeper refs reachable; values hidden",
		Meta:     meta,
	}, nil
}

func (p *Provider) Resolve(ctx context.Context, name string, b binding.CapabilityBinding) (*binding.Material, error) {
	_ = name
	if err := requireConfig(b); err != nil {
		return nil, err
	}
	env := make(map[string]string, len(b.Keeper.Refs))
	for envName, ref := range b.Keeper.Refs {
		val, err := p.readRef(ctx, ref)
		if err != nil {
			return nil, err
		}
		env[envName] = val
	}
	return &binding.Material{Provider: p.Name(), Env: env}, nil
}

func requireConfig(b binding.CapabilityBinding) error {
	if b.Keeper == nil {
		return fmt.Errorf("keeper binding config is missing")
	}
	if len(b.Keeper.Refs) == 0 {
		return fmt.Errorf("keeper.refs is required")
	}
	return nil
}

func keeperMeta(b binding.CapabilityBinding) map[string]string {
	meta := map[string]string{"resolvedValues": "[hidden]"}
	if b.Keeper == nil {
		return meta
	}
	parts := make([]string, 0, len(b.Keeper.Refs))
	for envName, ref := range b.Keeper.Refs {
		parts = append(parts, envName+"<-"+ref)
	}
	sort.Strings(parts)
	meta["refs"] = strings.Join(parts, ",")
	return meta
}

func (p *Provider) resolveBin() (string, error) {
	bin := p.KeeperBin
	if bin == "" {
		bin = "keeper"
	}
	look := p.LookPath
	if look == nil {
		look = exec.LookPath
	}
	path, err := look(bin)
	if err != nil {
		if strings.Contains(bin, string(os.PathSeparator)) {
			if st, statErr := os.Stat(bin); statErr == nil && !st.IsDir() {
				return bin, nil
			}
		}
		return "", fmt.Errorf("keeper CLI %q not found (install Keeper Commander or set PADE_KEEPER_BIN)", bin)
	}
	return path, nil
}

func parseUID(ref string) (string, error) {
	if !strings.HasPrefix(ref, refPrefix) {
		return "", fmt.Errorf("keeper ref %q must start with %s", ref, refPrefix)
	}
	uid := strings.TrimSpace(strings.TrimPrefix(ref, refPrefix))
	if uid == "" {
		return "", fmt.Errorf("keeper ref %q is missing a record UID", ref)
	}
	if strings.Contains(uid, "/") || strings.Contains(uid, "#") {
		return "", fmt.Errorf("keeper ref %q: only keeper://<recordUID> is supported (password field)", ref)
	}
	return uid, nil
}

func (p *Provider) readRef(ctx context.Context, ref string) (string, error) {
	uid, err := parseUID(ref)
	if err != nil {
		return "", err
	}
	bin, err := p.resolveBin()
	if err != nil {
		return "", err
	}
	cmdFn := p.CommandContext
	if cmdFn == nil {
		cmdFn = exec.CommandContext
	}
	cmd := cmdFn(ctx, bin, "get", "--format=password", uid)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Never include stdout/stderr bodies — they may contain secret material.
		return "", fmt.Errorf("keeper get failed for ref %s", ref)
	}
	val := strings.TrimRight(stdout.String(), "\r\n")
	if val == "" {
		return "", fmt.Errorf("keeper get returned empty value for ref %s", ref)
	}
	return val, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
