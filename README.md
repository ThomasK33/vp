# vp

A small, language-agnostic CLI for staging semver bump intent in Git.
`vp` stages **plans** that collapse into version-file updates at release
time. It deliberately does *not* generate changelogs, create git tags,
or publish anything.

## What `vp` is not

- **Not a changelog generator.** Pair `vp` with a tool like
  [`communiqué`](https://github.com/jdx/communique) for release notes.
- **Not a git driver.** `vp` never runs `git commit`, `git tag`, or
  `git push`. It prints the intended tag string; your CI executes it.
- **Not a publisher.** No `npm publish`, `cargo publish`, `helm push`.
- **Not dependency-graph aware.** If component A depends on component
  B, `vp` will not auto-bump A when B bumps. Declare both in your plan.
- **Not ecosystem-coupled.** `vp` reads the version files you point it
  at; it does not assume Node, Cargo, Helm, or anything else.

## Install

```sh
# Go users:
go install github.com/ThomasK33/vp@latest

# Anyone: download a prebuilt binary for darwin/linux/windows
# × amd64/arm64 from https://github.com/ThomasK33/vp/releases
```

## Quickstart

```sh
# 1. Initialise the config and edit it for your repo.
vp init                                # writes ./vp.yaml
$EDITOR vp.yaml                        # set components[*].paths and version.file

# 2. Stage a bump alongside your code change.
vp add cli patch -m "Fix reconnect race"
# wrote .version-plans/2026-05-03-fix-reconnect-race.yaml

# 3. Inspect what is pending.
vp status
# Pending plans:
#   2026-05-03-fix-reconnect-race.yaml
#     cli: patch
#
# Resolved bumps:
#   cli: patch (1.2.3 -> 1.2.4)

# 4. Preview, then apply at release time.
vp apply --dry-run
# component  current  next   bump   file         tag
# cli        1.2.3    1.2.4  patch  cli/VERSION  cli-v1.2.4
vp apply
# cli  1.2.3 -> 1.2.4  (patch)  cli/VERSION  cli-v1.2.4
# Wrote 1 file(s); consumed 1 plan(s).
```

That's the whole workflow. Step 1 happens once per repo; steps 2–4
repeat per release.

## Worked example: a polyglot repo

A repo with a Go CLI, a Helm chart, a published Node SDK, and a Rust
agent — one `vp.yaml` covers all four:

```yaml
# vp.yaml
plans:
  dir: .version-plans
  consumed: delete           # or "archive" + archive_dir

components:
  cli:
    paths: ["cli/**", "cmd/**"]
    version:
      file: cli/VERSION
      format: text
    tag: "cli-v{version}"

  helm:
    paths: ["charts/example/**"]
    version:
      file: charts/example/Chart.yaml
      format: yaml
      path: version
    tag: "helm-v{version}"

  node-sdk:
    paths: ["sdk/node/**"]
    version:
      file: sdk/node/package.json
      format: json
      path: version
    tag: "node-sdk-v{version}"

  rust-agent:
    paths: ["agent/**"]
    version:
      file: agent/Cargo.toml
      format: toml
      path: package.version
    tag: "agent-v{version}"
```

A single PR can touch multiple components, and one plan file can
declare multiple bumps:

```yaml
# .version-plans/2026-05-03-cross-cutting.yaml
releases:
  cli: minor
  node-sdk: patch
  rust-agent: none           # touched but not releasing
message: Add reconnect flag and SDK helper; agent docs only.
```

`vp status` resolves all four:

```text
Pending plans:
  2026-05-03-cross-cutting.yaml
    cli: minor
    node-sdk: patch
    rust-agent: none

Resolved bumps:
  cli: minor (1.2.3 -> 1.3.0)
  node-sdk: patch (0.4.1 -> 0.4.2)
  rust-agent: none (0.9.0)
```

## Dogfooding: vp manages its own version

This repo's [`vp.yaml`](./vp.yaml) is the smallest possible real-world
example — one component, one text file, one tag template:

```yaml
components:
  vp:
    paths: ["cmd/**", "internal/**", "main.go", "go.mod", "go.sum"]
    version:
      file: internal/version/VERSION
      format: text
    tag: "v{version}"
```

The string reported by `vp --version` is the contents of
[`internal/version/VERSION`](./internal/version/VERSION), embedded into
the binary via `//go:embed`. Cutting a release is `vp add vp <bump>`
→ `vp apply` → commit → tag — goreleaser does the rest.

## Release flow with a separate changelog tool

`vp` prints intended tag strings; your release pipeline executes them.
Compose with any changelog tool. Here is the flow with
[`communiqué`](https://github.com/jdx/communique):

```sh
# 1. Resolve and apply pending plans. --json so a release script can
#    consume the resulting tag values without parsing tables.
vp apply --json > apply.json

# 2. Commit the version-file edits and the consumed plans, so tags
#    can reference the bump commit.
git add -A && git commit -m "release"

# 3. Tag every component that bumped, using the tag string vp printed.
jq -r '.changes[].tag | select(. != null and . != "")' apply.json |
  while read -r tag; do git tag "$tag"; done

# 4. Generate release notes per tag, then push commit and tags.
jq -r '.changes[].tag | select(. != null and . != "")' apply.json |
  while read -r tag; do communique generate "$tag"; done
git push --follow-tags
```

`vp` never runs any of those `git` or `communique` commands itself —
this is your pipeline, plain shell.

## CI integration

Two jobs: a **PR gate** that fails when a PR changes a component
without staging a plan, and a **release** job that triggers on a
maintainer-pushed tag (or release branch). Both pin the toolchain
through [`mise`](https://mise.jdx.dev/) and
[`jdx/mise-action`](https://github.com/jdx/mise-action), matching the
patterns this repo uses for its own CI.

```yaml
# .github/workflows/release-gate.yml
name: release-gate
on: pull_request

permissions:
  contents: read

jobs:
  vp-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0           # vp check needs both refs
      - uses: jdx/mise-action@v4    # pins go, golangci-lint, etc. via mise.toml
      - run: go install github.com/ThomasK33/vp@latest
      - name: vp check
        run: vp check --base origin/${{ github.base_ref }} --head HEAD
```

`vp check` exits `1` when a component is affected by the diff but no
pending plan covers it. Other failures (missing config, bad arguments,
git error) exit `2` or `3`, so a CI script can branch on the specific
exit code:

```sh
vp check --base origin/main --head HEAD
case $? in
  0) echo "ok" ;;
  1) echo "missing plan coverage; add one with: vp add <component> <bump>" ;;
  *) echo "vp configuration or runtime error" ;;
esac
```

For releases, run `vp apply --json` from a tag-triggered workflow and
hand the output to your changelog/publish step:

```yaml
# .github/workflows/release.yml
on:
  push:
    tags: ['v*']
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jdx/mise-action@v4
      - run: go install github.com/ThomasK33/vp@latest
      - run: vp apply --json | tee apply.json
      # ... your existing publish + changelog steps consume apply.json
```

## Configuration reference (`vp.yaml`)

Loaded by walking up from the current directory until `vp.yaml` is
found, so you can run `vp` from any subdirectory of the repo.

```yaml
plans:
  dir: <string>            # required. directory holding pending plans.
  consumed: <string>       # optional. "delete" (default) | "archive".
  archive_dir: <string>    # required when consumed == "archive".

components:
  <name>:
    paths:                 # required. one or more globs (doublestar **).
      - "<glob>"
    version:
      file: <string>       # required. path to the version file.
      format: <string>     # required. one of: json | yaml | toml | text.
      path: <string>       # dotted accessor; omit for text format.
    tag: <string>          # optional. template with single {version} placeholder.
```

Per-format notes:

- **`text`** — entire file body is the version (whitespace stripped on
  read; trailing newline preserved on write). Omit `path`.
- **`json`** — dotted path into the JSON tree, e.g. `version`,
  `package.version`.
- **`yaml`** — dotted path into the YAML tree. Comments, anchors, and
  unrelated keys are preserved verbatim.
- **`toml`** — dotted path including TOML table headers, e.g.
  `package.version` for `[package]` then `version = "..."`.

All paths in `vp.yaml` are resolved relative to the directory
containing `vp.yaml`, not the working directory.

## Authoring a plan without `vp`

`vp add` is a convenience. Anyone — human or coding agent — can
author a plan with file-write capability alone. The rules:

**Filename:** `YYYY-MM-DD-<short-kebab-slug>.yaml` under the configured
`plans.dir` (default `.version-plans/`). The date is UTC. The slug is
free-form; collisions surface as ordinary git merge conflicts, which
is the correct outcome.

**Schema:** plain YAML with two keys.

```yaml
# .version-plans/2026-05-03-add-reconnect-flag.yaml
releases:
  <component>: <bump>      # bump is one of: major | minor | patch | none
  # ...
message: optional human-readable note shown by `vp status`.
```

`releases` must declare at least one component that exists in
`vp.yaml`. `none` is an explicit "touched but not releasing"
declaration; it satisfies `vp check` and is consumed by `vp apply`
without writing any version file.

This makes `vp` agent-friendly: a coding agent can read this README
plus `CONTEXT.md` and produce a valid plan file with nothing more
than file I/O. No binary, no network, no tooling installed.

## Command reference

| Command | Purpose | Notable flags |
|---|---|---|
| `vp init` | Write a starter `vp.yaml`. | `--force` (overwrite existing). |
| `vp add <comp> <bump>` <br> `vp add a=patch b=minor` | Write a plan file. | `-m, --message` (free text). |
| `vp status` | Print pending plans and resolved bumps. | `--all` (every component, not just bumped); `--json`. |
| `vp apply` | Collapse plans, write version files, consume plans. | `--dry-run` (preview only); `--json`. |
| `vp check --base <ref> --head <ref>` | Fail if any affected component is missing a plan. | `--json`. |

**Exit codes:**

| Code | Meaning |
|---|---|
| `0` | success |
| `1` | `vp check` failure: at least one affected component has no pending plan |
| `2` | usage or configuration error (bad flags, missing `vp.yaml`, unknown component, malformed plan) |
| `3` | runtime error (file I/O, git, version parse) |

## Comparison

Tools in the same neighbourhood, scored on the dimensions that matter
when picking one:

| | `vp` | [Changesets](https://github.com/changesets/changesets) | [Nx version plans](https://nx.dev/recipes/nx-release/configure-custom-versioning-logic) | [release-please](https://github.com/googleapis/release-please) |
|---|---|---|---|---|
| **Bump source** | Intent files (plans) | Intent files (changesets) | Intent files (version plans) | Conventional Commits in git history |
| **Language scope** | Any (json/yaml/toml/text version files) | JavaScript / npm packages | Whatever Nx manages (JS-leaning) | Multi-language (Go, Node, Rust, Python, …) |
| **Changelog generation** | No (out of scope) | Yes | Yes | Yes |
| **Ecosystem assumptions** | None — points at version files you configure | Assumes npm workspaces / package.json | Assumes an Nx workspace | Assumes Conventional Commits + release-PR workflow |

Pick `vp` when you want a small primitive that just answers *"what
needs a bump because of this PR?"*, and you already have (or want to
choose separately) a changelog and release-tagging tool.

## JSON output reference

`vp status`, `vp apply` (incl. `--dry-run`), and `vp check` accept
`--json` to emit a stable machine-readable payload on stdout. Plain
text remains the default. The `--json` flag only affects stdout —
error messages always go to stderr unchanged. The shape of every
payload is pinned by golden files in `cmd/testdata/output/json/`;
breaking changes will require a major-version bump once vp reaches
`v1`.

### `vp status --json`

```json
{
  "pending": [
    {
      "file": "2026-05-03-fix.yaml",
      "releases": { "cli": "minor" },
      "message": "fix the thing"
    }
  ],
  "resolved": [
    { "component": "cli", "bump": "minor", "current": "1.2.3", "next": "1.3.0" }
  ]
}
```

`pending` is sorted by filename. `releases` is the raw map from the
plan file. `message` is omitted when empty. `resolved` is sorted by
component name; with `--all`, every configured component appears,
including those with no pending bump (`bump: "none"`). `current` and
`next` are populated when the component's version file is readable in
any supported format (text/json/yaml/toml); they are omitted on read
failure. `next` is also omitted when the resolved bump is `none`.

### `vp apply --json` and `vp apply --dry-run --json`

```json
{
  "changes": [
    {
      "component": "cli",
      "current": "1.2.3",
      "next": "1.3.0",
      "bump": "minor",
      "file": "VERSION",
      "tag": "cli-v1.3.0"
    }
  ],
  "consumed": ["2026-05-03-bump.yaml"]
}
```

Both modes share one shape. `changes` is sorted by component name.
`file` is repo-relative. `tag` is omitted when no template is
configured for the component. `consumed` lists basenames of plans
that were removed (in `delete` mode) or moved to the archive
directory (in `archive` mode); it is always present as an empty
array under `--dry-run`.

### `vp check --json`

```json
{
  "affected": ["cli"],
  "planned": ["cli"],
  "missing": []
}
```

All three arrays are always present (never `null`) and sorted
alphabetically. Exit code `1` is still returned when `missing` is
non-empty; the JSON payload is emitted on stdout regardless, and the
error message goes to stderr.

## Further reading

- [`CONTEXT.md`](./CONTEXT.md) — domain glossary (Plan, Component,
  Bump, Pending plan, Affected component, Apply, Version file).
- [`docs/adr/`](./docs/adr/) — architectural decision records:
  [stack-coverage check semantics](./docs/adr/0001-stack-coverage-check-semantics.md),
  [surgical version-file edits](./docs/adr/0002-surgical-version-file-edits.md),
  [stable-promotion semver bumps](./docs/adr/0003-stable-promotion-semver-bumps.md).

## License

MIT — see [`LICENSE`](./LICENSE).
