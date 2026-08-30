package analyze

import (
	"math"
	"testing"
)

func TestStatsPassOnUniform(t *testing.T) {
	// A perfectly flat histogram: every byte value 256 times.
	buf := make([]byte, 256*256)
	for i := range buf {
		buf[i] = byte(i % 256)
	}
	r := &Result{}
	histogramPass(buf, r)
	statsPass(buf, r)

	if math.Abs(r.Stats.Mean-127.5) > 1e-6 {
		t.Errorf("mean = %v, want 127.5", r.Stats.Mean)
	}
	if r.Stats.ChiSquare > 1e-6 {
		t.Errorf("chi-square of a flat histogram = %v, want ~0", r.Stats.ChiSquare)
	}
	if r.Stats.ChiSquareProb < 0.99 {
		t.Errorf("chi-square prob = %v, want ~1 for a flat histogram", r.Stats.ChiSquareProb)
	}
}

func TestChiSquareProbMonotonic(t *testing.T) {
	// Larger chi-square must not yield a larger probability.
	prev := 1.1
	for _, chi := range []float64{0, 100, 255, 400, 1000, 100000} {
		p := chiSquareProb(chi, 255)
		if p > prev+1e-9 {
			t.Fatalf("prob not monotently decreasing: chi=%v p=%v prev=%v", chi, p, prev)
		}
		if p < 0 || p > 1 {
			t.Fatalf("prob out of range: %v", p)
		}
		prev = p
	}
}

func TestSerialCorrelation(t *testing.T) {
	// A ramp is highly autocorrelated.
	ramp := make([]byte, 4096)
	for i := range ramp {
		ramp[i] = byte(i)
	}
	if sc := serialCorrelation(ramp); sc < 0.9 {
		t.Errorf("serial correlation of a ramp = %v, want >= 0.9", sc)
	}

	// Pseudorandom bytes are near zero.
	if sc := serialCorrelation(lcgBytes(1<<15, 99)); math.Abs(sc) > 0.1 {
		t.Errorf("serial correlation of pseudorandom = %v, want ~0", sc)
	}
}

func TestMonteCarloPi(t *testing.T) {
	est, frErr := monteCarloPi(lcgBytes(1<<20, 7))
	if math.Abs(est-math.Pi) > 0.05 {
		t.Errorf("monte-carlo pi = %v (err %v), want within 0.05 of %v", est, frErr, math.Pi)
	}
	if _, e := monteCarloPi([]byte{1, 2}); e != 1 {
		t.Errorf("monteCarloPi on too-short input should report error 1, got %v", e)
	}
}
