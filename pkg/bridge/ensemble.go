package bridge

import (
	"context"

	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/translator"
)

// EnsembleFactory returns a context-only closure that yields the provider-diverse
// verified translators (one per distinct provider, strongest-first) for the given
// task. Its signature is exactly the structural shape of the per-component
// EnsembleTranslatorFactory seams (coordination / preparation / verification —
// each declares `func(ctx context.Context) ([]translator.Translator, error)`), so
// it can be passed directly to NewMultiLLMCoordinatorWithFactory,
// NewPreparationCoordinatorWithFactory, and NewBookPolisherWithFactory WITHOUT
// those packages importing the bridge (§11.4.28 decoupling — the seam is a plain
// function value).
//
// This is the SINGLE adapter the R-1 redirect uses to source ensemble components
// from the LLMsVerifier bridge instead of the built-in per-provider
// NewLLMTranslator discovery. No API key is captured or logged (§11.4.10): the
// closure only calls ProviderDiverseTranslators, which returns key-free
// translators.
func (b *Bridge) EnsembleFactory(task selection.TaskRequirements) func(context.Context) ([]translator.Translator, error) {
	return func(ctx context.Context) ([]translator.Translator, error) {
		return b.ProviderDiverseTranslators(ctx, task)
	}
}

// BestTranslatorFunc returns a context-only closure that yields the single
// strongest verified translator for the task (no fallback chain). It is the
// non-ensemble single-translator redirect helper for sites that need ONE
// translator.Translator from the bridge rather than the provider-diverse set.
// The fallback chain that BestTranslator also returns is discarded here; callers
// needing it call BestTranslator directly. No API key is captured or logged.
func (b *Bridge) BestTranslatorFunc(task selection.TaskRequirements) func(context.Context) (translator.Translator, error) {
	return func(ctx context.Context) (translator.Translator, error) {
		tr, _, err := b.BestTranslator(ctx, task)
		return tr, err
	}
}
