package language

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDetector_RussianWithShortI is a reproduce-first regression guard for the
// bug where plain Russian text containing the very common letter 'й' (short I)
// was misclassified as Bulgarian.
//
// Root cause: detectCyrillicLanguage classified 'й' as a Bulgarian-specific
// character. 'й' is one of the most common letters in Russian. For Russian text
// that lacks ё/ы/э and any Russian trigger word, russianScore was 0 while a
// single 'й' produced bulgarianScore > 0, so the function returned Bulgarian.
//
// Wrong language detection => wrong target language => wrong translation.
func TestDetector_RussianWithShortI(t *testing.T) {
	detector := NewDetector(nil)

	cases := []struct {
		name string
		text string
	}{
		{"War and Peace title", "Война и мир"},
		{"Good day", "Хороший день"},
		{"Big and strong bear", "Большой и сильный медведь"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := detector.Detect(context.Background(), tc.text)
			assert.NoError(t, err)
			assert.Equal(t, Russian, result,
				"Russian text %q with 'й' must detect as Russian, not %s", tc.text, result.Name)
		})
	}
}
