package tui

import "strings"

// logoArt is the susfile mark for the top-left box. Kept to ASCII and ≤ 12
// columns so it never needs rune-unsafe truncation and the info box keeps the
// width.
var logoArt = []string{
	" ___",
	"(o o)  sus",
	"/  V \\ file",
	"------",
	" F P C",
	" Z E R",
}

// renderLogo returns the art padded (rune-safely) to exactly w×h.
func renderLogo(w, h int) string {
	lines := make([]string, h)
	for i := 0; i < h; i++ {
		s := ""
		if i < len(logoArt) {
			s = logoArt[i]
		}
		r := []rune(s)
		if len(r) > w {
			r = r[:w]
		}
		lines[i] = string(r) + strings.Repeat(" ", w-len(r))
	}
	return strings.Join(lines, "\n")
}
