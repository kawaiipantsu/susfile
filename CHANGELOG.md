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
