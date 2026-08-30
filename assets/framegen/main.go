// Command framegen prints a single rendered susfile TUI frame to stdout. It
// exists so assets/screenshot.sh can capture a real frame without allocating a
// pseudo-terminal. Run it with CLICOLOR_FORCE=1 for a coloured frame. It is a
// dev tool and is not part of the susfile binary.
//
//	CLICOLOR_FORCE=1 go run ./assets/framegen -file /bin/ls -view map > frame.ansi
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/kawaiipantsu/susfile/internal/analyze"
	"github.com/kawaiipantsu/susfile/internal/tui"
)

func main() {
	file := flag.String("file", "/bin/ls", "file to analyse")
	vname := flag.String("view", "map", "map|entropy|histogram|hex|strings")
	w := flag.Int("w", 112, "frame width")
	h := flag.Int("h", 34, "frame height")
	color := flag.Bool("color", true, "render with colour")
	flag.Parse()

	if *color {
		// stdout is a pipe here, so force a rich profile for the capture.
		lipgloss.SetColorProfile(termenv.ANSI256)
	}

	out, err := tui.Frame(*file, analyze.Options{}, *w, *h, *vname, *color)
	if err != nil {
		fmt.Fprintln(os.Stderr, "framegen:", err)
		os.Exit(1)
	}
	fmt.Print(out)
}
