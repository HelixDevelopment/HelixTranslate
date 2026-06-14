package coordination

import (
	"context"
	"testing"
)

// newTieCoordinator builds a coordinator with an exact 2-2 split:
// instances 1+2 return "Aaa", instances 3+4 return "Bbb". requiredAgreement=4
// so all four vote. Both translations tie at count=2. The mockTranslator
// (defined in multi_llm_test.go) returns its single fixed response on every
// call, so the ONLY source of non-determinism in the result is the tie-break
// logic inside TranslateWithConsensus itself.
func newTieCoordinator() *MultiLLMCoordinator {
	mk := func(id, resp string) *LLMInstance {
		return &LLMInstance{
			ID:         id,
			Translator: &mockTranslator{responses: []string{resp}},
			Available:  true,
		}
	}
	return &MultiLLMCoordinator{
		instances: []*LLMInstance{
			mk("inst-1", "Aaa"),
			mk("inst-2", "Aaa"),
			mk("inst-3", "Bbb"),
			mk("inst-4", "Bbb"),
		},
		currentIndex: 0,
	}
}

// TestTranslateWithConsensus_TieIsDeterministic asserts that an exact vote tie
// resolves to a STABLE, documented winner across many runs. The documented
// tie-break rule is: highest count wins; on a count tie, the
// lexicographically-smallest translation wins ("Aaa" < "Bbb").
//
// RED on the pre-fix code: the winner is chosen by iterating a Go map, whose
// order is randomized, so across N runs both "Aaa" and "Bbb" appear -> the
// run-to-run set of winners has size 2 and/or the winner is not the
// lexicographically-smallest -> FAIL (non-determinism demonstrated).
//
// GREEN after the fix: every run returns "Aaa".
func TestTranslateWithConsensus_TieIsDeterministic(t *testing.T) {
	const iterations = 200
	winners := make(map[string]int)

	for i := 0; i < iterations; i++ {
		c := newTieCoordinator()
		got, err := c.TranslateWithConsensus(context.Background(), "src", "", 4)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		winners[got]++
	}

	if len(winners) != 1 {
		t.Fatalf("non-deterministic tie-break: across %d runs the winner varied: %v "+
			"(consensus must pick one stable winner on a tie)", iterations, winners)
	}

	// The single winner must be the documented tie-break value.
	for w := range winners {
		if w != "Aaa" {
			t.Fatalf("tie-break picked %q, want lexicographically-smallest %q", w, "Aaa")
		}
	}
}
