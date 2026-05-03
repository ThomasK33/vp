# vp

A small, language-agnostic CLI for staging semver bump intent in Git. Stages
**plans** that collapse into version-file updates at release time. Deliberately
does *not* generate changelogs or release notes.

> Status: pre-v0.1, under construction.

See [`CONTEXT.md`](./CONTEXT.md) for the domain glossary and
[`docs/adr/`](./docs/adr/) for architectural decisions.

## Output formats

`vp status`, `vp apply` (incl. `--dry-run`), and `vp check` accept `--json`
to emit a stable machine-readable payload on stdout. Plain text remains the
default. The `--json` flag only affects stdout — error messages always go to
stderr unchanged. The shape of every payload is pinned by golden files in
`cmd/testdata/output/json/`; breaking changes will require a major-version
bump once vp reaches `v1`.

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

`pending` is sorted by filename. `releases` is the raw map from the plan
file. `message` is omitted when empty. `resolved` is sorted by component
name; with `--all`, every configured component appears, including those
with no pending bump (`bump: "none"`). `current` and `next` are populated
when the component's version file is readable in any supported format
(text/json/yaml/toml); they are omitted on read failure. `next` is also
omitted when the resolved bump is `none`.

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

Both modes share one shape. `changes` is sorted by component name. `file`
is repo-relative. `tag` is omitted when no template is configured for the
component. `consumed` lists basenames of plans that were removed (in
`delete` mode) or moved to the archive directory (in `archive` mode);
it is always present as an empty array under `--dry-run`.

### `vp check --json`

```json
{
  "affected": ["cli"],
  "planned": ["cli"],
  "missing": []
}
```

All three arrays are always present (never `null`) and sorted alphabetically.
Exit code `1` is still returned when `missing` is non-empty; the JSON
payload is emitted on stdout regardless, and the error message goes to
stderr.

## License

MIT — see [`LICENSE`](./LICENSE).
