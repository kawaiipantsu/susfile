# Architecture

susfile is a measurement core with thin renderers over it.

```
cmd/susfile  ──▶  internal/analyze  ──▶  analyze.Result
   (flags)             (all measurement)        │
                                     ┌──────────┼───────────┐
                                     ▼          ▼           ▼
                          internal/report   internal/tui   (future)
                          (plain / JSON)   (Bubble Tea)
```

## `internal/analyze`

The only package that measures anything. It:

- **never writes** — the analysed file is opened read-only and nothing else is
  touched;
- **never uses the network** — there is no `net/http` import and CI keeps it
  that way;
- **imports nothing** from `internal/report` or `internal/tui`.

`Analyze(ctx, path, opt, progress)` returns one `*Result`. A Go `error` means
"could not read the file at all"; every recoverable condition is a populated
`Result` (see [analysis.md](analysis.md)).

Files: `read.go` (bounded input, regular-file check, stdin), `hashes.go`
(streaming multi-hash), `entropy.go`, `histogram.go`, `stats.go`,
`microblocks.go` (+ `Downsample`), `classify.go`, `magic.go`, `mime.go`,
`binfmt.go` (stdlib `debug/elf|pe|macho`, every call wrapped against panics),
`strings.go`, `verdict.go`.

## `internal/report`

Renders a `Result`. No measurement.

- `Plain` — label/value block, a downsampled ASCII class map, the legend, an
  entropy sparkline. Honours `--no-color` / `NO_COLOR`.
- `JSON` — `schema` + `tool` + the whole `Result` + a rendered `map` + the
  `legend`. Additive changes only within `susfile.report/v1`.

## `internal/tui`

*(lands in `feature/tui`)* A Bubble Tea app over `Result`: logo box + info box on
top, the file map filling the lower half, a footer with progress and the
`⟦THUGS⟧ (c) 2026` stamp. It downsamples the micro-blocks to the panel grid on
every resize — it does not re-read the file.

## `internal/version`

Build metadata (`Version`, `Commit`, `Date`, `Dirty`) stamped by `-ldflags`.

## Invariants

1. `internal/analyze` imports nothing from `internal/tui` or `internal/report`.
2. Analysis is read-only and offline.
3. Local development works without GitHub; CI is additive.
4. `main` is releases only; work flows through `develop`; releases are
   annotated tags.
5. Pure Go, `CGO_ENABLED=0`; the four Linux targets are a hard requirement.
