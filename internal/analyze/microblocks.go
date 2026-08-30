package analyze

import (
	"math"
	"sort"
)

// classWindow is the minimum number of bytes each block's entropy and byte
// ratios are measured over. A block narrower than this is widened symmetrically
// (clamped to the buffer) so the entropy figure is statistically meaningful
// even when a small file produces sub-kilobyte blocks.
const classWindow = 4096

// blockDivisor sets how many bytes of file map to one block, before the
// MicroBlockCount cap. A file smaller than MicroBlockCount*blockDivisor gets
// proportionally fewer, wider blocks.
const blockDivisor = 16

// MicroBlock is one slice of the file in the fixed-resolution scan. Every scan
// has exactly MicroBlockCount of these regardless of file size; renderers
// aggregate them down to whatever grid they draw.
type MicroBlock struct {
	Offset    int64   `json:"offset"`
	Len       int32   `json:"len"`
	Entropy   float64 `json:"entropy"` // 0..8
	Printable float32 `json:"printable"`
	NUL       float32 `json:"nul"`
	High      float32 `json:"high"`
	WS        float32 `json:"ws"`
	Distinct  uint16  `json:"distinct"`
	TopByte   byte    `json:"top_byte"`
	Section   string  `json:"section,omitempty"`
	Class     Class   `json:"class"`
}

// blocksPass builds the MicroBlock slice. For a file that fits in the detail
// buffer it scans the buffer directly; for a larger file it seeks and samples
// a window from each block position. Executable section ranges (already in
// r.BinFmt) sharpen the classification.
func blocksPass(src *input, buf []byte, r *Result, progress ProgressFunc) {
	var secs []Section
	if r.BinFmt != nil {
		secs = r.BinFmt.Sections
	}

	total := r.Size
	if total <= 0 {
		total = int64(len(buf))
	}

	nblocks := int(total / blockDivisor)
	if nblocks < 1 {
		nblocks = 1
	}
	if nblocks > MicroBlockCount {
		nblocks = MicroBlockCount
	}

	blocks := make([]MicroBlock, nblocks)
	counts := map[string]int{}

	sampled := r.Truncated && src.f != nil
	r.Sampled = sampled

	var sampleBuf []byte
	if sampled {
		sampleBuf = make([]byte, sampleWindow)
	}

	for i := 0; i < nblocks; i++ {
		start := int64(i) * total / int64(nblocks)
		end := int64(i+1) * total / int64(nblocks)
		blk := &blocks[i]
		blk.Offset = start
		blk.Len = int32(end - start)

		// Widen to at least classWindow bytes for the measurement.
		wLo, wHi := widen(start, end, total)

		var d []byte
		switch {
		case wHi <= wLo:
			// Empty file.
		case sampled:
			n := wHi - wLo
			if n > int64(len(sampleBuf)) {
				n = int64(len(sampleBuf))
			}
			got, _ := src.readAt(sampleBuf[:n], wLo)
			d = sampleBuf[:got]
		default:
			lo, hi := clamp64(wLo, int64(len(buf))), clamp64(wHi, int64(len(buf)))
			d = buf[lo:hi]
		}

		summariseBlock(d, blk)
		blk.Section = sectionAt(secs, (start+end)/2)
		blk.Class = classifyBlock(d, blk, i == 0, blk.Section)
		counts[blk.Class.Name()]++

		if progress != nil && i%256 == 0 {
			progress.report(StageBlocks, float64(i)/float64(nblocks))
		}
	}

	r.MicroBlocks = blocks
	r.ClassCounts = counts
}

// widen expands [start,end) to at least classWindow bytes, centred, clamped to
// [0,total).
func widen(start, end, total int64) (lo, hi int64) {
	const want = classWindow
	if end-start >= want {
		return start, end
	}
	pad := (want - (end - start)) / 2
	lo, hi = start-pad, end+pad
	if lo < 0 {
		hi -= lo
		lo = 0
	}
	if hi > total {
		lo -= hi - total
		hi = total
	}
	if lo < 0 {
		lo = 0
	}
	return lo, hi
}

func clamp64(v, max int64) int64 {
	if v < 0 {
		return 0
	}
	if v > max {
		return max
	}
	return v
}

// maxEntropyFor returns the highest Shannon entropy a sample of n bytes can
// reach: log2(min(n, 256)).
func maxEntropyFor(n int) float64 {
	k := n
	if k > 256 {
		k = 256
	}
	if k < 2 {
		return 1
	}
	return math.Log2(float64(k))
}

// summariseBlock fills the measured fields of blk from d.
func summariseBlock(d []byte, blk *MicroBlock) {
	if len(d) == 0 {
		return
	}
	var f [256]uint64
	for _, c := range d {
		f[c]++
	}
	n := float32(len(d))

	var printable, nul, high, ws int
	var distinct int
	var topByte byte
	var topCount uint64
	for v, c := range f {
		if c == 0 {
			continue
		}
		distinct++
		if c > topCount {
			topCount, topByte = c, byte(v)
		}
		b := byte(v)
		switch {
		case b == 0x00:
			nul += int(c)
		case b >= 0x80:
			high += int(c)
		}
		if isPrintable(b) {
			printable += int(c)
		}
		if isSpace(b) {
			ws += int(c)
		}
	}

	blk.Entropy = shannonBits(f[:], uint64(len(d)))
	blk.Printable = float32(printable) / n
	blk.NUL = float32(nul) / n
	blk.High = float32(high) / n
	blk.WS = float32(ws) / n
	blk.Distinct = uint16(distinct)
	blk.TopByte = topByte
}

// Downsample aggregates mb into exactly n cells for a renderer's grid. Each
// output cell takes the most common class of the micro-blocks it covers
// (ties broken by the "more interesting" class), the mean entropy, and the
// span of offsets. n is clamped to [1, len(mb)].
func Downsample(mb []MicroBlock, n int) []MicroBlock {
	if len(mb) == 0 || n < 1 {
		return nil
	}
	if n > len(mb) {
		n = len(mb)
	}
	out := make([]MicroBlock, n)
	for i := range out {
		lo := i * len(mb) / n
		hi := (i + 1) * len(mb) / n
		if hi <= lo {
			hi = lo + 1
		}
		grp := mb[lo:hi]

		var ent, pr, nul, high, ws float64
		var counts [16]int
		var maxLen int32
		cell := &out[i]
		cell.Offset = grp[0].Offset
		for _, g := range grp {
			ent += g.Entropy
			pr += float64(g.Printable)
			nul += float64(g.NUL)
			high += float64(g.High)
			ws += float64(g.WS)
			cell.Len += g.Len
			if g.Len > maxLen {
				maxLen, cell.TopByte, cell.Section = g.Len, g.TopByte, g.Section
			}
			if int(g.Class) < len(counts) {
				counts[g.Class]++
			}
		}
		fn := float64(len(grp))
		cell.Entropy = ent / fn
		cell.Printable = float32(pr / fn)
		cell.NUL = float32(nul / fn)
		cell.High = float32(high / fn)
		cell.WS = float32(ws / fn)
		cell.Class = dominantClass(counts)
	}
	return out
}

// dominantClass returns the most frequent class, breaking ties toward the
// class carrying more signal (higher enum value: code/compressed/encrypted beat
// data/text beat padding).
func dominantClass(counts [16]int) Class {
	best := ClassEmpty
	bestN := -1
	for c := len(counts) - 1; c >= 0; c-- {
		if counts[c] > bestN {
			bestN, best = counts[c], Class(c)
		}
	}
	return best
}

// sectionAt returns the name of the section whose byte range contains off, or
// "" if none. Sections are checked smallest-first so a nested match wins.
func sectionAt(secs []Section, off int64) string {
	if len(secs) == 0 {
		return ""
	}
	// Copy indices sorted by size ascending so the tightest enclosing section
	// is preferred.
	idx := make([]int, 0, len(secs))
	for i := range secs {
		if secs[i].Size > 0 {
			idx = append(idx, i)
		}
	}
	sort.Slice(idx, func(a, b int) bool { return secs[idx[a]].Size < secs[idx[b]].Size })
	for _, i := range idx {
		s := secs[i]
		if off >= s.Offset && off < s.Offset+s.Size {
			return s.Name
		}
	}
	return ""
}
