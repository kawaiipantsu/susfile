package analyze

import "math"

// shannonBits returns the Shannon entropy in bits per byte for a 256-entry
// frequency table summing to total. The range is [0, 8].
func shannonBits(freq []uint64, total uint64) float64 {
	if total == 0 {
		return 0
	}
	inv := 1.0 / float64(total)
	var h float64
	for _, c := range freq {
		if c == 0 {
			continue
		}
		p := float64(c) * inv
		h -= p * math.Log2(p)
	}
	if h < 0 {
		h = 0 // guard against tiny negative from rounding
	}
	return h
}

// entropyOf is shannonBits for a byte slice.
func entropyOf(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	var f [256]uint64
	for _, c := range b {
		f[c]++
	}
	return shannonBits(f[:], uint64(len(b)))
}

// EntropySeries splits b into buckets equal-width windows and returns the
// Shannon entropy of each, in order. It is the data behind the windowed
// entropy chart. buckets is clamped to [1, len(b)]; a shorter b yields a
// shorter series.
func EntropySeries(b []byte, buckets int) []float64 {
	if len(b) == 0 || buckets < 1 {
		return nil
	}
	if buckets > len(b) {
		buckets = len(b)
	}
	out := make([]float64, buckets)
	n := len(b)
	for i := range out {
		start := i * n / buckets
		end := (i + 1) * n / buckets
		if end <= start {
			end = start + 1
		}
		out[i] = entropyOf(b[start:end])
	}
	return out
}
