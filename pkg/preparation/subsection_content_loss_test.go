package preparation

import (
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
)

// TestExtractChapterContent_IncludesSubsections proves that nested subsection
// text is fed to the per-chapter analysis. FB2 parsing populates
// Section.Subsections (pkg/ebook/fb2_parser.go), and translateSection recurses
// into them — but extractChapterContent only read top-level Section.Content,
// silently dropping every nested subsection from the analysis input. The LLM
// then analyses an incomplete chapter and the resulting context/terms/caveats
// miss everything in the subsections.
func TestExtractChapterContent_IncludesSubsections(t *testing.T) {
	chapter := &ebook.Chapter{
		Title: "Chapter With Nesting",
		Sections: []ebook.Section{
			{
				Content: "TOP_LEVEL_SECTION_TEXT",
				Subsections: []ebook.Section{
					{
						Content: "FIRST_SUBSECTION_TEXT",
						Subsections: []ebook.Section{
							{Content: "DEEP_NESTED_TEXT"},
						},
					},
					{Content: "SECOND_SUBSECTION_TEXT"},
				},
			},
		},
	}

	pc := &PreparationCoordinator{}
	got := pc.extractChapterContent(chapter)

	for _, want := range []string{
		"TOP_LEVEL_SECTION_TEXT",
		"FIRST_SUBSECTION_TEXT",
		"DEEP_NESTED_TEXT",
		"SECOND_SUBSECTION_TEXT",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("extractChapterContent dropped %q — nested subsection content is missing from "+
				"the analysis input. Got:\n%s", want, got)
		}
	}
}

// TestExtractBookContent_IncludesSubsections proves the whole-book analysis
// input also includes nested subsection text, not just top-level sections.
func TestExtractBookContent_IncludesSubsections(t *testing.T) {
	book := &ebook.Book{
		Chapters: []ebook.Chapter{
			{
				Title: "Ch1",
				Sections: []ebook.Section{
					{
						Content:     "BOOK_TOP_TEXT",
						Subsections: []ebook.Section{{Content: "BOOK_SUB_TEXT"}},
					},
				},
			},
		},
	}

	pc := &PreparationCoordinator{}
	got := pc.extractBookContent(book)

	for _, want := range []string{"BOOK_TOP_TEXT", "BOOK_SUB_TEXT"} {
		if !strings.Contains(got, want) {
			t.Fatalf("extractBookContent dropped %q — nested subsection content is missing from "+
				"the whole-book analysis input. Got:\n%s", want, got)
		}
	}
}
