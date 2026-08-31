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

## File picker (`o`)

`o` opens a Midnight-Commander-style filesystem browser in the main panel (the
logo and info boxes stay put). Running `susfile` with **no file argument** opens
straight into it, rooted at the working directory — nothing is analysed until
you pick a file, and with nothing loaded `Esc` / `q` quit outright. It is a
single panel:

```
 /home/user/projects                                            42 items
   ▸ ..                                                    UP-DIR
   ▸ src/                                                     DIR  2026-08-30 22:14
   → link                  → ../elsewhere                    1.2 KiB  2026-08-11 08:56
     report.pdf                                            184.0 KiB  2026-08-29 17:02
   ≡ /home/user/projects/report.pdf  ·  -rw-r--r--  ·  184.0 KiB (188416 B)  ·  …
```

`..` sorts first, then directories, then files (case-insensitively). Symlinks
show `→ target` and are resolved to their target's type. The line under the list
is the selected entry's full path, mode, size and mtime. `Enter` on a **regular
file** closes the picker and re-runs the whole analysis on it — map, hex buffer,
info box and verdict all refresh. Non-regular files (devices, FIFOs) are refused
unless susfile was started with `--allow-special`, exactly as on the CLI. The
picker only lists directories and stats entries; it never writes.

| Key | Action |
|---|---|
| `o` | open the picker (from any view) |
| `↑ ↓` / `PgUp` `PgDn` / `g` `G` | move / page / top-end |
| `→` / `Enter` | enter the highlighted directory |
| `←` / `Backspace` | go to the parent directory |
| `Enter` (on a file) | analyse it |
| `.` | show / hide dotfiles |
| `~` / `/` | jump to `$HOME` / the filesystem root |
| `Esc` / `q` | close the picker |

## Keys

| Key | Action |
|---|---|
| `Tab` / `Shift-Tab` | next / previous view |
| `↑ ↓ ← →` | move the map inspector (Map view) |
| `Enter` | Map → Hex at the inspected offset |
| `↑ ↓` / `PgUp` `PgDn` / `g` `G` | scroll (Hex, Strings) |
| `o` | open the file picker |
| `r` | re-run the analysis |
| `q` / `Esc` / `Ctrl-C` | quit |

While the file picker is open, `q` / `Esc` close it instead of quitting;
`Ctrl-C` still quits.

## Colour

Colour follows the terminal (and `NO_COLOR` / `--no-color`). With colour, a
block's glyph is tinted on the blue→red entropy ramp. Without, the class letter
still carries the meaning and low-entropy padding shades with `░ ▒ ▓ █`.
