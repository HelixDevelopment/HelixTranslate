package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an API client for the LLM Verifier
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// New creates a new API client
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// Login authenticates with the API and sets the token
func (c *Client) Login(username, password string) error {
	loginReq := map[string]string{
		"username": username,
		"password": password,
	}

	resp, err := c.doRequest("POST", "/auth/login", loginReq)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	if token, ok := result["token"].(string); ok {
		c.token = token
		return nil
	}

	return fmt.Errorf("no token in login response")
}

// doRequest makes an HTTP request to the API
func (c *Client) doRequest(method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return resp, nil
}

// GetModels retrieves a list of models
func (c *Client) GetModels() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Models []map[string]any `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	return response.Models, nil
}

// GetModel retrieves a specific model by ID
func (c *Client) GetModel(id string) (map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/models/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var model map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode model response: %w", err)
	}

	return model, nil
}

// CreateModel creates a new model
func (c *Client) CreateModel(model map[string]any) (map[string]any, error) {
	resp, err := c.doRequest("POST", "/api/v1/models", model)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode create model response: %w", err)
	}

	return result, nil
}

// VerifyModel triggers verification for a model
func (c *Client) VerifyModel(id string) (map[string]any, error) {
	resp, err := c.doRequest("POST", "/api/v1/models/"+id+"/verify", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode verify model response: %w", err)
	}

	return result, nil
}

// GetProviders retrieves a list of providers
func (c *Client) GetProviders() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/providers", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Providers []map[string]any `json:"providers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode providers response: %w", err)
	}

	return response.Providers, nil
}

// GetProvider retrieves a specific provider by ID
func (c *Client) GetProvider(id string) (map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/providers/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var provider map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&provider); err != nil {
		return nil, fmt.Errorf("failed to decode provider response: %w", err)
	}

	return provider, nil
}

// CreateProvider creates a new provider
func (c *Client) CreateProvider(provider map[string]any) (map[string]any, error) {
	resp, err := c.doRequest("POST", "/api/v1/providers", provider)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode create provider response: %w", err)
	}

	return result, nil
}

// UpdateProvider updates an existing provider
func (c *Client) UpdateProvider(id string, provider map[string]any) (map[string]any, error) {
	resp, err := c.doRequest("PUT", "/api/v1/providers/"+id, provider)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode update provider response: %w", err)
	}

	return result, nil
}

// DeleteProvider deletes a provider by ID
func (c *Client) DeleteProvider(id string) error {
	resp, err := c.doRequest("DELETE", "/api/v1/providers/"+id, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("failed to delete provider: status %d", resp.StatusCode)
	}

	return nil
}

// GetVerificationResults retrieves verification results
func (c *Client) GetVerificationResults() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/verification-results", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode verification results response: %w", err)
	}

	return results, nil
}

// GetVerificationResult retrieves a specific verification result by ID
func (c *Client) GetVerificationResult(id string) (map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/verification-results/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode verification result response: %w", err)
	}

	return result, nil
}

// GetPricing retrieves pricing information
func (c *Client) GetPricing() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/pricing", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response struct {
		Pricing []map[string]any `json:"pricing"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode pricing response: %w", err)
	}

	return response.Pricing, nil
}

// GetLimits retrieves rate limit information
func (c *Client) GetLimits() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/limits", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var limits []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&limits); err != nil {
		return nil, fmt.Errorf("failed to decode limits response: %w", err)
	}

	return limits, nil
}

// GetIssues retrieves issue reports
func (c *Client) GetIssues() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/issues", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var issues []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("failed to decode issues response: %w", err)
	}

	return issues, nil
}

// GetEvents retrieves system events
func (c *Client) GetEvents() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/events", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode events response: %w", err)
	}

	return events, nil
}

// GetSchedules retrieves verification schedules
func (c *Client) GetSchedules() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/schedules", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var schedules []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
		return nil, fmt.Errorf("failed to decode schedules response: %w", err)
	}

	return schedules, nil
}

// GetConfigExports retrieves configuration exports
func (c *Client) GetConfigExports() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/exports", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var exports []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&exports); err != nil {
		return nil, fmt.Errorf("failed to decode config exports response: %w", err)
	}

	return exports, nil
}

// DownloadConfigExport downloads a configuration export
func (c *Client) DownloadConfigExport(id string) ([]byte, error) {
	resp, err := c.doRequest("GET", "/api/v1/exports/download/"+id, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download response: %w", err)
	}

	return data, nil
}

// GetLogs retrieves system logs
func (c *Client) GetLogs() ([]map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/logs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var logs []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("failed to decode logs response: %w", err)
	}

	return logs, nil
}

// GetConfig retrieves system configuration
func (c *Client) GetConfig() (map[string]any, error) {
	resp, err := c.doRequest("GET", "/api/v1/config", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var config map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config response: %w", err)
	}

	return config, nil
}

// ExportConfig exports configuration in specified format
func (c *Client) ExportConfig(format string) (map[string]any, error) {
	reqBody := map[string]string{"format": format}
	resp, err := c.doRequest("POST", "/api/v1/config/export", reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode export config response: %w", err)
	}

	return result, nil
}
