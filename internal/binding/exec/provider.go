// Package exec implements provider: exec — the first dogfood binding for
// independently implemented external providers (stdin/stdout JSON contract).
// See docs/provider-contract.md.
package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/ksteffe/pade/internal/binding"
)

const providerName = "exec"

// Provider runs an external fulfill/derive program.
type Provider struct{}

// New returns an exec provider adapter.
func New() *Provider { return &Provider{} }

func (p *Provider) Name() string { return providerName }

// Probe asks the external program whether the binding can be satisfied.
func (p *Provider) Probe(ctx context.Context, name string, b binding.CapabilityBinding) (binding.ProbeResult, error) {
	if err := requireExec(b); err != nil {
		return binding.ProbeResult{Provider: providerName, Status: "error", Message: err.Error()}, nil
	}
	resp, err := p.invoke(ctx, name, "probe", b.Exec)
	if err != nil {
		return binding.ProbeResult{Provider: providerName, Status: "error", Message: err.Error()}, nil
	}
	status := strings.TrimSpace(resp.Status)
	if status == "" {
		status = "error"
	}
	return binding.ProbeResult{
		Provider: providerName,
		Status:   status,
		Message:  resp.Message,
		Meta:     resp.Meta,
	}, nil
}

// Resolve invokes the external program and returns Material for injection.
func (p *Provider) Resolve(ctx context.Context, name string, b binding.CapabilityBinding) (*binding.Material, error) {
	if err := requireExec(b); err != nil {
		return nil, err
	}
	resp, err := p.invoke(ctx, name, "resolve", b.Exec)
	if err != nil {
		return nil, err
	}
	if resp.Env == nil {
		return nil, fmt.Errorf("exec provider returned no env map")
	}
	mat := &binding.Material{
		Provider: providerName,
		Env:      resp.Env,
	}
	if ts := strings.TrimSpace(resp.ExpiresAt); ts != "" {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("exec provider expiresAt: %w", err)
		}
		mat.ExpiresAt = &t
	}
	return mat, nil
}

type request struct {
	Capability string                 `json:"capability"`
	Operation  string                 `json:"operation"`
	Config     map[string]interface{} `json:"config,omitempty"`
}

type response struct {
	// Probe fields
	Status  string            `json:"status,omitempty"`
	Message string            `json:"message,omitempty"`
	Meta    map[string]string `json:"meta,omitempty"`
	// Resolve fields
	Env       map[string]string `json:"env,omitempty"`
	ExpiresAt string            `json:"expiresAt,omitempty"`
}

func requireExec(b binding.CapabilityBinding) error {
	if b.Exec == nil {
		return fmt.Errorf("exec config is required")
	}
	if len(b.Exec.Command) == 0 {
		return fmt.Errorf("exec.command is required")
	}
	return nil
}

func (p *Provider) invoke(ctx context.Context, capability, operation string, eb *binding.ExecBinding) (*response, error) {
	payload, err := json.Marshal(request{
		Capability: capability,
		Operation:  operation,
		Config:     eb.Config,
	})
	if err != nil {
		return nil, fmt.Errorf("encode provider request: %w", err)
	}

	cmd := osexec.CommandContext(ctx, eb.Command[0], eb.Command[1:]...)
	cmd.Dir = eb.Dir
	cmd.Env = os.Environ()
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		// Do not include stdout (may contain material) in the error.
		return nil, fmt.Errorf("exec provider %q failed: %s", eb.Command[0], msg)
	}

	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return nil, fmt.Errorf("exec provider returned empty stdout")
	}
	var resp response
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("exec provider stdout is not valid JSON")
	}
	return &resp, nil
}
