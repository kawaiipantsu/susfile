# Packaging

susfile ships as static Go binaries for four Linux targets, packaged two ways:
per-arch `.tar.gz` archives and `.deb` packages. No cgo, no runtime dependencies.

## Targets

| GOOS/GOARCH | GOARM | `.deb` arch | `uname -m` |
|---|---|---|---|
| `linux/amd64` | — | `amd64` | `x86_64` |
| `linux/386` | — | `i386` | `i686` |
| `linux/arm64` | — | `arm64` | `aarch64` |
| `linux/arm` | 7 | `armhf` | `armv7l` |

## Building

```bash
make build-linux     # dist/susfile_<ver>_linux_<arch>/susfile  (x4)
make man             # dist/susfile.1.gz
make dist            # dist/susfile_<ver>_linux_<arch>.tar.gz (x4) + SHA256SUMS
make deb             # dist/susfile_<ver>_<debarch>.deb (x4), folded into SHA256SUMS
```

`VERSION` is taken from the first `## [x.y.z]` heading in `CHANGELOG.md`
(fallback `0.1.0-dev`). Override with `make dist VERSION=1.2.3`.
`SOURCE_DATE_EPOCH` is honoured for reproducible builds.

## `.deb` layout

`scripts/package-deb.sh` builds each package with `dpkg-deb --root-owner-group`
from a staging tree — no `fpm`, no Ruby:

```
/usr/bin/susfile
/usr/share/man/man1/susfile.1.gz
/usr/share/doc/susfile/copyright              (DEP-5)
/usr/share/doc/susfile/changelog.Debian.gz
DEBIAN/control                                (from packaging/debian/control.in)
```

`control.in` declares `Section: utils`, `Priority: optional`, no `Depends`
(the binary is static). Inspect a built package:

```bash
dpkg-deb -I dist/susfile_0.1.0_amd64.deb     # control metadata
dpkg-deb -c dist/susfile_0.1.0_amd64.deb     # file list
sudo dpkg -i dist/susfile_0.1.0_amd64.deb
```

## Signing

`.deb` packages are currently **unsigned**. Integrity is via `SHA256SUMS`
published with each GitHub release. Repository signing (`dpkg-sig` / a hosted
apt repo) is tracked as future work.

## CI

`.github/workflows/release.yml` runs on a `v*` tag: it checks the tag matches the
changelog, runs `make dist` and `make deb`, and publishes a GitHub release with
every archive, every `.deb`, and `SHA256SUMS` attached.
