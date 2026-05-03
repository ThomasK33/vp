package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// goldenDir resolves to <cmd-package>/testdata/output/json regardless of the
// test's current working directory (tests use t.Chdir into temp dirs).
func goldenDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "output", "json")
}

// assertGoldenJSON checks that got matches the bytes in testdata/output/json/<name>.
// When VP_UPDATE_GOLDEN=1 is set in the environment, the golden file is rewritten
// instead. Source-of-truth lives in the pinned files: a downstream consumer can
// inspect or diff them directly.
func assertGoldenJSON(t *testing.T, name string, got []byte) {
	t.Helper()
	dir := goldenDir()
	path := filepath.Join(dir, name)
	if os.Getenv("VP_UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with VP_UPDATE_GOLDEN=1 to create)", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s",
			path, want, got)
	}
}
