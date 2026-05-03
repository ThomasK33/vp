package plan

import (
	"sort"
	"strings"
	"time"
)

// slugMaxLen caps the message-derived slug length. Longer slugs are truncated
// at a word boundary.
const slugMaxLen = 50

// Filename returns "YYYY-MM-DD-<slug>.yaml" using when.UTC() for the date.
// The slug is derived from p.Message, falling back to "add-<sorted-components>"
// when the message is empty or sanitises to nothing.
func Filename(p *Plan, when time.Time) string {
	slug := truncateSlug(slugFromMessage(p.Message), slugMaxLen)
	if slug == "" {
		slug = "add-" + sortedComponentSlug(p.Releases)
	}
	return when.UTC().Format("2006-01-02") + "-" + slug + ".yaml"
}

// slugFromMessage lowercases s and collapses runs of non-alphanumeric runes
// into single hyphens. Returns "" if s contains no alphanumeric runes.
func slugFromMessage(s string) string {
	var b strings.Builder
	prevDash := true
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// truncateSlug shortens slug to at most maxLen chars, ending on a word
// boundary. Words are dash-separated. If the first word alone exceeds maxLen,
// the slug is returned unchanged (better a long filename than a half-word).
func truncateSlug(slug string, maxLen int) string {
	if len(slug) <= maxLen {
		return slug
	}
	cut := strings.LastIndex(slug[:maxLen+1], "-")
	if cut <= 0 {
		return slug
	}
	return slug[:cut]
}

// sortedComponentSlug joins component names with "-" in sorted order.
// Component names are passed through slugFromMessage to ensure the result is
// safe (and to collapse any internal whitespace).
func sortedComponentSlug(releases map[string]string) string {
	names := make([]string, 0, len(releases))
	for name := range releases {
		names = append(names, slugFromMessage(name))
	}
	sort.Strings(names)
	return strings.Join(names, "-")
}
