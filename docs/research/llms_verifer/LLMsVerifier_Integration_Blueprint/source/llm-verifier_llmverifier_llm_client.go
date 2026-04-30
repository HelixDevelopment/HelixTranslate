package llmverifier

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

// LLMClient handles communication with LLM APIs
type LLMClient struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
	headers    map[string]string
}

// NewLLMClient creates a new LLM client
func NewLLMClient(endpoint, apiKey string, headers map[string]string) *LLMClient {
	return NewLLMClientWithTimeout(endpoint, apiKey, headers, 60*time.Second)
}

// NewLLMClientWithTimeout creates a new LLM client with custom timeout
func NewLLMClientWithTimeout(endpoint, apiKey string, headers map[string]string, timeout time.Duration) *LLMClient {
	return &LLMClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		endpoint: strings.TrimSuffix(endpoint, "/"),
		apiKey:   apiKey,
		headers:  headers,
	}
}

// ModelResponse represents the response from the models API endpoint
type ModelResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}

// ListModels retrieves all available models from the endpoint
func (c *LLMClient) ListModels(ctx context.Context) ([]ModelInfo, error) {
	url := fmt.Sprintf("%s/models", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)
	c.setAdditionalHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var modelResp ModelResponse
	if err := json.Unmarshal(body, &modelResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return modelResp.Data, nil
}

// ChatCompletionRequest represents a request to the chat completion endpoint
type ChatCompletionRequest struct {
	Model          string                 `json:"model"`
	Messages       []Message              `json:"messages"`
	Stream         bool                   `json:"stream,omitempty"`
	Temperature    *float64               `json:"temperature,omitempty"`
	MaxTokens      *int                   `json:"max_tokens,omitempty"`
	Tools          []Tool                 `json:"tools,omitempty"`
	ToolChoice     interface{}            `json:"tool_choice,omitempty"`
	ResponseFormat map[string]interface{} `json:"response_format,omitempty"`
}

// Message represents a message in the chat completion
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionChoice represents a choice in the chat completion response
type ChatCompletionChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatCompletionResponse represents the response from the chat completion endpoint
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   Usage                  `json:"usage"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletion sends a chat completion request to the model
func (c *LLMClient) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	url := fmt.Sprintf("%s/chat/completions", c.endpoint)
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(httpReq)
	c.setAdditionalHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

// EmbeddingRequest represents a request to the embeddings endpoint
type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

// Embedding represents a single embedding
type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingResponse represents the response from the embeddings endpoint
type EmbeddingResponse struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  Usage       `json:"usage"`
}

// GetEmbeddings gets embeddings for the input text
func (c *LLMClient) GetEmbeddings(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	url := fmt.Sprintf("%s/embeddings", c.endpoint)
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuthHeaders(httpReq)
	c.setAdditionalHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("embedding endpoint not supported")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embedResp EmbeddingResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &embedResp, nil
}

// setAuthHeaders sets authentication headers for the request
func (c *LLMClient) setAuthHeaders(req *http.Request) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
}

// setAdditionalHeaders sets additional custom headers for the request
func (c *LLMClient) setAdditionalHeaders(req *http.Request) {
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
}

// TestConnectivity tests if the endpoint is accessible
func (c *LLMClient) TestConnectivity(ctx context.Context) error {
	// We'll use the models endpoint to test connectivity
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.ListModels(ctxWithTimeout)
	return err
}

// CheckModelExists checks if a specific model exists
func (c *LLMClient) CheckModelExists(ctx context.Context, modelName string) (bool, error) {
	models, err := c.ListModels(ctx)
	if err != nil {
		return false, err
	}

	for _, model := range models {
		if model.ID == modelName {
			return true, nil
		}
	}

	return false, nil
}

// makeRequest performs a generic HTTP request to the LLM API
func (c *LLMClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.endpoint, endpoint)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)
	c.setAdditionalHeaders(req)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// makeRequestWithHeaders performs a generic HTTP request with custom headers
func (c *LLMClient) makeRequestWithHeaders(ctx context.Context, method, endpoint string, body interface{}, customHeaders map[string]string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.endpoint, endpoint)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)
	c.setAdditionalHeaders(req)

	// Add custom headers
	for key, value := range customHeaders {
		req.Header.Set(key, value)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}
