package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/ThomasK33/vp/internal/git"
)

// initRepo runs the minimum git invocations needed to produce a working tree
// with a configured identity at dir. Mise provides git on CI.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "vp-test@example.com"},
		{"config", "user.name", "vp test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func gitDo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestChangedFiles_TwoCommits(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)

	writeFile(t, dir, "cli/main.go", "package main\n")
	writeFile(t, dir, "README.md", "old\n")
	gitDo(t, dir, "add", ".")
	gitDo(t, dir, "commit", "-q", "-m", "base")

	writeFile(t, dir, "cli/main.go", "package main\n// changed\n")
	writeFile(t, dir, "agent/agent.go", "package agent\n")
	gitDo(t, dir, "add", ".")
	gitDo(t, dir, "commit", "-q", "-m", "head")

	got, err := git.ChangedFiles(dir, "HEAD~1", "HEAD")
	if err != nil {
		t.Fatalf("ChangedFiles: %v", err)
	}
	sort.Strings(got)
	want := []string{"agent/agent.go", "cli/main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ChangedFiles = %v, want %v", got, want)
	}
}

func TestChangedFiles_BogusRefReturnsError(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	writeFile(t, dir, "a.txt", "1\n")
	gitDo(t, dir, "add", ".")
	gitDo(t, dir, "commit", "-q", "-m", "init")

	_, err := git.ChangedFiles(dir, "no-such-ref", "HEAD")
	if err == nil {
		t.Fatal("ChangedFiles: want error for bogus ref, got nil")
	}
}
