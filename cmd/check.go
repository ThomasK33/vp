package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ThomasK33/vp/internal/config"
	"github.com/ThomasK33/vp/internal/git"
	"github.com/ThomasK33/vp/internal/output"
	"github.com/ThomasK33/vp/internal/plan"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate that every affected component is covered by a pending plan.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		base, err := cmd.Flags().GetString("base")
		if err != nil {
			return err
		}
		head, err := cmd.Flags().GetString("head")
		if err != nil {
			return err
		}
		if base == "" || head == "" {
			return usageError(errors.New("--base and --head are required"))
		}

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

		changed, err := git.ChangedFiles(cfg.Dir, base, head)
		if err != nil {
			return err
		}

		affected, err := cfg.Affected(changed)
		if err != nil {
			return usageError(err)
		}

		pending, err := plan.LoadDir(cfg.Plans.Dir)
		if err != nil {
			return usageError(err)
		}
		planned := plannedComponents(pending)

		var missing []string
		for _, name := range affected {
			if !planned[name] {
				missing = append(missing, name)
			}
		}

		jsonOut, err := cmd.Flags().GetBool("json")
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		plannedNames := sortedKeys(planned)
		if jsonOut {
			if err := output.WriteCheck(out, &output.CheckReport{
				Affected: affected,
				Planned:  plannedNames,
				Missing:  missing,
			}); err != nil {
				return err
			}
		} else {
			printCheckReport(out, affected, plannedNames, missing)
		}
		if len(missing) > 0 {
			return checkError(fmt.Errorf("missing plan coverage for: %s", strings.Join(missing, ", ")))
		}
		return nil
	},
}

func plannedComponents(pending []plan.Pending) map[string]bool {
	m := make(map[string]bool, len(pending))
	for _, p := range pending {
		for name := range p.Plan.Releases {
			m[name] = true
		}
	}
	return m
}

func printCheckReport(out io.Writer, affected, planned, missing []string) {
	_, _ = fmt.Fprintf(out, "Affected components: %s\n", joinOrNone(affected))
	_, _ = fmt.Fprintf(out, "Planned components:  %s\n", joinOrNone(planned))
	if len(missing) > 0 {
		_, _ = fmt.Fprintf(out, "Missing coverage:    %s\n", strings.Join(missing, ", "))
		return
	}
	_, _ = fmt.Fprintln(out, "All affected components are covered.")
}

func joinOrNone(s []string) string {
	if len(s) == 0 {
		return "(none)"
	}
	return strings.Join(s, ", ")
}

func init() {
	checkCmd.Flags().String("base", "", "git ref to compare from (required)")
	checkCmd.Flags().String("head", "", "git ref to compare to (required)")
	checkCmd.Flags().Bool("json", false, "emit machine-readable JSON to stdout")
	rootCmd.AddCommand(checkCmd)
}
