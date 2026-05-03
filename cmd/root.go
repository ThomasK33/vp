// Package cmd wires the vp Cobra command tree.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ThomasK33/vp/internal/version"
)

// Exit codes follow the convention documented in the PRD:
//
//	0 - success
//	1 - vp check failure (missing plans)
//	2 - usage / config error
//	3 - runtime error
const exitRuntimeError = 3

var rootCmd = &cobra.Command{
	Use:     "vp",
	Short:   "Stage semver bump intent in Git.",
	Long:    "vp stages plans that collapse into version-file updates at release time. It deliberately does not generate changelogs or release notes.",
	Version: version.Version,
	// Stub failures shouldn't dump the full help text.
	SilenceUsage: true,
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		return exitRuntimeError
	}
	return 0
}
