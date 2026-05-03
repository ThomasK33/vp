package yaml_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ThomasK33/vp/internal/versionfile/yaml"
)

func TestRead_Paths(t *testing.T) {
	cases := []struct {
		name string
		body string
		path string
		want string
	}{
		{"top-level", "version: 1.2.3\n", "version", "1.2.3"},
		{"nested", "package:\n  version: 1.2.3\n  other: x\n", "package.version", "1.2.3"},
		{"three-level", "workspace:\n  package:\n    version: 0.1.0\n", "workspace.package.version", "0.1.0"},
		{"skips non-matching keys", "name: foo\nversion: 1.2.3\ndescription: bar\n", "version", "1.2.3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Read(path, tc.path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != tc.want {
				t.Errorf("Read = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWrite_StylePreservation(t *testing.T) {
	cases := []struct {
		name   string
		before string
		path   string
		old    string
		new    string
		after  string
	}{
		{"plain", "version: 1.2.3\n", "version", "1.2.3", "1.2.4", "version: 1.2.4\n"},
		{"double-quoted", "version: \"1.2.3\"\n", "version", "1.2.3", "1.2.4", "version: \"1.2.4\"\n"},
		{"single-quoted", "version: '1.2.3'\n", "version", "1.2.3", "1.2.4", "version: '1.2.4'\n"},
		{"plain with inline comment", "version: 1.2.3 # pinned\n", "version", "1.2.3", "1.2.4", "version: 1.2.4 # pinned\n"},
		{"plain with trailing spaces", "version: 1.2.3   \n", "version", "1.2.3", "1.2.4", "version: 1.2.4   \n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.yaml")
			if err := os.WriteFile(path, []byte(tc.before), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Read(path, tc.path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != tc.old {
				t.Fatalf("Read = %q, want %q", got, tc.old)
			}
			if err := yaml.Write(path, tc.path, tc.new); err != nil {
				t.Fatalf("Write: %v", err)
			}
			gotAfter, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(gotAfter, []byte(tc.after)) {
				t.Errorf("after Write:\n got = %q\nwant = %q", gotAfter, tc.after)
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
		{"chart-toplevel", "chart-toplevel", "version", "1.2.3", "1.2.4"},
		{"nested", "nested", "package.version", "0.1.0", "0.2.0"},
		{"comments", "comments", "version", "1.2.3", "1.2.4"},
		{"anchors-elsewhere", "anchors-elsewhere", "version", "1.2.3", "1.2.4"},
		{"literal-elsewhere", "literal-elsewhere", "version", "1.2.3", "1.2.4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before, err := os.ReadFile(filepath.Join("testdata", tc.fixture, "before.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			wantAfter, err := os.ReadFile(filepath.Join("testdata", tc.fixture, "after.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "f.yaml")
			if err := os.WriteFile(path, before, 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := yaml.Read(path, tc.path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != tc.oldVer {
				t.Fatalf("Read = %q, want %q", got, tc.oldVer)
			}
			if err := yaml.Write(path, tc.path, tc.newVer); err != nil {
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
	cases := []struct {
		name string
		body string
	}{
		{"plain", "name: foo\nversion: 1.2.3\nextra: y\n"},
		{"double-quoted", "name: foo\nversion: \"1.2.3\"\nextra: y\n"},
		{"single-quoted", "name: foo\nversion: '1.2.3'\nextra: y\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			cur, err := yaml.Read(path, "version")
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if err := yaml.Write(path, "version", cur); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.body {
				t.Errorf("round-trip body = %q, want %q", got, tc.body)
			}
		})
	}
}

func TestWrite_ErrorsLeaveFileUnchanged(t *testing.T) {
	cases := []struct {
		name string
		body string
		path string
	}{
		{"missing key", "name: x\n", "version"},
		{"non-string leaf", "version: 123\n", "version"},
		{"non-mapping midway", "version: 1.0.0\n", "version.sub"},
		{"malformed yaml", "version: : :\n", "version"},
		{"empty path", "version: 1.0.0\n", ""},
		{"literal block leaf", "version: |\n  1.2.3\n", "version"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			err := yaml.Write(path, tc.path, "9.9.9")
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
		{"missing key", "name: foo\n", "version", "not found"},
		{"non-string int leaf", "version: 123\n", "version", "is not a string"},
		{"non-string float leaf", "version: 1.0\n", "version", "is not a string"},
		{"non-string bool leaf", "version: true\n", "version", "is not a string"},
		{"non-string null leaf", "version: ~\n", "version", "is not a string"},
		{"non-mapping midway", "version: 1.0.0\n", "version.sub", "expected mapping"},
		{"sequence leaf", "version:\n  - 1.0.0\n", "version", "is not a string"},
		{"mapping leaf", "version:\n  inner: 1.0.0\n", "version", "is not a string"},
		{"literal block leaf", "version: |\n  1.2.3\n", "version", "plain or quoted"},
		{"folded block leaf", "version: >\n  1.2.3\n", "version", "plain or quoted"},
		{"malformed yaml", "version: : :\n", "version", "parse"},
		{"empty path", "version: 1.0.0\n", "", "empty path"},
		{"empty segment", "a:\n  b: 1\n", "a..b", "empty segment"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "f.yaml")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := yaml.Read(path, tc.path)
			if err == nil {
				t.Fatalf("Read(%q, %q): want error, got nil", tc.body, tc.path)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("Read error = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}
