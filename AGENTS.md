# Repository guide

This repository contains a deterministic Go CLI and Docker GitHub Action.

Before changing public behavior, read `README.md`, `docs/README.md`,
`docs/rules.md`, `docs/architecture.md`, `CONTRIBUTING.md`, and `RELEASING.md`.

Keep package-manager parsing in `internal/pnpm`, classification in
`internal/review`, and presentation in `internal/report`. Core review logic must
not execute package-manager commands, install dependencies, query registries,
or use nondeterministic model output.

Run `make verify` and build the Docker image before release-affecting changes.
Every new finding or policy boundary requires a focused fixture and an update to
`docs/rules.md`.

Public documentation and commit messages are English-first. Never commit
credentials, private URLs, private repository content, or generated caches.
