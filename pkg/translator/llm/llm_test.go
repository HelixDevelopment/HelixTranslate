package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
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

// TestSplitBySentences tests sentence splitting
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
