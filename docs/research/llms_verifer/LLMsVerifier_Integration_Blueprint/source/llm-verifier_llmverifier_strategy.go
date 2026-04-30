// Copyright 2026 Vasic Digital. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package llmverifier

import (
	"context"
	"fmt"
	"math"
	"time"
)

// TestCategory classifies verification tests
type TestCategory string

const (
	TestCategoryExistence      TestCategory = "existence"
	TestCategoryResponsiveness TestCategory = "responsiveness"
	TestCategoryLatency        TestCategory = "latency"
	TestCategoryFeature        TestCategory = "feature"
	TestCategoryCode           TestCategory = "code"
	TestCategoryVision         TestCategory = "vision"
	TestCategoryInstruction    TestCategory = "instruction"
	TestCategoryStability      TestCategory = "stability"
)

// WeightConfig defines scoring dimension weights.
// Maps to PerformanceScore fields for DefaultStrategy,
// adds VisionCapability and InstructionFollowing for custom strategies.
type WeightConfig struct {
	Responsiveness       float64
	CodeCapability       float64
	FeatureRichness      float64
	Reliability          float64
	VisionCapability     float64
	InstructionFollowing float64
}

// Validate ensures weights sum to 1.0 within tolerance and are non-negative
func (wc WeightConfig) Validate() error {
	weights := []float64{
		wc.Responsiveness, wc.CodeCapability, wc.FeatureRichness,
		wc.Reliability, wc.VisionCapability, wc.InstructionFollowing,
	}
	for i, w := range weights {
		if w < 0 {
			return fmt.Errorf("weight at index %d is negative: %f", i, w)
		}
	}
	sum := wc.Responsiveness + wc.CodeCapability + wc.FeatureRichness +
		wc.Reliability + wc.VisionCapability + wc.InstructionFollowing
	if math.Abs(sum-1.0) > 0.001 {
		return fmt.Errorf("weights must sum to 1.0, got %f", sum)
	}
	return nil
}

// TestResult is the outcome of a single VerificationTest run
type TestResult struct {
	TestID    string
	Category  TestCategory
	Passed    bool
	Score     float64
	Latency   time.Duration
	Error     error
	Details   map[string]any
	Timestamp time.Time
}

// StrategyScore is returned by ScoringStrategy.ScoreModel()
type StrategyScore struct {
	Overall   float64
	Breakdown map[string]float64
	Passed    bool
	Details   map[string]any
}

// ToPerformanceScore converts to existing PerformanceScore for backward compatibility
func (ss StrategyScore) ToPerformanceScore() PerformanceScore {
	return PerformanceScore{
		OverallScore:     ss.Overall,
		Responsiveness:   ss.Breakdown["responsiveness"],
		CodeCapability:   ss.Breakdown["code_capability"],
		FeatureRichness:  ss.Breakdown["feature_richness"],
		Reliability:      ss.Breakdown["reliability"],
		ValueProposition: ss.Breakdown["value_proposition"],
	}
}

// Thresholds defines minimum requirements for model viability
type Thresholds struct {
	MinOverallScore      float64
	MaxLatency           time.Duration
	MinContextWindow     int
	RequiredCapabilities []string
}

// Check verifies if a model meets thresholds
func (th Thresholds) Check(score float64, latency time.Duration, contextWindow int, capabilities []string) bool {
	if score < th.MinOverallScore {
		return false
	}
	if th.MaxLatency > 0 && latency > th.MaxLatency {
		return false
	}
	if contextWindow < th.MinContextWindow {
		return false
	}
	capSet := make(map[string]bool, len(capabilities))
	for _, c := range capabilities {
		capSet[c] = true
	}
	for _, req := range th.RequiredCapabilities {
		if !capSet[req] {
			return false
		}
	}
	return true
}

// ScoringStrategy is the pluggable scoring mechanism interface.
// Implement this to customize how LLMs are evaluated and scored.
type ScoringStrategy interface {
	Name() string
	Description() string
	WeightConfig() WeightConfig
	CustomTests() []VerificationTest
	ScoreModel(ctx context.Context, model ModelInfo, results []TestResult) (StrategyScore, error)
	FilterModels(models []ModelInfo) []ModelInfo
	MinimumThresholds() Thresholds
}

// VerificationTest is an individual test that a strategy can inject
type VerificationTest interface {
	ID() string
	Name() string
	Category() TestCategory
	Run(ctx context.Context, client *LLMClient) (TestResult, error)
	Weight() float64
	Required() bool
}

// StrategyBuilder constructs ScoringStrategy instances via fluent API.
// Defined here as a forward declaration; implementation in strategy_builder.go.
type StrategyBuilder struct {
	name         string
	description  string
	weights      WeightConfig
	tests        []VerificationTest
	capabilities []string
	minScore     float64
	maxLatency   time.Duration
	minContext   int
}

// RecipeStep is a composable building block for StrategyBuilder
type RecipeStep interface {
	Apply(builder *StrategyBuilder) *StrategyBuilder
}
