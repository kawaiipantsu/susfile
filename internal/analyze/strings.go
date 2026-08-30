package analyze

// stringsPass extracts printable runs (ASCII and UTF-16LE) of at least
// opt.StringsMin characters from the buffered prefix, keeping their offsets.
// At most opt.MaxStrings are retained; the total found is still counted.
func stringsPass(buf []byte, r *Result, opt Options) {
	minLen := opt.StringsMin
	if minLen < 1 {
		minLen = 1
	}
	r.StringsMin = minLen

	hits := make([]StringHit, 0, 64)
	total := 0
	add := func(h StringHit) {
		total++
		if len(hits) < opt.MaxStrings {
			hits = append(hits, h)
		}
	}

	extractASCII(buf, minLen, add)
	extractUTF16LE(buf, minLen, add)

	r.Strings = hits
	r.StringsTotal = total
}

func strPrintable(c byte) bool {
	return (c >= 0x20 && c <= 0x7e) || c == '\t'
}

func extractASCII(buf []byte, minLen int, add func(StringHit)) {
	start := -1
	for i := 0; i < len(buf); i++ {
		if strPrintable(buf[i]) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 && i-start >= minLen {
			add(StringHit{Offset: int64(start), Encoding: "ascii", Text: string(buf[start:i])})
		}
		start = -1
	}
	if start >= 0 && len(buf)-start >= minLen {
		add(StringHit{Offset: int64(start), Encoding: "ascii", Text: string(buf[start:])})
	}
}

// extractUTF16LE finds runs of (printable-ascii, 0x00) pairs. It is deliberately
// little-endian only, which covers the common Windows/PE case.
func extractUTF16LE(buf []byte, minLen int, add func(StringHit)) {
	n := len(buf) - 1
	start := -1
	count := 0
	flush := func(end int) {
		if start >= 0 && count >= minLen {
			runes := make([]byte, 0, count)
			for j := start; j < end; j += 2 {
				runes = append(runes, buf[j])
			}
			add(StringHit{Offset: int64(start), Encoding: "utf16le", Text: string(runes)})
		}
		start, count = -1, 0
	}
	for i := 0; i+1 < n+1; i += 2 {
		if strPrintable(buf[i]) && buf[i+1] == 0x00 {
			if start < 0 {
				start = i
			}
			count++
			continue
		}
		flush(i)
	}
	flush(len(buf) &^ 1)
}
