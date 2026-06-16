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

// OpenAIClient implements OpenAI API client
type OpenAIClient struct {
	config     TranslationConfig
	httpClient *http.Client
	baseURL    string
}

// OpenAIRequest represents OpenAI API request
type OpenAIRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents OpenAI API response
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config TranslationConfig) (*OpenAIClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Validate model if provided (skip validation for delegated providers)
	if config.Model != "" {
		// Skip validation for non-OpenAI providers that delegate to OpenAI
		isDelegated := false
		if config.Provider != "" && config.Provider != "openai" {
			isDelegated = true
		}

		if !isDelegated {
			if strings.TrimSpace(config.Model) == "" {
				return nil, fmt.Errorf("model cannot be empty or whitespace")
			}
			validModels := ValidModels[ProviderOpenAI]
			modelValid := false
			for _, validModel := range validModels {
				if config.Model == validModel {
					modelValid = true
					break
				}
			}
			if !modelValid {
				return nil, fmt.Errorf("model '%s' is not valid for OpenAI. Valid models: %v",
					config.Model, validModels)
			}
		}
	}

	// Validate temperature if provided
	if temp, exists := config.Options["temperature"]; exists {
		if tempFloat, ok := temp.(float64); ok {
			if tempFloat < 0.0 || tempFloat > 2.0 {
				return nil, fmt.Errorf("temperature %.1f is invalid for OpenAI. Must be between 0.0 and 2.0", tempFloat)
			}
		}
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 600 * time.Second, // Increased to 10 minutes for very large book sections (up to 44KB)
		},
		baseURL: baseURL,
	}, nil
}

// GetProviderName returns the provider name
func (c *OpenAIClient) GetProviderName() string {
	return "openai"
}

// Translate translates text using OpenAI.
//
// Wire contract (FACT): the user message sent to the model is the 2nd arg
// (`prompt`); the 1st arg (`text`) is the translatable content some callers pass.
// Defense-in-depth (§11.4.69, fixes the empty-payload data-loss class — see
// docs/qa/translate_arg_audit_completion_20260616-154307/COMPLETION.md):
//  1. when `prompt == ""` fall back to using `text` as the user message, so
//     content-in-1st-arg callers (markdown-translator, preparation ensemble) no
//     longer send an empty user message that the model answers with boilerplate;
//  2. refuse an empty/whitespace-only user message with an explicit error rather
//     than letting provider boilerplate be stored as if it were a translation.
func (c *OpenAIClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	userMessage := prompt
	if strings.TrimSpace(userMessage) == "" {
		// prompt empty/whitespace → fall back to the content-bearing 1st arg.
		userMessage = text
	}
	if strings.TrimSpace(userMessage) == "" {
		// Both args empty/whitespace: never send an empty user message (the model
		// would reply with "you sent an empty message" boilerplate that downstream
		// code stores as a translation — silent data loss). Fail loudly instead.
		return "", fmt.Errorf("OpenAI Translate: refusing to send an empty user message (both text and prompt are empty/whitespace)")
	}

	model := c.config.Model
	if model == "" {
		model = "gpt-4"
	}

	// Precedence: Options[...] (explicit per-call override) > typed config field
	// (e.g. the -temperature CLI flag) > hardcoded default.
	//
	// temperature may arrive as float64 (literal), int, float32, or json.Number
	// because Options is map[string]interface{} populated from JSON/config/code.
	// A bare `.(float64)` assertion panics on any other concrete type, so coerce
	// defensively (sibling clients anthropic/qwen/zhipu use the checked form).
	temperature := 0.3
	if c.config.Temperature > 0 { // typed CLI/config field (0 == unset)
		temperature = c.config.Temperature
	}
	if v, ok := toFloat64(c.config.Options["temperature"]); ok {
		temperature = v
	}

	// Increase max_tokens for large translations (book sections can be very long)
	// DeepSeek/OpenAI compatible models support up to 8192 max_tokens
	maxTokens := 8192 // Increased from 4000 to handle book chapters (DeepSeek max)
	if c.config.MaxTokens > 0 { // typed CLI/config field (0 == unset)
		maxTokens = c.config.MaxTokens
	}
	if v, ok := toInt(c.config.Options["max_tokens"]); ok {
		maxTokens = v
	}

	request := OpenAIRequest{
		Model: model,
		Messages: []Message{
			{Role: "user", Content: userMessage},
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
		return "", fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}
