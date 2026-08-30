package analyze

import (
	"os"
	"testing"
)

// The test binary itself is an executable in the host's native format. Parsing
// it exercises the real ELF/PE/Mach-O path without committing a fixture.
func TestDetectBinFmtOnTestBinary(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skipf("no executable path: %v", err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Skipf("cannot read test binary: %v", err)
	}

	bf, warns := detectBinFmt(data)
	if bf == nil {
		t.Fatalf("expected a BinFmt for the test binary; warnings: %v", warns)
	}
	if bf.Format != "ELF" && bf.Format != "PE" && bf.Format != "Mach-O" {
		t.Errorf("unexpected format %q", bf.Format)
	}
	if bf.Bits != 32 && bf.Bits != 64 {
		t.Errorf("bits = %d", bf.Bits)
	}
	if len(bf.Sections) == 0 {
		t.Error("expected at least one section")
	}
	for _, s := range bf.Sections {
		if s.Entropy < 0 || s.Entropy > 8.0001 {
			t.Errorf("section %s entropy out of range: %v", s.Name, s.Entropy)
		}
	}
}

func TestDetectBinFmtMalformedNeverPanics(t *testing.T) {
	cases := [][]byte{
		{},
		[]byte("\x7fELF"),
		[]byte("\x7fELF\x02\x01\x01\x00 and then garbage that is not a real header"),
		append([]byte("MZ"), make([]byte, 128)...),
		[]byte("\xcf\xfa\xed\xfe\x00\x00\x00\x00"),
		append([]byte("\x7fELF"), randomish(4096)...),
	}
	for i, c := range cases {
		bf, warns := detectBinFmt(c)
		_ = bf
		_ = warns
		t.Logf("case %d: bf=%v warns=%v", i, bf != nil, warns)
	}
}

func TestStructurePassFillsMagicAndMIME(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	r := &Result{}
	structurePass(png, r)
	if !r.Magic.Matched || r.Magic.Label == "" {
		t.Errorf("magic not matched: %+v", r.Magic)
	}
	if r.ExtGuess == "" {
		t.Errorf("expected an extension guess")
	}
}

func randomish(n int) []byte { return lcgBytes(n, 0xC0FFEE) }
