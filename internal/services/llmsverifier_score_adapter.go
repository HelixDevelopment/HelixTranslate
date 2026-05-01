package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
)

// LLMsVerifierScoreAdapter bridges LLMsVerifier scores to HelixTranslate provider preferences.
type LLMsVerifierScoreAdapter struct {
	scores      map[string]float64 // modelID -> normalized score (0-10)
	mu          sync.RWMutex
	lastRefresh time.Time
	client      *verifier.Client
	engine      *scoring.Engine
	config      *verifier.Config
}

// ProviderPreference is the HelixTranslate-native representation of a scored model.
type ProviderPreference struct {
	ProviderID    string
	ModelID       string
	ModelName     string
	Weight        float64
	FallbackOrder int
	Score         float64
	Capabilities  map[string]bool
}

// NewLLMsVerifierScoreAdapter creates a score adapter.
func NewLLMsVerifierScoreAdapter(client *verifier.Client, engine *scoring.Engine, config *verifier.Config) *LLMsVerifierScoreAdapter {
	return &LLMsVerifierScoreAdapter{
		scores: make(map[string]float64),
		client: client,
		engine: engine,
		config: config,
	}
}

// RefreshScores fetches the latest scores from LLMsVerifier and updates the adapter.
func (a *LLMsVerifierScoreAdapter) RefreshScores(ctx context.Context) error {
	models, err := a.client.GetVerifiedModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch verified models: %w", err)
	}

	newScores := make(map[string]float64, len(models))
	for _, m := range models {
		// Normalize overall score from LLMsVerifier (typically 0-1 or 0-100) to 0-10
		normalized := normalizeScore(m.OverallScore)
		newScores[m.ID] = normalized

		// Also cache in scoring engine
		_, _ = a.engine.CalculateScore(m.ID,
			m.ResponsivenessScore,
			m.CodeCapabilityScore,
			m.FeatureRichnessScore,
			m.ReliabilityScore,
			m.Pricing.InputTokenCost,
		)
	}

	a.mu.Lock()
	a.scores = newScores
	a.lastRefresh = time.Now()
	a.mu.Unlock()

	return nil
}

// GetProviderScore returns the normalized score for a provider's model.
func (a *LLMsVerifierScoreAdapter) GetProviderScore(modelID string) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if score, ok := a.scores[modelID]; ok {
		return score
	}
	return 0
}

// GetPreferences returns all models as HelixTranslate provider preferences, sorted by score descending.
func (a *LLMsVerifierScoreAdapter) GetPreferences(ctx context.Context) ([]ProviderPreference, error) {
	models, err := a.client.GetVerifiedModels(ctx)
	if err != nil {
		return nil, err
	}

	prefs := make([]ProviderPreference, 0, len(models))
	for i, m := range models {
		score := a.GetProviderScore(m.ID)
		if score < a.config.MinScoreThreshold {
			continue
		}
		prefs = append(prefs, ProviderPreference{
			ProviderID:    m.ProviderID,
			ModelID:       m.ID,
			ModelName:     m.Name,
			Weight:        score / 10.0,
			FallbackOrder: i + 1,
			Score:         score,
			Capabilities:  m.Capabilities,
		})
	}

	return prefs, nil
}

// LastRefresh returns when scores were last updated.
func (a *LLMsVerifierScoreAdapter) LastRefresh() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastRefresh
}

// normalizeScore converts LLMsVerifier scores to a 0-10 scale.
func normalizeScore(score float64) float64 {
	if score <= 0 {
		return 0
	}
	if score > 100 {
		return 10
	}
	if score > 10 {
		// Likely 0-100 scale
		return score / 10.0
	}
	// Already 0-10 scale or 0-1 scale
	if score <= 1 {
		return score * 10
	}
	return score
}
