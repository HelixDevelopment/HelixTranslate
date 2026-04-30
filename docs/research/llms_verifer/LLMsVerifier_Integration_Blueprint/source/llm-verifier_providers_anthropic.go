// Package providers implements LLM provider adapters
package providers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"digital.vasic.llmsverifier/auth"
)

// AuthType represents the type of authentication used
type AuthType string

const (
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeOAuth  AuthType = "oauth"
)

// AnthropicAdapter provides Anthropic-specific functionality
type AnthropicAdapter struct {
	BaseAdapter
	authType        AuthType
	oauthCredReader *auth.OAuthCredentialReader
}

// NewAnthropicAdapter creates a new Anthropic adapter
func NewAnthropicAdapter(client *http.Client, endpoint, apiKey string) *AnthropicAdapter {
	return &AnthropicAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   apiKey,
			headers: map[string]string{
				"Content-Type":      "application/json",
				"anthropic-version": "2023-06-01",
				"x-api-key":         apiKey,
			},
		},
		authType: AuthTypeAPIKey,
	}
}

// NewAnthropicAdapterWithOAuth creates a new Anthropic adapter using OAuth credentials from Claude Code CLI
func NewAnthropicAdapterWithOAuth(client *http.Client, endpoint string) (*AnthropicAdapter, error) {
	credReader := auth.GetGlobalOAuthReader()

	// Verify credentials are available
	if !credReader.HasValidClaudeCredentials() {
		return nil, fmt.Errorf("no valid Claude OAuth credentials available: ensure you are logged in via Claude Code CLI")
	}

	token, err := credReader.GetClaudeAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get Claude OAuth token: %w", err)
	}

	return &AnthropicAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   "", // Will use OAuth token instead
			headers: map[string]string{
				"Content-Type":      "application/json",
				"anthropic-version": "2023-06-01",
				"Authorization":     "Bearer " + token,
			},
		},
		authType:        AuthTypeOAuth,
		oauthCredReader: credReader,
	}, nil
}

// NewAnthropicAdapterAuto creates an Anthropic adapter, automatically choosing OAuth if enabled and available
func NewAnthropicAdapterAuto(client *http.Client, endpoint, apiKey string) (*AnthropicAdapter, error) {
	// Check if OAuth is enabled and credentials are available
	if auth.IsClaudeOAuthEnabled() {
		credReader := auth.GetGlobalOAuthReader()
		if credReader.HasValidClaudeCredentials() {
			return NewAnthropicAdapterWithOAuth(client, endpoint)
		}
	}

	// Fall back to API key authentication
	if apiKey == "" {
		return nil, fmt.Errorf("no API key provided and OAuth credentials not available")
	}
	return NewAnthropicAdapter(client, endpoint, apiKey), nil
}

// GetAuthType returns the authentication type being used
func (a *AnthropicAdapter) GetAuthType() AuthType {
	return a.authType
}

// refreshAuthHeaders refreshes the authentication headers if using OAuth
func (a *AnthropicAdapter) refreshAuthHeaders() error {
	if a.authType != AuthTypeOAuth || a.oauthCredReader == nil {
		return nil
	}

	token, err := a.oauthCredReader.GetClaudeAccessToken()
	if err != nil {
		return fmt.Errorf("failed to refresh OAuth token: %w", err)
	}

	a.headers["Authorization"] = "Bearer " + token
	delete(a.headers, "x-api-key") // Ensure we're not mixing auth methods
	return nil
}

// AnthropicChatRequest represents a chat completion request for Anthropic
type AnthropicChatRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
	System        string             `json:"system,omitempty"`
	Metadata      *AnthropicMetadata `json:"metadata,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
}

// AnthropicMessage represents a message in Anthropic format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicMetadata represents metadata for Anthropic requests
type AnthropicMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

// AnthropicChatResponse represents a chat completion response from Anthropic
type AnthropicChatResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []AnthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      AnthropicUsage     `json:"usage"`
}

// AnthropicContent represents content in Anthropic responses
type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// AnthropicUsage represents token usage information
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicStreamResponse represents a streaming response from Anthropic
type AnthropicStreamResponse struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta struct {
		Type string `json:"type,omitempty"`
		Text string `json:"text,omitempty"`
	} `json:"delta,omitempty"`
}

// convertToAnthropicRequest converts OpenAI format to Anthropic format
func (a *AnthropicAdapter) convertToAnthropicRequest(openaiReq OpenAIChatRequest) AnthropicChatRequest {
	anthropicReq := AnthropicChatRequest{
		Model:     openaiReq.Model,
		MaxTokens: openaiReq.MaxTokens,
		Stream:    openaiReq.Stream,
	}

	// Convert messages from OpenAI to Anthropic format
	for _, msg := range openaiReq.Messages {
		anthropicMsg := AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		anthropicReq.Messages = append(anthropicReq.Messages, anthropicMsg)
	}

	// Set optional parameters if they are non-zero
	if openaiReq.Temperature != 0 {
		anthropicReq.Temperature = &openaiReq.Temperature
	}
	if openaiReq.TopP != 0 {
		anthropicReq.TopP = &openaiReq.TopP
	}

	return anthropicReq
}

// convertFromAnthropicResponse converts Anthropic format to OpenAI format
func (a *AnthropicAdapter) convertFromAnthropicResponse(anthropicResp AnthropicChatResponse) OpenAIChatResponse {
	openaiResp := OpenAIChatResponse{
		ID:      anthropicResp.ID,
		Object:  "chat.completion",
		Created: 0, // Anthropic doesn't provide creation time
		Model:   anthropicResp.Model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	// Convert content
	if len(anthropicResp.Content) > 0 {
		content := ""
		for _, c := range anthropicResp.Content {
			if c.Type == "text" {
				content += c.Text
			}
		}
		openaiResp.Choices = append(openaiResp.Choices, struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{
			Index: 0,
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    anthropicResp.Role,
				Content: content,
			},
		})
	}

	return openaiResp
}

// StreamChatCompletion streams a chat completion from Anthropic
func (a *AnthropicAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	// Refresh OAuth headers if needed (do this synchronously before starting goroutine)
	if err := a.refreshAuthHeaders(); err != nil {
		go func() {
			defer close(responseChan)
			defer close(errorChan)
			errorChan <- fmt.Errorf("failed to refresh auth: %w", err)
		}()
		return responseChan, errorChan
	}

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		anthropicReq := a.convertToAnthropicRequest(request)
		anthropicReq.Stream = true

		// Prepare request body
		requestBody, err := json.Marshal(anthropicReq)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/messages", a.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range a.headers {
			req.Header.Set(key, value)
		}

		// Send request
		resp, err := a.client.Do(req)
		if err != nil {
			errorChan <- fmt.Errorf("failed to send request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			return
		}

		// Parse streaming response
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					break
				}

				var anthropicStream AnthropicStreamResponse
				if err := json.Unmarshal([]byte(data), &anthropicStream); err != nil {
					continue // Skip malformed lines
				}

				// Convert to OpenAI format
				openaiStream := OpenAIStreamResponse{
					ID:      "anthropic-stream",
					Object:  "chat.completion.chunk",
					Created: 0,
					Model:   request.Model,
					Choices: []OpenAIChoice{},
				}

				if anthropicStream.Delta.Text != "" {
					openaiStream.Choices = append(openaiStream.Choices, OpenAIChoice{
						Index: anthropicStream.Index,
						Delta: OpenAIDelta{
							Content: anthropicStream.Delta.Text,
						},
					})
				}

				select {
				case responseChan <- openaiStream:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading response: %w", err)
		}
	}()

	return responseChan, errorChan
}

// ChatCompletion performs a non-streaming chat completion
func (a *AnthropicAdapter) ChatCompletion(ctx context.Context, request OpenAIChatRequest) (*OpenAIChatResponse, error) {
	// Refresh OAuth headers if needed
	if err := a.refreshAuthHeaders(); err != nil {
		return nil, fmt.Errorf("failed to refresh auth: %w", err)
	}

	anthropicReq := a.convertToAnthropicRequest(request)

	// Prepare request body
	requestBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/messages", a.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range a.headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp AnthropicChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	openaiResp := a.convertFromAnthropicResponse(anthropicResp)
	return &openaiResp, nil
}

// ListModels retrieves available models from Anthropic
// Note: Anthropic doesn't have a public models endpoint, so we maintain a curated list
// that is updated based on their documentation. Last verified: 2025-01
func (a *AnthropicAdapter) ListModels(ctx context.Context) (*OpenAIModelsResponse, error) {
	// First attempt to verify model availability by making a minimal API call
	// This ensures we only return models that are actually accessible with the current API key
	availableModels := a.discoverAvailableModels(ctx)

	modelsResp := &OpenAIModelsResponse{
		Object: "list",
		Data:   availableModels,
	}

	return modelsResp, nil
}

// discoverAvailableModels attempts to verify which models are accessible
func (a *AnthropicAdapter) discoverAvailableModels(ctx context.Context) []struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
} {
	// Known Anthropic models with their release timestamps
	// Updated based on Anthropic's model documentation
	knownModels := []struct {
		ID      string
		Created int64
	}{
		// Claude 3.5 family (latest)
		{ID: "claude-3-5-sonnet-latest", Created: 1729036800},    // Oct 2024
		{ID: "claude-3-5-sonnet-20241022", Created: 1729555200},  // Oct 2024
		{ID: "claude-3-5-haiku-latest", Created: 1730419200},     // Nov 2024
		{ID: "claude-3-5-haiku-20241022", Created: 1729555200},   // Oct 2024
		// Claude 3 family
		{ID: "claude-3-opus-latest", Created: 1709251200},        // Feb 2024
		{ID: "claude-3-opus-20240229", Created: 1709251200},      // Feb 2024
		{ID: "claude-3-sonnet-20240229", Created: 1709251200},    // Feb 2024
		{ID: "claude-3-haiku-20240307", Created: 1709856000},     // Mar 2024
	}

	var availableModels []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	// Try to verify at least one model to confirm API access
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	apiAccessible := a.verifyAPIAccess(testCtx)

	for _, model := range knownModels {
		availableModels = append(availableModels, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      model.ID,
			Object:  "model",
			Created: model.Created,
			OwnedBy: "anthropic",
		})
	}

	// If API is not accessible, mark models as potentially unavailable in logs
	if !apiAccessible {
		log.Printf("Warning: Anthropic API access could not be verified. Model list may be stale.")
	}

	return availableModels
}

// verifyAPIAccess checks if the API is accessible with current credentials
func (a *AnthropicAdapter) verifyAPIAccess(ctx context.Context) bool {
	// Make a minimal request to verify API access
	req, err := http.NewRequestWithContext(ctx, "POST", a.endpoint+"/messages", strings.NewReader(`{
		"model": "claude-3-haiku-20240307",
		"max_tokens": 1,
		"messages": [{"role": "user", "content": "test"}]
	}`))
	if err != nil {
		return false
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Any response (even error) means API is accessible
	return resp.StatusCode != 0
}
