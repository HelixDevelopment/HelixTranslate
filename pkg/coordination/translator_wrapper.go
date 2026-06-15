package coordination

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// MultiLLMTranslatorWrapper wraps MultiLLMCoordinator to implement the Translator interface
type MultiLLMTranslatorWrapper struct {
	Coordinator *MultiLLMCoordinator // Exported so CLI can access instance count
	config      translator.TranslationConfig
}

// NewMultiLLMTranslatorWrapper creates a new wrapper
func NewMultiLLMTranslatorWrapper(config translator.TranslationConfig, eventBus *events.EventBus, sessionID string) (*MultiLLMTranslatorWrapper, error) {
	return NewMultiLLMTranslatorWrapperWithConfig(config, eventBus, sessionID, false, false)
}

// NewMultiLLMTranslatorWrapperWithConfig creates a new wrapper with configuration options.
//
// Behaviour-preserving wrapper: it delegates to
// NewMultiLLMTranslatorWrapperWithFactory with a nil factory, so the default
// per-provider discovery path is provably identical to the prior implementation.
func NewMultiLLMTranslatorWrapperWithConfig(config translator.TranslationConfig, eventBus *events.EventBus, sessionID string, disableLocalLLMs bool, preferDistributed bool) (*MultiLLMTranslatorWrapper, error) {
	return NewMultiLLMTranslatorWrapperWithFactory(config, eventBus, sessionID, disableLocalLLMs, preferDistributed, nil)
}

// NewMultiLLMTranslatorWrapperWithFactory creates a wrapper whose underlying
// coordinator optionally sources its ensemble translators from an injected
// EnsembleTranslatorFactory (e.g. the provider-diverse LLMsVerifier bridge)
// instead of the built-in per-provider discovery. It is the additive plumbing
// (R-1c) that threads the optional factory from the binary main down to the
// coordinator leaf seam (NewMultiLLMCoordinatorWithFactory).
//
// When factory is nil the behaviour is byte-for-byte identical to the prior
// NewMultiLLMTranslatorWrapperWithConfig: discovery runs exactly as before and
// the same ErrNoLLMInstances is returned when no instances are available.
func NewMultiLLMTranslatorWrapperWithFactory(
	config translator.TranslationConfig,
	eventBus *events.EventBus,
	sessionID string,
	disableLocalLLMs bool,
	preferDistributed bool,
	factory EnsembleTranslatorFactory,
) (*MultiLLMTranslatorWrapper, error) {
	coordinator := NewMultiLLMCoordinatorWithFactory(CoordinatorConfig{
		MaxRetries:        3,
		RetryDelay:        0, // No delay between retries with different instances
		EventBus:          eventBus,
		SessionID:         sessionID,
		DisableLocalLLMs:  disableLocalLLMs,
		PreferDistributed: preferDistributed,
		DistributedCoord:  nil, // CLI doesn't use distributed coordinator
	}, factory)

	if coordinator.GetInstanceCount() == 0 {
		// Fall back to single translator if no instances available
		return nil, translator.ErrNoLLMInstances
	}

	return &MultiLLMTranslatorWrapper{
		Coordinator: coordinator,
		config:      config,
	}, nil
}

// Translate implements translator.Translator
func (w *MultiLLMTranslatorWrapper) Translate(ctx context.Context, text string, context string) (string, error) {
	return w.Coordinator.TranslateWithRetry(ctx, text, context)
}

// TranslateWithProgress implements translator.Translator
func (w *MultiLLMTranslatorWrapper) TranslateWithProgress(
	ctx context.Context,
	text string,
	contextHint string,
	eventBus *events.EventBus,
	sessionID string,
) (string, error) {
	return w.Coordinator.TranslateWithRetry(ctx, text, contextHint)
}

// GetName implements translator.Translator
func (w *MultiLLMTranslatorWrapper) GetName() string {
	return "multi-llm-coordinator"
}

// GetStats implements translator.Translator
func (w *MultiLLMTranslatorWrapper) GetStats() translator.TranslationStats {
	// Multi-LLM coordinator doesn't track individual stats the same way
	// Return zero stats for now - proper stats tracking can be added later
	return translator.TranslationStats{
		Total:      0,
		Translated: 0,
		Cached:     0,
		Errors:     0,
	}
}
