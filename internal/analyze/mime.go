package analyze

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

// detectMIME runs the pure-Go mimetype sniffer over the buffered prefix and
// returns the detected type, its parent chain (most specific first) and a
// best-guess extension without the leading dot.
func detectMIME(buf []byte) (mime string, tree []string, ext string) {
	if len(buf) == 0 {
		return "", nil, ""
	}
	mt := mimetype.Detect(buf)
	for m := mt; m != nil; m = m.Parent() {
		s := m.String()
		if s == "" {
			break
		}
		tree = append(tree, s)
		if s == "application/octet-stream" {
			break
		}
	}
	return mt.String(), tree, strings.TrimPrefix(mt.Extension(), ".")
}
