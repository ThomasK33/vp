package plan_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ThomasK33/vp/internal/plan"
)

func TestNew_RejectsEmptyReleases(t *testing.T) {
	if _, err := plan.New(map[string]string{}, ""); err == nil {
		t.Fatal("New: want error for empty releases, got nil")
	}
	if _, err := plan.New(nil, ""); err == nil {
		t.Fatal("New: want error for nil releases, got nil")
	}
}

func TestNew_RejectsEmptyComponentName(t *testing.T) {
	_, err := plan.New(map[string]string{"": "minor"}, "")
	if err == nil {
		t.Fatal("New: want error for empty component name, got nil")
	}
}

func TestNew_RejectsEmptyBump(t *testing.T) {
	_, err := plan.New(map[string]string{"cli": ""}, "")
	if err == nil {
		t.Fatal("New: want error for empty bump, got nil")
	}
}

func TestNew_RejectsUnknownBump(t *testing.T) {
	_, err := plan.New(map[string]string{"cli": "huge"}, "")
	if err == nil {
		t.Fatal("New: want error for unknown bump, got nil")
	}
}

func TestNew_AcceptsAllBumps(t *testing.T) {
	for _, bump := range []string{"major", "minor", "patch", "none"} {
		if _, err := plan.New(map[string]string{"cli": bump}, ""); err != nil {
			t.Errorf("New(cli=%s): unexpected error: %v", bump, err)
		}
	}
}

func TestSave_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "2026-05-03-existing.yaml")

	first, err := plan.New(map[string]string{"cli": "minor"}, "first")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := plan.Save(first, target); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}

	second, err := plan.New(map[string]string{"cli": "patch"}, "second")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := plan.Save(second, target); err == nil {
		t.Fatal("second Save: want error, got nil")
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("re-read target: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("file modified despite refusal:\nwant %q\ngot  %q", original, got)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "2026-05-03-roundtrip.yaml")

	p, err := plan.New(map[string]string{"cli": "minor", "agent": "patch"}, "Fix reconnect")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := plan.Save(p, target); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := plan.Load(target)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !reflect.DeepEqual(got.Releases, p.Releases) {
		t.Errorf("Releases roundtrip mismatch: got %v, want %v", got.Releases, p.Releases)
	}
	if got.Message != p.Message {
		t.Errorf("Message = %q, want %q", got.Message, p.Message)
	}
}
