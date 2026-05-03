package plan_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ThomasK33/vp/internal/plan"
)

func TestFilename_FallsBackWhenNoMessage(t *testing.T) {
	when := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	p, err := plan.New(map[string]string{"cli": "minor", "agent": "patch"}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := plan.Filename(p, when)
	want := "2026-05-03-add-agent-cli.yaml"
	if got != want {
		t.Errorf("Filename = %q, want %q", got, want)
	}
}

func TestFilename_FallsBackWhenSlugSanitisesEmpty(t *testing.T) {
	when := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	p, err := plan.New(map[string]string{"cli": "minor"}, "!!! ???")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := plan.Filename(p, when)
	want := "2026-05-03-add-cli.yaml"
	if got != want {
		t.Errorf("Filename = %q, want %q", got, want)
	}
}

func TestFilename_TruncatesLongSlugAtWordBoundary(t *testing.T) {
	when := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	p, err := plan.New(
		map[string]string{"cli": "minor"},
		"This is a fairly long message that should be truncated cleanly at a word boundary",
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := plan.Filename(p, when)
	const datePrefix = "2026-05-03-"
	const ext = ".yaml"
	if !strings.HasPrefix(got, datePrefix) || !strings.HasSuffix(got, ext) {
		t.Fatalf("Filename = %q, want %s<slug>%s", got, datePrefix, ext)
	}
	slug := strings.TrimSuffix(strings.TrimPrefix(got, datePrefix), ext)
	if len(slug) > 50 {
		t.Errorf("slug %q exceeds 50 chars (len=%d)", slug, len(slug))
	}
	if strings.HasSuffix(slug, "-") || strings.HasPrefix(slug, "-") {
		t.Errorf("slug %q has leading/trailing dash", slug)
	}
	// Truncation must happen on a word boundary: the slug must be a prefix
	// of the fully-slugified message split on dashes.
	full := "this-is-a-fairly-long-message-that-should-be-truncated-cleanly-at-a-word-boundary"
	if !strings.HasPrefix(full, slug) {
		t.Errorf("slug %q is not a word-boundary prefix of %q", slug, full)
	}
	for word := range strings.SplitSeq(slug, "-") {
		if word == "" {
			t.Errorf("slug %q contains empty word (mid-word truncation)", slug)
		}
	}
}

func TestFilename_HyphenatedComponentNamesNoDoubleDash(t *testing.T) {
	when := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	p, err := plan.New(map[string]string{"web-app": "minor", "cli": "patch"}, "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got := plan.Filename(p, when)
	if strings.Contains(got, "--") {
		t.Errorf("Filename %q contains a double-dash", got)
	}
	want := "2026-05-03-add-cli-web-app.yaml"
	if got != want {
		t.Errorf("Filename = %q, want %q", got, want)
	}
}

func TestFilename_UsesUTCDate(t *testing.T) {
	// 2026-05-03 23:30 in a +05:00 zone is 2026-05-03 18:30 UTC.
	// Confirms the date in the filename is computed in UTC, not local time.
	zone := time.FixedZone("test", 5*60*60)
	when := time.Date(2026, 5, 4, 2, 30, 0, 0, zone) // 2026-05-03 21:30 UTC

	p, err := plan.New(map[string]string{"cli": "minor"}, "Fix reconnect")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got := plan.Filename(p, when)
	want := "2026-05-03-fix-reconnect.yaml"
	if got != want {
		t.Errorf("Filename = %q, want %q", got, want)
	}
}
