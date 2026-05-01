// Package verifier provides LLMsVerifier integration for HelixTranslate.
// It serves as the single source of truth for LLM model provisioning,
// verification, scoring, and selection.
package verifier

import (
	"time"
)

// Config holds verifier-specific configuration derived from the global config.
type Config struct {
	Enabled             bool
	APIURL              string
	APIKey              string
	DBPath              string
	CacheTTL            time.Duration
	StrictMode          bool
	VerificationEnabled bool
	MinScoreThreshold   float64
	MaxProviders        int
	ScoringWeights      ScoreWeights
	Options             map[string]interface{}
}

// ScoreWeights defines the component weights for model scoring.
type ScoreWeights struct {
	ResponseSpeed     float64
	CostEffectiveness float64
	ModelEfficiency   float64
	Capability        float64
	Recency           float64
}

// ProviderConfig holds per-provider settings for discovery.
type ProviderConfig struct {
	ID      string
	APIKey  string
	BaseURL string
	Models  []string
}

// DefaultConfig returns a default verifier configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:             true,
		APIURL:              "http://localhost:8080",
		DBPath:              "./data/verifier.db",
		CacheTTL:            time.Hour,
		StrictMode:          false,
		VerificationEnabled: true,
		MinScoreThreshold:   0.0,
		MaxProviders:        25,
		ScoringWeights: ScoreWeights{
			ResponseSpeed:     0.20,
			CostEffectiveness: 0.30,
			ModelEfficiency:   0.25,
			Capability:        0.20,
			Recency:           0.05,
		},
	}
}
