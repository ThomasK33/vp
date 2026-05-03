package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ThomasK33/vp/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a starter vp.yaml.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if !force {
			if existing, err := config.FindUpwards(cwd); err == nil {
				return usageError(fmt.Errorf("vp.yaml already exists at %s; pass --force to overwrite", existing))
			} else if !errors.Is(err, config.ErrNotFound) {
				return err
			}
		}
		target := filepath.Join(cwd, config.Filename)
		if err := os.WriteFile(target, config.Starter(), 0o644); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", target)
		return nil
	},
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite an existing vp.yaml in the current directory")
	rootCmd.AddCommand(initCmd)
}
