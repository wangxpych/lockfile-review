# Contributing

Focused bug reports and pull requests are welcome.

## Before opening a change

1. Search existing issues and pull requests.
2. Add the smallest fixture that proves the lockfile behavior.
3. Keep parsing, classification, and rendering changes in their owning layer.
4. Update review rules or architecture documentation when public behavior changes.
5. Run `make verify` and `docker build -t lockfile-review:local .`.

Please do not submit generated fixture data from a private repository. Reduce a
case to fictional package names and the minimum graph that preserves the issue.

By participating, you agree to follow the [Code of Conduct](./CODE_OF_CONDUCT.md).
