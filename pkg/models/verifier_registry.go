package models

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.translator/internal/verifier"
)

// VerifierRegistry is a model registry that uses LLMsVerifier as the single source of truth.
type VerifierRegistry struct {
	client   *verifier.Client
	registry *verifier.Registry
	config   *verifier.Config
}

// NewVerifierRegistry creates a registry backed by LLMsVerifier.
func NewVerifierRegistry(cfg *verifier.Config) *VerifierRegistry {
	return &VerifierRegistry{
		client:   verifier.NewClient(cfg),
		registry: verifier.NewRegistry(),
		config:   cfg,
	}
}

// Refresh fetches the latest verified models from LLMsVerifier.
func (vr *VerifierRegistry) Refresh(ctx context.Context) error {
	models, err := vr.client.GetVerifiedModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh models from LLMsVerifier: %w", err)
	}

	for _, m := range models {
		vr.registry.AddModel(m)
	}
	return nil
}

// GetModel returns a model by ID, querying LLMsVerifier if not cached.
func (vr *VerifierRegistry) GetModel(ctx context.Context, modelID string) (*verifier.Model, error) {
	// Check cache first
	if m, ok := vr.registry.GetModel(modelID); ok {
		return &m, nil
	}

	// Fetch from LLMsVerifier
	m, err := vr.client.GetModel(ctx, modelID)
	if err != nil {
		return nil, err
	}

	vr.registry.AddModel(*m)
	return m, nil
}

// ListModels returns all verified models available for translation.
func (vr *VerifierRegistry) ListModels() []verifier.Model {
	return vr.registry.ListModels()
}

// ListVerifiedModels returns only models that pass the verification gate.
func (vr *VerifierRegistry) ListVerifiedModels() []verifier.Model {
	return vr.registry.FilterVerified(vr.config.MinScoreThreshold)
}

// IsModelVerified checks if a model has passed LLMsVerifier verification.
func (vr *VerifierRegistry) IsModelVerified(modelID string) bool {
	m, ok := vr.registry.GetModel(modelID)
	if !ok {
		return false
	}
	return m.VerificationStatus == "verified" && m.CanSeeCode && m.AffirmativeResponse && m.OverallScore > vr.config.MinScoreThreshold
}

// ToModelInfo converts a verifier.Model to a ModelInfo for backward compatibility.
func (vr *VerifierRegistry) ToModelInfo(m verifier.Model) *ModelInfo {
	return &ModelInfo{
		ID:            m.ID,
		Name:          m.Name,
		Description:   fmt.Sprintf("Verified model from %s (score: %.2f)", m.ProviderID, m.OverallScore),
		Quality:       vr.scoreToQuality(m.OverallScore),
		ContextLength: 0, // Would be populated from verifier capabilities
		Languages:     []string{},
		RequiresGPU:   m.Capabilities["gpu"],
	}
}

func (vr *VerifierRegistry) scoreToQuality(score float64) string {
	if score >= 0.9 {
		return "excellent"
	}
	if score >= 0.7 {
		return "good"
	}
	if score >= 0.5 {
		return "moderate"
	}
	return "poor"
}

// HealthCheck verifies connectivity to LLMsVerifier.
func (vr *VerifierRegistry) HealthCheck(ctx context.Context) error {
	return vr.client.Ping(ctx)
}

// LastRefresh returns the timestamp of the last successful cache refresh.
func (vr *VerifierRegistry) LastRefresh() time.Time {
	// The registry doesn't track this directly; we'd need to add it
	return time.Time{}
}
