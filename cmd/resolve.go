package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/ThomasK33/vp/internal/config"
	"github.com/ThomasK33/vp/internal/plan"
	"github.com/ThomasK33/vp/internal/semver"
)

// resolveBumps loads pending plans, validates that every release names a known
// component and a known bump, and returns the collapsed bump per component.
// All validation failures are wrapped as usage errors with the offending plan
// filename in the message.
func resolveBumps(cfg *config.Config) (pending []plan.Pending, collapsed map[string]semver.Bump, err error) {
	pending, err = plan.LoadDir(cfg.Plans.Dir)
	if err != nil {
		return nil, nil, usageError(err)
	}
	levels := map[string][]semver.Bump{}
	for _, p := range pending {
		base := filepath.Base(p.Path)
		for name, raw := range p.Plan.Releases {
			if _, ok := cfg.Components[name]; !ok {
				return nil, nil, usageError(fmt.Errorf("plan %s: unknown component %q", base, name))
			}
			b, err := semver.ParseBump(raw)
			if err != nil {
				return nil, nil, usageError(fmt.Errorf("plan %s: %w", base, err))
			}
			levels[name] = append(levels[name], b)
		}
	}
	collapsed = make(map[string]semver.Bump, len(levels))
	for name, bs := range levels {
		collapsed[name] = semver.Collapse(bs)
	}
	return pending, collapsed, nil
}
