package verifier

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ResolvedProvider is the concrete, materializable description of a verified
// model's provider: everything NewLLMTranslatorWithConfig needs to build a real
// LLMClient. It closes the §3.3 load-bearing gap — a verified api.Model carries
// ONLY a ProviderID (a provider-spec id from the in-process pipeline, OR a
// numeric string from the HTTP server path) and NO api_key / base_url. The
// resolver materializes (factoryProvider, baseURL, apiKey) from the canonical
// envProviderSpecs table so EVERY verified provider can be turned into a client.
type ResolvedProvider struct {
	// ProviderID is the original provider id carried by the verified model
	// (e.g. "deepseek" from the in-process path, or "7" from the HTTP path).
	ProviderID string
	// FactoryProvider is the provider string NewLLMTranslatorWithConfig's switch
	// recognizes. For every OpenAI-compatible provider in envProviderSpecs this is
	// "openai" (the generic OpenAI-compatible client honours BaseURL), so a
	// provider the factory does not natively case (openrouter/together/fireworks/
	// nvidia/...) still materializes through the generic client.
	FactoryProvider string
	// BaseURL is the provider's OpenAI-compatible v1 root, threaded into
	// TranslationConfig.BaseURL so the generic client targets the right endpoint.
	BaseURL string
	// APIKey is the provider's key, read from its environment variable IN MEMORY
	// ONLY (§11.4.10) — never logged, never persisted.
	APIKey string
	// EnvVar is the environment-variable NAME the key is read from. Safe to log
	// (it is a name, not a value) so callers can report a missing key honestly.
	EnvVar string
}

// providerByID indexes envProviderSpecs by provider id for O(1) lookup. Built
// once from the single source of truth so the resolver never drifts from the
// pipeline's provider set.
var providerByID = func() map[string]envProviderSpec {
	m := make(map[string]envProviderSpec, len(envProviderSpecs))
	for _, s := range envProviderSpecs {
		m[s.ID] = s
	}
	return m
}()

// numericProviderIndex maps a numeric ProviderID (the HTTP server path emits
// fmt.Sprintf("%d", sm.ProviderID) — a number, NOT a factory-recognized name)
// onto an envProviderSpec by position in the deterministic, append-only
// envProviderSpecs list. The server assigns provider rows monotonically; this
// mapping is the documented bridge for the numeric-id case (§3.3 part 1). When
// the numeric id is out of range an honest error is returned — never a guess.
func numericProviderIndex(n int) (envProviderSpec, bool) {
	if n < 0 || n >= len(envProviderSpecs) {
		return envProviderSpec{}, false
	}
	return envProviderSpecs[n], true
}

// ProviderResolver maps a verified model's ProviderID onto a ResolvedProvider.
// It is the single materialization seam used by BOTH the in-process pipeline
// path (ProviderID is a provider-spec id like "deepseek") AND the HTTP server
// path (ProviderID is a numeric string like "7"). It reads provider metadata
// from envProviderSpecs and the API key from the environment (§11.4.10).
type ProviderResolver struct {
	// getenv allows tests to inject a key source without mutating the real
	// environment. Defaults to os.Getenv.
	getenv func(string) string
}

// NewProviderResolver builds a resolver reading keys from the real environment.
func NewProviderResolver() *ProviderResolver {
	return &ProviderResolver{getenv: os.Getenv}
}

// NewProviderResolverWithEnv builds a resolver reading keys from the supplied
// function — used by unit tests to avoid touching the process environment.
func NewProviderResolverWithEnv(getenv func(string) string) *ProviderResolver {
	if getenv == nil {
		getenv = os.Getenv
	}
	return &ProviderResolver{getenv: getenv}
}

// ErrProviderUnknown is returned (wrapped) by Resolve / ResolveProvider when a
// ProviderID is neither a known named provider nor a valid numeric index — i.e.
// the provider is genuinely unmaterializable via the OpenAI-compatible resolver
// path. Callers MUST distinguish this (legitimate fallback to the standard
// constructor) from ErrProviderKeyMissing (provider KNOWN but no key in the
// environment — which is NOT a fallback-to-whitelist condition). §11.4.6.
var ErrProviderUnknown = fmt.Errorf("verified model provider id is not a known provider and is not a numeric index")

// ErrProviderKeyMissing is returned (wrapped) by Resolve when a provider is
// KNOWN and fully materialized (BaseURL/EnvVar/FactoryProvider populated) but its
// API-key environment variable is unset. The ResolvedProvider is still returned
// populated (only APIKey empty) so a caller holding a key from another source
// (e.g. config.json) can proceed, while a caller with no key can report exactly
// which env var to set. errors.Is(err, ErrProviderKeyMissing) is the seam that
// keeps a config-keyed-but-env-unset provider OFF the whitelist-rejecting path.
var ErrProviderKeyMissing = fmt.Errorf("verified provider resolved but its API key env var is not set")

// ResolveProvider materializes a verified model's ProviderID into a
// ResolvedProvider WITHOUT requiring the API-key environment variable to be set.
// It is the key-agnostic core of Resolve: it answers "is this provider KNOWN and
// materializable?" and populates ProviderID/FactoryProvider/BaseURL/EnvVar (plus
// APIKey if the env var happens to be set), returning a wrapped ErrProviderUnknown
// ONLY when the provider is genuinely not in envProviderSpecs / out of numeric
// range. A KNOWN provider with an unset key env var is NOT an error here —
// distinguishing "unknown" from "known-but-key-missing" is the §11.4.138 review
// gap this method closes. No key is logged (§11.4.10).
func (r *ProviderResolver) ResolveProvider(providerID string) (ResolvedProvider, error) {
	id := strings.TrimSpace(providerID)
	if id == "" {
		return ResolvedProvider{}, fmt.Errorf("verified model has empty provider id: %w", ErrProviderUnknown)
	}

	spec, ok := providerByID[id]
	if !ok {
		// HTTP server path: ProviderID is a numeric string. Map by index.
		if n, convErr := strconv.Atoi(id); convErr == nil {
			spec, ok = numericProviderIndex(n)
			if !ok {
				return ResolvedProvider{}, fmt.Errorf(
					"verified model provider id %q is a numeric index out of range [0,%d): %w",
					id, len(envProviderSpecs), ErrProviderUnknown)
			}
		} else {
			return ResolvedProvider{}, fmt.Errorf(
				"verified model provider id %q: %w", id, ErrProviderUnknown)
		}
	}

	return ResolvedProvider{
		ProviderID:      id,
		FactoryProvider: factoryProviderFor(spec.ID),
		BaseURL:         spec.BaseURL,
		EnvVar:          spec.EnvVar,
		APIKey:          r.getenv(spec.EnvVar),
	}, nil
}

// Resolve materializes a verified model's ProviderID into a ResolvedProvider.
//
// Resolution order (§3.3 / §11.4.6 — deterministic, no guessing):
//  1. Named id (e.g. "deepseek") → looked up directly in envProviderSpecs.
//  2. Numeric id (e.g. "7", from the HTTP server path) → mapped by index into
//     the deterministic envProviderSpecs list.
//  3. Unknown id → ErrProviderUnknown naming the id (never a silent fallback).
//
// A resolved provider whose API-key environment variable is unset yields a
// ResolvedProvider with an empty APIKey and a wrapped ErrProviderKeyMissing error
// so the caller can report exactly which env var to set — never a silent empty-key
// client. Callers that hold a key from another source (config.json) MUST use
// ResolveProvider (key-agnostic) instead of treating this error as an unknown
// provider, per the §11.4.138 review gap. Behaviour is preserved for existing
// callers: a non-nil error still accompanies a key-missing or unknown provider;
// only the error is now typed (errors.Is-classifiable).
func (r *ProviderResolver) Resolve(providerID string) (ResolvedProvider, error) {
	resolved, err := r.ResolveProvider(providerID)
	if err != nil {
		return resolved, err
	}
	if resolved.APIKey == "" {
		return resolved, fmt.Errorf(
			"verified provider %q resolved but its API key env var %s is not set: %w",
			resolved.ProviderID, resolved.EnvVar, ErrProviderKeyMissing)
	}
	return resolved, nil
}

// factoryProviderFor returns the NewLLMTranslatorWithConfig provider string for
// an envProviderSpec id. Every OpenAI-compatible provider in envProviderSpecs is
// served by the generic OpenAI-compatible client (which honours BaseURL), so the
// factory provider is "openai" for all of them. This is intentional: providers
// the factory switch does not natively case (openrouter/together/fireworks/...)
// still materialize through the generic client, with the right base_url threaded
// in. The id list is the single source of truth (envProviderSpecs); a new
// provider added there is resolvable here with zero extra wiring.
func factoryProviderFor(specID string) string {
	return "openai"
}
