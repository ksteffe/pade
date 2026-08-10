package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ksteffe/pade/internal/manifest"
	"github.com/ksteffe/pade/internal/planner"
)

// WriteJSON encodes v as indented JSON.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteValidateHuman prints validate checks in the design-doc style.
func WriteValidateHuman(w io.Writer, res *manifest.Result) {
	for _, c := range res.Checks {
		fmt.Fprintln(w, c.String())
	}
	if res.Valid {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Manifest OK.")
	}
}

// WritePlanHuman prints a plan without secrets.
func WritePlanHuman(w io.Writer, p *planner.Plan) {
	fmt.Fprintln(w, "Workspace")
	fmt.Fprintf(w, "  runtime: %s\n", p.Workspace.Runtime)
	if p.Workspace.Config != "" {
		fmt.Fprintf(w, "  config: %s\n", p.Workspace.Config)
	}
	fmt.Fprintf(w, "  ownedBy: %s\n", p.Workspace.OwnedBy)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Capabilities")
	if len(p.Capabilities) == 0 {
		fmt.Fprintln(w, "  (none declared)")
	}
	for _, c := range p.Capabilities {
		fmt.Fprintf(w, "  %s\n", c.Name)
		if c.Access != "" {
			fmt.Fprintf(w, "    access: %s\n", c.Access)
		}
		if c.Provider != "" {
			fmt.Fprintf(w, "    provider: %s\n", c.Provider)
		} else {
			fmt.Fprintln(w, "    provider: (unbound)")
		}
		fmt.Fprintf(w, "    required: %v\n", c.Required)
		fmt.Fprintf(w, "    status: %s\n", c.Status)
		if len(c.Requires) > 0 {
			fmt.Fprintln(w, "    requires:")
			for _, key := range c.Requires {
				fmt.Fprintf(w, "      %s\n", key)
			}
		}
	}
	fmt.Fprintln(w)

	if len(p.Services) > 0 {
		fmt.Fprintln(w, "Services")
		for _, s := range p.Services {
			fmt.Fprintf(w, "  %s\n", s.Name)
			fmt.Fprintf(w, "    command: %s\n", s.Command)
			fmt.Fprintf(w, "    port: %d\n", s.Port)
			if s.Ingress != "" {
				fmt.Fprintf(w, "    ingress: %s\n", s.Ingress)
			}
			if s.Note != "" {
				fmt.Fprintf(w, "    note: %s\n", s.Note)
			}
		}
		fmt.Fprintln(w)
	}

	if p.Lifecycle != nil {
		fmt.Fprintln(w, "Lifecycle")
		if p.Lifecycle.IdleTimeout != "" {
			fmt.Fprintf(w, "  idleTimeout: %s\n", p.Lifecycle.IdleTimeout)
		}
		if p.Lifecycle.MaximumLifetime != "" {
			fmt.Fprintf(w, "  maximumLifetime: %s\n", p.Lifecycle.MaximumLifetime)
		}
		fmt.Fprintln(w)
	}

	if len(p.Notes) > 0 {
		fmt.Fprintln(w, "Notes")
		for _, n := range p.Notes {
			fmt.Fprintf(w, "  - %s\n", n)
		}
	}
}

// FormatErrors joins error strings for stderr.
func FormatErrors(errs []string) string {
	return strings.Join(errs, "\n")
}
