package providers

import (
	"time"
)

// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	Name            string                 `json:"name"`
	Endpoint        string                 `json:"endpoint"`
	AuthType        string                 `json:"auth_type"`        // "bearer", "api_key", "oauth"
	StreamingFormat string                 `json:"streaming_format"` // "sse", "websocket", "json"
	DefaultModel    string                 `json:"default_model"`
	RateLimits      RateLimitConfig        `json:"rate_limits"`
	Timeouts        TimeoutConfig          `json:"timeouts"`
	RetryConfig     RetryConfig            `json:"retry_config"`
	Features        map[string]interface{} `json:"features"`
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	RequestsPerHour   int `json:"requests_per_hour"`
	BurstLimit        int `json:"burst_limit"`
}

// TimeoutConfig defines timeout settings
type TimeoutConfig struct {
	RequestTimeout time.Duration `json:"request_timeout"`
	StreamTimeout  time.Duration `json:"stream_timeout"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries      int           `json:"max_retries"`
	InitialDelay    time.Duration `json:"initial_delay"`
	MaxDelay        time.Duration `json:"max_delay"`
	BackoffFactor   float64       `json:"backoff_factor"`
	RetryableErrors []string      `json:"retryable_errors"`
}

// ProviderRegistry manages provider configurations
type ProviderRegistry struct {
	providers map[string]*ProviderConfig
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	pr := &ProviderRegistry{
		providers: make(map[string]*ProviderConfig),
	}
	pr.registerDefaultProviders()
	return pr
}

// GetConfig returns configuration for a provider
func (pr *ProviderRegistry) GetConfig(providerName string) (*ProviderConfig, bool) {
	config, exists := pr.providers[providerName]
	return config, exists
}

// RegisterProvider registers a custom provider configuration
func (pr *ProviderRegistry) RegisterProvider(config *ProviderConfig) {
	pr.providers[config.Name] = config
}

// registerDefaultProviders registers built-in provider configurations
func (pr *ProviderRegistry) registerDefaultProviders() {
	// OpenAI configuration
	pr.providers["openai"] = &ProviderConfig{
		Name:            "openai",
		Endpoint:        "https://api.openai.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "gpt-4",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": true,
			"supports_vision":    true,
			"supports_acp":       true,
			"max_context_length": 128000,
			"supported_models":   []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
		},
	}

	// DeepSeek configuration
	pr.providers["deepseek"] = &ProviderConfig{
		Name:            "deepseek",
		Endpoint:        "https://api.deepseek.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "deepseek-chat",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 32768,
			"supported_models":   []string{"deepseek-chat", "deepseek-coder"},
		},
	}

	// Anthropic configuration
	pr.providers["anthropic"] = &ProviderConfig{
		Name:            "anthropic",
		Endpoint:        "https://api.anthropic.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "claude-3-opus-20240229",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 50,
			RequestsPerHour:   1000,
			BurstLimit:        8,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 120 * time.Second, // Claude can be slower
			StreamTimeout:  600 * time.Second,
			ConnectTimeout: 15 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    2 * time.Second,
			MaxDelay:        60 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504", "529"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    true,
			"supports_acp":       true,
			"max_context_length": 200000,
			"supported_models":   []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
		},
	}

	// Gemini (Google AI) configuration
	pr.providers["gemini"] = &ProviderConfig{
		Name:            "gemini",
		Endpoint:        "https://generativelanguage.googleapis.com/v1beta",
		AuthType:        "api_key",
		StreamingFormat: "sse",
		DefaultModel:    "gemini-2.5-flash",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": true,
			"supports_vision":    true,
			"supports_acp":       true,
			"max_context_length": 1048576,
			"supported_models": []string{
				"gemini-3.1-pro-preview",
				"gemini-3-pro-preview",
				"gemini-3-flash-preview",
				"gemini-2.5-pro",
				"gemini-2.5-flash",
				"gemini-2.5-flash-lite",
				"gemini-2.0-flash",
			},
		},
	}

	// Groq configuration (NEW: High-performance inference, free tier)
	pr.providers["groq"] = &ProviderConfig{
		Name:            "groq",
		Endpoint:        "https://api.groq.com/openai/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "llama3-8b-8192",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 8192,
			"supported_models": []string{
				"llama3-8b-8192",
				"llama3-70b-8192",
				"mixtral-8x7b-32768",
				"gemma-7b-it",
				"gemma2-9b-it",
			},
		},
	}

	// Together AI configuration (NEW: Free trial, 50+ models)
	pr.providers["togetherai"] = &ProviderConfig{
		Name:            "togetherai",
		Endpoint:        "https://api.together.xyz/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "meta-llama/Llama-3-8b-chat-hf",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 4096,
			"supported_models": []string{
				"meta-llama/Llama-3-8b-chat-hf",
				"meta-llama/Llama-3-70b-chat-hf",
				"codellama/CodeLlama-34b-Instruct-hf",
				"Qwen/Qwen1.5-72B-Chat",
				"microsoft/WizardLM-2-8x22B",
			},
		},
	}
	// Qwen/Alibaba Cloud DashScope configuration
	pr.providers["qwen"] = &ProviderConfig{
		Name:            "qwen",
		Endpoint:        "https://dashscope.aliyuncs.com/api/v1",
		AuthType:        "bearer", // Supports both API key and OAuth
		StreamingFormat: "sse",
		DefaultModel:    "qwen-turbo",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": true,
			"supports_vision":    false,
			"supports_acp":       true,
			"supports_oauth":     true,
			"max_context_length": 32768,
			"supported_models": []string{
				"qwen-turbo",
				"qwen-plus",
				"qwen-max",
				"qwen-max-longcontext",
				"qwen-coder-turbo",
			},
		},
	}

	pr.providers["qwen-code"] = &ProviderConfig{
		Name:            "qwen-code",
		Endpoint:        "https://dashscope.aliyuncs.com/api/v1",
		AuthType:        "oauth",
		StreamingFormat: "sse",
		DefaultModel:    "coder-model",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 180 * time.Second,
			StreamTimeout:  600 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": true,
			"supports_vision":    true,
			"supports_acp":       true,
			"supports_oauth":     true,
			"supports_thinking":  true,
			"max_context_length": 1048576,
			"supported_models": []string{
				"coder-model",
				"vision-model",
			},
			"cli_proxy":          true,
			"oauth_storage_path": ".qwen/oauth_creds.json",
		},
	}

	// OpenCode Zen configuration (FREE: Big Pickle, Grok Code Fast, GLM 4.7, GPT 5 Nano)
	pr.providers["zen"] = &ProviderConfig{
		Name:            "zen",
		Endpoint:        "https://opencode.ai/zen/v1/chat/completions",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "opencode/grok-code",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": true,
			"supports_vision":    false,
			"supports_acp":       true,
			"free_tier":          true,
			"max_context_length": 128000,
			"supported_models": []string{
				"opencode/big-pickle",   // Big Pickle (stealth model)
				"opencode/grok-code",    // Grok Code Fast (xAI code model)
				"opencode/glm-4.7-free", // GLM 4.7 Free
				"opencode/gpt-5-nano",   // GPT 5 Nano free tier
			},
		},
	}

	// Public AI configuration (Swiss AI Apertus - free tier)
	pr.providers["publicai"] = &ProviderConfig{
		Name:            "publicai",
		Endpoint:        "https://api.publicai.co/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "swiss-ai/apertus-8b-instruct",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 20,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    2 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"free_tier":          true,
			"max_context_length": 65536,
			"max_output_tokens":  8192,
			"recommended_temp":   0.8,
			"recommended_top_p":  0.9,
			"supported_models": []string{
				"swiss-ai/apertus-8b-instruct",
			},
		},
	}

	// Kimi (Moonshot) configuration
	pr.providers["kimi"] = &ProviderConfig{
		Name:            "kimi",
		Endpoint:        "https://api.moonshot.cn/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "moonshot-v1-8k",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 131072,
			"supported_models": []string{
				"moonshot-v1-8k",
				"moonshot-v1-32k",
				"moonshot-v1-128k",
			},
		},
	}

	pr.providers["kimi-code"] = &ProviderConfig{
		Name:            "kimi-code",
		Endpoint:        "https://api.kimi.com/coding/v1",
		AuthType:        "oauth",
		StreamingFormat: "sse",
		DefaultModel:    "kimi-for-coding",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			BurstLimit:        10,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 180 * time.Second,
			StreamTimeout:  600 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    true,
			"supports_acp":       true,
			"supports_thinking":  true,
			"max_context_length": 262144,
			"supported_models": []string{
				"kimi-for-coding",
			},
			"cli_proxy":          true,
			"oauth_storage_path": ".kimi/credentials/kimi-code.json",
		},
	}

	// SambaNova configuration
	pr.providers["sambanova"] = &ProviderConfig{
		Name:            "sambanova",
		Endpoint:        "https://api.sambanova.ai/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "Meta-Llama-3.1-8B-Instruct",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 131072,
			"supported_models": []string{
				"Meta-Llama-3.1-8B-Instruct",
				"Meta-Llama-3.1-70B-Instruct",
				"Meta-Llama-3.2-1B-Instruct",
				"Meta-Llama-3.2-3B-Instruct",
			},
		},
	}

	// Upstage configuration
	pr.providers["upstage"] = &ProviderConfig{
		Name:            "upstage",
		Endpoint:        "https://api.upstage.ai/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "solar-1-mini-chat",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 32768,
			"supported_models": []string{
				"solar-1-mini-chat",
				"solar-1-mini-chat-ja",
				"solar-pro",
			},
		},
	}

	// Sarvam AI configuration
	pr.providers["sarvam"] = &ProviderConfig{
		Name:            "sarvam",
		Endpoint:        "https://api.sarvam.ai/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "sarvam-m",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 8192,
			"supported_models": []string{
				"sarvam-m",
				"sarvam-2b",
				"sarvam-7b",
			},
		},
	}

	// Zhipu AI configuration
	pr.providers["zhipu"] = &ProviderConfig{
		Name:            "zhipu",
		Endpoint:        "https://open.bigmodel.cn/api/paas/v4",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "glm-4-flash",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 128000,
			"supported_models": []string{
				"glm-4-flash",
				"glm-4",
				"glm-4-plus",
				"glm-4-air",
				"glm-4-airx",
				"glm-4-long",
			},
		},
	}

	// Hyperbolic configuration
	pr.providers["hyperbolic"] = &ProviderConfig{
		Name:            "hyperbolic",
		Endpoint:        "https://api.hyperbolic.xyz/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "meta-llama/Meta-Llama-3-8B-Instruct",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 8192,
			"supported_models": []string{
				"meta-llama/Meta-Llama-3-8B-Instruct",
				"meta-llama/Meta-Llama-3-70B-Instruct",
				"mistralai/Mistral-7B-Instruct-v0.2",
			},
		},
	}

	// SiliconFlow configuration
	pr.providers["siliconflow"] = &ProviderConfig{
		Name:            "siliconflow",
		Endpoint:        "https://api.siliconflow.cn/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "Qwen/Qwen2.5-7B-Instruct",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 64000,
			"supported_models": []string{
				"Qwen/Qwen2.5-7B-Instruct",
				"Qwen/Qwen2.5-72B-Instruct",
				"deepseek-ai/DeepSeek-V2.5",
			},
		},
	}

	// Novita configuration
	pr.providers["novita"] = &ProviderConfig{
		Name:            "novita",
		Endpoint:        "https://api.novita.ai/v3/openai",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "meta-llama/llama-3-8b-instruct",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 8192,
			"supported_models": []string{
				"meta-llama/llama-3-8b-instruct",
				"meta-llama/llama-3-70b-instruct",
				"mistralai/mistral-7b-instruct",
			},
		},
	}

	// Cloudflare Workers AI configuration
	pr.providers["cloudflare"] = &ProviderConfig{
		Name:            "cloudflare",
		Endpoint:        "https://api.cloudflare.com/client/v4/accounts",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "@cf/meta/llama-3.1-8b-instruct",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 8192,
			"supported_models": []string{
				"@cf/meta/llama-3.1-8b-instruct",
				"@cf/meta/llama-3.1-70b-instruct",
				"@cf/mistral/mistral-7b-instruct",
				"@cf/qwen/qwen1.5-14b-chat-awq",
			},
		},
	}

	// Kilo configuration
	pr.providers["kilo"] = &ProviderConfig{
		Name:            "kilo",
		Endpoint:        "https://api.kilocode.ai/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "kilocode-1.5",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 32768,
			"supported_models": []string{
				"kilocode-1.5",
				"kilocode-1",
			},
		},
	}

	// Modal configuration
	pr.providers["modal"] = &ProviderConfig{
		Name:            "modal",
		Endpoint:        "https://api.modal.com/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "llama-3.1-8b",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 131072,
			"supported_models": []string{
				"llama-3.1-8b",
				"llama-3.1-70b",
				"mistral-7b",
			},
		},
	}

	// Nia configuration
	pr.providers["nia"] = &ProviderConfig{
		Name:            "nia",
		Endpoint:        "https://api.nia.ai/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "nia-1.5",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 32768,
			"supported_models": []string{
				"nia-1.5",
				"nia-1",
			},
		},
	}

	// NLP Cloud configuration
	pr.providers["nlpcloud"] = &ProviderConfig{
		Name:            "nlpcloud",
		Endpoint:        "https://api.nlpcloud.io/v1/gpu",
		AuthType:        "api_key",
		StreamingFormat: "sse",
		DefaultModel:    "finetuned-llama-3-70b",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 8192,
			"supported_models": []string{
				"finetuned-llama-3-70b",
				"llama-3-70b",
				"mixtral-8x7b",
				"openchat-3-5",
			},
		},
	}

	// Vulavula configuration
	pr.providers["vulavula"] = &ProviderConfig{
		Name:            "vulavula",
		Endpoint:        "https://api.vulavula.ai/v1",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "vulavula-1.5",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   1000,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 60 * time.Second,
			StreamTimeout:  300 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      3,
			InitialDelay:    1 * time.Second,
			MaxDelay:        30 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 32768,
			"supported_models": []string{
				"vulavula-1.5",
				"vulavula-1",
			},
		},
	}

	// Generic configuration for unknown providers
	pr.providers["generic"] = &ProviderConfig{
		Name:            "generic",
		AuthType:        "bearer",
		StreamingFormat: "sse",
		DefaultModel:    "unknown",
		RateLimits: RateLimitConfig{
			RequestsPerMinute: 30,
			RequestsPerHour:   500,
			BurstLimit:        5,
		},
		Timeouts: TimeoutConfig{
			RequestTimeout: 30 * time.Second,
			StreamTimeout:  180 * time.Second,
			ConnectTimeout: 10 * time.Second,
		},
		RetryConfig: RetryConfig{
			MaxRetries:      2,
			InitialDelay:    1 * time.Second,
			MaxDelay:        15 * time.Second,
			BackoffFactor:   2.0,
			RetryableErrors: []string{"429", "500", "502", "503", "504"},
		},
		Features: map[string]interface{}{
			"supports_streaming": true,
			"supports_functions": false,
			"supports_vision":    false,
			"supports_acp":       true,
			"max_context_length": 4096,
			"supported_models":   []string{},
		},
	}
}

// GetProviderNames returns all registered provider names
func (pr *ProviderRegistry) GetProviderNames() []string {
	names := make([]string, 0, len(pr.providers))
	for name := range pr.providers {
		names = append(names, name)
	}
	return names
}

// IsProviderSupported checks if a provider is supported
func (pr *ProviderRegistry) IsProviderSupported(providerName string) bool {
	_, exists := pr.providers[providerName]
	return exists
}

// GetDefaultConfig returns a default configuration for unknown providers
func (pr *ProviderRegistry) GetDefaultConfig() *ProviderConfig {
	config := *pr.providers["generic"] // Copy the generic config
	return &config
}
