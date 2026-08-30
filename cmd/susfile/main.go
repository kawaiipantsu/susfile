// Command susfile is a CLI file-forensics visualiser: it reads one file and
// shows what it is — magic header, MIME/type, hashes, executable structure —
// with a defrag-screen "file map" of classified byte regions as the centrepiece.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/kawaiipantsu/susfile/internal/version"
)

const usageText = `susfile — CLI file-forensics visualiser

Usage:
  susfile [flags] <file>
  susfile version

Flags (planned):
  --no-tui         plain text report instead of the TUI
  --json           machine-readable JSON report
  --strings-min N  minimum length for extracted strings (default 4)
  --max-bytes N    cap how many bytes are read for analysis
  --no-color       disable colour output
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 {
		switch args[0] {
		case "version", "--version", "-V":
			fmt.Fprintln(stdout, version.String())
			return 0
		case "help", "--help", "-h":
			fmt.Fprint(stdout, usageText)
			return 0
		}
	}

	// Full analysis and the TUI land in feature/core-analysis and feature/tui.
	fmt.Fprintln(stderr, "susfile: analysis is not wired up yet on this build.")
	fmt.Fprintln(stderr, "Try: susfile version")
	return 2
}
