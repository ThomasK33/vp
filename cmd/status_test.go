package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type statusJSON struct {
	Pending  []pendingJSON  `json:"pending"`
	Resolved []resolvedJSON `json:"resolved"`
}

type pendingJSON struct {
	File     string            `json:"file"`
	Releases map[string]string `json:"releases"`
	Message  string            `json:"message,omitempty"`
}

type resolvedJSON struct {
	Component string `json:"component"`
	Bump      string `json:"bump"`
	Current   string `json:"current,omitempty"`
	Next      string `json:"next,omitempty"`
}

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

func TestStatus_JSONEmptyEmitsBothArrays(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}

	var got statusJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v\nstdout: %s", err, stdout.String())
	}
	if got.Pending == nil || len(got.Pending) != 0 {
		t.Errorf("pending = %v, want empty non-nil array", got.Pending)
	}
	if got.Resolved == nil || len(got.Resolved) != 0 {
		t.Errorf("resolved = %v, want empty non-nil array", got.Resolved)
	}
	if !strings.Contains(stdout.String(), `"pending": []`) || !strings.Contains(stdout.String(), `"resolved": []`) {
		t.Errorf("expected empty arrays in raw JSON, got:\n%s", stdout.String())
	}
}

func TestStatus_JSONIncludesPendingAndResolvedSorted(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-fix.yaml",
		"releases:\n  cli: minor\n  agent: none\nmessage: fix\n")
	writePlanFile(t, plansDir, "2026-05-04-bump.yaml",
		"releases:\n  cli: patch\n")

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}
	var got statusJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v\nstdout: %s", err, stdout.String())
	}

	if len(got.Pending) != 2 {
		t.Fatalf("pending len = %d, want 2\n%s", len(got.Pending), stdout.String())
	}
	if got.Pending[0].File != "2026-05-03-fix.yaml" || got.Pending[1].File != "2026-05-04-bump.yaml" {
		t.Errorf("pending order wrong: %v", []string{got.Pending[0].File, got.Pending[1].File})
	}
	if got.Pending[0].Message != "fix" {
		t.Errorf("first pending message = %q, want %q", got.Pending[0].Message, "fix")
	}
	if got.Pending[1].Message != "" {
		t.Errorf("second pending message = %q, want empty", got.Pending[1].Message)
	}
	if got.Pending[0].Releases["cli"] != "minor" || got.Pending[0].Releases["agent"] != "none" {
		t.Errorf("first pending releases = %v", got.Pending[0].Releases)
	}

	// Resolved is component-name sorted; cli collapses minor>patch; helm omitted (no plan, no --all).
	wantNames := []string{"agent", "cli"}
	gotNames := []string{}
	for _, r := range got.Resolved {
		gotNames = append(gotNames, r.Component)
	}
	if !slices.Equal(gotNames, wantNames) {
		t.Errorf("resolved components = %v, want %v", gotNames, wantNames)
	}
	for _, r := range got.Resolved {
		if r.Component == "cli" && r.Bump != "minor" {
			t.Errorf("cli bump = %q, want minor", r.Bump)
		}
		if r.Component == "agent" && r.Bump != "none" {
			t.Errorf("agent bump = %q, want none", r.Bump)
		}
	}

	// Verify omitempty: the second pending entry must have no "message" key.
	idx := strings.Index(stdout.String(), "2026-05-04-bump.yaml")
	if idx < 0 {
		t.Fatalf("second pending entry not found in JSON")
	}
	if strings.Contains(stdout.String()[idx:], `"message"`) {
		t.Errorf("second pending should omit message key:\n%s", stdout.String()[idx:])
	}
}

func TestStatus_JSONTextFormatHasCurrentAndNext(t *testing.T) {
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

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}
	var got statusJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	if len(got.Resolved) != 1 {
		t.Fatalf("resolved len = %d, want 1", len(got.Resolved))
	}
	r := got.Resolved[0]
	if r.Current != "1.2.3" || r.Next != "1.3.0" {
		t.Errorf("current/next = %q/%q, want 1.2.3/1.3.0", r.Current, r.Next)
	}
}

func TestStatus_JSONNoneOmitsNext(t *testing.T) {
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

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}
	var got statusJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	r := got.Resolved[0]
	if r.Current != "1.2.3" {
		t.Errorf("current = %q, want 1.2.3", r.Current)
	}
	if r.Next != "" {
		t.Errorf("next should be empty for bump=none, got %q", r.Next)
	}
	// raw JSON: ensure "next" key does not appear for the cli entry.
	if strings.Contains(stdout.String(), `"next"`) {
		t.Errorf("raw JSON should omit next when bump=none:\n%s", stdout.String())
	}
}

func TestStatus_JSONReadFailureOmitsCurrentAndNext(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(statusTextConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	// no VERSION file — text reader will fail
	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml", "releases:\n  cli: minor\n")

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}
	var got statusJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	r := got.Resolved[0]
	if r.Current != "" || r.Next != "" {
		t.Errorf("expected current/next empty on read failure, got %q/%q", r.Current, r.Next)
	}
	if r.Bump != "minor" {
		t.Errorf("bump = %q, want minor (bump must still appear even when version unreadable)", r.Bump)
	}
}

func TestStatus_JSONReadsAllSupportedFormats(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	const cfg = `
plans:
  dir: .version-plans
  consumed: delete
components:
  cli:
    paths: ["cli/**"]
    version: {file: cli/package.json, format: json, path: version}
  agent:
    paths: ["agent/**"]
    version: {file: agent/Cargo.toml, format: toml, path: package.version}
  helm:
    paths: ["chart/**"]
    version: {file: chart/Chart.yaml, format: yaml, path: version}
`
	if err := os.WriteFile(filepath.Join(dir, "vp.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(dir, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cli", "package.json"),
		[]byte(`{"version":"1.2.3"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agent", "Cargo.toml"),
		[]byte("[package]\nversion = \"0.4.1\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "chart"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chart", "Chart.yaml"),
		[]byte("name: thing\nversion: 2.5.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-bump.yaml",
		"releases:\n  cli: minor\n  agent: patch\n  helm: major\n")

	stdout, _, err := runVP(t, "status", "--json")
	if err != nil {
		t.Fatalf("vp status --json: %v", err)
	}
	var got statusJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v\n%s", err, stdout.String())
	}

	want := map[string][2]string{
		"agent": {"0.4.1", "0.4.2"},
		"cli":   {"1.2.3", "1.3.0"},
		"helm":  {"2.5.0", "3.0.0"},
	}
	for _, r := range got.Resolved {
		w, ok := want[r.Component]
		if !ok {
			continue
		}
		if r.Current != w[0] || r.Next != w[1] {
			t.Errorf("%s current/next = %q/%q, want %q/%q",
				r.Component, r.Current, r.Next, w[0], w[1])
		}
	}
}

func TestStatus_JSONWithAllShowsEveryComponent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeStatusConfig(t, dir)

	plansDir := filepath.Join(dir, ".version-plans")
	writePlanFile(t, plansDir, "2026-05-03-only-cli.yaml", "releases:\n  cli: patch\n")

	stdout, _, err := runVP(t, "status", "--json", "--all")
	if err != nil {
		t.Fatalf("vp status --json --all: %v", err)
	}
	var got statusJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("parse json: %v", err)
	}
	bumps := map[string]string{}
	for _, r := range got.Resolved {
		bumps[r.Component] = r.Bump
	}
	for _, want := range []string{"cli", "agent", "helm"} {
		if _, ok := bumps[want]; !ok {
			t.Errorf("--all resolved missing %q\n%s", want, stdout.String())
		}
	}
	if bumps["cli"] != "patch" {
		t.Errorf("cli bump = %q, want patch", bumps["cli"])
	}
	if bumps["agent"] != "none" || bumps["helm"] != "none" {
		t.Errorf("agent/helm bump = %q/%q, want none/none", bumps["agent"], bumps["helm"])
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
