package llm

import (
	"context"
	"fmt"
)

// GroqClient implements LLMClient for Groq API.
type GroqClient struct {
	apiKey string
	model  string
	baseURL string
}

// NewGroqClient creates a new Groq client.
func NewGroqClient(config TranslationConfig) (*GroqClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("groq API key is required")
	}
	return &GroqClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.groq.com/openai/v1"),
	}, nil
}

// Translate performs translation via Groq.
func (c *GroqClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	// Groq is OpenAI-compatible; delegates to generic OpenAI-compatible implementation
	// Full implementation would use the same HTTP client pattern as OpenAI
	return "", fmt.Errorf("groq translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *GroqClient) GetProviderName() string {
	return "groq"
}
