package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ThomasK33/vp/internal/config"
	"github.com/ThomasK33/vp/internal/plan"
)

var addCmd = &cobra.Command{
	Use:   "add <component> <bump>  |  add <component>=<bump> ...",
	Short: "Add a plan file declaring component bumps.",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		releases, err := parseAddArgs(args)
		if err != nil {
			return usageError(err)
		}
		for name := range releases {
			if _, ok := cfg.Components[name]; !ok {
				return usageError(fmt.Errorf("unknown component %q", name))
			}
		}

		message, err := cmd.Flags().GetString("message")
		if err != nil {
			return err
		}

		p, err := plan.New(releases, message)
		if err != nil {
			return usageError(err)
		}

		if err := os.MkdirAll(cfg.Plans.Dir, 0o755); err != nil {
			return err
		}
		target := filepath.Join(cfg.Plans.Dir, plan.Filename(p, time.Now().UTC()))
		if err := plan.Save(p, target); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", target)
		return nil
	},
}

// parseAddArgs accepts either a single bare pair (`cli minor`) or any number
// of `key=value` pairs. Any other shape is a usage error.
func parseAddArgs(args []string) (map[string]string, error) {
	if len(args) == 0 {
		return nil, errors.New("expected <component> <bump>, or one or more <component>=<bump> pairs")
	}
	if len(args) == 2 && !strings.Contains(args[0], "=") && !strings.Contains(args[1], "=") {
		return map[string]string{args[0]: args[1]}, nil
	}
	releases := make(map[string]string, len(args))
	for _, raw := range args {
		k, v, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("expected <component>=<bump>, got %q", raw)
		}
		if _, dup := releases[k]; dup {
			return nil, fmt.Errorf("component %q appears more than once", k)
		}
		releases[k] = v
	}
	return releases, nil
}

func init() {
	addCmd.Flags().StringP("message", "m", "", "human-readable message stored in the plan")
	rootCmd.AddCommand(addCmd)
}
