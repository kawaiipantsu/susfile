package tui

import (
	"fmt"
	"strings"
)

// renderHexView draws a scrollable hex dump of the bounded raw prefix.
func (m Model) renderHexView(l layout) string {
	t := m.theme
	rows := l.mainH
	out := make([]string, 0, rows)

	if len(m.raw) == 0 {
		return t.dim.Render("(no bytes buffered for hex view)")
	}

	start := m.hexTop * 16
	for i := 0; i < rows-1; i++ {
		off := start + i*16
		if off >= len(m.raw) {
			out = append(out, "")
			continue
		}
		end := off + 16
		if end > len(m.raw) {
			end = len(m.raw)
		}
		chunk := m.raw[off:end]

		var hexPart, asciiPart strings.Builder
		for j := 0; j < 16; j++ {
			if j == 8 {
				hexPart.WriteByte(' ')
			}
			if j < len(chunk) {
				b := chunk[j]
				cell := fmt.Sprintf("%02x ", b)
				if t.color {
					cell = t.entropyColor(byteHeat(b)).Render(cell)
				}
				hexPart.WriteString(cell)
				asciiPart.WriteString(printableOrDot(b))
			} else {
				hexPart.WriteString("   ")
			}
		}
		line := fmt.Sprintf("%08x  %s %s", off, hexPart.String(), asciiPart.String())
		out = append(out, clipVisible(line, l.mainW))
	}

	last := start + (rows-1)*16
	total := (len(m.raw) + 15) / 16
	footer := fmt.Sprintf("offset 0x%x of 0x%x   rows %d/%d", clampInt(start, 0, len(m.raw)), len(m.raw), m.hexTop+1, total)
	if len(m.raw) < int(m.res.Size) {
		footer += "   (first " + humanBytes(int64(len(m.raw))) + " only)"
	}
	_ = last
	out = append(out, t.dim.Render(clipVisible(footer, l.mainW)))

	for len(out) < rows {
		out = append(out, "")
	}
	return strings.Join(out[:rows], "\n")
}

func printableOrDot(b byte) string {
	if b >= 0x20 && b <= 0x7e {
		return string(rune(b))
	}
	return "·"
}

// byteHeat maps a byte to a pseudo-entropy value for colouring: nulls cold,
// high-bit hot, printable mid.
func byteHeat(b byte) float64 {
	switch {
	case b == 0:
		return 0
	case b >= 0x80:
		return 7.5
	case b >= 0x20 && b <= 0x7e:
		return 3.5
	default:
		return 5.5
	}
}

func (m *Model) hexScroll(delta int, l layout) {
	maxTop := (len(m.raw)+15)/16 - (l.mainH - 1)
	if maxTop < 0 {
		maxTop = 0
	}
	m.hexTop = clampInt(m.hexTop+delta, 0, maxTop)
}

func (m *Model) hexJumpTo(off int64, l layout) {
	m.view = viewHex
	m.hexScrollTo(int(off/16), l)
}

func (m *Model) hexScrollTo(row int, l layout) {
	maxTop := (len(m.raw)+15)/16 - (l.mainH - 1)
	if maxTop < 0 {
		maxTop = 0
	}
	m.hexTop = clampInt(row, 0, maxTop)
}
