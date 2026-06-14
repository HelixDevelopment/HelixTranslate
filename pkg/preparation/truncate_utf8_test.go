package preparation

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestTruncateContent_CyrillicNoMidRuneCorruption reproduces the byte-vs-rune
// defect in truncateContent: it slices content[:maxChars] / content[:cutPoint]
// by BYTE offset. For Cyrillic (multi-byte UTF-8) text with no '.' or "\n\n"
// near the cut, cutPoint == maxChars and the slice lands mid-rune, producing an
// invalid trailing UTF-8 sequence (a broken half-character) that is then fed to
// the analysis LLM as part of the prompt.
func TestTruncateContent_CyrillicNoMidRuneCorruption(t *testing.T) {
	// 8000 × "Ћ" (U+040B, 2 bytes each) = 16000 bytes, all Cyrillic, no '.'/"\n\n".
	content := strings.Repeat("Ћ", 8000)
	// Odd maxChars guarantees the byte cut lands in the middle of a 2-byte rune.
	out := truncateContent(content, 15001)

	if !utf8.ValidString(out) {
		t.Fatalf("truncateContent produced invalid UTF-8 (mid-rune byte slice): "+
			"last rune of truncated body is RuneError; len(out)=%d", len(out))
	}
}
