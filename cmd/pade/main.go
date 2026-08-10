package main

import (
	"fmt"
	"os"

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
		file    string
		jsonOut bool
	)

	root.PersistentFlags().StringVarP(&file, "file", "f", "", "path to pade.yaml (default: ./pade.yaml)")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")

	root.AddCommand(newValidateCmd(&file, &jsonOut))
	root.AddCommand(newPlanCmd(&file, &jsonOut))

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

func newPlanCmd(file *string, jsonOut *bool) *cobra.Command {
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
			plan := planner.Build(m)
			if *jsonOut {
				return output.WriteJSON(cmd.OutOrStdout(), plan)
			}
			output.WritePlanHuman(cmd.OutOrStdout(), plan)
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
