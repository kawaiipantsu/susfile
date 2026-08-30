package analyze

import (
	"strings"
	"testing"
)

func TestMatchMagic(t *testing.T) {
	pad := func(head []byte, n int) []byte {
		b := make([]byte, n)
		copy(b, head)
		return b
	}
	tests := []struct {
		name      string
		buf       []byte
		wantLabel string // substring
		wantMatch bool
	}{
		{"ELF", []byte("\x7fELF\x02\x01\x01"), "ELF", true},
		{"PE/MZ", []byte("MZ\x90\x00\x03\x00"), "PE", true},
		{"PNG", []byte("\x89PNG\r\n\x1a\n\x00\x00"), "PNG", true},
		{"JPEG", []byte("\xff\xd8\xff\xe0\x00\x10JFIF"), "JPEG", true},
		{"GIF89a", []byte("GIF89a...."), "GIF", true},
		{"gzip", []byte("\x1f\x8b\x08\x00"), "gzip", true},
		{"xz", []byte("\xfd7zXZ\x00\x00"), "xz", true},
		{"zstd", []byte("\x28\xb5\x2f\xfd\x00"), "Zstandard", true},
		{"zip", []byte("PK\x03\x04\x14\x00"), "ZIP", true},
		{"pdf", []byte("%PDF-1.7\n%"), "PDF", true},
		{"sqlite", []byte("SQLite format 3\x00"), "SQLite", true},
		{"wasm", []byte("\x00asm\x01\x00\x00\x00"), "WebAssembly", true},
		{"class or macho fat", []byte("\xca\xfe\xba\xbe\x00\x00\x00\x02"), "Mach-O", true},
		{"tar (ustar at 257)", tarHeader(), "tar", true},
		{"webp (RIFF)", []byte("RIFF\x00\x00\x00\x00WEBPVP8 "), "WebP", true},
		{"pem", []byte("-----BEGIN CERTIFICATE-----\n"), "PEM", true},
		{"plain text fallback", []byte("hello, this is just text\nwith another line\n"), "plain text", true},
		{"unknown", pad([]byte{0x11, 0x22, 0x33}, 64), "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := matchMagic(tc.buf)
			if m.Matched != tc.wantMatch {
				t.Fatalf("Matched = %v, want %v (label %q)", m.Matched, tc.wantMatch, m.Label)
			}
			if tc.wantLabel != "" && !strings.Contains(m.Label, tc.wantLabel) {
				t.Fatalf("Label = %q, want to contain %q", m.Label, tc.wantLabel)
			}
			if m.HeadHex == "" && len(tc.buf) > 0 {
				t.Error("HeadHex should never be empty for non-empty input")
			}
		})
	}
}

func TestHeadHex(t *testing.T) {
	got := headHex([]byte{0x89, 0x50, 0x4e, 0x47})
	if got != "89 50 4e 47" {
		t.Fatalf("headHex = %q", got)
	}
	if headHex(nil) != "" {
		t.Error("headHex(nil) should be empty")
	}
}

func tarHeader() []byte {
	b := make([]byte, 512)
	copy(b, []byte("hello.txt"))
	copy(b[257:], []byte("ustar\x0000"))
	return b
}
