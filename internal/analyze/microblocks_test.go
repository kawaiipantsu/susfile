package analyze

import "testing"

func TestDownsampleCountAndDeterminism(t *testing.T) {
	mb := make([]MicroBlock, 4096)
	for i := range mb {
		mb[i].Offset = int64(i * 16)
		mb[i].Len = 16
		mb[i].Entropy = float64(i%9) / 8 * 8
		mb[i].Class = Class(i % 12)
	}

	a := Downsample(mb, 100)
	b := Downsample(mb, 100)
	if len(a) != 100 {
		t.Fatalf("len = %d, want 100", len(a))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Downsample not deterministic at %d: %+v vs %+v", i, a[i], b[i])
		}
	}

	if got := Downsample(mb, 999999); len(got) != len(mb) {
		t.Errorf("n above len(mb) should clamp: got %d", len(got))
	}
	if Downsample(nil, 10) != nil {
		t.Error("Downsample(nil) should be nil")
	}
}

func TestDominantClassPrefersSignal(t *testing.T) {
	var c [16]int
	c[ClassData] = 2
	c[ClassCode] = 2
	if got := dominantClass(c); got != ClassCode {
		t.Fatalf("tie should go to the higher-signal class, got %s", got)
	}
	c = [16]int{}
	c[ClassText] = 5
	c[ClassCode] = 1
	if got := dominantClass(c); got != ClassText {
		t.Fatalf("clear majority should win, got %s", got)
	}
}

func TestWiden(t *testing.T) {
	lo, hi := widen(1000, 1016, 100000)
	if hi-lo < 4096 {
		t.Fatalf("widen produced %d bytes, want >= 4096", hi-lo)
	}
	// Near the start it must clamp at 0 and still deliver the width.
	lo, hi = widen(0, 16, 100000)
	if lo != 0 || hi-lo < 4096 {
		t.Fatalf("widen at start: lo=%d hi=%d", lo, hi)
	}
	// Tiny total: cannot exceed it.
	lo, hi = widen(0, 10, 10)
	if lo != 0 || hi != 10 {
		t.Fatalf("widen with tiny total: lo=%d hi=%d", lo, hi)
	}
}

func TestSectionAtPrefersTightest(t *testing.T) {
	secs := []Section{
		{Name: "big", Offset: 0, Size: 10000},
		{Name: "small", Offset: 100, Size: 50},
	}
	if got := sectionAt(secs, 120); got != "small" {
		t.Fatalf("sectionAt = %q, want small", got)
	}
	if got := sectionAt(secs, 5000); got != "big" {
		t.Fatalf("sectionAt = %q, want big", got)
	}
	if got := sectionAt(nil, 1); got != "" {
		t.Fatalf("sectionAt(nil) = %q", got)
	}
}
