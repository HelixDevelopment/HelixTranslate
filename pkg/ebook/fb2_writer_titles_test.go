package ebook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFB2Writer_SectionTitlesNotDropped is a reproduce-first guard (§11.4.115) for
// a real data-loss gap: the FB2 writer preserved section/subsection CONTENT (via
// collectSectionParagraphs) but DROPPED every Section.Title / Subsection.Title —
// only the chapter title was written. For a translated ebook that loses heading
// text. The fix prepends each non-empty section/subsection title to its paragraphs
// so headings survive the write. Real round-trip: write to a temp .fb2, read back,
// assert the titles are present.
func TestFB2Writer_SectionTitlesNotDropped(t *testing.T) {
	book := &Book{
		Metadata: Metadata{Title: "Titles Test", Language: "en", Authors: []string{"A B"}},
		Chapters: []Chapter{
			{
				Title: "Chapter One",
				Sections: []Section{
					{
						Title:   "SECTION_HEADING_L1",
						Content: "Section one body.",
						Subsections: []Section{
							{Title: "SUBSECTION_HEADING_L2", Content: "Subsection body."},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "titles.fb2")
	if err := NewFB2Writer().Write(book, path); err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out := string(raw)

	for _, want := range []string{
		"SECTION_HEADING_L1",    // dropped by the title-less writer
		"SUBSECTION_HEADING_L2", // dropped by the title-less writer
		"Section one body.",     // content (already preserved — guards against regression)
		"Subsection body.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("written FB2 missing %q — section/subsection title or content lost\n--- output ---\n%s", want, out)
		}
	}
}
