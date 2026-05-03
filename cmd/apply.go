package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

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

type plannedChange struct {
	name    string
	bump    semver.Bump
	current string
	next    string
	file    string
	format  string
	path    string
	tag     string
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Collapse pending plans, update version files, and consume the plans.",
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

		changes, err := planChanges(cfg, collapsed)
		if err != nil {
			return err
		}

		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		jsonOut, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if dryRun {
			if jsonOut {
				return output.WriteApply(out, applyReport(cfg, changes, nil))
			}
			renderDryRun(out, cfg, changes)
			return nil
		}

		if !jsonOut {
			if len(changes) == 0 {
				_, _ = fmt.Fprintln(out, "no version changes")
			} else {
				renderApplied(out, cfg, changes)
			}
		}

		for _, ch := range changes {
			var err error
			switch ch.format {
			case config.FormatText:
				err = text.Write(ch.file, ch.next)
			case config.FormatJSON:
				err = jsonvf.Write(ch.file, ch.path, ch.next)
			case config.FormatYAML:
				err = yamlvf.Write(ch.file, ch.path, ch.next)
			case config.FormatTOML:
				err = tomlvf.Write(ch.file, ch.path, ch.next)
			default:
				return fmt.Errorf("component %q: unsupported version format %q", ch.name, ch.format)
			}
			if err != nil {
				return err
			}
		}
		if err := consumePlans(cfg, pending); err != nil {
			return err
		}

		if jsonOut {
			return output.WriteApply(out, applyReport(cfg, changes, pending))
		}
		_, _ = fmt.Fprintf(out, "Wrote %d file(s); consumed %d plan(s).\n", len(changes), len(pending))
		return nil
	},
}

// applyReport builds the JSON payload for both dry-run and live apply.
// Pass consumed=nil for dry-run; the result's Consumed array is then empty.
func applyReport(cfg *config.Config, changes []plannedChange, consumed []plan.Pending) *output.ApplyReport {
	r := &output.ApplyReport{
		Changes:  make([]output.Change, 0, len(changes)),
		Consumed: []string{},
	}
	for _, ch := range changes {
		r.Changes = append(r.Changes, output.Change{
			Component: ch.name,
			Current:   ch.current,
			Next:      ch.next,
			Bump:      string(ch.bump),
			File:      relPath(cfg.Dir, ch.file),
			Tag:       ch.tag,
		})
	}
	for _, p := range consumed {
		r.Consumed = append(r.Consumed, filepath.Base(p.Path))
	}
	return r
}

// planChanges turns collapsed bumps into plannedChange entries, validating
// component formats and version files. Components whose collapsed bump is
// BumpNone produce no entry (no read or write needed).
func planChanges(cfg *config.Config, collapsed map[string]semver.Bump) ([]plannedChange, error) {
	var changes []plannedChange
	for _, name := range sortedKeys(collapsed) {
		level := collapsed[name]
		if level == semver.BumpNone {
			continue
		}
		comp := cfg.Components[name]
		var (
			cur string
			err error
		)
		switch comp.Version.Format {
		case config.FormatText:
			cur, err = text.Read(comp.Version.File)
		case config.FormatJSON:
			cur, err = jsonvf.Read(comp.Version.File, comp.Version.Path)
		case config.FormatYAML:
			cur, err = yamlvf.Read(comp.Version.File, comp.Version.Path)
		case config.FormatTOML:
			cur, err = tomlvf.Read(comp.Version.File, comp.Version.Path)
		default:
			return nil, fmt.Errorf("component %q: unsupported version format %q", name, comp.Version.Format)
		}
		if err != nil {
			return nil, err
		}
		next, err := semver.Next(cur, level)
		if err != nil {
			return nil, fmt.Errorf("component %q: %w", name, err)
		}
		changes = append(changes, plannedChange{
			name:    name,
			bump:    level,
			current: cur,
			next:    next,
			file:    comp.Version.File,
			format:  comp.Version.Format,
			path:    comp.Version.Path,
			tag:     renderTag(comp.Tag, next),
		})
	}
	return changes, nil
}

func renderDryRun(out io.Writer, cfg *config.Config, changes []plannedChange) {
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(out, "no version changes")
		return
	}
	withTag := hasTag(changes)
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	header := "component\tcurrent\tnext\tbump\tfile"
	if withTag {
		header += "\ttag"
	}
	_, _ = fmt.Fprintln(tw, header)
	for _, ch := range changes {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s",
			ch.name, ch.current, ch.next, ch.bump, relPath(cfg.Dir, ch.file))
		if withTag {
			_, _ = fmt.Fprintf(tw, "\t%s", ch.tag)
		}
		_, _ = fmt.Fprintln(tw)
	}
	_ = tw.Flush()
}

func renderApplied(out io.Writer, cfg *config.Config, changes []plannedChange) {
	withTag := hasTag(changes)
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, ch := range changes {
		_, _ = fmt.Fprintf(tw, "%s\t%s -> %s\t(%s)\t%s",
			ch.name, ch.current, ch.next, ch.bump, relPath(cfg.Dir, ch.file))
		if withTag {
			_, _ = fmt.Fprintf(tw, "\t%s", ch.tag)
		}
		_, _ = fmt.Fprintln(tw)
	}
	_ = tw.Flush()
}

// renderTag is intentionally a single-placeholder substitution, not a template
// language: any {xxx} other than {version} is left literal.
func renderTag(template, version string) string {
	return strings.ReplaceAll(template, "{version}", version)
}

func hasTag(changes []plannedChange) bool {
	return slices.ContainsFunc(changes, func(c plannedChange) bool { return c.tag != "" })
}

func consumePlans(cfg *config.Config, pending []plan.Pending) error {
	switch cfg.Plans.Consumed {
	case config.ConsumedArchive:
		if err := os.MkdirAll(cfg.Plans.ArchiveDir, 0o755); err != nil {
			return err
		}
		for _, p := range pending {
			dst := filepath.Join(cfg.Plans.ArchiveDir, filepath.Base(p.Path))
			if err := os.Rename(p.Path, dst); err != nil {
				return err
			}
		}
		return nil
	default:
		for _, p := range pending {
			if err := os.Remove(p.Path); err != nil {
				return err
			}
		}
		return nil
	}
}

func relPath(base, p string) string {
	rel, err := filepath.Rel(base, p)
	if err != nil {
		return p
	}
	return rel
}

func init() {
	applyCmd.Flags().Bool("dry-run", false, "show planned changes without writing files or consuming plans")
	applyCmd.Flags().Bool("json", false, "emit machine-readable JSON to stdout")
	rootCmd.AddCommand(applyCmd)
}
