# Releases

The full procedure lives in [`../RELEASE.md`](../RELEASE.md). In short:

1. `release/<version>` off `develop`; move `CHANGELOG.md` `[Unreleased]` into a
   dated `## [x.y.z]` section with a **Known limitations** list.
2. `make release-check` — clean tree, changelog heading, free tag, all four
   Linux cross-builds.
3. `make dist && make deb` — four `.tar.gz`, four `.deb`, `SHA256SUMS`.
4. PR `release/<version> → main`, merge `--no-ff`, annotated tag `v<version>`.
5. PR `release/<version> → develop`, merge. Delete the branch.
6. `.github/workflows/release.yml` fires on the tag and publishes the GitHub
   release with every artifact attached. (Or run `gh release create` by hand as
   in `RELEASE.md`.)

Versioning: semver, `v`-prefixed annotated tags on `main` only. Pre-1.0, the
minor version carries breaking changes.

Artifacts per release:

| Pattern | Platform |
|---|---|
| `susfile_<v>_linux_amd64.tar.gz` | Linux x86-64 |
| `susfile_<v>_linux_386.tar.gz` | Linux x86 (32-bit) |
| `susfile_<v>_linux_arm64.tar.gz` | Linux ARM64 |
| `susfile_<v>_linux_arm.tar.gz` | Linux ARMv7 |
| `susfile_<v>_{amd64,i386,arm64,armhf}.deb` | Debian / Ubuntu |
| `SHA256SUMS` | checksums for all of the above |

`.deb` packages are currently unsigned; verify with `SHA256SUMS`.
