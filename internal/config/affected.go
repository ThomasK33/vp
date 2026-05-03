package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Affected returns the sorted set of component names whose path globs match at
// least one entry in changedPaths. Paths under cfg.Plans.Dir and the vp.yaml
// file itself are ignored — planning churn never triggers a coverage check.
//
// changedPaths must be relative to cfg.Dir (the directory containing vp.yaml).
func (cfg *Config) Affected(changedPaths []string) ([]string, error) {
	plansRel, err := filepath.Rel(cfg.Dir, cfg.Plans.Dir)
	if err != nil {
		return nil, fmt.Errorf("plans.dir: %w", err)
	}
	plansRel = filepath.ToSlash(plansRel)

	seen := make(map[string]bool)
	for _, raw := range changedPaths {
		p := filepath.ToSlash(filepath.Clean(raw))
		if p == Filename {
			continue
		}
		if isUnderDir(p, plansRel) {
			continue
		}
		for name, comp := range cfg.Components {
			if seen[name] {
				continue
			}
			for _, glob := range comp.Paths {
				ok, err := doublestar.Match(filepath.ToSlash(glob), p)
				if err != nil {
					return nil, fmt.Errorf("component %s: bad glob %q: %w", name, glob, err)
				}
				if ok {
					seen[name] = true
					break
				}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func isUnderDir(p, dir string) bool {
	if dir == "" || dir == "." {
		return false
	}
	return p == dir || strings.HasPrefix(p, dir+"/")
}
