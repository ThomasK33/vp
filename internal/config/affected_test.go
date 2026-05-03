package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ThomasK33/vp/internal/config"
)

// loadCfg writes body to <dir>/vp.yaml and loads it.
func loadCfg(t *testing.T, body string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg
}

const oneComponentYAML = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["cli/**"]
    version: {file: VERSION, format: text}
`

func TestAffected_SingleComponentSingleMatch(t *testing.T) {
	cfg := loadCfg(t, oneComponentYAML)

	got, err := cfg.Affected([]string{"cli/main.go"})
	if err != nil {
		t.Fatalf("Affected: %v", err)
	}
	want := []string{"cli"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Affected = %v, want %v", got, want)
	}
}

const overlappingComponentsYAML = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["cli/**", "shared/**"]
    version: {file: VERSION, format: text}
  agent:
    paths: ["agent/**", "shared/**"]
    version: {file: AGENT_VERSION, format: text}
`

func TestAffected_OverlappingGlobsAffectMultipleComponents(t *testing.T) {
	cfg := loadCfg(t, overlappingComponentsYAML)

	got, err := cfg.Affected([]string{"shared/util.go"})
	if err != nil {
		t.Fatalf("Affected: %v", err)
	}
	want := []string{"agent", "cli"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Affected = %v, want %v", got, want)
	}
}

// Catch-all globs would otherwise match plan files and vp.yaml; Affected
// must filter those out so planning churn never triggers a coverage check.
const catchAllYAML = `
plans:
  dir: .version-plans
  consumed: delete
components:
  any:
    paths: ["**"]
    version: {file: VERSION, format: text}
`

func TestAffected_IgnoresPlanFiles(t *testing.T) {
	cfg := loadCfg(t, catchAllYAML)

	got, err := cfg.Affected([]string{".version-plans/2026-05-03-fix.yaml"})
	if err != nil {
		t.Fatalf("Affected: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Affected = %v, want empty (plan files must be ignored)", got)
	}
}

func TestAffected_IgnoresVpYaml(t *testing.T) {
	cfg := loadCfg(t, catchAllYAML)

	got, err := cfg.Affected([]string{"vp.yaml"})
	if err != nil {
		t.Fatalf("Affected: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Affected = %v, want empty (vp.yaml must be ignored)", got)
	}
}

// When the plans dir lives at a non-default nested path, Affected must still
// strip changes under it. The doublestar `**` glob would otherwise match.
const nestedPlansDirYAML = `
plans:
  dir: chart/.plans
  consumed: delete
components:
  any:
    paths: ["**"]
    version: {file: VERSION, format: text}
`

func TestAffected_IgnoresNestedPlansDir(t *testing.T) {
	cfg := loadCfg(t, nestedPlansDirYAML)

	got, err := cfg.Affected([]string{"chart/.plans/2026-05-03-fix.yaml"})
	if err != nil {
		t.Fatalf("Affected: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Affected = %v, want empty (nested plans dir must be ignored)", got)
	}
}

func TestAffected_NoMatchReturnsEmpty(t *testing.T) {
	cfg := loadCfg(t, oneComponentYAML)

	got, err := cfg.Affected([]string{"docs/README.md"})
	if err != nil {
		t.Fatalf("Affected: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Affected = %v, want empty", got)
	}
}

func TestAffected_BadGlobReturnsError(t *testing.T) {
	const badGlobYAML = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["[unclosed"]
    version: {file: VERSION, format: text}
`
	cfg := loadCfg(t, badGlobYAML)

	_, err := cfg.Affected([]string{"cli/main.go"})
	if err == nil {
		t.Fatal("Affected: want error for bad glob, got nil")
	}
}
