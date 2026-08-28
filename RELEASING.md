# Releasing

Releases are distributed as GitHub Action tags and prebuilt CLI archives. No
package registry publication is required.

## Procedure

1. Update `CHANGELOG.md` and any affected behavior documentation.
2. Run `make verify` with the minimum supported Go version and the current Go version.
3. Build and run the Docker Action against clean and failing fixtures.
4. Confirm the version tag does not already exist.
5. Merge through the protected `main` branch with all required checks passing.
6. Publish a GitHub Release tagged `v<version>`.
7. Verify the release workflow attaches archives and `checksums.txt`.
8. Download one public archive, verify its checksum, run `lockreview --version`,
   and review both a passing and failing fixture.

For each minor release line, maintain a floating major tag such as `v0` at the
same audited commit. Moving a floating tag must not create a second Release.

Never overwrite a versioned release tag or release asset. Publish a new patch
version if an artifact is incorrect.
