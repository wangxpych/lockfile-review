# lockfile-review

[![CI](https://github.com/wangxpych/lockfile-review/actions/workflows/ci.yml/badge.svg)](https://github.com/wangxpych/lockfile-review/actions/workflows/ci.yml)
[![release](https://img.shields.io/github/v/release/wangxpych/lockfile-review)](https://github.com/wangxpych/lockfile-review/releases)
[![license](https://img.shields.io/github/license/wangxpych/lockfile-review)](./LICENSE)

Review what a pnpm lockfile changed—not only whether its new packages are
vulnerable.

`lockfile-review` is a deterministic GitHub Action and single-file CLI that
detects:

- dependencies changed outside every updated direct dependency graph;
- unexpected resolved-version downgrades;
- `package.json` and root importer specifier drift;
- lockfile-only updates without an explicitly named target; and
- purely textual YAML churn with no semantic change.

It complements vulnerability and license scanners. It does not contact a
registry, execute a package manager, install dependencies, or run lifecycle
scripts.

## GitHub Action

```yaml
name: Lockfile review

on:
  pull_request:
    paths:
      - package.json
      - pnpm-lock.yaml

permissions:
  contents: read

jobs:
  lockfile-review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0
      - uses: wangxpych/lockfile-review@v0
```

Pull request events supply immutable base and head commits automatically. If a
revision is missing from the checkout, the Action attempts a depth-one fetch.
The result appears in both the job log and job summary.

For a lockfile-only security update, name its intended package explicitly:

```yaml
      - uses: wangxpych/lockfile-review@v0
        with:
          expected-packages: ws
```

Multiple names use a comma-separated list. Updated `package.json` dependencies
are inferred automatically and do not need to be listed.

## CLI

Download a binary from [GitHub Releases](https://github.com/wangxpych/lockfile-review/releases),
then compare two Git revisions:

```bash
lockreview --base-ref origin/main --head-ref HEAD
```

Review a project below the repository root:

```bash
lockreview \
  --base-ref origin/main \
  --head-ref HEAD \
  --root packages/web
```

Review two materialized directories or fixtures:

```bash
lockreview --base-dir before --head-dir after --format json
```

Exit status is `0` when the selected policy passes, `1` for review findings
that violate policy, and `2` for invalid input or execution errors.

## Example finding

```text
lockfile-review: failed
pnpm lockfile version: 9.0
changed direct dependencies: typescript
findings:
- [warning] unexpected-downgrade @types/node: highest resolved semantic version decreased (26.4.0 -> 26.3.0)
- [warning] unrelated-lockfile-change @types/node: dependency changed outside every updated direct dependency graph (26.4.0 -> 26.3.0)
- [warning] unexpected-downgrade rolldown: highest resolved semantic version decreased (1.2.6 -> 1.2.5)
- [warning] unrelated-lockfile-change rolldown: dependency changed outside every updated direct dependency graph (1.2.6 -> 1.2.5)
- [info] requested-change typescript: direct dependency changed in package.json and pnpm-lock.yaml (6.0.3 -> 7.0.2)
```

## Policy inputs

| Input | Default | Meaning |
| --- | --- | --- |
| `fail-on-unrelated` | `true` | Fail when a changed package is outside every inferred or explicit update graph. |
| `fail-on-downgrade` | `true` | Fail when the highest resolved semantic version for a package decreases. |
| `expected-packages` | empty | Additional update roots for lockfile-only changes. |
| `root` | repository root | Project directory containing the manifest and lockfile. |
| `manifest-path` | `package.json` | Manifest path relative to `root`. |
| `lockfile-path` | `pnpm-lock.yaml` | Lockfile path relative to `root`. |
| `format` | `text` | Console output: `text`, `markdown`, or `json`. |

Manifest/lockfile mismatches always fail because a frozen install cannot safely
rely on inconsistent intent.

## Current scope

The `0.1` line supports pnpm lockfile version 9 and the root importer (`.`).
Workspace projects can be reviewed independently with `root`. Git, HTTP, file,
workspace, and portal dependency edges are treated as external graph boundaries.

The classifier is deliberately conservative. Shared resolution, peer context,
and package-manager rewrites can produce a legitimate warning; use
`expected-packages` to state review intent rather than disabling all unrelated
change detection.

See the [documentation index](./docs/README.md), [review rules](./docs/rules.md),
and [architecture](./docs/architecture.md) for details.

## Development

Go 1.25 or newer is supported. CI also tests the current Go 1.27 release.

```bash
make verify
docker build -t lockfile-review:local .
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) before opening a pull request.

## License

MIT
