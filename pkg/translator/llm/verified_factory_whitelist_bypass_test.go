package llm

// §11.4.138 permanent regression guard for the §11.4.153-video-wave-1/wave-2
// defect: the VerifiedFactory / BestTranslator / BestTranslatorFunc path (the
// release-critical DEFAULT translate path used by unified-translator + gRPC)
// REJECTED a verifier-selected model when its id was not in the static
// `ValidModels` whitelist — e.g. novita "Sao10K/L3-8B-Stheno-v3.2" failed with
// "model '…' is not valid for provider 'novita'". A model that came through the
// LLMsVerifier is ALREADY proven working, so the static whitelist MUST NOT
// reject it on this path (the bridge Invoke/BestClient path correctly bypassed
// the whitelist; this constructor path did not — a §11.4.108 coverage gap the
// unit suite missed because it only ever registered models whose ids happened
// to be in `ValidModels`, e.g. "gpt-4").
//
// The wave-1 fix was PARTIAL: buildVerifiedTranslator fell back to the
// whitelist-rejecting NewLLMTranslatorWithConfig on ANY ProviderResolver.Resolve
// error — and Resolve errors for BOTH "provider unknown" AND "provider KNOWN but
// its *_API_KEY env var unset". So a provider whose key comes from config.json
// (env unset) STILL hit the reject path. The dev box masked it because
// NOVITA_API_KEY was set (len 46) → Resolve succeeded. The §11.4.142 review
// returned NO-GO. The complete fix: distinguish unknown (fallback) from
// known-but-key-missing-in-env (bypass when a key exists from ANY source) from
// known-but-no-key-anywhere (honest missing-key error, never a whitelist
// re-route).
//
// §11.4.50 ENV-INDEPENDENCE: every case controls NOVITA_API_KEY via t.Setenv so
// the RED→GREEN polarity flips on PRODUCT behaviour, not on the ambient env. The
// guard is GREEN on a clean/CI host with NO provider keys set, and RED (against
// the pre-fix artifact) reproduces the defect for the RIGHT reason.
//
// §11.4.115 polarity switch: with RED_MODE=1 the test reproduces the defect on
// the pre-fix artifact (asserts the whitelist-rejection error is present); with
// RED_MODE=0 (the default — the standing GREEN regression guard) it asserts the
// verifier-selected model is accepted and a real translator is constructed.
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
// BestTranslatorFunc) uses — with a verifier-selected, off-whitelist model whose
// key is supplied via the keyResolver (the unified-translator wires this to
// resolveProviderAPIKey, which can source from config.json). NOVITA_API_KEY is
// deterministically UNSET (t.Setenv) so the run reproduces the production
// "config key present, env key absent" scenario the review found, INDEPENDENT of
// the host's ambient environment (§11.4.50).
func TestVerifiedFactory_AcceptsVerifierSelectedModel(t *testing.T) {
	// §11.4.50: pin the env so the polarity flips on product behaviour, not on
	// whatever the host happens to have exported. Empty = env key absent.
	t.Setenv("NOVITA_API_KEY", "")

	factory := NewVerifiedFactory(verifier.DefaultConfig())
	m := registerVerifierSelectedOffWhitelistModel(factory)

	// The key arrives via the keyResolver ONLY (the config.json source on the real
	// path) — NOT via the env var (which is unset above). On the pre-fix code the
	// factory's resolver.Resolve() errors because NOVITA_API_KEY is unset, so the
	// code falls back to the whitelist-rejecting NewLLMTranslatorWithConfig even
	// though a usable key exists. No network call is made by construction.
	factory.SetKeyResolver(func(providerID string) string { return "config-sourced-key-novita" })

	tr, fallbacks, err := factory.CreateTranslatorWithFallback(
		context.Background(),
		selection.TaskRequirements{SourceLang: "ru", TargetLang: "sr"},
	)

	if redMode() {
		// RED: on the BROKEN (pre-fix) artifact the env-unset key makes Resolve
		// error → fallback → the static whitelist rejects the verifier-selected
		// model. Capture that as the defect-present evidence.
		require.Error(t, err, "RED_MODE: pre-fix code MUST reject the verifier-selected off-whitelist model when the env key is unset")
		assert.Contains(t, strings.ToLower(err.Error()), "is not valid for provider",
			"RED_MODE: the rejection MUST be the stale-ValidModels-whitelist error")
		assert.Contains(t, err.Error(), m.ID,
			"RED_MODE: the rejection MUST name the verifier-selected model id")
		return
	}

	// GREEN guard: with a config-sourced key (env unset), the verifier-selected
	// model MUST be accepted and a real translator constructed. The defect would
	// resurface as a non-nil err here.
	require.NoError(t, err,
		"verifier-selected model %q (provider %q) with a config-sourced key must NOT be rejected by the static ValidModels whitelist even when its env key is unset", m.ID, m.ProviderID)
	require.NotNil(t, tr, "a translator must be constructed for the verifier-selected model")
	require.NotNil(t, fallbacks, "fallback slice must be non-nil (may be empty)")
	// The constructed translator must report the SELECTED provider, proving it is
	// the verified model that was materialized (not silently swapped).
	assert.Equal(t, "llm-novita", tr.GetName(),
		"constructed translator must report the verifier-selected provider")
}

// TestVerifiedFactory_AcceptsVerifierSelectedModel_EnvKeyOnly proves the
// additive case still works: a verifier-selected off-whitelist model whose key
// comes ONLY from the env var (no keyResolver) is accepted — the original
// working scenario the fix must NOT regress (§11.4.146 extend). Env-controlled
// via t.Setenv so it is deterministic on any host.
func TestVerifiedFactory_AcceptsVerifierSelectedModel_EnvKeyOnly(t *testing.T) {
	t.Setenv("NOVITA_API_KEY", "env-sourced-key-novita")

	factory := NewVerifiedFactory(verifier.DefaultConfig())
	m := registerVerifierSelectedOffWhitelistModel(factory)
	// No keyResolver: the key must be picked up from the env via the resolver.

	tr, _, err := factory.CreateTranslatorWithFallback(
		context.Background(),
		selection.TaskRequirements{SourceLang: "ru", TargetLang: "sr"},
	)

	require.NoError(t, err,
		"verifier-selected model %q must be accepted when its env key is set (no config key)", m.ID)
	require.NotNil(t, tr)
	assert.Equal(t, "llm-novita", tr.GetName())
}

// TestVerifiedFactory_KnownProviderNoKeyAnywhere_HonestError is the §11.4.146
// extend + §11.4.6 case: a KNOWN provider with NO key from config OR env must
// produce an HONEST missing-key error naming the env var — NEVER a silent
// re-route through the whitelist (which would mask the real cause with a
// misleading "model not valid" error). Env-controlled deterministically.
func TestVerifiedFactory_KnownProviderNoKeyAnywhere_HonestError(t *testing.T) {
	t.Setenv("NOVITA_API_KEY", "")

	factory := NewVerifiedFactory(verifier.DefaultConfig())
	m := registerVerifierSelectedOffWhitelistModel(factory)
	// No keyResolver and no env key → no usable key anywhere.

	_, _, err := factory.CreateTranslatorWithFallback(
		context.Background(),
		selection.TaskRequirements{SourceLang: "ru", TargetLang: "sr"},
	)

	require.Error(t, err, "a known provider with no key anywhere must error honestly")
	low := strings.ToLower(err.Error())
	assert.Contains(t, low, "no api key",
		"the error MUST be the honest missing-key error, not the whitelist rejection")
	assert.NotContains(t, low, "is not valid for provider",
		"the missing-key case MUST NOT be reported as a whitelist rejection (that masks the real cause)")
	assert.Contains(t, err.Error(), "NOVITA_API_KEY",
		"the honest error MUST name the env var to set (the NAME only, never a value — §11.4.10)")
	_ = m
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
