package main

import (
	"bytes"
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
