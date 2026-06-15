package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
)

// TestTitlePathLeak_TXTParserUsesPathAsTitle documents the ROOT CAUSE (§11.4.115
// RED baseline): the real TXT parser the CLI uses sets the book metadata title to
// the raw input file PATH. Without sanitization that path is sent through the
// translator as if it were title text and leaks into the output <dc:title>.
//
// This is the defect-present assertion: it proves the leak exists in the parser
// output BEFORE cmd/cli sanitizes it. If a future parser change stops using the
// path as the title, this test surfaces that as a finding rather than silently
// passing.
func TestTitlePathLeak_TXTParserUsesPathAsTitle(t *testing.T) {
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "my_great_book.txt")
	if err := os.WriteFile(inputFile, []byte("Hello world.\nThis is a chapter.\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(inputFile)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// RED baseline: the unsanitized parser title IS the file path — the leak.
	if book.Metadata.Title != inputFile {
		t.Fatalf("expected TXT parser to leak the file path as title (root cause), got %q (input %q)",
			book.Metadata.Title, inputFile)
	}
}

// TestSanitizeBookTitle_RemovesLeakedPath is the GREEN regression guard: the
// cmd/cli sanitizer MUST replace a path-derived title with a clean human title
// and MUST NEVER leave a filesystem path as the title that would be translated.
func TestSanitizeBookTitle_RemovesLeakedPath(t *testing.T) {
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "my_great_book.txt")
	if err := os.WriteFile(inputFile, []byte("Hello world.\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(inputFile)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	got := sanitizeBookTitle(book.Metadata.Title, inputFile)

	// 1) The clean title must NOT contain the directory path.
	if strings.Contains(got, dir) {
		t.Errorf("sanitized title still contains the directory path: %q", got)
	}
	// 2) The clean title must NOT contain a path separator (no leaked path).
	if strings.ContainsRune(got, filepath.Separator) {
		t.Errorf("sanitized title still looks like a path: %q", got)
	}
	// 3) The clean title must NOT contain the file extension.
	if strings.HasSuffix(strings.ToLower(got), ".txt") {
		t.Errorf("sanitized title still has the file extension: %q", got)
	}
	// 4) Positive: derived from the base name with separators normalised.
	if got != "my great book" {
		t.Errorf("expected clean title %q, got %q", "my great book", got)
	}
}

// TestSanitizeBookTitle_PreservesGenuineTitle ensures the fix is surgical: a real
// title (EPUB/FB2 parsers set genuine titles) is returned unchanged.
func TestSanitizeBookTitle_PreservesGenuineTitle(t *testing.T) {
	cases := []struct {
		name      string
		title     string
		inputFile string
		want      string
	}{
		{"genuine title untouched", "War and Peace", "/tmp/war.epub", "War and Peace"},
		{"title with colon untouched", "Dune: Part One", "/books/dune.fb2", "Dune: Part One"},
		{"empty title derives from path", "", "/books/the_hobbit.txt", "the hobbit"},
		{"path title sanitized", "/abs/path/book_one.txt", "/abs/path/book_one.txt", "book one"},
		{"basename-as-title sanitized", "book_one.txt", "/abs/path/book_one.txt", "book one"},
		{"genuine title that contains a slash but no ebook ext", "A/B Testing", "/tmp/ab.epub", "A/B Testing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeBookTitle(tc.title, tc.inputFile)
			if got != tc.want {
				t.Errorf("sanitizeBookTitle(%q, %q) = %q, want %q", tc.title, tc.inputFile, got, tc.want)
			}
		})
	}
}
