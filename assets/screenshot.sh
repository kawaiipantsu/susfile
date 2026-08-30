#!/bin/sh
# Regenerate the TUI screenshots in assets/screenshots/.
#
# Needs `freeze` on PATH:  go install github.com/charmbracelet/freeze@latest
#
# framegen prints one rendered View() frame (no pseudo-terminal needed);
# freeze turns the ANSI into an SVG.
set -eu

cd "$(dirname "$0")/.."
command -v freeze >/dev/null 2>&1 || {
	echo "screenshot.sh: need 'freeze' (go install github.com/charmbracelet/freeze@latest)" >&2
	exit 1
}

FG="$(mktemp -d)/framegen"
go build -o "$FG" ./assets/framegen

shot() { # <view> <file> <out>
	"$FG" -file "$2" -view "$1" -w 100 -h 28 > "$FG.ansi"
	freeze --output "assets/screenshots/$3" "$FG.ansi"
}

shot map     /bin/ls  tui-map.svg
shot hex     /bin/ls  tui-hex.svg
shot strings /bin/tar tui-strings.svg

echo "wrote assets/screenshots/*.svg"
