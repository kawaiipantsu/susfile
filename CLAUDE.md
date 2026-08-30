# CLAUDE.md

Guidance for Claude Code (claude.ai/code) and any coding agent working in this
repository.

## What susfile is

A single static Go binary (`susfile`) that reads **one file** and visualises what
it is: magic header, MIME/type, size, timestamps, hashes, ELF/PE/Mach-O
structure, Shannon entropy and ent-style statistics — with a **defrag-screen
"file map"** of classified byte regions as the centrepiece of the TUI.

`PROJECT.md` is the authoritative specification. When the spec and the code
disagree, the spec wins; changing it is a deliberate act, not a side effect of
implementation.

**Repo:** `github.com/kawaiipantsu/susfile` · **Go 1.27+** · `CGO_ENABLED=0` everywhere.

## Commands

```bash
make help                    # every target
make fmt vet test build      # the pre-commit loop — run all four
make race coverage lint
make build-linux             # four Linux targets: amd64, 386, arm64, arm(v7)
make deb                     # four .deb packages (feature/build-system)
make security                # govulncheck
```

Narrower runs: `go test ./internal/analyze/ -run TestClassify -v`.

## Architecture

A UI-independent analysis core with thin renderers over it.

```
cmd/susfile  →  internal/analyze  →  analyze.Result
                                        │
                     ┌──────────────────┼──────────────────┐
                     ▼                  ▼                  ▼
              internal/report     internal/tui        (future frontends)
              (plain / JSON)   (Bubble Tea file map)
```

- **`internal/analyze`** owns all measurement: `read.go` (bounded/streaming
  reader), `entropy.go`, `hashes.go`, `histogram.go`, `stats.go`,
  `microblocks.go` (fixed 4096-block scan), `classify.go` (block → `Class`
  glyph), `magic.go` + `mime.go`, `binfmt.go` (stdlib `debug/elf|pe|macho`),
  `strings.go`, `verdict.go`. It returns one `Result` and **never writes, never
  opens a socket**.
- **`internal/report`** renders `Result` as text or JSON. No measurement here.
- **`internal/tui`** is a Bubble Tea app over `Result`. `filemap.go` is the
  centrepiece; `entropy.go`, `histogram.go`, `hexview.go`, `stringsview.go` are
  secondary Tab views. The TUI downsamples the 4096 micro-blocks to the current
  panel size on every resize — it does not re-read the file.
- **`internal/version`** is build metadata stamped by `-ldflags`.

### Things that will bite you

- **The micro-block count is fixed (4096), the grid is not.** `analyze` scans a
  resolution-independent set of blocks once; the TUI and the plain reporter both
  aggregate that set down to whatever grid they are drawing. Never make
  `analyze` depend on terminal size.
- **A recoverable problem is a populated `Result`, not an `error`.** A truncated
  file, an unparseable PE, an unknown magic — these produce a `Result` with the
  fields that could be filled and a verdict of `¯\_(ツ)_/¯`. A Go `error` is for
  "could not read the file at all".
- **Huge files must stream.** Hashing and the entropy pass read in chunks;
  full-buffer work is capped (`--max-bytes`, default in `PROJECT.md`). Above the
  cap, sample — do not allocate the whole file.
- **Non-regular files are refused** unless `--allow-special`. Do not follow a
  path into `/proc` or a device by accident.
- **Zero network. Ever.** There is no HTTP client in this module and there must
  never be one. No hash-reputation lookups, no update checks, no telemetry.

## Invariants

Violating these is a spec change, not an implementation detail:

1. `internal/analyze` imports nothing from `internal/tui` or `internal/report`.
2. Analysis is read-only and offline: no writes to the analysed file, no network.
3. Local development works without GitHub; CI is additive.
4. `main` is releases only; work flows through `develop`; releases are annotated tags.
5. Pure Go, `CGO_ENABLED=0`; the four Linux targets are a hard requirement.

## Git workflow

Git Flow. `feature/<name>` branches from `develop` and merges back via PR;
`release/<version>` merges into `main` **and** `develop` with an annotated tag on
`main`; `hotfix/<...>` branches from `main`. Conventional-commit prefixes
(`feat: fix: build: docs: test: chore: refactor:`). Merges use `--no-ff`. Never
commit feature work directly to `main` — the **Branch flow** check rejects it.

Remote is `git@github.com:kawaiipantsu/susfile.git`.

## Constraints

- **Dependency minimalism.** Standard library first. The only third-party deps
  are the Charm TUI stack (`bubbletea`, `lipgloss`, `bubbles`) and
  `gabriel-vasile/mimetype`. Anything that pulls in cgo breaks the release
  matrix and is a spec change.
- **No committed binary fixtures.** Tests build their inputs from `[]byte`
  literals or generate them in-test.
- **Cross-compilation is verified**, not assumed:
  `CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build ./...`.

## Task discipline

Do not claim it works without running `make test`. Do not invent format
behaviour — cite the spec or the stdlib. Avoid unrelated refactors. A change is
done when: implemented, tested, tests pass, cross-build passes, user-facing
behaviour documented in `docs/` and `CHANGELOG.md` under `[Unreleased]`, errors
handled, formatted.

## Current state

Done: repository foundation, Git Flow, CI, the cross-compile Makefile, and the
`susfile version` entry point. In progress: `.deb` packaging (feature/build-
system). Not started: the analysis engine, the reporters and the TUI.
