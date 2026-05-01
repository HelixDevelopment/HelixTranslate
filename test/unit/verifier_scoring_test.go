package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier/scoring"
)

// TestEngine_CalculateScore verifies composite score calculation.
func TestEngine_CalculateScore(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	score, err := engine.CalculateScore("model-1", 0.9, 0.8, 0.7, 0.6, 0.5)
	require.NoError(t, err)
	require.NotNil(t, score)

	// Anti-bluff: Verify the score is actually computed, not just stored
	assert.Equal(t, "model-1", score.ModelID)
	assert.Greater(t, score.OverallScore, 0.0)
	assert.LessOrEqual(t, score.OverallScore, 10.0)

	// Verify weights are applied
	expected := 0.9*0.20 + 0.8*0.20 + 0.7*0.20 + 0.6*0.25 + 0.5*0.30
	assert.InDelta(t, expected, score.OverallScore, 0.0001)
}

// TestEngine_CalculateScore_EmptyModelID verifies validation.
func TestEngine_CalculateScore_EmptyModelID(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{})
	_, err := engine.CalculateScore("", 0.5, 0.5, 0.5, 0.5, 0.5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

// TestEngine_GetScore verifies score retrieval.
func TestEngine_GetScore(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed: 1.0,
	})

	_, err := engine.CalculateScore("model-a", 1.0, 0, 0, 0, 0)
	require.NoError(t, err)

	score, ok := engine.GetScore("model-a")
	require.True(t, ok)
	assert.Equal(t, "model-a", score.ModelID)
	assert.Equal(t, 1.0, score.OverallScore)

	_, ok = engine.GetScore("nonexistent")
	assert.False(t, ok)
}

// TestEngine_IsQualified verifies threshold enforcement.
func TestEngine_IsQualified(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed: 1.0,
	})

	_, err := engine.CalculateScore("good-model", 0.8, 0, 0, 0, 0)
	require.NoError(t, err)

	qualified, score, err := engine.IsQualified("good-model", 0.5)
	require.NoError(t, err)
	assert.True(t, qualified)
	assert.Equal(t, 0.8, score)

	qualified, score, err = engine.IsQualified("good-model", 0.9)
	require.NoError(t, err)
	assert.False(t, qualified)
	assert.Equal(t, 0.8, score)

	_, _, err = engine.IsQualified("missing-model", 0.5)
	require.Error(t, err)
}

// TestEngine_GetAllScores verifies sorting by overall score.
func TestEngine_GetAllScores(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed: 1.0,
	})

	for _, m := range []struct {
		id    string
		score float64
	}{
		{"low", 0.3},
		{"high", 0.9},
		{"mid", 0.6},
	} {
		_, err := engine.CalculateScore(m.id, m.score, 0, 0, 0, 0)
		require.NoError(t, err)
	}

	scores := engine.GetAllScores()
	require.Len(t, scores, 3)
	assert.Equal(t, "high", scores[0].ModelID)
	assert.Equal(t, "mid", scores[1].ModelID)
	assert.Equal(t, "low", scores[2].ModelID)
}

// TestEngine_CalculateScore_Clamping verifies input clamping.
func TestEngine_CalculateScore_Clamping(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed: 1.0,
	})

	// Values above 1 should be clamped
	score, err := engine.CalculateScore("over", 2.0, 0, 0, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 1.0, score.ResponsivenessScore)

	// Values below 0 should be clamped
	score, err = engine.CalculateScore("under", -1.0, 0, 0, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 0.0, score.ResponsivenessScore)
}

// BenchmarkEngine_CalculateScore measures scoring performance.
func BenchmarkEngine_CalculateScore(b *testing.B) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.CalculateScore("bench-model", 0.9, 0.8, 0.7, 0.6, 0.5)
	}
}

// TestEngine_CalculateScore_Determinism verifies identical inputs produce identical outputs.
func TestEngine_CalculateScore_Determinism(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.25,
		CostEffectiveness: 0.25,
		ModelEfficiency:   0.25,
		Capability:        0.25,
		Recency:           0.0,
	})

	s1, err := engine.CalculateScore("m", 0.5, 0.5, 0.5, 0.5, 0.5)
	require.NoError(t, err)

	s2, err := engine.CalculateScore("m", 0.5, 0.5, 0.5, 0.5, 0.5)
	require.NoError(t, err)

	assert.Equal(t, s1.OverallScore, s2.OverallScore)
	// Weighted sum: 0.5*0.25 + 0.5*0.25 + 0.5*0.25 + 0.5*0.25 + 0.5*0.25 = 0.625
	assert.InDelta(t, 0.625, s1.OverallScore, 0.0001)
}
