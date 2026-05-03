package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Pending pairs a loaded plan with the file it came from.
type Pending struct {
	Path string // path to the YAML file, joined under the dir argument to LoadDir
	Plan *Plan
}

// LoadDir loads every *.yaml file in dir as a Plan, sorted by base filename.
// A missing directory returns (nil, nil): zero pending plans, no error.
func LoadDir(dir string) ([]Pending, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var out []Pending
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		p, err := Load(path)
		if err != nil {
			return nil, err
		}
		if err := validate(p); err != nil {
			return nil, fmt.Errorf("validate %s: %w", path, err)
		}
		out = append(out, Pending{Path: path, Plan: p})
	}
	sort.Slice(out, func(i, j int) bool {
		return filepath.Base(out[i].Path) < filepath.Base(out[j].Path)
	})
	return out, nil
}
