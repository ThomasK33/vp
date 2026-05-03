package text_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ThomasK33/vp/internal/versionfile/text"
)

func TestRead(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"trailing newline stripped", "1.2.3\n", "1.2.3"},
		{"no trailing newline", "1.2.3", "1.2.3"},
		{"surrounding whitespace stripped", "  1.2.3  \n", "1.2.3"},
		{"prerelease preserved", "v1.2.3-rc.1\n", "v1.2.3-rc.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "VERSION")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := text.Read(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if got != tc.want {
				t.Errorf("Read(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestRead_RejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := text.Read(path); err == nil {
		t.Fatal("Read(empty): want error, got nil")
	}
}

func TestRead_RejectsMissingFile(t *testing.T) {
	if _, err := text.Read(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Read(missing): want error, got nil")
	}
}

func TestWrite_PreservesTrailingNewline(t *testing.T) {
	cases := []struct {
		name string
		prev string
		next string
		want string
	}{
		{"with trailing newline", "1.2.3\n", "1.2.4", "1.2.4\n"},
		{"without trailing newline", "1.2.3", "1.2.4", "1.2.4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "VERSION")
			if err := os.WriteFile(path, []byte(tc.prev), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := text.Write(path, tc.next); err != nil {
				t.Fatalf("Write: %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("file after Write = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadWrite_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(path, []byte("1.2.3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cur, err := text.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if cur != "1.2.3" {
		t.Fatalf("Read = %q, want %q", cur, "1.2.3")
	}
	if err := text.Write(path, "1.2.4"); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1.2.4\n" {
		t.Errorf("round-trip body = %q, want %q", got, "1.2.4\n")
	}
}
