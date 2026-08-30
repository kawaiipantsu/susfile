# How susfile analyses a file

Everything here is computed locally in `internal/analyze`. Nothing is fetched,
nothing is written.

## Passes

| Pass | Reads | Produces |
|---|---|---|
| **stream** | the whole file, in 1 MiB chunks | MD5, SHA-1, SHA-256, CRC-32, global Shannon entropy |
| **buffer** | a prefix, capped at `--max-bytes` (default 256 MiB) | the working buffer for every pass below |
| **histogram** | the buffer | 256-bin byte histogram, printable / NUL / high-bit / whitespace fractions, distinct-byte count |
| **stats** | the buffer + histogram | arithmetic mean, chi-square (+ approx. p), Monte-Carlo π, lag-1 serial correlation |
| **structure** | the buffer head | magic-signature match, MIME (via `gabriel-vasile/mimetype`), extension guess, ELF/PE/Mach-O summary with per-section entropy |
| **blocks** | the buffer, or seeked samples | the micro-block scan (below) |
| **strings** | the buffer | ASCII and UTF-16LE runs ≥ `--strings-min` with offsets |
| **verdict** | everything above | one kaomoji + a one-line summary |

Only a Go `error` for "could not read the file at all" stops this. A truncated
file, an unparseable executable, an unknown magic — each leaves a populated
`Result` with the fields that could be filled.

## Global entropy

Shannon entropy in bits per byte over the **entire** file (the stream pass), not
just the buffered prefix:

```
H = −Σ p(b)·log2 p(b)     for each byte value b that occurs
```

Range `[0, 8]`. `0` = one byte value only; `8` = every value equally likely.

## Statistics (ent-style)

- **Mean** — average byte value. `127.5` for uniform random.
- **Chi-square** — `Σ (observed − expected)² / expected` over the 256 values,
  `expected = n/256`. Near `255` for random; large for structured data. The
  reported probability is a Wilson–Hilferty normal approximation with df = 255
  (good to a few percent at this df; it is a guide, not a test).
- **Monte-Carlo π** — successive 6-byte groups are `(x, y)` points (3 bytes
  each, big-endian) in the unit square; π ≈ 4·inside/total. The fractional error
  shrinks toward 0 as data approaches uniform random.
- **Serial correlation** — lag-1 autocorrelation of the byte values, wrapping at
  the end as `ent` does. Near `0` for independent bytes, near `±1` for a slowly
  varying stream.

## The micro-block scan

The file map is built from a set of **micro-blocks**:

- Count: `clamp(fileSize / 16, 1, 4096)`. Small files get fewer, wider blocks;
  files ≥ 64 KiB get the full 4096.
- Each block records its byte range, and — measured over a **window of at least
  4096 bytes** centred on the block and clamped to the file — its entropy,
  printable / NUL / high-bit / whitespace fractions, distinct-byte count, most
  common byte, and (for executables) the section it falls in.
- The windowing is what makes the entropy figure meaningful even when a block
  itself is only tens of bytes wide: entropy of 40 random bytes is ~5.3, not 8.
- For a file larger than `--max-bytes`, blocks are produced by seeking and
  reading an 8 KiB sample at each block position rather than from the buffer.

Renderers (`internal/report`, `internal/tui`) call `analyze.Downsample` to
aggregate the micro-blocks to whatever grid they draw — one glyph per cell, the
cell taking the most common class of the blocks it covers (ties broken toward
the higher-signal class), the mean entropy for colour.

## Block classification

`classify.go` assigns one `Class` per block. Checks run in this order; the first
match wins. `ratio = entropy / log2(min(windowLen, 256))` normalises entropy to
what the sample size could reach.

| # | Condition | Class | Glyph |
|---|---|---|---|
| 1 | window is empty (block past EOF) | `empty` | ` ` |
| 2 | NUL fraction ≥ 0.92 | `null/pad` | `·` |
| 3 | ≤ 6 distinct byte values and entropy < 1.6 | `repetitive` | `R` |
| 4 | printable ≥ 0.85 and NUL 0.04–0.5 and mean printable-run < 24 | `strings` | `S` |
| 5 | printable ≥ 0.85 and looks like source (punctuation density, line breaks, letter ratio) | `source` | `F` |
| 6 | printable ≥ 0.85 | `text` | `P` |
| 7 | printable ≥ 0.45 and NUL ≥ 0.08 and mean run < 24 | `strings` | `S` |
| 8 | section is `.text`-like and `ratio` in 0.62–0.995 | `code` | `C` |
| 9 | section is `.rodata`-like and printable ≥ 0.3 | `strings` | `S` |
| 10 | `ratio` ≥ 0.9985 and ≥ 254 distinct values | `encrypted` | `E` |
| 11 | `ratio` ≥ 0.965 | `compressed` | `Z` |
| 12 | `ratio` ≥ 0.90 | `media` | `M` |
| 13 | `ratio` ≥ 0.55 and not the first block | `code` | `C` |
| 14 | first block of the file | `header` | `H` |
| 15 | otherwise | `data` | `D` |

**`E` vs `Z`.** Distinguishing encryption from good compression by entropy alone
is unreliable below tens of kilobytes of sample. `E` therefore only triggers
when the window is large enough for `ratio` to get very close to 1 with a full
byte set — in practice, large files. Smaller high-entropy regions read as `Z`,
and the **verdict** carries the "packed / compressed / encrypted" nuance at the
whole-file level.

These thresholds are deliberately conservative and will be tuned; see the
`type:idea` issues and the Ideas discussion.

## Verdict

| Kaomoji | When |
|---|---|
| `-_-` | file is empty |
| `¯\_(ツ)_/¯` | magic says ELF/PE/Mach-O but no parser accepted the body, or a parse warning was recorded |
| `ಠ_ಠ` | global entropy ≥ 7.90 with a full byte set and no compression/media magic, and most blocks are `Z`/`E`; or an executable whose largest section is ≥ 7.8 entropy and ≥ half the file |
| `^_^` | ≥ 60% of non-padding blocks are `source`/`text` and printable ≥ 0.85 |
| `•_•` | anything else — an ordinary binary, executable, or a legitimate compressed/media container |
