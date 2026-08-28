# Architecture

The project is split into small layers so lockfile semantics remain independent
of GitHub Actions and report formatting.

```text
directory / Git revisions / pull request event
                    |
                    v
             immutable input bytes
                    |
          +---------+----------+
          |                    |
    package.json parser   pnpm v9 parser
          |                    |
          +---------+----------+
                    |
        dependency graph classification
                    |
          text / Markdown / JSON
                    |
          CLI exit code + job summary
```

## Boundaries

- `internal/input` reads directories and Git objects. It may fetch a missing
  immutable revision but never modifies the working tree.
- `internal/manifest` owns direct dependency intent.
- `internal/pnpm` owns pnpm importer, package, snapshot, and graph semantics.
- `internal/review` classifies changes and applies no presentation concerns.
- `internal/report` renders stable finding codes for humans and automation.
- `cmd/lockreview` maps flags, Action inputs, GitHub event context, and exit
  status onto those layers.

## Determinism

The same manifest and lockfile bytes, expected package list, and policy always
produce the same result. The core performs no registry lookup, clock read,
network request, dependency installation, or model inference.

The only optional network operation is an input-layer `git fetch` when a named
base or head revision is absent. Classification begins only after both exact Git
objects have been loaded.

## Security model

Lockfiles and manifests are untrusted data. The Action parses them but never
executes package scripts or package-manager commands. Project paths must remain
relative to the repository, and Git revisions are passed as individual process
arguments rather than interpreted by a shell.

The Docker image runs the statically linked reviewer with Git and CA
certificates. It does not require write permissions to repository contents or
pull requests.
