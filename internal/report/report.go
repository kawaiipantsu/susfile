// Package report renders an analyze.Result for humans (plain text) and for
// machines (JSON). It performs no measurement of its own.
package report

import "github.com/kawaiipantsu/susfile/internal/analyze"

// Options controls the plain-text renderer.
type Options struct {
	Color bool // emit ANSI colour
	MapW  int  // class-map width in cells  (<=0 -> 64)
	MapH  int  // class-map height in rows  (<=0 -> 16)
}

func (o Options) withDefaults() Options {
	if o.MapW <= 0 {
		o.MapW = 64
	}
	if o.MapH <= 0 {
		o.MapH = 16
	}
	return o
}

// LegendEntry pairs a class glyph with its meaning, for the map legend.
type LegendEntry struct {
	Glyph rune
	Name  string
}

// Legend is the ordered list of classes shown under a file map.
func Legend() []LegendEntry {
	cs := analyze.Classes()
	out := make([]LegendEntry, len(cs))
	for i, c := range cs {
		out[i] = LegendEntry{Glyph: c.Glyph(), Name: c.Name()}
	}
	return out
}
