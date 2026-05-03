package config_test

import (
	"strings"
	"testing"

	"github.com/ThomasK33/vp/internal/config"
)

// loadString writes contents to vp.yaml in a t.TempDir() and returns the result
// of config.Load.
func loadString(t *testing.T, contents string) (*config.Config, error) {
	t.Helper()
	dir := t.TempDir()
	writeConfig(t, dir, contents)
	return config.Load(dir)
}

func TestLoad_ValidationErrors(t *testing.T) {
	cases := []struct {
		name     string
		yaml     string
		wantSubs string // substring expected in the error message
	}{
		{
			name: "unknown plans.consumed",
			yaml: `
plans:
  consumed: yeet
components:
  cli:
    paths: ["cli/**"]
    version: {file: cli/package.json, format: json}
`,
			wantSubs: "plans.consumed",
		},
		{
			name: "archive without archive_dir",
			yaml: `
plans:
  consumed: archive
components:
  cli:
    paths: ["cli/**"]
    version: {file: cli/package.json, format: json}
`,
			wantSubs: "archive_dir",
		},
		{
			name: "empty components",
			yaml: `
plans: {consumed: delete}
components: {}
`,
			wantSubs: "components",
		},
		{
			name: "component without paths",
			yaml: `
plans: {consumed: delete}
components:
  cli:
    version: {file: cli/package.json, format: json}
`,
			wantSubs: "paths",
		},
		{
			name: "missing version.file",
			yaml: `
plans: {consumed: delete}
components:
  cli:
    paths: ["cli/**"]
    version: {format: json}
`,
			wantSubs: "version.file",
		},
		{
			name: "missing version.format",
			yaml: `
plans: {consumed: delete}
components:
  cli:
    paths: ["cli/**"]
    version: {file: cli/package.json}
`,
			wantSubs: "version.format",
		},
		{
			name: "unknown version.format",
			yaml: `
plans: {consumed: delete}
components:
  cli:
    paths: ["cli/**"]
    version: {file: cli/package.json, format: ini}
`,
			wantSubs: "version.format",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadString(t, tc.yaml)
			if err == nil {
				t.Fatalf("Load: want error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSubs) {
				t.Errorf("error %q does not contain %q", err, tc.wantSubs)
			}
		})
	}
}
