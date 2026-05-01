package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ReplicateClient implements LLMClient for Replicate API.
type ReplicateClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// replicatePredictionRequest represents the request to create a prediction.
type replicatePredictionRequest struct {
	Input map[string]interface{} `json:"input"`
}

// replicatePredictionResponse represents the response from creating a prediction.
type replicatePredictionResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// replicateGetResponse represents the response from polling a prediction.
type replicateGetResponse struct {
	ID     string      `json:"id"`
	Status string      `json:"status"`
	Output interface{} `json:"output"`
	Error  string      `json:"error,omitempty"`
}

// NewReplicateClient creates a new Replicate client.
func NewReplicateClient(config TranslationConfig) (*ReplicateClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("replicate API key is required")
	}

	if config.Model == "" {
		return nil, fmt.Errorf("replicate model is required")
	}

	validModels := ValidModels[ProviderReplicate]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Replicate. Valid models: %v",
			config.Model, validModels)
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.replicate.com/v1"
	}

	return &ReplicateClient{
		apiKey:     config.APIKey,
		model:      config.Model,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// GetProviderName returns the provider name.
func (c *ReplicateClient) GetProviderName() string {
	return "replicate"
}

// Translate performs translation via Replicate async API.
func (c *ReplicateClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	// Create prediction
	predictionID, err := c.createPrediction(ctx, prompt)
	if err != nil {
		return "", err
	}

	// Poll for result
	result, err := c.pollPrediction(ctx, predictionID)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (c *ReplicateClient) createPrediction(ctx context.Context, prompt string) (string, error) {
	reqBody := replicatePredictionRequest{
		Input: map[string]interface{}{
			"prompt":       prompt,
			"max_tokens":   8192,
			"temperature":  0.3,
			"top_p":        0.9,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s/predictions", c.baseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+c.apiKey)
	req.Header.Set("Prefer", "wait") // Try synchronous if supported

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("replicate API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response replicatePredictionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != "" {
		return "", fmt.Errorf("replicate prediction error: %s", response.Error)
	}

	if response.ID == "" {
		return "", fmt.Errorf("no prediction ID in response")
	}

	return response.ID, nil
}

func (c *ReplicateClient) pollPrediction(ctx context.Context, predictionID string) (string, error) {
	url := fmt.Sprintf("%s/predictions/%s", c.baseURL, predictionID)

	maxAttempts := 60
	pollInterval := 2 * time.Second

	for i := 0; i < maxAttempts; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create poll request: %w", err)
		}

		req.Header.Set("Authorization", "Token "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to send poll request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", fmt.Errorf("failed to read poll response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("replicate poll error (status %d): %s", resp.StatusCode, string(body))
		}

		var response replicateGetResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return "", fmt.Errorf("failed to unmarshal poll response: %w", err)
		}

		switch response.Status {
		case "succeeded":
			return c.extractOutput(response.Output)
		case "failed", "canceled":
			return "", fmt.Errorf("replicate prediction %s: %s", response.Status, response.Error)
		case "starting", "processing":
			// Continue polling
		}

		select {
		case <-ctx.Done():
			return "", fmt.Errorf("context cancelled while polling: %w", ctx.Err())
		case <-time.After(pollInterval):
			// Continue to next attempt
		}
	}

	return "", fmt.Errorf("replicate prediction timed out after %d attempts", maxAttempts)
}

func (c *ReplicateClient) extractOutput(output interface{}) (string, error) {
	if output == nil {
		return "", fmt.Errorf("empty output from replicate")
	}

	switch v := output.(type) {
	case string:
		return v, nil
	case []interface{}:
		var parts []string
		for _, part := range v {
			if s, ok := part.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ""), nil
	default:
		return "", fmt.Errorf("unexpected output type from replicate: %T", output)
	}
}
