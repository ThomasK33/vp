package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate that every affected component is covered by a pending plan.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
