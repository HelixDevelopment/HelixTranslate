package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateOutput_HonorsExtension is the permanent §11.4.135 regression guard
// for the "CLI always emits EPUB" defect.
//
// HISTORY (the bug this guard catches): the output step ALWAYS called
// generateEPUB regardless of the -o extension, so `-o book.txt` / `-o book.fb2`
// wrote EPUB (PK-zip) bytes into a misnamed file — a silent wrong-output defect
// (§11.4: user asks for one format, silently gets another). generateOutput now
// dispatches on the extension.
//
// MUTATION PROOF (§1.1): revert generateOutput to always `return generateEPUB(...)`
// and the .txt / .fb2 cases below FAIL (they would receive PK-zip bytes instead
// of plain text / FictionBook XML). That is what makes this guard real.
func TestGenerateOutput_HonorsExtension(t *testing.T) {
	const content = "Здраво свете.\n\nОво је други пасус."
	dir := t.TempDir()

	t.Run("txt is plain UTF-8 text, not a zip", func(t *testing.T) {
		out := filepath.Join(dir, "book.txt")
		if err := generateOutput(content, out, "in.pdf"); err != nil {
			t.Fatalf("generateOutput(.txt) failed: %v", err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.HasPrefix(b, []byte("PK")) {
			t.Errorf(".txt output is a ZIP/EPUB (regression: always-EPUB)")
		}
		if !strings.Contains(string(b), "Здраво свете") {
			t.Errorf(".txt missing translated content; got %q", string(b))
		}
	})

	t.Run("fb2 is FictionBook XML", func(t *testing.T) {
		out := filepath.Join(dir, "book.fb2")
		if err := generateOutput(content, out, "in.pdf"); err != nil {
			t.Fatalf("generateOutput(.fb2) failed: %v", err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if bytes.HasPrefix(b, []byte("PK")) {
			t.Errorf(".fb2 output is a ZIP/EPUB (regression: always-EPUB)")
		}
		if !strings.Contains(s, "FictionBook") {
			t.Errorf(".fb2 is not FictionBook XML; got prefix %q", s[:min(80, len(s))])
		}
		if !strings.Contains(s, "Здраво свете") {
			t.Errorf(".fb2 missing translated content")
		}
	})

	t.Run("html is well-formed escaped HTML", func(t *testing.T) {
		out := filepath.Join(dir, "book.html")
		if err := generateOutput(content, out, "in.pdf"); err != nil {
			t.Fatalf("generateOutput(.html) failed: %v", err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if bytes.HasPrefix(b, []byte("PK")) {
			t.Errorf(".html output is a ZIP/EPUB (regression: always-EPUB)")
		}
		if !strings.Contains(s, "<!DOCTYPE html>") || !strings.Contains(s, "<p>") {
			t.Errorf(".html is not a well-formed HTML document; got prefix %q", s[:min(80, len(s))])
		}
		if !strings.Contains(s, "Здраво свете") {
			t.Errorf(".html missing translated content")
		}
	})

	t.Run("html escapes content (no markup injection)", func(t *testing.T) {
		out := filepath.Join(dir, "inject.html")
		if err := generateOutput("Пас <script>alert(1)</script>", out, "in.pdf"); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(out)
		if strings.Contains(string(b), "<script>") {
			t.Errorf(".html did not escape content — markup injection (got raw <script>)")
		}
		if !strings.Contains(string(b), "&lt;script&gt;") {
			t.Errorf(".html should contain the escaped form &lt;script&gt;")
		}
	})

	t.Run("docx is a valid WordprocessingML package with translated content", func(t *testing.T) {
		out := filepath.Join(dir, "book.docx")
		if err := generateOutput(content, out, "in.pdf"); err != nil {
			t.Fatalf("generateOutput(.docx) failed: %v", err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		// A .docx is a ZIP package — it MUST start with the PK signature (unlike
		// txt/fb2/html). This is the inverse assertion of the always-EPUB guard:
		// here PK is correct, but the parts must be OOXML, not EPUB.
		if !bytes.HasPrefix(b, []byte("PK")) {
			t.Errorf(".docx output is not a ZIP package (got prefix %q)", b[:min(8, len(b))])
		}
		zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatalf(".docx is not a readable zip: %v", err)
		}
		var docXML string
		var haveCT, haveDoc bool
		for _, f := range zr.File {
			switch f.Name {
			case "[Content_Types].xml":
				haveCT = true
			case "word/document.xml":
				haveDoc = true
				rc, _ := f.Open()
				bb, _ := io.ReadAll(rc)
				rc.Close()
				docXML = string(bb)
			}
		}
		if !haveCT {
			t.Errorf(".docx missing [Content_Types].xml")
		}
		if !haveDoc {
			t.Errorf(".docx missing word/document.xml")
		}
		if !strings.Contains(docXML, "wordprocessingml") {
			t.Errorf("word/document.xml is not WordprocessingML; got prefix %q", docXML[:min(120, len(docXML))])
		}
		if !strings.Contains(docXML, "Здраво свете") {
			t.Errorf(".docx missing translated content in word/document.xml")
		}
	})

	t.Run("unsupported extension is an explicit error, not a misnamed EPUB", func(t *testing.T) {
		out := filepath.Join(dir, "book.rtf")
		err := generateOutput(content, out, "in.pdf")
		if err == nil {
			t.Errorf("expected an error for unsupported .rtf output, got nil (silent wrong-output)")
		}
		if _, statErr := os.Stat(out); statErr == nil {
			t.Errorf("an unsupported-format output file was written anyway: %s", out)
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
