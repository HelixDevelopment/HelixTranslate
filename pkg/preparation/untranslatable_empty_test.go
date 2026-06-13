package preparation

import "testing"

// TestIsUntranslatable_EmptyTermDoesNotMatchEverything proves that a single
// empty-string untranslatable term from the LLM must NOT cause every string to
// be classified untranslatable. strings.Contains(x, "") is always true, so an
// empty term would otherwise suppress translation of the book title and all
// other metadata.
func TestIsUntranslatable_EmptyTermDoesNotMatchEverything(t *testing.T) {
	pat := &PreparationAwareTranslator{
		preparationResult: &PreparationResult{
			FinalAnalysis: ContentAnalysis{
				UntranslatableTerms: []UntranslatableTerm{
					{Term: ""},        // malformed/empty entry an LLM can emit
					{Term: "Voldemort"}, // a genuine untranslatable term
				},
			},
		},
	}

	// Unrelated text must NOT be flagged just because an empty term exists.
	if pat.isUntranslatable("My Ordinary Book Title") {
		t.Fatalf("empty untranslatable term wrongly matched unrelated text — " +
			"every title would be treated as untranslatable and left untranslated")
	}

	// A real term must still match (whitespace-only must also not match-all).
	if !pat.isUntranslatable("the wizard Voldemort returns") {
		t.Fatalf("genuine untranslatable term 'Voldemort' was not matched")
	}

	patWS := &PreparationAwareTranslator{
		preparationResult: &PreparationResult{
			FinalAnalysis: ContentAnalysis{
				UntranslatableTerms: []UntranslatableTerm{{Term: "   "}},
			},
		},
	}
	if patWS.isUntranslatable("Some Title") {
		t.Fatalf("whitespace-only untranslatable term wrongly matched unrelated text")
	}
}
