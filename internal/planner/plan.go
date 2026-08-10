package planner

import (
	"fmt"
	"sort"

	"github.com/ksteffe/pade/internal/manifest"
)

// Plan is a side-effect-free description of what PADE intends to do.
// It must never include secret values.
type Plan struct {
	ManifestPath string              `json:"manifestPath"`
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
	Name     string   `json:"name"`
	Access   string   `json:"access,omitempty"`
	Provider string   `json:"provider,omitempty"`
	Requires []string `json:"requires,omitempty"`
	Required bool     `json:"required"`
	Status   string   `json:"status"`
}

// ServicePlan is an optional declared service from the manifest.
type ServicePlan struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Port    int    `json:"port"`
	Ingress string `json:"ingress,omitempty"`
	Note    string `json:"note,omitempty"`
}

// Build constructs a plan from a validated manifest.
func Build(m *manifest.Manifest) *Plan {
	p := &Plan{
		ManifestPath: m.SourcePath,
		Workspace: WorkspacePlan{
			Runtime: "devpod",
			OwnedBy: "DevPod (or equivalent existing runtime); PADE does not own workspace lifecycle",
		},
		Notes: []string{
			"Plan is side-effect free: no credentials are resolved or displayed.",
			"Start workspaces with DevPod (for example: devpod up .).",
		},
	}

	if m.Environment != nil && m.Environment.DevContainer != "" {
		p.Workspace.DevContainer = m.Environment.DevContainer
		p.Workspace.Config = m.Environment.DevContainer
	}

	names := make([]string, 0, len(m.Capabilities))
	for name := range m.Capabilities {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cap := m.Capabilities[name]
		provider := cap.Provider
		status := "declared"
		if provider == "" {
			status = "declared (binding unresolved)"
		}
		p.Capabilities = append(p.Capabilities, CapabilityPlan{
			Name:     name,
			Access:   cap.Access,
			Provider: provider,
			Requires: append([]string(nil), cap.Env...),
			Required: cap.IsRequired(),
			Status:   status,
		})
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

// SummaryLine is a short human-readable one-liner.
func (p *Plan) SummaryLine() string {
	return fmt.Sprintf("%d capabilities, %d services", len(p.Capabilities), len(p.Services))
}
