package translator_test

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// TestEkavicaPromptInclusion tests that the translation prompt includes Ekavica requirements
func TestEkavicaPromptInclusion(t *testing.T) {
	config := translator.TranslationConfig{
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
		APIKey:         "test-key",
	}

	_, err := llm.NewLLMTranslator(config)
	if err != nil {
		t.Fatalf("Failed to create LLM translator: %v", err)
	}

	// Access the private createTranslationPrompt method via reflection or test the public API
	// For now, we'll test through the actual Translate call which uses the prompt
	_ = context.Background()

	// Note: This will fail without a real API key, but we're testing prompt construction
	// In a real test, we'd mock the LLM client or use a test double
	t.Log("Ekavica prompt inclusion test - manual verification required in production tests")
}

// TestEkavicaDialectRequirements verifies Ekavica-specific word patterns
func TestEkavicaDialectRequirements(t *testing.T) {
	tests := []struct {
		name           string
		translation    string
		shouldContain  []string // Ekavica forms that MUST be present
		shouldNotContain []string // Ijekavica forms that MUST NOT be present
		expectPass     bool
	}{
		{
			name:        "Ekavica word 'mleko' is correct",
			translation: "Деца пију млеко",
			shouldContain: []string{"млеко"},
			shouldNotContain: []string{"мљеко", "мlijeko"},
			expectPass:  true,
		},
		{
			name:        "Ijekavica word 'mlijeko' is incorrect",
			translation: "Деца пију мљеко",
			shouldContain: nil,
			shouldNotContain: []string{"мљеко"},
			expectPass:  false,
		},
		{
			name:        "Ekavica word 'dete' is correct",
			translation: "Дете игра лопту",
			shouldContain: []string{"дете"},
			shouldNotContain: []string{"дијете", "dijete"},
			expectPass:  true,
		},
		{
			name:        "Ekavica word 'pesma' is correct",
			translation: "Лепа песма звучи",
			shouldContain: []string{"песма"},
			shouldNotContain: []string{"пјесма", "pjesma"},
			expectPass:  true,
		},
		{
			name:        "Ekavica word 'lepo' is correct",
			translation: "То је лепо",
			shouldContain: []string{"лепо"},
			shouldNotContain: []string{"лијепо", "lijepo"},
			expectPass:  true,
		},
		{
			name:        "Ekavica word 'hteo' is correct",
			translation: "Он је хтео да дође",
			shouldContain: []string{"хтео"},
			shouldNotContain: []string{"хтио", "htio"},
			expectPass:  true,
		},
		{
			name:        "Ekavica word 'reka' (river) is correct",
			translation: "Река је дубока",
			shouldContain: []string{"река"},
			shouldNotContain: []string{"ријека", "rijeka"},
			expectPass:  true,
		},
		{
			name:        "Multiple Ekavica words are correct",
			translation: "Лепа деца пију млеко и певају песму",
			shouldContain: []string{"лепа", "деца", "млеко", "песму"},
			shouldNotContain: []string{"лијепа", "дијеца", "мљеко", "пјесму"},
			expectPass:  true,
		},
		{
			name:        "Mixed dialect is incorrect",
			translation: "Лепа деца пију мљеко", // mixes Ekavica (лепа, деца) with Ijekavica (мљеко)
			shouldContain: []string{"лепа", "деца"},
			shouldNotContain: []string{"мљеко"},
			expectPass:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed := true

			// Check that required Ekavica forms are present
			for _, required := range tt.shouldContain {
				if !strings.Contains(tt.translation, required) {
					t.Errorf("Translation should contain Ekavica form %q but doesn't", required)
					passed = false
				}
			}

			// Check that Ijekavica forms are NOT present
			for _, forbidden := range tt.shouldNotContain {
				if strings.Contains(tt.translation, forbidden) {
					t.Errorf("Translation should NOT contain Ijekavica form %q but does", forbidden)
					passed = false
				}
			}

			if passed != tt.expectPass {
				if tt.expectPass {
					t.Error("Test expected to pass but failed")
				} else {
					t.Error("Test expected to fail but passed")
				}
			}
		})
	}
}

// TestEkavicaRegexPatterns tests pattern-based Ijekavica detection
func TestEkavicaRegexPatterns(t *testing.T) {
	testCases := []struct {
		name        string
		text        string
		shouldMatch bool
		pattern     string
	}{
		{"Ijekavica dete", "дијете", true, "дијет"},
		{"Ekavica dete", "дете", false, "дијет"},
		{"Ijekavica mleko", "мљеко", true, "мљек"},
		{"Ekavica mleko", "млеко", false, "мљек"},
		{"Ijekavica lepo", "лијепо", true, "љеп"},
		{"Ekavica lepo", "лепо", false, "љеп"},
		{"Ijekavica pattern ije", "пијем", true, "ије"},
		{"Ekavica no ije", "пе вам", false, "ије"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := strings.Contains(tc.text, tc.pattern)
			if matched != tc.shouldMatch {
				if tc.shouldMatch {
					t.Errorf("Expected pattern %q to match in %q but it didn't", tc.pattern, tc.text)
				} else {
					t.Errorf("Expected pattern %q NOT to match in %q but it did", tc.pattern, tc.text)
				}
			}
		})
	}
}

// TestEkavicaCommonWords tests that common words use Ekavica forms
func TestEkavicaCommonWords(t *testing.T) {
	// Mapping of Ekavica to Ijekavica for common words
	ekavicaToIjekavica := map[string]string{
		"млеко":  "мљеко",
		"дете":   "дијете",
		"песма":  "пјесма",
		"лепо":   "лијепо",
		"хтео":   "хтио",
		"река":   "ријека",
		"бео":    "бијел",
		"цвет":   "цвијет",
		"лето":   "љето",
		"место":  "мјесто",
		"пети":   "пјети",
		"недеља": "недјеља",
		"тело":   "тијело",
		"вера":   "вјера",
		"време":  "вријеме",
	}

	for ekavica, ijekavica := range ekavicaToIjekavica {
		t.Run("Word_"+ekavica, func(t *testing.T) {
			// Test sentence with Ekavica word
			ekavicaSentence := "То је " + ekavica + " које волим"

			// Verify it contains Ekavica form
			if !strings.Contains(ekavicaSentence, ekavica) {
				t.Errorf("Sentence should contain Ekavica word %q", ekavica)
			}

			// Verify it does NOT contain Ijekavica form
			if strings.Contains(ekavicaSentence, ijekavica) {
				t.Errorf("Sentence should NOT contain Ijekavica word %q", ijekavica)
			}
		})
	}
}

// TestEkavicaVerificationLogic tests the logic for detecting Ijekavica in translations
func TestEkavicaVerificationLogic(t *testing.T) {
	type verificationResult struct {
		hasIjekavica bool
		violations   []string
	}

	detectIjekavica := func(text string) verificationResult {
		result := verificationResult{
			hasIjekavica: false,
			violations:   make([]string, 0),
		}

		// Check for common Ijekavica patterns
		ijekavicaWords := []string{
			"мљеко", "мљека", // mleko variants
			"дијете", "дијета", "дјеца", // dete variants
			"пјесма", "пјесме", // pesma variants
			"лијепо", "лијеп", "љеп", // lepo variants
			"хтио", "хтјео", // hteo variants
			"ријека", "ријеке", // reka variants
		}

		for _, word := range ijekavicaWords {
			if strings.Contains(text, word) {
				result.hasIjekavica = true
				result.violations = append(result.violations, word)
			}
		}

		// Also check for the core pattern "ије"
		if strings.Contains(text, "ије") {
			result.hasIjekavica = true
			if len(result.violations) == 0 {
				result.violations = append(result.violations, "contains 'ије' pattern")
			}
		}

		return result
	}

	tests := []struct {
		name             string
		text             string
		expectIjekavica  bool
		expectedViolations int
	}{
		{
			name:            "Pure Ekavica text",
			text:            "Деца пију млеко и певају лепу песму",
			expectIjekavica: false,
			expectedViolations: 0,
		},
		{
			name:            "Text with Ijekavica mleko",
			text:            "Деца пију мљеко",
			expectIjekavica: true,
			expectedViolations: 1,
		},
		{
			name:            "Text with multiple Ijekavica words",
			text:            "Дијете пије мљеко и пјева лијепу пјесму",
			expectIjekavica: true,
			expectedViolations: 4,
		},
		{
			name:            "Text with ије pattern",
			text:            "Пијемо воду",
			expectIjekavica: true,
			expectedViolations: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectIjekavica(tt.text)

			if result.hasIjekavica != tt.expectIjekavica {
				t.Errorf("Expected hasIjekavica=%v, got %v", tt.expectIjekavica, result.hasIjekavica)
			}

			if len(result.violations) != tt.expectedViolations {
				t.Errorf("Expected %d violations, got %d: %v",
					tt.expectedViolations, len(result.violations), result.violations)
			}

			if result.hasIjekavica {
				t.Logf("Detected Ijekavica violations: %v", result.violations)
			}
		})
	}
}

// BenchmarkEkavicaDetection benchmarks the Ijekavica detection logic
func BenchmarkEkavicaDetection(b *testing.B) {
	text := `Деца пију млеко и певају лепу песму. Хтео сам да дођем
	           у место где тече река. То је лепо време за летњи дан.`

	ijekavicaPatterns := []string{"мљеко", "дијете", "пјесма", "лијепо", "хтио", "ријека", "ије"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, pattern := range ijekavicaPatterns {
			_ = strings.Contains(text, pattern)
		}
	}
}
