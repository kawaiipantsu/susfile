package tui

import (
	"fmt"
	"strings"

	"github.com/kawaiipantsu/susfile/internal/analyze"
)

// renderInfo draws the top-right label/value box body for w columns and h rows.
func (m Model) renderInfo(w, h int) string {
	t := m.theme
	var rows []string
	add := func(label, value string) {
		rows = append(rows, t.label.Render(fmt.Sprintf("%-10s", label))+" "+t.value.Render(value))
	}

	if m.res == nil {
		if m.hasNoFile() {
			add("Path", "—")
			add("Status", "waiting for a file — browse and press ⏎")
			return fitBox(rows, w, h)
		}
		add("Path", m.path)
		add("Status", m.stageText()+" …")
		return fitBox(rows, w, h)
	}
	r := m.res

	add("Type", clip(typeLine(r), w-12))
	add("Magic", r.Magic.HeadHex)
	if r.MIME != "" {
		add("MIME", clip(r.MIME, w-12))
	}
	add("Size", fmt.Sprintf("%s  (%d B)", humanBytes(r.Size), r.Size))
	add("Entropy", fmt.Sprintf("%.2f  %s", r.GlobalEntropy, bar(r.GlobalEntropy/8, 18)))

	strCount := fmt.Sprintf("%d", r.StringsTotal)
	secCount := "—"
	if r.BinFmt != nil {
		secCount = fmt.Sprintf("%d", len(r.BinFmt.Sections))
	}
	add("Strings", strCount+"   "+t.label.Render("Sections ")+t.value.Render(secCount))
	add("SHA-256", clip(r.SHA256, w-12))

	vk := t.verdict.Render(r.Verdict.Kaomoji)
	add("Verdict", vk+"  "+clip(r.Verdict.Summary, w-24))

	if len(r.Warnings) > 0 {
		rows = append(rows, t.warnText.Render(clip("! "+r.Warnings[0], w-2)))
	}
	return fitBox(rows, w, h)
}

func (m Model) stageText() string {
	switch m.stage {
	case analyze.StageOpen:
		return "opening"
	case analyze.StageHash:
		return "hashing"
	case analyze.StageHistogram:
		return "histogram"
	case analyze.StageStructure:
		return "reading structure"
	case analyze.StageBlocks:
		return "scanning blocks"
	case analyze.StageStrings:
		return "extracting strings"
	case analyze.StageDone:
		return "done"
	default:
		return "starting"
	}
}

func typeLine(r *analyze.Result) string {
	switch {
	case r.BinFmt != nil:
		return fmt.Sprintf("%s %s (%d-bit %s)", r.BinFmt.Format, r.BinFmt.Type, r.BinFmt.Bits, r.BinFmt.Machine)
	case r.Magic.Matched && r.Magic.Label != "":
		return r.Magic.Label
	case r.MIME != "":
		return r.MIME
	default:
		return "unknown / raw data"
	}
}

func fitBox(rows []string, w, h int) string {
	for len(rows) < h {
		rows = append(rows, "")
	}
	if len(rows) > h {
		rows = rows[:h]
	}
	for i := range rows {
		rows[i] = clipVisible(rows[i], w)
	}
	return strings.Join(rows, "\n")
}

func bar(frac float64, width int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	n := int(frac * float64(width))
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

func clip(s string, max int) string {
	if max < 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
