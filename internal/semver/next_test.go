package semver_test

import (
	"testing"

	"github.com/ThomasK33/vp/internal/semver"
)

func TestNext(t *testing.T) {
	cases := []struct {
		name    string
		version string
		level   semver.Bump
		want    string
	}{
		{"patch increments patch", "1.2.3", semver.BumpPatch, "1.2.4"},
		{"minor zeros patch", "1.2.3", semver.BumpMinor, "1.3.0"},
		{"major zeros minor and patch", "1.2.3", semver.BumpMajor, "2.0.0"},
		{"v-prefix preserved on patch", "v1.2.3", semver.BumpPatch, "v1.2.4"},
		{"v-prefix preserved on major", "v1.2.3", semver.BumpMajor, "v2.0.0"},
		{"prerelease dropped on patch", "1.2.3-rc.1", semver.BumpPatch, "1.2.3"},
		{"prerelease dropped on minor", "1.2.3-rc.1", semver.BumpMinor, "1.3.0"},
		{"build metadata dropped on patch", "1.2.3+build.42", semver.BumpPatch, "1.2.4"},
		{"none returns input unchanged", "1.2.3", semver.BumpNone, "1.2.3"},
		{"none preserves prerelease", "1.2.3-rc.1", semver.BumpNone, "1.2.3-rc.1"},
		{"none preserves v prefix", "v1.2.3", semver.BumpNone, "v1.2.3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := semver.Next(tc.version, tc.level)
			if err != nil {
				t.Fatalf("Next(%q, %s): unexpected error: %v", tc.version, tc.level, err)
			}
			if got != tc.want {
				t.Errorf("Next(%q, %s) = %q, want %q", tc.version, tc.level, got, tc.want)
			}
		})
	}
}

func TestNext_RejectsUnparseableVersion(t *testing.T) {
	_, err := semver.Next("not-a-version", semver.BumpPatch)
	if err == nil {
		t.Fatal("Next(not-a-version, patch): want error, got nil")
	}
}
