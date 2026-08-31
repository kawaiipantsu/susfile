package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// fileEntry is one row in the file picker: a directory, a regular file, or a
// symlink resolved to its target's type. The size/mode/modTime fields describe
// the target for a symlink; linkTarget keeps the raw link text for display.
type fileEntry struct {
	name       string
	isDir      bool
	isSymlink  bool
	broken     bool // symlink whose target could not be stat'd
	mode       fs.FileMode
	size       int64
	modTime    time.Time
	linkTarget string
}

// isParent reports whether this is the synthetic ".." row.
func (e fileEntry) isParent() bool { return e.name == ".." }

// analyzable reports whether Enter on this entry should hand it to analyze. It
// mirrors internal/analyze's openFile rules: regular files always, other
// non-directory types only with --allow-special.
func (e fileEntry) analyzable(allowSpecial bool) bool {
	if e.isDir || e.isParent() || e.broken {
		return false
	}
	if e.mode.IsRegular() {
		return true
	}
	return allowSpecial
}

// browser is the file-picker state: the directory being listed, its entries,
// the cursor and scroll offset, whether dotfiles are shown, and a transient
// status message (a permission error on a failed descent, say).
type browser struct {
	dir        string
	entries    []fileEntry
	cursor     int
	top        int
	showHidden bool
	err        error  // the current directory could not be read at all
	msg        string // transient note for the info line
}

// newBrowser opens a browser on start (resolved to an absolute path). A read
// error is left in the returned browser's err field.
func newBrowser(start string, showHidden bool) browser {
	abs, err := filepath.Abs(start)
	if err != nil || abs == "" {
		abs = string(filepath.Separator)
	}
	b := browser{dir: filepath.Clean(abs), showHidden: showHidden}
	b.load()
	return b
}

// load re-reads b.dir into b.entries. A read error is recorded in b.err and the
// previous listing is left untouched for the caller to decide on.
func (b *browser) load() {
	des, err := os.ReadDir(b.dir)
	if err != nil {
		b.err = err
		return
	}
	b.err = nil
	b.msg = ""

	entries := make([]fileEntry, 0, len(des))
	for _, de := range des {
		if !b.showHidden && strings.HasPrefix(de.Name(), ".") {
			continue
		}
		entries = append(entries, statEntry(b.dir, de))
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})
	if filepath.Dir(b.dir) != b.dir {
		entries = append([]fileEntry{{name: "..", isDir: true}}, entries...)
	}

	b.entries = entries
	if n := len(entries); n > 0 {
		b.cursor = clampInt(b.cursor, 0, n-1)
		b.top = clampInt(b.top, 0, n-1)
	} else {
		b.cursor, b.top = 0, 0
	}
}

// statEntry turns a directory entry into a fileEntry, resolving a symlink to
// its target's type while keeping the link text for display.
func statEntry(dir string, de os.DirEntry) fileEntry {
	e := fileEntry{name: de.Name()}
	info, err := de.Info()
	if err != nil {
		e.broken = true
		return e
	}
	e.mode = info.Mode()
	e.size = info.Size()
	e.modTime = info.ModTime()

	if e.mode&fs.ModeSymlink != 0 {
		e.isSymlink = true
		full := filepath.Join(dir, de.Name())
		if tgt, err := os.Readlink(full); err == nil {
			e.linkTarget = tgt
		}
		ti, err := os.Stat(full)
		if err != nil {
			e.broken = true
			return e
		}
		e.mode = ti.Mode()
		e.size = ti.Size()
		e.modTime = ti.ModTime()
	}
	e.isDir = e.mode.IsDir()
	return e
}

// selected returns the highlighted entry.
func (b *browser) selected() (fileEntry, bool) {
	if b.cursor < 0 || b.cursor >= len(b.entries) {
		return fileEntry{}, false
	}
	return b.entries[b.cursor], true
}

// move shifts the cursor by delta rows and keeps it inside a visible-row window.
func (b *browser) move(delta, visible int) {
	if len(b.entries) == 0 {
		return
	}
	b.cursor = clampInt(b.cursor+delta, 0, len(b.entries)-1)
	if visible < 1 {
		visible = 1
	}
	if b.cursor < b.top {
		b.top = b.cursor
	}
	if b.cursor >= b.top+visible {
		b.top = b.cursor - visible + 1
	}
}

// navigate switches to dir, keeping the current listing if dir cannot be read.
func (b *browser) navigate(dir string) {
	nb := newBrowser(dir, b.showHidden)
	if nb.err != nil {
		b.msg = "cannot open " + dir + ": " + rootCause(nb.err)
		return
	}
	*b = nb
}

// enter descends into the highlighted directory (following ".." upward).
func (b *browser) enter() {
	e, ok := b.selected()
	if !ok {
		return
	}
	switch {
	case e.isParent():
		b.navigate(filepath.Dir(b.dir))
	case e.isDir:
		b.navigate(filepath.Join(b.dir, e.name))
	}
}

// parent goes up one directory.
func (b *browser) parent() { b.navigate(filepath.Dir(b.dir)) }

// toggleHidden flips dotfile visibility, keeping the cursor on the same name
// where possible.
func (b *browser) toggleHidden() {
	var name string
	if e, ok := b.selected(); ok {
		name = e.name
	}
	b.showHidden = !b.showHidden
	b.load()
	for i, e := range b.entries {
		if e.name == name {
			b.cursor = i
			break
		}
	}
}

func rootCause(err error) string {
	if pe, ok := err.(*fs.PathError); ok {
		return pe.Err.Error()
	}
	return err.Error()
}

// renderBrowser draws the file picker into the main panel: a directory header,
// the Midnight-Commander-style listing, an info line for the selected entry and
// a key legend. It returns exactly l.mainH lines.
func (m Model) renderBrowser(l layout) string {
	t := m.theme
	b := m.browser
	w := l.mainW
	out := make([]string, 0, l.mainH)

	rows := l.mainH - 3
	if rows < 1 {
		rows = 1
	}

	count := fmt.Sprintf("%d items", len(b.entries))
	if b.showHidden {
		count += " · .hidden"
	}
	headW := w - dispWidth(count) - 1
	if headW < 1 {
		headW = 1
	}
	header := padRight(clip("  "+b.dir, headW), headW) + " " + count
	out = append(out, t.title.Render(clipVisible(header, w)))

	if b.err != nil {
		out = append(out, "")
		out = append(out, t.warnText.Render(clipVisible("  cannot read directory: "+rootCause(b.err), w)))
		for len(out) < l.mainH {
			out = append(out, "")
		}
		return strings.Join(out[:l.mainH], "\n")
	}

	// Keep the cursor inside the window for display without mutating state.
	top := b.top
	if b.cursor < top {
		top = b.cursor
	}
	if b.cursor >= top+rows {
		top = b.cursor - rows + 1
	}
	if top < 0 {
		top = 0
	}

	const dateW, sizeW = 16, 9
	nameW := w - dateW - sizeW - 6
	if nameW < 8 {
		nameW = 8
	}

	for i := 0; i < rows; i++ {
		idx := top + i
		if idx >= len(b.entries) {
			out = append(out, "")
			continue
		}
		e := b.entries[idx]

		name := e.name
		if e.isDir && !e.isParent() {
			name += "/"
		}
		glyph := " "
		switch {
		case e.isDir:
			glyph = "▸"
		case e.broken:
			glyph = "✗"
		case e.isSymlink:
			glyph = "→"
		}
		label := glyph + " " + name
		if e.isSymlink && e.linkTarget != "" {
			label += "  → " + e.linkTarget
		}
		label = padRight(clip(label, nameW), nameW)

		var size string
		switch {
		case e.isParent():
			size = "UP-DIR"
		case e.isDir:
			size = "DIR"
		case e.broken:
			size = "BROKEN"
		default:
			size = humanBytes(e.size)
		}
		size = padLeft(size, sizeW)

		date := ""
		if !e.modTime.IsZero() {
			date = e.modTime.Format("2006-01-02 15:04")
		}
		date = padLeft(date, dateW)

		row := padRight(clip("  "+label+"  "+size+"  "+date, w), w)
		switch {
		case idx == b.cursor:
			row = t.cursor.Render(row)
		case e.isDir:
			row = t.title.Render(row)
		case e.broken:
			row = t.warnText.Render(row)
		default:
			row = t.value.Render(row)
		}
		out = append(out, row)
	}

	out = append(out, t.value.Render(clipVisible(m.browserInfoLine(w), w)))
	legend := "  ⏎ open   → enter   ← up   . hidden   ~ home   / root   esc cancel"
	out = append(out, t.dim.Render(clipVisible(legend, w)))

	for len(out) < l.mainH {
		out = append(out, "")
	}
	return strings.Join(out[:l.mainH], "\n")
}

// browserInfoLine is the Midnight-Commander-style status line under the list:
// the selected entry's full path, mode, size and mtime (or a transient note).
func (m Model) browserInfoLine(w int) string {
	b := m.browser
	if b.msg != "" {
		return "  " + b.msg
	}
	e, ok := b.selected()
	if !ok {
		return "  (empty directory)"
	}
	if e.isParent() {
		return "  .. → " + filepath.Dir(b.dir)
	}
	parts := []string{filepath.Join(b.dir, e.name), e.mode.String()}
	if !e.isDir {
		parts = append(parts, fmt.Sprintf("%s (%d B)", humanBytes(e.size), e.size))
	}
	if !e.modTime.IsZero() {
		parts = append(parts, e.modTime.Format("2006-01-02 15:04:05"))
	}
	return "  " + strings.Join(parts, "   ·   ")
}
