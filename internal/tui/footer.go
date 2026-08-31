package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderFooter returns exactly two lines: key hints, then a progress/status
// line that always ends with the ⟦THUGS⟧ (c) 2026 stamp.
func (m Model) renderFooter(w int) string {
	t := m.theme

	legend := hintsFor(m.view)
	if m.browsing {
		legend = browseHints(m.hasNoFile())
	}

	var hints strings.Builder
	for i, hk := range legend {
		if i > 0 {
			hints.WriteString("  ")
		}
		fmt.Fprintf(&hints, "%s %s", t.title.Render("["+hk.key+"]"), t.dim.Render(hk.desc))
	}
	line1 := clipVisible(hints.String(), w)

	stamp := t.stamp.Render("⟦THUGS⟧") + " " + t.stampC.Render("(c) 2026")
	stampW := lipgloss.Width(stamp)

	var status string
	if m.hasNoFile() {
		status = t.dim.Render("no file yet · pick one from the browser")
	} else if m.res == nil && m.err == nil {
		status = m.spin.View() + " " + t.value.Render(m.stageText()) + " " + m.prog.ViewAs(m.frac)
	} else if m.err != nil {
		status = t.warnText.Render("analysis failed: " + m.err.Error())
	} else {
		status = t.dim.Render("done · " + m.res.Verdict.Kaomoji + " " + m.res.Verdict.Summary)
	}
	status = clipVisible(status, w-stampW-2)

	gap := w - lipgloss.Width(status) - stampW
	if gap < 1 {
		gap = 1
	}
	line2 := status + spaces(gap) + stamp

	return line1 + "\n" + clipVisible(line2, w)
}
