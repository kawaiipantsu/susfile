// Package analyze is susfile's measurement core. Analyze reads one file and
// returns a single Result describing what it is: magic header, MIME/type,
// hashes, byte statistics, Shannon entropy, a fixed-resolution micro-block scan
// with a content class per block, executable structure, extracted strings and a
// one-glance verdict.
//
// The package is deliberately self-contained: it never writes, never opens a
// network connection, and imports nothing from susfile's UI or reporting code.
// A Go error is returned only when the file cannot be read at all; every
// recoverable problem — truncation, an unparseable executable, an unknown
// magic — produces a populated Result instead.
package analyze

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"time"
)

// Tunable limits. Zero values in Options select the defaults.
const (
	// DefaultMaxBytes bounds the in-memory buffer used by the detail passes
	// (histogram, micro-blocks, strings, magic, structure). Hashing and the
	// global-entropy pass always stream and are not bounded by this.
	DefaultMaxBytes = 256 << 20

	// DefaultStringsMin is the shortest run reported by string extraction.
	DefaultStringsMin = 4

	// DefaultMaxStrings caps how many extracted strings are retained; the
	// total count is still reported.
	DefaultMaxStrings = 4096

	// MicroBlockCount is the fixed number of blocks in every scan, regardless
	// of file size. Renderers downsample this to whatever grid they draw.
	MicroBlockCount = 4096

	streamChunk  = 1 << 20 // 1 MiB streaming read size
	sampleWindow = 8 << 10 // bytes read per block when sampling a huge file
	headHexBytes = 16      // leading bytes shown as hex in the info box
)

// Options controls an Analyze run. The zero value is valid and uses the
// Default* constants above.
type Options struct {
	MaxBytes     int64 // detail-pass buffer cap; <= 0 selects DefaultMaxBytes
	StringsMin   int   // <= 0 selects DefaultStringsMin
	MaxStrings   int   // <= 0 selects DefaultMaxStrings
	AllowSpecial bool  // permit non-regular files (devices, FIFOs, ...)
}

func (o Options) withDefaults() Options {
	if o.MaxBytes <= 0 {
		o.MaxBytes = DefaultMaxBytes
	}
	if o.StringsMin <= 0 {
		o.StringsMin = DefaultStringsMin
	}
	if o.MaxStrings <= 0 {
		o.MaxStrings = DefaultMaxStrings
	}
	return o
}

// Stage identifies a phase of analysis for progress reporting.
type Stage string

// The phases Analyze moves through, in order.
const (
	StageOpen      Stage = "opening"
	StageHash      Stage = "hashing"
	StageHistogram Stage = "histogram"
	StageStructure Stage = "structure"
	StageBlocks    Stage = "blocks"
	StageStrings   Stage = "strings"
	StageDone      Stage = "done"
)

// ProgressFunc receives a stage and a 0..1 fraction. It may be nil.
type ProgressFunc func(Stage, float64)

func (p ProgressFunc) report(s Stage, frac float64) {
	if p != nil {
		p(s, frac)
	}
}

// Stats holds the ent-style arithmetic summaries of the byte stream.
type Stats struct {
	Mean          float64 // arithmetic mean of byte values (127.5 for random)
	ChiSquare     float64 // chi-square over the 256 byte values
	ChiSquareProb float64 // approx. p-value (normal approximation, df=255)
	MonteCarloPi  float64 // pi estimated from byte-triples as coordinates
	MonteCarloErr float64 // fractional error of MonteCarloPi vs math.Pi
	SerialCorr    float64 // lag-1 serial correlation coefficient
}

// MagicMatch is the result of the signature-table lookup.
type MagicMatch struct {
	Matched bool   `json:"matched"`
	Label   string `json:"label,omitempty"` // "PNG image"
	MIME    string `json:"mime,omitempty"`
	Ext     string `json:"ext,omitempty"`
	Offset  int    `json:"offset,omitempty"` // where the signature sits
	HeadHex string `json:"head_hex"`         // leading bytes, "89 50 4e 47 ..."
}

// Section is one section/segment of an executable, with its own entropy.
type Section struct {
	Name    string  `json:"name"`
	Offset  int64   `json:"offset"`
	Size    int64   `json:"size"`
	Entropy float64 `json:"entropy"`
	Flags   string  `json:"flags,omitempty"` // "rx", "rw", "r", ...
}

// BinFmt summarises an ELF, PE or Mach-O container.
type BinFmt struct {
	Format     string    `json:"format"` // "ELF" | "PE" | "Mach-O"
	Bits       int       `json:"bits"`   // 32 | 64
	Endian     string    `json:"endian,omitempty"`
	Machine    string    `json:"machine,omitempty"`
	Type       string    `json:"type,omitempty"` // "executable", "shared object", ...
	Interp     string    `json:"interp,omitempty"`
	Stripped   bool      `json:"stripped"`
	NumImports int       `json:"num_imports"`
	Sections   []Section `json:"sections,omitempty"`
}

// StringHit is one extracted printable run.
type StringHit struct {
	Offset   int64  `json:"offset"`
	Encoding string `json:"encoding"` // "ascii" | "utf16le"
	Text     string `json:"text"`
}

// Verdict is the whole-file summary.
type Verdict struct {
	Kaomoji string `json:"kaomoji"`
	Summary string `json:"summary"`
}

// Result is everything Analyze learned about a file.
type Result struct {
	Path      string      `json:"path"`
	Name      string      `json:"name"`
	Size      int64       `json:"size"`
	Mode      fs.FileMode `json:"-"`
	ModeText  string      `json:"mode,omitempty"`
	ModTime   time.Time   `json:"modtime,omitempty"`
	FromStdin bool        `json:"from_stdin"`

	// Truncated is set when the file is larger than Options.MaxBytes, so the
	// detail passes saw only a prefix. Sampled is set when the micro-block
	// scan seeked across the whole file instead of scanning a buffer.
	Truncated bool `json:"truncated"`
	Sampled   bool `json:"sampled"`

	Magic    MagicMatch `json:"magic"`
	MIME     string     `json:"mime,omitempty"`
	MIMETree []string   `json:"mime_tree,omitempty"`
	ExtGuess string     `json:"ext_guess,omitempty"`

	MD5    string `json:"md5"`
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
	CRC32  string `json:"crc32"`

	Histogram     [256]uint64 `json:"-"`
	HistogramList []uint64    `json:"histogram"`
	Counted       int64       `json:"counted_bytes"` // bytes fed to the detail passes
	PrintableFrac float64     `json:"printable_frac"`
	NULFrac       float64     `json:"nul_frac"`
	HighFrac      float64     `json:"high_frac"`
	WSFrac        float64     `json:"ws_frac"`
	DistinctBytes int         `json:"distinct_bytes"`

	GlobalEntropy float64 `json:"global_entropy"` // bits/byte over the whole file
	Stats         Stats   `json:"stats"`

	MicroBlocks []MicroBlock   `json:"microblocks"`
	ClassCounts map[string]int `json:"class_counts"`

	BinFmt *BinFmt `json:"binfmt,omitempty"`

	Strings      []StringHit `json:"strings"`
	StringsTotal int         `json:"strings_total"`
	StringsMin   int         `json:"strings_min"`

	Verdict  Verdict  `json:"verdict"`
	Warnings []string `json:"warnings,omitempty"`
}

func (r *Result) warn(format string, a ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
}

// ErrNotReadable wraps the underlying cause when the input cannot be read at
// all (missing, a directory, a non-regular file without AllowSpecial, ...).
var ErrNotReadable = errors.New("input is not readable")

// Analyze reads path (or standard input when path is "-") and returns a Result.
// progress may be nil.
func Analyze(ctx context.Context, path string, opt Options, progress ProgressFunc) (*Result, error) {
	opt = opt.withDefaults()
	progress.report(StageOpen, 0)

	src, err := openInput(path, opt)
	if err != nil {
		return nil, err
	}
	defer func() { _ = src.Close() }()

	r := &Result{
		Path:       src.displayPath,
		Name:       src.name,
		Size:       src.size,
		Mode:       src.mode,
		ModTime:    src.modTime,
		FromStdin:  src.fromStdin,
		StringsMin: opt.StringsMin,
	}
	if src.mode != 0 {
		r.ModeText = src.mode.String()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Pass 1: stream the whole file for hashes and global entropy.
	if err := streamPass(ctx, src, r, progress); err != nil {
		return nil, err
	}

	// Buffer a bounded prefix for the detail passes.
	buf, truncated, err := src.bufferPrefix(opt.MaxBytes)
	if err != nil {
		return nil, err
	}
	r.Truncated = truncated
	r.Counted = int64(len(buf))

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	histogramPass(buf, r)
	progress.report(StageHistogram, 1)

	statsPass(buf, r)

	progress.report(StageStructure, 0)
	structurePass(buf, r)
	progress.report(StageStructure, 1)

	progress.report(StageBlocks, 0)
	blocksPass(src, buf, r, progress)
	progress.report(StageBlocks, 1)

	progress.report(StageStrings, 0)
	stringsPass(buf, r, opt)
	progress.report(StageStrings, 1)

	r.Verdict = decideVerdict(r)
	r.HistogramList = r.Histogram[:]

	progress.report(StageDone, 1)
	return r, nil
}

// streamPass reads the entire input in chunks, feeding the hashers and a
// running frequency table for the whole-file entropy figure.
func streamPass(ctx context.Context, src *input, r *Result, progress ProgressFunc) error {
	h := newHashSet()
	var freq [256]uint64
	var total int64

	rd := src.reader()
	chunk := make([]byte, streamChunk)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := rd.Read(chunk)
		if n > 0 {
			b := chunk[:n]
			h.Write(b)
			for _, c := range b {
				freq[c]++
			}
			total += int64(n)
			if src.size > 0 {
				progress.report(StageHash, float64(total)/float64(src.size))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("%w: %v", ErrNotReadable, err)
		}
	}

	h.sum(r)
	r.GlobalEntropy = shannonBits(freq[:], uint64(total))
	if !src.fromStdin && total != src.size {
		// The file changed size under us, or Stat lied. Report what we read.
		r.warn("read %d bytes but Stat reported %d", total, src.size)
		r.Size = total
	}
	if src.fromStdin {
		r.Size = total
	}
	progress.report(StageHash, 1)
	return nil
}

func baseName(path string) string {
	if path == "-" {
		return "<stdin>"
	}
	return filepath.Base(path)
}
