package analyze

// histogramPass fills the byte histogram and the derived byte-class fractions
// from the buffered prefix.
func histogramPass(buf []byte, r *Result) {
	var h [256]uint64
	for _, c := range buf {
		h[c]++
	}
	r.Histogram = h

	n := len(buf)
	if n == 0 {
		return
	}

	var printable, nul, high, ws, distinct int
	for v, c := range h {
		if c == 0 {
			continue
		}
		distinct++
		b := byte(v)
		switch {
		case b == 0x00:
			nul += int(c)
		case b >= 0x80:
			high += int(c)
		}
		if isPrintable(b) {
			printable += int(c)
		}
		if isSpace(b) {
			ws += int(c)
		}
	}
	fn := float64(n)
	r.PrintableFrac = float64(printable) / fn
	r.NULFrac = float64(nul) / fn
	r.HighFrac = float64(high) / fn
	r.WSFrac = float64(ws) / fn
	r.DistinctBytes = distinct
}

// isPrintable reports whether b renders as visible text: printable ASCII plus
// the common whitespace controls.
func isPrintable(b byte) bool {
	return (b >= 0x20 && b <= 0x7e) || b == '\t' || b == '\n' || b == '\r'
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}
