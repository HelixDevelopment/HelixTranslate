package verification

import "testing"

// RED: buildConsensus selects the winning polished version by iterating a
// map[string]int (polishedVersions) and keeping the first entry that strictly
// exceeds the running maximum. When two distinct polished versions are agreed
// upon by an EQUAL number of providers (a tie at maxAgreement), Go map
// iteration order is randomized per run, so which version "wins" — and thus the
// PolishedText the verifier accepts and the Change it records — is
// nondeterministic. Same input, different verdict across runs: a §11.4.50
// determinism violation in the consensus/verdict path.
//
// Reproduce deterministically by running buildConsensus on identical input many
// times and asserting the chosen PolishedText is the same every time.
func TestBuildConsensus_TieIsDeterministic(t *testing.T) {
	bp := &BookPolisher{config: PolishingConfig{MinConsensus: 2}}

	makeVerifs := func() []llmVerification {
		// 4 providers: 2 vote "polish_A", 2 vote "polish_B"; both differ from the
		// current translation, so both tie at agreement == 2 (== MinConsensus).
		return []llmVerification{
			{Provider: "p1", PolishedText: "polish_A"},
			{Provider: "p2", PolishedText: "polish_A"},
			{Provider: "p3", PolishedText: "polish_B"},
			{Provider: "p4", PolishedText: "polish_B"},
		}
	}

	first := bp.buildConsensus("sec", "Chapter 1", "orig", "trans", makeVerifs())
	if first.Consensus != 2 {
		t.Fatalf("setup wrong: Consensus = %d, want 2 (a genuine tie at MinConsensus)", first.Consensus)
	}

	for i := 0; i < 200; i++ {
		got := bp.buildConsensus("sec", "Chapter 1", "orig", "trans", makeVerifs())
		if got.PolishedText != first.PolishedText {
			t.Fatalf("nondeterministic tie-break: iteration %d chose %q, first run chose %q "+
				"(map-iteration-order dependent verdict)", i, got.PolishedText, first.PolishedText)
		}
	}
}
