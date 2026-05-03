package json_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThomasK33/vp/internal/versionfile/json"
)

func TestRead_Paths(t *testing.T) {
	cases := []struct {
		name string
		body string
		path string
		want string
	}{
		{"top-level", `{"version":"1.2.3"}`, "version", "1.2.3"},
		{"nested", `{"name":"foo","package":{"version":"1.2.3","other":"x"}}`, "package.version", "1.2.3"},
		{"three-level", `{"workspace":{"package":{"version":"0.1.0"}}}`, "workspace.package.version", "0.1.0"},
		{"skips non-matching keys", `{"name":"foo","version":"1.2.3","description":"bar"}`, "version", "1.2.3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := json.Read(path, tc.path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != tc.want {
				t.Errorf("Read = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWrite_Golden(t *testing.T) {
	cases := []struct {
		name    string
		fixture string
		path    string
		oldVer  string
		newVer  string
	}{
		{"toplevel", "toplevel", "version", "1.2.3", "1.2.4"},
		{"nested-package-json", "nested-package-json", "version", "1.2.3", "1.2.4"},
		{"workspace-three-level", "workspace-three-level", "workspace.package.version", "0.1.0", "0.2.0"},
		{"unordered", "unordered", "version", "1.2.3", "1.2.4"},
		{"mixed-indent", "mixed-indent", "package.version", "0.9.0", "0.10.0"},
		{"trailing-whitespace", "trailing-whitespace", "version", "1.0.0", "1.0.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, err := os.ReadFile(filepath.Join("testdata", tc.fixture, "before.json"))
			if err != nil {
				t.Fatal(err)
			}
			wantAfter, err := os.ReadFile(filepath.Join("testdata", tc.fixture, "after.json"))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "f.json")
			if err := os.WriteFile(path, before, 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := json.Read(path, tc.path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != tc.oldVer {
				t.Fatalf("Read = %q, want %q", got, tc.oldVer)
			}
			if err := json.Write(path, tc.path, tc.newVer); err != nil {
				t.Fatalf("Write: %v", err)
			}
			gotAfter, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotAfter, wantAfter) {
				t.Errorf("after Write:\n--- got  ---\n%s\n--- want ---\n%s", gotAfter, wantAfter)
			}
		})
	}
}

func TestReadWrite_RoundTripIdempotent(t *testing.T) {
	body := `{"name":"x","version":"1.2.3","extra":"y"}`
	path := filepath.Join(t.TempDir(), "f.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cur, err := json.Read(path, "version")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if err := json.Write(path, "version", cur); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Errorf("round-trip body = %q, want %q", got, body)
	}
}

func TestWrite_ErrorsLeaveFileUnchanged(t *testing.T) {
	cases := []struct {
		name string
		body string
		path string
	}{
		{"missing key", `{"name":"x"}`, "version"},
		{"non-string leaf", `{"version":123}`, "version"},
		{"non-object midway", `{"version":"1.0.0"}`, "version.sub"},
		{"malformed json", `{"version":`, "version"},
		{"empty path", `{"version":"1.0.0"}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			err := json.Write(path, tc.path, "9.9.9")
			if err == nil {
				t.Fatalf("Write: want error, got nil")
			}
			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(got) != tc.body {
				t.Errorf("file changed after failed Write:\n got = %q\nwant = %q", got, tc.body)
			}
			if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
				t.Errorf("leftover %s.tmp", path)
			}
		})
	}
}

func TestRead_Errors(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		path    string
		wantSub string
	}{
		{"missing key", `{"name":"x"}`, "version", "not found"},
		{"non-string leaf", `{"version":123}`, "version", "is not a string"},
		{"non-object midway", `{"version":"1.0.0"}`, "version.sub", "expected object"},
		{"array midway", `{"version":["1.0.0"]}`, "version", "is not a string"},
		{"malformed json", `{"version":`, "version", "parse"},
		{"empty path", `{"version":"1.0.0"}`, "", "empty path"},
		{"empty segment", `{"a":{"b":"1"}}`, "a..b", "empty segment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.json")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := json.Read(path, tc.path)
			if err == nil {
				t.Fatalf("Read(%q, %q): want error, got nil", tc.body, tc.path)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("Read error = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}
