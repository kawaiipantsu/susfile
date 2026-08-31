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

// dispWidth is the number of terminal cells s occupies (ANSI-aware).
func dispWidth(s string) int { return lipgloss.Width(s) }

// padRight pads s with spaces to w display cells (no-op if already wider).
func padRight(s string, w int) string {
	if n := w - dispWidth(s); n > 0 {
		return s + spaces(n)
	}
	return s
}

// padLeft right-aligns s within w display cells (no-op if already wider).
func padLeft(s string, w int) string {
	if n := w - dispWidth(s); n > 0 {
		return spaces(n) + s
	}
	return s
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
