package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const checkTestConfig = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["cli/**"]
    version: {file: VERSION, format: text}
  agent:
    paths: ["agent/**"]
    version: {file: AGENT_VERSION, format: text}
`

type checkJSON struct {
	Affected []string `json:"affected"`
	Planned  []string `json:"planned"`
	Missing  []string `json:"missing"`
}

// gitInRepo runs git args in dir, failing the test on non-zero exit.
func gitInRepo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// initCheckRepo creates a fresh git repo at dir with the canonical vp.yaml,
// commits an initial state, then returns. Subsequent commits are the caller's
// responsibility — that's what the diff under check will see.
func initCheckRepo(t *testing.T, dir string) {
	t.Helper()
	gitInRepo(t, dir, "init", "-q", "-b", "main")
	gitInRepo(t, dir, "config", "user.email", "vp-test@example.com")
	gitInRepo(t, dir, "config", "user.name", "vp test")
	gitInRepo(t, dir, "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(checkTestConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENT_VERSION"), []byte("0.1.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cli", "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent", "agent.go"), []byte("package agent\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "base")
}

func TestCheck_AllAffectedCovered(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	// Modify cli; add a plan covering cli; commit both.
	if err := os.WriteFile(filepath.Join(dir, "cli", "main.go"), []byte("package main\n// changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-cli.yaml", "releases:\n  cli: minor\n")
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "feat: cli change with plan")

	stdout, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD")
	if err != nil {
		t.Fatalf("vp check: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "All affected components are covered.") {
		t.Errorf("stdout missing success line:\n%s", got)
	}
	if !strings.Contains(got, "Affected components: cli") {
		t.Errorf("stdout missing affected-cli line:\n%s", got)
	}
}

func TestCheck_MissingCoverageExits1(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	// Modify agent without a covering plan.
	if err := os.WriteFile(filepath.Join(dir, "agent", "agent.go"), []byte("package agent\n// new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "feat: agent change")

	stdout, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD")
	if err == nil {
		t.Fatal("vp check: want error, got nil")
	}
	coded, ok := errors.AsType[*exitCodeError](err)
	if !ok || coded.code != exitCheckError {
		t.Fatalf("vp check error = %v (want code %d)", err, exitCheckError)
	}
	got := stdout.String()
	if !strings.Contains(got, "Missing coverage:") || !strings.Contains(got, "agent") {
		t.Errorf("stdout missing 'Missing coverage: ... agent':\n%s", got)
	}
}

func TestCheck_PlansDirChurnIgnored(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	// Only change is a new plan file under .version-plans/.
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-noop.yaml", "releases:\n  cli: none\n")
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "chore: add plan")

	stdout, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD")
	if err != nil {
		t.Fatalf("vp check: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Affected components: (none)") {
		t.Errorf("stdout should report no affected components when only plans churned:\n%s", got)
	}
}

func TestCheck_VpYamlChangeIgnored(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	// Touch vp.yaml; nothing else.
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(checkTestConfig+"\n# touch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "chore: touch config")

	stdout, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD")
	if err != nil {
		t.Fatalf("vp check: %v", err)
	}
	if !strings.Contains(stdout.String(), "Affected components: (none)") {
		t.Errorf("vp.yaml change should yield no affected components:\n%s", stdout.String())
	}
}

// Stack coverage (ADR-0001): a plan committed in an earlier diff still
// satisfies the check for a later code-only diff.
func TestCheck_StackCoveragePlanFromEarlierDiff(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	// Commit 1 (HEAD~1 from base view): add a plan covering cli.
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-02-early-cli.yaml", "releases:\n  cli: minor\n")
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "chore: pre-stage cli plan")

	// Commit 2 (HEAD): code-only change to cli. The covering plan is NOT in this diff.
	if err := os.WriteFile(filepath.Join(dir, "cli", "main.go"), []byte("package main\n// later\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "feat: cli code change")

	stdout, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD")
	if err != nil {
		t.Fatalf("vp check (stack-coverage): %v", err)
	}
	if !strings.Contains(stdout.String(), "All affected components are covered.") {
		t.Errorf("stack-coverage diff should pass:\n%s", stdout.String())
	}
}

func TestCheck_MissingFlagsIsUsageError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	_, _, err := runVP(t, "check")
	assertUsageError(t, err)
}

func TestCheck_MissingConfigIsUsageError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// no vp.yaml anywhere up the tree

	_, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD")
	assertUsageError(t, err)
}

func TestCheck_JSONCoveredEmitsArrays(t *testing.T) {
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

	var got checkJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v\nstdout: %s", err, stdout.String())
	}
	if !slices.Equal(got.Affected, []string{"cli"}) {
		t.Errorf("affected = %v, want [cli]", got.Affected)
	}
	if !slices.Equal(got.Planned, []string{"cli"}) {
		t.Errorf("planned = %v, want [cli]", got.Planned)
	}
	if got.Missing == nil || len(got.Missing) != 0 {
		t.Errorf("missing = %v, want empty non-nil array", got.Missing)
	}
	if !strings.Contains(stdout.String(), `"missing": []`) {
		t.Errorf("expected empty missing to render as `[]` (not null), got:\n%s", stdout.String())
	}
}

func TestCheck_JSONMissingCoverageExits1WithStdoutPayload(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	if err := os.WriteFile(filepath.Join(dir, "agent", "agent.go"), []byte("package agent\n// new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "feat: agent change")

	stdout, stderr, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD", "--json")
	if err == nil {
		t.Fatal("vp check --json: want check error, got nil")
	}
	coded, ok := errors.AsType[*exitCodeError](err)
	if !ok || coded.code != exitCheckError {
		t.Fatalf("vp check --json error = %v (want code %d)", err, exitCheckError)
	}

	var got checkJSON
	if uerr := json.Unmarshal(stdout.Bytes(), &got); uerr != nil {
		t.Fatalf("parse json: %v\nstdout: %s", uerr, stdout.String())
	}
	if !slices.Equal(got.Missing, []string{"agent"}) {
		t.Errorf("missing = %v, want [agent]", got.Missing)
	}
	if !slices.Equal(got.Affected, []string{"agent"}) {
		t.Errorf("affected = %v, want [agent]", got.Affected)
	}
	if got.Planned == nil || len(got.Planned) != 0 {
		t.Errorf("planned = %v, want empty non-nil array", got.Planned)
	}
	// Stderr-only contract: error text on stderr; stdout is pure JSON.
	if strings.Contains(stdout.String(), "missing plan coverage") {
		t.Errorf("stdout leaked the error message:\n%s", stdout.String())
	}
	if stderr.Len() != 0 && !strings.Contains(stderr.String(), "missing plan coverage") {
		t.Errorf("stderr should mention missing coverage, got:\n%s", stderr.String())
	}
}

func TestCheck_JSONEmptyEmitsAllArrays(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	// Touch only vp.yaml — yields zero affected components and no plans.
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(checkTestConfig+"\n# touch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitInRepo(t, dir, "add", ".")
	gitInRepo(t, dir, "commit", "-q", "-m", "chore: touch config")

	stdout, _, err := runVP(t, "check", "--base", "HEAD~1", "--head", "HEAD", "--json")
	if err != nil {
		t.Fatalf("vp check --json: %v", err)
	}
	// All three arrays must be present and empty (not null).
	for _, key := range []string{`"affected": []`, `"planned": []`, `"missing": []`} {
		if !strings.Contains(stdout.String(), key) {
			t.Errorf("stdout missing %q\n%s", key, stdout.String())
		}
	}
}

func TestCheck_BogusRefIsRuntimeError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	initCheckRepo(t, dir)

	_, _, err := runVP(t, "check", "--base", "no-such-ref", "--head", "HEAD")
	if err == nil {
		t.Fatal("vp check: want error, got nil")
	}
	if _, ok := errors.AsType[*exitCodeError](err); ok {
		t.Fatalf("vp check error wrapped as exit-coded; want bare runtime error: %v", err)
	}
}
