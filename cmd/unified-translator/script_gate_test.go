package main

import "testing"

// TestApplyTargetScript_NonSerbianTargetNotTransliterated is an anti-bluff
// regression test (§11.4.115) for a defect surfaced by a real video recording:
// a real EN->Spanish DeepSeek translation came out as Spanish-in-Cyrillic
// gibberish ("Hola, mundo" -> "Хола, мундо") because the Serbian Cyrillic<->Latin
// script normalization (default -script cyrillic) was applied UNCONDITIONALLY to
// every target language. The Serbian converter must only run for a Serbian
// target; for any other language the LLM output is already in the correct script
// and MUST pass through unchanged.
func TestApplyTargetScript_NonSerbianTargetNotTransliterated(t *testing.T) {
	spanish := "Hola, mundo. Esto es una prueba real. El rápido zorro marrón."

	// Non-Serbian targets: the text must survive byte-for-byte (no transliteration),
	// regardless of the (Serbian-context) -script flag value.
	for _, target := range []string{"Spanish", "es", "French", "fr", "German", "English", "en"} {
		for _, sc := range []string{"cyrillic", "latin"} {
			got := applyTargetScript(spanish, target, sc)
			if got != spanish {
				t.Errorf("applyTargetScript(target=%q, script=%q) mangled non-Serbian text:\n got:  %q\n want: %q\n(the Serbian script converter must NOT run for non-Serbian targets)",
					target, sc, got, spanish)
			}
		}
	}
}

// TestApplyTargetScript_SerbianTargetStillConverts guards that the fix did NOT
// break the legitimate Serbian use case: a Serbian target with -script cyrillic
// still converts Serbian Latin -> Serbian Cyrillic.
func TestApplyTargetScript_SerbianTargetStillConverts(t *testing.T) {
	// "Ljubav" (Serbian Latin) -> "Љубав" (Serbian Cyrillic).
	for _, target := range []string{"sr", "Serbian", "serbian", "Serbian Cyrillic"} {
		got := applyTargetScript("Ljubav", target, "cyrillic")
		if got != "Љубав" {
			t.Errorf("applyTargetScript(target=%q, cyrillic) = %q, want Serbian-Cyrillic %q — Serbian conversion must still work",
				target, got, "Љубав")
		}
	}
	// Serbian target with -script latin yields Latin (idempotent on Latin input).
	if got := applyTargetScript("Ljubav", "sr", "latin"); got != "Ljubav" {
		t.Errorf("applyTargetScript(sr, latin) = %q, want %q", got, "Ljubav")
	}
}

// TestIsSerbianTarget covers the closed-set detector.
func TestIsSerbianTarget(t *testing.T) {
	for _, yes := range []string{"sr", "SR", "Serbian", "serbian", "srpski", "српски", "Serbian Latin", " serbian "} {
		if !isSerbianTarget(yes) {
			t.Errorf("isSerbianTarget(%q) = false, want true", yes)
		}
	}
	for _, no := range []string{"Spanish", "es", "English", "en", "Russian", "ru", "French", ""} {
		if isSerbianTarget(no) {
			t.Errorf("isSerbianTarget(%q) = true, want false", no)
		}
	}
}
