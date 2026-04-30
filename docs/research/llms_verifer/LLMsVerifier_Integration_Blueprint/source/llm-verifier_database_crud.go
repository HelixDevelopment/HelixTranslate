package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// ==================== Provider CRUD Operations ====================

// CreateProvider creates a new provider
func (d *Database) CreateProvider(provider *Provider) error {
	query := `
		INSERT INTO providers (
			name, endpoint, api_key_encrypted, description, website, 
			support_email, documentation_url, is_active, reliability_score, 
			average_response_time_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		provider.Name,
		provider.Endpoint,
		provider.APIKeyEncrypted,
		provider.Description,
		provider.Website,
		provider.SupportEmail,
		provider.DocumentationURL,
		provider.IsActive,
		provider.ReliabilityScore,
		provider.AverageResponseTimeMs,
	)

	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	provider.ID = id
	return nil
}

// GetProvider retrieves a provider by ID
func (d *Database) GetProvider(id int64) (*Provider, error) {
	query := `
		SELECT id, name, endpoint, api_key_encrypted, description, website,
			support_email, documentation_url, created_at, updated_at, last_checked,
			is_active, reliability_score, average_response_time_ms
		FROM providers WHERE id = ?
	`

	var provider Provider
	var lastChecked sql.NullTime

	err := d.conn.QueryRow(query, id).Scan(
		&provider.ID,
		&provider.Name,
		&provider.Endpoint,
		&provider.APIKeyEncrypted,
		&provider.Description,
		&provider.Website,
		&provider.SupportEmail,
		&provider.DocumentationURL,
		&provider.CreatedAt,
		&provider.UpdatedAt,
		&lastChecked,
		&provider.IsActive,
		&provider.ReliabilityScore,
		&provider.AverageResponseTimeMs,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	provider.LastChecked = scanNullableTime(lastChecked)
	return &provider, nil
}

// GetProviderByName retrieves a provider by name
func (d *Database) GetProviderByName(name string) (*Provider, error) {
	query := `
		SELECT id, name, endpoint, api_key_encrypted, description, website,
			support_email, documentation_url, created_at, updated_at, last_checked,
			is_active, reliability_score, average_response_time_ms
		FROM providers WHERE name = ?
	`

	var provider Provider
	var lastChecked sql.NullTime

	err := d.conn.QueryRow(query, name).Scan(
		&provider.ID,
		&provider.Name,
		&provider.Endpoint,
		&provider.APIKeyEncrypted,
		&provider.Description,
		&provider.Website,
		&provider.SupportEmail,
		&provider.DocumentationURL,
		&provider.CreatedAt,
		&provider.UpdatedAt,
		&lastChecked,
		&provider.IsActive,
		&provider.ReliabilityScore,
		&provider.AverageResponseTimeMs,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get provider by name: %w", err)
	}

	provider.LastChecked = scanNullableTime(lastChecked)
	return &provider, nil
}

// UpdateProvider updates an existing provider
func (d *Database) UpdateProvider(provider *Provider) error {
	query := `
		UPDATE providers SET
			name = ?, endpoint = ?, api_key_encrypted = ?, description = ?,
			website = ?, support_email = ?, documentation_url = ?, 
			last_checked = ?, is_active = ?, reliability_score = ?, 
			average_response_time_ms = ?
		WHERE id = ?
	`

	var lastChecked sql.NullTime
	if provider.LastChecked != nil {
		lastChecked.Valid = true
		lastChecked.Time = *provider.LastChecked
	}

	_, err := d.conn.Exec(query,
		provider.Name,
		provider.Endpoint,
		provider.APIKeyEncrypted,
		provider.Description,
		provider.Website,
		provider.SupportEmail,
		provider.DocumentationURL,
		lastChecked,
		provider.IsActive,
		provider.ReliabilityScore,
		provider.AverageResponseTimeMs,
		provider.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	return nil
}

// DeleteProvider deletes a provider by ID
func (d *Database) DeleteProvider(id int64) error {
	query := `DELETE FROM providers WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	return nil
}

// ListProviders retrieves all providers with optional filtering
func (d *Database) ListProviders(filters map[string]interface{}) ([]*Provider, error) {
	query := `
		SELECT id, name, endpoint, api_key_encrypted, description, website,
			support_email, documentation_url, created_at, updated_at, last_checked,
			is_active, reliability_score, average_response_time_ms
		FROM providers
	`

	var conditions []string
	var args []interface{}

	if isActive, ok := filters["is_active"]; ok {
		conditions = append(conditions, "is_active = ?")
		args = append(args, isActive)
	}

	if search, ok := filters["search"]; ok {
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		searchPattern := fmt.Sprintf("%%%s%%", search)
		args = append(args, searchPattern, searchPattern)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		var provider Provider
		var lastChecked sql.NullTime

		err := rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.Endpoint,
			&provider.APIKeyEncrypted,
			&provider.Description,
			&provider.Website,
			&provider.SupportEmail,
			&provider.DocumentationURL,
			&provider.CreatedAt,
			&provider.UpdatedAt,
			&lastChecked,
			&provider.IsActive,
			&provider.ReliabilityScore,
			&provider.AverageResponseTimeMs,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}

		provider.LastChecked = scanNullableTime(lastChecked)
		providers = append(providers, &provider)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating providers: %w", err)
	}

	return providers, nil
}

// ==================== Model CRUD Operations ====================

// CreateModel creates a new model
func (d *Database) CreateModel(model *Model) error {
	tagsJSON, err := json.Marshal(model.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	langSupportJSON, err := json.Marshal(model.LanguageSupport)
	if err != nil {
		return fmt.Errorf("failed to marshal language support: %w", err)
	}

	query := `
		INSERT INTO models (
			provider_id, model_id, name, description, version, architecture,
			parameter_count, context_window_tokens, max_output_tokens,
			training_data_cutoff, release_date, is_multimodal, supports_vision,
			supports_audio, supports_video, supports_reasoning, open_source,
			deprecated, tags, language_support, use_case, verification_status,
			overall_score, code_capability_score, responsiveness_score,
			reliability_score, feature_richness_score, value_proposition_score
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		model.ProviderID,
		model.ModelID,
		model.Name,
		model.Description,
		model.Version,
		model.Architecture,
		model.ParameterCount,
		model.ContextWindowTokens,
		model.MaxOutputTokens,
		model.TrainingDataCutoff,
		model.ReleaseDate,
		model.IsMultimodal,
		model.SupportsVision,
		model.SupportsAudio,
		model.SupportsVideo,
		model.SupportsReasoning,
		model.OpenSource,
		model.Deprecated,
		string(tagsJSON),
		string(langSupportJSON),
		model.UseCase,
		model.VerificationStatus,
		model.OverallScore,
		model.CodeCapabilityScore,
		model.ResponsivenessScore,
		model.ReliabilityScore,
		model.FeatureRichnessScore,
		model.ValuePropositionScore,
	)

	if err != nil {
		return fmt.Errorf("failed to create model: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	model.ID = id
	return nil
}

// GetModel retrieves a model by ID
func (d *Database) GetModel(id int64) (*Model, error) {
	query := `
		SELECT id, provider_id, model_id, name, description, version, architecture,
			parameter_count, context_window_tokens, max_output_tokens,
			training_data_cutoff, release_date, is_multimodal, supports_vision,
			supports_audio, supports_video, supports_reasoning, open_source,
			deprecated, tags, language_support, use_case, created_at, updated_at,
			last_verified, verification_status, overall_score, code_capability_score,
			responsiveness_score, reliability_score, feature_richness_score,
			value_proposition_score
		FROM models WHERE id = ?
	`

	var model Model
	var tagsJSON, langSupportJSON sql.NullString
	var trainingDataCutoff, releaseDate, lastVerified sql.NullTime
	var parameterCount, contextWindowTokens, maxOutputTokens sql.NullInt64

	err := d.conn.QueryRow(query, id).Scan(
		&model.ID,
		&model.ProviderID,
		&model.ModelID,
		&model.Name,
		&model.Description,
		&model.Version,
		&model.Architecture,
		&parameterCount,
		&contextWindowTokens,
		&maxOutputTokens,
		&trainingDataCutoff,
		&releaseDate,
		&model.IsMultimodal,
		&model.SupportsVision,
		&model.SupportsAudio,
		&model.SupportsVideo,
		&model.SupportsReasoning,
		&model.OpenSource,
		&model.Deprecated,
		&tagsJSON,
		&langSupportJSON,
		&model.UseCase,
		&model.CreatedAt,
		&model.UpdatedAt,
		&lastVerified,
		&model.VerificationStatus,
		&model.OverallScore,
		&model.CodeCapabilityScore,
		&model.ResponsivenessScore,
		&model.ReliabilityScore,
		&model.FeatureRichnessScore,
		&model.ValuePropositionScore,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	model.ParameterCount = scanNullableInt64(parameterCount)
	model.ContextWindowTokens = scanNullableIntFromInt64(contextWindowTokens)
	model.MaxOutputTokens = scanNullableIntFromInt64(maxOutputTokens)
	model.TrainingDataCutoff = scanNullableTime(trainingDataCutoff)
	model.ReleaseDate = scanNullableTime(releaseDate)
	model.Tags = scanJSONString(tagsJSON)
	model.LanguageSupport = scanJSONString(langSupportJSON)
	model.LastVerified = scanNullableTime(lastVerified)

	return &model, nil
}

// UpdateModel updates an existing model
func (d *Database) UpdateModel(model *Model) error {
	tagsJSON, err := json.Marshal(model.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	langSupportJSON, err := json.Marshal(model.LanguageSupport)
	if err != nil {
		return fmt.Errorf("failed to marshal language support: %w", err)
	}

	query := `
		UPDATE models SET
			provider_id = ?, model_id = ?, name = ?, description = ?, version = ?,
			architecture = ?, parameter_count = ?, context_window_tokens = ?,
			max_output_tokens = ?, training_data_cutoff = ?, release_date = ?,
			is_multimodal = ?, supports_vision = ?, supports_audio = ?,
			supports_video = ?, supports_reasoning = ?, open_source = ?,
			deprecated = ?, tags = ?, language_support = ?, use_case = ?,
			last_verified = ?, verification_status = ?, overall_score = ?,
			code_capability_score = ?, responsiveness_score = ?, reliability_score = ?,
			feature_richness_score = ?, value_proposition_score = ?
		WHERE id = ?
	`

	_, err = d.conn.Exec(query,
		model.ProviderID,
		model.ModelID,
		model.Name,
		model.Description,
		model.Version,
		model.Architecture,
		model.ParameterCount,
		model.ContextWindowTokens,
		model.MaxOutputTokens,
		model.TrainingDataCutoff,
		model.ReleaseDate,
		model.IsMultimodal,
		model.SupportsVision,
		model.SupportsAudio,
		model.SupportsVideo,
		model.SupportsReasoning,
		model.OpenSource,
		model.Deprecated,
		string(tagsJSON),
		string(langSupportJSON),
		model.UseCase,
		model.LastVerified,
		model.VerificationStatus,
		model.OverallScore,
		model.CodeCapabilityScore,
		model.ResponsivenessScore,
		model.ReliabilityScore,
		model.FeatureRichnessScore,
		model.ValuePropositionScore,
		model.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	return nil
}

// DeleteModel deletes a model by ID
func (d *Database) DeleteModel(id int64) error {
	query := `DELETE FROM models WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	return nil
}

// ListModels retrieves models with optional filtering
func (d *Database) ListModels(filters map[string]interface{}) ([]*Model, error) {
	query := `
		SELECT id, provider_id, model_id, name, description, version, architecture,
			parameter_count, context_window_tokens, max_output_tokens,
			training_data_cutoff, release_date, is_multimodal, supports_vision,
			supports_audio, supports_video, supports_reasoning, open_source,
			deprecated, tags, language_support, use_case, created_at, updated_at,
			last_verified, verification_status, overall_score, code_capability_score,
			responsiveness_score, reliability_score, feature_richness_score,
			value_proposition_score
		FROM models
	`

	var conditions []string
	var args []interface{}

	if providerID, ok := filters["provider_id"]; ok {
		conditions = append(conditions, "provider_id = ?")
		args = append(args, providerID)
	}

	if modelID, ok := filters["model_id"]; ok {
		conditions = append(conditions, "model_id = ?")
		args = append(args, modelID)
	}

	if verificationStatus, ok := filters["verification_status"]; ok {
		conditions = append(conditions, "verification_status = ?")
		args = append(args, verificationStatus)
	}

	if minScore, ok := filters["min_score"]; ok {
		conditions = append(conditions, "overall_score >= ?")
		args = append(args, minScore)
	}

	if search, ok := filters["search"]; ok {
		conditions = append(conditions, "(name LIKE ? OR description LIKE ?)")
		searchPattern := fmt.Sprintf("%%%s%%", search)
		args = append(args, searchPattern, searchPattern)
	}

	if supportsToolUse, ok := filters["supports_tool_use"]; ok {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM verification_results vr WHERE vr.model_id = models.id AND vr.supports_tool_use = ?)")
		args = append(args, supportsToolUse)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	if limit, ok := filters["limit"]; ok {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer rows.Close()

	var models []*Model
	for rows.Next() {
		var model Model
		var tagsJSON, langSupportJSON sql.NullString
		var trainingDataCutoff, releaseDate, lastVerified sql.NullTime
		var parameterCount, contextWindowTokens, maxOutputTokens sql.NullInt64

		err := rows.Scan(
			&model.ID,
			&model.ProviderID,
			&model.ModelID,
			&model.Name,
			&model.Description,
			&model.Version,
			&model.Architecture,
			&parameterCount,
			&contextWindowTokens,
			&maxOutputTokens,
			&trainingDataCutoff,
			&releaseDate,
			&model.IsMultimodal,
			&model.SupportsVision,
			&model.SupportsAudio,
			&model.SupportsVideo,
			&model.SupportsReasoning,
			&model.OpenSource,
			&model.Deprecated,
			&tagsJSON,
			&langSupportJSON,
			&model.UseCase,
			&model.CreatedAt,
			&model.UpdatedAt,
			&lastVerified,
			&model.VerificationStatus,
			&model.OverallScore,
			&model.CodeCapabilityScore,
			&model.ResponsivenessScore,
			&model.ReliabilityScore,
			&model.FeatureRichnessScore,
			&model.ValuePropositionScore,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}

		model.ParameterCount = scanNullableInt64(parameterCount)
		model.ContextWindowTokens = scanNullableIntFromInt64(contextWindowTokens)
		model.MaxOutputTokens = scanNullableIntFromInt64(maxOutputTokens)
		model.TrainingDataCutoff = scanNullableTime(trainingDataCutoff)
		model.ReleaseDate = scanNullableTime(releaseDate)
		model.Tags = scanJSONString(tagsJSON)
		model.LanguageSupport = scanJSONString(langSupportJSON)
		model.LastVerified = scanNullableTime(lastVerified)

		models = append(models, &model)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating models: %w", err)
	}

	return models, nil
}

// ==================== Verification Result CRUD Operations ====================

// CreateVerificationResult creates a new verification result
func (d *Database) CreateVerificationResult(verificationResult *VerificationResult) error {
	langSupportJSON, err := json.Marshal(verificationResult.CodeLanguageSupport)
	if err != nil {
		return fmt.Errorf("failed to marshal code language support: %w", err)
	}

	query := `
		INSERT INTO verification_results (
			model_id, verification_type, started_at, completed_at, status, error_message,
			model_exists, responsive, overloaded, latency_ms, supports_tool_use,
			supports_function_calling, supports_code_generation, supports_code_completion,
			supports_code_review, supports_code_explanation, supports_embeddings,
			supports_reranking, supports_image_generation, supports_audio_generation,
						supports_video_generation, supports_mcps, supports_lsps, supports_acps, supports_multimodal,
			supports_streaming, supports_json_mode, supports_structured_output,
			supports_reasoning, supports_parallel_tool_use, max_parallel_calls,
	supports_batch_processing, supports_brotli, code_language_support, code_debugging,
	code_optimization, test_generation, documentation_generation, refactoring,
	error_resolution, architecture_design, security_assessment,
	pattern_recognition, debugging_accuracy, max_handled_depth,
	code_quality_score, logic_correctness_score, runtime_efficiency_score,
	overall_score, code_capability_score, responsiveness_score,
	reliability_score, feature_richness_score, value_proposition_score,
	score_details, avg_latency_ms, p95_latency_ms, min_latency_ms,
	max_latency_ms, throughput_rps, raw_request, raw_response, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		verificationResult.ModelID,
		verificationResult.VerificationType,
		verificationResult.StartedAt,
		verificationResult.CompletedAt,
		verificationResult.Status,
		verificationResult.ErrorMessage,
		verificationResult.ModelExists,
		verificationResult.Responsive,
		verificationResult.Overloaded,
		verificationResult.LatencyMs,
		verificationResult.SupportsToolUse,
		verificationResult.SupportsFunctionCalling,
		verificationResult.SupportsCodeGeneration,
		verificationResult.SupportsCodeCompletion,
		verificationResult.SupportsCodeReview,
		verificationResult.SupportsCodeExplanation,
		verificationResult.SupportsEmbeddings,
		verificationResult.SupportsReranking,
		verificationResult.SupportsImageGeneration,
		verificationResult.SupportsAudioGeneration,
		verificationResult.SupportsVideoGeneration,
		verificationResult.SupportsMCPs,
		verificationResult.SupportsLSPs,
		verificationResult.SupportsACPs,
		verificationResult.SupportsMultimodal,
		verificationResult.SupportsStreaming,
		verificationResult.SupportsJSONMode,
		verificationResult.SupportsStructuredOutput,
		verificationResult.SupportsReasoning,
		verificationResult.SupportsParallelToolUse,
		verificationResult.MaxParallelCalls,
		verificationResult.SupportsBatchProcessing,
		verificationResult.SupportsBrotli,
		string(langSupportJSON),
		verificationResult.CodeDebugging,
		verificationResult.CodeOptimization,
		verificationResult.TestGeneration,
		verificationResult.DocumentationGeneration,
		verificationResult.Refactoring,
		verificationResult.ErrorResolution,
		verificationResult.ArchitectureDesign,
		verificationResult.SecurityAssessment,
		verificationResult.PatternRecognition,
		verificationResult.DebuggingAccuracy,
		verificationResult.MaxHandledDepth,
		verificationResult.CodeQualityScore,
		verificationResult.LogicCorrectnessScore,
		verificationResult.RuntimeEfficiencyScore,
		verificationResult.OverallScore,
		verificationResult.CodeCapabilityScore,
		verificationResult.ResponsivenessScore,
		verificationResult.ReliabilityScore,
		verificationResult.FeatureRichnessScore,
		verificationResult.ValuePropositionScore,
		verificationResult.ScoreDetails,
		verificationResult.AvgLatencyMs,
		verificationResult.P95LatencyMs,
		verificationResult.MinLatencyMs,
		verificationResult.MaxLatencyMs,
		verificationResult.ThroughputRPS,
		verificationResult.RawRequest,
		verificationResult.RawResponse,
		verificationResult.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create verification result: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	verificationResult.ID = id
	return nil
}

// GetVerificationResult retrieves a verification result by ID
func (d *Database) GetVerificationResult(id int64) (*VerificationResult, error) {
	query := `
		SELECT id, model_id, verification_type, started_at, completed_at, status, error_message,
			model_exists, responsive, overloaded, latency_ms, supports_tool_use,
			supports_function_calling, supports_code_generation, supports_code_completion,
			supports_code_review, supports_code_explanation, supports_embeddings,
			supports_reranking, supports_image_generation, supports_audio_generation,
			supports_video_generation, supports_mcps, supports_lsps, supports_acps, supports_multimodal,
			supports_streaming, supports_json_mode, supports_structured_output,
			supports_reasoning, supports_parallel_tool_use, max_parallel_calls,
			supports_batch_processing, supports_brotli, code_language_support, code_debugging,
			code_optimization, test_generation, documentation_generation, refactoring,
			error_resolution, architecture_design, security_assessment,
			pattern_recognition, debugging_accuracy, max_handled_depth,
			code_quality_score, logic_correctness_score, runtime_efficiency_score,
			overall_score, code_capability_score, responsiveness_score,
			reliability_score, feature_richness_score, value_proposition_score,
			score_details, avg_latency_ms, p95_latency_ms, min_latency_ms,
			max_latency_ms, throughput_rps, raw_request, raw_response, created_at
		FROM verification_results WHERE id = ?
	`

	var result VerificationResult
	var langSupportJSON sql.NullString
	var completedAt, errorMessage, exists, responsive, overloaded, latencyMs, rawRequest, rawResponse sql.NullString

	err := d.conn.QueryRow(query, id).Scan(
		&result.ID,
		&result.ModelID,
		&result.VerificationType,
		&result.StartedAt,
		&completedAt,
		&result.Status,
		&errorMessage,
		&exists,
		&responsive,
		&overloaded,
		&latencyMs,
		&result.SupportsToolUse,
		&result.SupportsFunctionCalling,
		&result.SupportsCodeGeneration,
		&result.SupportsCodeCompletion,
		&result.SupportsCodeReview,
		&result.SupportsCodeExplanation,
		&result.SupportsEmbeddings,
		&result.SupportsReranking,
		&result.SupportsImageGeneration,
		&result.SupportsAudioGeneration,
		&result.SupportsVideoGeneration,
		&result.SupportsMCPs,
		&result.SupportsLSPs,
		&result.SupportsACPs,
		&result.SupportsMultimodal,
		&result.SupportsStreaming,
		&result.SupportsJSONMode,
		&result.SupportsStructuredOutput,
		&result.SupportsReasoning,
		&result.SupportsParallelToolUse,
		&result.MaxParallelCalls,
		&result.SupportsBatchProcessing,
		&result.SupportsBrotli,
		&langSupportJSON,
		&result.CodeDebugging,
		&result.CodeOptimization,
		&result.TestGeneration,
		&result.DocumentationGeneration,
		&result.Refactoring,
		&result.ErrorResolution,
		&result.ArchitectureDesign,
		&result.SecurityAssessment,
		&result.PatternRecognition,
		&result.DebuggingAccuracy,
		&result.MaxHandledDepth,
		&result.CodeQualityScore,
		&result.LogicCorrectnessScore,
		&result.RuntimeEfficiencyScore,
		&result.OverallScore,
		&result.CodeCapabilityScore,
		&result.ResponsivenessScore,
		&result.ReliabilityScore,
		&result.FeatureRichnessScore,
		&result.ValuePropositionScore,
		&result.ScoreDetails,
		&result.AvgLatencyMs,
		&result.P95LatencyMs,
		&result.MinLatencyMs,
		&result.MaxLatencyMs,
		&result.ThroughputRPS,
		&rawRequest,
		&rawResponse,
		&result.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("verification result not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get verification result: %w", err)
	}

	result.CompletedAt = scanNullableTimeFromString(completedAt)
	result.ErrorMessage = scanNullableString(errorMessage)
	result.ModelExists = scanNullableBoolFromString(exists)
	result.Responsive = scanNullableBoolFromString(responsive)
	result.Overloaded = scanNullableBoolFromString(overloaded)
	result.LatencyMs = scanNullableIntFromString(latencyMs)
	result.CodeLanguageSupport = scanJSONString(langSupportJSON)
	result.RawRequest = scanNullableString(rawRequest)
	result.RawResponse = scanNullableString(rawResponse)

	return &result, nil
}

// ListVerificationResults retrieves verification results with filtering
func (d *Database) ListVerificationResults(filters map[string]interface{}) ([]*VerificationResult, error) {
	query := `
		SELECT id, model_id, verification_type, started_at, completed_at, status, error_message,
			model_exists, responsive, overloaded, latency_ms, supports_tool_use,
			supports_function_calling, supports_code_generation, supports_code_completion,
			supports_code_review, supports_code_explanation, supports_embeddings,
			supports_reranking, supports_image_generation, supports_audio_generation,
			supports_video_generation, supports_mcps, supports_lsps, supports_acps, supports_multimodal,
			supports_streaming, supports_json_mode, supports_structured_output,
			supports_reasoning, supports_parallel_tool_use, max_parallel_calls,
			supports_batch_processing, supports_brotli, code_language_support, code_debugging,
			code_optimization, test_generation, documentation_generation, refactoring,
			error_resolution, architecture_design, security_assessment,
			pattern_recognition, debugging_accuracy, max_handled_depth,
			code_quality_score, logic_correctness_score, runtime_efficiency_score,
			overall_score, code_capability_score, responsiveness_score,
			reliability_score, feature_richness_score, value_proposition_score,
			score_details, avg_latency_ms, p95_latency_ms, min_latency_ms,
			max_latency_ms, throughput_rps, raw_request, raw_response, created_at
		FROM verification_results
	`

	var conditions []string
	var args []interface{}

	if modelID, ok := filters["model_id"]; ok {
		conditions = append(conditions, "model_id = ?")
		args = append(args, modelID)
	}

	if status, ok := filters["status"]; ok {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	if fromDate, ok := filters["from_date"]; ok {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, fromDate)
	}

	if toDate, ok := filters["to_date"]; ok {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, toDate)
	}

	if minScore, ok := filters["min_score"]; ok {
		conditions = append(conditions, "overall_score >= ?")
		args = append(args, minScore)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"]; ok {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list verification results: %w", err)
	}
	defer rows.Close()

	var results []*VerificationResult
	for rows.Next() {
		var result VerificationResult
		var langSupportJSON sql.NullString
		var completedAt, errorMessage, exists, responsive, overloaded, latencyMs, rawRequest, rawResponse sql.NullString

		err := rows.Scan(
			&result.ID,
			&result.ModelID,
			&result.VerificationType,
			&result.StartedAt,
			&completedAt,
			&result.Status,
			&errorMessage,
			&exists,
			&responsive,
			&overloaded,
			&latencyMs,
			&result.SupportsToolUse,
			&result.SupportsFunctionCalling,
			&result.SupportsCodeGeneration,
			&result.SupportsCodeCompletion,
			&result.SupportsCodeReview,
			&result.SupportsCodeExplanation,
			&result.SupportsEmbeddings,
			&result.SupportsReranking,
			&result.SupportsImageGeneration,
			&result.SupportsAudioGeneration,
			&result.SupportsVideoGeneration,
			&result.SupportsMCPs,
			&result.SupportsLSPs,
			&result.SupportsACPs,
			&result.SupportsMultimodal,
			&result.SupportsStreaming,
			&result.SupportsJSONMode,
			&result.SupportsStructuredOutput,
			&result.SupportsReasoning,
			&result.SupportsParallelToolUse,
			&result.MaxParallelCalls,
			&result.SupportsBatchProcessing,
			&result.SupportsBrotli,
			&langSupportJSON,
			&result.CodeDebugging,
			&result.CodeOptimization,
			&result.TestGeneration,
			&result.DocumentationGeneration,
			&result.Refactoring,
			&result.ErrorResolution,
			&result.ArchitectureDesign,
			&result.SecurityAssessment,
			&result.PatternRecognition,
			&result.DebuggingAccuracy,
			&result.MaxHandledDepth,
			&result.CodeQualityScore,
			&result.LogicCorrectnessScore,
			&result.RuntimeEfficiencyScore,
			&result.OverallScore,
			&result.CodeCapabilityScore,
			&result.ResponsivenessScore,
			&result.ReliabilityScore,
			&result.FeatureRichnessScore,
			&result.ValuePropositionScore,
			&result.ScoreDetails,
			&result.AvgLatencyMs,
			&result.P95LatencyMs,
			&result.MinLatencyMs,
			&result.MaxLatencyMs,
			&result.ThroughputRPS,
			&rawRequest,
			&rawResponse,
			&result.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan verification result: %w", err)
		}

		result.CompletedAt = scanNullableTimeFromString(completedAt)
		result.ErrorMessage = scanNullableString(errorMessage)
		result.ModelExists = scanNullableBoolFromString(exists)
		result.Responsive = scanNullableBoolFromString(responsive)
		result.Overloaded = scanNullableBoolFromString(overloaded)
		result.LatencyMs = scanNullableIntFromString(latencyMs)
		result.CodeLanguageSupport = scanJSONString(langSupportJSON)
		result.RawRequest = scanNullableString(rawRequest)
		result.RawResponse = scanNullableString(rawResponse)

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating verification results: %w", err)
	}

	return results, nil
}

// GetLatestVerificationResults gets the latest verification results for models
func (d *Database) GetLatestVerificationResults(modelIDs []int64) ([]*VerificationResult, error) {
	if len(modelIDs) == 0 {
		return []*VerificationResult{}, nil
	}

	// Create placeholders for the IN clause
	placeholders := make([]string, len(modelIDs))
	args := make([]interface{}, len(modelIDs))
	for i, id := range modelIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT vr.* FROM verification_results vr
		INNER JOIN (
			SELECT model_id, MAX(id) as max_id
			FROM verification_results
			WHERE model_id IN (%s) AND status = 'completed'
			GROUP BY model_id
		) latest ON vr.model_id = latest.model_id AND vr.id = latest.max_id
	`, strings.Join(placeholders, ",")) // #nosec G201 - placeholders are only "?" strings generated by code

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest verification results: %w", err)
	}
	defer rows.Close()

	var results []*VerificationResult
	for rows.Next() {
		var result VerificationResult
		var langSupportJSON sql.NullString
		var completedAt, errorMessage, exists, responsive, overloaded, latencyMs, rawRequest, rawResponse sql.NullString

		err := rows.Scan(
			&result.ID,
			&result.ModelID,
			&result.VerificationType,
			&result.StartedAt,
			&completedAt,
			&result.Status,
			&errorMessage,
			&exists,
			&responsive,
			&overloaded,
			&latencyMs,
			&result.SupportsToolUse,
			&result.SupportsFunctionCalling,
			&result.SupportsCodeGeneration,
			&result.SupportsCodeCompletion,
			&result.SupportsCodeReview,
			&result.SupportsCodeExplanation,
			&result.SupportsEmbeddings,
			&result.SupportsReranking,
			&result.SupportsImageGeneration,
			&result.SupportsAudioGeneration,
			&result.SupportsVideoGeneration,
			&result.SupportsMCPs,
			&result.SupportsLSPs,
			&result.SupportsACPs,
			&result.SupportsMultimodal,
			&result.SupportsStreaming,
			&result.SupportsJSONMode,
			&result.SupportsStructuredOutput,
			&result.SupportsReasoning,
			&result.SupportsParallelToolUse,
			&result.MaxParallelCalls,
			&result.SupportsBatchProcessing,
			&result.SupportsBrotli,
			&langSupportJSON,
			&result.CodeDebugging,
			&result.CodeOptimization,
			&result.TestGeneration,
			&result.DocumentationGeneration,
			&result.Refactoring,
			&result.ErrorResolution,
			&result.ArchitectureDesign,
			&result.SecurityAssessment,
			&result.PatternRecognition,
			&result.DebuggingAccuracy,
			&result.MaxHandledDepth,
			&result.CodeQualityScore,
			&result.LogicCorrectnessScore,
			&result.RuntimeEfficiencyScore,
			&result.OverallScore,
			&result.CodeCapabilityScore,
			&result.ResponsivenessScore,
			&result.ReliabilityScore,
			&result.FeatureRichnessScore,
			&result.ValuePropositionScore,
			&result.ScoreDetails,
			&result.AvgLatencyMs,
			&result.P95LatencyMs,
			&result.MinLatencyMs,
			&result.MaxLatencyMs,
			&result.ThroughputRPS,
			&rawRequest,
			&rawResponse,
			&result.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan latest verification result: %w", err)
		}

		result.CompletedAt = scanNullableTimeFromString(completedAt)
		result.ErrorMessage = scanNullableString(errorMessage)
		result.ModelExists = scanNullableBoolFromString(exists)
		result.Responsive = scanNullableBoolFromString(responsive)
		result.Overloaded = scanNullableBoolFromString(overloaded)
		result.LatencyMs = scanNullableIntFromString(latencyMs)
		result.CodeLanguageSupport = scanJSONString(langSupportJSON)
		result.RawRequest = scanNullableString(rawRequest)
		result.RawResponse = scanNullableString(rawResponse)

		results = append(results, &result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating latest verification results: %w", err)
	}

	return results, nil
}

// UpdateVerificationResult updates an existing verification result
func (d *Database) UpdateVerificationResult(verificationResult *VerificationResult) error {
	query := `
		UPDATE verification_results SET
			model_id = ?,
			verification_type = ?,
			started_at = ?,
			completed_at = ?,
			status = ?,
			error_message = ?,
			model_exists = ?,
			responsive = ?,
			overloaded = ?,
			latency_ms = ?,
			supports_tool_use = ?,
			supports_function_calling = ?,
			supports_code_generation = ?,
			supports_code_completion = ?,
			supports_code_review = ?,
			supports_code_explanation = ?,
			supports_embeddings = ?,
			supports_reranking = ?,
			supports_image_generation = ?,
			supports_audio_generation = ?,
			supports_video_generation = ?,
			supports_mcps = ?,
			supports_lsps = ?,
			supports_acps = ?,
			supports_multimodal = ?,
			supports_streaming = ?,
			supports_json_mode = ?,
			supports_structured_output = ?,
			supports_reasoning = ?,
			supports_parallel_tool_use = ?,
			max_parallel_calls = ?,
		supports_batch_processing = ?,
		supports_brotli = ?,
		code_language_support = ?,
		code_debugging = ?,
			code_optimization = ?,
			test_generation = ?,
			documentation_generation = ?,
			refactoring = ?,
			error_resolution = ?,
			architecture_design = ?,
			security_assessment = ?,
			pattern_recognition = ?,
			debugging_accuracy = ?,
			max_handled_depth = ?,
			code_quality_score = ?,
			logic_correctness_score = ?,
			runtime_efficiency_score = ?,
			overall_score = ?,
			code_capability_score = ?,
			responsiveness_score = ?,
			reliability_score = ?,
			feature_richness_score = ?,
			value_proposition_score = ?,
			raw_request = ?,
			raw_response = ?
		WHERE id = ?
	`

	// Convert nullable fields to SQL values
	completedAt := toNullTime(verificationResult.CompletedAt)
	errorMessage := toNullString(verificationResult.ErrorMessage)
	exists := toNullBool(verificationResult.ModelExists)
	responsive := toNullBool(verificationResult.Responsive)
	overloaded := toNullBool(verificationResult.Overloaded)
	latencyMs := toNullInt(verificationResult.LatencyMs)
	rawRequest := toNullString(verificationResult.RawRequest)
	rawResponse := toNullString(verificationResult.RawResponse)

	// Convert code language support to JSON
	var langSupportJSON sql.NullString
	if verificationResult.CodeLanguageSupport != nil {
		jsonData, err := json.Marshal(verificationResult.CodeLanguageSupport)
		if err != nil {
			return fmt.Errorf("failed to marshal code language support: %w", err)
		}
		langSupportJSON = sql.NullString{String: string(jsonData), Valid: true}
	}

	_, err := d.conn.Exec(query,
		verificationResult.ModelID,
		verificationResult.VerificationType,
		verificationResult.StartedAt,
		completedAt,
		verificationResult.Status,
		errorMessage,
		exists,
		responsive,
		overloaded,
		latencyMs,
		verificationResult.SupportsToolUse,
		verificationResult.SupportsFunctionCalling,
		verificationResult.SupportsCodeGeneration,
		verificationResult.SupportsCodeCompletion,
		verificationResult.SupportsCodeReview,
		verificationResult.SupportsCodeExplanation,
		verificationResult.SupportsEmbeddings,
		verificationResult.SupportsReranking,
		verificationResult.SupportsImageGeneration,
		verificationResult.SupportsAudioGeneration,
		verificationResult.SupportsVideoGeneration,
		verificationResult.SupportsMCPs,
		verificationResult.SupportsLSPs,
		verificationResult.SupportsACPs,
		verificationResult.SupportsMultimodal,
		verificationResult.SupportsStreaming,
		verificationResult.SupportsJSONMode,
		verificationResult.SupportsStructuredOutput,
		verificationResult.SupportsReasoning,
		verificationResult.SupportsParallelToolUse,
		verificationResult.MaxParallelCalls,
		verificationResult.SupportsBatchProcessing,
		verificationResult.SupportsBrotli,
		langSupportJSON,
		verificationResult.CodeDebugging,
		verificationResult.CodeOptimization,
		verificationResult.TestGeneration,
		verificationResult.DocumentationGeneration,
		verificationResult.Refactoring,
		verificationResult.ErrorResolution,
		verificationResult.ArchitectureDesign,
		verificationResult.SecurityAssessment,
		verificationResult.PatternRecognition,
		verificationResult.DebuggingAccuracy,
		verificationResult.MaxHandledDepth,
		verificationResult.CodeQualityScore,
		verificationResult.LogicCorrectnessScore,
		verificationResult.RuntimeEfficiencyScore,
		verificationResult.OverallScore,
		verificationResult.CodeCapabilityScore,
		verificationResult.ResponsivenessScore,
		verificationResult.ReliabilityScore,
		verificationResult.FeatureRichnessScore,
		verificationResult.ValuePropositionScore,
		rawRequest,
		rawResponse,
		verificationResult.ID,
	)

	if err != nil {
		log.Printf("UpdateVerificationResult error: %v", err)
		return fmt.Errorf("failed to update verification result: %w", err)
	}

	return nil
}

// DeleteVerificationResult deletes a verification result by ID
func (d *Database) DeleteVerificationResult(id int64) error {
	query := `DELETE FROM verification_results WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete verification result: %w", err)
	}

	return nil
}

// GetModelCount returns the total number of models
func (d *Database) GetModelCount() (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM models`

	err := d.conn.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get model count: %w", err)
	}

	return count, nil
}

// GetProviderCount returns the total number of providers
func (d *Database) GetProviderCount() (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM providers`

	err := d.conn.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get provider count: %w", err)
	}

	return count, nil
}

// GetVerificationResultCount returns the total number of verification results
func (d *Database) GetVerificationResultCount() (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM verification_results`

	err := d.conn.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get verification result count: %w", err)
	}

	return count, nil
}

// ==================== Notification CRUD Operations ====================

// CreateNotification creates a new notification record
func (d *Database) CreateNotification(notification *Notification) error {
	dataJSON, err := json.Marshal(notification.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	query := `
		INSERT INTO notifications (
			type, channel, priority, title, message, data, recipient,
			sent, error, retry_count, sent_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		notification.Type,
		notification.Channel,
		notification.Priority,
		notification.Title,
		notification.Message,
		string(dataJSON),
		notification.Recipient,
		notification.Sent,
		notification.Error,
		notification.RetryCount,
		notification.SentAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	notification.ID = id
	return nil
}

// GetNotifications retrieves notifications with optional filtering
func (d *Database) GetNotifications(limit int, offset int, filters map[string]interface{}) ([]*Notification, error) {
	whereClause := ""
	args := make([]interface{}, 0)

	// Build WHERE clause based on filters
	if len(filters) > 0 {
		conditions := make([]string, 0, len(filters))
		for key, value := range filters {
			// Validate column name to prevent SQL injection
			if err := ValidateColumnName(key); err != nil {
				return nil, fmt.Errorf("invalid filter column name '%s': %w", key, err)
			}
			conditions = append(conditions, fmt.Sprintf("%s = ?", key)) // #nosec G201
			args = append(args, value)
		}
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(` // #nosec G201
		SELECT id, type, channel, priority, title, message, data, recipient,
		       sent, error, retry_count, created_at, sent_at, updated_at
		FROM notifications
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		var notification Notification
		var dataJSON string
		var sentAt *time.Time

		err := rows.Scan(
			&notification.ID,
			&notification.Type,
			&notification.Channel,
			&notification.Priority,
			&notification.Title,
			&notification.Message,
			&dataJSON,
			&notification.Recipient,
			&notification.Sent,
			&notification.Error,
			&notification.RetryCount,
			&notification.CreatedAt,
			&sentAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}

		// Parse JSON data
		if err := json.Unmarshal([]byte(dataJSON), &notification.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal notification data: %w", err)
		}

		notification.SentAt = sentAt
		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

// UpdateNotification updates a notification record
func (d *Database) UpdateNotification(notification *Notification) error {
	dataJSON, err := json.Marshal(notification.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal notification data: %w", err)
	}

	query := `
		UPDATE notifications SET
			type = ?, channel = ?, priority = ?, title = ?, message = ?,
			data = ?, recipient = ?, sent = ?, error = ?, retry_count = ?,
			sent_at = ?
		WHERE id = ?
	`

	_, err = d.conn.Exec(query,
		notification.Type,
		notification.Channel,
		notification.Priority,
		notification.Title,
		notification.Message,
		string(dataJSON),
		notification.Recipient,
		notification.Sent,
		notification.Error,
		notification.RetryCount,
		notification.SentAt,
		notification.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	return nil
}

// DeleteNotification deletes a notification by ID
func (d *Database) DeleteNotification(id int64) error {
	query := `DELETE FROM notifications WHERE id = ?`

	result, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// GetNotificationStats returns notification statistics
func (d *Database) GetNotificationStats() (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_notifications,
			SUM(CASE WHEN sent = 1 THEN 1 ELSE 0 END) as sent_notifications,
			SUM(CASE WHEN sent = 0 THEN 1 ELSE 0 END) as failed_notifications,
			AVG(retry_count) as avg_retry_count,
			COUNT(DISTINCT type) as unique_types,
			COUNT(DISTINCT channel) as unique_channels
		FROM notifications
		WHERE created_at >= datetime('now', '-30 days')
	`

	var total, sent, failed int
	var avgRetry float64
	var uniqueTypes, uniqueChannels int

	err := d.conn.QueryRow(query).Scan(&total, &sent, &failed, &avgRetry, &uniqueTypes, &uniqueChannels)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification stats: %w", err)
	}

	return map[string]interface{}{
		"total_notifications":  total,
		"sent_notifications":   sent,
		"failed_notifications": failed,
		"success_rate":         float64(sent) / float64(total) * 100,
		"avg_retry_count":      avgRetry,
		"unique_types":         uniqueTypes,
		"unique_channels":      uniqueChannels,
	}, nil
}

// Helper function to scan nullable int from string
func scanNullableIntFromString(nullString sql.NullString) *int {
	if !nullString.Valid || nullString.String == "" {
		return nil
	}

	var val int
	if _, err := fmt.Sscanf(nullString.String, "%d", &val); err != nil {
		return nil
	}

	return &val
}
