package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient implements Ollama API client (local LLM)
type OllamaClient struct {
	config     TranslationConfig
	httpClient *http.Client
	baseURL    string
}

// OllamaRequest represents Ollama API request
type OllamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  bool           `json:"stream"`
	Options *OllamaOptions `json:"options,omitempty"`
}

// OllamaOptions carries generation parameters. Per the Ollama /api/generate
// reference (https://github.com/ollama/ollama/blob/main/docs/api.md, verified
// 2026-06-14) sampling/length controls such as temperature and num_predict are
// passed inside an "options" object — NOT as top-level request fields.
type OllamaOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// OllamaResponse represents Ollama API response
type OllamaResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(config TranslationConfig) (*OllamaClient, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 600 * time.Second, // Increased to 10 minutes for very large book sections (up to 44KB)
		},
		baseURL: baseURL,
	}, nil
}

// GetProviderName returns the provider name
func (c *OllamaClient) GetProviderName() string {
	return "ollama"
}

// Translate translates text using Ollama
func (c *OllamaClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	model := c.config.Model
	if model == "" {
		model = "llama3:8b"
	}

	// Precedence: Options[...] override > typed config field (CLI flag) > default.
	// Mirrors the openai/anthropic/qwen/zhipu clients so the Ollama path honours
	// the same configured temperature/max_tokens instead of silently dropping
	// them (Ollama would otherwise fall back to its model-default sampling).
	temperature := 0.3
	if c.config.Temperature > 0 {
		temperature = c.config.Temperature
	}
	if t, ok := toFloat64(c.config.Options["temperature"]); ok {
		temperature = t
	}

	// num_predict 0 == "use model default" for Ollama, so leave it unset unless
	// the user configured a budget (typed field or Options override).
	numPredict := 0
	if c.config.MaxTokens > 0 {
		numPredict = c.config.MaxTokens
	}
	if mt, ok := toInt(c.config.Options["max_tokens"]); ok {
		numPredict = mt
	}

	request := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: &OllamaOptions{
			Temperature: temperature,
			NumPredict:  numPredict,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response OllamaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Response, nil
}
