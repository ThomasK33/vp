# Surgical edits to version files via validate-then-splice

`vp apply` writes version files in two phases per file: **(1) parse with the format's standard library** to validate the configured path exists and resolves to a string value parseable as semver; **(2) splice the value at byte level** using the position information surfaced by parsing — never round-tripping the document through a marshaler. The rest of the file is preserved byte-for-byte: key order, indentation, comments, trailing whitespace, line endings.

We chose this over the simpler round-trip approach (decode → mutate → re-encode) because round-trip produces noisy multi-line diffs in real-world `package.json` and `Cargo.toml` files, triggers downstream formatter/lint hooks, and makes the tool feel hostile in code review. Changesets gets this right; we should too.

Implementation per format:
- **YAML** — `go.yaml.in/yaml/v3` (the maintained fork; the original `gopkg.in/yaml.v3` is effectively unmaintained). Parse to `yaml.Node` to find the target's `Line`/`Column`; splice the value span in the source bytes. We never call `yaml.Marshal` on the parsed document.
- **TOML** — `pelletier/go-toml/v2/unstable` for position-aware AST. The stable `v2` API removed `toml.Position` deliberately and won't surface it; the `unstable` subpackage exposes `Position{Offset, Line, Column}` and `Parser.Raw()`. We accept the API-instability caveat — the alternative (a hand-rolled lexer or regex) is brittler and the unstable surface area we touch is small.
- **JSON** — `encoding/json.Decoder` with `InputOffset()` to walk to the target path and recover the value's byte span; splice in place. We deliberately do **not** use `tidwall/sjson` despite initial appearances: its `Set` reformats surrounding JSON and does not guarantee key-order preservation, defeating the whole point.
- **Text** — trivial: trim trailing whitespace, replace contents, restore single trailing newline.

If the configured path does not exist in the target file, `vp apply` errors before writing anything — we never auto-create missing keys, since "where to insert" is genuinely ambiguous in YAML/TOML and is a per-project policy decision the tool shouldn't make.

The validate-and-splice approach has one residual risk: the parser tells us the *start* of a value but not always the *end*. We compute the end either from (a) the next sibling/parent's start position, (b) re-scanning the source from the start position using the format's value-token grammar, or (c) for the simple case where the value is a quoted string on a single line, by scanning forward to the closing quote. The version-string case lives almost entirely in (c) territory.
