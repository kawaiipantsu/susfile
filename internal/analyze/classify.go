package analyze

// Class is the content category assigned to one micro-block. It is what the
// file map draws as a glyph.
type Class uint8

// The numeric order is deliberate: higher = "more interesting". Downsample
// breaks ties toward the higher value, so a packer boundary (code next to
// encrypted) shows the encrypted glyph rather than being averaged away.
const (
	ClassEmpty      Class = iota // no data (block lies beyond EOF)
	ClassNull                    // 0x00 padding
	ClassData                    // structured binary data, none of the below
	ClassRepetitive              // very low entropy, few distinct bytes, non-zero
	ClassHeader                  // structured header / metadata
	ClassText                    // printable text
	ClassMedia                   // media / already-compressed image data
	ClassStrings                 // NUL-separated short strings (string table)
	ClassSource                  // source code / function-dense text
	ClassCode                    // machine code
	ClassCompressed              // generic compressed
	ClassEncrypted               // encrypted / near-uniform random
)

var classGlyph = [...]rune{
	ClassEmpty:      ' ',
	ClassNull:       '·',
	ClassText:       'P',
	ClassSource:     'F',
	ClassStrings:    'S',
	ClassCode:       'C',
	ClassMedia:      'M',
	ClassCompressed: 'Z',
	ClassEncrypted:  'E',
	ClassHeader:     'H',
	ClassData:       'D',
	ClassRepetitive: 'R',
}

var className = [...]string{
	ClassEmpty:      "empty",
	ClassNull:       "null/pad",
	ClassText:       "text",
	ClassSource:     "source",
	ClassStrings:    "strings",
	ClassCode:       "code",
	ClassMedia:      "media",
	ClassCompressed: "compressed",
	ClassEncrypted:  "encrypted",
	ClassHeader:     "header",
	ClassData:       "data",
	ClassRepetitive: "repetitive",
}

// Glyph is the single character the file map prints for this class.
func (c Class) Glyph() rune {
	if int(c) < len(classGlyph) {
		return classGlyph[c]
	}
	return '?'
}

// Name is a short lowercase label.
func (c Class) Name() string {
	if int(c) < len(className) {
		return className[c]
	}
	return "unknown"
}

// String implements fmt.Stringer and is what JSON encodes.
func (c Class) String() string { return c.Name() }

// MarshalText lets encoding/json write the class name instead of a number.
func (c Class) MarshalText() ([]byte, error) { return []byte(c.Name()), nil }

// Classes is every class in map order, for building legends.
func Classes() []Class {
	return []Class{
		ClassNull, ClassText, ClassSource, ClassStrings, ClassCode, ClassMedia,
		ClassCompressed, ClassEncrypted, ClassHeader, ClassData, ClassRepetitive,
	}
}

// classifyBlock decides a Class from a block's bytes, its pre-computed
// summary, whether it sits at the very start of the file, and the name of the
// executable section it falls in ("" if none/unknown).
func classifyBlock(d []byte, m *MicroBlock, atStart bool, section string) Class {
	if len(d) == 0 {
		return ClassEmpty
	}

	// Overwhelmingly zero → padding.
	if m.NUL >= 0.92 {
		return ClassNull
	}

	// A handful of distinct byte values and almost no entropy → RLE-friendly
	// filler that is not plain zeros.
	if m.Distinct <= 6 && m.Entropy < 1.6 {
		return ClassRepetitive
	}

	printable := float64(m.Printable)

	// Mostly text.
	if printable >= 0.85 {
		// Interspersed NULs with short printable runs → a string table.
		if m.NUL >= 0.04 && m.NUL <= 0.5 && meanRunLen(d) < 24 {
			return ClassStrings
		}
		if looksLikeSource(d) {
			return ClassSource
		}
		return ClassText
	}

	// Partly printable with regular NUL separators → still a string table
	// (common in .rodata / .dynstr).
	if printable >= 0.45 && m.NUL >= 0.08 && meanRunLen(d) < 24 {
		return ClassStrings
	}

	// Entropy relative to the most this sample size could reach. This keeps the
	// thresholds meaningful whether the window is 4 KiB or a whole small file.
	ratio := m.Entropy / maxEntropyFor(len(d))

	// Executable section hints.
	switch sectionKind(section) {
	case sectText:
		if ratio >= 0.62 && ratio < 0.995 {
			return ClassCode
		}
	case sectRodata:
		if printable >= 0.3 {
			return ClassStrings
		}
	}

	// Entropy ladder for opaque data, normalised to the sample size. The
	// encrypted/compressed split is only trustworthy over a large sample, so
	// ClassEncrypted needs both a near-perfect ratio and a full byte set —
	// which in practice only happens for big files. Smaller high-entropy
	// regions fall to ClassCompressed and the verdict adds the nuance.
	switch {
	case ratio >= 0.9985 && m.Distinct >= 254:
		return ClassEncrypted
	case ratio >= 0.965:
		return ClassCompressed
	case ratio >= 0.90:
		return ClassMedia
	case ratio >= 0.55 && !atStart:
		return ClassCode // non-printable, mid-entropy: most likely code
	}

	if atStart {
		return ClassHeader
	}
	return ClassData
}

// looksLikeSource is a cheap test for "this printable block is code, not prose":
// enough structural punctuation, real line breaks, and a plausible line length.
func looksLikeSource(d []byte) bool {
	if len(d) < 48 {
		return false
	}
	var punct, nl, letters int
	for _, c := range d {
		switch c {
		case '{', '}', '(', ')', ';', '=', '<', '>', '[', ']':
			punct++
		case '\n':
			nl++
		default:
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				letters++
			}
		}
	}
	if nl < 2 {
		return false
	}
	density := float64(punct) / float64(len(d))
	avgLine := float64(len(d)) / float64(nl+1)
	return density >= 0.02 && avgLine <= 200 && letters >= len(d)/5
}

// meanRunLen is the average length of a maximal run of printable bytes,
// treating any non-printable byte as a separator. Short runs indicate a table
// of small strings rather than flowing text.
func meanRunLen(d []byte) float64 {
	var runs, inRun, curLen, totalLen int
	for _, c := range d {
		if isPrintable(c) && c != '\n' {
			curLen++
			inRun = 1
		} else if inRun == 1 {
			runs++
			totalLen += curLen
			curLen = 0
			inRun = 0
		}
	}
	if inRun == 1 {
		runs++
		totalLen += curLen
	}
	if runs == 0 {
		return 0
	}
	return float64(totalLen) / float64(runs)
}

type sectKind int

const (
	sectOther sectKind = iota
	sectText
	sectRodata
)

func sectionKind(name string) sectKind {
	switch name {
	case ".text", "__text", "text", ".init", ".fini", ".plt", "CODE":
		return sectText
	case ".rodata", ".rdata", "__cstring", "__const", ".dynstr", ".strtab", "DATA":
		return sectRodata
	}
	return sectOther
}
