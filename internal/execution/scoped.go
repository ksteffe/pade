package execution

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ksteffe/pade/internal/binding"
)

// Options configures a scoped capability execution.
type Options struct {
	// Command is the argv to run (required).
	Command []string
	// Dir is the working directory (optional).
	Dir string
	// Env is the base environment; defaults to os.Environ().
	Env []string
	// Stdout/Stderr/Stdin default to os streams when nil.
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	// Quiet suppresses the non-secret injection notice on stderr.
	Quiet bool
}

// Result is the outcome of a scoped run. It never includes secret values.
type Result struct {
	Capabilities []string `json:"capabilities"`
	Providers    []string `json:"providers"`
	ExitCode     int      `json:"exitCode"`
}

// Runner executes commands with process-scoped capability material.
type Runner struct {
	Registry *binding.Registry
}

// Run resolves the named capabilities, injects their env only into the child
// process, waits for completion, then discards resolved material.
func (r *Runner) Run(ctx context.Context, cfg *binding.Config, capabilityNames []string, opts Options) (*Result, error) {
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("command is required")
	}
	if r.Registry == nil {
		return nil, fmt.Errorf("provider registry is required")
	}

	results, err := binding.ResolveMaterials(ctx, r.Registry, cfg, capabilityNames)
	if err != nil {
		return nil, err
	}
	defer binding.ClearMaterials(results)

	base := opts.Env
	if base == nil {
		base = os.Environ()
	}
	childEnv := binding.MergeEnv(base, results)

	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	stdin := opts.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}

	res := &Result{
		Capabilities: make([]string, 0, len(results)),
		Providers:    make([]string, 0, len(results)),
	}
	for _, r := range results {
		res.Capabilities = append(res.Capabilities, r.Name)
		res.Providers = append(res.Providers, r.Provider)
	}

	if !opts.Quiet {
		fmt.Fprintf(stderr, "Injecting capabilities: %s\n", formatInjectionNotice(results))
	}

	cmd := exec.CommandContext(ctx, opts.Command[0], opts.Command[1:]...)
	cmd.Dir = opts.Dir
	cmd.Env = childEnv
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin

	err = cmd.Run()
	if err == nil {
		res.ExitCode = 0
		return res, nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		res.ExitCode = ee.ExitCode()
		return res, &ExitError{Code: res.ExitCode, Err: err}
	}
	return res, err
}

// ExitError means the child process ran but exited non-zero.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}

func (e *ExitError) Unwrap() error { return e.Err }

func formatInjectionNotice(results []binding.ResolveResult) string {
	parts := make([]string, 0, len(results))
	for _, r := range results {
		parts = append(parts, fmt.Sprintf("%s (%s)", r.Name, r.Provider))
	}
	return strings.Join(parts, ", ")
}
