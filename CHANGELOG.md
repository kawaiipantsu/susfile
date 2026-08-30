# Changelog

All notable changes to susfile are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Repository foundation: MIT licence, Git Flow layout, community health files,
  `PROJECT.md` specification, `CLAUDE.md` engineering guide.
- Build system: `Makefile` with the development loop and a Linux-only
  cross-compile matrix — `linux/amd64`, `linux/386`, `linux/arm64`,
  `linux/arm` (v7) — all `CGO_ENABLED=0`, `-trimpath`, version-stamped.
- CI: `test` (fmt-check, vet, test, race), `Cross-build`, `Lint`
  (golangci-lint) and `Vulnerability scan` (govulncheck) on every push and PR;
  a **Branch flow** check that fails a PR opened against the wrong base.
- `susfile version` entry point with build metadata.
- Packaging: `make dist` produces a `.tar.gz` per Linux arch plus `SHA256SUMS`;
  `make deb` produces four `.deb` packages (`amd64`, `i386`, `arm64`, `armhf`)
  via `dpkg-deb` with a man page, DEP-5 copyright and a Debian changelog. No
  `Depends` — the binary is static.
- `make release-check` verifies a clean tree, a matching changelog heading, a
  free tag and cross-compilation to all four targets.
- `Release` workflow: a `v*` tag builds every archive and package and publishes
  a GitHub release with `SHA256SUMS` attached.
- `install.sh`: arch-detecting installer that verifies the download against
  `SHA256SUMS` and never calls `sudo`.
- `susfile.1` man page; `docs/packaging.md`.
- Analysis engine (`internal/analyze`): streaming MD5/SHA-1/SHA-256/CRC-32,
  whole-file Shannon entropy, 256-bin histogram with byte-class fractions,
  ent-style statistics (mean, chi-square, Monte-Carlo π, serial correlation),
  a curated magic-signature table plus pure-Go MIME detection, ELF/PE/Mach-O
  structure with per-section entropy (every parser wrapped against panics),
  ASCII + UTF-16LE string extraction, and a windowed micro-block scan
  (`clamp(size/16, 1, 4096)` blocks) with an 11-class content classifier and a
  kaomoji verdict. Nothing is written and no network call is made.
- Reporters (`internal/report`): `susfile --no-tui` plain text with an ASCII
  class map, legend and entropy sparkline; `susfile --json` with the stable
  `susfile.report/v1` schema (full result + rendered map + legend).
- `susfile <file>` CLI: `--no-tui`, `--json`, `--strings-min`, `--max-bytes`,
  `--map-size`, `--allow-special`, `--no-color`, and `-` for stdin.
- `docs/analysis.md` (the classifier spec) and `docs/architecture.md`.
- Interactive TUI (`internal/tui`, Bubble Tea): a logo box and a label/value
  info box on top, the classified **file map** filling the lower half with a
  movable inspector cursor, secondary Tab views (windowed entropy chart, byte
  histogram, scrollable hex dump, extracted-strings list), and a footer with a
  live progress bar and the `⟦THUGS⟧ (c) 2026` stamp on every frame. Colour
  follows the terminal and `NO_COLOR` / `--no-color`; below 80×24 it shows a
  resize prompt. `susfile <file>` launches it by default when a terminal is
  attached. `docs/tui.md`.
