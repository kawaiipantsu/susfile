package tui

// minW and minH are the smallest terminal the TUI will draw in. Below this it
// shows a resize prompt instead.
const (
	minW = 80
	minH = 24
)

// layout holds the computed pixel budget for one frame.
type layout struct {
	w, h int

	ok bool // terminal is at least minW x minH

	logoW int // width of the logo box (outer, incl. border)
	topH  int // height of the top region (logo + info), incl. borders

	// main panel inner content area (inside its border)
	mainW, mainH int

	// file-map grid inside the main panel (leaves room for inspector + legend)
	gridW, gridH int
}

func computeLayout(w, h int) layout {
	l := layout{w: w, h: h, ok: w >= minW && h >= minH}
	if !l.ok {
		return l
	}

	l.logoW = 14
	l.topH = 10
	if half := h / 2; l.topH > half {
		l.topH = half
	}
	if l.topH < 8 {
		l.topH = 8
	}

	// Rows: topH + 1 tab bar + mainOuter + 2 footer == h
	mainOuter := h - l.topH - 1 - 2
	if mainOuter < 5 {
		mainOuter = 5
	}
	l.mainW = w - 2
	l.mainH = mainOuter - 2

	l.gridW = l.mainW
	l.gridH = l.mainH - 3 // 1 inspector line + 2 legend lines
	if l.gridH < 1 {
		l.gridH = 1
	}
	if l.gridW < 1 {
		l.gridW = 1
	}
	return l
}
