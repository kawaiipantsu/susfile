package analyze

import (
	"bytes"
	"testing"
)

func TestExtractASCII(t *testing.T) {
	buf := append([]byte{0, 1, 2}, []byte("hello")...)
	buf = append(buf, 0xff, 0xfe)
	buf = append(buf, []byte("world!!")...)

	var hits []StringHit
	extractASCII(buf, 4, func(h StringHit) { hits = append(hits, h) })

	if len(hits) != 2 {
		t.Fatalf("got %d hits, want 2: %+v", len(hits), hits)
	}
	if hits[0].Text != "hello" || hits[0].Offset != 3 {
		t.Errorf("hit0 = %+v", hits[0])
	}
	if hits[1].Text != "world!!" || hits[1].Encoding != "ascii" {
		t.Errorf("hit1 = %+v", hits[1])
	}
}

func TestExtractASCIIMinLength(t *testing.T) {
	buf := []byte("ab\x00abcd\x00abcde")
	var got []string
	extractASCII(buf, 4, func(h StringHit) { got = append(got, h.Text) })
	want := []string{"abcd", "abcde"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestExtractUTF16LE(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte{0x00, 0x00})
	for _, r := range "C:\\Windows" {
		buf.WriteByte(byte(r))
		buf.WriteByte(0x00)
	}
	buf.Write([]byte{0xAA, 0xBB})

	var hits []StringHit
	extractUTF16LE(buf.Bytes(), 4, func(h StringHit) { hits = append(hits, h) })
	if len(hits) != 1 || hits[0].Text != "C:\\Windows" || hits[0].Encoding != "utf16le" {
		t.Fatalf("utf16 hits = %+v", hits)
	}
}

func TestStringsPassCap(t *testing.T) {
	// Many short strings separated by NULs; cap retention but keep counting.
	var buf bytes.Buffer
	for i := 0; i < 500; i++ {
		buf.WriteString("abcd")
		buf.WriteByte(0)
	}
	r := &Result{}
	stringsPass(buf.Bytes(), r, Options{StringsMin: 4, MaxStrings: 10})
	if len(r.Strings) != 10 {
		t.Errorf("retained %d, want 10", len(r.Strings))
	}
	if r.StringsTotal < 500 {
		t.Errorf("total = %d, want >= 500", r.StringsTotal)
	}
}

func TestStringsPassGiantRun(t *testing.T) {
	// One enormous printable run must not blow the cap or hang.
	buf := bytes.Repeat([]byte("A"), 2<<20)
	r := &Result{}
	stringsPass(buf, r, Options{StringsMin: 4, MaxStrings: 8})
	if len(r.Strings) != 1 {
		t.Fatalf("expected a single hit for one big run, got %d", len(r.Strings))
	}
	if len(r.Strings[0].Text) != len(buf) {
		t.Errorf("hit text length = %d, want %d", len(r.Strings[0].Text), len(buf))
	}
}
