package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <component> <bump>",
	Short: "Add a plan file declaring component bumps.",
	RunE: func(_ *cobra.Command, _ []string) error {
		return errors.New("not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
