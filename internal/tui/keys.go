package tui

import tea "github.com/charmbracelet/bubbletea"

// keyHint is one entry in the footer's key legend.
type keyHint struct {
	key  string
	desc string
}

// hintsFor returns the context-sensitive key hints for a view.
func hintsFor(v view) []keyHint {
	common := []keyHint{{"tab", "view"}, {"o", "open"}, {"r", "rescan"}, {"q", "quit"}}
	switch v {
	case viewMap:
		return append([]keyHint{{"↑↓←→", "inspect"}, {"enter", "hex here"}}, common...)
	case viewHex, viewStrings:
		return append([]keyHint{{"↑↓", "scroll"}, {"pgup/pgdn", "page"}, {"g/G", "top/end"}}, common...)
	default:
		return common
	}
}

// browseHints is the footer legend while the file picker is open. With no file
// loaded yet, Esc quits rather than returning to a previous view.
func browseHints(noFile bool) []keyHint {
	last := keyHint{"esc", "cancel"}
	if noFile {
		last = keyHint{"esc", "quit"}
	}
	return []keyHint{
		{"↑↓", "move"}, {"⏎", "open"}, {"→←", "dir"},
		{".", "hidden"}, {"~", "home"}, last,
	}
}

func isQuit(m tea.KeyMsg) bool {
	switch m.String() {
	case "q", "ctrl+c", "esc":
		return true
	}
	return false
}
