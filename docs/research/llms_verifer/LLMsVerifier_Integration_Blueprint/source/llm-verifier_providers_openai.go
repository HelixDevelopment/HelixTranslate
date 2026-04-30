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
)

// OpenAIAdapter provides OpenAI-specific functionality
type OpenAIAdapter struct {
	BaseAdapter
}

// NewOpenAIAdapter creates a new OpenAI adapter
func NewOpenAIAdapter(client *http.Client, endpoint, apiKey string) *OpenAIAdapter {
	return &OpenAIAdapter{
		BaseAdapter: BaseAdapter{
			client:   client,
			endpoint: strings.TrimSuffix(endpoint, "/"),
			apiKey:   apiKey,
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": fmt.Sprintf("Bearer %s", apiKey),
			},
		},
	}
}

// StreamChatCompletion streams a chat completion from OpenAI
func (o *OpenAIAdapter) StreamChatCompletion(ctx context.Context, request OpenAIChatRequest) (<-chan OpenAIStreamResponse, <-chan error) {
	responseChan := make(chan OpenAIStreamResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// Prepare request body
		requestBody, err := json.Marshal(request)
		if err != nil {
			errorChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request
		url := fmt.Sprintf("%s/chat/completions", o.endpoint)
		req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
		if err != nil {
			errorChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		// Set headers
		for key, value := range o.headers {
			req.Header.Set(key, value)
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")

		// Make request
		log.Printf("<-- %s %s", req.Method, req.URL.String())
		resp, err := o.client.Do(req)
		if err != nil {
			log.Printf("<-- %s %s --> ERROR %v", req.Method, req.URL.String(), err)
			errorChan <- fmt.Errorf("failed to make request: %w", err)
			return
		}
		log.Printf("<-- %s %s --> %d %s", req.Method, req.URL.String(), resp.StatusCode, resp.Status)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errorChan <- fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
			return
		}

		// Parse SSE stream
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if strings.TrimSpace(line) == "" {
				continue
			}

			// Parse SSE format: "data: {...}"
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				// Handle special cases
				if data == "[DONE]" {
					break
				}

				var streamResp OpenAIStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					// Log error but continue processing
					continue
				}

				select {
				case responseChan <- streamResp:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errorChan <- fmt.Errorf("error reading stream: %w", err)
		}
	}()

	return responseChan, errorChan
}

// OpenAIChatRequest represents a chat completion request for OpenAI
type OpenAIChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// OpenAIStreamResponse represents a streaming response from OpenAI
type OpenAIStreamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
}

// OpenAIChoice represents a choice in the OpenAI response
type OpenAIChoice struct {
	Index        int         `json:"index"`
	Delta        OpenAIDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// OpenAIDelta represents the delta in a streaming response
type OpenAIDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ValidateRequest validates an OpenAI chat request
func (o *OpenAIAdapter) ValidateRequest(request OpenAIChatRequest) error {
	if request.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(request.Messages) == 0 {
		return fmt.Errorf("at least one message is required")
	}
	if request.MaxTokens < 0 {
		return fmt.Errorf("max_tokens cannot be negative")
	}
	if request.Temperature < 0 || request.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if request.TopP < 0 || request.TopP > 1 {
		return fmt.Errorf("top_p must be between 0 and 1")
	}
	return nil
}

// GetModelInfo retrieves model information from OpenAI
func (o *OpenAIAdapter) GetModelInfo(ctx context.Context, model string) (*ModelInfo, error) {
	url := fmt.Sprintf("%s/models/%s", o.endpoint, model)
	log.Printf("API REQUEST: GET %s", url)
	log.Printf("API REQUEST HEADERS: %v", o.headers)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range o.headers {
		req.Header.Set(key, value)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		log.Printf("API RESPONSE ERROR: %v", err)
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("API RESPONSE STATUS: %d", resp.StatusCode)
	log.Printf("API RESPONSE HEADERS: %v", resp.Header)
	log.Printf("API RESPONSE BODY: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var modelResp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	if err := json.Unmarshal(body, &modelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &ModelInfo{
		ID:      modelResp.ID,
		Object:  modelResp.Object,
		Created: modelResp.Created,
		OwnedBy: modelResp.OwnedBy,
	}, nil
}

// ModelInfo represents model information
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}
