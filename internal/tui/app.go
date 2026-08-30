// Package tui is susfile's interactive terminal UI: a logo box and a label/value
// info box on top, the classified file map filling the lower half, secondary Tab
// views (entropy, histogram, hex, strings), and a footer carrying analysis
// progress and the ⟦THUGS⟧ (c) 2026 stamp. It renders an analyze.Result and
// never measures anything itself.
package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kawaiipantsu/susfile/internal/analyze"
)

// hexBufferCap bounds how many bytes the hex view holds for a large file.
const hexBufferCap = 4 << 20

type view int

const (
	viewMap view = iota
	viewEntropy
	viewHistogram
	viewHex
	viewStrings
	numViews
)

func (v view) title() string {
	return [...]string{"Map", "Entropy", "Histogram", "Hex", "Strings"}[v]
}

type progressMsg struct {
	stage analyze.Stage
	frac  float64
}
type doneMsg struct {
	res *analyze.Result
	err error
}
type rawMsg struct{ data []byte }

// Model is the Bubble Tea model for one susfile session.
type Model struct {
	path string
	opt  analyze.Options

	res *analyze.Result
	err error
	raw []byte

	w, h  int
	lay   layout
	view  view
	theme theme

	stage analyze.Stage
	frac  float64
	prog  progress.Model
	spin  spinner.Model

	cells []analyze.MicroBlock // downsampled grid for the current layout
	cur   int                  // inspector index into cells

	hexTop int
	strTop int
}

// Run analyses path and shows the TUI until the user quits.
func Run(path string, opt analyze.Options, color bool) error {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := Model{
		path:  path,
		opt:   opt,
		theme: newTheme(color),
		prog:  progress.New(progressOpts(color)...),
		spin:  sp,
		stage: analyze.StageOpen,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		res, err := analyze.Analyze(context.Background(), path, opt, func(s analyze.Stage, f float64) {
			p.Send(progressMsg{s, f})
		})
		p.Send(doneMsg{res, err})
	}()
	go func() { p.Send(rawMsg{loadRaw(path, opt)}) }()

	_, err := p.Run()
	return err
}

// Frame analyses path and returns a single rendered frame at size w×h with the
// named view active ("map", "entropy", "histogram", "hex", "strings"). It runs
// no Bubble Tea program and needs no terminal — it is used for documentation
// screenshots and golden tests.
func Frame(path string, opt analyze.Options, w, h int, viewName string, color bool) (string, error) {
	res, err := analyze.Analyze(context.Background(), path, opt, nil)
	if err != nil {
		return "", err
	}
	m := Model{theme: newTheme(color), path: path, prog: progress.New(progressOpts(color)...), spin: spinner.New()}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = nm.(Model)
	nm, _ = m.Update(doneMsg{res: res})
	m = nm.(Model)
	nm, _ = m.Update(rawMsg{data: loadRaw(path, opt)})
	m = nm.(Model)
	for v := view(0); v < numViews; v++ {
		if v.title() == viewNameTitle(viewName) {
			m.view = v
		}
	}
	return m.View(), nil
}

func viewNameTitle(s string) string {
	switch s {
	case "map":
		return "Map"
	case "entropy":
		return "Entropy"
	case "histogram":
		return "Histogram"
	case "hex":
		return "Hex"
	case "strings":
		return "Strings"
	}
	return "Map"
}

func progressOpts(color bool) []progress.Option {
	if color {
		return []progress.Option{progress.WithScaledGradient("#3b5bdb", "#e03131"), progress.WithoutPercentage()}
	}
	return []progress.Option{progress.WithSolidFill("15"), progress.WithoutPercentage()}
}

// loadRaw reads a bounded prefix for the hex view. Errors are swallowed — the
// hex view simply shows "(no bytes buffered)".
func loadRaw(path string, opt analyze.Options) []byte {
	if path == "-" {
		b, _ := io.ReadAll(io.LimitReader(os.Stdin, hexBufferCap))
		return b
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	cap := int64(hexBufferCap)
	if opt.MaxBytes > 0 && opt.MaxBytes < cap {
		cap = opt.MaxBytes
	}
	b, _ := io.ReadAll(io.LimitReader(f, cap))
	return b
}

// Init starts the spinner; analysis is kicked off from Run.
func (m Model) Init() tea.Cmd {
	return m.spin.Tick
}

// Update handles resize, keys, analysis progress and completion.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.lay = computeLayout(m.w, m.h)
		m.rebuildCells(m.lay)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case progressMsg:
		m.stage, m.frac = msg.stage, msg.frac
		return m, nil

	case doneMsg:
		m.res, m.err = msg.res, msg.err
		m.stage = analyze.StageDone
		m.frac = 1
		if m.lay.ok {
			m.rebuildCells(m.lay)
		}
		return m, nil

	case rawMsg:
		m.raw = msg.data
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if isQuit(k) {
		return m, tea.Quit
	}
	switch k.String() {
	case "tab", "l":
		m.view = (m.view + 1) % numViews
	case "shift+tab", "h":
		m.view = (m.view + numViews - 1) % numViews
	case "r":
		return m.rescan()
	}

	switch m.view {
	case viewMap:
		switch k.String() {
		case "up":
			m.moveCursor(0, -1, m.lay.gridW)
		case "down":
			m.moveCursor(0, 1, m.lay.gridW)
		case "left":
			m.moveCursor(-1, 0, m.lay.gridW)
		case "right":
			m.moveCursor(1, 0, m.lay.gridW)
		case "enter":
			if len(m.cells) > 0 {
				m.hexJumpTo(m.cells[clampInt(m.cur, 0, len(m.cells)-1)].Offset, m.lay)
			}
		}
	case viewHex:
		switch k.String() {
		case "up":
			m.hexScroll(-1, m.lay)
		case "down":
			m.hexScroll(1, m.lay)
		case "pgup":
			m.hexScroll(-(m.lay.mainH - 2), m.lay)
		case "pgdown", " ":
			m.hexScroll(m.lay.mainH-2, m.lay)
		case "g", "home":
			m.hexTop = 0
		case "G", "end":
			m.hexScroll(1<<30, m.lay)
		}
	case viewStrings:
		switch k.String() {
		case "up":
			m.strScroll(-1, m.lay)
		case "down":
			m.strScroll(1, m.lay)
		case "pgup":
			m.strScroll(-(m.lay.mainH - 2), m.lay)
		case "pgdown", " ":
			m.strScroll(m.lay.mainH-2, m.lay)
		case "g", "home":
			m.strTop = 0
		case "G", "end":
			m.strScroll(1<<30, m.lay)
		}
	}
	return m, nil
}

func (m Model) rescan() (tea.Model, tea.Cmd) {
	m.res, m.err = nil, nil
	m.stage, m.frac = analyze.StageOpen, 0
	m.cells = nil
	path, opt := m.path, m.opt
	return m, func() tea.Msg {
		res, err := analyze.Analyze(context.Background(), path, opt, nil)
		return doneMsg{res, err}
	}
}

// View renders one frame: logo + info box, the tab bar, the active panel, and
// the footer with progress and the ⟦THUGS⟧ (c) 2026 stamp.
func (m Model) View() string {
	if m.w == 0 {
		return "starting susfile…"
	}
	if !m.lay.ok {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("susfile needs at least %d×%d — this terminal is %d×%d", minW, minH, m.w, m.h))
	}
	l := m.lay
	t := m.theme

	// Top: logo box | info box.
	logoBox := t.border.Width(l.logoW - 2).Height(l.topH - 2).Render(renderLogo(l.logoW-2, l.topH-2))
	infoW := m.w - l.logoW
	infoBox := t.border.Width(infoW - 2).Height(l.topH - 2).Render(m.renderInfo(infoW-2, l.topH-2))
	top := lipgloss.JoinHorizontal(lipgloss.Top, logoBox, infoBox)

	// Tab bar.
	var tabs strings.Builder
	for v := view(0); v < numViews; v++ {
		if v == m.view {
			tabs.WriteString(t.tabOn.Render(v.title()))
		} else {
			tabs.WriteString(t.tabOff.Render(v.title()))
		}
		tabs.WriteString(" ")
	}

	// Main panel.
	var body string
	switch m.view {
	case viewMap:
		body = m.renderMap(l)
	case viewEntropy:
		body = m.renderEntropyView(l)
	case viewHistogram:
		body = m.renderHistogramView(l)
	case viewHex:
		body = m.renderHexView(l)
	case viewStrings:
		body = m.renderStringsView(l)
	}
	mainBox := t.border.Width(l.mainW).Height(l.mainH).Render(body)

	footer := m.renderFooter(m.w)

	return strings.Join([]string{top, clipVisible(tabs.String(), m.w), mainBox, footer}, "\n")
}
