package semver

import "fmt"

// Bump is a semver level: none, patch, minor, or major.
type Bump string

const (
	BumpNone  Bump = "none"
	BumpPatch Bump = "patch"
	BumpMinor Bump = "minor"
	BumpMajor Bump = "major"
)

// ParseBump returns the canonical Bump for s, or an error if s is not one of
// none, patch, minor, or major.
func ParseBump(s string) (Bump, error) {
	switch Bump(s) {
	case BumpNone, BumpPatch, BumpMinor, BumpMajor:
		return Bump(s), nil
	}
	return "", fmt.Errorf("unknown bump %q (want one of: none, patch, minor, major)", s)
}

// Collapse returns the highest-precedence bump in levels using
// major > minor > patch > none. An empty slice returns BumpNone.
func Collapse(levels []Bump) Bump {
	highest := BumpNone
	for _, b := range levels {
		if rank(b) > rank(highest) {
			highest = b
		}
	}
	return highest
}

func rank(b Bump) int {
	switch b {
	case BumpMajor:
		return 3
	case BumpMinor:
		return 2
	case BumpPatch:
		return 1
	}
	return 0
}
