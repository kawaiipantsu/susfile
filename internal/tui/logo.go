package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// logoArt is the susfile mark for the top-left box: a magnifier over a
// fingerprint, the wordmark, and the hex for "SUS". ASCII only and ≤ 20 columns
// so it never needs rune-unsafe truncation.
var logoArt = []string{
	"   ______",
	"  / (~~) \\",
	" | (~~~~) |   s u s",
	" | (~~~~) |   ------",
	"  \\ (~~) /    f i l e",
	"   \\____/\\",
	"         \\\\  forensic",
	" -=[ 53 . 55 . 53 ]=-",
}

// renderLogo returns the art tinted (magnifier grey, fingerprint + "s u s" +
// hex blue) and padded rune-safely to exactly w×h.
func (m Model) renderLogo(w, h int) string {
	blue := m.theme.title // bold blue in colour mode, bold otherwise
	dim := m.theme.dim    // grey
	ink := m.theme.value  // default text

	lines := make([]string, h)
	for i := 0; i < h; i++ {
		raw := ""
		if i < len(logoArt) {
			raw = logoArt[i]
		}
		r := []rune(raw)
		if len(r) > w {
			r = r[:w]
		}
		raw = string(r)

		var styled string
		switch {
		case strings.Contains(raw, "s u s"):
			styled = dim.Render(strings.Replace(raw, "s u s", "", 1)) + blue.Render("s u s")
		case strings.Contains(raw, "f i l e"):
			styled = dim.Render(strings.Replace(raw, "f i l e", "", 1)) + ink.Render("f i l e")
		case strings.Contains(raw, "forensic"):
			styled = dim.Render(strings.Replace(raw, "forensic", "", 1)) + dim.Render("forensic")
		case strings.Contains(raw, "53"):
			styled = blue.Render(raw)
		case strings.Contains(raw, "~"):
			styled = tintTilde(raw, dim, blue)
		default:
			styled = dim.Render(raw)
		}
		lines[i] = styled + strings.Repeat(" ", w-len(r))
	}
	return strings.Join(lines, "\n")
}

// tintTilde renders a magnifier line with the '~' fingerprint runs in blue and
// everything else dim.
func tintTilde(s string, dim, blue lipgloss.Style) string {
	var b strings.Builder
	var run []rune
	tilde := false
	flush := func() {
		if len(run) == 0 {
			return
		}
		if tilde {
			b.WriteString(blue.Render(string(run)))
		} else {
			b.WriteString(dim.Render(string(run)))
		}
		run = run[:0]
	}
	for _, c := range s {
		isT := c == '~'
		if isT != tilde {
			flush()
			tilde = isT
		}
		run = append(run, c)
	}
	flush()
	return b.String()
}
