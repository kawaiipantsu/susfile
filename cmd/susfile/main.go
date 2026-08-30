// Command susfile is a CLI file-forensics visualiser: it reads one file and
// shows what it is — magic header, MIME/type, hashes, executable structure —
// with a defrag-screen "file map" of classified byte regions as the centrepiece.
package main

import (
	"fmt"
	"os"

	"github.com/kawaiipantsu/susfile/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version" || args[0] == "-V") {
		fmt.Println(version.String())
		return 0
	}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
		usage(os.Stdout)
		return 0
	}

	// Full analysis and TUI land in feature/core-analysis and feature/tui.
	fmt.Fprintln(os.Stderr, "susfile: analysis is not wired up yet on this build.")
	fmt.Fprintln(os.Stderr, "Try: susfile version")
	return 2
}

func usage(w *os.File) {
	fmt.Fprintf(w, `susfile — CLI file-forensics visualiser

Usage:
  susfile [flags] <file>
  susfile version

Flags (planned):
  --no-tui        plain text report instead of the TUI
  --json          machine-readable JSON report
  --strings-min N minimum length for extracted strings (default 4)
  --max-bytes N   cap how many bytes are read for analysis
  --no-color      disable colour output
`)
}
