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
	"github.com/ThomasK33/vp/internal/output"
	"github.com/ThomasK33/vp/internal/plan"
	"github.com/ThomasK33/vp/internal/semver"
	jsonvf "github.com/ThomasK33/vp/internal/versionfile/json"
	"github.com/ThomasK33/vp/internal/versionfile/text"
	tomlvf "github.com/ThomasK33/vp/internal/versionfile/toml"
	yamlvf "github.com/ThomasK33/vp/internal/versionfile/yaml"
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
		jsonOut, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if jsonOut {
			return output.WriteStatus(out, buildStatusReport(cfg, pending, collapsed, showAll))
		}

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

func buildStatusReport(cfg *config.Config, pending []plan.Pending, collapsed map[string]semver.Bump, showAll bool) *output.StatusReport {
	report := &output.StatusReport{
		Pending:  make([]output.PendingPlan, 0, len(pending)),
		Resolved: []output.ResolvedComponent{},
	}
	for _, p := range pending {
		report.Pending = append(report.Pending, output.PendingPlan{
			File:     filepath.Base(p.Path),
			Releases: p.Plan.Releases,
			Message:  p.Plan.Message,
		})
	}
	names := sortedKeys(collapsed)
	if showAll {
		names = sortedKeys(cfg.Components)
	}
	for _, n := range names {
		level := collapsed[n]
		if level == "" {
			level = semver.BumpNone
		}
		entry := output.ResolvedComponent{Component: n, Bump: string(level)}
		if cur, ok := readVersion(cfg, n); ok {
			entry.Current = cur
			if level != semver.BumpNone {
				if next, err := semver.Next(cur, level); err == nil {
					entry.Next = next
				}
			}
		}
		report.Resolved = append(report.Resolved, entry)
	}
	return report
}

// readVersion returns the current version for the named component when its
// version file is readable. The boolean is false when the format is unknown,
// the file is missing, or the format-specific parser errors out — mirroring
// the silent-fallback policy of the text-output path.
func readVersion(cfg *config.Config, name string) (string, bool) {
	comp, ok := cfg.Components[name]
	if !ok {
		return "", false
	}
	var (
		v   string
		err error
	)
	switch comp.Version.Format {
	case config.FormatText:
		v, err = text.Read(comp.Version.File)
	case config.FormatJSON:
		v, err = jsonvf.Read(comp.Version.File, comp.Version.Path)
	case config.FormatYAML:
		v, err = yamlvf.Read(comp.Version.File, comp.Version.Path)
	case config.FormatTOML:
		v, err = tomlvf.Read(comp.Version.File, comp.Version.Path)
	default:
		return "", false
	}
	if err != nil {
		return "", false
	}
	return v, true
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

// formatSummaryLine renders one resolved-bump row. When the component's
// version file is readable in any supported format, it appends current/next
// (or current alone for BumpNone); otherwise it falls back to "name: bump".
func formatSummaryLine(cfg *config.Config, name string, level semver.Bump) string {
	bare := fmt.Sprintf("%s: %s", name, level)
	cur, ok := readVersion(cfg, name)
	if !ok {
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
	statusCmd.Flags().Bool("json", false, "emit machine-readable JSON to stdout")
	rootCmd.AddCommand(statusCmd)
}
