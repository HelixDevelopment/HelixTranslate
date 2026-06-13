package language

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

// BUG A — FormatLanguageCode byte-truncates a multibyte first character, returning
// invalid UTF-8 (e.g. "中文" -> "\xe4\xb8"). It must return either a clean 2-rune
// prefix or empty, never invalid UTF-8.
//
// RED on current code: FormatLanguageCode("中文") yields invalid UTF-8.
func TestFormatLanguageCode_NeverReturnsInvalidUTF8(t *testing.T) {
	inputs := []string{
		"中文",      // CJK, multibyte first rune
		"Русский", // Cyrillic
		"日本語",     // CJK
		"français",
		"eng",
		"en-US",
		"  EN  ",
		"é",  // single multibyte rune
		"ñç", // two multibyte runes
	}
	for _, in := range inputs {
		out := FormatLanguageCode(in)
		if !utf8.ValidString(out) {
			t.Errorf("FormatLanguageCode(%q) = %q which is INVALID UTF-8 (% x)", in, out, out)
		}
	}
}

// FormatLanguageCode must still produce the expected ISO prefixes for normal inputs
// (regression guard — fix must not change the happy path the existing test relies on).
func TestFormatLanguageCode_ASCIIHappyPathUnchanged(t *testing.T) {
	cases := map[string]string{
		"en":      "en",
		"EN":      "en",
		"eng":     "en",
		"english": "en",
		"":        "",
		"xyz":     "xy",
	}
	for in, want := range cases {
		if got := FormatLanguageCode(in); got != want {
			t.Errorf("FormatLanguageCode(%q) = %q, want %q", in, got, want)
		}
	}
}

// BUG B — detectHeuristic samples text[:1000] by BYTES, not runes. Multi-byte
// scripts (Cyrillic=2B, CJK=3B) cost more bytes per rune, so a byte-truncated
// sample systematically UNDER-COUNTS a trailing multi-byte script and can flip the
// detected language to the wrong one for genuinely long documents.
//
// Proven flip (captured): 400 Latin runes + 800 Cyrillic runes is genuinely
// Cyrillic-dominant (600 cyr / 400 lat over the first 1000 runes => Russian), but
// byte[:1000] sees only 300 cyrillic runes (400 latin bytes + 600 cyrillic bytes =
// 300 cyrillic runes) => a near-balanced mix => mis-detected as English.
//
// RED on current code: returns English; want Russian.
func TestDetect_LongCyrillicAfterLatin_NotMisdetectedAsEnglish(t *testing.T) {
	detector := NewDetector(nil)

	// Use 'ы' (Russian-specific marker) so the Cyrillic branch resolves to Russian.
	longDoc := strings.Repeat("a", 400) + strings.Repeat("ы", 800)

	result, err := detector.Detect(context.Background(), longDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != Russian {
		t.Errorf("Detect(400 Latin + 800 Cyrillic) = %v, want Russian "+
			"(byte-truncation under-counts the trailing Cyrillic body)", result)
	}
}

// Companion: a long CJK body after a Latin prefix must still detect as Chinese.
// '世' is 3 bytes, so byte-truncation under-counts it even harder than Cyrillic.
func TestDetect_LongCJKAfterLatin_NotMisdetectedAsEnglish(t *testing.T) {
	detector := NewDetector(nil)

	// 300 Latin runes (300B) + 800 CJK runes (2400B). First 1000 runes =
	// 300 Latin + 700 CJK => CJK fraction 0.7 > 0.3 => Chinese.
	// byte[:1000] = 300 Latin + ~233 CJK => CJK fraction ~0.43 but Latin large =>
	// can mis-route. Robust detector must return Chinese.
	longDoc := strings.Repeat("a", 300) + strings.Repeat("世", 800)

	result, err := detector.Detect(context.Background(), longDoc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != Chinese {
		t.Errorf("Detect(300 Latin + 800 CJK) = %v, want Chinese", result)
	}
}

// Robustness guard: the analysed sample must always be valid UTF-8, so a long
// document whose 1000-byte boundary lands mid-rune is never fed invalid UTF-8.
func TestDetect_LongText_NoPanicNoInvalidUTF8(t *testing.T) {
	detector := NewDetector(nil)
	inputs := []string{
		"a" + strings.Repeat("я", 700), // 2-byte runes, misaligned boundary
		"x" + strings.Repeat("世", 400), // 3-byte runes, misaligned boundary
		strings.Repeat("مرحبا ", 300),   // Arabic, multi-byte
	}
	for _, in := range inputs {
		// Must not panic; result must be a real (non-empty-code) language.
		res, err := detector.Detect(context.Background(), in)
		if err != nil {
			t.Errorf("unexpected error for long input: %v", err)
		}
		if res.Code == "" {
			t.Errorf("empty language code for long input %q...", in[:10])
		}
	}
}
