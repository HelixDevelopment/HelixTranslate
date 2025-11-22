package language

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

// SimpleLLMDetector implements LLM-based language detection
type SimpleLLMDetector struct {
	apiKey   string
	provider string
	baseURL  string
	model    string
	client   *http.Client
}

// NewSimpleLLMDetector creates a new LLM detector
func NewSimpleLLMDetector(provider, apiKey string) *SimpleLLMDetector {
	detector := &SimpleLLMDetector{
		apiKey:   apiKey,
		provider: provider,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set provider-specific defaults
	switch provider {
	case "openai":
		detector.baseURL = "https://api.openai.com/v1"
		detector.model = "gpt-3.5-turbo"
	case "deepseek":
		detector.baseURL = "https://api.deepseek.com/v1"
		detector.model = "deepseek-chat"
	case "anthropic":
		detector.baseURL = "https://api.anthropic.com/v1"
		detector.model = "claude-3-haiku-20240307"
	case "zhipu":
		detector.baseURL = "https://open.bigmodel.cn/api/paas/v4"
		detector.model = "glm-4"
	default:
		detector.baseURL = "https://api.openai.com/v1"
		detector.model = "gpt-3.5-turbo"
	}

	return detector
}

// DetectLanguage detects language using LLM
func (d *SimpleLLMDetector) DetectLanguage(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("empty text provided")
	}

	// Sample text (first 500 characters)
	sample := text
	if len(text) > 500 {
		sample = text[:500]
	}

	// Create prompt for language detection
	prompt := fmt.Sprintf(`Identify the language of the following text.
Respond with ONLY the ISO 639-1 language code (e.g., "en" for English, "ru" for Russian, "sr" for Serbian, "de" for German).
Do not include any explanation, just the 2-letter code.

Text:
%s

Language code:`, sample)

	// Call LLM API based on provider
	switch d.provider {
	case "openai", "deepseek":
		return d.callOpenAICompatible(ctx, prompt)
	case "anthropic":
		return d.callAnthropic(ctx, prompt)
	case "zhipu":
		return d.callZhipu(ctx, prompt)
	default:
		return d.callOpenAICompatible(ctx, prompt)
	}
}

// callOpenAICompatible calls OpenAI-compatible APIs (OpenAI, DeepSeek)
func (d *SimpleLLMDetector) callOpenAICompatible(ctx context.Context, prompt string) (string, error) {
	request := map[string]interface{}{
		"model": d.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0, // Deterministic response
		"max_tokens":  10,  // Only need a short response
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	return FormatLanguageCode(content), nil
}

// callAnthropic calls Anthropic Claude API
func (d *SimpleLLMDetector) callAnthropic(ctx context.Context, prompt string) (string, error) {
	request := map[string]interface{}{
		"model":      d.model,
		"max_tokens": 10,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", d.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	content := strings.TrimSpace(response.Content[0].Text)
	return FormatLanguageCode(content), nil
}

// callZhipu calls Zhipu AI API
func (d *SimpleLLMDetector) callZhipu(ctx context.Context, prompt string) (string, error) {
	request := map[string]interface{}{
		"model": d.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
		"max_tokens":  10,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.apiKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := strings.TrimSpace(response.Choices[0].Message.Content)
	return FormatLanguageCode(content), nil
}

// FormatLanguageCode normalizes language codes
func FormatLanguageCode(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))

	// Handle common variations
	if len(code) > 2 {
		code = code[:2]
	}

	return code
}
