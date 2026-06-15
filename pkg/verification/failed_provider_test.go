package verification

import "testing"

// TestPolishSection_FailedProviderZeroValueCorruptsConsensus reproduces a
// genuine product defect in polishSection: when a provider's verifyWithLLM
// returns an error, its slot in the `verifications` slice is left as the
// zero-value llmVerification{} (SpiritScore=0, PolishedText="") and that
// phantom entry is fed into buildConsensus.
//
// Two real consequences, both proven here against the consensus aggregator that
// polishSection feeds:
//
//  1. SCORE DILUTION: the failed provider contributes a 0.0 to every score
//     average even though it never produced a verification — pulling the
//     section's quality scores toward zero.
//
//  2. CONTENT CORRUPTION: the failed provider's empty PolishedText ("") is
//     counted as a distinct "polished version". With enough failures to reach
//     MinConsensus, the empty string WINS consensus and the section's
//     translated content is wiped to "".
//
// This test simulates exactly the slice polishSection constructs when 2 of 3
// providers error and 1 succeeds suggesting "no change".
func TestPolishSection_FailedProviderZeroValueCorruptsConsensus(t *testing.T) {
	bp := &BookPolisher{
		config: PolishingConfig{
			Providers:    []string{"ok", "fail1", "fail2"},
			MinConsensus: 2,
		},
	}

	const translatedText = "Здраво свете"

	// What polishSection builds: index 0 succeeded (no change), indices 1 & 2
	// errored and remain zero-value llmVerification{} (PolishedText == "").
	verifications := []llmVerification{
		{
			Provider:        "ok",
			SpiritScore:     0.9,
			LanguageScore:   0.9,
			ContextScore:    0.9,
			VocabularyScore: 0.9,
			PolishedText:    translatedText, // suggests no change
		},
		{}, // fail1 — zero value, PolishedText == ""
		{}, // fail2 — zero value, PolishedText == ""
	}

	result := bp.buildConsensus("sec", "Chapter 1", "Здраво свете (orig)", translatedText, verifications)

	// (1) Content corruption: the empty string must NOT become the polished
	// output. A successful provider said "no change", so the content must be
	// preserved.
	if result.PolishedText != translatedText {
		t.Errorf("CONTENT CORRUPTION: translated content was wiped/changed by failed providers; "+
			"want %q, got %q", translatedText, result.PolishedText)
	}

	// (2) Score dilution: only one provider actually scored (all 0.9). Failed
	// providers must be excluded, so the average must be 0.9, not diluted by
	// phantom zeros (0.9+0+0)/3 = 0.3.
	if result.SpiritScore < 0.85 {
		t.Errorf("SCORE DILUTION: SpiritScore diluted by failed-provider zeros; "+
			"want ~0.9 (only real verifications counted), got %.3f", result.SpiritScore)
	}
}
