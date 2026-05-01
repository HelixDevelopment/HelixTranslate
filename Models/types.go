// Package models provides shared data types for the Helix ecosystem.
package models

// LLMRequest represents a request to an LLM provider.
type LLMRequest struct {
	Model       string                 `json:"model"`
	Prompt      string                 `json:"prompt"`
	Messages    []Message              `json:"messages,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents a response from an LLM provider.
type LLMResponse struct {
	Text         string                 `json:"text"`
	Model        string                 `json:"model"`
	Usage        Usage                  `json:"usage,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProviderCapabilities describes what a provider can do.
type ProviderCapabilities struct {
	SupportedModels         []string          `json:"supported_models,omitempty"`
	SupportedFeatures       []string          `json:"supported_features,omitempty"`
	SupportedRequestTypes   []string          `json:"supported_request_types,omitempty"`
	SupportsStreaming       bool              `json:"supports_streaming"`
	SupportsFunctionCalling bool              `json:"supports_function_calling"`
	SupportsVision          bool              `json:"supports_vision"`
	SupportsTools           bool              `json:"supports_tools"`
	SupportsSearch          bool              `json:"supports_search"`
	SupportsReasoning       bool              `json:"supports_reasoning"`
	SupportsCodeCompletion  bool              `json:"supports_code_completion"`
	SupportsCodeAnalysis    bool              `json:"supports_code_analysis"`
	SupportsRefactoring     bool              `json:"supports_refactoring"`
	Limits                  ModelLimits       `json:"limits"`
	Metadata                map[string]string `json:"metadata,omitempty"`
}

// ModelLimits defines usage constraints.
type ModelLimits struct {
	MaxTokens             int `json:"max_tokens"`
	MaxInputLength        int `json:"max_input_length"`
	MaxOutputLength       int `json:"max_output_length"`
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
}
