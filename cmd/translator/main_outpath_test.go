package main

import "testing"

// generateOutputFilename must strip the input extension regardless of case and
// pick the right suffix (.fb2 -> _sr.epub, everything else -> _translated.epub).
// Before the fix, an UPPERCASE/mixed-case extension was lowercased before
// TrimSuffix, so it was NOT stripped and produced names like "Book.FB2_sr.epub".
func TestGenerateOutputFilename_CaseInsensitiveExt(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase fb2", "/b/war.fb2", "/b/war_sr.epub"},
		{"uppercase FB2", "/b/War.FB2", "/b/War_sr.epub"},
		{"mixed Fb2", "/b/Tale.Fb2", "/b/Tale_sr.epub"},
		{"lowercase epub other", "/b/story.epub", "/b/story_translated.epub"},
		{"uppercase EPUB other", "/b/Story.EPUB", "/b/Story_translated.epub"},
		{"no ext", "/b/plain", "/b/plain_translated.epub"},
		{"dotted basename fb2", "/b/my.book.v2.FB2", "/b/my.book.v2_sr.epub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateOutputFilename(tt.input); got != tt.want {
				t.Fatalf("generateOutputFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
