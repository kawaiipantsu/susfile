package tui

import (
	"fmt"
	"strings"

	"github.com/kawaiipantsu/susfile/internal/analyze"
	"github.com/kawaiipantsu/susfile/internal/report"
)

// renderMap draws the centrepiece: the file as a grid of classified block
// cells, a movable inspector cursor, and a two-row legend. It returns exactly
// l.mainH lines.
func (m Model) renderMap(l layout) string {
	t := m.theme
	lines := make([]string, 0, l.mainH)

	rows := (len(m.cells) + l.gridW - 1) / l.gridW
	if rows > l.gridH {
		rows = l.gridH
	}

	for row := 0; row < l.gridH; row++ {
		var b strings.Builder
		for col := 0; col < l.gridW; col++ {
			idx := row*l.gridW + col
			if idx >= len(m.cells) {
				b.WriteByte(' ')
				continue
			}
			c := m.cells[idx]
			glyph := string(c.Class.Glyph())
			switch {
			case idx == m.cur:
				b.WriteString(t.cursor.Render(glyph))
			case !t.color:
				if c.Class == analyze.ClassEmpty {
					b.WriteByte(' ')
				} else {
					b.WriteRune(shadeGlyph(c))
				}
			default:
				b.WriteString(t.entropyColor(c.Entropy).Render(glyph))
			}
		}
		lines = append(lines, b.String())
		_ = rows
	}

	lines = append(lines, m.inspectorLine(l.mainW))
	lines = append(lines, legendLines(t, l.mainW)...)

	for len(lines) < l.mainH {
		lines = append(lines, "")
	}
	return strings.Join(lines[:l.mainH], "\n")
}

// shadeGlyph is used on the no-colour path: keep the class letter but let a
// space stand in for the lowest-entropy padding so structure still reads.
func shadeGlyph(c analyze.MicroBlock) rune {
	if c.Class == analyze.ClassNull || c.Class == analyze.ClassEmpty {
		return shadeFor(c.Entropy)
	}
	return c.Class.Glyph()
}

func (m Model) inspectorLine(w int) string {
	t := m.theme
	if len(m.cells) == 0 {
		return t.dim.Render("(no data)")
	}
	c := m.cells[clampInt(m.cur, 0, len(m.cells)-1)]
	end := c.Offset + int64(c.Len)
	s := fmt.Sprintf("▸ 0x%x–0x%x  entropy %.2f  class %s (%s)  top 0x%02x",
		c.Offset, end, c.Entropy, string(c.Class.Glyph()), c.Class.Name(), c.TopByte)
	if c.Section != "" {
		s += "  §" + c.Section
	}
	return clipVisible(t.value.Render(s), w)
}

func legendLines(t theme, w int) []string {
	var b strings.Builder
	for i, e := range report.Legend() {
		if i > 0 {
			b.WriteString("  ")
		}
		fmt.Fprintf(&b, "%c %s", e.Glyph, e.Name)
	}
	l1 := clipVisible(t.dim.Render(b.String()), w)
	l2 := clipVisible(t.dim.Render("colour = entropy  (blue low → red high)   ↑↓←→ move  ⏎ hex here"), w)
	return []string{l1, l2}
}

// rebuildCells recomputes the downsampled grid for the current layout.
func (m *Model) rebuildCells(l layout) {
	if m.res == nil {
		m.cells = nil
		return
	}
	m.cells = analyze.Downsample(m.res.MicroBlocks, l.gridW*l.gridH)
	if m.cur >= len(m.cells) {
		m.cur = len(m.cells) - 1
	}
	if m.cur < 0 {
		m.cur = 0
	}
}

// moveCursor shifts the inspector within the grid.
func (m *Model) moveCursor(dcol, drow, gridW int) {
	if len(m.cells) == 0 {
		return
	}
	m.cur = clampInt(m.cur+dcol+drow*gridW, 0, len(m.cells)-1)
}
