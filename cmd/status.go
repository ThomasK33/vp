package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print pending plans, resolved bumps, and current/next versions.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
