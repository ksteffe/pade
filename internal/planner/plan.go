package planner

import (
	"fmt"
	"sort"

	"github.com/ksteffe/pade/internal/binding"
	"github.com/ksteffe/pade/internal/manifest"
)

// Plan is a side-effect-free description of what PADE intends to do.
// It must never include secret values.
type Plan struct {
	ManifestPath string              `json:"manifestPath"`
	BindingsPath string              `json:"bindingsPath,omitempty"`
	Workspace    WorkspacePlan       `json:"workspace"`
	Capabilities []CapabilityPlan    `json:"capabilities"`
	Services     []ServicePlan       `json:"services,omitempty"`
	Lifecycle    *manifest.Lifecycle `json:"lifecycle,omitempty"`
	Notes        []string            `json:"notes,omitempty"`
}

// WorkspacePlan describes environment ownership for the plan.
type WorkspacePlan struct {
	Runtime      string `json:"runtime"`
	Config       string `json:"config,omitempty"`
	OwnedBy      string `json:"ownedBy"`
	DevContainer string `json:"devcontainer,omitempty"`
}

// CapabilityPlan is inspectable capability intent without secrets.
type CapabilityPlan struct {
	Name     string            `json:"name"`
	Access   string            `json:"access,omitempty"`
	Provider string            `json:"provider,omitempty"`
	Requires []string          `json:"requires,omitempty"`
	Required bool              `json:"required"`
	Bound    bool              `json:"bound"`
	Status   string            `json:"status"`
	Message  string            `json:"message,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// ServicePlan is an optional declared service from the manifest.
type ServicePlan struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Port    int    `json:"port"`
	Ingress string `json:"ingress,omitempty"`
	Note    string `json:"note,omitempty"`
}

// BuildOptions configures plan construction.
type BuildOptions struct {
	Bindings *binding.Config
	Statuses []binding.Status
}

// Build constructs a plan from a validated manifest and optional binding statuses.
func Build(m *manifest.Manifest, opts BuildOptions) *Plan {
	p := &Plan{
		ManifestPath: m.SourcePath,
		Workspace: WorkspacePlan{
			Runtime: "devpod",
			OwnedBy: "DevPod (or equivalent existing runtime); PADE does not own workspace lifecycle",
		},
		Notes: []string{
			"Plan is side-effect free: no credentials are resolved or displayed.",
			"Start workspaces with DevPod (for example: devpod up .).",
			"Capability bindings live in local config (.pade/bindings.yaml or ~/.config/pade/bindings.yaml), not in pade.yaml.",
		},
	}
	if opts.Bindings != nil && opts.Bindings.SourcePath != "" {
		p.BindingsPath = opts.Bindings.SourcePath
	}

	if m.Environment != nil && m.Environment.DevContainer != "" {
		p.Workspace.DevContainer = m.Environment.DevContainer
		p.Workspace.Config = m.Environment.DevContainer
	}

	statusByName := map[string]binding.Status{}
	for _, st := range opts.Statuses {
		statusByName[st.Name] = st
	}

	names := make([]string, 0, len(m.Capabilities))
	for name := range m.Capabilities {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cap := m.Capabilities[name]
		cp := CapabilityPlan{
			Name:     name,
			Access:   cap.Access,
			Provider: cap.Provider,
			Requires: append([]string(nil), cap.Env...),
			Required: cap.IsRequired(),
			Status:   "declared",
		}
		if st, ok := statusByName[name]; ok {
			cp.Bound = st.Bound
			if st.Provider != "" {
				cp.Provider = st.Provider
			}
			cp.Status = st.Status
			cp.Message = st.Message
			cp.Meta = st.Meta
			if metaRequires := requiresFromMeta(st); len(metaRequires) > 0 {
				cp.Requires = metaRequires
			}
		} else if cp.Provider == "" {
			cp.Status = "unbound"
		}
		p.Capabilities = append(p.Capabilities, cp)
	}

	svcNames := make([]string, 0, len(m.Services))
	for name := range m.Services {
		svcNames = append(svcNames, name)
	}
	sort.Strings(svcNames)
	for _, name := range svcNames {
		svc := m.Services[name]
		p.Services = append(p.Services, ServicePlan{
			Name:    name,
			Command: svc.Command,
			Port:    svc.Port,
			Ingress: svc.Ingress,
			Note:    "Service lifecycle is outside capability-first v0.1; shown for manifest compatibility",
		})
	}

	if m.Lifecycle != nil {
		p.Lifecycle = m.Lifecycle
		p.Notes = append(p.Notes, "Lifecycle fields are accepted but not enforced by PADE in v0.1.")
	}

	return p
}

func requiresFromMeta(st binding.Status) []string {
	if st.Meta == nil {
		return nil
	}
	if env, ok := st.Meta["env"]; ok && env != "" {
		return splitCSV(env)
	}
	return nil
}

func splitCSV(s string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				parts = append(parts, part)
			}
			start = i + 1
		}
	}
	return parts
}

// SummaryLine is a short human-readable one-liner.
func (p *Plan) SummaryLine() string {
	return fmt.Sprintf("%d capabilities, %d services", len(p.Capabilities), len(p.Services))
}
