package plan_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThomasK33/vp/internal/plan"
)

func TestLoadDir_MissingDirReturnsEmpty(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	got, err := plan.LoadDir(missing)
	if err != nil {
		t.Fatalf("LoadDir(missing): unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("LoadDir(missing) = %v, want empty", got)
	}
}

func TestLoadDir_EmptyDirReturnsEmpty(t *testing.T) {
	got, err := plan.LoadDir(t.TempDir())
	if err != nil {
		t.Fatalf("LoadDir(empty): unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("LoadDir(empty) = %v, want empty", got)
	}
}

func TestLoadDir_LoadsValidPlansSorted(t *testing.T) {
	got, err := plan.LoadDir("testdata/plans/valid")
	if err != nil {
		t.Fatalf("LoadDir(valid): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("LoadDir(valid) returned %d plans, want 2", len(got))
	}

	if base := filepath.Base(got[0].Path); base != "2026-05-03-one.yaml" {
		t.Errorf("got[0].Path basename = %q, want 2026-05-03-one.yaml", base)
	}
	if base := filepath.Base(got[1].Path); base != "2026-05-04-two.yaml" {
		t.Errorf("got[1].Path basename = %q, want 2026-05-04-two.yaml", base)
	}

	if got[0].Plan.Releases["cli"] != "minor" {
		t.Errorf("got[0].Plan.Releases[cli] = %q, want minor", got[0].Plan.Releases["cli"])
	}
	if got[0].Plan.Message != "Fix reconnect" {
		t.Errorf("got[0].Plan.Message = %q, want Fix reconnect", got[0].Plan.Message)
	}
	if got[1].Plan.Releases["agent"] != "patch" || got[1].Plan.Releases["helm"] != "major" {
		t.Errorf("got[1].Plan.Releases = %v, want {agent: patch, helm: major}", got[1].Plan.Releases)
	}
}

func TestLoadDir_IgnoresNonYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# notes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	planYAML := "releases:\n  cli: minor\n"
	if err := os.WriteFile(filepath.Join(dir, "2026-05-03-only.yaml"), []byte(planYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := plan.LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("LoadDir returned %d entries, want 1 (yaml only)", len(got))
	}
	if base := filepath.Base(got[0].Path); base != "2026-05-03-only.yaml" {
		t.Errorf("base = %q, want 2026-05-03-only.yaml", base)
	}
}

func TestLoadDir_ErrorsOnMalformedYAML(t *testing.T) {
	_, err := plan.LoadDir("testdata/plans/malformed")
	if err == nil {
		t.Fatal("LoadDir(malformed): want error, got nil")
	}
	if !strings.Contains(err.Error(), "2026-05-03-bad.yaml") {
		t.Errorf("error = %v, want it to mention the offending filename", err)
	}
}

func TestLoadDir_ErrorsOnInvalidBump(t *testing.T) {
	_, err := plan.LoadDir("testdata/plans/invalid-bump")
	if err == nil {
		t.Fatal("LoadDir(invalid-bump): want error, got nil")
	}
	if !strings.Contains(err.Error(), "2026-05-03-huge.yaml") {
		t.Errorf("error = %v, want it to mention the offending filename", err)
	}
}
