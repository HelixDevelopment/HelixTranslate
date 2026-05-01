package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/services"
	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/scoring"
	"digital.vasic.translator/internal/verifier/selection"
)

// VerifierHandler provides API endpoints for LLMsVerifier integration.
type VerifierHandler struct {
	config   *verifier.Config
	client   *verifier.Client
	registry *verifier.Registry
	scoring  *scoring.Engine
	adapter  *services.LLMsVerifierScoreAdapter
	selector *selection.Engine
}

// NewVerifierHandler creates a verifier API handler.
func NewVerifierHandler(cfg *verifier.Config) *VerifierHandler {
	client := verifier.NewClient(cfg)
	registry := verifier.NewRegistry()
	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     cfg.ScoringWeights.ResponseSpeed,
		CostEffectiveness: cfg.ScoringWeights.CostEffectiveness,
		ModelEfficiency:   cfg.ScoringWeights.ModelEfficiency,
		Capability:        cfg.ScoringWeights.Capability,
		Recency:           cfg.ScoringWeights.Recency,
	})
	adapter := services.NewLLMsVerifierScoreAdapter(client, scoreEngine, cfg)
	selector := selection.NewEngine(registry, scoreEngine, cfg)

	return &VerifierHandler{
		config:   cfg,
		client:   client,
		registry: registry,
		scoring:  scoreEngine,
		adapter:  adapter,
		selector: selector,
	}
}

// RegisterVerifierRoutes adds LLMsVerifier routes to the router.
func (h *VerifierHandler) RegisterVerifierRoutes(router *gin.RouterGroup) {
	router.GET("/verified-models", h.listVerifiedModels)
	router.GET("/verified-models/:id", h.getVerifiedModel)
	router.GET("/verification-status", h.getVerificationStatus)
	router.POST("/verification/refresh", h.refreshVerification)
	router.GET("/providers/verified", h.listVerifiedProviders)
	router.POST("/translate-with-verification", h.translateWithVerification)
}

// listVerifiedModels returns all LLMsVerifier-verified models.
func (h *VerifierHandler) listVerifiedModels(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	models, err := h.client.GetVerifiedModels(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "failed to fetch verified models",
			"details": err.Error(),
		})
		return
	}

	// Filter to only verified models with positive scores
	var result []gin.H
	for _, m := range models {
		if m.VerificationStatus != "verified" || !m.CanSeeCode || !m.AffirmativeResponse {
			continue
		}
		if m.OverallScore <= h.config.MinScoreThreshold {
			continue
		}
		result = append(result, gin.H{
			"id":                 m.ID,
			"provider_id":        m.ProviderID,
			"name":               m.Name,
			"verification_status": m.VerificationStatus,
			"overall_score":      m.OverallScore,
			"capabilities":       m.Capabilities,
			"pricing":            m.Pricing,
			"last_verified_at":   m.LastVerifiedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": result,
		"count":  len(result),
	})
}

// getVerifiedModel returns details for a single verified model.
func (h *VerifierHandler) getVerifiedModel(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model ID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	model, err := h.client.GetModel(ctx, modelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                   model.ID,
		"provider_id":          model.ProviderID,
		"name":                 model.Name,
		"verification_status":  model.VerificationStatus,
		"can_see_code":         model.CanSeeCode,
		"affirmative_response": model.AffirmativeResponse,
		"overall_score":        model.OverallScore,
		"responsiveness_score": model.ResponsivenessScore,
		"code_capability_score": model.CodeCapabilityScore,
		"feature_richness_score": model.FeatureRichnessScore,
		"reliability_score":    model.ReliabilityScore,
		"capabilities":         model.Capabilities,
		"pricing":              model.Pricing,
		"last_verified_at":     model.LastVerifiedAt,
	})
}

// getVerificationStatus returns the LLMsVerifier sync status.
func (h *VerifierHandler) getVerificationStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := h.client.Ping(ctx)
	connected := err == nil

	c.JSON(http.StatusOK, gin.H{
		"llmsverifier_connected": connected,
		"last_score_refresh":     h.adapter.LastRefresh(),
		"cache_ttl_seconds":      h.config.CacheTTL.Seconds(),
		"min_score_threshold":    h.config.MinScoreThreshold,
		"strict_mode":            h.config.StrictMode,
	})
}

// refreshVerification forces a refresh of model scores from LLMsVerifier.
func (h *VerifierHandler) refreshVerification(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := h.adapter.RefreshScores(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "failed to refresh scores",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "refreshed",
		"last_refresh":     h.adapter.LastRefresh(),
	})
}

// listVerifiedProviders returns providers that have verified models.
func (h *VerifierHandler) listVerifiedProviders(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	models, err := h.client.GetVerifiedModels(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	providerMap := make(map[string]gin.H)
	for _, m := range models {
		if m.VerificationStatus != "verified" {
			continue
		}
		if _, exists := providerMap[m.ProviderID]; !exists {
			providerMap[m.ProviderID] = gin.H{
				"provider_id":   m.ProviderID,
				"models_count":  0,
				"highest_score": 0.0,
			}
		}
		info := providerMap[m.ProviderID]
		info["models_count"] = info["models_count"].(int) + 1
		if m.OverallScore > info["highest_score"].(float64) {
			info["highest_score"] = m.OverallScore
		}
		providerMap[m.ProviderID] = info
	}

	var result []gin.H
	for _, info := range providerMap {
		result = append(result, info)
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": result,
		"count":     len(result),
	})
}

// translateWithVerification performs translation using only verified models.
func (h *VerifierHandler) translateWithVerification(c *gin.Context) {
	var req struct {
		Text       string  `json:"text" binding:"required"`
		SourceLang string  `json:"source_lang" binding:"required"`
		TargetLang string  `json:"target_lang" binding:"required"`
		ModelID    string  `json:"model_id,omitempty"`
		Provider   string  `json:"provider,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// If model ID specified, verify it; otherwise select best model
	var selectedModel *verifier.Model
	var err error

	if req.ModelID != "" {
		selectedModel, err = h.client.GetModel(ctx, req.ModelID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		selectedModel, err = h.selector.SelectModel(ctx, selection.TaskRequirements{
			SourceLang: req.SourceLang,
			TargetLang: req.TargetLang,
		})
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "translation_initiated",
		"model_id":     selectedModel.ID,
		"model_name":   selectedModel.Name,
		"provider_id":  selectedModel.ProviderID,
		"score":        selectedModel.OverallScore,
		"source_lang":  req.SourceLang,
		"target_lang":  req.TargetLang,
		"text_preview": req.Text,
	})
}

// InitVerifierFromConfig creates a VerifierHandler from the global config.
func InitVerifierFromConfig(cfg *config.Config) *VerifierHandler {
	vCfg := &verifier.Config{
		APIURL:            cfg.LLMsVerifier.APIURL,
		APIKey:            cfg.LLMsVerifier.APIKey,
		CacheTTL:          cfg.LLMsVerifier.CacheTTL,
		StrictMode:        cfg.LLMsVerifier.StrictMode,
		MinScoreThreshold: cfg.LLMsVerifier.MinScoreThreshold,
		MaxProviders:      cfg.LLMsVerifier.MaxProviders,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed:     cfg.LLMsVerifier.ScoringWeights.ResponseSpeed,
			CostEffectiveness: cfg.LLMsVerifier.ScoringWeights.CostEffectiveness,
			ModelEfficiency:   cfg.LLMsVerifier.ScoringWeights.ModelEfficiency,
			Capability:        cfg.LLMsVerifier.ScoringWeights.Capability,
			Recency:           cfg.LLMsVerifier.ScoringWeights.Recency,
		},
	}
	return NewVerifierHandler(vCfg)
}
