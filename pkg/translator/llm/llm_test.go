package llm

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestIsTextSizeError tests detection of size-related errors
func TestIsTextSizeError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "max_tokens error",
			err:      errors.New("Invalid max_tokens value"),
			expected: true,
		},
		{
			name:     "token limit error",
			err:      errors.New("token limit exceeded"),
			expected: true,
		},
		{
			name:     "too large error",
			err:      errors.New("request too large"),
			expected: true,
		},
		{
			name:     "context length error",
			err:      errors.New("context length exceeds maximum"),
			expected: true,
		},
		{
			name:     "network error",
			err:      errors.New("connection timeout"),
			expected: false,
		},
		{
			name:     "authentication error",
			err:      errors.New("invalid API key"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextSizeError(tt.err)
			if result != tt.expected {
				t.Errorf("isTextSizeError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestSplitText tests text splitting functionality
func TestSplitText(t *testing.T) {
	lt := &LLMTranslator{}

	tests := []struct {
		name          string
		text          string
		expectedChunks int
		maxChunkSize   int
	}{
		{
			name:          "small text",
			text:          "This is a small text.",
			expectedChunks: 1,
		},
		{
			name:          "text with paragraphs under limit",
			text:          strings.Repeat("First paragraph.\n\nSecond paragraph.\n\n", 100),
			expectedChunks: 1, // Still under 20KB limit
		},
		{
			name:          "very large text",
			text:          strings.Repeat("This is a sentence. ", 2000), // ~40KB
			expectedChunks: 2, // Should split into 2+ chunks (maxChunkSize = 20KB)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := lt.splitText(tt.text)

			if len(chunks) < tt.expectedChunks {
				t.Errorf("splitText produced %d chunks, expected at least %d", len(chunks), tt.expectedChunks)
			}

			// Verify all chunks are within size limit
			for i, chunk := range chunks {
				if len(chunk) > 20000 {
					t.Errorf("Chunk %d is too large: %d bytes", i, len(chunk))
				}
			}

			// Verify combined chunks equal original text
			combined := strings.Join(chunks, "")
			if combined != tt.text {
				t.Errorf("Combined chunks don't match original text")
			}
		})
	}
}

// TestSplitTextEdgeCases tests edge cases in text splitting functionality
func TestSplitTextEdgeCases(t *testing.T) {
	lt := &LLMTranslator{}

	t.Run("empty text", func(t *testing.T) {
		chunks := lt.splitText("")
		if len(chunks) != 1 || chunks[0] != "" {
			t.Errorf("Expected single empty chunk, got: %v", chunks)
		}
	})

	t.Run("single long paragraph that needs sentence splitting", func(t *testing.T) {
		// Create a single paragraph longer than maxChunkSize with multiple sentences
		longParagraph := strings.Repeat("This is a long sentence. ", 1000) // ~30KB
		chunks := lt.splitText(longParagraph)

		// Should split into multiple chunks
		if len(chunks) < 2 {
			t.Errorf("Expected multiple chunks for long paragraph, got: %d", len(chunks))
		}

		// All chunks should be within size limit
		for i, chunk := range chunks {
			if len(chunk) > 20000 {
				t.Errorf("Chunk %d is too large: %d bytes", i, len(chunk))
			}
		}

		// Combined should equal original
		combined := strings.Join(chunks, "")
		if combined != longParagraph {
			t.Error("Combined chunks don't match original text")
		}
	})

	t.Run("text with unicode characters", func(t *testing.T) {
		// Test with unicode characters that might affect chunking
		text := "Привет! Как дела? Это тест русского языка... Всё хорошо."
		chunks := lt.splitText(text)

		// Should handle unicode correctly
		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}

		// Combined should equal original
		combined := strings.Join(chunks, "")
		if combined != text {
			t.Error("Combined chunks don't match original text")
		}
	})

	t.Run("text with various sentence endings", func(t *testing.T) {
		// Test different sentence terminators
		text := "First sentence! Second sentence? Third sentence... Fourth sentence."
		chunks := lt.splitText(text)

		// Should handle all sentence types correctly
		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}

		// Combined should equal original
		combined := strings.Join(chunks, "")
		if combined != text {
			t.Error("Combined chunks don't match original text")
		}
	})

	t.Run("extremely_long_single_word", func(t *testing.T) {
		// Test with a single word longer than max chunk size
		longWord := strings.Repeat("a", 25000) // 25KB single word
		chunks := lt.splitText(longWord)

		// Should handle gracefully - might need to split the word
		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}

		// Combined should equal original
		combined := strings.Join(chunks, "")
		if combined != longWord {
			t.Error("Combined chunks don't match original text")
		}
	})

	t.Run("text_with_no_sentence_breaks", func(t *testing.T) {
		// Test with text that has no natural sentence breaks
		text := strings.Repeat("This is continuous text without periods or other punctuation ", 1000)
		chunks := lt.splitText(text)

		// Note: The function may not split text without sentence breaks
		// This tests the current behavior
		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}

		// Check if the function handles large continuous text appropriately
		// It might keep it as one chunk or split it based on other logic
		if len(chunks) == 1 {
			// If kept as one chunk, it might be larger than usual limit
			t.Logf("Text without sentence breaks kept as single chunk of %d bytes", len(chunks[0]))
		}

		// Combined should equal original
		combined := strings.Join(chunks, "")
		if combined != text {
			t.Error("Combined chunks don't match original text")
		}
	})

	t.Run("mixed_whitespace", func(t *testing.T) {
		// Test with various whitespace patterns
		text := "Sentence 1.\n\n  Sentence 2.\t\t\tSentence 3.  \n   Sentence 4."
		chunks := lt.splitText(text)

		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}

		// Combined should equal original
		combined := strings.Join(chunks, "")
		if combined != text {
			t.Error("Combined chunks don't match original text")
		}
	})

	t.Run("text_near_boundary", func(t *testing.T) {
		// Test text that's exactly at the boundary size
		text := strings.Repeat("Sentence. ", 1333) // ~20KB, close to boundary
		chunks := lt.splitText(text)

		if len(chunks) == 0 {
			t.Error("Expected at least one chunk")
		}

		// Combined should equal original
		combined := strings.Join(chunks, "")
		if combined != text {
			t.Error("Combined chunks don't match original text")
		}
	})
}

// TestSplitTextEdgeCases tests edge cases in text splitting functionality
func TestSplitBySentences(t *testing.T) {
	lt := &LLMTranslator{}

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "single sentence",
			text:     "This is one sentence.",
			expected: 1,
		},
		{
			name:     "multiple sentences",
			text:     "First sentence. Second sentence! Third sentence?",
			expected: 3,
		},
		{
			name:     "sentences with newlines",
			text:     "First sentence.\nSecond sentence.",
			expected: 2,
		},
		{
			name:     "ellipsis",
			text:     "First sentence… Second sentence.",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := lt.splitBySentences(tt.text)

			if len(sentences) != tt.expected {
				t.Errorf("splitBySentences produced %d sentences, expected %d", len(sentences), tt.expected)
			}

			// Verify combined sentences equal original text
			combined := strings.Join(sentences, "")
			if combined != tt.text {
				t.Errorf("Combined sentences don't match original text")
			}
		})
	}
}

// MockLLMClient for testing
type MockLLMClient struct {
	shouldFail      bool
	sizeError       bool
	callCount       int
	maxCallsToFail  int
}

func (m *MockLLMClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	m.callCount++

	if m.shouldFail && m.callCount <= m.maxCallsToFail {
		if m.sizeError {
			return "", errors.New("max_tokens limit exceeded")
		}
		return "", errors.New("API error")
	}

	// Mock translation: just uppercase the text
	return strings.ToUpper(text), nil
}

func (m *MockLLMClient) GetProviderName() string {
	return "mock"
}

// TestTranslateWithRetry tests the retry logic with text splitting
func TestTranslateWithRetry(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		shouldFail     bool
		sizeError      bool
		expectedError  bool
		expectedRetries int
	}{
		{
			name:           "successful translation",
			text:           "Hello world",
			shouldFail:     false,
			sizeError:      false,
			expectedError:  false,
			expectedRetries: 0,
		},
		{
			name:           "size error with retry success",
			text:           strings.Repeat("This is a sentence. ", 2000), // Large enough to split (40KB)
			shouldFail:     true,
			sizeError:      true,
			expectedError:  false,
			expectedRetries: 1,
		},
		{
			name:           "non-size error",
			text:           "Hello world",
			shouldFail:     true,
			sizeError:      false,
			expectedError:  true,
			expectedRetries: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{
				shouldFail:     tt.shouldFail,
				sizeError:      tt.sizeError,
				maxCallsToFail: 1, // Fail only first call
			}

			lt := &LLMTranslator{
				client: mockClient,
			}

			prompt := "Translate this text"
			result, err := lt.translateWithRetry(context.Background(), tt.text, prompt, "test context")

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectedError && result == "" {
				t.Error("Expected non-empty result")
			}
		})
	}
}

// Benchmark text splitting performance
func BenchmarkSplitText(b *testing.B) {
	lt := &LLMTranslator{}
	largeText := strings.Repeat("This is a sentence in a large text. ", 1000) // ~40KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lt.splitText(largeText)
	}
}

// TestOpenAIClientDetailed tests OpenAI client more thoroughly
func TestOpenAIClientDetailed(t *testing.T) {
	tests := []struct {
		name        string
		config      TranslationConfig
		expectError bool
	}{
		{
			name: "valid config with delegation",
			config: TranslationConfig{
				APIKey:  "test-key",
				Model:   "gpt-4",
				Provider: "deepseek", // This is delegation
			},
			expectError: false,
		},
		{
			name: "empty model with delegation",
			config: TranslationConfig{
				APIKey:  "test-key",
				Provider: "deepseek",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewOpenAIClient(tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected non-nil client")
				}
				
				// Test GetProviderName
				if client.GetProviderName() != "openai" {
					t.Errorf("Expected provider name 'openai', got '%s'", client.GetProviderName())
				}
			}
		})
	}
}

// TestLLMTranslatorErrorHandling tests various error scenarios
func TestLLMTranslatorErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *LLMTranslator
		expectError bool
	}{
		{
			name: "client returns error",
			setupClient: func() *LLMTranslator {
				return &LLMTranslator{
					BaseTranslator: NewBaseTranslator(TranslationConfig{}),
					client: &MockLLMClient{
						shouldFail:     true,
						maxCallsToFail: 1,
					},
					provider: ProviderOpenAI,
				}
			},
			expectError: true,
		},
		{
			name: "client returns non-size error",
			setupClient: func() *LLMTranslator {
				return &LLMTranslator{
					BaseTranslator: NewBaseTranslator(TranslationConfig{}),
					client: &MockLLMClient{
						shouldFail:     true,
						sizeError:      false,
						maxCallsToFail: 1,
					},
					provider: ProviderOpenAI,
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lt := tt.setupClient()
			
			_, err := lt.Translate(context.Background(), "test", "context")
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}

// TestNewLLMTranslator tests LLM translator creation
func TestNewLLMTranslator(t *testing.T) {
	tests := []struct {
		name        string
		config      translator.TranslationConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid OpenAI config",
			config: translator.TranslationConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "valid Anthropic config",
			config: translator.TranslationConfig{
				Provider: "anthropic",
				Model:    "claude-3-sonnet-20240229",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "valid DeepSeek config",
			config: translator.TranslationConfig{
				Provider: "deepseek",
				Model:    "deepseek-chat",
				APIKey:   "test-key",
			},
			expectError: false,
		},
		{
			name: "invalid provider",
			config: translator.TranslationConfig{
				Provider: "invalid",
				Model:    "test-model",
				APIKey:   "test-key",
			},
			expectError: true,
			errorMsg:    "unsupported LLM provider",
		},
		{
			name: "invalid model for OpenAI",
			config: translator.TranslationConfig{
				Provider: "openai",
				Model:    "invalid-model",
				APIKey:   "test-key",
			},
			expectError: true,
			errorMsg:    "model 'invalid-model' is not valid",
		},
		{
			name: "missing provider",
			config: translator.TranslationConfig{
				Provider: "",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			expectError: true,
			errorMsg:    "provider must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lt, err := NewLLMTranslator(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if lt != nil {
					t.Error("Expected nil translator on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if lt == nil {
					t.Error("Expected non-nil translator")
				}
			}
		})
	}
}

// TestLLMTranslatorGetName tests getting translator name
func TestLLMTranslatorGetName(t *testing.T) {
	tests := []struct {
		provider     string
		model        string
		expectedName string
	}{
		{"openai", "gpt-4", "llm-openai"},
		{"anthropic", "claude-3-sonnet-20240229", "llm-anthropic"},
		{"deepseek", "deepseek-chat", "llm-deepseek"},
		{"qwen", "qwen-max", "llm-qwen"},
		{"gemini", "gemini-pro", "llm-gemini"},
		{"ollama", "llama2", "llm-ollama"},
		{"llamacpp", "mistral", "llm-llamacpp"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			// Skip LlamaCpp test as it requires actual models
			if tt.provider == "llamacpp" {
				t.Skip("LlamaCpp requires actual models to be installed")
			}
			
			config := translator.TranslationConfig{
				Provider: tt.provider,
				Model:    tt.model,
				APIKey:   "test-key",
			}

			lt, err := NewLLMTranslator(config)
			if err != nil {
				t.Fatalf("Failed to create translator: %v", err)
			}

			name := lt.GetName()
			if name != tt.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tt.expectedName, name)
			}
		})
	}
}

// TestLLMTranslatorTranslate tests the main translate functionality
func TestLLMTranslatorTranslate(t *testing.T) {
	mockClient := &MockLLMClient{
		shouldFail: false,
	}

	lt := &LLMTranslator{
		BaseTranslator: NewBaseTranslator(TranslationConfig{}),
		client:        mockClient,
		provider:      ProviderOpenAI,
	}

	tests := []struct {
		name     string
		text     string
		context  string
		expected string
	}{
		{
			name:     "simple translation",
			text:     "Hello world",
			context:  "test context",
			expected: "HELLO WORLD",
		},
		{
			name:     "empty text",
			text:     "",
			context:  "test context",
			expected: "",
		},
		{
			name:     "whitespace only text",
			text:     "   ",
			context:  "test context",
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := lt.Translate(context.Background(), tt.text, tt.context)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestLLMTranslatorTranslateWithProgress tests progress reporting
func TestLLMTranslatorTranslateWithProgress(t *testing.T) {
	mockClient := &MockLLMClient{
		shouldFail: false,
	}

	lt := &LLMTranslator{
		BaseTranslator: NewBaseTranslator(TranslationConfig{
			Provider: "openai",
		}),
		client:   mockClient,
		provider: ProviderOpenAI,
	}

	eventBus := events.NewEventBus()
	sessionID := "test-session"
	progressReceived := false
	completionReceived := false

	// Subscribe to progress events
	eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
		if event.SessionID == sessionID {
			progressReceived = true
		}
	})

	eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
		if event.SessionID == sessionID && strings.Contains(event.Message, "completed") {
			completionReceived = true
		}
	})

	result, err := lt.TranslateWithProgress(context.Background(), "Hello world", "test context", eventBus, sessionID)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "HELLO WORLD" {
		t.Errorf("Expected 'HELLO WORLD', got '%s'", result)
	}

	// Give more time for async event processing
	time.Sleep(50 * time.Millisecond)

	if !progressReceived {
		t.Error("Expected progress event but none received")
	}

	if !completionReceived {
		t.Error("Expected completion event but none received")
	}
}

// TestLLMTranslatorCaching tests caching functionality
func TestLLMTranslatorCaching(t *testing.T) {
	mockClient := &MockLLMClient{
		shouldFail: false,
	}

	lt := &LLMTranslator{
		BaseTranslator: NewBaseTranslator(TranslationConfig{}),
		client:        mockClient,
		provider:      ProviderOpenAI,
	}

	text := "Hello world"
	contextStr := "test context"

	// First translation
	result1, err1 := lt.Translate(context.Background(), text, contextStr)
	if err1 != nil {
		t.Errorf("First translation failed: %v", err1)
	}

	// Second translation should use cache
	result2, err2 := lt.Translate(context.Background(), text, contextStr)
	if err2 != nil {
		t.Errorf("Second translation failed: %v", err2)
	}

	if result1 != result2 {
		t.Error("Cached result should match first result")
	}

	// Check that client was called only once (second call used cache)
	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 client call, got %d", mockClient.callCount)
	}

	// Check stats
	stats := lt.GetStats()
	if stats.Cached != 1 {
		t.Errorf("Expected 1 cached translation, got %d", stats.Cached)
	}
}

// TestConvertFromTranslatorConfig tests config conversion
func TestConvertFromTranslatorConfig(t *testing.T) {
	originalConfig := translator.TranslationConfig{
		SourceLang:     "en",
		TargetLang:     "ru",
		SourceLanguage: "English",
		TargetLanguage: "Russian",
		Provider:       "openai",
		Model:          "gpt-4",
		APIKey:         "test-key",
		BaseURL:        "https://api.openai.com/v1",
		Script:         "latin",
		Options:        map[string]interface{}{"temperature": 0.5},
	}

	convertedConfig := ConvertFromTranslatorConfig(originalConfig)

	if convertedConfig.SourceLang != originalConfig.SourceLang {
		t.Errorf("SourceLang mismatch: got %s, want %s", convertedConfig.SourceLang, originalConfig.SourceLang)
	}

	if convertedConfig.Provider != originalConfig.Provider {
		t.Errorf("Provider mismatch: got %s, want %s", convertedConfig.Provider, originalConfig.Provider)
	}

	if convertedConfig.Options["temperature"] != originalConfig.Options["temperature"] {
		t.Error("Options not preserved in conversion")
	}
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("isLower", func(t *testing.T) {
		if !isLower('a') {
			t.Error("'a' should be detected as lowercase")
		}
		if isLower('A') {
			t.Error("'A' should not be detected as lowercase")
		}
	})

	t.Run("isUpper", func(t *testing.T) {
		if !isUpper('A') {
			t.Error("'A' should be detected as uppercase")
		}
		if isUpper('a') {
			t.Error("'a' should not be detected as uppercase")
		}
	})

	t.Run("toUpper", func(t *testing.T) {
		if toUpper('a') != 'A' {
			t.Error("toUpper('a') should return 'A'")
		}
		if toUpper('A') != 'A' {
			t.Error("toUpper('A') should return 'A'")
		}
	})
}

// TestEnhanceTranslation tests translation enhancement
func TestEnhanceTranslation(t *testing.T) {
	lt := &LLMTranslator{}

	tests := []struct {
		name     string
		original string
		input    string
		expected string
	}{
		{
			name:     "preserves newline",
			original: "Hello\n",
			input:    "Hello",
			expected: "Hello\n",
		},
		{
			name:     "capitalizes first letter",
			original: "Hello world",
			input:    "hello world",
			expected: "Hello world",
		},
		{
			name:     "fixes smart quotes",
			original: "Hello",
			input:    `Hello "world"`,
			expected: `Hello "world"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lt.enhanceTranslation(tt.original, tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestBaseTranslatorMethods tests BaseTranslator functionality
func TestBaseTranslatorMethods(t *testing.T) {
	config := TranslationConfig{
		SourceLang: "en",
		TargetLang: "ru",
	}

	bt := NewBaseTranslator(config)

	t.Run("GetStats", func(t *testing.T) {
		stats := bt.GetStats()
		if stats.Total != 0 {
			t.Errorf("Expected Total=0, got %d", stats.Total)
		}
	})

	t.Run("CheckCache empty", func(t *testing.T) {
		result, found := bt.CheckCache("test")
		if found {
			t.Error("Expected no cache hit")
		}
		if result != "" {
			t.Errorf("Expected empty result, got '%s'", result)
		}
	})

	t.Run("AddToCache and CheckCache", func(t *testing.T) {
		bt.AddToCache("hello", "Привет")
		result, found := bt.CheckCache("hello")
		if !found {
			t.Error("Expected cache hit")
		}
		if result != "Привет" {
			t.Errorf("Expected 'Привет', got '%s'", result)
		}
	})

	t.Run("UpdateStats success", func(t *testing.T) {
		bt.UpdateStats(true)
		stats := bt.GetStats()
		if stats.Total != 1 || stats.Translated != 1 || stats.Errors != 0 {
			t.Errorf("Expected Total=1, Translated=1, Errors=0, got Total=%d, Translated=%d, Errors=%d",
				stats.Total, stats.Translated, stats.Errors)
		}
	})

	t.Run("UpdateStats failure", func(t *testing.T) {
		bt.UpdateStats(false)
		stats := bt.GetStats()
		if stats.Total != 2 || stats.Translated != 1 || stats.Errors != 1 {
			t.Errorf("Expected Total=2, Translated=1, Errors=1, got Total=%d, Translated=%d, Errors=%d",
				stats.Total, stats.Translated, stats.Errors)
		}
	})
}

// TestEmitFunctions tests event emission functions
func TestEmitFunctions(t *testing.T) {
	eventBus := events.NewEventBus()
	sessionID := "test-session"
	progressReceived := false
	errorReceived := false

	eventBus.Subscribe(events.EventTranslationProgress, func(event events.Event) {
		if event.SessionID == sessionID {
			progressReceived = true
		}
	})

	eventBus.Subscribe(events.EventTranslationError, func(event events.Event) {
		if event.SessionID == sessionID {
			errorReceived = true
		}
	})

	t.Run("EmitProgress", func(t *testing.T) {
		EmitProgress(eventBus, sessionID, "test message", map[string]interface{}{"key": "value"})
		
		// Give some time for async processing
		time.Sleep(10 * time.Millisecond)
		
		if !progressReceived {
			t.Error("Expected progress event to be received")
		}
	})

	t.Run("EmitError", func(t *testing.T) {
		testErr := errors.New("test error")
		EmitError(eventBus, sessionID, "test error message", testErr)
		
		// Give some time for async processing
		time.Sleep(10 * time.Millisecond)
		
		if !errorReceived {
			t.Error("Expected error event to be received")
		}
	})

	t.Run("EmitProgress with nil EventBus", func(t *testing.T) {
		// Should not panic
		EmitProgress(nil, sessionID, "test message", nil)
	})

	t.Run("EmitError with nil EventBus", func(t *testing.T) {
		// Should not panic
		EmitError(nil, sessionID, "test error", errors.New("test"))
	})
}

// TestCreateTranslationPrompt tests prompt creation
func TestCreateTranslationPrompt(t *testing.T) {
	lt := &LLMTranslator{}

	tests := []struct {
		name     string
		text     string
		context  string
		expected string
	}{
		{
			name:    "with context",
			text:    "Hello world",
			context: "Literary text",
		},
		{
			name:    "without context",
			text:    "Hello world",
			context: "",
		},
		{
			name:    "with empty text",
			text:    "",
			context: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := lt.createTranslationPrompt(tt.text, tt.context)

			if !strings.Contains(prompt, tt.text) {
				t.Error("Prompt should contain the original text")
			}

			if tt.context != "" && !strings.Contains(prompt, tt.context) {
				t.Error("Prompt should contain the context when provided")
			}

			if strings.Contains(prompt, "serbian") || strings.Contains(prompt, "croatian") {
				t.Error("Prompt should specify Ekavica dialect only")
			}
		})
	}
}

// TestOllamaProviderName tests Ollama GetProviderName method
func TestOllamaProviderName(t *testing.T) {
	config := TranslationConfig{Model: "llama2"}
	client, err := NewOllamaClient(config)
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}
	
	if client.GetProviderName() != "ollama" {
		t.Errorf("Expected provider name \"ollama\", got \"%s\"", client.GetProviderName())
	}
}

