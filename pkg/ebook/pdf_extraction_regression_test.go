package ebook

import (
	"strings"
	"testing"
)

// TestPDFParser_RealTextExtraction_RegressionGuard is the permanent §11.4.135
// regression guard for the unipdf-license-gate defect.
//
// HISTORY (the bug this guard catches): PDF input shipped backed by
// github.com/unidoc/unipdf, whose text extractor is LICENSE-GATED — ExtractText
// returned "unipdf license code required" for every real PDF, so PDF input was
// non-functional ("ships but cannot be used", §11.4). The old unit tests only
// fed invalid bytes and asserted failure, so the dead feature passed CI — a
// §11.4 PASS-bluff. The fix swapped to the MIT github.com/ledongthuc/pdf
// extractor.
//
// MUTATION PROOF (§1.1): revert pdf_parser.go to the unipdf extractor and this
// test FAILs — the extracted text comes back empty / a license-error string
// instead of the known sentences. That is what makes this guard real, not a
// rubber stamp.
//
// The fixture pkg/ebook/testdata/sample_text.pdf is a minimal valid single-page
// PDF carrying three known English sentences.
func TestPDFParser_RealTextExtraction_RegressionGuard(t *testing.T) {
	parser := NewPDFParser(nil)
	book, err := parser.Parse("testdata/sample_text.pdf")
	if err != nil {
		t.Fatalf("PDF parse failed (regression: extractor broken): %v", err)
	}
	if len(book.Chapters) == 0 || len(book.Chapters[0].Sections) == 0 {
		t.Fatalf("no content extracted from PDF (regression: empty extraction)")
	}
	got := book.Chapters[0].Sections[0].Content

	// Known sentences embedded in the fixture — extraction MUST return real text.
	wants := []string{
		"brave knight rode across the silent valley",
		"old letter sealed with red wax",
		"mountains stood quiet under a pale morning sky",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("extracted text missing %q\n--- got ---\n%s", w, got)
		}
	}

	// Guard against the exact failure mode: a license-gated extractor returned a
	// license-error string instead of document text.
	if strings.Contains(strings.ToLower(got), "license") {
		t.Errorf("extracted text contains a license-error string (regression to license-gated extractor): %q", got)
	}
}
