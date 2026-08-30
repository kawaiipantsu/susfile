package analyze

import (
	"encoding/hex"
	"strings"
)

// signature is one entry in susfile's magic table. A match requires buf to hold
// magic at position off, honouring mask when non-nil (buf[off+i]&mask[i] ==
// magic[i]&mask[i]).
type signature struct {
	off   int
	magic []byte
	mask  []byte
	label string
	mime  string
	ext   string
}

func b(s string) []byte { return []byte(s) }

// signatures is checked in order; put more specific entries first. It is a
// curated heuristic, not libmagic — misses and occasional mislabels are
// expected and documented.
var signatures = []signature{
	// Executables and objects
	{0, b("\x7fELF"), nil, "ELF binary", "application/x-elf", "elf"},
	{0, b("MZ"), nil, "DOS/PE executable", "application/vnd.microsoft.portable-executable", "exe"},
	// CAFEBABE is shared by Mach-O "fat" binaries and Java .class files; the
	// mimetype cross-check and the structure pass disambiguate. Mach-O fat is
	// listed here; a real .class still reports "Java class file" via MIME.
	{0, b("\xca\xfe\xba\xbe"), nil, "Mach-O universal binary or Java class", "application/x-mach-binary", ""},
	{0, b("\xcf\xfa\xed\xfe"), nil, "Mach-O 64-bit (LE)", "application/x-mach-binary", ""},
	{0, b("\xce\xfa\xed\xfe"), nil, "Mach-O 32-bit (LE)", "application/x-mach-binary", ""},
	{0, b("\xfe\xed\xfa\xcf"), nil, "Mach-O 64-bit (BE)", "application/x-mach-binary", ""},
	{0, b("\xfe\xed\xfa\xce"), nil, "Mach-O 32-bit (BE)", "application/x-mach-binary", ""},
	{0, b("\x00\x61\x73\x6d"), nil, "WebAssembly module", "application/wasm", "wasm"},
	{0, b("#!"), nil, "script with shebang", "text/x-shellscript", "sh"},
	{0, b("\xde\xc0\x17\x0b"), nil, "LLVM bitcode", "application/x-llvm", "bc"},
	{0, b("!<arch>\n"), nil, "Unix ar archive / .deb", "application/x-archive", "a"},

	// Compression / archives
	{0, b("\x1f\x8b"), nil, "gzip compressed data", "application/gzip", "gz"},
	{0, b("BZh"), nil, "bzip2 compressed data", "application/x-bzip2", "bz2"},
	{0, b("\xfd7zXZ\x00"), nil, "xz compressed data", "application/x-xz", "xz"},
	{0, b("\x28\xb5\x2f\xfd"), nil, "Zstandard compressed data", "application/zstd", "zst"},
	{0, b("\x04\x22\x4d\x18"), nil, "LZ4 frame", "application/x-lz4", "lz4"},
	{0, b("\x5d\x00\x00"), nil, "LZMA compressed data", "application/x-lzma", "lzma"},
	{0, b("PK\x03\x04"), nil, "ZIP archive", "application/zip", "zip"},
	{0, b("PK\x05\x06"), nil, "ZIP archive (empty)", "application/zip", "zip"},
	{0, b("PK\x07\x08"), nil, "ZIP archive (spanned)", "application/zip", "zip"},
	{0, b("Rar!\x1a\x07\x00"), nil, "RAR archive v4", "application/vnd.rar", "rar"},
	{0, b("Rar!\x1a\x07\x01\x00"), nil, "RAR archive v5", "application/vnd.rar", "rar"},
	{0, b("7z\xbc\xaf\x27\x1c"), nil, "7-Zip archive", "application/x-7z-compressed", "7z"},
	{257, b("ustar"), nil, "POSIX tar archive", "application/x-tar", "tar"},
	{0, b("\x1f\x9d"), nil, "compress'd data (.Z)", "application/x-compress", "Z"},

	// Documents
	{0, b("%PDF-"), nil, "PDF document", "application/pdf", "pdf"},
	{0, b("{\\rtf"), nil, "Rich Text Format", "application/rtf", "rtf"},
	{0, b("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1"), nil, "MS Office / OLE2 compound", "application/x-ole-storage", ""},

	// Images
	{0, b("\x89PNG\r\n\x1a\n"), nil, "PNG image", "image/png", "png"},
	{0, b("\xff\xd8\xff"), nil, "JPEG image", "image/jpeg", "jpg"},
	{0, b("GIF87a"), nil, "GIF image (87a)", "image/gif", "gif"},
	{0, b("GIF89a"), nil, "GIF image (89a)", "image/gif", "gif"},
	{0, b("BM"), nil, "BMP image", "image/bmp", "bmp"},
	{0, b("II\x2a\x00"), nil, "TIFF image (little-endian)", "image/tiff", "tiff"},
	{0, b("MM\x00\x2a"), nil, "TIFF image (big-endian)", "image/tiff", "tiff"},
	{0, b("\x00\x00\x01\x00"), nil, "Windows icon", "image/vnd.microsoft.icon", "ico"},
	{0, b("qoif"), nil, "QOI image", "image/qoi", "qoi"},
	// RIFF containers: bytes 0-3 "RIFF", 8-11 form type.
	{8, b("WEBP"), nil, "WebP image", "image/webp", "webp"},
	{8, b("WAVE"), nil, "WAV audio", "audio/wav", "wav"},
	{8, b("AVI "), nil, "AVI video", "video/x-msvideo", "avi"},

	// Audio / video
	{0, b("OggS"), nil, "Ogg container", "application/ogg", "ogg"},
	{0, b("fLaC"), nil, "FLAC audio", "audio/flac", "flac"},
	{0, b("ID3"), nil, "MP3 audio (ID3)", "audio/mpeg", "mp3"},
	{0, b("\x1a\x45\xdf\xa3"), nil, "Matroska / WebM", "video/x-matroska", "mkv"},
	{4, b("ftyp"), nil, "ISO Base Media (MP4/MOV)", "video/mp4", "mp4"},

	// Filesystems / disk / db
	{0, b("SQLite format 3\x00"), nil, "SQLite 3 database", "application/vnd.sqlite3", "sqlite"},
	{0, b("\x53\xef"), nil, "ext2/3/4 superblock fragment", "application/octet-stream", ""},
	{0x8001, b("CD001"), nil, "ISO 9660 CD image", "application/x-iso9660-image", "iso"},
	{0, b("KDMV"), nil, "VMDK disk image", "application/octet-stream", "vmdk"},
	{0, b("conectix"), nil, "VHD disk image", "application/octet-stream", "vhd"},
	{0, b("QFI\xfb"), nil, "QCOW image", "application/octet-stream", "qcow"},

	// Certs / keys / text-ish
	{0, b("-----BEGIN "), nil, "PEM-encoded block", "application/x-pem-file", "pem"},
	{0, b("\x30\x82"), nil, "DER-encoded ASN.1 (likely certificate/key)", "application/pkix-cert", "der"},
	{0, b("ssh-rsa "), nil, "OpenSSH public key", "text/plain", "pub"},
	{0, b("\xef\xbb\xbf"), nil, "UTF-8 text (BOM)", "text/plain", "txt"},
	{0, b("\xff\xfe"), nil, "UTF-16 text (LE BOM)", "text/plain", "txt"},
	{0, b("\xfe\xff"), nil, "UTF-16 text (BE BOM)", "text/plain", "txt"},

	// Fonts
	{0, b("\x00\x01\x00\x00\x00"), nil, "TrueType font", "font/ttf", "ttf"},
	{0, b("OTTO"), nil, "OpenType font", "font/otf", "otf"},
	{0, b("wOFF"), nil, "WOFF font", "font/woff", "woff"},
	{0, b("wOF2"), nil, "WOFF2 font", "font/woff2", "woff2"},

	// Misc dev
	{0, b("SIMPLE  ="), nil, "FITS data", "application/fits", "fits"},
	{0, b("\xed\xab\xee\xdb"), nil, "RPM package", "application/x-rpm", "rpm"},
	{0, b("dex\n"), nil, "Android DEX", "application/octet-stream", "dex"},
	{0, b("\xcf\x84\x01"), nil, "PGP/GPG data", "application/pgp", "gpg"},
}

// matchMagic runs the signature table against the buffer's head and records the
// leading bytes as hex regardless of whether anything matched.
func matchMagic(buf []byte) MagicMatch {
	m := MagicMatch{HeadHex: headHex(buf)}
	for _, s := range signatures {
		if sigMatches(buf, s) {
			m.Matched = true
			m.Label = s.label
			m.MIME = s.mime
			m.Ext = s.ext
			m.Offset = s.off
			return m
		}
	}
	// Fall back to "looks like text" if the head is overwhelmingly printable.
	if looksTextual(buf) {
		m.Matched = true
		m.Label = "plain text"
		m.MIME = "text/plain"
		m.Ext = "txt"
	}
	return m
}

func sigMatches(buf []byte, s signature) bool {
	if s.off+len(s.magic) > len(buf) {
		return false
	}
	seg := buf[s.off : s.off+len(s.magic)]
	if s.mask == nil {
		return string(seg) == string(s.magic)
	}
	for i := range s.magic {
		if seg[i]&s.mask[i] != s.magic[i]&s.mask[i] {
			return false
		}
	}
	return true
}

func headHex(buf []byte) string {
	n := headHexBytes
	if len(buf) < n {
		n = len(buf)
	}
	if n == 0 {
		return ""
	}
	dst := make([]byte, n*3-1)
	for i := 0; i < n; i++ {
		hex.Encode(dst[i*3:], buf[i:i+1])
		if i < n-1 {
			dst[i*3+2] = ' '
		}
	}
	return string(dst)
}

func looksTextual(buf []byte) bool {
	n := len(buf)
	if n == 0 {
		return false
	}
	if n > 1024 {
		n = 1024
	}
	var ok int
	for _, c := range buf[:n] {
		if isPrintable(c) {
			ok++
		}
	}
	return float64(ok)/float64(n) >= 0.95
}

// magicIsContainer reports whether label denotes a format whose payload is
// expected to be high entropy (so a near-8.0 reading is not "suspicious").
func magicIsContainer(label string) bool {
	l := strings.ToLower(label)
	for _, k := range []string{"zip", "gzip", "bzip2", "xz", "zstandard", "lz4", "lzma", "7-zip", "rar", "compress", "png", "jpeg", "webp", "flac", "ogg", "matroska", "mp4", "mp3", "pdf", "woff"} {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}
