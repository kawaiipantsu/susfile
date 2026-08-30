package analyze

import (
	"bytes"
	"strings"
	"testing"
)

// classifyProbe runs the same measurement the block scan does, over d as its
// own window, then classifies it.
func classifyProbe(d []byte, atStart bool, section string) Class {
	var mb MicroBlock
	summariseBlock(d, &mb)
	return classifyBlock(d, &mb, atStart, section)
}

func TestClassifyBlock(t *testing.T) {
	goSource := []byte(strings.Repeat(
		"func add(a, b int) int {\n\tif a > b {\n\t\treturn a + b\n\t}\n\treturn b - a\n}\n\n", 40))
	prose := []byte(strings.Repeat(
		"The quick brown fox jumps over the lazy dog near the river bank at dawn. ", 60))
	stringTable := bytes.Join([][]byte{
		[]byte("GetProcAddress"), []byte("LoadLibraryA"), []byte("kernel32.dll"),
		[]byte("VirtualAlloc"), []byte("ExitProcess"), []byte("CreateFileW"),
	}, []byte{0})
	stringTable = append(bytes.Repeat(stringTable, 8), 0)

	tests := []struct {
		name    string
		data    []byte
		atStart bool
		section string
		want    Class
	}{
		{"zeros", bytes.Repeat([]byte{0}, 4096), false, "", ClassNull},
		{"ff filler", bytes.Repeat([]byte{0xFF}, 4096), false, "", ClassRepetitive},
		{"go source", goSource, false, "", ClassSource},
		{"prose", prose, false, "", ClassText},
		{"string table", stringTable, false, ".dynstr", ClassStrings},
		{"pseudorandom", lcgBytes(4096, 5), false, "", ClassCompressed},
		{"text section code", lcgLowEntropy(4096), false, ".text", ClassCode},
		{"empty", nil, false, "", ClassEmpty},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyProbe(tc.data, tc.atStart, tc.section); got != tc.want {
				t.Fatalf("classify(%s) = %s, want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestLooksLikeSource(t *testing.T) {
	src := []byte(strings.Repeat("x = f(a) + g(b); // note\nif (x > 0) { y[i] = x; }\n", 30))
	if !looksLikeSource(src) {
		t.Error("expected source-like block to be recognised")
	}
	prose := []byte(strings.Repeat("just some flowing words without much punctuation here today\n", 20))
	if looksLikeSource(prose) {
		t.Error("prose should not look like source")
	}
}

func TestClassRoundTripNames(t *testing.T) {
	for _, c := range Classes() {
		if c.Glyph() == 0 || c.Glyph() == '?' {
			t.Errorf("class %d has no glyph", c)
		}
		if c.Name() == "" || c.Name() == "unknown" {
			t.Errorf("class %d has no name", c)
		}
		b, err := c.MarshalText()
		if err != nil || string(b) != c.Name() {
			t.Errorf("MarshalText(%v) = %q, %v", c, b, err)
		}
	}
}

// lcgLowEntropy builds a block that looks like machine code: non-printable,
// mid-range entropy (roughly 0.6-0.8 of the maximum), no long runs.
func lcgLowEntropy(n int) []byte {
	b := make([]byte, n)
	x := uint64(42)
	for i := range b {
		x = x*2862933555777941757 + 3037000493
		// Bias toward a subset of byte values so entropy stays below random.
		b[i] = byte(x>>40) & 0x3F
		if i%7 == 0 {
			b[i] |= 0x80
		}
	}
	return b
}
