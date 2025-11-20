package llm

import (
	"bytes"
	"context"
	"digital.vasic.translator/pkg/translator"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicClient implements Anthropic Claude API client
type AnthropicClient struct {
	config     translator.TranslationConfig
	httpClient *http.Client
	baseURL    string
}

// AnthropicRequest represents Anthropic API request
type AnthropicRequest struct {
	Model       string            `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens"`
	Temperature float64           `json:"temperature,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents Anthropic API response
type AnthropicResponse struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Role    string   `json:"role"`
	Content []Content `json:"content"`
	Model   string   `json:"model"`
	Usage   AnthropicUsage `json:"usage"`
}

// Content represents content block
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicUsage represents token usage
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(config translator.TranslationConfig) (*AnthropicClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	return &AnthropicClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: baseURL,
	}, nil
}

// GetProviderName returns the provider name
func (c *AnthropicClient) GetProviderName() string {
	return "anthropic"
}

// Translate translates text using Anthropic Claude
func (c *AnthropicClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	model := c.config.Model
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	temperature := 0.3
	if c.config.Options["temperature"] != nil {
		if t, ok := c.config.Options["temperature"].(float64); ok {
			temperature = t
		}
	}

	maxTokens := 4096
	if c.config.Options["max_tokens"] != nil {
		if mt, ok := c.config.Options["max_tokens"].(int); ok {
			maxTokens = mt
		}
	}

	request := AnthropicRequest{
		Model: model,
		Messages: []AnthropicMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response AnthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return response.Content[0].Text, nil
}
