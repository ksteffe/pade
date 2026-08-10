package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ksteffe/pade/internal/binding"
	envprovider "github.com/ksteffe/pade/internal/binding/env"
	vaultprovider "github.com/ksteffe/pade/internal/binding/vault"
	"github.com/ksteffe/pade/internal/manifest"
	"github.com/ksteffe/pade/internal/output"
	"github.com/ksteffe/pade/internal/planner"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:           "pade",
		Short:         "Portable Agent Development Environments CLI",
		Long:          "PADE validates and plans portable capability declarations for agent development environments. Workspace lifecycle is owned by DevPod (or equivalent).",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	var (
		file     string
		bindings string
		jsonOut  bool
	)

	root.PersistentFlags().StringVarP(&file, "file", "f", "", "path to pade.yaml (default: ./pade.yaml)")
	root.PersistentFlags().StringVar(&bindings, "bindings", "", "path to local bindings.yaml (default: .pade/bindings.yaml, PADE_BINDINGS, or ~/.config/pade/bindings.yaml)")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")

	root.AddCommand(newValidateCmd(&file, &jsonOut))
	root.AddCommand(newPlanCmd(&file, &bindings, &jsonOut))
	root.AddCommand(newCapabilitiesCmd(&file, &bindings, &jsonOut))

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newValidateCmd(file *string, jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate pade.yaml against the PADE schema",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, res, err := loadAndValidate(*file)
			if err != nil {
				return err
			}
			if *jsonOut {
				if err := output.WriteJSON(cmd.OutOrStdout(), res); err != nil {
					return err
				}
			} else {
				output.WriteValidateHuman(cmd.OutOrStdout(), res)
			}
			if !res.Valid {
				return fmt.Errorf("validation failed")
			}
			return nil
		},
	}
}

func newPlanCmd(file, bindings *string, jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "plan",
		Short: "Show a side-effect-free execution plan for the manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, res, err := loadAndValidate(*file)
			if err != nil {
				return err
			}
			if !res.Valid {
				if !*jsonOut {
					output.WriteValidateHuman(cmd.OutOrStdout(), res)
				}
				return fmt.Errorf("validation failed; fix the manifest before planning")
			}
			cfg, statuses, err := resolveBindings(cmd.Context(), m, *bindings)
			if err != nil {
				return err
			}
			plan := planner.Build(m, planner.BuildOptions{Bindings: cfg, Statuses: statuses})
			if *jsonOut {
				return output.WriteJSON(cmd.OutOrStdout(), plan)
			}
			output.WritePlanHuman(cmd.OutOrStdout(), plan)
			return nil
		},
	}
}

func newCapabilitiesCmd(file, bindings *string, jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "capabilities",
		Short: "Show declared capabilities and local binding status (never secret values)",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, res, err := loadAndValidate(*file)
			if err != nil {
				return err
			}
			if !res.Valid {
				if !*jsonOut {
					output.WriteValidateHuman(cmd.OutOrStdout(), res)
				}
				return fmt.Errorf("validation failed; fix the manifest before inspecting capabilities")
			}
			cfg, statuses, err := resolveBindings(cmd.Context(), m, *bindings)
			if err != nil {
				return err
			}
			path := ""
			if cfg != nil {
				path = cfg.SourcePath
			}
			if *jsonOut {
				return output.WriteJSON(cmd.OutOrStdout(), map[string]any{
					"bindingsPath": path,
					"capabilities": statuses,
				})
			}
			output.WriteCapabilitiesHuman(cmd.OutOrStdout(), statuses, path)
			return nil
		},
	}
}

func loadAndValidate(file string) (*manifest.Manifest, *manifest.Result, error) {
	path, err := manifest.Find("", file)
	if err != nil {
		return nil, nil, err
	}
	m, err := manifest.Load(path)
	if err != nil {
		return nil, nil, err
	}
	res, err := manifest.Validate(m)
	if err != nil {
		return nil, nil, err
	}
	return m, res, nil
}

func resolveBindings(ctx context.Context, m *manifest.Manifest, bindingsPath string) (*binding.Config, []binding.Status, error) {
	cfg, err := binding.LoadOptional(filepath.Dir(m.SourcePath), bindingsPath)
	if err != nil {
		return nil, nil, err
	}
	reg := binding.NewRegistry(envprovider.New(), vaultprovider.New())
	views := map[string]binding.CapabilityRequestView{}
	for name, cap := range m.Capabilities {
		views[name] = binding.CapabilityRequestView{
			Access:   cap.Access,
			Required: cap.IsRequired(),
		}
	}
	statuses, err := binding.ResolveAll(ctx, reg, views, cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfg, statuses, nil
}
