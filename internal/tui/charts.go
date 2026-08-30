package tui

import (
	"fmt"
	"math"
	"strings"
)

var barRamp = []rune(" ▁▂▃▄▅▆▇█")

// columnChart draws values (each already scaled to 0..1) as vertical bars in a
// w×h grid, bottom-aligned, using eighth-block glyphs. Returns h lines.
func columnChart(values []float64, w, h int, style func(v float64) string) []string {
	cols := make([][]rune, w)
	for c := 0; c < w; c++ {
		var v float64
		if len(values) > 0 {
			lo := c * len(values) / w
			hi := (c + 1) * len(values) / w
			if hi <= lo {
				hi = lo + 1
			}
			if hi > len(values) {
				hi = len(values)
			}
			for _, x := range values[lo:hi] {
				v += x
			}
			v /= float64(hi - lo)
		}
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		units := int(math.Round(v * float64(h*8)))
		col := make([]rune, h)
		for row := 0; row < h; row++ {
			rowFromBottom := h - 1 - row
			rem := units - rowFromBottom*8
			switch {
			case rem >= 8:
				col[row] = '█'
			case rem <= 0:
				col[row] = ' '
			default:
				col[row] = barRamp[rem]
			}
		}
		cols[c] = col
	}

	lines := make([]string, h)
	for row := 0; row < h; row++ {
		var b strings.Builder
		for c := 0; c < w; c++ {
			b.WriteRune(cols[c][row])
		}
		lines[row] = b.String()
	}
	_ = style
	return lines
}

// renderEntropyView is the windowed entropy chart Tab view.
func (m Model) renderEntropyView(l layout) string {
	t := m.theme
	if m.res == nil {
		return ""
	}
	vals := make([]float64, len(m.res.MicroBlocks))
	for i, b := range m.res.MicroBlocks {
		vals[i] = b.Entropy / 8
	}

	chartH := l.mainH - 2
	if chartH < 1 {
		chartH = 1
	}
	lines := columnChart(vals, l.mainW-6, chartH, nil)

	out := make([]string, 0, l.mainH)
	for i, ln := range lines {
		// y-axis labels at top / mid / bottom
		lbl := "    "
		switch i {
		case 0:
			lbl = "8 ┤ "
		case chartH / 2:
			lbl = "4 ┤ "
		case chartH - 1:
			lbl = "0 ┼ "
		}
		out = append(out, t.dim.Render(lbl)+t.entropyColor(8*float64(chartH-i)/float64(chartH)).Render(ln))
	}
	axis := "     " + strings.Repeat("─", clampInt(l.mainW-6, 0, l.mainW))
	out = append(out, t.dim.Render(axis))
	out = append(out, t.dim.Render(fmt.Sprintf("     0x0%s0x%x",
		strings.Repeat(" ", clampInt(l.mainW-18, 1, l.mainW)), m.res.Size)))

	for len(out) < l.mainH {
		out = append(out, "")
	}
	return strings.Join(out[:l.mainH], "\n")
}

// renderHistogramView is the 256-bucket byte-frequency Tab view.
func (m Model) renderHistogramView(l layout) string {
	t := m.theme
	if m.res == nil {
		return ""
	}
	var maxc uint64
	for _, c := range m.res.Histogram {
		if c > maxc {
			maxc = c
		}
	}
	vals := make([]float64, 256)
	if maxc > 0 {
		lm := math.Log1p(float64(maxc))
		for i, c := range m.res.Histogram {
			vals[i] = math.Log1p(float64(c)) / lm
		}
	}

	chartH := l.mainH - 2
	if chartH < 1 {
		chartH = 1
	}
	lines := columnChart(vals, l.mainW, chartH, nil)
	out := make([]string, 0, l.mainH)
	for _, ln := range lines {
		out = append(out, t.value.Render(ln))
	}
	out = append(out, t.dim.Render(strings.Repeat("─", clampInt(l.mainW, 0, l.mainW))))
	out = append(out, t.dim.Render(fmt.Sprintf("byte 0x00 → 0xff   (log scale, peak %d)", maxc)))
	for len(out) < l.mainH {
		out = append(out, "")
	}
	return strings.Join(out[:l.mainH], "\n")
}
