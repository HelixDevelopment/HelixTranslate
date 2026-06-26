package llm

import (
	"strings"
	"testing"
)

// newPromptTranslator builds an LLMTranslator whose embedded BaseTranslator
// carries the given language-pair / script config, so createTranslationPrompt
// can read the real configuration at the prompt site. No network client is
// constructed — createTranslationPrompt is a pure function of (config, text,
// context).
func newPromptTranslator(sourceLang, targetLang, script string) *LLMTranslator {
	return &LLMTranslator{
		BaseTranslator: NewBaseTranslator(TranslationConfig{
			SourceLang: sourceLang,
			TargetLang: targetLang,
			Script:     script,
		}),
		provider: ProviderMock,
	}
}

// TestPromptHonorsConfiguredLanguagePair is the §11.4.146 STEP 1 reproduce-first
// RED test for the hardcoded "Russian to Serbian" prompt defect. On the pre-fix
// code createTranslationPrompt ignores SourceLang/TargetLang and always emits
// the Russian→Serbian Ekavica instruction, so an en→fr translation receives the
// WRONG prompt. After the fix the prompt MUST reference the configured pair and
// MUST NOT instruct Russian→Serbian.
func TestPromptHonorsConfiguredLanguagePair(t *testing.T) {
	lt := newPromptTranslator("en", "fr", "")

	prompt := lt.createTranslationPrompt("Hello world", "Literary text")
	lower := strings.ToLower(prompt)

	// MUST NOT hardcode Russian→Serbian for a non-RU→SR pair.
	if strings.Contains(lower, "russian to serbian") {
		t.Errorf("en→fr prompt MUST NOT instruct 'Russian to Serbian'; got prompt:\n%s", prompt)
	}
	if strings.Contains(lower, "ekavica") || strings.Contains(prompt, "екавица") {
		t.Errorf("en→fr prompt MUST NOT carry Serbian Ekavica guidance; got prompt:\n%s", prompt)
	}

	// MUST reference the configured language pair (full language names).
	if !strings.Contains(lower, "english") {
		t.Errorf("en→fr prompt MUST reference the source language 'English'; got prompt:\n%s", prompt)
	}
	if !strings.Contains(lower, "french") {
		t.Errorf("en→fr prompt MUST reference the target language 'French'; got prompt:\n%s", prompt)
	}

	// The text and context must still be embedded.
	if !strings.Contains(prompt, "Hello world") {
		t.Errorf("prompt MUST contain the source text; got prompt:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Literary text") {
		t.Errorf("prompt MUST contain the context; got prompt:\n%s", prompt)
	}
}

// TestPromptRussianToSerbianPreservesEkavica is the §11.4.124 no-regression
// guard: the primary Russian→Serbian path MUST keep its exact Ekavica /
// pure-Serbian-vocabulary / Cyrillic guidance.
func TestPromptRussianToSerbianPreservesEkavica(t *testing.T) {
	lt := newPromptTranslator("ru", "sr", "")
	prompt := lt.createTranslationPrompt("Привет мир", "Literary text")

	mustContain := []string{
		"Russian to Serbian translation",
		"Ekavica dialect (екавица)",
		"pure Serbian vocabulary",
		"Serbian Cyrillic script (ћирилица)",
		"mleko (not mlijeko)",
		"avion",
		"Serbian translation (Ekavica only):",
	}
	for _, frag := range mustContain {
		if !strings.Contains(prompt, frag) {
			t.Errorf("ru→sr prompt MUST preserve Ekavica fragment %q; got prompt:\n%s", frag, prompt)
		}
	}
}

// TestPromptRussianToSerbianLatinScript proves the Serbian-target Latin-script
// override switches the script line while keeping Ekavica guidance.
func TestPromptRussianToSerbianLatinScript(t *testing.T) {
	lt := newPromptTranslator("ru", "sr", "latin")
	prompt := lt.createTranslationPrompt("Привет", "ctx")

	if !strings.Contains(prompt, "Serbian Latin script (latinica)") {
		t.Errorf("ru→sr latin prompt MUST request Serbian Latin script; got prompt:\n%s", prompt)
	}
	if strings.Contains(prompt, "Serbian Cyrillic script (ћирилица)") {
		t.Errorf("ru→sr latin prompt MUST NOT request Cyrillic; got prompt:\n%s", prompt)
	}
	// Ekavica guidance still preserved regardless of script.
	if !strings.Contains(prompt, "Ekavica dialect (екавица)") {
		t.Errorf("ru→sr latin prompt MUST still carry Ekavica guidance; got prompt:\n%s", prompt)
	}
}

// TestPromptGenericPairsAndScript fans out across several non-RU→SR pairs and
// script settings (§11.4.146 STEP 3 extend-to-all-cases). Each case asserts the
// correct configured languages appear, no Russian→Serbian leakage, and the
// script instruction matches the configured Script.
func TestPromptGenericPairsAndScript(t *testing.T) {
	cases := []struct {
		name         string
		source       string
		target       string
		script       string
		wantSrc      string // language name expected in prompt (lowercased match)
		wantTgt      string
		wantScriptIn string // substring the script line must contain
	}{
		{"en→fr default script", "en", "fr", "", "english", "french", "natural writing system for French"},
		{"en→es latin script", "en", "es", "latin", "english", "spanish", "Latin script"},
		{"de→ru cyrillic target", "de", "ru", "cyrillic", "german", "russian", "Cyrillic script"},
		{"en→de latin script", "en", "de", "latin", "english", "german", "Latin script"},
		{"full-name input fr→en", "French", "English", "", "french", "english", "natural writing system for English"},
		{"unknown target code passthrough", "en", "xyz", "", "english", "xyz", "writing system for xyz"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lt := newPromptTranslator(tc.source, tc.target, tc.script)
			prompt := lt.createTranslationPrompt("Some text", "ctx")
			lower := strings.ToLower(prompt)

			if strings.Contains(lower, "russian to serbian") {
				t.Errorf("%s MUST NOT instruct Russian→Serbian; got:\n%s", tc.name, prompt)
			}
			if strings.Contains(lower, "ekavica") {
				t.Errorf("%s MUST NOT carry Ekavica guidance; got:\n%s", tc.name, prompt)
			}
			if !strings.Contains(lower, tc.wantSrc) {
				t.Errorf("%s MUST reference source %q; got:\n%s", tc.name, tc.wantSrc, prompt)
			}
			if !strings.Contains(lower, tc.wantTgt) {
				t.Errorf("%s MUST reference target %q; got:\n%s", tc.name, tc.wantTgt, prompt)
			}
			if !strings.Contains(prompt, tc.wantScriptIn) {
				t.Errorf("%s MUST contain script instruction %q; got:\n%s", tc.name, tc.wantScriptIn, prompt)
			}
			if !strings.Contains(prompt, "Some text") {
				t.Errorf("%s MUST embed the source text; got:\n%s", tc.name, prompt)
			}
		})
	}
}

// TestPromptEmptyConfigDefaultsToRussianSerbian guards the no-config / zero-value
// default: an LLMTranslator with no configured pair MUST fall back to the legacy
// Russian→Serbian Ekavica prompt (this is what TestCreateTranslationPrompt and
// the historical behaviour rely on).
func TestPromptEmptyConfigDefaultsToRussianSerbian(t *testing.T) {
	// Zero-value (no BaseTranslator at all) — must not panic, must default.
	zero := &LLMTranslator{}
	p1 := zero.createTranslationPrompt("x", "")
	if !strings.Contains(p1, "Russian to Serbian translation") {
		t.Errorf("zero-value translator MUST default to Russian→Serbian; got:\n%s", p1)
	}

	// Empty langs but present config — same default.
	empty := newPromptTranslator("", "", "")
	p2 := empty.createTranslationPrompt("x", "")
	if !strings.Contains(p2, "Ekavica dialect (екавица)") {
		t.Errorf("empty-lang config MUST default to Ekavica guidance; got:\n%s", p2)
	}
}

// TestPromptNamesUnderspecifiedCyrillicAndEasternLangs is the regression test
// for the defect where languageName() lacked entries for Belarusian (be),
// Kazakh (kk) and Persian (fa). The generic prompt then fell back to the raw
// code ("...into natural, idiomatic be"), and the model guessed — producing
// Bulgarian for "be" in particular (caught by the independent §11.4.141 review).
// The prompt MUST name the real language so the model is unambiguously steered.
func TestPromptNamesUnderspecifiedCyrillicAndEasternLangs(t *testing.T) {
	cases := map[string]string{"be": "Belarusian", "kk": "Kazakh", "fa": "Persian"}
	for code, name := range cases {
		t.Run(code, func(t *testing.T) {
			p := newPromptTranslator("en", code, "").createTranslationPrompt("hello", "")
			if !strings.Contains(p, name) {
				t.Errorf("prompt for en->%s must name %q; got:\n%s", code, name, p)
			}
			// Belarusian must NOT be confused with Bulgarian.
			if code == "be" && strings.Contains(p, "Bulgarian") {
				t.Errorf("en->be prompt must not mention Bulgarian")
			}
		})
	}
}
