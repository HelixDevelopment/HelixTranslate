package preparation

import (
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
)

// TestExtractChapterContent_IncludesSectionTitles proves that section (and
// subsection) TITLES are fed to the per-chapter analysis input.
//
// The translator translates Section.Title (translator.go translateSection), so
// titles are real end-user-visible text that can carry character names,
// untranslatable terms, and cultural references. writeSectionContent wrote only
// Section.Content, silently dropping every section/subsection title from the
// analysis the LLM sees — so the resulting terminology / caveats / context miss
// everything that appears only in a title. Same data-loss class as the
// already-fixed subsection-content drop.
func TestExtractChapterContent_IncludesSectionTitles(t *testing.T) {
	chapter := &ebook.Chapter{
		Title: "Chapter Heading",
		Sections: []ebook.Section{
			{
				Title:   "TOP_SECTION_TITLE",
				Content: "top body",
				Subsections: []ebook.Section{
					{Title: "SUB_SECTION_TITLE", Content: "sub body"},
				},
			},
		},
	}

	pc := &PreparationCoordinator{}
	got := pc.extractChapterContent(chapter)

	for _, want := range []string{"TOP_SECTION_TITLE", "SUB_SECTION_TITLE"} {
		if !strings.Contains(got, want) {
			t.Fatalf("extractChapterContent dropped section title %q — section titles are "+
				"translated but excluded from the analysis input. Got:\n%s", want, got)
		}
	}
}

// TestExtractBookContent_IncludesSectionTitles proves the whole-book analysis
// input also includes section/subsection titles.
func TestExtractBookContent_IncludesSectionTitles(t *testing.T) {
	book := &ebook.Book{
		Chapters: []ebook.Chapter{
			{
				Title: "Ch1",
				Sections: []ebook.Section{
					{
						Title:       "BOOK_SECTION_TITLE",
						Content:     "body",
						Subsections: []ebook.Section{{Title: "BOOK_SUBSECTION_TITLE", Content: "subbody"}},
					},
				},
			},
		},
	}

	pc := &PreparationCoordinator{}
	got := pc.extractBookContent(book)

	for _, want := range []string{"BOOK_SECTION_TITLE", "BOOK_SUBSECTION_TITLE"} {
		if !strings.Contains(got, want) {
			t.Fatalf("extractBookContent dropped section title %q from whole-book analysis input. Got:\n%s", want, got)
		}
	}
}
