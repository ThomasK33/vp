package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/ThomasK33/vp/internal/config"
	"github.com/ThomasK33/vp/internal/semver"
	"github.com/ThomasK33/vp/internal/versionfile/text"
)

type plannedChange struct {
	name    string
	bump    semver.Bump
	current string
	next    string
	file    string
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

		if cfg.Plans.Consumed == config.ConsumedArchive {
			return errors.New("archive consumption mode not yet implemented (planned for a later release)")
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

		out := cmd.OutOrStdout()
		if dryRun {
			renderDryRun(out, cfg, changes)
			return nil
		}

		if len(changes) == 0 {
			_, _ = fmt.Fprintln(out, "no version changes")
		} else {
			renderApplied(out, cfg, changes)
		}

		for _, ch := range changes {
			if err := text.Write(ch.file, ch.next); err != nil {
				return err
			}
		}
		for _, p := range pending {
			if err := os.Remove(p.Path); err != nil {
				return err
			}
		}

		_, _ = fmt.Fprintf(out, "Wrote %d file(s); consumed %d plan(s).\n", len(changes), len(pending))
		return nil
	},
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
		if comp.Version.Format != config.FormatText {
			return nil, fmt.Errorf(
				"component %q: version file format %q not yet supported (text only in this release)",
				name, comp.Version.Format,
			)
		}
		cur, err := text.Read(comp.Version.File)
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
		})
	}
	return changes, nil
}

func renderDryRun(out io.Writer, cfg *config.Config, changes []plannedChange) {
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(out, "no version changes")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "component\tcurrent\tnext\tbump\tfile")
	for _, ch := range changes {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			ch.name, ch.current, ch.next, ch.bump, relPath(cfg.Dir, ch.file))
	}
	_ = tw.Flush()
}

func renderApplied(out io.Writer, cfg *config.Config, changes []plannedChange) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, ch := range changes {
		_, _ = fmt.Fprintf(tw, "%s\t%s -> %s\t(%s)\t%s\n",
			ch.name, ch.current, ch.next, ch.bump, relPath(cfg.Dir, ch.file))
	}
	_ = tw.Flush()
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
	rootCmd.AddCommand(applyCmd)
}
