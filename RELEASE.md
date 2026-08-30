# Publishing a susfile release

## 1. Choosing a tag

Semantic versioning, `v` prefix, annotated tags only, on `main` only.

| Situation | Tag |
|:--|:--|
| Testing before a real release | `v0.1.0-rc.1` (bump the `rc` each attempt) |
| First usable release | `v0.1.0` |
| Bug fixes only | `v0.1.1` |
| New features, pre-1.0 (may include breaking changes) | `v0.2.0` |
| First stable API / JSON-schema commitment | `v1.0.0` |

Release candidates are marked pre-release on GitHub.

## 2. Cutting the release

```bash
git switch develop
git switch -c release/0.1.0

# Move [Unreleased] entries into a new "## [0.1.0] - <date>" section in
# CHANGELOG.md. The Makefile reads VERSION from the first "## [x.y.z]" heading.

make release-check           # fmt-check, vet, lint, test, build-linux, clean tree, tag free
make dist                    # four tar.gz + SHA256SUMS
make deb                     # four .deb

git commit -am "chore: prepare v0.1.0"
git switch main   && git merge --no-ff release/0.1.0
git tag -a v0.1.0 -m "susfile v0.1.0"
git switch develop && git merge --no-ff release/0.1.0
git branch -d release/0.1.0

git push origin main develop
git push origin v0.1.0
```

Merges to `main` go through a PR from the `release/*` branch so the **Branch
flow** check and CI run; the commands above are the shape, not a bypass.

## 3. GitHub release

```bash
gh release create v0.1.0 \
  --title "susfile v0.1.0" \
  --notes-file dist/notes.md \
  dist/*.tar.gz dist/*.deb dist/SHA256SUMS
```

Add `--prerelease` for an `-rc`. **Always upload `SHA256SUMS`.**

### What to write

Write for someone deciding whether to upgrade.

```markdown
One or two sentences on what this release is for.

## Highlights
- The three or four things a reader actually cares about
- Lead with anything affecting safety or output format

## Install
Download a binary or `.deb` below and verify it:

    sha256sum -c SHA256SUMS --ignore-missing

    sudo dpkg -i susfile_0.1.0_amd64.deb

## Known limitations
- Be specific and honest. Copy this section from CHANGELOG.md verbatim.

## Full changelog
https://github.com/kawaiipantsu/susfile/compare/v0.0.0...v0.1.0
```

No marketing language. "Faster" needs a number.

## 4. Artifacts

| File | Platform |
|:--|:--|
| `susfile_<ver>_linux_amd64.tar.gz` | Linux x86-64 |
| `susfile_<ver>_linux_386.tar.gz` | Linux x86 (32-bit) |
| `susfile_<ver>_linux_arm64.tar.gz` | Linux ARM64 |
| `susfile_<ver>_linux_arm.tar.gz` | Linux ARMv7 |
| `susfile_<ver>_amd64.deb` / `_i386.deb` / `_arm64.deb` / `_armhf.deb` | Debian/Ubuntu |
| `SHA256SUMS` | checksums for all of the above |

All binaries are static (`CGO_ENABLED=0`, `-trimpath`). `.deb` packages are
currently unsigned; the checksums are the integrity check.

## 5. After publishing

Add a fresh `## [Unreleased]` section to `CHANGELOG.md` on `develop`.
