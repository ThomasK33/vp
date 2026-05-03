package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/ThomasK33/vp/internal/config"
	"github.com/ThomasK33/vp/internal/plan"
	"github.com/ThomasK33/vp/internal/semver"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Print pending plans and resolved bumps.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		cfg, err := config.Load(cwd)
		if err != nil {
			if errors.Is(err, config.ErrNotFound) {
				return usageError(err)
			}
			return err
		}

		pending, err := plan.LoadDir(cfg.Plans.Dir)
		if err != nil {
			return usageError(err)
		}

		showAll, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		// Collect bumps in a first pass so an unknown component fails before
		// any output is written.
		levels := map[string][]semver.Bump{}
		for _, p := range pending {
			base := filepath.Base(p.Path)
			for name, raw := range p.Plan.Releases {
				if _, ok := cfg.Components[name]; !ok {
					return usageError(fmt.Errorf("plan %s: unknown component %q", base, name))
				}
				b, err := semver.ParseBump(raw)
				if err != nil {
					return usageError(fmt.Errorf("plan %s: %w", base, err))
				}
				levels[name] = append(levels[name], b)
			}
		}

		if len(pending) == 0 {
			fmt.Fprintln(out, "No pending plans.")
		} else {
			fmt.Fprintln(out, "Pending plans:")
			for _, p := range pending {
				fmt.Fprintf(out, "  %s\n", filepath.Base(p.Path))
				for _, n := range sortedKeys(p.Plan.Releases) {
					fmt.Fprintf(out, "    %s: %s\n", n, p.Plan.Releases[n])
				}
			}
			fmt.Fprintln(out)
		}
		if len(pending) > 0 || showAll {
			printSummary(out, cfg, levels, showAll)
		}
		return nil
	},
}

func printSummary(out io.Writer, cfg *config.Config, levels map[string][]semver.Bump, showAll bool) {
	names := sortedKeys(levels)
	if showAll {
		names = sortedKeys(cfg.Components)
	}
	fmt.Fprintln(out, "Resolved bumps:")
	for _, n := range names {
		fmt.Fprintf(out, "  %s: %s\n", n, semver.Collapse(levels[n]))
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func init() {
	statusCmd.Flags().Bool("all", false, "show every component from config, not just those with pending bumps")
	rootCmd.AddCommand(statusCmd)
}
