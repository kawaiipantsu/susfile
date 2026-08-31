package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"version"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "susfile v") {
		t.Errorf("version output = %q", out.String())
	}
}

func TestRunNoArgsNonInteractive(t *testing.T) {
	// stdout here is a bytes.Buffer, not a TTY, so the picker cannot open and
	// a missing <file> is an error rather than a launch.
	var out, errb bytes.Buffer
	if code := run(nil, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "no <file> given") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRunTooManyArgs(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"a", "b"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "at most one") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRunMissingFile(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{filepath.Join(t.TempDir(), "nope")}, &out, &errb)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
}

func TestRunPlainAndJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(p, []byte(strings.Repeat("hello world\n", 300)), 0o644); err != nil {
		t.Fatal(err)
	}

	var out, errb bytes.Buffer
	if code := run([]string{"--no-tui", p}, &out, &errb); code != 0 {
		t.Fatalf("plain exit %d: %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "File map") {
		t.Errorf("plain output missing File map")
	}

	out.Reset()
	errb.Reset()
	if code := run([]string{"--json", p}, &out, &errb); code != 0 {
		t.Fatalf("json exit %d: %s", code, errb.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(out.String()), "{") {
		t.Errorf("json output does not look like JSON: %q", out.String()[:40])
	}
}

func TestParseMapSize(t *testing.T) {
	for _, tc := range []struct {
		in   string
		w, h int
	}{
		{"64x16", 64, 16},
		{"120x40", 120, 40},
		{"bad", 64, 16},
		{"2x1", 8, 2}, // clamped to minimums
	} {
		w, h := parseMapSize(tc.in)
		if w != tc.w || h != tc.h {
			t.Errorf("parseMapSize(%q) = %dx%d, want %dx%d", tc.in, w, h, tc.w, tc.h)
		}
	}
}
