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

// GeminiClient implements the LLMClient interface for Google Gemini
type GeminiClient struct {
	config     TranslationConfig
	httpClient *http.Client
	baseURL    string
}

// GeminiRequest represents a request to the Gemini API
type GeminiRequest struct {
	Contents         []GeminiContent         `json:"contents"`
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafetySetting   `json:"safetySettings,omitempty"`
}

// GeminiContent represents content in a Gemini request
type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

// GeminiPart represents a part of content
type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents generation configuration
type GeminiGenerationConfig struct {
	Temperature     float64  `json:"temperature,omitempty"`
	TopK            int      `json:"topK,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// GeminiSafetySetting represents safety settings
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents a response from the Gemini API
type GeminiResponse struct {
	Candidates    []GeminiCandidate    `json:"candidates"`
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a candidate response
type GeminiCandidate struct {
	Content       GeminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"`
	Index         int                  `json:"index"`
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings"`
}

// GeminiSafetyRating represents safety ratings
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GeminiUsageMetadata represents usage metadata
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(config TranslationConfig) (*GeminiClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Gemini")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &GeminiClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Default timeout
		},
		baseURL: baseURL,
	}, nil
}

// Translate performs translation using Google Gemini
func (g *GeminiClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("text is required")
	}

	// Build the full prompt
	fullPrompt := g.buildPrompt(text, prompt)

	// Precedence: Options[...] (explicit per-call override) > typed config field
	// (the -temperature / -max-tokens CLI flags) > hardcoded default. Sibling
	// clients (openai/anthropic/zhipu/qwen) already plumb these through; Gemini
	// previously hardcoded 0.3 / 4000, silently dropping the operator's flags and
	// capping large book sections at 4000 output tokens.
	temperature := 0.3
	if g.config.Temperature > 0 { // typed CLI/config field (0 == unset)
		temperature = g.config.Temperature
	}
	if v, ok := toFloat64(g.config.Options["temperature"]); ok {
		temperature = v
	}

	maxTokens := 4000
	if g.config.MaxTokens > 0 { // typed CLI/config field (0 == unset)
		maxTokens = g.config.MaxTokens
	}
	if v, ok := toInt(g.config.Options["max_tokens"]); ok {
		maxTokens = v
	}

	// Create the request
	geminiReq := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: fullPrompt},
				},
				Role: "user",
			},
		},
		GenerationConfig: &GeminiGenerationConfig{
			Temperature:     temperature,
			TopK:            40,
			TopP:            0.95,
			MaxOutputTokens: maxTokens,
		},
		SafetySettings: []GeminiSafetySetting{
			{
				Category:  "HARM_CATEGORY_HARASSMENT",
				Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			},
			{
				Category:  "HARM_CATEGORY_HATE_SPEECH",
				Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			},
			{
				Category:  "HARM_CATEGORY_SEXUALLY_EXPLICIT",
				Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			},
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_MEDIUM_AND_ABOVE",
			},
		},
	}

	// Make the API request
	resp, err := g.makeRequest(ctx, geminiReq)
	if err != nil {
		return "", fmt.Errorf("failed to make Gemini request: %w", err)
	}

	// Parse the response
	translatedText, err := g.parseResponse(resp)
	if err != nil {
		return "", fmt.Errorf("failed to parse Gemini response: %w", err)
	}

	return translatedText, nil
}

// buildPrompt creates the translation prompt
func (g *GeminiClient) buildPrompt(text, prompt string) string {
	var fullPrompt strings.Builder

	if prompt != "" {
		fullPrompt.WriteString(prompt)
		fullPrompt.WriteString("\n\n")
	}

	fullPrompt.WriteString("Text to translate:\n")
	fullPrompt.WriteString(text)
	fullPrompt.WriteString("\n\n")
	fullPrompt.WriteString("Provide only the translated text without any explanations or additional formatting.")

	return fullPrompt.String()
}

// makeRequest sends a request to the Gemini API
func (g *GeminiClient) makeRequest(ctx context.Context, req GeminiRequest) (*GeminiResponse, error) {
	// Build the URL
	model := g.config.Model
	if model == "" {
		model = "gemini-pro"
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s",
		g.baseURL,
		model,
		g.config.APIKey)

	// Marshal the request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for candidates
	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in Gemini response")
	}

	return &geminiResp, nil
}

// parseResponse extracts the translated text from the Gemini response
func (g *GeminiClient) parseResponse(resp *GeminiResponse) (string, error) {
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]

	// Extract text from parts first — MAX_TOKENS returns usable partial text per
	// the Gemini API reference (ai.google.dev/api/generate-content#FinishReason),
	// so we must not discard it.
	var translatedText strings.Builder
	for _, part := range candidate.Content.Parts {
		translatedText.WriteString(part.Text)
	}
	result := strings.TrimSpace(translatedText.String())

	// STOP (natural completion) and MAX_TOKENS (truncated-but-usable) both yield
	// usable text, so return it. An empty/omitted reason is treated the same way
	// (some responses omit the field). Any other reason
	// (SAFETY/RECITATION/OTHER/...) is a genuine block where no usable text was
	// produced: surface it as an error.
	switch candidate.FinishReason {
	case "STOP", "MAX_TOKENS", "":
		return result, nil
	default:
		return "", fmt.Errorf("generation did not complete successfully: %s", candidate.FinishReason)
	}
}

// GetProviderName returns the provider name
func (g *GeminiClient) GetProviderName() string {
	return "gemini"
}
