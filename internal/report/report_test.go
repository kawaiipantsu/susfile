package report

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kawaiipantsu/susfile/internal/analyze"
)

func analyzeBytes(t *testing.T, name string, data []byte) *analyze.Result {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := analyze.Analyze(context.Background(), p, analyze.Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestPlain(t *testing.T) {
	r := analyzeBytes(t, "hello.txt", []byte(strings.Repeat("hello world\n", 500)))
	var buf bytes.Buffer
	Plain(&buf, r, Options{Color: false})
	out := buf.String()

	for _, want := range []string{
		"susfile — hello.txt", "SHA-256", "Entropy", "File map", "Legend", "Verdict",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plain output missing %q", want)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Error("colour escape present with Color:false")
	}
}

func TestPlainColorEmitsANSI(t *testing.T) {
	r := analyzeBytes(t, "x.bin", []byte(strings.Repeat("abc\x00\x01\x02", 3000)))
	var buf bytes.Buffer
	Plain(&buf, r, Options{Color: true})
	if !strings.Contains(buf.String(), "\x1b[38;5;") {
		t.Error("expected 256-colour escapes with Color:true")
	}
}

func TestJSON(t *testing.T) {
	r := analyzeBytes(t, "prog", []byte(strings.Repeat("package main\nfunc x(){}\n", 400)))
	var buf bytes.Buffer
	if err := JSON(&buf, r); err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Schema string   `json:"schema"`
		Tool   string   `json:"tool"`
		Map    []string `json:"map"`
		Legend []struct {
			Glyph string `json:"glyph"`
			Name  string `json:"name"`
		} `json:"legend"`
		Result struct {
			SHA256      string `json:"sha256"`
			MicroBlocks []struct {
				Class string `json:"class"`
			} `json:"microblocks"`
			Verdict struct {
				Kaomoji string `json:"kaomoji"`
			} `json:"verdict"`
		} `json:"result"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if doc.Schema != JSONSchema || doc.Tool != "susfile" {
		t.Errorf("schema/tool = %q/%q", doc.Schema, doc.Tool)
	}
	if len(doc.Map) == 0 || len(doc.Legend) == 0 {
		t.Error("map/legend missing from JSON")
	}
	if len(doc.Result.MicroBlocks) == 0 {
		t.Error("microblocks missing from JSON")
	}
	if doc.Result.MicroBlocks[0].Class == "" {
		t.Error("micro-block class should encode as a name string")
	}
	if doc.Result.Verdict.Kaomoji == "" {
		t.Error("verdict kaomoji missing")
	}
	if len(doc.Result.SHA256) != 64 {
		t.Errorf("sha256 length = %d", len(doc.Result.SHA256))
	}
}

func TestLegendCoversClasses(t *testing.T) {
	if len(Legend()) != len(analyze.Classes()) {
		t.Fatalf("legend has %d entries, classes has %d", len(Legend()), len(analyze.Classes()))
	}
}
