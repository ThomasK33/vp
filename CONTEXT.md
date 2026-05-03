# vp

A small, language-agnostic CLI for staging semver bump intent in Git. Stages **plans** that collapse into version-file updates at release time. Deliberately does *not* generate changelogs or release notes.

## Language

**Plan**:
A YAML file under `.version-plans/` declaring intended semver bumps for one or more components. The unit of staging.
_Avoid_: intent, changeset, bump request

**Component**:
A unit that gets independently versioned — e.g. `cli`, `agent`, `helm`. Defined in `vp.yaml` with a path glob and a version-file target.
_Avoid_: package, module, project

**Bump**:
A semver level applied to a component: `major` | `minor` | `patch` | `none`. `none` is an explicit "no release" declaration that still satisfies `vp check`.
_Avoid_: increment, version change

**Pending plan**:
Any plan currently sitting in `.version-plans/` that has not yet been consumed by `vp apply`.
_Avoid_: unapplied plan, staged plan

**Affected component**:
A component whose path globs match at least one file in `git diff <base>...<head>`. Determined by `vp check`.
_Avoid_: changed component, dirty component

**Apply**:
The act of collapsing pending plans into final bumps, writing version files, and consuming the plans (delete or archive).
_Avoid_: release, publish, ship

**Version file**:
The file owning a component's canonical version string — e.g. `package.json`, `Cargo.toml`, `Chart.yaml`. Configured per component with a format and dotted path.

## Relationships

- A **Plan** declares one or more **Bumps**, each targeting a **Component**.
- A **Component** has exactly one **Version file**.
- `vp check` finds **Affected components** and verifies each is covered by at least one **Pending plan**.
- `vp apply` collapses all **Pending plans** for each **Component** using `major > minor > patch > none`, writes the **Version file**, and consumes the plans.

## Flagged ambiguities

- "intent" vs "plan" — resolved as **plan**. North-star tagline ("git-tracked semver intent tool") may keep "intent" as descriptive prose, but in-product language is uniformly **plan**.
