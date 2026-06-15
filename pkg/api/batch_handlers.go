package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/batch"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TranslateStringRequest represents a string translation request
type TranslateStringRequest struct {
	Text           string `json:"text" binding:"required"`
	SourceLanguage string `json:"source_language,omitempty"`
	TargetLanguage string `json:"target_language" binding:"required"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
}

// TranslateStringResponse represents a string translation response
type TranslateStringResponse struct {
	TranslatedText string  `json:"translated_text"`
	SourceLanguage string  `json:"source_language"`
	TargetLanguage string  `json:"target_language"`
	Provider       string  `json:"provider"`
	Duration       float64 `json:"duration_seconds"`
	SessionID      string  `json:"session_id"`
}

// TranslateDirectoryRequest represents a directory translation request
type TranslateDirectoryRequest struct {
	InputPath      string `json:"input_path" binding:"required"`
	OutputPath     string `json:"output_path"`
	SourceLanguage string `json:"source_language,omitempty"`
	TargetLanguage string `json:"target_language" binding:"required"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	Recursive      bool   `json:"recursive"`
	Parallel       bool   `json:"parallel"`
	MaxConcurrency int    `json:"max_concurrency,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
}

// TranslateDirectoryResponse represents a directory translation response
type TranslateDirectoryResponse struct {
	SessionID  string       `json:"session_id"`
	TotalFiles int          `json:"total_files"`
	Successful int          `json:"successful"`
	Failed     int          `json:"failed"`
	Duration   float64      `json:"duration_seconds"`
	Results    []FileResult `json:"results"`
}

// FileResult represents the result of a single file translation
type FileResult struct {
	InputPath  string `json:"input_path"`
	OutputPath string `json:"output_path"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// HandleTranslateString handles string translation requests
func (h *Handler) HandleTranslateString(c *gin.Context) {
	var req TranslateStringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Parse target language
	targetLang, err := language.ParseLanguage(req.TargetLanguage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid target language: %v", err)})
		return
	}

	// Parse source language if provided
	var sourceLang language.Language
	if req.SourceLanguage != "" {
		sourceLang, err = language.ParseLanguage(req.SourceLanguage)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid source language: %v", err)})
			return
		}
	}

	// Create translator
	var trans translator.Translator

	provider := req.Provider
	if provider == "" {
		provider = h.config.Translation.DefaultProvider
		if provider == "" {
			provider = "openai"
		}
	}

	// R-1b/R2: route translator construction through the LLMsVerifier bridge —
	// the strongest verified model for this language pair — instead of a local
	// NewLLMTranslator. The provider switch is retained ONLY as request
	// validation (reject an unsupported provider with a 400); the actual model is
	// selected by the bridge, and on no provider keys the bridge hard-errors
	// (no silent local fallback, §11.4.69).
	switch provider {
	case "openai", "anthropic", "zhipu", "deepseek", "mock":
		trans, err = h.bridgeFor()(c.Request.Context(), selection.TaskRequirements{
			SourceLang: sourceLang.Code,
			TargetLang: targetLang.Code,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to create translator: %v", err)})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported provider: %s", provider)})
		return
	}

	// Emit start event
	h.eventBus.Publish(events.Event{
		Type:      events.EventTranslationStarted,
		SessionID: sessionID,
		Message:   "String translation started",
		Data: map[string]interface{}{
			"text_length":     len(req.Text),
			"source_language": req.SourceLanguage,
			"target_language": req.TargetLanguage,
			"provider":        provider,
		},
	})

	// Translate
	startTime := time.Now()
	ctx := context.Background()
	translatedText, err := trans.Translate(ctx, req.Text, "")
	duration := time.Since(startTime).Seconds()

	if err != nil {
		h.eventBus.Publish(events.Event{
			Type:      events.EventTranslationError,
			SessionID: sessionID,
			Message:   fmt.Sprintf("Translation failed: %v", err),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("translation failed: %v", err)})
		return
	}

	// Emit completion event
	h.eventBus.Publish(events.Event{
		Type:      events.EventTranslationCompleted,
		SessionID: sessionID,
		Message:   "String translation completed",
		Data: map[string]interface{}{
			"duration": duration,
		},
	})

	// Return response
	c.JSON(http.StatusOK, TranslateStringResponse{
		TranslatedText: translatedText,
		SourceLanguage: sourceLang.Code,
		TargetLanguage: targetLang.Code,
		Provider:       provider,
		Duration:       duration,
		SessionID:      sessionID,
	})
}

// HandleTranslateDirectory handles directory translation requests
func (h *Handler) HandleTranslateDirectory(c *gin.Context) {
	var req TranslateDirectoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Parse target language
	targetLang, err := language.ParseLanguage(req.TargetLanguage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid target language: %v", err)})
		return
	}

	// Parse source language if provided
	var sourceLang language.Language
	if req.SourceLanguage != "" {
		sourceLang, err = language.ParseLanguage(req.SourceLanguage)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid source language: %v", err)})
			return
		}
	}

	// Create translator
	var trans translator.Translator
	provider := req.Provider
	if provider == "" {
		provider = h.config.Translation.DefaultProvider
		if provider == "" {
			provider = "openai"
		}
	}

	// R-1b/R2: route translator construction through the LLMsVerifier bridge (the
	// strongest verified model for this language pair) instead of a local
	// NewLLMTranslator. The provider switch is retained ONLY as request validation;
	// the bridge selects the model and hard-errors on no provider keys (no silent
	// local fallback, §11.4.69).
	switch provider {
	case "openai", "anthropic", "zhipu", "deepseek", "mock":
		trans, err = h.bridgeFor()(c.Request.Context(), selection.TaskRequirements{
			SourceLang: sourceLang.Code,
			TargetLang: targetLang.Code,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to create translator: %v", err)})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported provider: %s", provider)})
		return
	}

	// Create batch processor
	options := &batch.ProcessingOptions{
		InputType:      batch.InputTypeDirectory,
		InputPath:      req.InputPath,
		OutputPath:     req.OutputPath,
		OutputFormat:   req.OutputFormat,
		SourceLanguage: sourceLang,
		TargetLanguage: targetLang,
		Provider:       provider,
		Model:          req.Model,
		Translator:     trans,
		Recursive:      req.Recursive,
		Parallel:       req.Parallel,
		MaxConcurrency: req.MaxConcurrency,
		EventBus:       h.eventBus,
		SessionID:      sessionID,
	}

	processor := batch.NewBatchProcessor(options)

	// Emit start event
	h.eventBus.Publish(events.Event{
		Type:      events.EventTranslationStarted,
		SessionID: sessionID,
		Message:   "Directory translation started",
		Data: map[string]interface{}{
			"input_path":      req.InputPath,
			"output_path":     req.OutputPath,
			"target_language": req.TargetLanguage,
			"provider":        provider,
			"recursive":       req.Recursive,
			"parallel":        req.Parallel,
		},
	})

	// Process directory
	startTime := time.Now()
	ctx := context.Background()
	results, err := processor.Process(ctx)
	duration := time.Since(startTime).Seconds()

	if err != nil {
		h.eventBus.Publish(events.Event{
			Type:      events.EventTranslationError,
			SessionID: sessionID,
			Message:   fmt.Sprintf("Directory translation failed: %v", err),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("directory translation failed: %v", err)})
		return
	}

	// Count successes and failures
	successful := 0
	failed := 0
	fileResults := make([]FileResult, len(results))
	for i, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}

		errMsg := ""
		if result.Error != nil {
			errMsg = result.Error.Error()
		}

		fileResults[i] = FileResult{
			InputPath:  result.InputPath,
			OutputPath: result.OutputPath,
			Success:    result.Success,
			Error:      errMsg,
		}
	}

	// Emit completion event
	h.eventBus.Publish(events.Event{
		Type:      events.EventTranslationCompleted,
		SessionID: sessionID,
		Message:   fmt.Sprintf("Directory translation completed: %d successful, %d failed", successful, failed),
		Data: map[string]interface{}{
			"total":      len(results),
			"successful": successful,
			"failed":     failed,
			"duration":   duration,
		},
	})

	// Return response
	c.JSON(http.StatusOK, TranslateDirectoryResponse{
		SessionID:  sessionID,
		TotalFiles: len(results),
		Successful: successful,
		Failed:     failed,
		Duration:   duration,
		Results:    fileResults,
	})
}

// RegisterBatchRoutes registers batch translation routes
func (h *Handler) RegisterBatchRoutes(router *gin.RouterGroup) {
	router.POST("/translate/string", h.HandleTranslateString)
	router.POST("/translate/directory", h.HandleTranslateDirectory)
}
