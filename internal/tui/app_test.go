package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kawaiipantsu/susfile/internal/analyze"
)

func testResult(t *testing.T, data []byte) *analyze.Result {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sample")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := analyze.Analyze(context.Background(), p, analyze.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func readyModel(t *testing.T, data []byte) Model {
	t.Helper()
	m := Model{theme: newTheme(false)}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)
	nm, _ = m.Update(doneMsg{res: testResult(t, data), err: nil})
	m = nm.(Model)
	nm, _ = m.Update(rawMsg{data: data})
	return nm.(Model)
}

func TestComputeLayout(t *testing.T) {
	if computeLayout(40, 20).ok {
		t.Error("40x20 should be below minimum")
	}
	l := computeLayout(120, 40)
	if !l.ok {
		t.Fatal("120x40 should be ok")
	}
	if l.gridW < 1 || l.gridH < 1 || l.mainW < 1 || l.mainH < 1 {
		t.Errorf("degenerate layout: %+v", l)
	}
	if l.topH+1+l.mainH+2+2 > 40+2 { // borders give a little slack
		t.Errorf("layout rows overflow: %+v", l)
	}
}

func TestViewResizePrompt(t *testing.T) {
	m := Model{theme: newTheme(false)}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	out := nm.(Model).View()
	if !strings.Contains(out, "needs at least") {
		t.Errorf("small terminal should show resize prompt, got:\n%s", out)
	}
}

func TestViewRendersAllTabsNoPanic(t *testing.T) {
	m := readyModel(t, []byte(strings.Repeat("package main\nfunc x(){}\n", 400)))
	for v := view(0); v < numViews; v++ {
		m.view = v
		out := m.View()
		if !strings.Contains(out, "THUGS") {
			t.Errorf("view %s: stamp missing", v.title())
		}
		if strings.Count(out, "\n") < 10 {
			t.Errorf("view %s: output suspiciously short:\n%s", v.title(), out)
		}
	}
}

func TestTabCycling(t *testing.T) {
	m := readyModel(t, []byte("hello world\n"))
	start := m.view
	for i := 0; i < int(numViews); i++ {
		nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
		m = nm.(Model)
	}
	if m.view != start {
		t.Errorf("tab did not cycle back to start: %v -> %v", start, m.view)
	}

	nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	if nm.(Model).view != numViews-1 {
		t.Errorf("shift-tab from Map should wrap to last view, got %v", nm.(Model).view)
	}
}

func TestQuitKeys(t *testing.T) {
	m := readyModel(t, []byte("x"))
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyCtrlC},
		{Type: tea.KeyEsc},
	} {
		_, cmd := m.handleKey(k)
		if cmd == nil {
			t.Errorf("key %v should quit", k)
		}
	}
}

func TestMapCursorClamps(t *testing.T) {
	m := readyModel(t, []byte(strings.Repeat("abcd\x00\x01\x02\x03", 4000)))
	m.view = viewMap

	for i := 0; i < 10000; i++ {
		nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRight})
		m = nm.(Model)
	}
	if m.cur != len(m.cells)-1 {
		t.Errorf("cursor should clamp at last cell: cur=%d cells=%d", m.cur, len(m.cells))
	}
	for i := 0; i < 10000; i++ {
		nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyLeft})
		m = nm.(Model)
	}
	if m.cur != 0 {
		t.Errorf("cursor should clamp at 0, got %d", m.cur)
	}
}

func TestEnterFromMapJumpsToHex(t *testing.T) {
	m := readyModel(t, []byte(strings.Repeat("abcdefgh", 8000)))
	m.view = viewMap
	m.cur = len(m.cells) / 2
	want := m.cells[m.cur].Offset

	nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.view != viewHex {
		t.Fatalf("enter should switch to hex, view=%v", m.view)
	}
	if int64(m.hexTop*16) > want || int64((m.hexTop+m.lay.mainH)*16) < want {
		t.Errorf("hex not scrolled near offset %d (hexTop row %d)", want, m.hexTop)
	}
}

func TestHexAndStringsScrollClamp(t *testing.T) {
	m := readyModel(t, []byte(strings.Repeat("word\x00", 5000)))

	m.view = viewHex
	m.hexScroll(1<<20, m.lay)
	if m.hexTop < 0 {
		t.Error("hexTop negative")
	}
	maxTop := (len(m.raw)+15)/16 - (m.lay.mainH - 1)
	if maxTop < 0 {
		maxTop = 0
	}
	if m.hexTop > maxTop {
		t.Errorf("hexTop %d exceeds max %d", m.hexTop, maxTop)
	}

	m.view = viewStrings
	m.strScroll(1<<20, m.lay)
	if m.strTop < 0 || m.strTop > len(m.res.Strings) {
		t.Errorf("strTop out of range: %d", m.strTop)
	}
	m.strScroll(-1<<20, m.lay)
	if m.strTop != 0 {
		t.Errorf("strTop should clamp to 0, got %d", m.strTop)
	}
}

func TestProgressAndErrorFooter(t *testing.T) {
	m := Model{theme: newTheme(false), stage: analyze.StageHash, frac: 0.4}
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = nm.(Model)
	if !strings.Contains(m.renderFooter(100), "THUGS") {
		t.Error("footer missing stamp during progress")
	}

	nm, _ = m.Update(doneMsg{res: nil, err: context.Canceled})
	if !strings.Contains(nm.(Model).renderFooter(100), "failed") {
		t.Error("footer should show failure")
	}
}
