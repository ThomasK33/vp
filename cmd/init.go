package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a starter vp.yaml.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
