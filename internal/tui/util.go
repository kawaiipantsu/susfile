package tui

import "github.com/charmbracelet/lipgloss"

// clipVisible truncates s to at most w visible cells, leaving ANSI escapes
// intact.
func clipVisible(s string, w int) string {
	if w < 0 {
		w = 0
	}
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
