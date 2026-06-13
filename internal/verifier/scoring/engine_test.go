package scoring

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineCalculateScore(t *testing.T) {
	weights := ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	}
	engine := NewEngine(weights)

	score, err := engine.CalculateScore("gpt-4", 0.9, 0.95, 0.8, 0.85, 0.7)
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", score.ModelID)
	assert.Greater(t, score.OverallScore, 0.0)
	assert.Less(t, score.OverallScore, 1.0)
}

func TestEngineCalculateScoreEmptyModelID(t *testing.T) {
	engine := NewEngine(ScoreWeights{})
	_, err := engine.CalculateScore("", 0.5, 0.5, 0.5, 0.5, 0.5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestEngineCalculateScoreClamping(t *testing.T) {
	weights := ScoreWeights{
		ResponseSpeed:     1.0,
		CostEffectiveness: 0.0,
		ModelEfficiency:   0.0,
		Capability:        0.0,
		Recency:           0.0,
	}
	engine := NewEngine(weights)

	// Values above 1 should be clamped
	score, err := engine.CalculateScore("test", 2.0, 999.0, -5.0, 0.5, 0.5)
	require.NoError(t, err)
	assert.Equal(t, 1.0, score.ResponsivenessScore)
	assert.Equal(t, 1.0, score.CodeCapabilityScore)
	assert.Equal(t, 0.0, score.FeatureRichnessScore)
}

func TestEngineGetScore(t *testing.T) {
	weights := ScoreWeights{ResponseSpeed: 1.0}
	engine := NewEngine(weights)

	_, ok := engine.GetScore("gpt-4")
	assert.False(t, ok)

	_, err := engine.CalculateScore("gpt-4", 0.8, 0.5, 0.5, 0.5, 0.5)
	require.NoError(t, err)

	score, ok := engine.GetScore("gpt-4")
	require.True(t, ok)
	assert.Equal(t, "gpt-4", score.ModelID)
}

func TestEngineGetAllScores(t *testing.T) {
	weights := ScoreWeights{ResponseSpeed: 1.0}
	engine := NewEngine(weights)

	_, err := engine.CalculateScore("model-b", 0.5, 0.5, 0.5, 0.5, 0.5)
	require.NoError(t, err)
	_, err = engine.CalculateScore("model-a", 0.9, 0.5, 0.5, 0.5, 0.5)
	require.NoError(t, err)

	scores := engine.GetAllScores()
	require.Len(t, scores, 2)
	// Should be sorted by overall score descending
	assert.Equal(t, "model-a", scores[0].ModelID)
	assert.Equal(t, "model-b", scores[1].ModelID)
}

func TestEngineIsQualified(t *testing.T) {
	weights := ScoreWeights{ResponseSpeed: 1.0}
	engine := NewEngine(weights)

	_, err := engine.CalculateScore("high", 0.9, 0.5, 0.5, 0.5, 0.5)
	require.NoError(t, err)

	qualified, score, err := engine.IsQualified("high", 0.5)
	require.NoError(t, err)
	assert.True(t, qualified)
	assert.Greater(t, score, 0.5)

	qualified, _, err = engine.IsQualified("high", 0.99)
	require.NoError(t, err)
	assert.False(t, qualified)

	_, _, err = engine.IsQualified("missing", 0.5)
	require.Error(t, err)
}

func TestEngineWeightedCalculation(t *testing.T) {
	// With uniform weights, overall should be average of clamped inputs
	weights := ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.20,
		ModelEfficiency:   0.20,
		Capability:        0.20,
		Recency:           0.20,
	}
	engine := NewEngine(weights)

	score, err := engine.CalculateScore("test", 0.5, 0.5, 0.5, 0.5, 0.5)
	require.NoError(t, err)
	// responsiveness*0.2 + codeCapability*0.2 + featureRichness*0.2 + reliability*0.2 + costEfficiency*0.2
	// = 0.5 * (0.2 + 0.2 + 0.2 + 0.2 + 0.2) = 0.5
	assert.InDelta(t, 0.5, score.OverallScore, 0.01)
}

func TestEngineConcurrentAccess(t *testing.T) {
	weights := ScoreWeights{ResponseSpeed: 1.0}
	engine := NewEngine(weights)

	done := make(chan bool, 100)
	for i := 0; i < 50; i++ {
		go func(idx int) {
			_, _ = engine.CalculateScore(string(rune('a'+idx%26)), float64(idx)/100.0, 0.5, 0.5, 0.5, 0.5)
			done <- true
		}(i)
	}
	for i := 0; i < 50; i++ {
		go func(idx int) {
			_, _ = engine.GetScore(string(rune('a' + idx%26)))
			done <- true
		}(i)
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	// No deadlock or data race
	assert.GreaterOrEqual(t, len(engine.GetAllScores()), 1)
}

func TestRound(t *testing.T) {
	// Test via public behavior: overall score should be rounded to 4 decimals
	weights := ScoreWeights{ResponseSpeed: 1.0 / 3.0}
	engine := NewEngine(weights)

	score, err := engine.CalculateScore("test", 1.0, 0.0, 0.0, 0.0, 0.0)
	require.NoError(t, err)
	// Check that rounding occurred (no more than 4 decimal places)
	multiplier := math.Pow(10, 5)
	assert.Equal(t, math.Floor(score.OverallScore*multiplier), score.OverallScore*multiplier,
		"score should not have more than 4 decimal places of precision")
}

// TestEngineWeightsAppliedOnceEach is a RED regression test (Wave bug-hunt).
// FACT: with the production-default asymmetric weights (sum == 1.0), all five
// distinct component weights MUST be applied exactly once, so a model with all
// component inputs == 1.0 MUST score exactly 1.0. The pre-fix formula reused
// Capability twice and dropped Recency, producing 1.15 (sum of applied weights),
// breaking the 0..1 contract that IsQualified / clamping rely on.
func TestEngineWeightsAppliedOnceEach(t *testing.T) {
	weights := ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	}
	engine := NewEngine(weights)

	// All component inputs maxed -> overall MUST equal the weight sum == 1.0.
	score, err := engine.CalculateScore("max", 1.0, 1.0, 1.0, 1.0, 1.0)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, score.OverallScore, 1e-9,
		"all-max inputs with sum-1.0 weights must score 1.0; got %v (weights applied wrong / Recency dropped)", score.OverallScore)

	// Distinct-input probe: each weight must touch its own component exactly once.
	// responsiveness=1,0,0,0,0 isolates ResponseSpeed.
	s2, err := engine.CalculateScore("resp", 1.0, 0.0, 0.0, 0.0, 0.0)
	require.NoError(t, err)
	assert.InDelta(t, 0.20, s2.OverallScore, 1e-9, "ResponseSpeed weight")
	// featureRichness=0,0,1,0,0 isolates the 5th weight (Recency), which the bug dropped.
	s3, err := engine.CalculateScore("feat", 0.0, 0.0, 1.0, 0.0, 0.0)
	require.NoError(t, err)
	assert.InDelta(t, 0.05, s3.OverallScore, 1e-9, "featureRichness must use Recency weight (0.05), not Capability (0.20)")
}
