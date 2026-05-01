package llm

import (
	"context"
	"fmt"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
	"digital.vasic.translator/internal/verifier/selection"
)

// VerifiedFactory creates LLM translators using LLMsVerifier-verified models.
// This is the CONST-034-compliant factory that only permits verified models.
type VerifiedFactory struct {
	selector *selection.Engine
	config   *verifier.Config
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
	}
}

// CreateTranslator builds an LLM translator for the best verified model
// matching the given task requirements.
func (f *VerifiedFactory) CreateTranslator(ctx context.Context, task selection.TaskRequirements) (*LLMTranslator, error) {
	model, err := f.selector.SelectModel(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("no verified model available: %w", err)
	}

	// Build translation config for the selected model
	transConfig := TranslationConfig{
		Provider: model.ProviderID,
		Model:    model.ID,
	}

	return NewLLMTranslatorWithConfig(transConfig)
}

// CreateTranslatorWithFallback builds a translator with fallback chain.
func (f *VerifiedFactory) CreateTranslatorWithFallback(ctx context.Context, task selection.TaskRequirements) (*LLMTranslator, []string, error) {
	primary, err := f.selector.SelectModel(ctx, task)
	if err != nil {
		return nil, nil, fmt.Errorf("no verified model available: %w", err)
	}

	transConfig := TranslationConfig{
		Provider: primary.ProviderID,
		Model:    primary.ID,
	}

	translator, err := NewLLMTranslatorWithConfig(transConfig)
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
