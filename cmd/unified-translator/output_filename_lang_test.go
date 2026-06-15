package main

import "testing"

// TestGenerateOutputFilename_HonorsTargetLang is the reproduce-first
// (§11.4.115) regression guard for the auto-named-output wrong-language defect.
//
// HISTORY (the bug this guard catches): generateOutputFilename hardcoded the
// "_sr.epub" suffix and ignored the user's -target-lang. Running
//
//	unified-translator -i book.txt -target-lang fr   (no -o)
//
// produced book_sr.epub (+ book_sr_session_report.md) — a French translation
// silently labelled Serbian ("sr"). Proven at runtime: the mock-provider run
// logged `output=/tmp/probe_book_sr.epub` for `-target-lang fr`. The user-
// visible filename, the session report name, and any downstream consumer keying
// off the language tag all carried the wrong language — a §11.4 silent
// wrong-output / mislabelling defect.
//
// §11.4.120 reconciliation: the prior TestGenerateOutputFilename pinned
// "_sr.epub" using the DEFAULT target lang (sr), which is still correct. The
// fix threads targetLang through the signature; "sr" cases keep producing
// "_sr.epub", and a non-default target now produces the matching tag.
//
// MUTATION PROOF (§1.1): revert generateOutputFilename to hardcode "_sr.epub"
// (drop the targetLang into the suffix) and every non-"sr" case below FAILs.
func TestGenerateOutputFilename_HonorsTargetLang(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		targetLang string
		want       string
	}{
		{"french target", "/books/roman.txt", "fr", "/books/roman_fr.epub"},
		{"german target", "/books/buch.fb2", "de", "/books/buch_de.epub"},
		{"english target", "novel.epub", "en", "novel_en.epub"},
		{"serbian default still sr", "/books/war.fb2", "sr", "/books/war_sr.epub"},
		{"uppercase ext + fr", "/books/Story.EPUB", "fr", "/books/Story_fr.epub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateOutputFilename(tt.input, tt.targetLang); got != tt.want {
				t.Fatalf("generateOutputFilename(%q, %q) = %q, want %q",
					tt.input, tt.targetLang, got, tt.want)
			}
		})
	}
}
