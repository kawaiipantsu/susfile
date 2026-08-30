package analyze

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAnalyzeEmptyFile(t *testing.T) {
	r, err := Analyze(context.Background(), writeTemp(t, "empty", nil), Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Size != 0 || r.Verdict.Kaomoji != kaoEmpty {
		t.Fatalf("empty file: size=%d verdict=%q", r.Size, r.Verdict.Kaomoji)
	}
}

func TestAnalyzeTextFile(t *testing.T) {
	src := strings.Repeat("package main\n\nfunc main() { println(\"hi\") }\n\n", 200)
	r, err := Analyze(context.Background(), writeTemp(t, "main.go", []byte(src)), Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Verdict.Kaomoji != kaoText {
		t.Errorf("verdict = %q (%s), want %q", r.Verdict.Kaomoji, r.Verdict.Summary, kaoText)
	}
	if r.PrintableFrac < 0.9 {
		t.Errorf("printable frac = %v", r.PrintableFrac)
	}
	if len(r.MicroBlocks) == 0 {
		t.Error("no micro-blocks")
	}
	if r.ClassCounts["source"]+r.ClassCounts["text"] == 0 {
		t.Errorf("expected source/text blocks, got %v", r.ClassCounts)
	}
}

func TestAnalyzePseudorandomFile(t *testing.T) {
	data := lcgBytes(400*1024, 2024)
	r, err := Analyze(context.Background(), writeTemp(t, "rnd.bin", data), Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.GlobalEntropy < 7.9 {
		t.Errorf("global entropy = %v, want >= 7.9", r.GlobalEntropy)
	}
	if r.Verdict.Kaomoji != kaoHighEnt {
		t.Errorf("verdict = %q (%s), want %q", r.Verdict.Kaomoji, r.Verdict.Summary, kaoHighEnt)
	}
	high := r.ClassCounts["compressed"] + r.ClassCounts["encrypted"]
	if high < len(r.MicroBlocks)/2 {
		t.Errorf("expected mostly high-entropy blocks, got %v of %d", r.ClassCounts, len(r.MicroBlocks))
	}
}

func TestAnalyzeHashesMatch(t *testing.T) {
	data := []byte("the quick brown fox\n")
	p := writeTemp(t, "f", data)
	r, err := Analyze(context.Background(), p, Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := hex.EncodeToString(sha256Sum(data))
	if r.SHA256 != want {
		t.Fatalf("sha256 = %s, want %s", r.SHA256, want)
	}
}

func TestAnalyzeTruncationAndProgress(t *testing.T) {
	data := lcgBytes(300*1024, 1)
	var stages []Stage
	r, err := Analyze(context.Background(), writeTemp(t, "big", data), Options{MaxBytes: 64 * 1024},
		func(s Stage, _ float64) { stages = append(stages, s) })
	if err != nil {
		t.Fatal(err)
	}
	if !r.Truncated {
		t.Error("expected Truncated with a small MaxBytes")
	}
	if r.Counted > 64*1024 {
		t.Errorf("counted %d bytes, want <= 65536", r.Counted)
	}
	if len(r.MicroBlocks) == 0 {
		t.Error("micro-blocks should still be produced when sampling")
	}
	if !containsStage(stages, StageDone) {
		t.Error("progress never reported StageDone")
	}
}

func TestAnalyzeStdin(t *testing.T) {
	r, err := Analyze(context.Background(), writeTemp(t, "x", []byte("hello stdin world\n")), Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.FromStdin {
		t.Error("file input wrongly marked FromStdin")
	}
}

func TestAnalyzeRejectsDirectory(t *testing.T) {
	_, err := Analyze(context.Background(), t.TempDir(), Options{}, nil)
	if err == nil {
		t.Fatal("expected an error for a directory")
	}
}

func TestAnalyzeMissingFile(t *testing.T) {
	_, err := Analyze(context.Background(), filepath.Join(t.TempDir(), "nope"), Options{}, nil)
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func sha256Sum(b []byte) []byte { h := sha256.Sum256(b); return h[:] }

func containsStage(ss []Stage, want Stage) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
