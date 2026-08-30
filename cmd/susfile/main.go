// Command susfile is a CLI file-forensics visualiser: it reads one file and
// shows what it is — magic header, MIME/type, hashes, executable structure —
// with a defrag-screen "file map" of classified byte regions as the centrepiece.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kawaiipantsu/susfile/internal/analyze"
	"github.com/kawaiipantsu/susfile/internal/report"
	"github.com/kawaiipantsu/susfile/internal/tui"
	"github.com/kawaiipantsu/susfile/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type flags struct {
	noTUI        bool
	jsonOut      bool
	noColor      bool
	allowSpecial bool
	stringsMin   int
	maxBytes     int64
	mapSize      string
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 {
		switch args[0] {
		case "version", "--version", "-V":
			fmt.Fprintln(stdout, version.String())
			return 0
		}
	}

	fs := flag.NewFlagSet("susfile", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usageText) }

	var f flags
	fs.BoolVar(&f.noTUI, "no-tui", false, "plain text report instead of the TUI")
	fs.BoolVar(&f.jsonOut, "json", false, "machine-readable JSON report")
	fs.BoolVar(&f.noColor, "no-color", false, "disable colour output")
	fs.BoolVar(&f.allowSpecial, "allow-special", false, "permit non-regular files")
	fs.IntVar(&f.stringsMin, "strings-min", analyze.DefaultStringsMin, "minimum length for extracted strings")
	fs.Int64Var(&f.maxBytes, "max-bytes", analyze.DefaultMaxBytes, "cap bytes read for the detail passes")
	fs.StringVar(&f.mapSize, "map-size", "64x16", "plain-mode class-map grid, WxH")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(stderr, "susfile: expected exactly one <file> argument (use - for stdin)")
		fmt.Fprint(stderr, usageText)
		return 2
	}
	path := rest[0]

	mapW, mapH := parseMapSize(f.mapSize)
	opt := analyze.Options{
		MaxBytes:     f.maxBytes,
		StringsMin:   f.stringsMin,
		AllowSpecial: f.allowSpecial,
	}
	color := !f.noColor && os.Getenv("NO_COLOR") == ""

	// Interactive TUI is the default when we have a terminal and no output mode
	// was requested.
	if !f.noTUI && !f.jsonOut && isTTY(stdout) && isTTY(os.Stdin) && path != "-" {
		if err := tui.Run(path, opt, color); err != nil {
			fmt.Fprintf(stderr, "susfile: %v\n", err)
			return 1
		}
		return 0
	}

	res, err := analyze.Analyze(context.Background(), path, opt, nil)
	if err != nil {
		fmt.Fprintf(stderr, "susfile: %v\n", err)
		return 1
	}

	if f.jsonOut {
		if err := report.JSON(stdout, res); err != nil {
			fmt.Fprintf(stderr, "susfile: writing JSON: %v\n", err)
			return 1
		}
		return 0
	}

	report.Plain(stdout, res, report.Options{Color: color && isTTY(stdout), MapW: mapW, MapH: mapH})
	return 0
}

func parseMapSize(s string) (w, h int) {
	w, h = 64, 16
	_, _ = fmt.Sscanf(s, "%dx%d", &w, &h)
	if w < 8 {
		w = 8
	}
	if h < 2 {
		h = 2
	}
	return w, h
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

const usageText = `susfile — CLI file-forensics visualiser

Usage:
  susfile [flags] <file>      analyse a file (use - for stdin)
  susfile version             print build metadata

Flags:
  --no-tui          plain text report instead of the TUI
  --json            machine-readable JSON report
  --strings-min N   minimum length for extracted strings (default 4)
  --max-bytes N     cap bytes read for the detail passes (default 256 MiB)
  --map-size WxH    plain-mode class-map grid (default 64x16)
  --allow-special   permit analysing non-regular files (devices, FIFOs)
  --no-color        disable colour output (also honours NO_COLOR)
`
