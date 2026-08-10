package binding

import (
	"context"
	"fmt"
	"sort"
)

// ResolveRequest names a capability to materialize for scoped execution.
type ResolveRequest struct {
	Name string
}

// ResolveResult is the process-scoped material plus safe metadata.
// Material must not be logged or persisted.
type ResolveResult struct {
	Name     string
	Provider string
	Material *Material
	Meta     map[string]string
}

// ResolveMaterials resolves one or more capability bindings into process env material.
func ResolveMaterials(ctx context.Context, reg *Registry, cfg *Config, names []string) ([]ResolveResult, error) {
	if cfg == nil {
		return nil, fmt.Errorf("no bindings configured")
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one capability is required")
	}
	seen := map[string]struct{}{}
	out := make([]ResolveResult, 0, len(names))
	for _, name := range names {
		if name == "" {
			return nil, fmt.Errorf("capability name is empty")
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}

		b, ok := cfg.Capabilities[name]
		if !ok {
			return nil, fmt.Errorf("capability %q has no local binding", name)
		}
		p, ok := reg.Get(b.Provider)
		if !ok {
			return nil, fmt.Errorf("capability %q: unknown provider %q", name, b.Provider)
		}
		mat, err := p.Resolve(ctx, name, b)
		if err != nil {
			// Do not wrap provider errors with secret-bearing context.
			return nil, fmt.Errorf("capability %q: resolve failed: %w", name, err)
		}
		probe, err := p.Probe(ctx, name, b)
		meta := map[string]string{}
		if err == nil {
			meta = probe.Meta
		}
		out = append(out, ResolveResult{
			Name:     name,
			Provider: b.Provider,
			Material: mat,
			Meta:     meta,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// MergeEnv overlays resolved materials onto a base environment list (os.Environ style).
// Later capabilities override earlier keys on conflict.
func MergeEnv(base []string, results []ResolveResult) []string {
	envMap := environToMap(base)
	for _, r := range results {
		if r.Material == nil {
			continue
		}
		for k, v := range r.Material.Env {
			envMap[k] = v
		}
	}
	return mapToEnviron(envMap)
}

// ClearMaterials best-effort zeroes resolved secret maps after use.
func ClearMaterials(results []ResolveResult) {
	for i := range results {
		if results[i].Material == nil {
			continue
		}
		for k := range results[i].Material.Env {
			results[i].Material.Env[k] = ""
			delete(results[i].Material.Env, k)
		}
		results[i].Material.Env = nil
		results[i].Material = nil
	}
}

func environToMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, e := range env {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				out[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return out
}

func mapToEnviron(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+m[k])
	}
	return out
}
