package tui

import (
	"fmt"
	"strings"
)

// renderStringsView lists the extracted strings with their offsets.
func (m Model) renderStringsView(l layout) string {
	t := m.theme
	rows := l.mainH
	out := make([]string, 0, rows)

	if m.res == nil || len(m.res.Strings) == 0 {
		return t.dim.Render("(no strings found)")
	}
	hits := m.res.Strings

	for i := 0; i < rows-1; i++ {
		idx := m.strTop + i
		if idx >= len(hits) {
			out = append(out, "")
			continue
		}
		h := hits[idx]
		enc := "a"
		if h.Encoding == "utf16le" {
			enc = "u"
		}
		text := strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f {
				return '·'
			}
			return r
		}, h.Text)
		line := fmt.Sprintf("%08x %s  %s", h.Offset, enc, text)
		out = append(out, clipVisible(t.value.Render(line), l.mainW))
	}

	shown := clampInt(m.strTop+rows-1, 0, len(hits))
	footer := fmt.Sprintf("%d–%d of %d strings (min length %d)", m.strTop+1, shown, m.res.StringsTotal, m.res.StringsMin)
	if m.res.StringsTotal > len(hits) {
		footer += fmt.Sprintf("   (%d retained)", len(hits))
	}
	out = append(out, t.dim.Render(clipVisible(footer, l.mainW)))

	for len(out) < rows {
		out = append(out, "")
	}
	return strings.Join(out[:rows], "\n")
}

func (m *Model) strScroll(delta int, l layout) {
	if m.res == nil {
		return
	}
	maxTop := len(m.res.Strings) - (l.mainH - 1)
	if maxTop < 0 {
		maxTop = 0
	}
	m.strTop = clampInt(m.strTop+delta, 0, maxTop)
}
