package main

import "testing"

// TestResolveLanguageCodes_HonorsSourceTargetFlags is the regression guard for
// the wrong-output bug where cmd/preparation-translator hardcoded the language
// CODES to source="ru" / target="sr":
//
//	sourceLanguage := language.Language{Code: "ru", Name: *sourceLang}
//	targetLanguage := language.Language{Code: "sr", Name: *targetLang}
//
// Those codes drive (a) the translator's TranslationConfig.SourceLang/TargetLang
// which build the LLM translation-direction prompt, and (b) the output book's
// Metadata.Language tag. So a user running `-source English -target Spanish`
// (the program's own DEFAULTS!) got a Russian→Serbian prompt and an EPUB tagged
// language "sr" — the -source/-target flags only ever changed the cosmetic Name.
//
// §11.4.115 polarity: with the fix wired (resolveLanguageCodes maps the flag
// values through language.ParseLanguage) this is the GREEN regression-guard. On
// the pre-fix code the resolver did not exist; reverting resolveLanguageCodes to
// the hardcoded "ru"/"sr" pair makes every non-Russian/non-Serbian assertion
// below FAIL (mutation proof).
func TestResolveLanguageCodes_HonorsSourceTargetFlags(t *testing.T) {
	tests := []struct {
		name           string
		source, target string
		wantSrcCode    string
		wantTgtCode    string
	}{
		{
			name:        "program default English->Spanish resolves to en->es (not ru->sr)",
			source:      "English",
			target:      "Spanish",
			wantSrcCode: "en",
			wantTgtCode: "es",
		},
		{
			name:        "lowercase names resolve",
			source:      "german",
			target:      "french",
			wantSrcCode: "de",
			wantTgtCode: "fr",
		},
		{
			name:        "ISO codes pass through",
			source:      "en",
			target:      "ja",
			wantSrcCode: "en",
			wantTgtCode: "ja",
		},
		{
			name:        "the historical ru->sr pair still resolves correctly (no regression)",
			source:      "Russian",
			target:      "Serbian",
			wantSrcCode: "ru",
			wantTgtCode: "sr",
		},
		{
			name:        "unknown language falls back to the trimmed input code (no hardcoded sr)",
			source:      "Klingon",
			target:      "  Dothraki  ",
			wantSrcCode: "Klingon",
			wantTgtCode: "Dothraki",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, tgt := resolveLanguageCodes(tt.source, tt.target)
			if src.Code != tt.wantSrcCode {
				t.Errorf("source code for %q = %q, want %q (the -source flag is being ignored / hardcoded)",
					tt.source, src.Code, tt.wantSrcCode)
			}
			if tgt.Code != tt.wantTgtCode {
				t.Errorf("target code for %q = %q, want %q (the -target flag is being ignored / hardcoded)",
					tt.target, tgt.Code, tt.wantTgtCode)
			}
		})
	}
}
