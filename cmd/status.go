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
	"github.com/ThomasK33/vp/internal/semver"
	"github.com/ThomasK33/vp/internal/versionfile/text"
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

		pending, collapsed, err := resolveBumps(cfg)
		if err != nil {
			return err
		}

		showAll, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if len(pending) == 0 {
			_, _ = fmt.Fprintln(out, "No pending plans.")
		} else {
			_, _ = fmt.Fprintln(out, "Pending plans:")
			for _, p := range pending {
				_, _ = fmt.Fprintf(out, "  %s\n", filepath.Base(p.Path))
				for _, n := range sortedKeys(p.Plan.Releases) {
					_, _ = fmt.Fprintf(out, "    %s: %s\n", n, p.Plan.Releases[n])
				}
			}
			_, _ = fmt.Fprintln(out)
		}
		if len(pending) > 0 || showAll {
			printSummary(out, cfg, collapsed, showAll)
		}
		return nil
	},
}

func printSummary(out io.Writer, cfg *config.Config, collapsed map[string]semver.Bump, showAll bool) {
	names := sortedKeys(collapsed)
	if showAll {
		names = sortedKeys(cfg.Components)
	}
	_, _ = fmt.Fprintln(out, "Resolved bumps:")
	for _, n := range names {
		level := collapsed[n]
		if level == "" {
			level = semver.BumpNone
		}
		_, _ = fmt.Fprintf(out, "  %s\n", formatSummaryLine(cfg, n, level))
	}
}

// formatSummaryLine renders one resolved-bump row. For text-format components
// it appends current/next version info; for any other format (or when reading
// the version file fails) it falls back to the bare "name: bump" form.
func formatSummaryLine(cfg *config.Config, name string, level semver.Bump) string {
	bare := fmt.Sprintf("%s: %s", name, level)
	comp, ok := cfg.Components[name]
	if !ok || comp.Version.Format != config.FormatText {
		return bare
	}
	cur, err := text.Read(comp.Version.File)
	if err != nil {
		return bare
	}
	if level == semver.BumpNone {
		return fmt.Sprintf("%s (%s)", bare, cur)
	}
	next, err := semver.Next(cur, level)
	if err != nil {
		return bare
	}
	return fmt.Sprintf("%s (%s -> %s)", bare, cur, next)
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
