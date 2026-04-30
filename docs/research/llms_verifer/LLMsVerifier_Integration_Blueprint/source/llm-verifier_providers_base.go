package providers

import (
	"net/http"
)

// BaseAdapter provides common functionality for all provider adapters
type BaseAdapter struct {
	client   *http.Client
	endpoint string
	apiKey   string
	headers  map[string]string
}

// SetClient sets the HTTP client
func (b *BaseAdapter) SetClient(client *http.Client) {
	b.client = client
}

// GetClient returns the HTTP client
func (b *BaseAdapter) GetClient() *http.Client {
	return b.client
}

// SetEndpoint sets the API endpoint
func (b *BaseAdapter) SetEndpoint(endpoint string) {
	b.endpoint = endpoint
}

// GetEndpoint returns the API endpoint
func (b *BaseAdapter) GetEndpoint() string {
	return b.endpoint
}

// SetAPIKey sets the API key
func (b *BaseAdapter) SetAPIKey(apiKey string) {
	b.apiKey = apiKey
}

// GetAPIKey returns the API key
func (b *BaseAdapter) GetAPIKey() string {
	return b.apiKey
}

// SetHeaders sets additional headers
func (b *BaseAdapter) SetHeaders(headers map[string]string) {
	b.headers = headers
}

// GetHeaders returns the headers
func (b *BaseAdapter) GetHeaders() map[string]string {
	return b.headers
}

// AddHeader adds a single header
func (b *BaseAdapter) AddHeader(key, value string) {
	if b.headers == nil {
		b.headers = make(map[string]string)
	}
	b.headers[key] = value
}
