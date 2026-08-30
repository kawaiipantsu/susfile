package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// theme holds every style the TUI draws with, so colour can be turned off in
// one place.
type theme struct {
	color bool

	border   lipgloss.Style
	title    lipgloss.Style
	label    lipgloss.Style
	value    lipgloss.Style
	dim      lipgloss.Style
	tabOn    lipgloss.Style
	tabOff   lipgloss.Style
	cursor   lipgloss.Style
	stamp    lipgloss.Style
	stampC   lipgloss.Style
	verdict  lipgloss.Style
	warnText lipgloss.Style
}

func newTheme(color bool) theme {
	mk := func(s lipgloss.Style) lipgloss.Style {
		return s
	}
	t := theme{
		color:    color,
		border:   mk(lipgloss.NewStyle().Border(lipgloss.RoundedBorder())),
		title:    mk(lipgloss.NewStyle().Bold(true)),
		label:    mk(lipgloss.NewStyle()),
		value:    mk(lipgloss.NewStyle()),
		dim:      mk(lipgloss.NewStyle()),
		tabOn:    mk(lipgloss.NewStyle().Bold(true)),
		tabOff:   mk(lipgloss.NewStyle()),
		cursor:   mk(lipgloss.NewStyle().Reverse(true)),
		stamp:    mk(lipgloss.NewStyle().Bold(true)),
		stampC:   mk(lipgloss.NewStyle().Bold(true)),
		verdict:  mk(lipgloss.NewStyle().Bold(true)),
		warnText: mk(lipgloss.NewStyle()),
	}
	if color {
		fg := func(s string) lipgloss.Color { return lipgloss.Color(s) }
		t.border = t.border.BorderForeground(fg("240"))
		t.title = t.title.Foreground(fg("81"))
		t.label = t.label.Foreground(fg("245"))
		t.value = t.value.Foreground(fg("252"))
		t.dim = t.dim.Foreground(fg("240"))
		t.tabOn = t.tabOn.Foreground(fg("231")).Background(fg("57")).Padding(0, 1)
		t.tabOff = t.tabOff.Foreground(fg("245")).Padding(0, 1)
		t.stamp = t.stamp.Foreground(fg("196"))
		t.stampC = t.stampC.Foreground(fg("240"))
		t.verdict = t.verdict.Foreground(fg("214"))
		t.warnText = t.warnText.Foreground(fg("203"))
	} else {
		t.tabOn = t.tabOn.Padding(0, 1).Underline(true)
		t.tabOff = t.tabOff.Padding(0, 1)
	}
	return t
}

// entropyRamp is the blue→red 256-colour ramp keyed by entropy 0..8.
var entropyRamp = []string{"27", "33", "39", "45", "48", "83", "154", "220", "208", "202", "196"}

// entropyColor returns the lipgloss colour for an entropy value, or an empty
// style when colour is disabled.
func (t theme) entropyColor(e float64) lipgloss.Style {
	if !t.color {
		return lipgloss.NewStyle()
	}
	i := int(e / 8 * float64(len(entropyRamp)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(entropyRamp) {
		i = len(entropyRamp) - 1
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(entropyRamp[i]))
}

// shadeRamp renders entropy as a block-shade glyph for the no-colour path.
var shadeRamp = []rune(" ░▒▓█")

func shadeFor(e float64) rune {
	i := int(e / 8 * float64(len(shadeRamp)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(shadeRamp) {
		i = len(shadeRamp) - 1
	}
	return shadeRamp[i]
}
