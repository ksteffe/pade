package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/ksteffe/pade/internal/binding"
	"github.com/ksteffe/pade/internal/binding/cliproc"
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

	// Commander often masks secrets unless --unmask is set. Keep this to one
	// primary CLI round-trip (Commander startup/sync is expensive), with a
	// single JSON fallback for records that store the PAT outside "password".
	if val, err := p.runPasswordCmd(ctx, cmdFn, bin, ref, "get", "--format=password", "--unmask", uid); err == nil {
		return val, nil
	} else if shouldNotFallback(ctx, err) {
		return "", err
	}
	if val, err := p.readPasswordFromJSON(ctx, cmdFn, bin, ref, uid); err == nil {
		return val, nil
	} else if shouldNotFallback(ctx, err) {
		return "", err
	}
	return "", fmt.Errorf("keeper returned empty password for ref %s (try get --format=password --unmask / password|secret|credential field)", ref)
}

func shouldNotFallback(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if ctx.Err() != nil {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "exceeded size limit") || strings.Contains(msg, "not found")
}

func (p *Provider) runPasswordCmd(ctx context.Context, cmdFn func(context.Context, string, ...string) *exec.Cmd, bin, ref string, args ...string) (string, error) {
	cmd := cmdFn(ctx, bin, args...)
	cmd.Env = cliproc.Environ(nil, []string{"KEEPER_"})
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = 5 * time.Second
	}
	var stdout, stderr cliproc.LimitedBuffer
	stdout.Limit = cliproc.MaxOutput
	stderr.Limit = cliproc.MaxOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Avoid inheriting a TTY so Commander does not mix interactive prompts into stdout.
	cmd.Stdin = nil
	if err := cmd.Run(); err != nil {
		if stdout.Exceed || stderr.Exceed {
			return "", fmt.Errorf("keeper %s output exceeded size limit for ref %s", args[0], ref)
		}
		// Never include stdout/stderr bodies — they may contain secret material.
		return "", fmt.Errorf("keeper %s failed for ref %s", args[0], ref)
	}
	if stdout.Exceed || stderr.Exceed {
		return "", fmt.Errorf("keeper %s output exceeded size limit for ref %s", args[0], ref)
	}
	val := passwordFromCLIOutput(stdout.String())
	if val == "" {
		return "", fmt.Errorf("keeper %s returned empty password for ref %s (use Login password field + --unmask; record UID not title)", args[0], ref)
	}
	return val, nil
}

func (p *Provider) readPasswordFromJSON(ctx context.Context, cmdFn func(context.Context, string, ...string) *exec.Cmd, bin, ref, uid string) (string, error) {
	cmd := cmdFn(ctx, bin, "get", "--format=json", "--unmask", uid)
	cmd.Env = cliproc.Environ(nil, []string{"KEEPER_"})
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = 5 * time.Second
	}
	var stdout, stderr cliproc.LimitedBuffer
	stdout.Limit = cliproc.MaxOutput
	stderr.Limit = cliproc.MaxOutput
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil
	if err := cmd.Run(); err != nil {
		if stdout.Exceed || stderr.Exceed {
			return "", fmt.Errorf("keeper get --format=json output exceeded size limit for ref %s", ref)
		}
		return "", fmt.Errorf("keeper get --format=json failed for ref %s", ref)
	}
	if stdout.Exceed || stderr.Exceed {
		return "", fmt.Errorf("keeper get --format=json output exceeded size limit for ref %s", ref)
	}
	val, err := secretFromKeeperJSON(stdout.Bytes())
	if err != nil {
		return "", fmt.Errorf("keeper json secret extract failed for ref %s: %w", ref, err)
	}
	return val, nil
}

// passwordFromCLIOutput picks the secret line from Commander stdout, skipping
// common login/sync chatter that can appear ahead of --format=password output.
func passwordFromCLIOutput(s string) string {
	s = stripANSI(s)
	var last string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "logging in"),
			strings.HasPrefix(lower, "successfully authenticated"),
			strings.HasPrefix(lower, "syncing"),
			strings.HasPrefix(lower, "decrypted "),
			strings.HasPrefix(lower, "not logged in"),
			strings.HasPrefix(lower, "password:"),
			strings.HasPrefix(lower, "enter password"),
			strings.HasPrefix(lower, "user("):
			continue
		}
		last = line
	}
	return last
}

func secretFromKeeperJSON(data []byte) (string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("empty json")
	}
	// Commander may print login chatter before the JSON object.
	if idx := bytes.IndexByte(trimmed, '{'); idx > 0 {
		trimmed = trimmed[idx:]
	}
	var doc map[string]any
	if err := json.Unmarshal(trimmed, &doc); err != nil {
		return "", fmt.Errorf("parse json")
	}
	if v := stringFromAny(doc["password"]); v != "" {
		return v, nil
	}
	preferTypes := []string{"password", "secret", "oneTimeCode"}
	preferLabels := []string{"password", "credential", "token", "secret", "pat", "api key", "apikey"}
	for _, typ := range preferTypes {
		if v := firstFieldValue(doc, typ, ""); v != "" {
			return v, nil
		}
	}
	for _, label := range preferLabels {
		if v := firstFieldValue(doc, "", label); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("no password/secret field found")
}

func firstFieldValue(doc map[string]any, wantType, wantLabel string) string {
	for _, key := range []string{"fields", "custom", "custom_fields"} {
		arr, ok := doc[key].([]any)
		if !ok {
			continue
		}
		for _, raw := range arr {
			field, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			typ := strings.ToLower(strings.TrimSpace(stringFromAny(field["type"])))
			label := strings.ToLower(strings.TrimSpace(firstNonEmpty(
				stringFromAny(field["label"]),
				stringFromAny(field["name"]),
			)))
			if wantType != "" && typ != strings.ToLower(wantType) {
				continue
			}
			if wantLabel != "" && label != strings.ToLower(wantLabel) {
				continue
			}
			if v := fieldValue(field); v != "" {
				return v
			}
		}
	}
	return ""
}

func fieldValue(field map[string]any) string {
	if v := stringFromAny(field["value"]); v != "" {
		return v
	}
	if arr, ok := field["value"].([]any); ok {
		for _, item := range arr {
			if v := stringFromAny(item); v != "" {
				return v
			}
		}
	}
	return ""
}

func stringFromAny(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case float64:
		// JSON numbers are uncommon for secrets; ignore.
		return ""
	case []any:
		for _, item := range t {
			if s := stringFromAny(item); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func stripANSI(s string) string {
	// Minimal CSI sequence stripper for Commander colorized banners.
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
					break
				}
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
