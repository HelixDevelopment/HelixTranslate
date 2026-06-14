package verification

import (
	"context"
	"testing"

	"digital.vasic.translator/pkg/ebook"
)

// RED: polishChapter ranges `for i := range original.Sections` but indexes
// translated.Sections[i]. When the translated chapter legitimately has FEWER
// sections than the original (a section translated to empty / merged, or a
// parser producing fewer sections), `&translated.Sections[i]` panics with
// index-out-of-range, crashing the whole polish run. The prior wave fixed the
// SAME class of bug at the chapter level in multipass.go but left the
// section/subsection level here unguarded.
func TestPolishChapter_FewerTranslatedSectionsNoPanic(t *testing.T) {
	// Empty providers => polishSection never makes an LLM call; the loop's
	// indexing of translated.Sections[i] happens BEFORE any content check, so
	// this isolates the bounds bug with no network/mock dependency.
	bp := &BookPolisher{config: PolishingConfig{}}

	original := &ebook.Chapter{
		Title: "Chapter One",
		Sections: []ebook.Section{
			{Title: "Sec A", Content: "alpha"},
			{Title: "Sec B", Content: "beta"}, // original has 2 sections
		},
	}
	translated := &ebook.Chapter{
		Title: "Chapter One",
		Sections: []ebook.Section{
			{Title: "Sec A", Content: "alfa"}, // translated has only 1
		},
	}

	report := NewPolishingReport(bp.config)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("polishChapter panicked on fewer translated sections "+
				"(original=%d, translated=%d): %v",
				len(original.Sections), len(translated.Sections), r)
		}
	}()

	if err := bp.polishChapter(context.Background(), original, translated, 1, report); err != nil {
		t.Fatalf("polishChapter returned error: %v", err)
	}
}

// RED: polishSectionRecursive ranges `for i := range original.Subsections` but
// indexes translated.Subsections[i]. Same index-out-of-range crash when the
// translated section has fewer subsections than the original.
func TestPolishSectionRecursive_FewerTranslatedSubsectionsNoPanic(t *testing.T) {
	bp := &BookPolisher{config: PolishingConfig{}}

	original := &ebook.Section{
		Title:   "Parent",
		Content: "parent body",
		Subsections: []ebook.Section{
			{Title: "Sub A", Content: "a"},
			{Title: "Sub B", Content: "b"}, // original has 2 subsections
		},
	}
	translated := &ebook.Section{
		Title:   "Parent",
		Content: "parent body",
		Subsections: []ebook.Section{
			{Title: "Sub A", Content: "a"}, // translated has only 1
		},
	}

	report := NewPolishingReport(bp.config)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("polishSectionRecursive panicked on fewer translated "+
				"subsections (original=%d, translated=%d): %v",
				len(original.Subsections), len(translated.Subsections), r)
		}
	}()

	if err := bp.polishSectionRecursive(
		context.Background(), original, translated, "Chapter 1, Section 1", report,
	); err != nil {
		t.Fatalf("polishSectionRecursive returned error: %v", err)
	}
}
