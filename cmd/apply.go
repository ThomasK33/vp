package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Collapse pending plans, update version files, and consume the plans.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
