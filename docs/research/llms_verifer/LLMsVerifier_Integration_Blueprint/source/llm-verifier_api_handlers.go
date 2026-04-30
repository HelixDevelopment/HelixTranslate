// Package api contains HTTP API handlers
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"digital.vasic.llmsverifier/database"
)

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	health := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	if s.database != nil {
		health["database"] = "connected"
		if err := s.database.Ping(); err == nil {
			health["database_status"] = "ok"
		} else {
			health["database_status"] = "error"
			health["database_error"] = err.Error()
		}
	} else {
		health["database"] = "not_configured"
	}

	json.NewEncoder(w).Encode(health)
}

// ListModelsHandler handles listing all models
func (s *Server) ListModelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters for filtering
	filters := make(map[string]interface{})
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		if id, err := strconv.ParseInt(providerID, 10, 64); err == nil {
			filters["provider_id"] = id
		}
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters["verification_status"] = status
	}
	if minScore := r.URL.Query().Get("min_score"); minScore != "" {
		if score, err := strconv.ParseFloat(minScore, 64); err == nil {
			filters["min_score"] = score
		}
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}

	// Get models from database
	models, err := s.database.ListModels(filters)
	if err != nil {
		http.Error(w, "Failed to retrieve models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform models to API response format
	modelResponses := make([]map[string]any, 0, len(models))
	for _, model := range models {
		// Get provider name if available
		providerName := ""
		if provider, err := s.database.GetProvider(model.ProviderID); err == nil {
			providerName = provider.Name
		}

		// Build capabilities list from model features
		capabilities := buildCapabilitiesList(model)

		modelResponses = append(modelResponses, map[string]any{
			"id":           model.ID,
			"model_id":     model.ModelID,
			"name":         model.Name,
			"provider":     providerName,
			"provider_id":  model.ProviderID,
			"status":       model.VerificationStatus,
			"score":        model.OverallScore,
			"capabilities": capabilities,
			"description":  model.Description,
			"version":      model.Version,
			"deprecated":   model.Deprecated,
			"created_at":   model.CreatedAt,
			"updated_at":   model.UpdatedAt,
		})
	}

	json.NewEncoder(w).Encode(map[string]any{
		"models": modelResponses,
		"count":  len(modelResponses),
	})
}

// buildCapabilitiesList builds a list of capabilities from model features
func buildCapabilitiesList(model *database.Model) []string {
	capabilities := []string{}
	if model.IsMultimodal {
		capabilities = append(capabilities, "multimodal")
	}
	if model.SupportsVision {
		capabilities = append(capabilities, "vision")
	}
	if model.SupportsAudio {
		capabilities = append(capabilities, "audio")
	}
	if model.SupportsVideo {
		capabilities = append(capabilities, "video")
	}
	if model.SupportsReasoning {
		capabilities = append(capabilities, "reasoning")
	}
	// Add default text capability
	capabilities = append(capabilities, "text")
	return capabilities
}

// GetModelHandler handles getting a single model
func (s *Server) GetModelHandler(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from path: /api/models/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/models/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	// Check for verify suffix
	if strings.HasSuffix(path, "/verify") {
		s.VerifyModelHandler(w, r)
		return
	}

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse model ID
	modelID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get model from database
	model, err := s.database.GetModel(modelID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to retrieve model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get provider name
	providerName := ""
	if provider, err := s.database.GetProvider(model.ProviderID); err == nil {
		providerName = provider.Name
	}

	// Build capabilities list
	capabilities := buildCapabilitiesList(model)

	// Format context window
	contextWindow := ""
	if model.ContextWindowTokens != nil {
		contextWindow = formatTokenCount(*model.ContextWindowTokens)
	}

	// Format parameter count
	parameters := ""
	if model.ParameterCount != nil {
		parameters = formatParameterCount(*model.ParameterCount)
	}

	response := map[string]any{
		"id":                     model.ID,
		"model_id":               model.ModelID,
		"name":                   model.Name,
		"provider":               providerName,
		"provider_id":            model.ProviderID,
		"status":                 model.VerificationStatus,
		"score":                  model.OverallScore,
		"capabilities":           capabilities,
		"description":            model.Description,
		"version":                model.Version,
		"architecture":           model.Architecture,
		"parameters":             parameters,
		"context":                contextWindow,
		"context_window_tokens":  model.ContextWindowTokens,
		"max_output_tokens":      model.MaxOutputTokens,
		"is_multimodal":          model.IsMultimodal,
		"supports_vision":        model.SupportsVision,
		"supports_audio":         model.SupportsAudio,
		"supports_video":         model.SupportsVideo,
		"supports_reasoning":     model.SupportsReasoning,
		"open_source":            model.OpenSource,
		"deprecated":             model.Deprecated,
		"tags":                   model.Tags,
		"use_case":               model.UseCase,
		"code_capability_score":  model.CodeCapabilityScore,
		"responsiveness_score":   model.ResponsivenessScore,
		"reliability_score":      model.ReliabilityScore,
		"feature_richness_score": model.FeatureRichnessScore,
		"last_verified":          model.LastVerified,
		"created_at":             model.CreatedAt,
		"updated_at":             model.UpdatedAt,
	}

	json.NewEncoder(w).Encode(response)
}

// formatTokenCount formats token count to human readable string
func formatTokenCount(tokens int) string {
	if tokens >= 1000000 {
		return strconv.FormatFloat(float64(tokens)/1000000, 'f', 1, 64) + "M tokens"
	}
	if tokens >= 1000 {
		return strconv.Itoa(tokens/1000) + "K tokens"
	}
	return strconv.Itoa(tokens) + " tokens"
}

// formatParameterCount formats parameter count to human readable string
func formatParameterCount(params int64) string {
	if params >= 1000000000000 {
		return strconv.FormatFloat(float64(params)/1000000000000, 'f', 2, 64) + " trillion"
	}
	if params >= 1000000000 {
		return strconv.FormatFloat(float64(params)/1000000000, 'f', 1, 64) + " billion"
	}
	if params >= 1000000 {
		return strconv.FormatFloat(float64(params)/1000000, 'f', 1, 64) + " million"
	}
	return strconv.FormatInt(params, 10)
}

// VerifyModelHandler handles model verification
func (s *Server) VerifyModelHandler(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from path: /api/models/{id}/verify
	path := strings.TrimPrefix(r.URL.Path, "/api/models/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	modelIDStr := parts[0]

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse model ID
	modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	// Verify model exists
	model, err := s.database.GetModel(modelID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to retrieve model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new verification result record
	verificationResult := &database.VerificationResult{
		ModelID:          modelID,
		VerificationType: "comprehensive",
		StartedAt:        time.Now(),
		Status:           "running",
	}

	err = s.database.CreateVerificationResult(verificationResult)
	if err != nil {
		http.Error(w, "Failed to create verification job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]any{
		"status":          "verification_started",
		"model_id":        modelID,
		"model_name":      model.Name,
		"message":         "Verification process initiated",
		"job_id":          verificationResult.ID,
		"verification_id": verificationResult.ID,
		"started_at":      verificationResult.StartedAt,
	})
}

// ListProvidersHandler handles listing all providers
func (s *Server) ListProvidersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters for filtering
	filters := make(map[string]interface{})
	if isActive := r.URL.Query().Get("is_active"); isActive != "" {
		filters["is_active"] = isActive == "true"
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}

	// Get providers from database
	providers, err := s.database.ListProviders(filters)
	if err != nil {
		http.Error(w, "Failed to retrieve providers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform providers to API response format
	providerResponses := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		// Count models for this provider
		modelFilters := map[string]interface{}{"provider_id": provider.ID}
		models, _ := s.database.ListModels(modelFilters)
		modelCount := len(models)

		status := "inactive"
		if provider.IsActive {
			status = "active"
		}

		providerResponses = append(providerResponses, map[string]any{
			"id":                       provider.ID,
			"name":                     provider.Name,
			"status":                   status,
			"is_active":                provider.IsActive,
			"models":                   modelCount,
			"api_url":                  provider.Endpoint,
			"endpoint":                 provider.Endpoint,
			"description":              provider.Description,
			"website":                  provider.Website,
			"support_email":            provider.SupportEmail,
			"documentation_url":        provider.DocumentationURL,
			"reliability_score":        provider.ReliabilityScore,
			"average_response_time_ms": provider.AverageResponseTimeMs,
			"last_checked":             provider.LastChecked,
			"created_at":               provider.CreatedAt,
			"updated_at":               provider.UpdatedAt,
		})
	}

	json.NewEncoder(w).Encode(map[string]any{
		"providers": providerResponses,
		"count":     len(providerResponses),
	})
}

// ProvidersHandler handles both GET (list) and POST (add) for providers
func (s *Server) ProvidersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.ListProvidersHandler(w, r)
	case http.MethodPost:
		s.AddProviderHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AddProviderHandler handles adding a new provider
func (s *Server) AddProviderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	var providerData map[string]any
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&providerData); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	name, nameOK := providerData["name"].(string)
	endpoint, endpointOK := providerData["endpoint"].(string)
	if !nameOK || name == "" {
		http.Error(w, "Missing required field: name", http.StatusBadRequest)
		return
	}
	if !endpointOK || endpoint == "" {
		// Use api_url as fallback
		if apiURL, ok := providerData["api_url"].(string); ok && apiURL != "" {
			endpoint = apiURL
		} else {
			http.Error(w, "Missing required field: endpoint or api_url", http.StatusBadRequest)
			return
		}
	}

	// Create provider object
	provider := &database.Provider{
		Name:     name,
		Endpoint: endpoint,
		IsActive: true,
	}

	// Optional fields
	if desc, ok := providerData["description"].(string); ok {
		provider.Description = desc
	}
	if website, ok := providerData["website"].(string); ok {
		provider.Website = website
	}
	if email, ok := providerData["support_email"].(string); ok {
		provider.SupportEmail = email
	}
	if docURL, ok := providerData["documentation_url"].(string); ok {
		provider.DocumentationURL = docURL
	}
	if isActive, ok := providerData["is_active"].(bool); ok {
		provider.IsActive = isActive
	}

	// Create in database
	err := s.database.CreateProvider(provider)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			http.Error(w, "Provider with this name already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create provider: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]any{
		"status":     "provider_added",
		"id":         provider.ID,
		"name":       provider.Name,
		"endpoint":   provider.Endpoint,
		"is_active":  provider.IsActive,
		"created_at": provider.CreatedAt,
	})
}
