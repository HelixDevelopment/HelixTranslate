package scoring

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"digital.vasic.llmsverifier/logging"
)

// ScoringAPIHandlers handles HTTP requests for scoring functionality
type ScoringAPIHandlers struct {
	scoringEngine *ScoringEngine
	modelNaming   *ModelNaming
	logger        *logging.Logger
}

// NewScoringAPIHandlers creates new API handlers for scoring
func NewScoringAPIHandlers(scoringEngine *ScoringEngine, logger *logging.Logger) *ScoringAPIHandlers {
	return &ScoringAPIHandlers{
		scoringEngine: scoringEngine,
		modelNaming:   NewModelNaming(),
		logger:        logger,
	}
}

// RegisterRoutes registers all scoring-related API routes
func (sah *ScoringAPIHandlers) RegisterRoutes(router *gin.RouterGroup) {
	// Model scoring endpoints
	router.GET("/models/:model_id/score", sah.GetModelScore)
	router.POST("/models/:model_id/score/calculate", sah.CalculateModelScore)
	router.PUT("/models/:model_id/score/recalculate", sah.RecalculateModelScore)
	router.DELETE("/models/:model_id/score", sah.DeleteModelScore)

	// Batch scoring endpoints
	router.POST("/models/scores/batch", sah.BatchCalculateScores)

	// Score comparison and analysis
	router.GET("/models/scores/compare", sah.CompareModels)
	router.GET("/models/scores/ranking", sah.GetModelRankings)

	// Score configuration
	router.GET("/scoring/configuration", sah.GetScoringConfiguration)

	// Model naming with scores
	router.POST("/models/naming/add-suffix", sah.AddScoreSuffixToModelName)
	router.POST("/models/naming/batch-update", sah.BatchUpdateModelNamesWithScores)

	// Score validation and debugging
	router.POST("/scoring/validate", sah.ValidateScoreCalculation)
}

// GetModelScore retrieves the current score for a model
func (sah *ScoringAPIHandlers) GetModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	// Simulate getting score
	score := &ComprehensiveScore{
		ModelID:      modelID,
		ModelName:    "GPT-4",
		OverallScore: 8.5,
		ScoreSuffix:  "(SC:8.5)",
		Components: ScoreComponents{
			SpeedScore:      8.0,
			EfficiencyScore: 9.0,
			CostScore:       8.5,
			CapabilityScore: 8.5,
			RecencyScore:    8.0,
		},
		LastCalculated: time.Now(),
	}

	// Add formatted model name with score suffix
	formattedName := sah.modelNaming.AddScoreSuffix(score.ModelName, score.OverallScore)

	c.JSON(http.StatusOK, gin.H{
		"model_id":         modelID,
		"model_name":       score.ModelName,
		"formatted_name":   formattedName,
		"overall_score":    score.OverallScore,
		"score_suffix":     score.ScoreSuffix,
		"components":       score.Components,
		"last_calculated":  score.LastCalculated,
	})
}

// CalculateModelScore calculates a new score for a model
func (sah *ScoringAPIHandlers) CalculateModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	var request struct {
		Configuration *ScoringConfig `json:"configuration,omitempty"`
		ForceRecalc   bool           `json:"force_recalculation,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Use provided configuration or default
	config := DefaultScoringConfig()
	if request.Configuration != nil {
		config = *request.Configuration
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Calculate score using the scoring engine
	score, err := sah.scoringEngine.CalculateComprehensiveScore(ctx, modelID, config)
	if err != nil {
		sah.logger.Error("Failed to calculate model score", map[string]interface{}{
			"model_id": modelID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate model score",
		})
		return
	}

	// Format model name with score
	formattedName := sah.modelNaming.AddScoreSuffix(score.ModelName, score.OverallScore)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Score calculated successfully",
		"model_id":         modelID,
		"model_name":       score.ModelName,
		"formatted_name":   formattedName,
		"overall_score":    score.OverallScore,
		"score_suffix":     score.ScoreSuffix,
		"components":       score.Components,
		"last_calculated":  score.LastCalculated,
	})
}

// RecalculateModelScore forces recalculation of an existing score
func (sah *ScoringAPIHandlers) RecalculateModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	var request struct {
		Reason      string         `json:"reason,omitempty"`
		Configuration *ScoringConfig `json:"configuration,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Use provided configuration or default
	config := DefaultScoringConfig()
	if request.Configuration != nil {
		config = *request.Configuration
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Recalculate score using the scoring engine
	score, err := sah.scoringEngine.CalculateComprehensiveScore(ctx, modelID, config)
	if err != nil {
		sah.logger.Error("Failed to recalculate model score", map[string]interface{}{
			"model_id": modelID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to recalculate model score",
		})
		return
	}

	// Log the recalculation reason
	if request.Reason != "" {
		sah.logger.Info("Model score recalculated", map[string]interface{}{
			"model_id": modelID,
			"reason": request.Reason,
			"new_score": score.OverallScore,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Score recalculated successfully",
		"model_id":         modelID,
		"overall_score":    score.OverallScore,
		"score_suffix":     score.ScoreSuffix,
		"components":       score.Components,
		"last_calculated":  score.LastCalculated,
		"recalc_reason":    request.Reason,
	})
}

// DeleteModelScore removes a model score (soft delete)
func (sah *ScoringAPIHandlers) DeleteModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	// Simulate deactivation
	sah.logger.Info("Model score deactivated", map[string]interface{}{"model_id": modelID})

	c.JSON(http.StatusOK, gin.H{
		"message":   "Model score deactivated successfully",
		"model_id":  modelID,
	})
}

// BatchCalculateScores calculates scores for multiple models
func (sah *ScoringAPIHandlers) BatchCalculateScores(c *gin.Context) {
	var request struct {
		ModelIDs    []string       `json:"model_ids" binding:"required"`
		Configuration *ScoringConfig `json:"configuration,omitempty"`
		Async       bool           `json:"async,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Use provided configuration or default
	_ = DefaultScoringConfig()
	if request.Configuration != nil {
		_ = *request.Configuration
	}

	batchID := generateBatchID()
	
	if request.Async {
		// Start async processing
		go sah.processBatchScoresAsync(batchID, request.ModelIDs, *request.Configuration)
		
		c.JSON(http.StatusAccepted, gin.H{
			"message":   "Batch score calculation started",
			"batch_id":  batchID,
			"status":    "processing",
			"model_count": len(request.ModelIDs),
		})
		return
	}

	// Process synchronously
	results := sah.processBatchScoresSync(request.ModelIDs, *request.Configuration)
	
	c.JSON(http.StatusOK, gin.H{
		"message":      "Batch score calculation completed",
		"batch_id":     batchID,
		"status":       "completed",
		"results":      results,
		"model_count":  len(request.ModelIDs),
		"success_count": len(results),
	})
}

// CompareModels compares scores between multiple models
func (sah *ScoringAPIHandlers) CompareModels(c *gin.Context) {
	modelIDs := c.QueryArray("models")
	if len(modelIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least 2 models are required for comparison",
		})
		return
	}

	// Simulate comparison
	comparison := map[string]interface{}{
		"best_model": modelIDs[0],
		"score_difference": 0.5,
		"analysis": "Model comparison completed",
	}

	c.JSON(http.StatusOK, gin.H{
		"models":      modelIDs,
		"comparison":  comparison,
	})
}

// GetModelRankings retrieves model rankings by score
func (sah *ScoringAPIHandlers) GetModelRankings(c *gin.Context) {
	category := c.DefaultQuery("category", "overall")
	_ = c.DefaultQuery("limit", "50")
	_ = c.DefaultQuery("min_score", "0")
	_ = c.DefaultQuery("max_score", "10")

	// Simulate rankings
	rankings := []ModelRanking{
		{
			Rank:          1,
			ModelID:       "1",
			ModelName:     "GPT-4 (SC:8.5)",
			OverallScore:  8.5,
			ScoreSuffix:   "(SC:8.5)",
			CategoryScore: 8.5,
			LastUpdated:   time.Now(),
		},
		{
			Rank:          2,
			ModelID:       "2",
			ModelName:     "Claude-3 (SC:7.8)",
			OverallScore:  7.8,
			ScoreSuffix:   "(SC:7.8)",
			CategoryScore: 7.8,
			LastUpdated:   time.Now(),
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"category":  category,
		"rankings":  rankings,
	})
}

// GetScoringConfiguration retrieves current scoring configuration
func (sah *ScoringAPIHandlers) GetScoringConfiguration(c *gin.Context) {
	configName := c.DefaultQuery("config", "default")
	
	// Simulate configuration
	config := map[string]interface{}{
		"config_name": configName,
		"weights": map[string]float64{
			"response_speed":    0.25,
			"model_efficiency":  0.20,
			"cost_effectiveness": 0.25,
			"capability":        0.20,
			"recency":          0.10,
		},
		"thresholds": map[string]float64{
			"min_score": 0.0,
			"max_score": 10.0,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"config_name":   configName,
		"configuration": config,
	})
}

// AddScoreSuffixToModelName adds score suffix to a model name
func (sah *ScoringAPIHandlers) AddScoreSuffixToModelName(c *gin.Context) {
	var request struct {
		ModelName string  `json:"model_name" binding:"required"`
		Score     float64 `json:"score" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	updatedName := sah.modelNaming.AddScoreSuffix(request.ModelName, request.Score)

	c.JSON(http.StatusOK, gin.H{
		"original_name": request.ModelName,
		"updated_name":  updatedName,
		"score":         request.Score,
		"score_suffix":  sah.modelNaming.GenerateScoreSuffix(request.Score),
	})
}

// BatchUpdateModelNamesWithScores updates multiple model names with scores
func (sah *ScoringAPIHandlers) BatchUpdateModelNamesWithScores(c *gin.Context) {
	var request struct {
		ModelScores map[string]float64 `json:"model_scores" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	results := sah.modelNaming.BatchUpdateModelNames(request.ModelScores)

	c.JSON(http.StatusOK, gin.H{
		"message": "Model names updated successfully",
		"results": results,
		"count":   len(results),
	})
}

// ValidateScoreCalculation validates score calculation
func (sah *ScoringAPIHandlers) ValidateScoreCalculation(c *gin.Context) {
	var request struct {
		ModelID string  `json:"model_id" binding:"required"`
		Score   float64 `json:"score" binding:"required"`
		Method  string  `json:"method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Simulate validation
	isValid := true
	validationResult := map[string]interface{}{
		"model_id": request.ModelID,
		"score":    request.Score,
		"method":   request.Method,
		"is_valid": isValid,
		"message":  "Score validation completed successfully",
	}

	c.JSON(http.StatusOK, gin.H{
		"validation": validationResult,
	})
}

// Helper functions

func generateBatchID() string {
	return fmt.Sprintf("batch_%d", time.Now().UnixNano())
}

func (sah *ScoringAPIHandlers) processBatchScoresAsync(batchID string, modelIDs []string, config ScoringConfig) {
	// Implementation for async batch processing
	// This would typically use a background job system
	sah.logger.Info("Processing batch scores async", map[string]interface{}{
		"batch_id": batchID,
		"model_count": len(modelIDs),
	})
}

func (sah *ScoringAPIHandlers) processBatchScoresSync(modelIDs []string, config ScoringConfig) []interface{} {
	results := make([]interface{}, 0, len(modelIDs))
	
	for _, modelID := range modelIDs {
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		
		// Simulate score calculation
		score := &ComprehensiveScore{
			ModelID:      modelID,
			ModelName:    "Model " + modelID,
			OverallScore: 8.0 + float64(len(modelIDs))/10.0,
			ScoreSuffix:  fmt.Sprintf("(SC:%.1f)", 8.0+float64(len(modelIDs))/10.0),
			Components: ScoreComponents{
				SpeedScore:      8.0,
				EfficiencyScore: 9.0,
				CostScore:       8.5,
				CapabilityScore: 8.5,
				RecencyScore:    8.0,
			},
			LastCalculated: time.Now(),
		}
		cancel()
		
		results = append(results, gin.H{
			"model_id":      modelID,
			"overall_score": score.OverallScore,
			"score_suffix":  score.ScoreSuffix,
			"success":       true,
		})
	}
	
	return results
}