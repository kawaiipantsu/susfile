package analyze

import "strings"

// Kaomoji verdicts. These are the whole-file summary shown in the info box.
const (
	kaoText      = "^_^"
	kaoBinary    = "•_•"
	kaoHighEnt   = "ಠ_ಠ"
	kaoMalformed = `¯\_(ツ)_/¯`
	kaoEmpty     = "-_-"
)

// decideVerdict summarises the whole Result. It is a lead, not a conclusion.
func decideVerdict(r *Result) Verdict {
	if r.Size == 0 {
		return Verdict{kaoEmpty, "empty file"}
	}

	// Magic claims an executable container but no parser accepted it.
	if r.BinFmt == nil && magicClaimsExecutable(r.Magic.Label) {
		return Verdict{kaoMalformed, "executable header, but the body did not parse"}
	}
	if hasParseWarning(r.Warnings) {
		return Verdict{kaoMalformed, "structure present but could not be fully parsed"}
	}

	// Dominant block classes.
	src := r.ClassCounts["source"] + r.ClassCounts["text"]
	enc := r.ClassCounts["encrypted"]
	comp := r.ClassCounts["compressed"]
	total := len(r.MicroBlocks)
	nonEmpty := total - r.ClassCounts["empty"] - r.ClassCounts["null/pad"]
	if nonEmpty < 1 {
		nonEmpty = 1
	}

	highEntropy := r.GlobalEntropy >= 7.90 && r.DistinctBytes >= 250
	container := magicIsContainer(r.Magic.Label) || mimeIsCompressed(r.MIME)

	if highEntropy && !container && float64(enc+comp)/float64(nonEmpty) >= 0.6 {
		return Verdict{kaoHighEnt, "high entropy throughout — likely packed, compressed or encrypted"}
	}

	if r.BinFmt != nil {
		desc := strings.ToLower(r.BinFmt.Format)
		if r.BinFmt.Type != "" {
			desc += " " + r.BinFmt.Type
		}
		if hasPackedShape(r) {
			return Verdict{kaoHighEnt, desc + " with a high-entropy body — possibly packed"}
		}
		return Verdict{kaoBinary, "ordinary " + desc}
	}

	if float64(src)/float64(nonEmpty) >= 0.6 && r.PrintableFrac >= 0.85 {
		return Verdict{kaoText, "clean text / source"}
	}

	if highEntropy && !container {
		return Verdict{kaoHighEnt, "high entropy — likely packed, compressed or encrypted"}
	}
	if container {
		return Verdict{kaoBinary, "compressed / media container (high entropy is expected)"}
	}
	if r.PrintableFrac >= 0.85 {
		return Verdict{kaoText, "mostly text"}
	}
	return Verdict{kaoBinary, "binary data"}
}

func magicClaimsExecutable(label string) bool {
	l := strings.ToLower(label)
	for _, k := range []string{"elf binary", "dos/pe", "mach-o"} {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}

func hasParseWarning(ws []string) bool {
	for _, w := range ws {
		if strings.Contains(w, "unparseable") || strings.Contains(w, "panicked") {
			return true
		}
	}
	return false
}

func mimeIsCompressed(mime string) bool {
	switch mime {
	case "application/zip", "application/gzip", "application/x-bzip2", "application/x-xz",
		"application/zstd", "application/x-7z-compressed", "application/vnd.rar",
		"application/x-tar", "image/png", "image/jpeg", "image/webp", "audio/flac",
		"video/mp4", "application/pdf":
		return true
	}
	return false
}

// hasPackedShape looks for the classic packer signature: a small number of
// sections, most of the file's mass in one very-high-entropy section.
func hasPackedShape(r *Result) bool {
	if r.BinFmt == nil || len(r.BinFmt.Sections) == 0 {
		return false
	}
	var big Section
	for _, s := range r.BinFmt.Sections {
		if s.Size > big.Size {
			big = s
		}
	}
	return big.Entropy >= 7.8 && big.Size >= r.Size/2
}
