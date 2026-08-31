package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func mustW(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func entryNames(b browser) []string {
	names := make([]string, len(b.entries))
	for i, e := range b.entries {
		names[i] = e.name
	}
	return names
}

func hasName(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func selectName(b *browser, name string) bool {
	for i, e := range b.entries {
		if e.name == name {
			b.cursor = i
			return true
		}
	}
	return false
}

// drain runs a (possibly batched) command and reports whether any leaf message
// is a doneMsg for a completed analysis.
func drainHasDone(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case doneMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if drainHasDone(c) {
				return true
			}
		}
	}
	return false
}

func TestBrowserListsDirsFirstWithParent(t *testing.T) {
	dir := t.TempDir()
	mustW(t, os.Mkdir(filepath.Join(dir, "zsub"), 0o755))
	mustW(t, os.Mkdir(filepath.Join(dir, "asub"), 0o755))
	mustW(t, os.WriteFile(filepath.Join(dir, "bfile.txt"), []byte("hi"), 0o644))
	mustW(t, os.WriteFile(filepath.Join(dir, "afile.txt"), []byte("hi"), 0o644))
	mustW(t, os.WriteFile(filepath.Join(dir, ".dotfile"), []byte("hi"), 0o644))

	b := newBrowser(dir, false)
	if b.err != nil {
		t.Fatalf("newBrowser: %v", b.err)
	}
	got := strings.Join(entryNames(b), ",")
	want := "..,asub,zsub,afile.txt,bfile.txt"
	if got != want {
		t.Fatalf("listing order = %q, want %q", got, want)
	}
	if hasName(entryNames(b), ".dotfile") {
		t.Error("dotfile shown while showHidden is false")
	}

	b.toggleHidden()
	if !hasName(entryNames(b), ".dotfile") {
		t.Errorf("toggleHidden did not reveal .dotfile: %v", entryNames(b))
	}
}

func TestBrowserEnterAndParent(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "child")
	mustW(t, os.Mkdir(sub, 0o755))
	mustW(t, os.WriteFile(filepath.Join(sub, "leaf.bin"), []byte("data"), 0o644))

	b := newBrowser(root, false)
	if !selectName(&b, "child") {
		t.Fatal("child dir not listed")
	}
	b.enter()
	if b.dir != sub {
		t.Fatalf("enter: dir = %q, want %q", b.dir, sub)
	}
	if !hasName(entryNames(b), "leaf.bin") {
		t.Fatalf("child listing missing leaf.bin: %v", entryNames(b))
	}
	b.parent()
	if b.dir != root {
		t.Fatalf("parent: dir = %q, want %q", b.dir, root)
	}
}

func TestBrowserNavigateToUnreadableIsNoop(t *testing.T) {
	root := t.TempDir()
	b := newBrowser(root, false)
	b.navigate(filepath.Join(root, "nope"))
	if b.dir != root {
		t.Errorf("failed navigate changed dir to %q", b.dir)
	}
	if b.msg == "" {
		t.Error("failed navigate left no status message")
	}
}

func TestBrowserOpenKeyAndCancel(t *testing.T) {
	m := readyModel(t, []byte("x"))
	m.path = filepath.Join(t.TempDir(), "placeholder")

	nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	m = nm.(Model)
	if !m.browsing {
		t.Fatal("o did not open the file picker")
	}
	if !strings.Contains(m.View(), "Pick file") {
		t.Error("picker banner missing from the frame")
	}

	nm, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.browsing {
		t.Error("esc did not close the picker")
	}
	if cmd != nil {
		t.Error("esc inside the picker must not quit the program")
	}
}

func TestBrowserEnterAnalysesPickedFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "pick-me.txt")
	mustW(t, os.WriteFile(target, []byte(strings.Repeat("abcd", 128)), 0o644))

	m := readyModel(t, []byte("original bytes"))
	m.browser = newBrowser(dir, false)
	m.browsing = true
	if !selectName(&m.browser, "pick-me.txt") {
		t.Fatal("target file not listed")
	}

	nm, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if m.browsing {
		t.Error("picker still open after choosing a file")
	}
	if m.path != target {
		t.Errorf("path = %q, want %q", m.path, target)
	}
	if m.res != nil || m.raw != nil || m.cells != nil {
		t.Error("derived state not cleared for the new file")
	}
	if !drainHasDone(cmd) {
		t.Error("choosing a file did not kick off a fresh analysis")
	}
}

func TestBrowserRefusesDirectoryOnEnter(t *testing.T) {
	dir := t.TempDir()
	mustW(t, os.Mkdir(filepath.Join(dir, "adir"), 0o755))

	m := readyModel(t, []byte("x"))
	m.browser = newBrowser(dir, false)
	m.browsing = true
	selectName(&m.browser, "adir")

	nm, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)
	if !m.browsing {
		t.Error("Enter on a directory should stay in the picker")
	}
	if m.browser.dir != filepath.Join(dir, "adir") {
		t.Errorf("Enter on a directory did not descend: %q", m.browser.dir)
	}
	if cmd != nil {
		t.Error("Enter on a directory should not start an analysis")
	}
}

func TestBrowserViewKeepsStamp(t *testing.T) {
	m := readyModel(t, []byte(strings.Repeat("x", 300)))
	m.path = filepath.Join(t.TempDir(), "f")
	nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	out := nm.(Model).View()
	if !strings.Contains(out, "THUGS") {
		t.Errorf("stamp missing while browsing:\n%s", out)
	}
	if strings.Count(out, "\n") < 10 {
		t.Errorf("browsing frame suspiciously short:\n%s", out)
	}
}
