package llm

import "testing"

// TestDeepSeek_V4Models_Accepted is the §11.4.135 regression guard for the
// stale DeepSeek allowlist defect.
//
// HISTORY (the bug): ValidModels[ProviderDeepSeek] listed only the legacy
// {"deepseek-chat","deepseek-coder"}, while the live DeepSeek /models endpoint
// (verified 2026-06-14) returns the current flagship models deepseek-v4-flash
// and deepseek-v4-pro. deepseek.go HARD-REJECTS any unlisted model, so a user
// requesting the current model got "model 'deepseek-v4-pro' is not valid for
// DeepSeek" — a real user-impacting staleness defect (§11.4.150). v4-flash was
// proven to genuinely translate ("Good morning" -> "Добро јутро") this session.
//
// MUTATION PROOF (§1.1): remove deepseek-v4-flash / deepseek-v4-pro from the
// allowlist and the corresponding subtest FAILs (the constructor rejects it).
func TestDeepSeek_V4Models_Accepted(t *testing.T) {
	currentModels := []string{"deepseek-v4-flash", "deepseek-v4-pro"}
	for _, m := range currentModels {
		t.Run(m, func(t *testing.T) {
			_, err := NewDeepSeekClient(TranslationConfig{
				Provider: "deepseek",
				APIKey:   "test-key",
				Model:    m,
			})
			if err != nil {
				t.Errorf("current DeepSeek model %q rejected by allowlist (stale): %v", m, err)
			}
		})
	}

	// Backcompat: the legacy alias must remain accepted (still works at runtime;
	// §11.4.122 no silent removal of an existing capability).
	t.Run("deepseek-chat backcompat", func(t *testing.T) {
		if _, err := NewDeepSeekClient(TranslationConfig{
			Provider: "deepseek", APIKey: "test-key", Model: "deepseek-chat",
		}); err != nil {
			t.Errorf("legacy deepseek-chat must remain accepted: %v", err)
		}
	})
}
