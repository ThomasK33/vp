package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type applyJSON struct {
	Changes  []applyChangeJSON `json:"changes"`
	Consumed []string          `json:"consumed"`
}

type applyChangeJSON struct {
	Component string `json:"component"`
	Current   string `json:"current"`
	Next      string `json:"next"`
	Bump      string `json:"bump"`
	File      string `json:"file"`
	Tag       string `json:"tag,omitempty"`
}

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

func TestApply_ArchiveModeMovesPlansToArchiveDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := strings.Replace(applyTestConfigText, "consumed: delete", "consumed: archive\n  archive_dir: .version-plans/archive", 1)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	versionPath := filepath.Join(dir, "VERSION")
	if err := os.WriteFile(versionPath, []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	if _, _, err := runVP(t, "apply"); err != nil {
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
		t.Errorf("plan still in plans dir after archive-mode apply: %v", err)
	}
	archivedPath := filepath.Join(plansDir, "archive", "2026-05-03-bump.yaml")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Errorf("plan not found at archive path %s: %v", archivedPath, err)
	}
}

func TestApply_ArchiveModeCreatesMissingArchiveDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// Point archive_dir at a multi-level path that does not exist before apply.
	archiveRel := "history/archive/plans"
	cfg := strings.Replace(applyTestConfigText, "consumed: delete",
		"consumed: archive\n  archive_dir: "+archiveRel, 1)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	if _, err := os.Stat(filepath.Join(dir, archiveRel)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("precondition: archive dir already exists: %v", err)
	}

	if _, _, err := runVP(t, "apply"); err != nil {
		t.Fatalf("vp apply: %v", err)
	}

	archivedPath := filepath.Join(dir, archiveRel, "2026-05-03-bump.yaml")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Errorf("plan not found at archive path %s: %v", archivedPath, err)
	}
}

func TestApply_TagTemplateRenderedInOutput(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := applyTestConfigText + "    tag: \"cli-v{version}\"\n"
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	dryStdout, _, err := runVP(t, "apply", "--dry-run")
	if err != nil {
		t.Fatalf("vp apply --dry-run: %v", err)
	}
	dryOut := dryStdout.String()
	if !strings.Contains(dryOut, "tag") {
		t.Errorf("dry-run header missing tag column:\n%s", dryOut)
	}
	if !strings.Contains(dryOut, "cli-v1.3.0") {
		t.Errorf("dry-run output missing rendered tag cli-v1.3.0:\n%s", dryOut)
	}

	applyStdout, _, err := runVP(t, "apply")
	if err != nil {
		t.Fatalf("vp apply: %v", err)
	}
	if !strings.Contains(applyStdout.String(), "cli-v1.3.0") {
		t.Errorf("apply output missing rendered tag cli-v1.3.0:\n%s", applyStdout.String())
	}
}

func TestApply_DryRunJSONEmitsObjectWithEmptyConsumed(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	planPath := filepath.Join(plansDir, "2026-05-03-bump.yaml")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("vp apply --dry-run --json: %v", err)
	}

	// Dry run must not write or consume.
	got, _ := os.ReadFile(filepath.Join(dir, "VERSION"))
	if string(got) != "1.2.3\n" {
		t.Errorf("VERSION changed during dry-run: %q", got)
	}
	if _, err := os.Stat(planPath); err != nil {
		t.Errorf("plan file consumed during dry-run: %v", err)
	}

	var report applyJSON
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parse json: %v\nstdout: %s", err, stdout.String())
	}
	if len(report.Changes) != 1 {
		t.Fatalf("changes len = %d, want 1", len(report.Changes))
	}
	c := report.Changes[0]
	if c.Component != "cli" || c.Current != "1.2.3" || c.Next != "1.3.0" || c.Bump != "minor" || c.File != "VERSION" {
		t.Errorf("change = %+v", c)
	}
	if c.Tag != "" {
		t.Errorf("tag should be empty when no template configured, got %q", c.Tag)
	}
	if report.Consumed == nil || len(report.Consumed) != 0 {
		t.Errorf("consumed = %v, want empty non-nil array", report.Consumed)
	}
	if !strings.Contains(stdout.String(), `"consumed": []`) {
		t.Errorf("expected empty consumed array in raw JSON, got:\n%s", stdout.String())
	}
}

func TestApply_LiveJSONListsConsumedAndSuppressesTrailer(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply", "--json")
	if err != nil {
		t.Fatalf("vp apply --json: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "VERSION"))
	if string(got) != "1.3.0\n" {
		t.Errorf("VERSION after apply = %q, want 1.3.0\\n", got)
	}

	var report applyJSON
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parse json: %v\nstdout: %s", err, stdout.String())
	}
	if !slices.Equal(report.Consumed, []string{"2026-05-03-bump.yaml"}) {
		t.Errorf("consumed = %v, want [2026-05-03-bump.yaml]", report.Consumed)
	}
	if len(report.Changes) != 1 || report.Changes[0].Component != "cli" {
		t.Errorf("changes = %+v", report.Changes)
	}

	// The text trailer "Wrote N file(s); consumed M plan(s)." must not leak into JSON stdout.
	if strings.Contains(stdout.String(), "Wrote ") {
		t.Errorf("stdout contains text trailer alongside JSON:\n%s", stdout.String())
	}
}

func TestApply_JSONEmptyChangesAndConsumedWhenNothingPending(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runVP(t, "apply", "--json")
	if err != nil {
		t.Fatalf("vp apply --json: %v", err)
	}
	if !strings.Contains(stdout.String(), `"changes": []`) {
		t.Errorf("expected empty changes array:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"consumed": []`) {
		t.Errorf("expected empty consumed array:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "no version changes") {
		t.Errorf("text-mode message leaked into JSON:\n%s", stdout.String())
	}
}

func TestApply_LiveJSONIncludesTagWhenConfigured(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := applyTestConfigText + "    tag: \"cli-v{version}\"\n"
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("vp apply --dry-run --json: %v", err)
	}
	var report applyJSON
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if report.Changes[0].Tag != "cli-v1.3.0" {
		t.Errorf("tag = %q, want cli-v1.3.0", report.Changes[0].Tag)
	}
}

func TestApply_ArchiveModeJSONStillReportsConsumed(t *testing.T) {
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

	stdout, _, err := runVP(t, "apply", "--json")
	if err != nil {
		t.Fatalf("vp apply --json: %v", err)
	}
	var report applyJSON
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if !slices.Equal(report.Consumed, []string{"2026-05-03-bump.yaml"}) {
		t.Errorf("consumed = %v, want [2026-05-03-bump.yaml] (basename, not archive path)", report.Consumed)
	}
}

func TestApply_JSONUnknownComponentLeavesStdoutEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-typo.yaml", "releases:\n  nope: minor\n")

	stdout, _, err := runVP(t, "apply", "--json")
	assertUsageError(t, err)
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty when apply errors before printing JSON, got:\n%s", stdout.String())
	}
}

func TestRenderTag(t *testing.T) {
	tests := []struct {
		name     string
		template string
		version  string
		want     string
	}{
		{"empty template returns empty", "", "1.2.3", ""},
		{"version only", "{version}", "1.2.3", "1.2.3"},
		{"prefix and version", "cli-v{version}", "1.2.3", "cli-v1.2.3"},
		{"version repeated", "{version}+{version}", "0.1.0", "0.1.0+0.1.0"},
		{"unknown placeholder left literal", "cli-{component}-v{version}", "2.0.0", "cli-{component}-v2.0.0"},
		{"no placeholder", "static-tag", "9.9.9", "static-tag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTag(tt.template, tt.version)
			if got != tt.want {
				t.Errorf("renderTag(%q, %q) = %q, want %q", tt.template, tt.version, got, tt.want)
			}
		})
	}
}

func TestApply_TagPreservesNonVersionPlaceholders(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	cfg := applyTestConfigText + "    tag: \"cli-{component}-v{version}\"\n"
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply", "--dry-run")
	if err != nil {
		t.Fatalf("vp apply --dry-run: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "cli-{component}-v1.3.0") {
		t.Errorf("rendered tag should leave {component} literal, got:\n%s", out)
	}
}

func TestApply_NoTagColumnWhenNoComponentHasTag(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply", "--dry-run")
	if err != nil {
		t.Fatalf("vp apply --dry-run: %v", err)
	}
	header, _, _ := strings.Cut(stdout.String(), "\n")
	if strings.Contains(header, "tag") {
		t.Errorf("dry-run header includes tag column when no component has tag set:\n%s", stdout.String())
	}
}

func TestApply_DeleteModeStillDeletesPlans(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	planPath := filepath.Join(plansDir, "2026-05-03-bump.yaml")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	if _, _, err := runVP(t, "apply"); err != nil {
		t.Fatalf("vp apply: %v", err)
	}

	if _, err := os.Stat(planPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("plan should have been deleted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(plansDir, "archive")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("delete mode unexpectedly created archive directory: %v", err)
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
