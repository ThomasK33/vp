package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ThomasK33/vp/internal/config"
)

const minimalValidYAML = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths:
      - "cli/**"
    version:
      file: "cli/package.json"
      format: "json"
      path: "version"
`

func writeConfig(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad_ParsesValidConfig(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, minimalValidYAML)

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Dir != dir {
		t.Errorf("Dir = %q, want %q", cfg.Dir, dir)
	}
	if _, ok := cfg.Components["cli"]; !ok {
		t.Errorf("Components missing key %q (got keys %v)", "cli", keys(cfg.Components))
	}
	if cfg.Plans.Consumed != "delete" {
		t.Errorf("Plans.Consumed = %q, want %q", cfg.Plans.Consumed, "delete")
	}
}

func TestLoad_ResolvesPlansDirRelativeToConfigDir(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, `
plans:
  dir: .version-plans
  consumed: archive
  archive_dir: .version-plans/archive
components:
  cli:
    paths: ["cli/**"]
    version: {file: cli/package.json, format: json}
`)

	cfg, err := config.Load(deep)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := filepath.Join(root, ".version-plans"); cfg.Plans.Dir != want {
		t.Errorf("Plans.Dir = %q, want %q", cfg.Plans.Dir, want)
	}
	if want := filepath.Join(root, ".version-plans/archive"); cfg.Plans.ArchiveDir != want {
		t.Errorf("Plans.ArchiveDir = %q, want %q", cfg.Plans.ArchiveDir, want)
	}
}

func TestLoad_ResolvesVersionFileRelativeToConfigDir(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, minimalValidYAML)

	cfg, err := config.Load(deep)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cli := cfg.Components["cli"]
	want := filepath.Join(root, "cli/package.json")
	if cli.Version.File != want {
		t.Errorf("cli.version.file = %q, want %q", cli.Version.File, want)
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
