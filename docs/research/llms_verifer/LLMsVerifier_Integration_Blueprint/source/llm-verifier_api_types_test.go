package api

import "time"

// Test types for API testing
type Model struct {
	ModelID      string  `json:"model_id"`
	Name         string  `json:"name"`
	Provider     string  `json:"provider"`
	OverallScore float64 `json:"overall_score"`
	IsActive     bool    `json:"is_active"`
}

type VerificationResult struct {
	ID          string    `json:"id"`
	ModelID     string    `json:"model_id"`
	Prompt      string    `json:"prompt"`
	Response    string    `json:"response"`
	Score       float64   `json:"score"`
	ScoreSuffix string    `json:"score_suffix"`
	Success     bool      `json:"success"`
	Timestamp   time.Time `json:"timestamp"`
	Duration    int64     `json:"duration"`
	Result      *Result   `json:"result,omitempty"`
}

type Result struct {
	ScoreSuffix string `json:"score_suffix"`
}

type ModelScore struct {
	ModelID     string          `json:"model_id"`
	ModelName   string          `json:"model_name"`
	Score       float64         `json:"score"`
	ScoreSuffix string          `json:"score_suffix"`
	Components  ScoreComponents `json:"components"`
	Timestamp   time.Time       `json:"timestamp"`
}

type ScoreComponents struct {
	ResponseSpeed   float64 `json:"response_speed"`
	ModelEfficiency float64 `json:"model_efficiency"`
	CostEffectiveness float64 `json:"cost_effectiveness"`
	Capability      float64 `json:"capability"`
	Recency         float64 `json:"recency"`
}

type ScoreWeights struct {
	ResponseSpeed   float64 `json:"response_speed"`
	ModelEfficiency float64 `json:"model_efficiency"`
	CostEffectiveness float64 `json:"cost_effectiveness"`
	Capability      float64 `json:"capability"`
	Recency         float64 `json:"recency"`
}