package analyze

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
)

// hashSet computes MD5, SHA-1, SHA-256 and CRC-32 in a single streaming pass.
type hashSet struct {
	md5    hash.Hash
	sha1   hash.Hash
	sha256 hash.Hash
	crc    hash.Hash32
	w      io.Writer
}

func newHashSet() *hashSet {
	h := &hashSet{
		md5:    md5.New(),
		sha1:   sha1.New(),
		sha256: sha256.New(),
		crc:    crc32.NewIEEE(),
	}
	h.w = io.MultiWriter(h.md5, h.sha1, h.sha256, h.crc)
	return h
}

func (h *hashSet) Write(p []byte) { _, _ = h.w.Write(p) }

func (h *hashSet) sum(r *Result) {
	r.MD5 = hex.EncodeToString(h.md5.Sum(nil))
	r.SHA1 = hex.EncodeToString(h.sha1.Sum(nil))
	r.SHA256 = hex.EncodeToString(h.sha256.Sum(nil))
	r.CRC32 = fmt.Sprintf("%08x", h.crc.Sum32())
}
