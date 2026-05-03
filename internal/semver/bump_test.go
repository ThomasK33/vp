package semver_test

import (
	"testing"

	"github.com/ThomasK33/vp/internal/semver"
)

func TestParseBump_AcceptsCanonicalStrings(t *testing.T) {
	cases := map[string]semver.Bump{
		"none":  semver.BumpNone,
		"patch": semver.BumpPatch,
		"minor": semver.BumpMinor,
		"major": semver.BumpMajor,
	}
	for s, want := range cases {
		got, err := semver.ParseBump(s)
		if err != nil {
			t.Errorf("ParseBump(%q): unexpected error: %v", s, err)
			continue
		}
		if got != want {
			t.Errorf("ParseBump(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestParseBump_RejectsUnknown(t *testing.T) {
	for _, s := range []string{"", "Major", "MINOR", "huge", "release", " minor"} {
		if _, err := semver.ParseBump(s); err == nil {
			t.Errorf("ParseBump(%q): want error, got nil", s)
		}
	}
}

func TestCollapse(t *testing.T) {
	cases := []struct {
		name string
		in   []semver.Bump
		want semver.Bump
	}{
		{"empty", nil, semver.BumpNone},
		{"single none", []semver.Bump{semver.BumpNone}, semver.BumpNone},
		{"single patch", []semver.Bump{semver.BumpPatch}, semver.BumpPatch},
		{"single minor", []semver.Bump{semver.BumpMinor}, semver.BumpMinor},
		{"single major", []semver.Bump{semver.BumpMajor}, semver.BumpMajor},
		{"patch beats none", []semver.Bump{semver.BumpNone, semver.BumpPatch}, semver.BumpPatch},
		{"minor beats patch", []semver.Bump{semver.BumpPatch, semver.BumpMinor}, semver.BumpMinor},
		{"major beats minor", []semver.Bump{semver.BumpMinor, semver.BumpMajor}, semver.BumpMajor},
		{"major beats all", []semver.Bump{semver.BumpNone, semver.BumpMajor, semver.BumpPatch, semver.BumpMinor}, semver.BumpMajor},
		{"order independent", []semver.Bump{semver.BumpMajor, semver.BumpNone}, semver.BumpMajor},
		{"all none", []semver.Bump{semver.BumpNone, semver.BumpNone, semver.BumpNone}, semver.BumpNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := semver.Collapse(tc.in); got != tc.want {
				t.Errorf("Collapse(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
