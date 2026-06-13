package main

import (
	"strings"
	"testing"
)

// hasCyrillic reports whether s contains any codepoint in the Cyrillic block
// U+0400–U+04FF.
func hasCyrillic(s string) bool {
	for _, r := range s {
		if r >= 0x0400 && r <= 0x04FF {
			return true
		}
	}
	return false
}

// TestNormalizeScriptLatinLeavesNoCyrillic is the W10 RED-baseline test
// (§11.4.115). It feeds known Cyrillic Serbian text through the latin
// normalization path the `-script latin` flag is supposed to trigger and
// asserts the output contains NO Cyrillic codepoints. Before the wiring fix
// this FAILs because the translated output is never converted to Latin.
func TestNormalizeScriptLatinLeavesNoCyrillic(t *testing.T) {
	cyrillicInput := "Пример текста на српском. Љубав је лепа ствар. Ђорђе је добар човек."

	out := normalizeScript(cyrillicInput, "latin")

	if hasCyrillic(out) {
		t.Fatalf("`-script latin` left Cyrillic codepoints in output: %q", out)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatalf("normalizeScript returned empty/blank output for non-empty input")
	}
}

// TestNormalizeScriptLatinTransliterates verifies the exact Latin form.
func TestNormalizeScriptLatinTransliterates(t *testing.T) {
	out := normalizeScript("Пример текста на српском", "latin")
	const want = "Primer teksta na srpskom"
	if out != want {
		t.Fatalf("latin normalization wrong: got %q want %q", out, want)
	}
}

// TestNormalizeScriptCyrillic verifies `-script cyrillic` converts Latin
// Serbian to Cyrillic.
func TestNormalizeScriptCyrillic(t *testing.T) {
	out := normalizeScript("Ljubav je lepa stvar", "cyrillic")
	if !hasCyrillic(out) {
		t.Fatalf("`-script cyrillic` produced no Cyrillic codepoints: %q", out)
	}
	const want = "Љубав је лепа ствар"
	if out != want {
		t.Fatalf("cyrillic normalization wrong: got %q want %q", out, want)
	}
}

// TestNormalizeScriptLatinIdempotent ensures already-Latin input is unchanged
// (no double-conversion, no failure mode introduced).
func TestNormalizeScriptLatinIdempotent(t *testing.T) {
	const latin = "Primer teksta na srpskom"
	out := normalizeScript(latin, "latin")
	if out != latin {
		t.Fatalf("latin normalization not idempotent on Latin input: got %q want %q", out, latin)
	}
	if hasCyrillic(out) {
		t.Fatalf("idempotent latin path introduced Cyrillic: %q", out)
	}
}

// TestNormalizeScriptCyrillicIdempotent ensures already-Cyrillic input is
// unchanged under `-script cyrillic`.
func TestNormalizeScriptCyrillicIdempotent(t *testing.T) {
	const cyr = "Љубав је лепа ствар"
	out := normalizeScript(cyr, "cyrillic")
	if out != cyr {
		t.Fatalf("cyrillic normalization not idempotent on Cyrillic input: got %q want %q", out, cyr)
	}
}

// TestNormalizeScriptUnknownPassthrough ensures an unrecognised script value
// leaves text untouched (no panic, no data loss).
func TestNormalizeScriptUnknownPassthrough(t *testing.T) {
	const in = "Пример текста на српском"
	if got := normalizeScript(in, ""); got != in {
		t.Fatalf("empty script should pass through unchanged: got %q", got)
	}
	if got := normalizeScript(in, "klingon"); got != in {
		t.Fatalf("unknown script should pass through unchanged: got %q", got)
	}
}
