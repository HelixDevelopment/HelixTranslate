package llm

import (
	"bytes"
	"context"
	"digital.vasic.translator/pkg/translator"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// QwenClient implements Qwen (Alibaba Cloud) LLM API client with OAuth support
type QwenClient struct {
	config       translator.TranslationConfig
	httpClient   *http.Client
	baseURL      string
	oauthToken   *QwenOAuthToken
	credFilePath string
}

// QwenOAuthToken represents OAuth credentials for Qwen
type QwenOAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ResourceURL  string `json:"resource_url"`
	ExpiryDate   int64  `json:"expiry_date"`
}

// QwenRequest represents Qwen API request
type QwenRequest struct {
	Model      string        `json:"model"`
	Messages   []QwenMessage `json:"messages"`
	Stream     bool          `json:"stream"`
	MaxTokens  int           `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
}

// QwenMessage represents a message
type QwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// QwenResponse represents Qwen API response
type QwenResponse struct {
	ID      string       `json:"id"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []QwenChoice `json:"choices"`
	Usage   QwenUsage    `json:"usage"`
}

// QwenChoice represents a response choice
type QwenChoice struct {
	Index        int         `json:"index"`
	Message      QwenMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// QwenUsage represents token usage
type QwenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewQwenClient creates a new Qwen client with OAuth support
func NewQwenClient(config translator.TranslationConfig) (*QwenClient, error) {
	credDir := os.Getenv("HOME")
	if credDir == "" {
		credDir = "."
	}

	// Primary location for translator-specific credentials
	credFile := filepath.Join(credDir, ".translator", "qwen_credentials.json")

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/api/v1"
	}

	client := &QwenClient{
		config:       config,
		httpClient:   &http.Client{Timeout: 600 * time.Second}, // Increased to 10 minutes for very large book sections (up to 44KB)
		baseURL:      baseURL,
		credFilePath: credFile,
	}

	// Load OAuth token from file or use API key
	if config.APIKey != "" {
		// API key provided - use it directly
		return client, nil
	}

	// Try loading OAuth token from translator-specific location
	if err := client.loadOAuthToken(); err != nil {
		// Try Qwen Code standard location as fallback
		qwenCodeCredFile := filepath.Join(credDir, ".qwen", "oauth_creds.json")
		client.credFilePath = qwenCodeCredFile
		if err := client.loadOAuthToken(); err != nil {
			return nil, fmt.Errorf("no API key or valid OAuth token found: %w\nPlease set QWEN_API_KEY environment variable or authenticate via OAuth", err)
		}
	}

	// Note: We don't pre-emptively refresh expired tokens on initialization
	// Instead, we'll attempt to use the token and only refresh if we get a 401 error
	// This allows tokens to work even if the expiry date calculation is off
	if client.isTokenExpired() {
		// Log warning but continue - will refresh on 401
		fmt.Fprintf(os.Stderr, "Warning: Qwen OAuth token appears expired, will attempt to use it anyway\n")
	}

	return client, nil
}

// loadOAuthToken loads OAuth token from credentials file
func (c *QwenClient) loadOAuthToken() error {
	data, err := os.ReadFile(c.credFilePath)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	var token QwenOAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	c.oauthToken = &token
	return nil
}

// saveOAuthToken saves OAuth token to credentials file
func (c *QwenClient) saveOAuthToken(token *QwenOAuthToken) error {
	// Ensure directory exists
	dir := filepath.Dir(c.credFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(c.credFilePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	c.oauthToken = token
	return nil
}

// SetOAuthToken sets OAuth token from external source (e.g., OAuth flow)
func (c *QwenClient) SetOAuthToken(accessToken, refreshToken, resourceURL string, expiryDate int64) error {
	token := &QwenOAuthToken{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: refreshToken,
		ResourceURL:  resourceURL,
		ExpiryDate:   expiryDate,
	}
	return c.saveOAuthToken(token)
}

// isTokenExpired checks if the OAuth token is expired
func (c *QwenClient) isTokenExpired() bool {
	if c.oauthToken == nil {
		return true
	}
	// Consider token expired if less than 5 minutes until expiry
	return time.Now().Unix() > (c.oauthToken.ExpiryDate/1000 - 300)
}

// refreshToken refreshes the OAuth token
func (c *QwenClient) refreshToken() error {
	if c.oauthToken == nil || c.oauthToken.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Qwen OAuth refresh endpoint (based on Alibaba Cloud API)
	refreshURL := "https://oauth.aliyun.com/v1/token"

	// Prepare refresh request
	reqData := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": c.oauthToken.RefreshToken,
		"client_id":     os.Getenv("QWEN_CLIENT_ID"),
		"client_secret": os.Getenv("QWEN_CLIENT_SECRET"),
	}

	// Check for required environment variables
	if reqData["client_id"] == "" {
		return fmt.Errorf("QWEN_CLIENT_ID environment variable not set")
	}
	if reqData["client_secret"] == "" {
		return fmt.Errorf("QWEN_CLIENT_SECRET environment variable not set")
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal refresh request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "POST", refreshURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var refreshResponse struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read refresh response: %w", err)
	}

	if err := json.Unmarshal(body, &refreshResponse); err != nil {
		return fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Update OAuth token with new values
	if refreshResponse.AccessToken == "" {
		return fmt.Errorf("refresh response missing access token")
	}

	c.oauthToken.AccessToken = refreshResponse.AccessToken
	c.oauthToken.TokenType = refreshResponse.TokenType
	c.oauthToken.ExpiryDate = time.Now().UnixMilli() + (refreshResponse.ExpiresIn * 1000)

	// Update refresh token if provided (some providers rotate refresh tokens)
	if refreshResponse.RefreshToken != "" {
		c.oauthToken.RefreshToken = refreshResponse.RefreshToken
	}

	// Save updated token to file
	if err := c.saveOAuthToken(); err != nil {
		return fmt.Errorf("failed to save refreshed token: %w", err)
	}

	return nil
}

// GetProviderName returns the provider name
func (c *QwenClient) GetProviderName() string {
	return "qwen"
}

// Translate translates text using Qwen (Alibaba Cloud) LLM
func (c *QwenClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	model := c.config.Model
	if model == "" {
		model = "qwen-plus" // Default model
	}

	temperature := 0.3
	if c.config.Options["temperature"] != nil {
		if t, ok := c.config.Options["temperature"].(float64); ok {
			temperature = t
		}
	}

	maxTokens := 4000
	if c.config.Options["max_tokens"] != nil {
		if mt, ok := c.config.Options["max_tokens"].(int); ok {
			maxTokens = mt
		}
	}

	request := QwenRequest{
		Model: model,
		Messages: []QwenMessage{
			{Role: "user", Content: prompt},
		},
		Stream:      false,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/services/aigc/text-generation/generation", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Use OAuth token or API key
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	} else if c.oauthToken != nil {
		req.Header.Set("Authorization", c.oauthToken.TokenType+" "+c.oauthToken.AccessToken)
	} else {
		return "", fmt.Errorf("no authentication credentials available")
	}

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
		// Check if token expired
		if resp.StatusCode == http.StatusUnauthorized && c.oauthToken != nil {
			if err := c.refreshToken(); err == nil {
				// Retry with refreshed token
				return c.Translate(ctx, text, prompt)
			}
		}
		return "", fmt.Errorf("Qwen API error (status %d): %s", resp.StatusCode, string(body))
	}

	var response QwenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}
