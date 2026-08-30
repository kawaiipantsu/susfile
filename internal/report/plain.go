package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/kawaiipantsu/susfile/internal/analyze"
)

var sparkRamp = []rune("▁▂▃▄▅▆▇█")

// Plain writes a human-readable report for r to w.
func Plain(w io.Writer, r *analyze.Result, opt Options) {
	opt = opt.withDefaults()
	p := &painter{w: w, color: opt.Color}

	p.header("susfile — %s", r.Name)
	p.kv("Path", r.Path)
	p.kvf("Size", "%s  (%d bytes)", humanBytes(r.Size), r.Size)
	if !r.ModTime.IsZero() {
		p.kv("Modified", r.ModTime.Format("2006-01-02 15:04:05"))
	}
	if r.ModeText != "" {
		p.kv("Mode", r.ModeText)
	}
	p.kv("Type", typeLine(r))
	p.kv("Magic", r.Magic.HeadHex)
	if r.MIME != "" {
		p.kv("MIME", r.MIME)
	}
	if r.ExtGuess != "" {
		p.kv("Ext guess", "."+r.ExtGuess)
	}

	p.blank()
	p.kvf("Entropy", "%.2f bits/byte  %s", r.GlobalEntropy, entropyBar(r.GlobalEntropy, 24))
	p.kvf("Printable", "%.1f%%   NUL %.1f%%   high-bit %.1f%%", 100*r.PrintableFrac, 100*r.NULFrac, 100*r.HighFrac)
	p.kvf("Distinct", "%d / 256 byte values", r.DistinctBytes)
	p.kvf("Chi-square", "%.1f  (p ≈ %.4f)", r.Stats.ChiSquare, r.Stats.ChiSquareProb)
	p.kvf("Mean", "%.2f  (127.5 = random)", r.Stats.Mean)
	p.kvf("Monte-Carlo π", "%.6f  (error %.4f%%)", r.Stats.MonteCarloPi, 100*r.Stats.MonteCarloErr)
	p.kvf("Serial corr.", "%.5f", r.Stats.SerialCorr)

	if r.BinFmt != nil {
		p.blank()
		bf := r.BinFmt
		p.kvf(bf.Format, "%d-bit %s, %s%s", bf.Bits, bf.Machine, bf.Type, strippedNote(bf.Stripped))
		if bf.Interp != "" {
			p.kv("Interpreter", bf.Interp)
		}
		p.kvf("Sections", "%d   Imports %d", len(bf.Sections), bf.NumImports)
		for _, s := range topSections(bf.Sections, 6) {
			p.kvf("  "+s.Name, "%8d bytes   entropy %.2f  %s", s.Size, s.Entropy, s.Flags)
		}
	}

	p.blank()
	p.kv("MD5", r.MD5)
	p.kv("SHA-1", r.SHA1)
	p.kv("SHA-256", r.SHA256)
	p.kv("CRC-32", r.CRC32)

	p.blank()
	p.kvf("Strings", "%d found (min length %d), showing offsets in the map", r.StringsTotal, r.StringsMin)

	p.blank()
	p.line("Entropy  %s", sparkline(r.MicroBlocks, 64))
	p.blank()
	p.line("File map  (%d×%d cells, %s):", opt.MapW, opt.MapH, "glyph = class, rows top→bottom = start→end")
	writeClassMap(p, r.MicroBlocks, opt.MapW, opt.MapH)
	p.blank()
	writeLegend(p)

	p.blank()
	p.kvf("Verdict", "%s  %s", r.Verdict.Kaomoji, r.Verdict.Summary)
	if len(r.Warnings) > 0 {
		p.blank()
		for _, ww := range r.Warnings {
			p.line("warning: %s", ww)
		}
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

func strippedNote(stripped bool) string {
	if stripped {
		return ", stripped"
	}
	return ", with symbols"
}

func topSections(secs []analyze.Section, n int) []analyze.Section {
	if len(secs) <= n {
		return secs
	}
	cp := make([]analyze.Section, len(secs))
	copy(cp, secs)
	// simple partial selection by size
	for i := 0; i < n; i++ {
		mx := i
		for j := i + 1; j < len(cp); j++ {
			if cp[j].Size > cp[mx].Size {
				mx = j
			}
		}
		cp[i], cp[mx] = cp[mx], cp[i]
	}
	return cp[:n]
}

func writeClassMap(p *painter, mb []analyze.MicroBlock, wCells, hRows int) {
	cells := analyze.Downsample(mb, wCells*hRows)
	if len(cells) == 0 {
		p.line("(no data)")
		return
	}
	var sb strings.Builder
	for row := 0; row < hRows; row++ {
		sb.Reset()
		for col := 0; col < wCells; col++ {
			idx := row*wCells + col
			if idx >= len(cells) {
				break
			}
			g := cells[idx].Class.Glyph()
			if p.color {
				sb.WriteString(colorFor(cells[idx].Entropy))
				sb.WriteRune(g)
				sb.WriteString("\x1b[0m")
			} else {
				sb.WriteRune(g)
			}
		}
		p.line("  %s", sb.String())
	}
}

func writeLegend(p *painter) {
	var sb strings.Builder
	for i, e := range Legend() {
		if i > 0 {
			sb.WriteString("   ")
		}
		fmt.Fprintf(&sb, "%c %s", e.Glyph, e.Name)
	}
	p.line("Legend  %s", sb.String())
	p.line("Shade   low ░ ▒ ▓ █ high  (colour = entropy)")
}

func sparkline(mb []analyze.MicroBlock, n int) string {
	cells := analyze.Downsample(mb, n)
	if len(cells) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, c := range cells {
		lvl := int(c.Entropy / 8 * float64(len(sparkRamp)-1))
		if lvl < 0 {
			lvl = 0
		}
		if lvl >= len(sparkRamp) {
			lvl = len(sparkRamp) - 1
		}
		sb.WriteRune(sparkRamp[lvl])
	}
	return sb.String()
}

func entropyBar(e float64, width int) string {
	fill := int(e / 8 * float64(width))
	if fill < 0 {
		fill = 0
	}
	if fill > width {
		fill = width
	}
	return strings.Repeat("█", fill) + strings.Repeat("░", width-fill)
}

// colorFor maps entropy 0..8 to a 256-colour ANSI SGR prefix (blue→red).
func colorFor(e float64) string {
	ramp := []int{27, 39, 45, 42, 46, 190, 226, 214, 208, 196}
	i := int(e / 8 * float64(len(ramp)-1))
	if i < 0 {
		i = 0
	}
	if i >= len(ramp) {
		i = len(ramp) - 1
	}
	return fmt.Sprintf("\x1b[38;5;%dm", ramp[i])
}

// --- tiny writer helper -----------------------------------------------------

type painter struct {
	w     io.Writer
	color bool
}

func (p *painter) line(format string, a ...any) { fmt.Fprintf(p.w, format+"\n", a...) }
func (p *painter) blank()                       { fmt.Fprintln(p.w) }

func (p *painter) header(format string, a ...any) {
	s := fmt.Sprintf(format, a...)
	fmt.Fprintf(p.w, "%s\n%s\n", s, strings.Repeat("─", len([]rune(s))))
}

func (p *painter) kv(key, val string) {
	fmt.Fprintf(p.w, "%-14s %s\n", key, val)
}

func (p *painter) kvf(key, format string, a ...any) {
	fmt.Fprintf(p.w, "%-14s %s\n", key, fmt.Sprintf(format, a...))
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
