package analyze

import "math"

// statsPass computes the ent-style arithmetic summaries over the buffered
// prefix. These are diagnostics, not verdicts: random data sits near
// mean 127.5, chi-square ~255, serial correlation ~0 and a Monte-Carlo pi
// close to math.Pi.
func statsPass(buf []byte, r *Result) {
	n := len(buf)
	if n == 0 {
		return
	}

	// Mean and chi-square from the histogram (already computed).
	var sum float64
	for v, c := range r.Histogram {
		sum += float64(v) * float64(c)
	}
	mean := sum / float64(n)

	expected := float64(n) / 256.0
	var chi float64
	if expected > 0 {
		for _, c := range r.Histogram {
			d := float64(c) - expected
			chi += d * d / expected
		}
	}

	r.Stats = Stats{
		Mean:          mean,
		ChiSquare:     chi,
		ChiSquareProb: chiSquareProb(chi, 255),
		SerialCorr:    serialCorrelation(buf),
	}
	r.Stats.MonteCarloPi, r.Stats.MonteCarloErr = monteCarloPi(buf)
}

// chiSquareProb approximates P(X >= chi) for a chi-square distribution with df
// degrees of freedom, using the Wilson-Hilferty normal approximation. It is
// good to a few percent for df >= 50, which is always the case here (df = 255).
func chiSquareProb(chi float64, df int) float64 {
	if chi <= 0 || df <= 0 {
		return 1
	}
	d := float64(df)
	t := math.Cbrt(chi / d)
	mean := 1 - 2/(9*d)
	sd := math.Sqrt(2 / (9 * d))
	z := (t - mean) / sd
	return 0.5 * math.Erfc(z/math.Sqrt2)
}

// serialCorrelation is the lag-1 autocorrelation coefficient of the byte
// values, as ent reports it. It is near 0 for independent bytes and near 1 for
// a slowly varying stream.
func serialCorrelation(b []byte) float64 {
	n := len(b)
	if n < 2 {
		return 0
	}
	var t1, t2, t3 float64
	last := float64(b[n-1])
	for i := 0; i < n; i++ {
		cur := float64(b[i])
		var next float64
		if i+1 < n {
			next = float64(b[i+1])
		} else {
			next = last // wrap, as ent does
		}
		t1 += cur * next
		t2 += cur
		t3 += cur * cur
	}
	fn := float64(n)
	num := fn*t1 - t2*t2
	den := fn*t3 - t2*t2
	if den == 0 {
		return 0
	}
	return num / den
}

// monteCarloPi treats successive 6-byte groups as (x, y) points in the unit
// square (three bytes each, big-endian) and estimates pi from the fraction
// inside the quarter circle. Returns the estimate and its fractional error
// against math.Pi.
func monteCarloPi(b []byte) (est, frErr float64) {
	const grp = 6
	if len(b) < grp {
		return 0, 1
	}
	const scale = float64(1 << 24) // three bytes
	var inside, total int
	for i := 0; i+grp <= len(b); i += grp {
		x := float64(uint32(b[i])<<16|uint32(b[i+1])<<8|uint32(b[i+2])) / scale
		y := float64(uint32(b[i+3])<<16|uint32(b[i+4])<<8|uint32(b[i+5])) / scale
		total++
		if x*x+y*y <= 1.0 {
			inside++
		}
	}
	if total == 0 {
		return 0, 1
	}
	est = 4.0 * float64(inside) / float64(total)
	frErr = math.Abs(est-math.Pi) / math.Pi
	return est, frErr
}
