package scoring

import (
	"context"
	"time"
)

// ScoreComponents represents the individual components of a model score
type ScoreComponents struct {
	SpeedScore      float64 `json:"speed_score"`
	EfficiencyScore float64 `json:"efficiency_score"`
	CostScore       float64 `json:"cost_score"`
	CapabilityScore float64 `json:"capability_score"`
	RecencyScore    float64 `json:"recency_score"`
}

// ComprehensiveScore represents a comprehensive model score with all components
type ComprehensiveScore struct {
	ModelID        string         `json:"model_id"`
	ModelName      string         `json:"model_name"`
	OverallScore   float64        `json:"overall_score"`
	ScoreSuffix    string         `json:"score_suffix"`
	Components     ScoreComponents `json:"components"`
	LastCalculated time.Time      `json:"last_calculated"`
	CalculationHash string        `json:"calculation_hash"`
	DataSource     string         `json:"data_source"`
}

// ModelScore represents a model score (used in scoring engine)
type ModelScore struct {
	ID                int64           `json:"id"`
	ModelID           string          `json:"model_id"`
	ModelName         string          `json:"model_name"`
	Score             float64         `json:"score"`
	ScoreSuffix       string          `json:"score_suffix"`
	Components        ScoreComponents `json:"components"`
	CalculationHash   string          `json:"calculation_hash"`
	CalculationDetails string         `json:"calculation_details,omitempty"`
	LastCalculated    time.Time       `json:"last_calculated"`
	ValidUntil        *time.Time      `json:"valid_until,omitempty"`
	IsActive          bool            `json:"is_active"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	DataSource        string          `json:"data_source"`
}

// ScoreWeights represents the weights for different scoring components
type ScoreWeights struct {
	ResponseSpeed   float64 `json:"response_speed"`
	ModelEfficiency float64 `json:"model_efficiency"`
	CostEffectiveness float64 `json:"cost_effectiveness"`
	Capability      float64 `json:"capability"`
	Recency         float64 `json:"recency"`
}

// ScoreThresholds represents score thresholds
type ScoreThresholds struct {
	MinScore float64 `json:"min_score"`
	MaxScore float64 `json:"max_score"`
}

// ScoringConfig represents scoring system configuration
type ScoringConfig struct {
	ConfigName  string         `json:"config_name"`
	Weights     ScoreWeights   `json:"weights"`
	Thresholds  ScoreThresholds `json:"thresholds"`
	Enabled     bool           `json:"enabled"`
	LastUpdated time.Time      `json:"last_updated"`
}



// BatchScoreRequest represents a batch scoring request
type BatchScoreRequest struct {
	ModelIDs []string     `json:"model_ids"`
	Weights  *ScoreWeights `json:"weights,omitempty"`
}

// BatchScoreResponse represents a batch scoring response
type BatchScoreResponse struct {
	Scores      []*ComprehensiveScore `json:"scores"`
	Processed   int                   `json:"processed"`
	Failed      int                   `json:"failed"`
	Total       int                   `json:"total"`
	ProcessTime float64               `json:"process_time_seconds"`
}

// ScoreComparison represents a score comparison between models
type ScoreComparison struct {
	ModelID1   string  `json:"model_id_1"`
	ModelID2   string  `json:"model_id_2"`
	Score1     float64 `json:"score_1"`
	Score2     float64 `json:"score_2"`
	Difference float64 `json:"difference"`
	BetterModel string `json:"better_model"`
}

// ScoreHistory represents score history for a model
type ScoreHistory struct {
	ModelID      string    `json:"model_id"`
	Scores       []float64 `json:"scores"`
	Timestamps   []time.Time `json:"timestamps"`
	ScoreChanges []float64 `json:"score_changes"`
}

// ScoreAnalytics represents score analytics data
type ScoreAnalytics struct {
	AverageScore    float64           `json:"average_score"`
	MedianScore     float64           `json:"median_score"`
	MinScore        float64           `json:"min_score"`
	MaxScore        float64           `json:"max_score"`
	StdDev          float64           `json:"std_dev"`
	TotalModels     int               `json:"total_models"`
	ScoreDistribution []ScoreDistribution `json:"score_distribution"`
}

// ModelData represents data from models.dev API
type ModelData struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Provider        string    `json:"provider"`
	Description     string    `json:"description"`
	Capabilities    []string  `json:"capabilities"`
	ContextWindow   int       `json:"context_window"`
	MaxTokens       int       `json:"max_tokens"`
	InputTokenCost  float64   `json:"input_token_cost"`
	OutputTokenCost float64   `json:"output_token_cost"`
	ThroughputRPS   float64   `json:"throughput_rps"`
	LatencyMs       int       `json:"latency_ms"`
	ReleaseDate     time.Time `json:"release_date"`
	TrainingCutoff  time.Time `json:"training_cutoff"`
	ParameterCount  int64     `json:"parameter_count"`
	OpenSource      bool      `json:"open_source"`
	Multimodal      bool      `json:"multimodal"`
	Reasoning       bool      `json:"reasoning"`
	LastUpdated     time.Time `json:"last_updated"`
}

// ModelRanking represents a model's ranking information
type ModelRanking struct {
	Rank          int        `json:"rank"`
	ModelID       string     `json:"model_id"`
	ModelName     string     `json:"model_name"`
	OverallScore  float64    `json:"overall_score"`
	ScoreSuffix   string     `json:"score_suffix"`
	Category      string     `json:"category"`
	CategoryScore float64    `json:"category_score"`
	LastUpdated   time.Time  `json:"last_updated"`
}

// ModelsDevClientInterface defines the interface for models.dev client
type ModelsDevClientInterface interface {
	FetchAllModels(ctx context.Context) (*ModelsDevAPIResponse, error)
	FetchModelByID(ctx context.Context, modelID string) (*ModelsDevModel, error)
	FetchModelsByProvider(ctx context.Context, providerID string) ([]ModelsDevModel, error)
}

// DefaultScoringConfig returns the default scoring configuration
func DefaultScoringConfig() ScoringConfig {
	return ScoringConfig{
		ConfigName: "default",
		Weights: ScoreWeights{
			ResponseSpeed:   0.25,
			ModelEfficiency: 0.20,
			CostEffectiveness: 0.25,
			Capability:      0.20,
			Recency:         0.10,
		},
		Thresholds: ScoreThresholds{
			MinScore: 0.0,
			MaxScore: 10.0,
		},
		Enabled:     true,
		LastUpdated: time.Now(),
	}
}