package verification

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
)

// RED: verifySection records an untranslated block for the WHOLE section content
// AND, separately, one untranslated block per paragraph of that same content.
// calculateQualityScore then sums block.Length over ALL blocks, so a fully-
// untranslated multi-paragraph section contributes ~2x its character count to
// untranslatedChars while totalChars counts the content only once. The
// character-based quality score (1 - untranslatedChars/totalChars) is therefore
// driven below 0 and clamped to 0 — over-penalizing far past the true
// "100% untranslated => score 0" floor, and double-counting untranslated chars.
//
// Concretely: a single section whose content is entirely source-language (so the
// TRUE untranslated fraction is 100%, charScore should be exactly 0) should NOT
// produce untranslatedChars greater than totalChars.
func TestCalculateQualityScore_NoDoubleCountAcrossParagraphs(t *testing.T) {
	v := NewVerifier(language.Russian, language.English, events.NewEventBus(), "s")

	// Two Cyrillic paragraphs (each > 10 letters so isSourceLanguage triggers),
	// separated by a blank line so splitIntoParagraphs yields 2 paragraphs.
	para1 := "Это первый длинный абзац на русском языке."
	para2 := "Это второй длинный абзац на русском языке."
	content := para1 + "\n\n" + para2

	book := &ebook.Book{
		Chapters: []ebook.Chapter{
			{Sections: []ebook.Section{{Content: content}}},
		},
	}

	result, err := v.VerifyBook(context.Background(), book)
	if err != nil {
		t.Fatalf("VerifyBook error: %v", err)
	}

	// Sum what calculateQualityScore sums.
	untranslatedChars := 0
	for _, b := range result.UntranslatedBlocks {
		untranslatedChars += b.Length
	}
	totalChars := len(content)

	if untranslatedChars > totalChars {
		t.Fatalf("double-counted untranslated chars: untranslatedChars=%d > totalChars=%d "+
			"(blocks=%d) — section content counted once plus once per paragraph",
			untranslatedChars, totalChars, len(result.UntranslatedBlocks))
	}

	// And the score must be a sane 0 (fully untranslated), not artificially driven
	// negative-then-clamped by the double count.
	if result.QualityScore < 0 {
		t.Fatalf("QualityScore = %v, must be clamped >= 0", result.QualityScore)
	}
	_ = strings.TrimSpace
}
