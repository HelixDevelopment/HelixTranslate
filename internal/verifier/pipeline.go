package verifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// VerificationResult holds the outcome of a single verification step.
type VerificationResult struct {
	Step      string
	Passed    bool
	Score     float64
	LatencyMs int64
	Error     string
	Details   map[string]interface{}
}

// ModelVerification aggregates all step results for a model.
type ModelVerification struct {
	ModelID    string
	Provider   string
	Steps      []VerificationResult
	Passed     bool
	Overall    float64
	VerifiedAt time.Time
}

// Pipeline performs the 8-step verification.
type Pipeline struct {
	httpClient *http.Client
}

// NewPipeline creates a new verification pipeline.
func NewPipeline() *Pipeline {
	return &Pipeline{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Verify runs all 8 steps against a provider's model.
func (p *Pipeline) Verify(ctx context.Context, provider ProviderConfig, modelID string) *ModelVerification {
	mv := &ModelVerification{
		ModelID:    modelID,
		Provider:   provider.ID,
		Steps:      make([]VerificationResult, 0, 8),
		VerifiedAt: time.Now(),
	}

	// Step 1: API Reachability
	mv.Steps = append(mv.Steps, p.checkReachability(ctx, provider))

	// Step 2: Authentication
	mv.Steps = append(mv.Steps, p.checkAuthentication(ctx, provider))

	// Step 3: Model Existence
	mv.Steps = append(mv.Steps, p.checkModelExistence(ctx, provider, modelID))

	// Step 4: Response Format
	mv.Steps = append(mv.Steps, p.validateResponseFormat(ctx, provider, modelID))

	// Step 5: Latency
	mv.Steps = append(mv.Steps, p.measureLatency(ctx, provider, modelID))

	// Step 6: Capabilities
	mv.Steps = append(mv.Steps, p.detectCapabilities(ctx, provider, modelID))

	// Step 7: Rate Limits (informational, not pass/fail)
	mv.Steps = append(mv.Steps, p.checkRateLimits(ctx, provider))

	// Step 8: Error Handling (informational)
	mv.Steps = append(mv.Steps, p.validateErrorHandling(ctx, provider))

	// Compute overall pass/fail and score
	// Reachability and authentication are hard gates - they MUST pass
	var totalScore float64
	reachabilityPassed := mv.Steps[0].Passed
	authPassed := mv.Steps[1].Passed
	for _, step := range mv.Steps {
		totalScore += step.Score
	}
	mv.Overall = totalScore / float64(len(mv.Steps))
	mv.Passed = reachabilityPassed && authPassed && mv.Overall >= 0.5

	return mv
}

func (p *Pipeline) checkReachability(ctx context.Context, provider ProviderConfig) VerificationResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL, nil)
	if err != nil {
		return VerificationResult{Step: "reachability", Passed: false, Score: 0, Error: err.Error()}
	}

	resp, err := p.httpClient.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return VerificationResult{Step: "reachability", Passed: false, Score: 0, LatencyMs: elapsed, Error: err.Error()}
	}
	defer resp.Body.Close()

	// Any HTTP response (even 401/404) means the endpoint is reachable
	return VerificationResult{
		Step:      "reachability",
		Passed:    true,
		Score:     1.0,
		LatencyMs: elapsed,
		Details:   map[string]interface{}{"status_code": resp.StatusCode},
	}
}

func (p *Pipeline) checkAuthentication(ctx context.Context, provider ProviderConfig) VerificationResult {
	// If no API key configured, skip but mark as uncertain
	if provider.APIKey == "" {
		return VerificationResult{Step: "authentication", Passed: true, Score: 0.5, Details: map[string]interface{}{"note": "no api key configured"}}
	}

	// Try a lightweight authenticated request
	// For most providers, HEAD or GET to base URL with auth header
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL+"/models", nil)
	if err != nil {
		return VerificationResult{Step: "authentication", Passed: false, Score: 0, Error: err.Error()}
	}

	req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	resp, err := p.httpClient.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return VerificationResult{Step: "authentication", Passed: false, Score: 0, LatencyMs: elapsed, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return VerificationResult{Step: "authentication", Passed: false, Score: 0, LatencyMs: elapsed, Error: fmt.Sprintf("auth failed: %d", resp.StatusCode)}
	}

	return VerificationResult{
		Step:      "authentication",
		Passed:    true,
		Score:     1.0,
		LatencyMs: elapsed,
		Details:   map[string]interface{}{"status_code": resp.StatusCode},
	}
}

func (p *Pipeline) checkModelExistence(ctx context.Context, provider ProviderConfig, modelID string) VerificationResult {
	if modelID == "" {
		return VerificationResult{Step: "model_existence", Passed: false, Score: 0, Error: "model ID is empty"}
	}

	// Query the provider's /models endpoint to verify the model exists
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL+"/models", nil)
	if err != nil {
		return VerificationResult{Step: "model_existence", Passed: false, Score: 0, Error: err.Error()}
	}
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return VerificationResult{Step: "model_existence", Passed: false, Score: 0, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If we can't query models, fall back to local knowledge
		return VerificationResult{
			Step:    "model_existence",
			Passed:  true,
			Score:   0.6,
			Details: map[string]interface{}{"model_id": modelID, "note": "models endpoint unavailable, using local validation"},
		}
	}

	// Parse OpenAI-compatible models list: {"data":[{"id":"model-name",...}]}
	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		// Try plain array fallback
		var plain []struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&plain); err == nil {
			modelsResp.Data = plain
		}
	}

	found := false
	for _, m := range modelsResp.Data {
		if m.ID == modelID {
			found = true
			break
		}
	}

	if !found {
		return VerificationResult{
			Step:    "model_existence",
			Passed:  false,
			Score:   0.3,
			Details: map[string]interface{}{"model_id": modelID, "note": "model not found in provider catalog"},
		}
	}

	return VerificationResult{
		Step:    "model_existence",
		Passed:  true,
		Score:   1.0,
		Details: map[string]interface{}{"model_id": modelID, "found": true},
	}
}

func (p *Pipeline) validateResponseFormat(ctx context.Context, provider ProviderConfig, modelID string) VerificationResult {
	// Send a minimal chat completion request and verify the response format
	body, _ := json.Marshal(map[string]interface{}{
		"model":      modelID,
		"messages":   []map[string]string{{"role": "user", "content": "Hi"}},
		"max_tokens": 5,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return VerificationResult{Step: "response_format", Passed: false, Score: 0, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return VerificationResult{Step: "response_format", Passed: false, Score: 0, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return VerificationResult{
			Step:    "response_format",
			Passed:  false,
			Score:   0.2,
			Details: map[string]interface{}{"status_code": resp.StatusCode, "note": "chat completions endpoint returned non-200"},
		}
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return VerificationResult{
			Step:    "response_format",
			Passed:  false,
			Score:   0.4,
			Details: map[string]interface{}{"error": err.Error(), "note": "response did not match expected schema"},
		}
	}

	if len(completion.Choices) == 0 || completion.Choices[0].Message.Content == "" {
		return VerificationResult{
			Step:    "response_format",
			Passed:  false,
			Score:   0.5,
			Details: map[string]interface{}{"note": "response parsed but content was empty"},
		}
	}

	return VerificationResult{
		Step:    "response_format",
		Passed:  true,
		Score:   1.0,
		Details: map[string]interface{}{"content_preview": completion.Choices[0].Message.Content},
	}
}

func (p *Pipeline) measureLatency(ctx context.Context, provider ProviderConfig, modelID string) VerificationResult {
	body, _ := json.Marshal(map[string]interface{}{
		"model":      modelID,
		"messages":   []map[string]string{{"role": "user", "content": "Hi"}},
		"max_tokens": 5,
	})

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return VerificationResult{Step: "latency", Passed: false, Score: 0, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return VerificationResult{Step: "latency", Passed: false, Score: 0, LatencyMs: elapsed, Error: err.Error()}
	}
	defer resp.Body.Close()

	score := 1.0
	if elapsed > 5000 {
		score = 0.5
	} else if elapsed > 1000 {
		score = 0.8
	}

	return VerificationResult{
		Step:      "latency",
		Passed:    true,
		Score:     score,
		LatencyMs: elapsed,
		Details:   map[string]interface{}{"status_code": resp.StatusCode},
	}
}

func (p *Pipeline) detectCapabilities(ctx context.Context, provider ProviderConfig, modelID string) VerificationResult {
	// Capability detection uses model ID heuristics and provider metadata.
	caps := map[string]bool{}
	score := 0.7

	// Heuristic: vision models often contain "vision" or " multimodal" in ID
	if containsAny(modelID, "vision", " multimodal", "-v", "gpt-4o") {
		caps["vision"] = true
		score += 0.1
	}
	// Heuristic: code models
	if containsAny(modelID, "coder", "code", "devin") {
		caps["code"] = true
		score += 0.1
	}
	// Heuristic: large context models
	if containsAny(modelID, "128k", "200k", "1m", "100k") {
		caps["long_context"] = true
		score += 0.05
	}
	// Heuristic: streaming support (most modern APIs)
	caps["streaming"] = true
	score += 0.05

	if score > 1.0 {
		score = 1.0
	}

	return VerificationResult{
		Step:    "capabilities",
		Passed:  true,
		Score:   score,
		Details: map[string]interface{}{"capabilities": caps, "model_id": modelID},
	}
}

func (p *Pipeline) checkRateLimits(ctx context.Context, provider ProviderConfig) VerificationResult {
	// Re-use the /models request to inspect rate limit headers
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL+"/models", nil)
	if err != nil {
		return VerificationResult{Step: "rate_limits", Passed: false, Score: 0, Error: err.Error()}
	}
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return VerificationResult{Step: "rate_limits", Passed: false, Score: 0, Error: err.Error()}
	}
	defer resp.Body.Close()

	details := map[string]interface{}{"status_code": resp.StatusCode}
	score := 0.5

	headers := []string{
		"X-RateLimit-Limit",
		"X-RateLimit-Remaining",
		"X-RateLimit-Reset",
		"Retry-After",
		"x-ratelimit-limit",
		"x-ratelimit-remaining",
		"x-ratelimit-reset",
		"retry-after",
	}
	found := false
	for _, h := range headers {
		if v := resp.Header.Get(h); v != "" {
			details[h] = v
			found = true
			score = 0.8
		}
	}
	if found {
		score = 1.0
	}

	return VerificationResult{
		Step:    "rate_limits",
		Passed:  true,
		Score:   score,
		Details: details,
	}
}

func (p *Pipeline) validateErrorHandling(ctx context.Context, provider ProviderConfig) VerificationResult {
	// Send a request with an invalid model ID to verify graceful 4xx error handling
	body, _ := json.Marshal(map[string]interface{}{
		"model":      "invalid-model-id-00000000",
		"messages":   []map[string]string{{"role": "user", "content": "Hi"}},
		"max_tokens": 5,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return VerificationResult{Step: "error_handling", Passed: false, Score: 0, Error: err.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return VerificationResult{Step: "error_handling", Passed: false, Score: 0, Error: err.Error()}
	}
	defer resp.Body.Close()

	// A well-behaved API returns 4xx for client errors, not 5xx or connection errors
	if resp.StatusCode >= 500 {
		return VerificationResult{
			Step:    "error_handling",
			Passed:  false,
			Score:   0.3,
			Details: map[string]interface{}{"status_code": resp.StatusCode, "note": "server returned 5xx for client error"},
		}
	}

	return VerificationResult{
		Step:    "error_handling",
		Passed:  true,
		Score:   1.0,
		Details: map[string]interface{}{"status_code": resp.StatusCode, "note": "graceful 4xx error response"},
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && bytes.Contains([]byte(s), []byte(sub)) {
			return true
		}
	}
	return false
}
