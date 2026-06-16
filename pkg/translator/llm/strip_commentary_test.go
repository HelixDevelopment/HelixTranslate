package llm

import (
	"os"
	"strings"
	"testing"
)

// TestEnhanceTranslation_StripsModelCommentary is the permanent §11.4.135
// regression guard for the LLM-commentary-contamination defect discovered by
// heavy real-service testing against the live nezha stack (2026-06-16).
//
// Root cause (FACT): createTranslationPrompt ends with "<Lang> translation:" and
// never instructs the model to return ONLY the translation; capable instruct
// models (observed: provider llm-novita on nezha) helpfully append an
// explanatory paragraph ("This translation maintains...", "(Note: ...)",
// "[Note: ...]") which the System passed through verbatim, polluting every
// translated paragraph an end user would receive.
//
// The fixtures below are the LITERAL real responses captured from the live
// nezha server-TLS :18443 /api/v1/translate endpoint (sink-side evidence,
// §11.4.69) — see qa-results/nezha_heavy_test_*/.
//
// §11.4.115 polarity: RED_MODE=1 (default in this guard set to 0) would assert
// the defect present on the pre-fix artifact; RED_MODE=0 is the standing GREEN
// guard asserting the commentary is ABSENT after the fix. Pre-fix, the raw
// string is returned unchanged (commentary present) → these asserts FAIL.
func TestEnhanceTranslation_StripsModelCommentary(t *testing.T) {
	lt := &LLMTranslator{}

	cases := []struct {
		name       string
		raw        string // real captured contaminated model output
		wantClean  string // the actual translation, commentary stripped
		mustNotHas []string
	}{
		{
			name:      "es_paragraph_commentary",
			raw:       "El anciano y el mar eran uno.\n\nThis translation aims to preserve the poetic, almost mystical tone of the original English text. The phrase \"The old man and the sea were one\" suggests a deep spiritual connection.",
			wantClean: "El anciano y el mar eran uno.",
			mustNotHas: []string{
				"This translation", "preserve the poetic", "spiritual connection",
			},
		},
		{
			name:      "sr_paren_note",
			raw:       "Dobar dan, moj prijatelj.\n\n(Note: I've translated \"Good morning\" as \"Dobar dan\", which is a common greeting in Serbia.)",
			wantClean: "Dobar dan, moj prijatelj.",
			mustNotHas: []string{
				"(Note:", "I've translated", "common greeting",
			},
		},
		{
			name:      "ru_bracket_note",
			raw:       "Быть или не быть, это вопрос.\n\n[Note: This is a direct, word-for-word translation of the famous opening line from William Shakespeare's Hamlet.]",
			wantClean: "Быть или не быть, это вопрос.",
			mustNotHas: []string{
				"[Note:", "word-for-word", "Shakespeare",
			},
		},
		{
			name:      "fr_this_translation_maintains",
			raw:       "Le savoir est le pouvoir.\n\nThis translation maintains the concise, aphoristic style of the original English phrase.",
			wantClean: "Le savoir est le pouvoir.",
			mustNotHas: []string{"This translation maintains", "aphoristic"},
		},
		{
			// Live-captured on nezha 2026-06-16 (docs/qa/nezha_coverage_*): the
			// model appended a parenthetical *style/dialect* aside that the
			// original keyword filter (only "note"/"translat") missed, leaking
			// English meta-commentary into a Serbian translation. §11.4.115 RED
			// fixture / §11.4.135 standing guard.
			name:      "sr_paren_dialect_aside",
			raw:       "Ја пијем кафу јутро.\n\n(Using Ekavica dialect and pure Serbian vocabulary as per guidelines)",
			wantClean: "Ја пијем кафу јутро.",
			mustNotHas: []string{"Using Ekavica", "dialect", "vocabulary", "guidelines"},
		},
		{
			name:      "en_bracket_style_aside",
			raw:       "The book is on the table.\n\n[Using formal register and standard vocabulary]",
			wantClean: "The book is on the table.",
			mustNotHas: []string{"Using formal", "register", "vocabulary"},
		},
		{
			name:      "clean_unchanged_no_commentary",
			raw:       "El anciano y el mar eran uno.",
			wantClean: "El anciano y el mar eran uno.",
		},
		{
			name:      "multi_paragraph_translation_preserved",
			raw:       "Primer párrafo de la novela.\n\nSegundo párrafo de la novela.",
			wantClean: "Primer párrafo de la novela.\n\nSegundo párrafo de la novela.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lt.enhanceTranslation(tc.raw, tc.raw)
			got = strings.TrimSpace(got)
			if got != tc.wantClean {
				t.Errorf("commentary not stripped correctly:\n got  = %q\n want = %q", got, tc.wantClean)
			}
			for _, frag := range tc.mustNotHas {
				if strings.Contains(got, frag) {
					t.Errorf("commentary fragment leaked into output: %q\n full = %q", frag, got)
				}
			}
		})
	}
}

// TestEnhanceTranslation_StripCommentary_DoesNotEatRealParagraphs proves the
// stripper is NOT over-eager: a genuine multi-paragraph translation (no
// commentary markers) MUST pass through untouched. This is the anti-bluff guard
// that the fix does not silently truncate real content.
func TestEnhanceTranslation_StripCommentary_DoesNotEatRealParagraphs(t *testing.T) {
	lt := &LLMTranslator{}
	in := "Capítulo uno.\n\nEra una noche oscura y tormentosa.\n\nEl viento aullaba."
	got := strings.TrimSpace(lt.enhanceTranslation(in, in))
	if got != in {
		t.Errorf("real multi-paragraph translation was mutated:\n got  = %q\n want = %q", got, in)
	}
}

// TestEnhanceTranslation_StripCommentary_KeepsBenignTrailingParenthetical proves
// the widened parenthetical-aside detector does NOT over-strip a genuine trailing
// parenthetical that is real translated content (no meta-commentary signal words).
// Anti-bluff false-positive guard for the sr_paren_dialect_aside fix.
func TestEnhanceTranslation_StripCommentary_KeepsBenignTrailingParenthetical(t *testing.T) {
	lt := &LLMTranslator{}
	in := "Stigao je u grad.\n\n(I bio je veoma umoran.)"
	got := strings.TrimSpace(lt.enhanceTranslation(in, in))
	if got != in {
		t.Errorf("benign trailing parenthetical content was wrongly stripped:\n got  = %q\n want = %q", got, in)
	}
}

// keep os import meaningful for a potential RED_MODE polarity switch.
var _ = os.Getenv
