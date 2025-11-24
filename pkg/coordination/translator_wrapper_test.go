package coordination

import (
	"context"
	"testing"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

func TestNewMultiLLMTranslatorWrapper(t *testing.T) {
	config := translator.TranslationConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}

	wrapper, err := NewMultiLLMTranslatorWrapper(config, events.NewEventBus(), "test-session")

	// The wrapper may succeed if there are OAuth tokens or other providers available
	// Just test that it returns something valid if no error
	if err == nil && wrapper == nil {
		t.Error("Expected non-nil wrapper when no error returned")
	}

	if err != nil && wrapper != nil {
		t.Error("Expected nil wrapper when error returned")
	}
}

func TestNewMultiLLMTranslatorWrapperWithConfig(t *testing.T) {
	config := translator.TranslationConfig{
		Provider: "openai",
		Model:    "gpt-4",
	}

	// Test with disabled local LLMs
	wrapper, err := NewMultiLLMTranslatorWrapperWithConfig(config, events.NewEventBus(), "test-session", true, false)

	// The wrapper may succeed if there are API keys available
	// Just test that it returns something valid if no error
	if err == nil && wrapper == nil {
		t.Error("Expected non-nil wrapper when no error returned")
	}

	if err != nil && wrapper != nil {
		t.Error("Expected nil wrapper when error returned")
	}
}

func TestMultiLLMTranslatorWrapper_Interface(t *testing.T) {
	config := translator.TranslationConfig{
		Provider: "test",
	}

	// Create a mock coordinator
	coordinator := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{
				ID:         "test-instance",
				Provider:   "test",
				Model:      "test-model",
				Priority:   1,
				Available:  true,
				Translator: &MockTranslator{},
			},
		},
	}

	wrapper := &MultiLLMTranslatorWrapper{
		Coordinator: coordinator,
		config:      config,
	}

	// Test that it implements the Translator interface
	var _ translator.Translator = wrapper

	// Test GetName
	name := wrapper.GetName()
	if name != "multi-llm-coordinator" {
		t.Errorf("Expected 'multi-llm-coordinator', got '%s'", name)
	}

	// Test GetStats
	stats := wrapper.GetStats()
	if stats.Total != 0 || stats.Translated != 0 || stats.Cached != 0 || stats.Errors != 0 {
		t.Errorf("Expected zero stats, got %+v", stats)
	}
}

// MockTranslator implements translator.Translator for testing
type MockTranslator struct{}

func (m *MockTranslator) Translate(ctx context.Context, text, contextHint string) (string, error) {
	return "translated text", nil
}

func (m *MockTranslator) TranslateWithProgress(ctx context.Context, text, contextHint string, eventBus *events.EventBus, sessionID string) (string, error) {
	return "translated text", nil
}

func (m *MockTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (m *MockTranslator) GetName() string {
	return "mock-translator"
}

func TestMultiLLMTranslatorWrapper_Translate(t *testing.T) {
	// Create a mock coordinator
	coordinator := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{
				ID:         "test-instance",
				Provider:   "test",
				Model:      "test-model",
				Priority:   1,
				Available:  true,
				Translator: &MockTranslator{},
			},
		},
	}

	wrapper := &MultiLLMTranslatorWrapper{
		Coordinator: coordinator,
		config:      translator.TranslationConfig{},
	}

	// Test basic translation
	result, err := wrapper.Translate(context.Background(), "test text", "test context")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "translated text" {
		t.Errorf("Expected 'translated text', got '%s'", result)
	}
}

func TestMultiLLMTranslatorWrapper_TranslateWithProgress(t *testing.T) {
	eventBus := events.NewEventBus()
	
	// Create a mock coordinator
	coordinator := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{
				ID:         "test-instance",
				Provider:   "test",
				Model:      "test-model",
				Priority:   1,
				Available:  true,
				Translator: &MockTranslator{},
			},
		},
	}

	wrapper := &MultiLLMTranslatorWrapper{
		Coordinator: coordinator,
		config:      translator.TranslationConfig{},
	}

	// Test translation with progress
	result, err := wrapper.TranslateWithProgress(context.Background(), "test text", "test context", eventBus, "test-session")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "translated text" {
		t.Errorf("Expected 'translated text', got '%s'", result)
	}
}
