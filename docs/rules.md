# Review rules

`lockfile-review` compares dependency intent in `package.json` with resolution
state in `pnpm-lock.yaml`. It does not decide whether a package is trustworthy;
it decides whether the observed lockfile change fits the declared update scope.

## Update roots

An update root comes from either:

1. a direct dependency declaration added, removed, moved between scopes, or
   changed in `package.json`; or
2. an explicit `expected-packages` entry, intended for lockfile-only updates.

The reviewer walks dependency edges from each root in both the base and head
lockfiles. Any changed package reachable in either graph is considered in
scope. Looking at both sides avoids classifying removed and newly introduced
transitive packages as unrelated.

## Findings

### `requested-change`

The package is an inferred or explicit update root. Informational.

### `transitive-change`

The package changed inside an update root's dependency graph. Informational.

### `unrelated-lockfile-change`

The package changed outside all update graphs. This is a warning and fails by
default. It can expose stale bot branches, non-reproducible package-manager
output, accidental lockfile maintenance, or a grouped change larger than its
manifest diff suggests.

### `unexpected-downgrade`

The highest resolved `major.minor.patch` version decreased. This warning fails
by default. Pre-release labels and build metadata do not affect the comparison;
non-semantic versions are excluded from downgrade analysis.

### `manifest-lockfile-mismatch`

A non-peer dependency is missing from one side of the root importer, or its
specifier differs from `package.json`. Extra root-importer dependencies are
also mismatches. This structural error always fails.

### `cosmetic-lockfile-churn`

The lockfile text changed while parsed YAML data stayed identical. This is a
warning but does not fail because format normalization may be intentional.

## Policy guidance

Keep both failure policies enabled in protected branches. For a known grouped
or transitive update, name the additional roots with `expected-packages` so the
review remains explicit and auditable.

Avoid using `fail-on-unrelated: false` as a permanent substitute for declaring
intent. Advisory mode is appropriate while first adopting the Action or
investigating an existing noisy lockfile workflow.
