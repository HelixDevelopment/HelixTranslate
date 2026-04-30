package providers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ProviderError represents a provider-specific error
type ProviderError struct {
	Provider    string
	Type        ErrorType
	Code        string
	Message     string
	HTTPStatus  int
	Retryable   bool
	RetryAfter  time.Duration
	RawResponse []byte
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s API error [%s]: %s", e.Provider, e.Code, e.Message)
}

// ErrorType categorizes different types of errors
type ErrorType int

const (
	ErrorTypeNetwork ErrorType = iota
	ErrorTypeAuth
	ErrorTypeRateLimit
	ErrorTypeQuota
	ErrorTypeInvalidRequest
	ErrorTypeServer
	ErrorTypeTimeout
	ErrorTypeUnknown
)

// ErrorClassifier analyzes HTTP responses and classifies errors
type ErrorClassifier struct {
	provider string
}

// NewErrorClassifier creates a new error classifier for a provider
func NewErrorClassifier(provider string) *ErrorClassifier {
	return &ErrorClassifier{provider: provider}
}

// ClassifyError analyzes an HTTP response and returns a classified error
func (ec *ErrorClassifier) ClassifyError(resp *http.Response, body []byte) *ProviderError {
	if resp == nil {
		return &ProviderError{
			Provider:  ec.provider,
			Type:      ErrorTypeNetwork,
			Code:      "NETWORK_ERROR",
			Message:   "Network connection failed",
			Retryable: true,
		}
	}

	// Extract error details based on provider
	var errorCode, errorMessage string
	var retryable bool
	var retryAfter time.Duration

	switch strings.ToLower(ec.provider) {
	case "openai":
		errorCode, errorMessage, retryable = ec.classifyOpenAIError(resp.StatusCode, body)
	case "anthropic":
		errorCode, errorMessage, retryable = ec.classifyAnthropicError(resp.StatusCode, body)
	case "deepseek":
		errorCode, errorMessage, retryable = ec.classifyDeepSeekError(resp.StatusCode, body)
	case "gemini", "google":
		errorCode, errorMessage, retryable = ec.classifyGoogleError(resp.StatusCode, body)
	default:
		errorCode, errorMessage, retryable = ec.classifyGenericError(resp.StatusCode, body)
	}

	// Check for rate limit headers
	if resp.StatusCode == 429 {
		if retryAfterHeader := resp.Header.Get("Retry-After"); retryAfterHeader != "" {
			if seconds, err := time.ParseDuration(retryAfterHeader + "s"); err == nil {
				retryAfter = seconds
			}
		}
	}

	errorType := ec.getErrorType(resp.StatusCode, errorCode)

	return &ProviderError{
		Provider:    ec.provider,
		Type:        errorType,
		Code:        errorCode,
		Message:     errorMessage,
		HTTPStatus:  resp.StatusCode,
		Retryable:   retryable,
		RetryAfter:  retryAfter,
		RawResponse: body,
	}
}

// classifyOpenAIError classifies OpenAI-specific errors
func (ec *ErrorClassifier) classifyOpenAIError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", "Invalid request parameters", false
	case 401:
		return "AUTHENTICATION_FAILED", "Invalid API key", false
	case 403:
		return "PERMISSION_DENIED", "Access forbidden", false
	case 404:
		return "NOT_FOUND", "Resource not found", false
	case 429:
		return "RATE_LIMIT_EXCEEDED", "Rate limit exceeded", true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", "Internal server error", true
	default:
		return "UNKNOWN_ERROR", "Unknown error occurred", false
	}
}

// classifyAnthropicError classifies Anthropic-specific errors
func (ec *ErrorClassifier) classifyAnthropicError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", "Invalid request parameters", false
	case 401:
		return "AUTHENTICATION_FAILED", "Invalid API key", false
	case 403:
		return "PERMISSION_DENIED", "Access forbidden", false
	case 404:
		return "NOT_FOUND", "Resource not found", false
	case 429:
		return "RATE_LIMIT_EXCEEDED", "Rate limit exceeded", true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", "Internal server error", true
	case 529:
		return "OVERLOADED", "Service overloaded", true
	default:
		return "UNKNOWN_ERROR", "Unknown error occurred", false
	}
}

// classifyDeepSeekError classifies DeepSeek-specific errors
func (ec *ErrorClassifier) classifyDeepSeekError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", "Invalid request parameters", false
	case 401:
		return "AUTHENTICATION_FAILED", "Invalid API key", false
	case 403:
		return "PERMISSION_DENIED", "Access forbidden", false
	case 404:
		return "NOT_FOUND", "Resource not found", false
	case 429:
		return "RATE_LIMIT_EXCEEDED", "Rate limit exceeded", true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", "Internal server error", true
	default:
		return "UNKNOWN_ERROR", "Unknown error occurred", false
	}
}

// classifyGoogleError classifies Google AI-specific errors
func (ec *ErrorClassifier) classifyGoogleError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400:
		return "INVALID_REQUEST", "Invalid request parameters", false
	case 401:
		return "AUTHENTICATION_FAILED", "Invalid API key", false
	case 403:
		return "PERMISSION_DENIED", "Access forbidden", false
	case 404:
		return "NOT_FOUND", "Resource not found", false
	case 429:
		return "RATE_LIMIT_EXCEEDED", "Rate limit exceeded", true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", "Internal server error", true
	default:
		return "UNKNOWN_ERROR", "Unknown error occurred", false
	}
}

// classifyGenericError provides generic error classification
func (ec *ErrorClassifier) classifyGenericError(statusCode int, body []byte) (string, string, bool) {
	switch statusCode {
	case 400, 422:
		return "INVALID_REQUEST", "Invalid request", false
	case 401, 403:
		return "AUTHENTICATION_FAILED", "Authentication failed", false
	case 404:
		return "NOT_FOUND", "Resource not found", false
	case 408:
		return "TIMEOUT", "Request timeout", true
	case 429:
		return "RATE_LIMIT_EXCEEDED", "Rate limit exceeded", true
	case 500, 502, 503, 504:
		return "SERVER_ERROR", "Server error", true
	default:
		return "UNKNOWN_ERROR", "Unknown error", false
	}
}

// getErrorType maps HTTP status codes and error codes to error types
func (ec *ErrorClassifier) getErrorType(statusCode int, errorCode string) ErrorType {
	switch {
	case statusCode >= 500:
		return ErrorTypeServer
	case statusCode == 408 || strings.Contains(errorCode, "TIMEOUT"):
		return ErrorTypeTimeout
	case statusCode == 401 || statusCode == 403 || strings.Contains(errorCode, "AUTH"):
		return ErrorTypeAuth
	case statusCode == 429 || strings.Contains(errorCode, "RATE_LIMIT"):
		return ErrorTypeRateLimit
	case strings.Contains(errorCode, "QUOTA"):
		return ErrorTypeQuota
	case statusCode >= 400 && statusCode < 500:
		return ErrorTypeInvalidRequest
	case statusCode >= 200 && statusCode < 300:
		return ErrorTypeUnknown // Should not happen for errors
	default:
		return ErrorTypeUnknown
	}
}

// ErrorHandler provides error handling and retry logic
type ErrorHandler struct {
	classifier  *ErrorClassifier
	retryConfig RetryConfig
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(provider string, retryConfig RetryConfig) *ErrorHandler {
	return &ErrorHandler{
		classifier:  NewErrorClassifier(provider),
		retryConfig: retryConfig,
	}
}

// HandleError processes an error and determines retry behavior
func (eh *ErrorHandler) HandleError(resp *http.Response, body []byte, attempt int) (*ProviderError, bool, time.Duration) {
	providerError := eh.classifier.ClassifyError(resp, body)

	// Check if we should retry
	shouldRetry := providerError.Retryable &&
		attempt < eh.retryConfig.MaxRetries &&
		eh.isRetryableError(providerError)

	// Calculate retry delay
	var retryDelay time.Duration
	if shouldRetry {
		retryDelay = eh.calculateRetryDelay(attempt, providerError.RetryAfter)
	}

	return providerError, shouldRetry, retryDelay
}

// isRetryableError checks if an error is retryable based on configuration
func (eh *ErrorHandler) isRetryableError(err *ProviderError) bool {
	errorCode := err.Code

	for _, retryableCode := range eh.retryConfig.RetryableErrors {
		if strings.Contains(errorCode, retryableCode) {
			return true
		}
		// Also check HTTP status as string
		if strings.Contains(fmt.Sprintf("%d", err.HTTPStatus), retryableCode) {
			return true
		}
	}

	return false
}

// calculateRetryDelay calculates the delay before retrying
func (eh *ErrorHandler) calculateRetryDelay(attempt int, retryAfter time.Duration) time.Duration {
	// If server specified Retry-After, use it
	if retryAfter > 0 {
		return retryAfter
	}

	// Exponential backoff
	delay := time.Duration(float64(eh.retryConfig.InitialDelay) * pow(eh.retryConfig.BackoffFactor, attempt))

	// Cap at max delay
	if delay > eh.retryConfig.MaxDelay {
		delay = eh.retryConfig.MaxDelay
	}

	return delay
}

// pow calculates base^exponent for float64
func pow(base float64, exp int) float64 {
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

// RecoveryStrategies provides different recovery strategies
type RecoveryStrategies struct {
	handler *ErrorHandler
}

// NewRecoveryStrategies creates recovery strategies
func NewRecoveryStrategies(provider string, retryConfig RetryConfig) *RecoveryStrategies {
	return &RecoveryStrategies{
		handler: NewErrorHandler(provider, retryConfig),
	}
}

// ExecuteWithRetry executes a function with retry logic
func (rs *RecoveryStrategies) ExecuteWithRetry(fn func() (*http.Response, []byte, error)) (*http.Response, []byte, error) {
	var lastResp *http.Response
	var lastBody []byte
	var lastErr error

	for attempt := 0; attempt <= rs.handler.retryConfig.MaxRetries; attempt++ {
		resp, body, err := fn()

		if err != nil || resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, body, err
		}

		// Classify error and check if retryable
		providerError, shouldRetry, retryDelay := rs.handler.HandleError(resp, body, attempt)

		lastResp = resp
		lastBody = body
		lastErr = providerError

		if !shouldRetry {
			break
		}

		if retryDelay > 0 {
			time.Sleep(retryDelay)
		}
	}

	return lastResp, lastBody, lastErr
}

// CircuitBreakerRecovery provides circuit breaker-based recovery
func (rs *RecoveryStrategies) CircuitBreakerRecovery(cb CircuitBreaker, fn func() (*http.Response, []byte, error)) (*http.Response, []byte, error) {
	// Execute with circuit breaker protection
	err := cb.Call(func() error {
		resp, _, err := fn()
		if err != nil {
			return err
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Get the actual response
	return fn()
}

// FallbackRecovery provides fallback to alternative endpoints
func (rs *RecoveryStrategies) FallbackRecovery(endpoints []string, fn func(endpoint string) (*http.Response, []byte, error)) (*http.Response, []byte, error) {
	var lastErr error

	for _, endpoint := range endpoints {
		resp, body, err := fn(endpoint)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, body, nil
		}
		lastErr = err
	}

	return nil, nil, lastErr
}

// CircuitBreaker interface for circuit breaker functionality
type CircuitBreaker interface {
	Call(func() error) error
}
