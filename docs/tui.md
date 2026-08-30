# The TUI

Run `susfile <file>` with a terminal attached and no `--no-tui` / `--json` and
you get the interactive UI. It renders an `analyze.Result` — it measures
nothing itself.

## Regions

```
┌ logo ┐┌ file info (label / value) ───────────────┐
│ mark ││ Type · Magic · MIME · Size · Entropy      │   top
└──────┘│ Strings/Sections · SHA-256 · Verdict      │
        └──────────────────────────────────────────┘
 Map · Entropy · Histogram · Hex · Strings              tab bar
┌──────────────────────────────────────────────────┐
│ the active view fills the lower half             │   main
│ … inspector line …                              │
│ … legend (2 rows) …                             │
└──────────────────────────────────────────────────┘
 [keys …]                                              footer
 ▓▓▓░ scanning blocks…                 ⟦THUGS⟧ (c) 2026
```

Below **80×24** the whole screen is a single "resize" prompt.

Analysis runs in a goroutine; the footer shows a spinner, the current stage and
a progress bar until it finishes, then the verdict. The `⟦THUGS⟧ (c) 2026` stamp
is drawn on every frame, right-aligned.

## Views (`Tab` / `Shift-Tab` to cycle)

| View | What it shows |
|---|---|
| **Map** | the centrepiece — the file as a grid of class glyphs, coloured by entropy. A cursor (`↑ ↓ ← →`) reports the hovered block's offset range, entropy, class and top byte; `Enter` jumps to that offset in Hex. |
| **Entropy** | windowed entropy as a column chart, 0–8 on the y-axis, file offset on the x-axis. |
| **Histogram** | the 256-bucket byte-frequency distribution (log scale). |
| **Hex** | scrollable hex dump of the first 4 MiB, bytes tinted by class. `↑↓` line, `PgUp/PgDn` page, `g`/`G` top/end. |
| **Strings** | the extracted strings with offsets and encoding (`a` = ASCII, `u` = UTF-16LE). Same scroll keys. |

## Keys

| Key | Action |
|---|---|
| `Tab` / `Shift-Tab` | next / previous view |
| `↑ ↓ ← →` | move the map inspector (Map view) |
| `Enter` | Map → Hex at the inspected offset |
| `↑ ↓` / `PgUp` `PgDn` / `g` `G` | scroll (Hex, Strings) |
| `r` | re-run the analysis |
| `q` / `Esc` / `Ctrl-C` | quit |

## Colour

Colour follows the terminal (and `NO_COLOR` / `--no-color`). With colour, a
block's glyph is tinted on the blue→red entropy ramp. Without, the class letter
still carries the meaning and low-entropy padding shades with `░ ▒ ▓ █`.
