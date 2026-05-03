package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ThomasK33/vp/internal/config"
)

func TestFindUpwards_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := config.FindUpwards(dir)
	if !errors.Is(err, config.ErrNotFound) {
		t.Fatalf("FindUpwards(%q) error = %v, want %v", dir, err, config.ErrNotFound)
	}
}

func TestFindUpwards_InAncestor(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "vp.yaml")
	if err := os.WriteFile(want, []byte("plans: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := config.FindUpwards(deep)
	if err != nil {
		t.Fatalf("FindUpwards: %v", err)
	}
	if got != want {
		t.Fatalf("FindUpwards = %q, want %q", got, want)
	}
}

func TestFindUpwards_InStartDir(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join(dir, "vp.yaml")
	if err := os.WriteFile(want, []byte("plans: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := config.FindUpwards(dir)
	if err != nil {
		t.Fatalf("FindUpwards: %v", err)
	}
	if got != want {
		t.Fatalf("FindUpwards = %q, want %q", got, want)
	}
}
