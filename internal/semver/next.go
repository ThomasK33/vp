package semver

import (
	"fmt"
	"strings"

	masterminds "github.com/Masterminds/semver/v3"
)

// Next returns the version string produced by applying level to version.
// An optional leading "v" is preserved on output. For major/minor/patch bumps,
// any prerelease and build metadata are dropped per ADR-0003. BumpNone returns
// version unchanged after parse validation.
func Next(version string, level Bump) (string, error) {
	body, hadV := strings.CutPrefix(version, "v")
	v, err := masterminds.NewVersion(body)
	if err != nil {
		return "", fmt.Errorf("parse version %q: %w", version, err)
	}
	var next masterminds.Version
	switch level {
	case BumpNone:
		return version, nil
	case BumpPatch:
		next = v.IncPatch()
	case BumpMinor:
		next = v.IncMinor()
	case BumpMajor:
		next = v.IncMajor()
	default:
		return "", fmt.Errorf("unknown bump %q", level)
	}
	out := next.String()
	if hadV {
		out = "v" + out
	}
	return out, nil
}
