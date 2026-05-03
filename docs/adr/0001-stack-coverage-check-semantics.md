# `vp check` validates stack coverage, not per-diff novelty

`vp check --base <ref> --head <ref>` succeeds when every **Affected component** (one whose path globs match a file in `git diff <base>...<head>`) is mentioned by **at least one Pending plan** currently sitting in `.version-plans/`. It does *not* require that the covering plan was added within `<base>...<head>` — a plan authored on a previous diff still satisfies the check.

We chose stack-coverage over per-diff novelty because (a) it matches Changesets' established mental model, (b) "what counts as this diff" is messy under squash-merges and rebases, and (c) the tool is a release-stack backstop, not a code-review referee — reviewers catch undersized bumps.

A future opt-in flag (working name `--strict`) may additionally require that the covering plan was *added* in `<base>...<head>`. That is deferred until a concrete need surfaces; it is not PR-centric, just diff-centric, and remains compatible with the default behaviour.
