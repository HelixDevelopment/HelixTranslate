package llm

import (
	"strings"
	"testing"
)

// TestPromptEmptySourceExplicitSerbianTargetDoesNotClaimRussian is the
// §11.4.146 STEP 1 reproduce-first RED test for a genuine wrong-prompt defect in
// isRussianToSerbian (llm.go:622-632).
//
// Root cause (FACT): isRussianToSerbian treats an EMPTY source language as
// "Russian" (srcRU := src == "Russian" || src == ""). When a caller configures
// an EXPLICIT non-Russian/unknown nothing-but-target pair — e.g. TargetLang="sr"
// while SourceLang is left blank because the source language was auto-detected /
// not plumbed through — the function returns true and createTranslationPrompt
// emits the primary Russian→Serbian Ekavica prompt whose first sentences say
// "translate the following Russian text". The actual input is NOT Russian, so
// the LLM is told the wrong source language and the translation degrades.
//
// The true no-config default (BOTH source AND target empty) MUST still resolve
// to the legacy Russian→Serbian default — that case is intentional and is guarded
// by TestPromptEmptyConfigDefaultsToRussianSerbian. This test asserts the
// DIFFERENT case: empty source + an EXPLICIT target.
func TestPromptEmptySourceExplicitSerbianTargetDoesNotClaimRussian(t *testing.T) {
	// Source intentionally empty (auto-detect / unset), target explicitly Serbian.
	lt := newPromptTranslator("", "sr", "cyrillic")

	prompt := lt.createTranslationPrompt("Hello world", "Literary text")
	lower := strings.ToLower(prompt)

	// The prompt MUST NOT assert the input is Russian when the source is unset.
	if strings.Contains(lower, "russian to serbian translation") {
		t.Errorf("empty-source + explicit Serbian target MUST NOT emit the "+
			"'Russian to Serbian translation' prompt (claims input is Russian); got:\n%s", prompt)
	}
	if strings.Contains(lower, "following russian text") {
		t.Errorf("empty-source + explicit Serbian target MUST NOT instruct "+
			"'translate the following Russian text'; got:\n%s", prompt)
	}

	// It MUST still target Serbian (target was explicitly configured).
	if !strings.Contains(lower, "serbian") {
		t.Errorf("explicit Serbian target MUST keep Serbian as the target language; got:\n%s", prompt)
	}

	// Sanity: text and context still embedded.
	if !strings.Contains(prompt, "Hello world") {
		t.Errorf("prompt MUST contain the source text; got:\n%s", prompt)
	}
}
