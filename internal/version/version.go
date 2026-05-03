// Package version exposes the vp binary's version string.
//
// Version is sourced from the embedded VERSION file (managed by vp itself
// via vp.yaml). Release builds override it via
// `-ldflags "-X github.com/ThomasK33/vp/internal/version.Version=..."`,
// which goreleaser does on tag-triggered builds so snapshot suffixes win
// over the embedded baseline.
package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var embedded string

// Version is the vp binary's version string. It defaults to the embedded
// VERSION file contents (whitespace stripped) and is overridden at link
// time by goreleaser for tag builds.
var Version = strings.TrimSpace(embedded)
