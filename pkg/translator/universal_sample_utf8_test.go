package translator

import (
	"context"
	"testing"
	"unicode/utf8"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"

	"github.com/stretchr/testify/require"
)

// sampleSpyDetector records the exact text it is asked to detect so the test
// can assert the language-detection sample is well-formed UTF-8.
type sampleSpyDetector struct {
	gotSample string
}

func (s *sampleSpyDetector) DetectLanguage(_ context.Context, text string) (string, error) {
	s.gotSample = text
	return "en", nil // force LLM path so the raw sample reaches the detector
}

// noopTranslator is an identity Translator so TranslateBook runs without network.
type noopTranslator struct{}

func (noopTranslator) Translate(_ context.Context, text, _ string) (string, error) {
	return text, nil
}

func (noopTranslator) TranslateWithProgress(_ context.Context, text, _ string, _ *events.EventBus, _ string) (string, error) {
	return text, nil
}

func (noopTranslator) GetStats() TranslationStats { return TranslationStats{} }
func (noopTranslator) GetName() string            { return "noop" }

// TestTranslateBook_DetectionSampleIsValidUTF8 proves that the 2000-byte sample
// cut in TranslateBook does not split a multi-byte rune, which would hand
// invalid UTF-8 to the language detector. Ebooks are overwhelmingly non-ASCII
// (Cyrillic, CJK, accented Latin), so a byte-index cut routinely lands
// mid-rune. This is a data-correctness defect at the detection boundary.
func TestTranslateBook_DetectionSampleIsValidUTF8(t *testing.T) {
	// Build a book whose ExtractText() output is > 2000 bytes of 2-byte
	// Cyrillic runes so that byte index 2000 straddles a rune boundary.
	bigCyrillic := ""
	for i := 0; i < 1500; i++ {
		bigCyrillic += "б" // U+0431, 2 bytes each -> ~3000 bytes
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "наслов"}, // non-ASCII title
		Chapters: []ebook.Chapter{
			{
				Title: "поглавље",
				Sections: []ebook.Section{
					{Content: bigCyrillic},
				},
			},
		},
	}

	full := book.ExtractText()
	require.Greater(t, len(full), 2000, "precondition: extracted text must exceed 2000 bytes")
	// The naive byte-cut at 2000 must actually be invalid UTF-8 for this
	// fixture, otherwise the test would be a tautology.
	require.False(t, utf8.ValidString(full[:2000]),
		"precondition: naive byte-cut at 2000 must split a rune for this fixture")

	spy := &sampleSpyDetector{}
	det := language.NewDetector(spy)

	ut := NewUniversalTranslator(
		noopTranslator{},
		det,
		language.Language{}, // empty source -> triggers detection
		language.English,
	)

	err := ut.TranslateBook(context.Background(), book, nil, "sess")
	require.NoError(t, err)

	require.NotEmpty(t, spy.gotSample, "detector must have received a sample")
	require.True(t, utf8.ValidString(spy.gotSample),
		"detection sample handed to the language detector must be valid UTF-8 "+
			"(byte-index slicing split a multi-byte rune)")
}
