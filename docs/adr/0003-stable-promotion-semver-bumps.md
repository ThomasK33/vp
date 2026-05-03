# Bumps promote prereleases to the stable line; no `prerelease` bump type in v1

vp accepts version strings with prerelease (`-rc.1`) and build metadata (`+build.42`) on input via `Masterminds/semver/v3`. Any `major`/`minor`/`patch` **Bump** drops both prerelease and build metadata before incrementing — `1.2.3-rc.1 + patch → 1.2.3`, `1.2.3-rc.1 + minor → 1.3.0`. The `none` **Bump** leaves the version string untouched, so prerelease tags persist for that path.

We chose this over preserving prerelease tags through bumps (no library does it by default, and rolling our own opens edge cases like `-rc.1` → `-rc.2`) and over adding a dedicated `prerelease` **Bump** type (that requires owning a whole prerelease-tag subsystem: counter rules, tag transitions like `alpha → beta`, mixed numeric/non-numeric tags). Stable-line promotion matches npm, Cargo, and Helm conventions.

Teams that need to ship multiple releases on a prerelease line (e.g. `1.0.0-beta.1`, `-beta.2`, …) edit the version file manually in v1. A `prerelease` bump type and/or a `--keep-prerelease` flag may arrive in a later version once the use case is concrete.
