package ebook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTXTParser_Parse_VeryLongLine_NoTruncation is a reproduce-first RED test for
// the bufio.Scanner default-buffer data-loss bug. bufio.Scanner has a default
// MaxScanTokenSize of 64 KiB; a single line longer than that makes Scan() stop
// and scanner.Err() return bufio.ErrTooLong. For an ebook translator a chapter
// emitted as one physical line (common in stripped TXT exports) exceeds 64 KiB —
// Parse then returns an error and the ENTIRE book content is lost.
//
// Uses Cyrillic text so the failure is also exercised on multi-byte content.
func TestTXTParser_Parse_VeryLongLine_NoTruncation(t *testing.T) {
	parser := NewTXTParser()

	// One physical line well over the 64 KiB default scanner buffer.
	// "Привет мир. " is 22 bytes (UTF-8); 10000 reps ≈ 220 KB on a single line.
	unit := "Привет мир. "
	longLine := strings.Repeat(unit, 10000)

	dir := t.TempDir()
	tmpFile := filepath.Join(dir, "verylong.txt")
	if err := os.WriteFile(tmpFile, []byte(longLine+"\n"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	book, err := parser.Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed on a >64KiB single line (data loss bug): %v", err)
	}
	if book == nil || len(book.Chapters) == 0 {
		t.Fatal("Book/chapters empty")
	}

	got := book.Chapters[0].Sections[0].Content
	if !strings.Contains(got, longLine) {
		t.Fatalf("long line truncated/lost: got %d bytes, want >= %d bytes",
			len(got), len(longLine))
	}
	if !utf8.ValidString(got) {
		t.Fatalf("content is not valid UTF-8 after parse")
	}
}
