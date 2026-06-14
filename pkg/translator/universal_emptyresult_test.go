package translator

import (
	"context"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/language"

	"github.com/stretchr/testify/mock"
)

// TestTranslateBook_EmptySuccessfulResultDoesNotWipeContent reproduces a
// data-loss defect: when the underlying translator returns a SUCCESSFUL
// (nil error) but empty/whitespace-only result for non-empty source text,
// the universal translator must NOT overwrite the original content with the
// empty string. A "successful" empty translation that silently destroys the
// chapter/section/title/metadata text is a §11.4 data-loss bluff — the run
// reports success while the book is gutted.
func TestTranslateBook_EmptySuccessfulResultDoesNotWipeContent(t *testing.T) {
	mockT := new(MockTranslator)

	// Every translation call "succeeds" but returns an empty string.
	mockT.On("TranslateWithProgress",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", nil)

	ut := NewUniversalTranslator(
		mockT,
		nil, // no detector
		language.Language{Code: "ru", Name: "Russian"},
		language.Language{Code: "sr", Name: "Serbian"},
	)

	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       "Original Title",
			Description: "Original Description",
			Language:    "ru",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Original Chapter Title",
				Sections: []ebook.Section{
					{
						Title:   "Original Section Title",
						Content: "Original section content that must survive.",
						Subsections: []ebook.Section{
							{
								Title:   "Original Subsection Title",
								Content: "Original subsection content.",
							},
						},
					},
				},
			},
		},
	}

	if err := ut.TranslateBook(context.Background(), book, nil, "sess-1"); err != nil {
		t.Fatalf("TranslateBook returned error: %v", err)
	}

	// Assert no field was silently emptied by a successful-but-empty result.
	checks := []struct {
		name string
		got  string
	}{
		{"metadata.Title", book.Metadata.Title},
		{"metadata.Description", book.Metadata.Description},
		{"chapter.Title", book.Chapters[0].Title},
		{"section.Title", book.Chapters[0].Sections[0].Title},
		{"section.Content", book.Chapters[0].Sections[0].Content},
		{"subsection.Title", book.Chapters[0].Sections[0].Subsections[0].Title},
		{"subsection.Content", book.Chapters[0].Sections[0].Subsections[0].Content},
	}
	for _, c := range checks {
		if c.got == "" {
			t.Errorf("%s was wiped to empty by a successful-but-empty translation (data loss)", c.name)
		}
	}
}
