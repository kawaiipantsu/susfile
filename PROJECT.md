# susfile — Project Specification

> The authoritative implementation contract for **susfile**, a CLI file-forensics
> visualiser. When this document and the code disagree, this document wins.
> Changing it is a deliberate act.
>
> **Executable:** `susfile` · **Language:** Go 1.27+ · **Repo model:** Git Flow.

---

## 1. Purpose

`susfile <path>` reads exactly one file and answers *"what is this?"* visually:

- the **file map** — the whole file as a grid of classified block cells (the
  centrepiece);
- the **facts** — magic bytes, MIME, detected type, size, timestamps, mode,
  MD5/SHA-1/SHA-256, section and string counts, global Shannon entropy;
- **drill-downs** — windowed entropy chart, byte histogram, hex dump, strings.

susfile is not a disassembler, not a hex editor, not a malware scanner, and not a
file manager. It reads; it never writes; it never uses the network.

## 2. Platforms

Release targets (all `CGO_ENABLED=0`, static):

| GOOS/GOARCH | GOARM | `.deb` Architecture |
|---|---|---|
| `linux/amd64` | — | `amd64` |
| `linux/386` | — | `i386` |
| `linux/arm64` | — | `arm64` |
| `linux/arm` | 7 | `armhf` |

macOS and Windows are not release targets. The code must nonetheless avoid
OS-specific assumptions outside clearly marked files.

## 3. Technology

- **Language:** Go, standard library first.
- **TUI:** `charmbracelet/bubbletea` + `lipgloss` + `bubbles`.
- **MIME:** `gabriel-vasile/mimetype` (pure Go), cross-checked against susfile's
  own magic-signature table.
- **Executable parsing:** stdlib `debug/elf`, `debug/pe`, `debug/macho`.
- **Everything else** — hashing, entropy, statistics, classification, string
  extraction — is standard library and local code.
- **Permitted third-party dependencies:** the four named above. Adding a fifth,
  or anything requiring cgo, is a spec change.

## 4. Repository layout

```
cmd/susfile/main.go          flag parsing; TUI vs --no-tui vs --json; stdin ("-")
internal/version/            build metadata (ldflags)
internal/analyze/            ALL measurement; returns one Result; no writes, no network
  read.go entropy.go hashes.go histogram.go stats.go
  microblocks.go classify.go magic.go mime.go binfmt.go strings.go verdict.go
internal/report/             plain.go, json.go — render Result, no measurement
internal/tui/                Bubble Tea app over Result
  app.go layout.go logo.go fileinfo.go
  filemap.go                 CENTREPIECE
  entropy.go histogram.go hexview.go stringsview.go footer.go theme.go keys.go
packaging/debian/            control.in, copyright, changelog.in, susfile.1
scripts/                     package.sh, package-deb.sh, release-check.sh
docs/                        architecture.md, analysis.md, tui.md, packaging.md, releases.md
```

## 5. Analysis engine (`internal/analyze`)

### 5.1 Result

`Analyze(ctx, path string, progress func(Stage, float64)) (*Result, error)`
returns a `Result` with: file metadata; magic match + leading-bytes hex; MIME +
extension guess; hashes; `Histogram [256]uint64` + printable/null/high ratios;
`Stats` (chi-square, arithmetic mean, Monte-Carlo π estimate, serial
correlation); `GlobalEntropy float64`; `MicroBlocks []MicroBlock` (up to
**4096**, fewer for small files so each block spans a sensible number of bytes);
`BinFmt *BinFmt` (nil if not an executable); `Strings []StringHit` (capped);
`Verdict` (text + kaomoji).

A Go `error` is returned only when the file cannot be read at all (not found,
permission denied, is a directory, non-regular without `--allow-special`).
Everything else — truncation, unparseable executable, unknown magic — is a
populated `Result` with `Verdict.Kaomoji == "¯\\_(ツ)_/¯"`.

### 5.2 Reading limits

- Hashing and the global-entropy pass **stream** in fixed chunks; file size does
  not bound memory.
- The full-buffer passes (histogram, micro-block scan, strings) read at most
  `--max-bytes` (default **256 MiB**). Above that, micro-blocks are sampled
  (seek + read a window per block) rather than the file being materialised.
- `MicroBlocks` holds `clamp(size/16, 1, 4096)` entries. Each block's entropy
  and byte ratios are measured over a window of at least 4096 bytes centred on
  the block (clamped to the file), so the figures are meaningful even when a
  block itself is only tens of bytes wide. See `docs/analysis.md`.

### 5.3 MicroBlock and Class

`MicroBlock{ Offset, Len int64; Entropy float64; Printable, NUL, High, WS
float32; Distinct int; TopByte byte; Section string; Class Class }`.

`Class` is one of: `·` null/pad · `P` text · `F` source/functions · `S`
string-table · `C` code · `M` media · `Z` compressed · `E` encrypted · `H`
header · `D` data · `R` repetitive. `classify.go` decides from entropy, the byte
ratios, distinct-byte count, code-token density (for printable blocks), and the
`Section` hint when `BinFmt` is present. The exact thresholds live in
`docs/analysis.md` and are the spec for the classifier.

### 5.4 Verdict

Whole-file summary derived from `GlobalEntropy`, the histogram flatness, the
class histogram of the micro-blocks and `BinFmt`:

| Kaomoji | Meaning |
|---|---|
| `^_^` | clean text / source |
| `•_•` | ordinary binary / executable |
| `ಠ_ಠ` | high entropy — likely packed, compressed or encrypted |
| `¯\_(ツ)_/¯` | truncated, malformed, or could not be fully parsed |
| `-_-` | empty file |

## 6. Reporters (`internal/report`)

- **plain.go** — label/value info block, a downsampled ASCII class-map (64×16 by
  default, `--map-size` overrides), the legend, and a one-line entropy sparkline.
  Honours `--no-color`.
- **json.go** — a documented, stable schema: all `Result` fields, the full
  `microblocks` array, a rendered `map` string, and `legend`. Additive changes
  only within a major version.

## 7. TUI (`internal/tui`)

Three stacked regions:

1. **Top half** — a fixed ~14-column ASCII logo box on the left; a flexible
   label/value info box on the right (`Type`, `Magic`, `MIME`, `Entropy` +
   inline bar, `Strings`, `Sections`, `SHA-256`, `Verdict` + kaomoji).
2. **Main** (fills the rest) — the active view. `Tab` cycles **Map → Entropy →
   Histogram → Hex → Strings**; Map is default. Map draws one cell per character
   from the 4096 micro-blocks downsampled to the panel grid; glyph = `Class`,
   colour = entropy ramp (blue→cyan→green→yellow→red), or `░▒▓█` shade under
   `--no-color`. A cursor (`↑↓←→`) shows the hovered block's offset range,
   entropy, class and top bytes; `enter` switches to Hex at that offset. A
   two-row legend sits under the grid.
3. **Footer** — row 1: context key hints. Row 2: a progress bar + spinner +
   stage text while analysing, and — always, every frame, right-aligned — the
   stamp `⟦THUGS⟧` in bright-red bold followed by `(c) 2026` dimmed.

Analysis runs in a goroutine; the model consumes `progress`, `done` and `err`
messages. Below **80×24** the TUI renders only a centred resize prompt. On
`WindowSizeMsg` the map re-aggregates from the cached micro-blocks — the file is
not re-read.

### 7.1 File picker

`o` opens a filesystem browser in the **Main** region (the logo and info boxes
and the footer stay). It is a single Midnight-Commander-style panel: `..` first,
then directories, then files (case-insensitive); `name / size / mtime` columns; a
highlighted cursor row; an info line for the selected entry (full path, mode,
size, mtime); a dotfile toggle (`.`). `↑↓ PgUp PgDn g G` move, `→`/`⏎` enters a
directory, `←`/`Backspace` goes up, `~` `$HOME`, `/` the root. `⏎` on a **regular
file** re-runs `analyze.Analyze` on it and every derived field (map, hex buffer,
info box, verdict) refreshes; non-regular files need `--allow-special`, as on the
CLI. `Esc`/`q` close the picker; `Ctrl-C` quits. The picker only lists
directories and stats entries — it performs no file operations, so susfile stays
"not a file manager" (§10).

## 8. CLI

```
susfile [flags] <file>
susfile version | --version | -V
susfile help | --help | -h
```

| Flag | Default | Effect |
|---|---|---|
| `--no-tui` | false | plain text report |
| `--json` | false | JSON report on stdout |
| `--strings-min N` | 4 | minimum extracted-string length |
| `--max-bytes N` | 268435456 | full-buffer read cap |
| `--map-size WxH` | 64x16 | plain-mode class-map grid |
| `--allow-special` | false | permit non-regular files |
| `--no-color` | auto | disable colour (also honours `NO_COLOR`) |

`<file>` of `-` reads stdin into a bounded buffer. TUI is the default when stdout
is a TTY and a file is given; otherwise plain mode.

## 9. Build & release

- `Makefile` is the interface. Required targets: `help deps fmt fmt-check vet
  lint test race bench coverage run build build-linux build-all install clean
  generate security dist deb snapshot release-check`.
- `LDFLAGS` stamps `internal/version`. `VERSION` comes from the first
  `## [x.y.z]` heading in `CHANGELOG.md`, else `0.1.0-dev`. `SOURCE_DATE_EPOCH`
  is honoured for reproducible builds.
- `make dist` → `dist/susfile_<ver>_linux_<arch>.tar.gz` ×4 + `SHA256SUMS`.
- `make deb` → `dist/susfile_<ver>_<debarch>.deb` ×4 via `dpkg-deb
  --root-owner-group`, containing `/usr/bin/susfile`,
  `/usr/share/man/man1/susfile.1.gz`,
  `/usr/share/doc/susfile/{copyright,changelog.Debian.gz}`. No `Depends`.
- Semantic versioning, `v`-prefixed annotated tags on `main` only. Release
  procedure is in `RELEASE.md`.

## 10. Non-goals

Disassembly · editing / patching files · signature-based malware detection ·
recursive archive extraction · any network feature (hash lookups, reputation,
update checks, telemetry) · a config file or persistent state · Windows/macOS
release artifacts · a plugin marketplace · file operations in the TUI picker —
it only navigates the filesystem to choose the single input (§7.1).
