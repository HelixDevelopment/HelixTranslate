package llm

import "testing"

// TestEnhanceTranslation_CyrillicCapitalization is the permanent regression
// guard for the byte-indexed capitalization bug in enhanceTranslation:
// `enhanced[0]` / `original[0]` indexed UTF-8 bytes, so for multibyte first
// characters (Cyrillic, accented Latin — this translator's primary target
// scripts) the capitalization-restoration was silently dead. Root cause: byte
// indexing + ASCII-only isLower/isUpper/toUpper. Fix: decode the first rune and
// use the unicode package.
//
// RED proof (pre-fix): firstByte of "Привет"/"привет" is 0xD0 (208), a UTF-8
// lead byte, so isLower(208)==false → guard never fired → "привет свет" returned
// uncapitalized.
func TestEnhanceTranslation_CyrillicCapitalization(t *testing.T) {
	lt := &LLMTranslator{}

	cases := []struct {
		name     string
		original string
		input    string
		wantHead rune
	}{
		{"cyrillic", "Привет мир", "привет свет", 'П'},
		{"accented_latin", "École active", "école nouvelle", 'É'},
		{"ascii_still_works", "Hello world", "hello there", 'H'},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lt.enhanceTranslation(tc.original, tc.input)
			head := []rune(got)[0]
			if head != tc.wantHead {
				t.Errorf("enhanceTranslation(%q,%q): first rune = %q, want %q (full=%q)",
					tc.original, tc.input, head, tc.wantHead, got)
			}
		})
	}
}

// TestEnhanceTranslation_NoSpuriousCapitalization proves the guard does NOT
// capitalize when the source is itself lowercase (must not corrupt).
func TestEnhanceTranslation_NoSpuriousCapitalization(t *testing.T) {
	lt := &LLMTranslator{}
	got := lt.enhanceTranslation("привет мир", "привет свет")
	if []rune(got)[0] != 'п' {
		t.Errorf("must not capitalize when source first letter is lowercase; got %q", got)
	}
}
