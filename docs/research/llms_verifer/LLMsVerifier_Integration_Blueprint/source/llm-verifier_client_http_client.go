package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"digital.vasic.llmsverifier/monitoring"
)

// HTTPClient represents an HTTP client for making LLM API requests
type HTTPClient struct {
	client           *http.Client
	brotliCache      map[string]BrotliCacheEntry
	brotliCacheMutex sync.RWMutex
	metricsTracker   *monitoring.MetricsTracker
}

type BrotliCacheEntry struct {
	Value      bool
	Expiration time.Time
}

// MetricsTrackerInterface defines the interface for tracking metrics
type MetricsTrackerInterface interface {
	RecordBrotliTest(supportsBrotli bool, duration time.Duration)
	RecordBrotliCacheHit()
	RecordBrotliCacheMiss()
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		brotliCache:    make(map[string]BrotliCacheEntry),
		metricsTracker: nil, // Default to nil - can be set later
	}
}

// SetMetricsTracker sets the metrics tracker for the HTTP client
func (c *HTTPClient) SetMetricsTracker(tracker *monitoring.MetricsTracker) {
	c.metricsTracker = tracker
}

// TestModelExists checks if a model is available on provider's API
func (c *HTTPClient) TestModelExists(ctx context.Context, provider, apiKey, modelID string) (bool, error) {
	endpoint := getProviderEndpoint(provider)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err == nil {
		// Check if model exists in response
		if models, ok := data["data"].([]interface{}); ok {
			for _, m := range models {
				if model, ok := m.(map[string]interface{}); ok {
					if id, ok := model["id"].(string); ok && id == modelID {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// TestResponsiveness measures how quickly a model responds to a test prompt
func (c *HTTPClient) TestResponsiveness(ctx context.Context, provider, apiKey, modelID, prompt string) (time.Duration, time.Duration, error, string, bool, int, error) {
	endpoint := getModelEndpoint(provider, modelID)

	requestBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens": 10,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, nil
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.client.Do(req)
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, err
	}
	defer resp.Body.Close()

	totalTime := time.Since(start)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return time.Duration(0), time.Duration(0), err, "", false, 0, nil
	}

	// Parse response for TTFT (time to first token)
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err == nil {
		// Estimate TTFT as 20% of total time for non-streaming
		ttft := time.Duration(float64(totalTime) * 0.2)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return totalTime, ttft, nil, "", true, resp.StatusCode, nil
		}

		return totalTime, ttft, nil, fmt.Sprintf("HTTP %d", resp.StatusCode), false, resp.StatusCode, nil
	}

	return totalTime, time.Duration(0), nil, "Invalid response format", false, resp.StatusCode, nil
}

// TestStreaming tests if a model supports streaming responses
func (c *HTTPClient) TestStreaming(ctx context.Context, provider, apiKey, modelID, prompt string) (bool, error) {
	endpoint := getModelEndpoint(provider, modelID)

	requestBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"stream":     true,
		"max_tokens": 50,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Count chunks in streaming response
	chunkCount := 0
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "data: ") {
			chunkCount++
		}
		if line == "[DONE]" {
			break
		}
	}

	return chunkCount > 0, nil
}

// getProviderEndpoint returns the models list endpoint for a provider
func getProviderEndpoint(provider string) string {
	providerEndpoints := map[string]string{
		// Core providers
		"openai":      "https://api.openai.com/v1/models",
		"anthropic":   "https://api.anthropic.com/v1/models",
		"google":      "https://generativelanguage.googleapis.com/v1/models",
		"gemini":      "https://generativelanguage.googleapis.com/v1beta/models",
		
		// OpenAI-compatible providers
		"openrouter":  "https://openrouter.ai/api/v1/models",
		"deepseek":    "https://api.deepseek.com/v1/models",
		"mistral":     "https://api.mistral.ai/v1/models",
		"mistralaistudio": "https://api.mistral.ai/v1/models",
		"groq":        "https://api.groq.com/openai/v1/models",
		"togetherai":  "https://api.together.xyz/v1/models",
		"fireworksai": "https://api.fireworks.ai/v1/models",
		"fireworks":   "https://api.fireworks.ai/v1/models",
		"chutes":      "https://llm.chutes.ai/v1/models",
		"siliconflow": "https://api.siliconflow.cn/v1/models",
		"kimi":        "https://api.moonshot.cn/v1/models",
		"zai":         "https://open.bigmodel.cn/api/paas/v4/models",
		"nebius":      "https://api.studio.nebius.ai/v1/models",
		"hyperbolic":  "https://api.hyperbolic.xyz/v1/models",
		"baseten":     "https://inference.baseten.co/v1/models",
		"novita":      "https://api.novita.ai/v1/models",
		"upstage":     "https://api.upstage.ai/v1/models",
		"inference":   "https://api.inference.net/v1/models",
		"cerebras":    "https://api.cerebras.ai/v1/models",
		"modal":       "https://api.modal.com/v1/models",
		"sambanova":   "https://api.sambanova.ai/v1/models",
		
		// Special API providers
		"huggingface": "https://api-inference.huggingface.co/models",
		"cohere":      "https://api.cohere.com/v2/models",
		"replicate":   "https://api.replicate.com/v1/models",
		"nlpcloud":    "https://api.nlpcloud.com/v1/models",
		"poe":         "https://api.poe.com/v1/models",
		"navigator":   "https://api.ai.it.ufl.edu/v1/models",
		"codestral":   "https://api.mistral.ai/v1/models",
		"nvidia":      "https://integrate.api.nvidia.com/v1/models",
		
		// Cloud providers
		"cloudflare":  "https://api.cloudflare.com/client/v4/accounts/{{account_id}}/ai/models",
		
		// Vercel AI Gateway
		"vercelai":    "https://api.vercel.com/v1/ai/models",
		"vercel":      "https://api.vercel.com/v1/ai/models",
		"vercelaigateway": "https://api.vercel.com/v1/ai/models",
	}

	if endpoint, ok := providerEndpoints[strings.ToLower(provider)]; ok {
		return endpoint
	}
	
	// Return empty string for unknown providers
	// This allows the caller to handle the error gracefully
	return ""
}



// getModelEndpoint returns the chat/completion endpoint for a provider
func getModelEndpoint(provider, modelID string) string {
	providerEndpoints := map[string]string{
		// Core providers
		"openai":      "https://api.openai.com/v1/chat/completions",
		"anthropic":   "https://api.anthropic.com/v1/messages",
		"google":      "https://generativelanguage.googleapis.com/v1beta/models/" + modelID + ":generateContent",
		"gemini":      "https://generativelanguage.googleapis.com/v1beta/models/" + modelID + ":generateContent",
		
		// OpenAI-compatible providers
		"openrouter":  "https://openrouter.ai/api/v1/chat/completions",
		"deepseek":    "https://api.deepseek.com/v1/chat/completions",
		"mistral":     "https://api.mistral.ai/v1/chat/completions",
		"mistralaistudio": "https://api.mistral.ai/v1/chat/completions",
		"groq":        "https://api.groq.com/openai/v1/chat/completions",
		"togetherai":  "https://api.together.xyz/v1/chat/completions",
		"fireworksai": "https://api.fireworks.ai/v1/chat/completions",
		"fireworks":   "https://api.fireworks.ai/v1/chat/completions",
		"chutes":      "https://llm.chutes.ai/v1/chat/completions",
		"siliconflow": "https://api.siliconflow.cn/v1/chat/completions",
		"kimi":        "https://api.moonshot.cn/v1/chat/completions",
		"zai":         "https://open.bigmodel.cn/api/paas/v4/chat/completions",
		"nebius":      "https://api.studio.nebius.ai/v1/chat/completions",
		"hyperbolic":  "https://api.hyperbolic.xyz/v1/chat/completions",
		"baseten":     "https://inference.baseten.co/v1/chat/completions",
		"novita":      "https://api.novita.ai/v1/chat/completions",
		"upstage":     "https://api.upstage.ai/v1/chat/completions",
		"inference":   "https://api.inference.net/v1/chat/completions",
		"cerebras":    "https://api.cerebras.ai/v1/chat/completions",
		"modal":       "https://api.modal.com/v1/chat/completions",
		"sambanova":   "https://api.sambanova.ai/v1/chat/completions",
		
		// Special API providers
		"huggingface": "https://api-inference.huggingface.co/models/" + modelID,
		"cohere":      "https://api.cohere.com/v2/chat",
		"replicate":   "https://api.replicate.com/v1/predictions",
		"nlpcloud":    "https://api.nlpcloud.com/v1/gpu",
		"poe":         "https://api.poe.com/v1/chat/completions",
		"navigator":   "https://api.ai.it.ufl.edu/v1/chat/completions",
		"codestral":   "https://api.mistral.ai/v1/fim/completions",
		"nvidia":      "https://integrate.api.nvidia.com/v1/chat/completions",
		
		// Cloud providers (special handling needed)
		"cloudflare":  "https://api.cloudflare.com/client/v4/accounts/{{account_id}}/ai/run/" + modelID,
		
		// Vercel AI Gateway
		"vercelai":    "https://api.vercel.com/v1/ai/chat/completions",
		"vercel":      "https://api.vercel.com/v1/ai/chat/completions",
		"vercelaigateway": "https://api.vercel.com/v1/ai/chat/completions",
	}

	if endpoint, ok := providerEndpoints[strings.ToLower(provider)]; ok {
		return endpoint
	}
	
	// Return empty string for unknown providers
	return ""
}



// DetectErrorType categorizes HTTP errors
func DetectErrorType(statusCode int, body []byte) string {
	switch statusCode {
	case 401:
		return "authentication_error"
	case 429:
		return "rate_limit_exceeded"
	case 404:
		return "model_not_found"
	case 500:
		return "server_error"
	default:
		if statusCode >= 400 && statusCode < 500 {
			return "client_error"
		}
		return "unknown_error"
	}
}

// TestBrotliSupport tests if a model supports Brotli compression
func (c *HTTPClient) TestBrotliSupport(ctx context.Context, provider, apiKey, modelID string) (bool, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("%s:%s:%s", provider, modelID, apiKey)

	// Check cache first
	c.brotliCacheMutex.RLock()
	if cachedEntry, exists := c.brotliCache[cacheKey]; exists {
		// Check if cache entry is still valid
		if time.Now().Before(cachedEntry.Expiration) {
			c.brotliCacheMutex.RUnlock()
			// Track cache hit
			if c.metricsTracker != nil {
				c.metricsTracker.RecordBrotliCacheHit()
			}
			return cachedEntry.Value, nil
		} else {
			// Entry expired, remove it
			c.brotliCacheMutex.RUnlock()
			c.brotliCacheMutex.Lock()
			delete(c.brotliCache, cacheKey)
			c.brotliCacheMutex.Unlock()
		}
	} else {
		c.brotliCacheMutex.RUnlock()
	}

	// Track cache miss
	if c.metricsTracker != nil {
		c.metricsTracker.RecordBrotliCacheMiss()
	}

	startTime := time.Now()
	endpoint := getModelEndpoint(provider, modelID)

	// Create a minimal request body
	requestBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "test",
			},
		},
		"max_tokens": 1,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(jsonBody)))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	// Request Brotli compression
	req.Header.Set("Accept-Encoding", "br")

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check if response indicates Brotli support
	contentEncoding := resp.Header.Get("Content-Encoding")

	// Brotli compression is indicated by "br" in Content-Encoding header
	supportsBrotli := strings.Contains(contentEncoding, "br")

	// Also check if server accepts Brotli requests
	var encodingAccepted bool
	if acceptEncoding := resp.Header.Get("Accept-Encoding"); acceptEncoding != "" {
		encodingAccepted = strings.Contains(acceptEncoding, "br")
	}

	// If either the response is compressed with Brotli or the server accepts Brotli requests
	result := supportsBrotli || encodingAccepted

	// Cache the result with 24-hour TTL
	c.brotliCacheMutex.Lock()
	c.brotliCache[cacheKey] = BrotliCacheEntry{
		Value:      result,
		Expiration: time.Now().Add(24 * time.Hour),
	}
	c.brotliCacheMutex.Unlock()

	// Track Brotli test result
	if c.metricsTracker != nil {
		duration := time.Since(startTime)
		c.metricsTracker.RecordBrotliTest(result, duration)
	}

	return result, nil
}

// ClearBrotliCache clears the Brotli support cache
func (c *HTTPClient) ClearBrotliCache() {
	c.brotliCacheMutex.Lock()
	c.brotliCache = make(map[string]BrotliCacheEntry)
	c.brotliCacheMutex.Unlock()
}
