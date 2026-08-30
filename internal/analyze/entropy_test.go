package analyze

import (
	"bytes"
	"math"
	"testing"
)

func TestEntropyOf(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want float64
		tol  float64
	}{
		{"empty", nil, 0, 0},
		{"all zeros", bytes.Repeat([]byte{0}, 4096), 0, 0},
		{"single repeated byte", bytes.Repeat([]byte{0xAA}, 1000), 0, 0},
		{"two values half each", append(bytes.Repeat([]byte{0}, 512), bytes.Repeat([]byte{1}, 512)...), 1, 1e-9},
		{"uniform 256", uniformBytes(), 8, 1e-9},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := entropyOf(tc.data)
			if math.Abs(got-tc.want) > tc.tol {
				t.Fatalf("entropyOf = %v, want %v (±%v)", got, tc.want, tc.tol)
			}
		})
	}
}

func TestEntropyOfPseudorandomHigh(t *testing.T) {
	// A long LCG stream should be close to 8 bits/byte.
	b := lcgBytes(1<<16, 12345)
	if h := entropyOf(b); h < 7.9 {
		t.Fatalf("entropy of pseudorandom stream = %v, want >= 7.9", h)
	}
}

func TestEntropySeries(t *testing.T) {
	// First half zeros (entropy 0), second half two-valued (entropy 1).
	half := bytes.Repeat([]byte{0}, 2000)
	alt := make([]byte, 2000)
	for i := range alt {
		alt[i] = byte(i & 1)
	}
	data := append(half, alt...)

	s := EntropySeries(data, 4)
	if len(s) != 4 {
		t.Fatalf("len(series) = %d, want 4", len(s))
	}
	if s[0] > 1e-9 || s[1] > 1e-9 {
		t.Errorf("first-half buckets not ~0: %v", s[:2])
	}
	if math.Abs(s[2]-1) > 1e-6 || math.Abs(s[3]-1) > 1e-6 {
		t.Errorf("second-half buckets not ~1: %v", s[2:])
	}

	if EntropySeries(nil, 8) != nil {
		t.Error("EntropySeries(nil) should be nil")
	}
	if got := EntropySeries([]byte{1, 2, 3}, 10); len(got) != 3 {
		t.Errorf("buckets should clamp to len(b): got %d", len(got))
	}
}

func uniformBytes() []byte {
	b := make([]byte, 256*16)
	for i := range b {
		b[i] = byte(i % 256)
	}
	return b
}

// lcgBytes returns n bytes from a linear congruential generator — deterministic
// pseudorandom, good enough to exercise the high-entropy paths without a
// committed fixture.
func lcgBytes(n int, seed uint64) []byte {
	b := make([]byte, n)
	x := seed
	for i := range b {
		x = x*6364136223846793005 + 1442695040888963407
		b[i] = byte(x >> 33)
	}
	return b
}
