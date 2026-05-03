package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const applyTestConfigText = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["cli/**"]
    version: {file: VERSION, format: text}
`

func writeApplyTextConfig(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(applyTestConfigText), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestApply_DryRunDoesNotTouchFilesOrPlans(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)

	versionPath := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(versionPath, []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	planPath := filepath.Join(plansDir, "2026-05-03-bump.yaml")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply", "--dry-run")
	if err != nil {
		t.Fatalf("vp apply --dry-run: %v", err)
	}

	got, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1.2.3\n" {
		t.Errorf("VERSION changed during dry-run: got %q", got)
	}
	if _, err := os.Stat(planPath); err != nil {
		t.Errorf("plan file removed during dry-run: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"component", "current", "next", "bump", "file", "cli", "1.2.3", "1.3.0", "minor", "VERSION"} {
		if !strings.Contains(out, want) {
			t.Errorf("dry-run stdout missing %q\n---\n%s", want, out)
		}
	}
}

func TestApply_AllNoneConsumesPlansWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)

	versionPath := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(versionPath, []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-noop.yaml", "releases:\n  cli: none\n")

	stdout, _, err := runVP(t, "apply")
	if err != nil {
		t.Fatalf("vp apply: %v", err)
	}

	got, _ := os.ReadFile(versionPath)
	if string(got) != "1.2.3\n" {
		t.Errorf("VERSION modified for all-none plan: got %q", got)
	}
	if _, err := os.Stat(filepath.Join(plansDir, "2026-05-03-noop.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("plan file not consumed: %v", err)
	}
	if !strings.Contains(stdout.String(), "no version changes") {
		t.Errorf("stdout missing 'no version changes':\n%s", stdout.String())
	}
}

func TestApply_RejectsArchiveConsumptionMode(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := strings.Replace(applyTestConfigText, "consumed: delete", "consumed: archive\n  archive_dir: .version-plans/archive", 1)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	_, _, err := runVP(t, "apply")
	if err == nil {
		t.Fatal("vp apply: want error, got nil")
	}
	if !strings.Contains(err.Error(), "archive") {
		t.Errorf("error = %v, want it to mention archive", err)
	}
	// Ensure no side effects.
	got, _ := os.ReadFile(filepath.Join(dir, "VERSION"))
	if string(got) != "1.2.3\n" {
		t.Errorf("VERSION modified despite archive-mode rejection: %q", got)
	}
	if _, err := os.Stat(filepath.Join(plansDir, "2026-05-03-bump.yaml")); err != nil {
		t.Errorf("plan removed despite archive-mode rejection: %v", err)
	}
}

func TestApply_UnknownComponentInPlan(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-typo.yaml", "releases:\n  nope: minor\n")

	_, _, err := runVP(t, "apply")
	assertUsageError(t, err)
	got, _ := os.ReadFile(filepath.Join(dir, "VERSION"))
	if string(got) != "1.2.3\n" {
		t.Errorf("VERSION modified despite Phase 1 failure: %q", got)
	}
	if _, err := os.Stat(filepath.Join(plansDir, "2026-05-03-typo.yaml")); err != nil {
		t.Errorf("plan removed despite Phase 1 failure: %v", err)
	}
}

func TestApply_RejectsNonTextFormatWithBump(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	_, _, err := runVP(t, "apply")
	if err == nil {
		t.Fatal("vp apply: want error, got nil")
	}
	if !strings.Contains(err.Error(), "not yet supported") {
		t.Errorf("error = %v, want it to mention 'not yet supported'", err)
	}
	if _, err := os.Stat(filepath.Join(plansDir, "2026-05-03-bump.yaml")); err != nil {
		t.Errorf("plan removed despite Phase 1 failure: %v", err)
	}
}

func TestApply_RejectsMissingVersionFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	// no VERSION file written

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	_, _, err := runVP(t, "apply")
	if err == nil {
		t.Fatal("vp apply: want error, got nil")
	}
	if _, err := os.Stat(filepath.Join(plansDir, "2026-05-03-bump.yaml")); err != nil {
		t.Errorf("plan removed despite Phase 1 failure: %v", err)
	}
}

func TestApply_NonTextFormatWithAllNoneSucceeds(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-noop.yaml", "releases:\n  cli: none\n")

	stdout, _, err := runVP(t, "apply")
	if err != nil {
		t.Fatalf("vp apply: %v", err)
	}
	if !strings.Contains(stdout.String(), "no version changes") {
		t.Errorf("stdout missing 'no version changes': %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(plansDir, "2026-05-03-noop.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("plan should have been consumed: %v", err)
	}
}

func TestApply_HappyPathWritesAndConsumes(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)

	versionPath := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(versionPath, []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply")
	if err != nil {
		t.Fatalf("vp apply: %v", err)
	}

	got, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1.3.0\n" {
		t.Errorf("VERSION after apply = %q, want %q", got, "1.3.0\n")
	}

	if _, err := os.Stat(filepath.Join(plansDir, "2026-05-03-bump.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("plan file still exists after apply: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"cli", "1.2.3", "1.3.0", "minor"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q\n---\n%s", want, out)
		}
	}
}
