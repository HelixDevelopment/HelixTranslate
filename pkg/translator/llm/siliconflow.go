package llm

import (
	"context"
	"fmt"
)

// SiliconFlowClient implements LLMClient for SiliconFlow API.
type SiliconFlowClient struct {
	apiKey  string
	model   string
	baseURL string
}

// NewSiliconFlowClient creates a new SiliconFlow client.
func NewSiliconFlowClient(config TranslationConfig) (*SiliconFlowClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("siliconflow API key is required")
	}
	return &SiliconFlowClient{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: defaultString(config.BaseURL, "https://api.siliconflow.cn/v1"),
	}, nil
}

// Translate performs translation via SiliconFlow.
func (c *SiliconFlowClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	return "", fmt.Errorf("siliconflow translation not yet implemented")
}

// GetProviderName returns the provider name.
func (c *SiliconFlowClient) GetProviderName() string {
	return "siliconflow"
}
