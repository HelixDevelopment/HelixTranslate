package llm

import (
	"context"
	"fmt"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
	"digital.vasic.translator/internal/verifier/selection"
)

// verifiedMaxTokens bounds the completion budget for a translator materialized
// from a verifier-selected model. Sized to fit comfortably within the smallest
// common verified-model context window (8192 total) alongside the translation
// prompt, so no verified model is rejected with an HTTP 400 max_tokens overflow,
// while still allowing multi-paragraph translation segments (larger inputs are
// chunked by translateWithRetry). Mirrors the bridge's invokeMaxTokens rationale
// for the translator path.
const verifiedMaxTokens = 4096

// VerifiedFactory creates LLM translators using LLMsVerifier-verified models.
// This is the CONST-034-compliant factory that only permits verified models.
// When a verifier.Client is provided, the factory queries LLMsVerifier as the
// single source of truth (SSOT). When unavailable, it falls back to the local
// registry with a warning.
type VerifiedFactory struct {
	selector    *selection.Engine
	config      *verifier.Config
	client      *verifier.Client
	keyResolver func(providerID string) string
	resolver    *verifier.ProviderResolver
}

// NewVerifiedFactory creates a verified translator factory.
func NewVerifiedFactory(cfg *verifier.Config) *VerifiedFactory {
	registry := verifier.NewRegistry()
	scoringEngine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     cfg.ScoringWeights.ResponseSpeed,
		CostEffectiveness: cfg.ScoringWeights.CostEffectiveness,
		ModelEfficiency:   cfg.ScoringWeights.ModelEfficiency,
		Capability:        cfg.ScoringWeights.Capability,
		Recency:           cfg.ScoringWeights.Recency,
	})
	selector := selection.NewEngine(registry, scoringEngine, cfg)

	return &VerifiedFactory{
		selector: selector,
		config:   cfg,
		resolver: verifier.NewProviderResolver(),
	}
}

// SetClient configures the LLMsVerifier API client. When set, the factory
// will fetch verified models from LLMsVerifier at runtime instead of relying
// solely on manually registered models.
func (f *VerifiedFactory) SetClient(client *verifier.Client) {
	f.client = client
}

// refreshRegistry fetches verified models from LLMsVerifier and populates
// the local registry. Returns true if fresh data was loaded.
func (f *VerifiedFactory) refreshRegistry(ctx context.Context) (bool, error) {
	if f.client == nil {
		return false, nil
	}

	models, err := f.client.GetVerifiedModels(ctx)
	if err != nil {
		return false, fmt.Errorf("LLMsVerifier unreachable: %w", err)
	}

	// Clear and repopulate registry with canonical data from SSOT
	for _, m := range models {
		f.selector.GetRegistry().AddModel(m)
	}
	return true, nil
}

// SetKeyResolver configures the API key resolver for provider-specific keys.
func (f *VerifiedFactory) SetKeyResolver(resolver func(providerID string) string) {
	f.keyResolver = resolver
}

func (f *VerifiedFactory) resolveAPIKey(providerID string) string {
	if f.keyResolver != nil {
		return f.keyResolver(providerID)
	}
	return ""
}

// CreateTranslator builds an LLM translator for the best verified model
// matching the given task requirements.
func (f *VerifiedFactory) CreateTranslator(ctx context.Context, task selection.TaskRequirements) (*LLMTranslator, error) {
	// Attempt to refresh from LLMsVerifier SSOT before selection
	if _, err := f.refreshRegistry(ctx); err != nil {
		// If we have no models at all, fail hard. If we have cached models,
		// warn and continue with stale data.
		if len(f.selector.GetRegistry().ListModels()) == 0 {
			return nil, fmt.Errorf("no verified models available and LLMsVerifier unreachable: %w", err)
		}
	}

	model, err := f.selector.SelectModel(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("no verified model available: %w", err)
	}

	return f.buildVerifiedTranslator(model.ProviderID, model.ID, task)
}

// buildVerifiedTranslator materializes a translator for a model that has ALREADY
// passed LLMsVerifier verification (§11.4.108 runtime-signature: a model the
// verifier selected is proven working). It MUST NOT be rejected by the static
// `ValidModels` whitelist (the §11.4.153-wave-1 / §11.4.138 defect: the default
// translate path died on verifier-selected ids like novita
// "Sao10K/L3-8B-Stheno-v3.2"). It reuses the SAME whitelist-immune
// materialization the bridge Invoke/BestClient path uses: resolve the provider
// to its OpenAI-compatible BaseURL, then build the generic OpenAI-compatible
// client with the provider id as a DELEGATION MARKER (Provider != "" && !=
// "openai" → NewOpenAIClient skips its whitelist). The whitelist remains in
// force for the EXPLICIT / non-verifier construction path
// (NewLLMTranslatorWithConfig) — validation is NOT weakened for unverified
// models.
//
// Resolution is deterministic (§11.4.6): if the resolver cannot materialize the
// provider (not in the canonical envProviderSpecs table — e.g. a natively-cased
// provider such as "anthropic"/"gemini" whose endpoint the generic client does
// not serve), it falls back to the normal NewLLMTranslatorWithConfig path so
// those providers keep their existing, correct construction. No API key is
// logged (§11.4.10).
func (f *VerifiedFactory) buildVerifiedTranslator(
	providerID, modelID string, task selection.TaskRequirements,
) (*LLMTranslator, error) {
	apiKey := f.resolveAPIKey(providerID)

	// Full config so the prompt builder honours the requested language pair.
	transConfig := TranslationConfig{
		Provider:   providerID,
		Model:      modelID,
		APIKey:     apiKey,
		SourceLang: task.SourceLang,
		TargetLang: task.TargetLang,
	}

	resolver := f.resolver
	if resolver == nil {
		resolver = verifier.NewProviderResolver()
	}
	rp, rerr := resolver.Resolve(providerID)
	if rerr != nil {
		// Provider not materializable via the OpenAI-compatible resolver path
		// (e.g. not in envProviderSpecs, or its key env var is unset). Fall back
		// to the standard constructor so natively-cased providers and the honest
		// missing-key error are preserved. This is NOT the whitelist-bypass path.
		return NewLLMTranslatorWithConfig(transConfig)
	}

	// Delegation marker: the concrete provider id (NOT "openai") so the generic
	// OpenAI-compatible client skips its hardcoded model whitelist; the genuine
	// OpenAI provider keeps "openai" so its real model whitelist still applies.
	providerMarker := rp.ProviderID
	if providerMarker == "" || providerMarker == "openai" {
		providerMarker = rp.FactoryProvider
	}

	// Prefer the key the factory's resolver already produced (it is the
	// authoritative source the bridge wires); fall back to the env-resolved key.
	clientKey := apiKey
	if clientKey == "" {
		clientKey = rp.APIKey
	}

	client, err := NewOpenAIClient(TranslationConfig{
		Provider: providerMarker, // delegation marker → whitelist-immune
		Model:    modelID,
		APIKey:   clientKey,
		BaseURL:  rp.BaseURL,
		// Bound the completion budget. The generic OpenAI-compatible client
		// defaults max_tokens to 8192 when unset, which a verifier-selected model
		// with an 8192-TOTAL context window rejects (HTTP 400: completion 8192 +
		// prompt > 8192 context). Verified models span a wide range of context
		// sizes, so cap the completion conservatively — same rationale as the
		// bridge's invokeMaxTokens cap — leaving headroom for the prompt while
		// still sized for multi-paragraph translation segments. Larger inputs are
		// chunked by translateWithRetry.
		MaxTokens: verifiedMaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("build verified translator for %s/%s: %w", providerID, modelID, err)
	}

	// Wrap the whitelist-immune client in an LLMTranslator carrying the full
	// config (the prompt builder reads SourceLang/TargetLang/Script from it).
	// The reported provider is the verifier-selected provider id, not the
	// underlying client's hardcoded "openai".
	return &LLMTranslator{
		BaseTranslator: NewBaseTranslator(transConfig),
		provider:       Provider(providerID),
		client:         &markedVerifiedClient{LLMClient: client, provider: providerID},
	}, nil
}

// markedVerifiedClient wraps the generic OpenAI-compatible client so its
// GetProviderName reports the verifier-selected provider id (e.g. "novita")
// rather than the underlying *OpenAIClient hardcoded "openai". Mirrors the
// bridge's markedClient so both materialization paths report identically.
type markedVerifiedClient struct {
	LLMClient
	provider string
}

func (m *markedVerifiedClient) GetProviderName() string { return m.provider }

// CreateTranslatorWithFallback builds a translator with fallback chain.
func (f *VerifiedFactory) CreateTranslatorWithFallback(ctx context.Context, task selection.TaskRequirements) (*LLMTranslator, []string, error) {
	// Attempt to refresh from LLMsVerifier SSOT before selection
	if _, err := f.refreshRegistry(ctx); err != nil {
		if len(f.selector.GetRegistry().ListModels()) == 0 {
			return nil, nil, fmt.Errorf("no verified models available and LLMsVerifier unreachable: %w", err)
		}
	}

	primary, err := f.selector.SelectModel(ctx, task)
	if err != nil {
		return nil, nil, fmt.Errorf("no verified model available: %w", err)
	}

	translator, err := f.buildVerifiedTranslator(primary.ProviderID, primary.ID, task)
	if err != nil {
		return nil, nil, err
	}

	// Build fallback chain
	fallback, err := f.selector.SelectFallback(primary.ID, task)
	if err != nil {
		// No fallback available, but primary is usable
		return translator, []string{}, nil
	}

	return translator, []string{fallback.ID}, nil
}

// IsModelVerified checks if a specific model passes CONST-034 verification.
func (f *VerifiedFactory) IsModelVerified(modelID string) bool {
	_, ok := f.selector.GetRegistry().GetModel(modelID)
	return ok
}

// ListVerifiedModels returns all models that pass verification.
func (f *VerifiedFactory) ListVerifiedModels() []verifier.Model {
	return f.selector.GetRegistry().FilterVerified(f.config.MinScoreThreshold)
}

// RegisterModel adds a verified model to the factory's registry.
func (f *VerifiedFactory) RegisterModel(model verifier.Model) {
	f.selector.GetRegistry().AddModel(model)
}

// RegisterProvider adds a provider configuration to the factory's registry.
func (f *VerifiedFactory) RegisterProvider(cfg verifier.ProviderConfig) {
	f.selector.GetRegistry().RegisterProvider(cfg)
}
