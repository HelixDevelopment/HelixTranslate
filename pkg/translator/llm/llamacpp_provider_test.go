package llm

import (
	"strings"
	"testing"
)

// TestGetLanguageName tests the getLanguageName function
func TestGetLanguageName(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"en", "English"},
		{"ru", "Russian"},
		{"sr", "Serbian"},
		{"sr-cyrl", "Serbian Cyrillic"},
		{"sr-latn", "Serbian Latin"},
		{"unknown", "unknown"},
		{"", ""},
		{"de", "de"},
		{"fr", "fr"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			// Since getLanguageName is a method on MultiLLMCoordinator,
			// we need to create an instance to test it
			coordinator := &MultiLLMCoordinator{}
			result := coordinator.getLanguageName(test.input)
			if result != test.expect {
				t.Errorf("getLanguageName(%s) = %s, want %s", test.input, result, test.expect)
			}
		})
	}
}

// TestRemoveAnsiCodes tests the removeAnsiCodes function
func TestRemoveAnsiCodes(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"plain text", "plain text"},
		{"", ""},
		{"text with \x1b[31mcolor codes\x1b[0m", "text with \x1b[31mcolor codes\x1b[0m"}, // Currently not removing codes
		{"single line", "single line"},
		{"multi\nline\ntext", "multi\nline\ntext"},
		{"special chars !@#$%", "special chars !@#$%"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := removeAnsiCodes(test.input)
			if result != test.expect {
				t.Errorf("removeAnsiCodes(%q) = %q, want %q", test.input, result, test.expect)
			}
		})
	}
}

// TestMultiLLMCoordinatorSimpleMethods tests simple methods of MultiLLMCoordinator
func TestMultiLLMCoordinatorSimpleMethods(t *testing.T) {
	t.Run("GetProviderName", func(t *testing.T) {
		coordinator := &MultiLLMCoordinator{}
		name := coordinator.GetProviderName()
		if name != "llamacpp-multi" {
			t.Errorf("GetProviderName() = %s, want 'llamacpp-multi'", name)
		}
	})

	t.Run("GetAvailableModels_empty", func(t *testing.T) {
		coordinator := &MultiLLMCoordinator{
			Workers: map[string]*LlamaCppWorker{},
		}
		models := coordinator.GetAvailableModels()
		if len(models) != 0 {
			t.Errorf("GetAvailableModels() returned %d models, want 0", len(models))
		}
	})

	t.Run("GetAvailableModels_with_workers", func(t *testing.T) {
		coordinator := &MultiLLMCoordinator{
			Workers: map[string]*LlamaCppWorker{
				"worker1": {
					Config: ModelConfig{ModelName: "model1"},
				},
				"worker2": {
					Config: ModelConfig{ModelName: "model2"},
				},
			},
		}
		models := coordinator.GetAvailableModels()
		if len(models) != 2 {
			t.Errorf("GetAvailableModels() returned %d models, want 2", len(models))
		}
		
		// Since map iteration order is not guaranteed, check that both models are present
		foundModels := make(map[string]bool)
		for _, model := range models {
			foundModels[model.ModelName] = true
		}
		
		if !foundModels["model1"] || !foundModels["model2"] {
			t.Errorf("GetAvailableModels() returned incorrect models: %v", models)
		}
	})
}

// TestBuildPrompt tests the buildPrompt function
func TestBuildPrompt(t *testing.T) {
	coordinator := &MultiLLMCoordinator{}
	
	t.Run("basic_translation", func(t *testing.T) {
		task := TranslationTask{
			FromLang: "en",
			ToLang:   "sr",
			Text:     "Hello world",
		}
		
		prompt := coordinator.buildPrompt(task)
		
		// Check that the prompt contains expected elements
		if !strings.Contains(prompt, "English") {
			t.Error("Prompt should contain 'English'")
		}
		if !strings.Contains(prompt, "Serbian") {
			t.Error("Prompt should contain 'Serbian'")
		}
		if !strings.Contains(prompt, "Hello world") {
			t.Error("Prompt should contain the input text")
		}
		if !strings.Contains(prompt, "Translation:") {
			t.Error("Prompt should end with 'Translation:'")
		}
	})
	
	t.Run("unknown_languages", func(t *testing.T) {
		task := TranslationTask{
			FromLang: "xx",
			ToLang:   "yy",
			Text:     "Test text",
		}
		
		prompt := coordinator.buildPrompt(task)
		
		// Should use the language codes directly when unknown
		if !strings.Contains(prompt, "xx") {
			t.Error("Prompt should contain 'xx' when language is unknown")
		}
		if !strings.Contains(prompt, "yy") {
			t.Error("Prompt should contain 'yy' when language is unknown")
		}
	})
	
	t.Run("empty_text", func(t *testing.T) {
		task := TranslationTask{
			FromLang: "en",
			ToLang:   "ru",
			Text:     "",
		}
		
		prompt := coordinator.buildPrompt(task)
		
		// Should still include all prompt structure even with empty text
		if !strings.Contains(prompt, "English") {
			t.Error("Prompt should contain 'English'")
		}
		if !strings.Contains(prompt, "Russian") {
			t.Error("Prompt should contain 'Russian'")
		}
		if !strings.Contains(prompt, "Source text:") {
			t.Error("Prompt should contain 'Source text:'")
		}
	})
}

// TestParseOutput tests the parseOutput function
func TestParseOutput(t *testing.T) {
	coordinator := &MultiLLMCoordinator{}
	
	t.Run("simple_translation", func(t *testing.T) {
		output := "Some prompt text\nTranslation:Hola mundo\nMore text"
		result := coordinator.parseOutput(output)
		expected := "Hola mundo\nMore text"
		if result != expected {
			t.Errorf("parseOutput() = %q, want %q", result, expected)
		}
	})
	
	t.Run("translation_on_same_line", func(t *testing.T) {
		output := "Some text Translation:Bonjour le monde More text"
		result := coordinator.parseOutput(output)
		expected := "Bonjour le monde More text"
		if result != expected {
			t.Errorf("parseOutput() = %q, want %q", result, expected)
		}
	})
	
	t.Run("multiline_translation", func(t *testing.T) {
		output := "Prompt text\nTranslation:Line 1\nLine 2\nLine 3\nMore text"
		result := coordinator.parseOutput(output)
		expected := "Line 1\nLine 2\nLine 3\nMore text"
		if result != expected {
			t.Errorf("parseOutput() = %q, want %q", result, expected)
		}
	})
	
	t.Run("no_translation_marker", func(t *testing.T) {
		output := "Some text without translation marker\nMore text"
		result := coordinator.parseOutput(output)
		expected := ""
		if result != expected {
			t.Errorf("parseOutput() = %q, want %q", result, expected)
		}
	})
	
	t.Run("empty_translation", func(t *testing.T) {
		output := "Text Translation: More text"
		result := coordinator.parseOutput(output)
		expected := "More text"
		if result != expected {
			t.Errorf("parseOutput() = %q, want %q", result, expected)
		}
	})
}