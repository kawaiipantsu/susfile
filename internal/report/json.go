package report

import (
	"encoding/json"
	"io"

	"github.com/kawaiipantsu/susfile/internal/analyze"
)

// jsonDoc is the stable wire shape. It embeds the analyze.Result (whose fields
// carry their own json tags) and adds a rendered map plus the legend so a
// consumer does not have to know susfile's glyph set.
type jsonDoc struct {
	Schema  string           `json:"schema"`
	Tool    string           `json:"tool"`
	Result  *analyze.Result  `json:"result"`
	MapRows []string         `json:"map"`
	Legend  []jsonLegendItem `json:"legend"`
}

type jsonLegendItem struct {
	Glyph string `json:"glyph"`
	Name  string `json:"name"`
}

// JSONSchema is the identifier for the current JSON output shape. Bump the
// major only on a breaking change.
const JSONSchema = "susfile.report/v1"

// JSON writes r to w as indented JSON following JSONSchema.
func JSON(w io.Writer, r *analyze.Result) error {
	doc := jsonDoc{
		Schema:  JSONSchema,
		Tool:    "susfile",
		Result:  r,
		MapRows: mapRows(r.MicroBlocks, 64, 16),
		Legend:  legendItems(),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}

func mapRows(mb []analyze.MicroBlock, wCells, hRows int) []string {
	cells := analyze.Downsample(mb, wCells*hRows)
	rows := make([]string, 0, hRows)
	for row := 0; row < hRows; row++ {
		r := make([]rune, 0, wCells)
		for col := 0; col < wCells; col++ {
			idx := row*wCells + col
			if idx >= len(cells) {
				break
			}
			r = append(r, cells[idx].Class.Glyph())
		}
		if len(r) == 0 {
			break
		}
		rows = append(rows, string(r))
	}
	return rows
}

func legendItems() []jsonLegendItem {
	out := make([]jsonLegendItem, 0)
	for _, e := range Legend() {
		out = append(out, jsonLegendItem{Glyph: string(e.Glyph), Name: e.Name})
	}
	return out
}
