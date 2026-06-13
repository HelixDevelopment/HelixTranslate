package scoring

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// Engine computes composite scores for verified models.
type Engine struct {
	weights ScoreWeights
	mu      sync.RWMutex
	scores  map[string]*ModelScore
}

// ScoreWeights defines component weights.
type ScoreWeights struct {
	ResponseSpeed     float64
	CostEffectiveness float64
	ModelEfficiency   float64
	Capability        float64
	Recency           float64
}

// ModelScore holds per-model scoring data.
type ModelScore struct {
	ModelID              string
	ResponsivenessScore  float64
	CodeCapabilityScore  float64
	FeatureRichnessScore float64
	ReliabilityScore     float64
	CostEfficiencyScore  float64
	OverallScore         float64
	ComputedAt           time.Time
}

// NewEngine creates a scoring engine with the given weights.
func NewEngine(weights ScoreWeights) *Engine {
	return &Engine{
		weights: weights,
		scores:  make(map[string]*ModelScore),
	}
}

// CalculateScore computes the composite score for a model.
func (e *Engine) CalculateScore(modelID string, responsiveness, codeCapability, featureRichness, reliability, costEfficiency float64) (*ModelScore, error) {
	if modelID == "" {
		return nil, fmt.Errorf("model ID cannot be empty")
	}

	// Normalize inputs to 0-1 range using sigmoid-like clamping
	responsiveness = clamp01(responsiveness)
	codeCapability = clamp01(codeCapability)
	featureRichness = clamp01(featureRichness)
	reliability = clamp01(reliability)
	costEfficiency = clamp01(costEfficiency)

	// Apply weights from requirements document. Each of the five distinct
	// component weights MUST be applied exactly once so that, when the weight
	// set sums to 1.0 (the production default), an all-max model scores exactly
	// 1.0 and stays inside the 0..1 contract IsQualified/threshold logic relies
	// on. Previously Capability was applied twice and Recency was dropped, which
	// summed applied weights to 1.15 and silently ignored the recency component.
	overall := (responsiveness * e.weights.ResponseSpeed) +
		(codeCapability * e.weights.Capability) +
		(featureRichness * e.weights.Recency) +
		(reliability * e.weights.ModelEfficiency) +
		(costEfficiency * e.weights.CostEffectiveness)

	score := &ModelScore{
		ModelID:              modelID,
		ResponsivenessScore:  responsiveness,
		CodeCapabilityScore:  codeCapability,
		FeatureRichnessScore: featureRichness,
		ReliabilityScore:     reliability,
		CostEfficiencyScore:  costEfficiency,
		OverallScore:         round(overall, 4),
		ComputedAt:           time.Now(),
	}

	e.mu.Lock()
	e.scores[modelID] = score
	e.mu.Unlock()

	return score, nil
}

// GetScore retrieves a cached score.
func (e *Engine) GetScore(modelID string) (*ModelScore, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	score, ok := e.scores[modelID]
	return score, ok
}

// GetAllScores returns all cached scores sorted by overall score descending.
func (e *Engine) GetAllScores() []*ModelScore {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*ModelScore, 0, len(e.scores))
	for _, s := range e.scores {
		result = append(result, s)
	}

	// Sort by overall score descending
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].OverallScore > result[i].OverallScore {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// IsQualified checks if a model meets the minimum score threshold.
func (e *Engine) IsQualified(modelID string, minScore float64) (bool, float64, error) {
	score, ok := e.GetScore(modelID)
	if !ok {
		return false, 0, fmt.Errorf("no score found for model %s", modelID)
	}
	return score.OverallScore >= minScore, score.OverallScore, nil
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}
