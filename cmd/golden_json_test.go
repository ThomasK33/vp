package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// These tests pin the exact byte output of `--json` for stable CI consumers.
// Update with VP_UPDATE_GOLDEN=1 go test ./cmd/...

func TestGolden_StatusEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}
	assertGoldenJSON(t, "status-empty.json", stdout.Bytes())
}

func TestGolden_StatusPending(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTextConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-fix.yaml",
		"releases:\n  cli: minor\nmessage: fix the thing\n")

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}
	assertGoldenJSON(t, "status-pending.json", stdout.Bytes())
}

func TestGolden_ApplyDryRun(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeApplyTextConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "apply", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("vp apply --dry-run --json: %v", err)
	}
	assertGoldenJSON(t, "apply-dryrun.json", stdout.Bytes())
}

func TestGolden_ApplyLive(t *testing.T) {
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
	assertGoldenJSON(t, "apply-live.json", stdout.Bytes())
}

func TestGolden_ApplyMultiFormat(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeMultiFormatFixture(t, dir)
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml",
		"releases:\n  txt: minor\n  js: patch\n  yml: major\n  tml: minor\n")

	stdout, _, err := runVP(t, "apply", "--json")
	if err != nil {
		t.Fatalf("vp apply --json: %v", err)
	}
	assertGoldenJSON(t, "apply-multiformat.json", stdout.Bytes())
}

func TestGolden_ApplyEmpty(t *testing.T) {
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
	assertGoldenJSON(t, "apply-empty.json", stdout.Bytes())
}

func TestGolden_CheckCovered(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "cli", "main.go"), []byte("package main\n// changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-cli.yaml", "releases:\n  cli: minor\n")
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "feat: cli with plan")

	stdout, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD", "--json")
	if err != nil {
		t.Fatalf("vp check --json: %v", err)
	}
	assertGoldenJSON(t, "check-covered.json", stdout.Bytes())
}

func TestGolden_CheckMissing(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "agent", "agent.go"), []byte("package agent\n// new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "feat: agent change")

	stdout, _, _ := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD", "--json")
	// Missing-coverage exits non-zero, but stdout still carries the JSON payload.
	assertGoldenJSON(t, "check-missing.json", stdout.Bytes())
}
