package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThomasK33/vp/internal/plan"
)

const addTestConfig = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["cli/**"]
    version: {file: cli/package.json, format: json}
  agent:
    paths: ["agent/**"]
    version: {file: agent/Cargo.toml, format: toml, path: package.version}
`

// writeAddConfig writes a minimal vp.yaml declaring cli and agent components.
func writeAddConfig(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(addTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}
}

// onePlanFile returns the single .yaml plan file in plansDir, failing otherwise.
func onePlanFile(t *testing.T, plansDir string) string {
	t.Helper()
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		t.Fatalf("read plans dir: %v", err)
	}
	var planFile string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			if planFile != "" {
				t.Fatalf("expected exactly one plan file, got at least 2: %s, %s", planFile, e.Name())
			}
			planFile = e.Name()
		}
	}
	if planFile == "" {
		t.Fatal("no plan files found")
	}
	return filepath.Join(plansDir, planFile)
}

func TestAdd_RequiresConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// no vp.yaml

	_, _, err := runVP(t, "add", "cli", "minor")
	if err == nil {
		t.Fatal("vp add: want error, got nil")
	}
	if coded, ok := errors.AsType[*exitCodeError](err); !ok || coded.code != exitUsageError {
		t.Fatalf("vp add error = %v (code want %d)", err, exitUsageError)
	}
}

// assertUsageError checks runVP returned a usage-coded error.
func assertUsageError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("want error, got nil")
	}
	coded, ok := errors.AsType[*exitCodeError](err)
	if !ok || coded.code != exitUsageError {
		t.Fatalf("error = %v (code want %d)", err, exitUsageError)
	}
}

func TestAdd_RejectsUnknownComponent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	_, _, err := runVP(t, "add", "nope", "minor")
	assertUsageError(t, err)
}

func TestAdd_RejectsUnknownBump(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	_, _, err := runVP(t, "add", "cli", "foo")
	assertUsageError(t, err)
}

func TestAdd_RejectsDuplicateComponent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	_, _, err := runVP(t, "add", "cli=minor", "cli=patch")
	assertUsageError(t, err)
}

func TestAdd_RejectsMixedArgForms(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	_, _, err := runVP(t, "add", "cli", "minor=patch")
	assertUsageError(t, err)
}

func TestAdd_RejectsThreeBareArgs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	_, _, err := runVP(t, "add", "cli", "minor", "patch")
	assertUsageError(t, err)
}

func TestAdd_RejectsEmptyKey(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	_, _, err := runVP(t, "add", "=minor")
	assertUsageError(t, err)
}

func TestAdd_RejectsEmptyValue(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	_, _, err := runVP(t, "add", "cli=")
	assertUsageError(t, err)
}

func TestAdd_FromSubdirectoryWritesAtConfigRoot(t *testing.T) {
	root := t.TempDir()
	writeAddConfig(t, root)
	sub := filepath.Join(root, "sub", "deeper")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	if _, _, err := runVP(t, "add", "cli", "minor"); err != nil {
		t.Fatalf("vp add: %v", err)
	}

	planFile := onePlanFile(t, filepath.Join(root, ".version-plans"))
	if _, err := os.Stat(planFile); err != nil {
		t.Fatalf("plan file not at config root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sub, ".version-plans")); !os.IsNotExist(err) {
		t.Errorf("plans dir created in subdir, want only at config root")
	}
}

func TestAdd_FallsBackToComponentsSlugWhenNoMessage(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	if _, _, err := runVP(t, "add", "cli=minor", "agent=patch"); err != nil {
		t.Fatalf("vp add: %v", err)
	}

	planFile := onePlanFile(t, filepath.Join(dir, ".version-plans"))
	if !strings.HasSuffix(filepath.Base(planFile), "-add-agent-cli.yaml") {
		t.Errorf("plan filename = %q, want suffix -add-agent-cli.yaml", filepath.Base(planFile))
	}
}

func TestAdd_RefusesSameDayOverwrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	if _, _, err := runVP(t, "add", "cli", "minor"); err != nil {
		t.Fatalf("first vp add: %v", err)
	}
	planFile := onePlanFile(t, filepath.Join(dir, ".version-plans"))
	originalBytes, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}

	_, _, err = runVP(t, "add", "cli", "minor")
	if err == nil {
		t.Fatal("second vp add: want error, got nil")
	}

	got, err := os.ReadFile(planFile)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if string(got) != string(originalBytes) {
		t.Errorf("plan file modified by failed second add")
	}
}

func TestAdd_KeyValueMultiWithMessage(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	if _, _, err := runVP(t, "add", "cli=minor", "agent=patch", "-m", "Fix reconnect"); err != nil {
		t.Fatalf("vp add: %v", err)
	}

	planFile := onePlanFile(t, filepath.Join(dir, ".version-plans"))
	if !strings.HasSuffix(planFile, "-fix-reconnect.yaml") {
		t.Errorf("plan filename = %q, want suffix -fix-reconnect.yaml", filepath.Base(planFile))
	}
	p, err := plan.Load(planFile)
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	if got, want := p.Releases["cli"], "minor"; got != want {
		t.Errorf("Releases[cli] = %q, want %q", got, want)
	}
	if got, want := p.Releases["agent"], "patch"; got != want {
		t.Errorf("Releases[agent] = %q, want %q", got, want)
	}
	if got, want := p.Message, "Fix reconnect"; got != want {
		t.Errorf("Message = %q, want %q", got, want)
	}
}

func TestAdd_BareSinglePair(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeAddConfig(t, dir)

	if _, _, err := runVP(t, "add", "cli", "minor"); err != nil {
		t.Fatalf("vp add cli minor: %v", err)
	}

	planFile := onePlanFile(t, filepath.Join(dir, ".version-plans"))
	p, err := plan.Load(planFile)
	if err != nil {
		t.Fatalf("plan.Load: %v", err)
	}
	if got, want := p.Releases["cli"], "minor"; got != want {
		t.Errorf("Releases[cli] = %q, want %q", got, want)
	}
	if len(p.Releases) != 1 {
		t.Errorf("Releases = %v, want exactly 1 entry", p.Releases)
	}
}
