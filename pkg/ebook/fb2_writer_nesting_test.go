package ebook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFB2Writer_DeepNestingNotDropped is a reproduce-first guard (§11.4.115) for
// a real data-loss bug: the FB2 writer only descended two levels
// (chapter.Sections -> sec.Content, then sec.Subsections -> sub.Content). Any
// content nested deeper (sub.Subsections and below) was never iterated, so
// translated text at depth >= 3 silently vanished from the written FB2 file —
// content the EPUB writer (recursive formatSection) preserves.
//
// The test does a real round-trip: write the book to a temp .fb2 on disk, read
// it back, and assert every level's content (including the deepest) is present.
func TestFB2Writer_DeepNestingNotDropped(t *testing.T) {
	book := &Book{
		Metadata: Metadata{Title: "Nesting Test", Language: "en", Authors: []string{"A B"}},
		Chapters: []Chapter{
			{
				Title: "Chapter One",
				Sections: []Section{
					{
						Title:   "Sec L1",
						Content: "CONTENT_LEVEL_1",
						Subsections: []Section{
							{
								Title:   "Sec L2",
								Content: "CONTENT_LEVEL_2",
								Subsections: []Section{
									{
										Title:   "Sec L3",
										Content: "CONTENT_LEVEL_3_DEEP",
										Subsections: []Section{
											{Content: "CONTENT_LEVEL_4_DEEPEST"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "nesting.fb2")
	if err := NewFB2Writer().Write(book, path); err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	out := string(raw)

	for _, want := range []string{
		"CONTENT_LEVEL_1",
		"CONTENT_LEVEL_2",
		"CONTENT_LEVEL_3_DEEP",    // dropped by the 2-level-only writer
		"CONTENT_LEVEL_4_DEEPEST", // dropped by the 2-level-only writer
	} {
		if !strings.Contains(out, want) {
			t.Errorf("DATA LOSS: nested content %q missing from written FB2 — deep section "+
				"content was dropped during write", want)
		}
	}
}
