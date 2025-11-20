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

// ZhipuClient implements Zhipu AI (GLM) API client
type ZhipuClient struct {
	config     translator.TranslationConfig
	httpClient *http.Client
	baseURL    string
}

// ZhipuRequest represents Zhipu API request
type ZhipuRequest struct {
	Model       string          `json:"model"`
	Messages    []ZhipuMessage  `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// ZhipuMessage represents a message
type ZhipuMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ZhipuResponse represents Zhipu API response
type ZhipuResponse struct {
	ID      string        `json:"id"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ZhipuChoice `json:"choices"`
	Usage   ZhipuUsage    `json:"usage"`
}

// ZhipuChoice represents a response choice
type ZhipuChoice struct {
	Index        int          `json:"index"`
	Message      ZhipuMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// ZhipuUsage represents token usage
type ZhipuUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewZhipuClient creates a new Zhipu client
func NewZhipuClient(config translator.TranslationConfig) (*ZhipuClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Zhipu API key is required")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}

	return &ZhipuClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: baseURL,
	}, nil
}

// GetProviderName returns the provider name
func (c *ZhipuClient) GetProviderName() string {
	return "zhipu"
}

// Translate translates text using Zhipu AI
func (c *ZhipuClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	model := c.config.Model
	if model == "" {
		model = "glm-4"
	}

	temperature := 0.3
	if c.config.Options["temperature"] != nil {
		if t, ok := c.config.Options["temperature"].(float64); ok {
			temperature = t
		}
	}

	maxTokens := 4000
	if c.config.Options["max_tokens"] != nil {
		if mt, ok := c.config.Options["max_tokens"].(int); ok {
			maxTokens = mt
		}
	}

	request := ZhipuRequest{
		Model: model,
		Messages: []ZhipuMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

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
		return "", fmt.Errorf("Zhipu API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response ZhipuResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}
