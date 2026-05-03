package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const statusTestConfig = `
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
  helm:
    paths: ["chart/**"]
    version: {file: chart/Chart.yaml, format: yaml, path: version}
`

// writeStatusConfig writes a vp.yaml declaring cli, agent, and helm.
func writeStatusConfig(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writePlanFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStatus_NoPlansDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	stdout, _, err := runVP(t, "status")
	if err != nil {
		t.Fatalf("vp status: %v", err)
	}
	if !strings.Contains(stdout.String(), "No pending plans.") {
		t.Errorf("stdout = %q, want to contain %q", stdout.String(), "No pending plans.")
	}
}

func TestStatus_ListsPendingPlansAndSummary(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-fix.yaml",
		"releases:\n  cli: minor\n  agent: none\nmessage: fix\n")
	writePlanFile(t, plansDir, "2026-05-04-bump.yaml",
		"releases:\n  cli: patch\n")

	stdout, _, err := runVP(t, "status")
	if err != nil {
		t.Fatalf("vp status: %v", err)
	}
	got := stdout.String()

	for _, want := range []string{
		"Pending plans:",
		"2026-05-03-fix.yaml",
		"2026-05-04-bump.yaml",
		"agent: none",
		"cli: minor",
		"cli: patch",
		"Resolved bumps:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stdout missing %q\n---\n%s", want, got)
		}
	}

	// In the resolved-bumps summary, cli should collapse minor>patch.
	summary := got[strings.Index(got, "Resolved bumps:"):]
	if !strings.Contains(summary, "cli: minor") {
		t.Errorf("summary missing cli: minor\n---\n%s", summary)
	}
	if !strings.Contains(summary, "agent: none") {
		t.Errorf("summary missing agent: none\n---\n%s", summary)
	}

	// Without --all, helm has no plan and must not appear in the summary.
	if strings.Contains(summary, "helm") {
		t.Errorf("summary unexpectedly contains helm without --all\n---\n%s", summary)
	}

	// Plan listing order is filename-sorted.
	idxFix := strings.Index(got, "2026-05-03-fix.yaml")
	idxBump := strings.Index(got, "2026-05-04-bump.yaml")
	if idxFix == -1 || idxBump == -1 || idxFix > idxBump {
		t.Errorf("plan listing not sorted by filename\n---\n%s", got)
	}
}

func TestStatus_AllFlagShowsEveryConfigComponent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-only-cli.yaml",
		"releases:\n  cli: patch\n")

	stdout, _, err := runVP(t, "status", "--all")
	if err != nil {
		t.Fatalf("vp status --all: %v", err)
	}
	summary := stdout.String()
	summary = summary[strings.Index(summary, "Resolved bumps:"):]

	for _, want := range []string{"cli: patch", "agent: none", "helm: none"} {
		if !strings.Contains(summary, want) {
			t.Errorf("--all summary missing %q\n---\n%s", want, summary)
		}
	}
}

func TestStatus_ErrorsOnUnknownComponent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-typo.yaml",
		"releases:\n  nope: minor\n")

	_, _, err := runVP(t, "status")
	assertUsageError(t, err)
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error = %v, want it to mention the unknown component", err)
	}
	if !strings.Contains(err.Error(), "2026-05-03-typo.yaml") {
		t.Errorf("error = %v, want it to mention the offending plan filename", err)
	}
}

func TestStatus_ErrorsOnMalformedPlan(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bad.yaml",
		"releases:\n  cli: [oops, not a string\n")

	_, _, err := runVP(t, "status")
	assertUsageError(t, err)
}

func TestStatus_ErrorsWithoutConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// no vp.yaml anywhere up the tree

	_, _, err := runVP(t, "status")
	assertUsageError(t, err)
}

const statusTextConfig = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["cli/**"]
    version: {file: VERSION, format: text}
`

func TestStatus_TextComponentShowsCurrentAndNext(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTextConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "status")
	if err != nil {
		t.Fatalf("vp status: %v", err)
	}
	got := stdout.String()
	summary := got[strings.Index(got, "Resolved bumps:"):]
	if !strings.Contains(summary, "1.2.3") || !strings.Contains(summary, "1.3.0") {
		t.Errorf("summary missing current/next versions:\n%s", summary)
	}
	if !strings.Contains(summary, "->") && !strings.Contains(summary, "→") {
		t.Errorf("summary missing arrow between current and next:\n%s", summary)
	}
}

func TestStatus_TextComponentNoneShowsCurrentOnly(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTextConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-noop.yaml", "releases:\n  cli: none\n")

	stdout, _, err := runVP(t, "status")
	if err != nil {
		t.Fatalf("vp status: %v", err)
	}
	summary := stdout.String()[strings.Index(stdout.String(), "Resolved bumps:"):]
	if !strings.Contains(summary, "1.2.3") {
		t.Errorf("summary missing current version: %s", summary)
	}
	if strings.Contains(summary, "->") || strings.Contains(summary, "→") {
		t.Errorf("summary unexpectedly contains arrow for none bump: %s", summary)
	}
}

func TestStatus_TextComponentMissingFileFallsBack(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTextConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	// no VERSION file

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "status")
	if err != nil {
		t.Fatalf("vp status: %v", err)
	}
	summary := stdout.String()[strings.Index(stdout.String(), "Resolved bumps:"):]
	if !strings.Contains(summary, "cli: minor") {
		t.Errorf("summary missing fallback 'cli: minor':\n%s", summary)
	}
}

func TestStatus_NonTextComponentUnchangedFormat(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "status")
	if err != nil {
		t.Fatalf("vp status: %v", err)
	}
	summary := stdout.String()[strings.Index(stdout.String(), "Resolved bumps:"):]
	// cli is json format here; no arrow, no current/next.
	if strings.Contains(summary, "->") || strings.Contains(summary, "→") {
		t.Errorf("summary unexpectedly contains arrow for non-text component:\n%s", summary)
	}
	if !strings.Contains(summary, "cli: minor") {
		t.Errorf("summary missing bare 'cli: minor':\n%s", summary)
	}
}
