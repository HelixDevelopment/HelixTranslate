package language

import (
	"context"
	"testing"
)

func TestSimpleLLMDetector_DetectLanguage(t *testing.T) {
	// Test with invalid API key (should fail gracefully)
	detector := NewSimpleLLMDetector("openai", "invalid-key")

	ctx := context.Background()

	// Test empty text
	_, err := detector.DetectLanguage(ctx, "")
	if err == nil {
		t.Error("Expected error for empty text")
	}

	// Test with text (will fail due to invalid API key, but should not panic)
	_, err = detector.DetectLanguage(ctx, "This is English text")
	if err == nil {
		t.Error("Expected error due to invalid API key")
	}
}

func TestFormatLanguageCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"en", "en"},
		{"EN", "en"},
		{"eng", "en"},
		{"english", "en"},
		{"", ""},
		{"xyz", "xy"},
	}

	for _, test := range tests {
		result := FormatLanguageCode(test.input)
		if result != test.expected {
			t.Errorf("FormatLanguageCode(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}
