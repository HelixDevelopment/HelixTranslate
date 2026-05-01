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

// CloudflareClient implements LLMClient for Cloudflare Workers AI.
type CloudflareClient struct {
	apiKey    string
	model     string
	accountID string
	baseURL   string
	httpClient *http.Client
}

// cloudflareRequest represents Cloudflare Workers AI request.
type cloudflareRequest struct {
	Messages []cloudflareMessage `json:"messages"`
}

type cloudflareMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// cloudflareResponse represents Cloudflare Workers AI response.
type cloudflareResponse struct {
	Result struct {
		Response string `json:"response"`
	} `json:"result"`
	Success bool     `json:"success"`
	Errors  []string `json:"errors"`
}

// NewCloudflareClient creates a new Cloudflare client.
func NewCloudflareClient(config TranslationConfig) (*CloudflareClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("cloudflare API key is required")
	}

	accountID := ""
	if id, ok := config.Options["account_id"].(string); ok && id != "" {
		accountID = id
	}
	if accountID == "" {
		return nil, fmt.Errorf("cloudflare account_id is required (set in config.Options[\"account_id\"])")
	}

	if config.Model == "" {
		return nil, fmt.Errorf("cloudflare model is required")
	}

	validModels := ValidModels[ProviderCloudflare]
	modelValid := false
	for _, validModel := range validModels {
		if config.Model == validModel {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return nil, fmt.Errorf("model '%s' is not valid for Cloudflare. Valid models: %v",
			config.Model, validModels)
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.cloudflare.com/client/v4"
	}

	return &CloudflareClient{
		apiKey:     config.APIKey,
		model:      config.Model,
		accountID:  accountID,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 600 * time.Second},
	}, nil
}

// GetProviderName returns the provider name.
func (c *CloudflareClient) GetProviderName() string {
	return "cloudflare"
}

// Translate performs translation via Cloudflare Workers AI.
func (c *CloudflareClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	reqBody := cloudflareRequest{
		Messages: []cloudflareMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/accounts/%s/ai/run/%s", c.baseURL, c.accountID, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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
		return "", fmt.Errorf("cloudflare API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response cloudflareResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !response.Success && len(response.Errors) > 0 {
		return "", fmt.Errorf("cloudflare API error: %v", response.Errors)
	}

	if response.Result.Response == "" {
		return "", fmt.Errorf("empty response from cloudflare")
	}

	return response.Result.Response, nil
}
