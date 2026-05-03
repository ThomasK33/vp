package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ThomasK33/vp/internal/config"
)

// runVP executes the root command with the given args inside the current
// working directory, returning stdout/stderr buffers and the error returned
// by Cobra. It is hermetic: it snapshots and restores rootCmd's args and
// writers around the call.
func runVP(t *testing.T, args ...string) (*bytes.Buffer, *bytes.Buffer, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer

	prevArgs := os.Args
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	t.Cleanup(func() {
		os.Args = prevArgs
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		resetAllFlags(rootCmd)
	})

	err := rootCmd.Execute()
	return &stdout, &stderr, err
}

// resetAllFlags walks the command tree and restores every flag to its declared
// default. Cobra/pflag retain parsed flag values across Execute() calls on the
// same command object, which leaks state between tests.
func resetAllFlags(c *cobra.Command) {
	c.Flags().VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
	for _, sub := range c.Commands() {
		resetAllFlags(sub)
	}
}

func TestInit_RefusesWhenExistsInCWD(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	target := filepath.Join(dir, "vp.yaml")
	original := []byte("# pre-existing\n")
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := runVP(t, "init")
	if err == nil {
		t.Fatalf("vp init: want error, got nil")
	}
	if coded, ok := errors.AsType[*exitCodeError](err); !ok || coded.code != exitUsageError {
		t.Fatalf("vp init error = %v (code want %d)", err, exitUsageError)
	}

	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("reading vp.yaml: %v", readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("vp.yaml was modified despite refusal:\nwant %q\ngot  %q", original, got)
	}
}

func TestInit_RefusesWhenExistsInAncestor(t *testing.T) {
	root := t.TempDir()
	ancestorVP := filepath.Join(root, "vp.yaml")
	if err := os.WriteFile(ancestorVP, []byte("# parent\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(child)

	_, _, err := runVP(t, "init")
	if err == nil {
		t.Fatalf("vp init: want error, got nil")
	}
	if coded, ok := errors.AsType[*exitCodeError](err); !ok || coded.code != exitUsageError {
		t.Fatalf("vp init error = %v (code want %d)", err, exitUsageError)
	}
	if _, statErr := os.Stat(filepath.Join(child, "vp.yaml")); !os.IsNotExist(statErr) {
		t.Fatalf("vp.yaml was created in child despite ancestor conflict")
	}
}

func TestInit_StarterIsValidConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if _, _, err := runVP(t, "init"); err != nil {
		t.Fatalf("vp init: %v", err)
	}

	if _, err := config.Load(dir); err != nil {
		t.Fatalf("starter vp.yaml does not load cleanly: %v", err)
	}
}

func TestInit_FirstRunWritesStarter(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if _, _, err := runVP(t, "init"); err != nil {
		t.Fatalf("vp init: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "vp.yaml"))
	if err != nil {
		t.Fatalf("reading written vp.yaml: %v", err)
	}
	if !bytes.Equal(got, config.Starter()) {
		t.Fatalf("vp.yaml content does not match embedded starter")
	}
}

func TestInit_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	target := filepath.Join(dir, "vp.yaml")
	if err := os.WriteFile(target, []byte("# stale\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := runVP(t, "init", "--force"); err != nil {
		t.Fatalf("vp init --force: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading vp.yaml: %v", err)
	}
	if !bytes.Equal(got, config.Starter()) {
		t.Fatalf("vp.yaml not overwritten with starter")
	}
}
