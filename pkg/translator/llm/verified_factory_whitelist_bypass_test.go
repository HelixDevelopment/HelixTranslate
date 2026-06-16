package llm

// §11.4.138 permanent regression guard for the §11.4.153-video-wave-1 defect:
// the VerifiedFactory / BestTranslator / BestTranslatorFunc path (the release-
// critical DEFAULT translate path used by unified-translator + gRPC) REJECTED a
// verifier-selected model when its id was not in the static `ValidModels`
// whitelist — e.g. novita "Sao10K/L3-8B-Stheno-v3.2" failed with
// "model '…' is not valid for provider 'novita'". A model that came through the
// LLMsVerifier is ALREADY proven working, so the static whitelist MUST NOT
// reject it on this path (the bridge Invoke/BestClient path correctly bypassed
// the whitelist; this constructor path did not — a §11.4.108 coverage gap the
// unit suite missed because it only ever registered models whose ids happened
// to be in `ValidModels`, e.g. "gpt-4").
//
// §11.4.115 polarity switch: with RED_MODE=1 the test reproduces the defect on
// the pre-fix artifact (asserts the whitelist-rejection error is present);
// with RED_MODE=0 (the default — the standing GREEN regression guard) it
// asserts the verifier-selected model is accepted and a real translator is
// constructed. Flip via the RED_MODE env var:
//
//	RED_MODE=1 go test -run TestVerifiedFactory_AcceptsVerifierSelectedModel ./pkg/translator/llm/   # reproduce on broken artifact
//	          go test -run TestVerifiedFactory_AcceptsVerifierSelectedModel ./pkg/translator/llm/   # GREEN guard (default)

import (
	"context"
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/selection"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// redMode reports whether the test runs in defect-reproduction mode (RED_MODE=1)
// rather than the default standing-GREEN-guard mode.
func redMode() bool {
	return strings.TrimSpace(os.Getenv("RED_MODE")) == "1"
}

// registerVerifierSelectedOffWhitelistModel registers ONE verified, selectable
// model whose id is deliberately NOT in `ValidModels` for its provider — the
// exact shape of the §11.4.153-wave-1 defect (novita / Sao10K-class). It is
// selectable under DefaultConfig (threshold 0.0): verified + can-see-code +
// affirmative + OverallScore > 0.
func registerVerifierSelectedOffWhitelistModel(f *VerifiedFactory) verifier.Model {
	m := verifier.Model{
		ID:                  "Sao10K/L3-8B-Stheno-v3.2",
		ProviderID:          "novita",
		Name:                "L3-8B-Stheno (verifier-selected)",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        0.91,
		Capabilities:        map[string]bool{"streaming": true},
	}
	// Sanity: the model id must be ABSENT from the static whitelist, otherwise
	// the test would not exercise the bypass at all (it would be a no-op pass).
	for _, valid := range ValidModels[ProviderNovita] {
		if valid == m.ID {
			panic("test precondition broken: chosen model id is in ValidModels[novita] — pick an off-whitelist id")
		}
	}
	f.RegisterModel(m)
	return m
}

// TestVerifiedFactory_AcceptsVerifierSelectedModel is the §11.4.138 permanent
// guard. It exercises CreateTranslatorWithFallback — the REAL path the release-
// critical default translate path (unified-translator BestTranslator + gRPC
// BestTranslatorFunc) uses — with a verifier-selected, off-whitelist model.
func TestVerifiedFactory_AcceptsVerifierSelectedModel(t *testing.T) {
	factory := NewVerifiedFactory(verifier.DefaultConfig())
	m := registerVerifierSelectedOffWhitelistModel(factory)

	// A real (non-empty) key so construction reaches — and on the pre-fix code,
	// is rejected by — the model-whitelist gate rather than failing earlier on a
	// missing key. The value is a placeholder; no network call is made by
	// construction (NewOpenAIClient/NewNovitaClient only validate + store).
	factory.SetKeyResolver(func(providerID string) string { return "test-key-novita" })

	tr, fallbacks, err := factory.CreateTranslatorWithFallback(
		context.Background(),
		selection.TaskRequirements{SourceLang: "ru", TargetLang: "sr"},
	)

	if redMode() {
		// RED: on the BROKEN (pre-fix) artifact the static whitelist rejects the
		// verifier-selected model. Capture that as the defect-present evidence.
		require.Error(t, err, "RED_MODE: pre-fix code MUST reject the verifier-selected off-whitelist model")
		assert.Contains(t, strings.ToLower(err.Error()), "is not valid for provider",
			"RED_MODE: the rejection MUST be the stale-ValidModels-whitelist error")
		assert.Contains(t, err.Error(), m.ID,
			"RED_MODE: the rejection MUST name the verifier-selected model id")
		return
	}

	// GREEN guard: the verifier-selected model MUST be accepted and a real
	// translator constructed. The defect would resurface as a non-nil err here.
	require.NoError(t, err,
		"verifier-selected model %q (provider %q) must NOT be rejected by the static ValidModels whitelist", m.ID, m.ProviderID)
	require.NotNil(t, tr, "a translator must be constructed for the verifier-selected model")
	require.NotNil(t, fallbacks, "fallback slice must be non-nil (may be empty)")
	// The constructed translator must report the SELECTED provider, proving it is
	// the verified model that was materialized (not silently swapped).
	assert.Equal(t, "llm-novita", tr.GetName(),
		"constructed translator must report the verifier-selected provider")
}

// TestVerifiedFactory_StillRejectsUnverifiedExplicitModel is the NEGATIVE half
// of the §11.4.146 extend step: the fix must NOT weaken validation for the
// EXPLICIT / non-verifier construction path. A model built directly through
// NewLLMTranslatorWithConfig (no verifier provenance) with an off-whitelist id
// MUST still be rejected by the static whitelist.
func TestVerifiedFactory_StillRejectsUnverifiedExplicitModel(t *testing.T) {
	_, err := NewLLMTranslatorWithConfig(TranslationConfig{
		Provider: "novita",
		Model:    "Sao10K/L3-8B-Stheno-v3.2", // off-whitelist, no verifier provenance
		APIKey:   "test-key-novita",
	})
	require.Error(t, err, "explicit (non-verifier) off-whitelist model must still be rejected — whitelist not weakened")
	assert.Contains(t, strings.ToLower(err.Error()), "is not valid for provider")
}
