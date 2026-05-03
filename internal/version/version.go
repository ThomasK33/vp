// Package version exposes the vp binary's version string.
//
// Version is a var (not a const) so that release builds can override it via
// `-ldflags "-X github.com/ThomasK33/vp/internal/version.Version=..."`.
package version

// Version is the vp binary's version string. Release builds override it via ldflags.
var Version = "0.1.0"
