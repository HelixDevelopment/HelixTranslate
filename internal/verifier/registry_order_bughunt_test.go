package verifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegistryFilterVerifiedDeterministicOrder is a RED regression test
// (Wave bug-hunt). FACT: FilterVerified/ListModels iterated the backing
// map[string]Model, whose iteration order Go randomizes per run, so the
// returned slice order was non-deterministic — flaky for any order-sensitive
// consumer and for the existing TestRegistryFilterVerified assertion. The
// contract MUST be deterministic: highest OverallScore first, ID as tiebreak.
// Running the registration in varied orders must always yield the same result.
func TestRegistryFilterVerifiedDeterministicOrder(t *testing.T) {
	mk := func(id string, score float64) Model {
		return Model{
			ID:                  id,
			VerificationStatus:  "verified",
			CanSeeCode:          true,
			AffirmativeResponse: true,
			OverallScore:        score,
		}
	}

	// Insert in an order that differs from the expected score-desc order.
	r := NewRegistry()
	r.AddModel(mk("claude-3", 5.0))
	r.AddModel(mk("gpt-4", 9.0))
	r.AddModel(mk("mistral", 7.0))

	for i := 0; i < 50; i++ {
		got := r.FilterVerified(0)
		require.Len(t, got, 3)
		assert.Equal(t, "gpt-4", got[0].ID, "score-desc: highest first")
		assert.Equal(t, "mistral", got[1].ID)
		assert.Equal(t, "claude-3", got[2].ID, "score-desc: lowest last")
	}

	// Tie-break by ID when scores are equal.
	r2 := NewRegistry()
	r2.AddModel(mk("zeta", 8.0))
	r2.AddModel(mk("alpha", 8.0))
	for i := 0; i < 50; i++ {
		got := r2.FilterVerified(0)
		require.Len(t, got, 2)
		assert.Equal(t, "alpha", got[0].ID, "equal scores tie-break by ID asc")
		assert.Equal(t, "zeta", got[1].ID)
	}
}
