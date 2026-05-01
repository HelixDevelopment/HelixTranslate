package selection

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
)

// Engine selects the best model for a given translation task.
type Engine struct {
	registry *verifier.Registry
	scoring  *scoring.Engine
	config   *verifier.Config
	mu       sync.RWMutex
	fallback map[string][]string // modelID -> ordered fallback model IDs
}

// TaskRequirements describes what a translation task needs.
type TaskRequirements struct {
	SourceLang      string
	TargetLang      string
	RequireStreaming bool
	RequireVision    bool
	RequireCode      bool
	MaxLatencyMs     int
	MaxCostPer1K     float64
}

// NewEngine creates a model selection engine.
func NewEngine(registry *verifier.Registry, scoringEngine *scoring.Engine, config *verifier.Config) *Engine {
	return &Engine{
		registry: registry,
		scoring:  scoringEngine,
		config:   config,
		fallback: make(map[string][]string),
	}
}

// SelectModel chooses the best verified model for the task.
func (e *Engine) SelectModel(ctx context.Context, task TaskRequirements) (*verifier.Model, error) {
	candidates := e.registry.FilterVerified(e.config.MinScoreThreshold)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no verified models available meeting threshold %.2f", e.config.MinScoreThreshold)
	}

	// Score and rank candidates
	type candidateScore struct {
		model verifier.Model
		score float64
	}

	scored := make([]candidateScore, 0, len(candidates))
	for _, m := range candidates {
		score := e.calculateTaskScore(m, task)
		scored = append(scored, candidateScore{model: m, score: score})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Apply capability filters
	for _, cs := range scored {
		if e.meetsRequirements(cs.model, task) {
			return &cs.model, nil
		}
	}

	return nil, fmt.Errorf("no model meets task requirements")
}

// SelectFallback returns the next best model when the primary fails.
func (e *Engine) SelectFallback(primaryModelID string, task TaskRequirements) (*verifier.Model, error) {
	e.mu.RLock()
	chain, ok := e.fallback[primaryModelID]
	e.mu.RUnlock()

	if !ok || len(chain) == 0 {
		// Build fallback chain on demand
		chain = e.buildFallbackChain(primaryModelID)
		e.mu.Lock()
		e.fallback[primaryModelID] = chain
		e.mu.Unlock()
	}

	for _, modelID := range chain {
		if m, ok := e.registry.GetModel(modelID); ok {
			if e.meetsRequirements(m, task) {
				return &m, nil
			}
		}
	}

	return nil, fmt.Errorf("no fallback model available for %s", primaryModelID)
}

// buildFallbackChain creates an ordered list of fallback models.
func (e *Engine) buildFallbackChain(excludeModelID string) []string {
	candidates := e.registry.FilterVerified(e.config.MinScoreThreshold)
	var chain []string
	for _, m := range candidates {
		if m.ID != excludeModelID {
			chain = append(chain, m.ID)
		}
	}
	// Shuffle slightly to ensure provider diversity
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(chain), func(i, j int) {
		chain[i], chain[j] = chain[j], chain[i]
	})
	return chain
}

// calculateTaskScore computes a suitability score for a model given a task.
func (e *Engine) calculateTaskScore(model verifier.Model, task TaskRequirements) float64 {
	baseScore := model.OverallScore

	// Penalize models that don't support required capabilities
	if task.RequireStreaming && !model.Capabilities["streaming"] {
		baseScore *= 0.5
	}
	if task.RequireVision && !model.Capabilities["vision"] {
		baseScore *= 0.5
	}
	if task.RequireCode && !model.CanSeeCode {
		baseScore *= 0.3
	}

	return baseScore
}

// GetRegistry returns the underlying model registry.
func (e *Engine) GetRegistry() *verifier.Registry {
	return e.registry
}

// meetsRequirements checks if a model satisfies hard task requirements.
func (e *Engine) meetsRequirements(model verifier.Model, task TaskRequirements) bool {
	if task.RequireStreaming && !model.Capabilities["streaming"] {
		return false
	}
	if task.RequireVision && !model.Capabilities["vision"] {
		return false
	}
	if task.RequireCode && !model.CanSeeCode {
		return false
	}
	return true
}
