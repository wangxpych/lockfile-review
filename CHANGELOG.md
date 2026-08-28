# Changelog

All notable changes are documented here. The project follows Semantic Versioning.

## [Unreleased]

## [0.1.1] - 2026-08-29

### Changed

- Updated the Docker Action runtime from Alpine 3.22 to Alpine 3.24.
- Updated the repository's official GitHub Actions to `actions/checkout@v7`
  and `actions/setup-go@v7`.

## [0.1.0] - 2026-08-29

### Added

- Deterministic pnpm lockfile v9 review for direct and transitive changes.
- Detection for unrelated package changes, semantic-version downgrades,
  manifest/importer drift, and cosmetic YAML churn.
- Explicit expected package roots for lockfile-only updates.
- Text, Markdown, and JSON reports.
- Docker GitHub Action and cross-platform release binaries.

[Unreleased]: https://github.com/wangxpych/lockfile-review/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/wangxpych/lockfile-review/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/wangxpych/lockfile-review/releases/tag/v0.1.0
