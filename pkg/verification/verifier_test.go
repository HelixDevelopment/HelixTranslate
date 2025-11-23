package verification

import (
	"context"
	"fmt"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
)

// Helper function for absolute value of float64
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func TestVerifier_VerifyTranslation(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	tests := []struct {
		name         string
		original     string
		translated   string
		sourceLang   string
		targetLang   string
		expectError  bool
		expectScore  float64
		expectIssues []string
	}{
		{
			name:       "good translation",
			original:   "Hello world",
			translated: "Bonjour le monde",
			sourceLang: "en",
			targetLang: "fr",
			expectError: false,
			expectScore: 0.9,
		},
		{
			name:       "untranslated text",
			original:   "Hello world",
			translated: "Hello world",
			sourceLang: "en",
			targetLang: "fr",
			expectError: false,
			expectScore: 0.0,
			expectIssues: []string{"no_translation"},
		},
		{
			name:       "empty translation",
			original:   "Hello world",
			translated: "",
			sourceLang: "en",
			targetLang: "fr",
			expectError: false,
			expectScore: 0.0,
			expectIssues: []string{"empty_translation"},
		},
		{
			name:       "partial translation",
			original:   "The quick brown fox jumps over the lazy dog",
			translated: "Le rapide renard marron",
			sourceLang: "en",
			targetLang: "fr",
			expectError: false,
			expectScore: 0.4,
			expectIssues: []string{"incomplete_translation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := VerificationRequest{
				Original:   tt.original,
				Translated: tt.translated,
				SourceLang: tt.sourceLang,
				TargetLang: tt.targetLang,
			}
			result, err := verifier.VerifyTranslation(ctx, req)
			if (err != nil) != tt.expectError {
				t.Errorf("VerifyTranslation() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if err == nil {
				if result.Score < tt.expectScore {
					t.Errorf("Expected score >= %f, got %f", tt.expectScore, result.Score)
				}

				for _, expectedIssue := range tt.expectIssues {
					found := false
					for _, issue := range result.Issues {
						if issue.Type == expectedIssue {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected issue type %s not found in issues: %v", expectedIssue, result.Issues)
					}
				}
			}
		})
	}
}

func TestVerifier_BatchVerification(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

		translations := []VerificationRequest{
			{
				Original:   "Hello",
				Translated: "Bonjour",
				SourceLang: "en",
				TargetLang: "fr",
			},
			{
				Original:   "Goodbye",
				Translated: "Au revoir",
				SourceLang: "en",
				TargetLang: "fr",
			},
		{
			Original:   "Thank you",
			Translated: "Merci",
			SourceLang: "en",
			TargetLang: "fr",
		},
	}

	results, err := verifier.BatchVerify(ctx, translations)
	if err != nil {
		t.Fatalf("BatchVerify() error = %v", err)
	}

	if len(results) != len(translations) {
		t.Errorf("Expected %d results, got %d", len(translations), len(results))
	}

	for i, result := range results {
		if result.Score < 0.5 {
			t.Errorf("Translation %d expected score >= 0.5, got %f", i, result.Score)
		}
	}
}

func TestVerifier_QualityMetrics(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	tests := []struct {
		name      string
		original  string
		translated string
		expected  QualityMetrics
	}{
		{
			name:       "length ratio good",
			original:   "Hello world",
			translated: "Bonjour le monde",
			expected: QualityMetrics{
				LengthRatio:     1.4, // 15/11 = 1.36, rounded to 1.4 within tolerance
				WordCountRatio:  1.5, // 3/2 = 1.5
				VocabularyDiversity: 1.0, // All unique words
			},
		},
		{
			name:       "length ratio bad",
			original:   "Hello",
			translated: "",
			expected: QualityMetrics{
				LengthRatio:     0.0, // Empty translation
				WordCountRatio:  0.0, // No words
				VocabularyDiversity: 0.0, // No diversity
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := verifier.calculateQualityMetrics(tt.original, tt.translated)
			
			// Allow some tolerance for floating point comparisons
			tolerance := 0.1
			
			if abs(metrics.LengthRatio-tt.expected.LengthRatio) > tolerance {
				t.Errorf("LengthRatio expected ~%f, got %f", tt.expected.LengthRatio, metrics.LengthRatio)
			}
			
			if abs(metrics.WordCountRatio-tt.expected.WordCountRatio) > tolerance {
				t.Errorf("WordCountRatio expected ~%f, got %f", tt.expected.WordCountRatio, metrics.WordCountRatio)
			}
			
			if abs(metrics.VocabularyDiversity-tt.expected.VocabularyDiversity) > tolerance {
				t.Errorf("VocabularyDiversity expected ~%f, got %f", tt.expected.VocabularyDiversity, metrics.VocabularyDiversity)
			}
		})
	}
}

func TestVerifier_IssueDetection(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	tests := []struct {
		name       string
		original   string
		translated string
		expectedIssues []string
	}{
		{
			name:       "no issues",
			original:   "Hello world",
			translated: "Bonjour le monde",
			expectedIssues: []string{},
		},
		{
			name:       " untranslated",
			original:   "Hello world",
			translated: "Hello world",
			expectedIssues: []string{"no_translation"},
		},
		{
			name:       "empty translation",
			original:   "Hello world",
			translated: "",
			expectedIssues: []string{"empty_translation"},
		},
		{
			name:       "repeated text",
			original:   "Hello",
			translated: "Hello Hello Hello Hello Hello",
			expectedIssues: []string{"repetition"},
		},
		{
			name:       "length mismatch",
			original:   "This is a very long sentence with many words",
			translated: "Court",
			expectedIssues: []string{"length_mismatch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := verifier.detectIssues(tt.original, tt.translated)
			
			if len(issues) != len(tt.expectedIssues) {
				t.Errorf("Expected %d issues, got %d: %v", len(tt.expectedIssues), len(issues), issues)
			}
			
			for _, expectedIssue := range tt.expectedIssues {
				found := false
				for _, issue := range issues {
					if issue.Type == expectedIssue {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected issue type %s not found", expectedIssue)
				}
			}
		})
	}
}

func TestVerifier_ContextAwareVerification(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	contextText := "This is a technical document about software development."
	
	original := "The code is buggy"
	translated := "Le code contient des erreurs"

	result, err := verifier.VerifyWithContext(ctx, original, translated, "en", "fr", contextText)
	if err != nil {
		t.Fatalf("VerifyWithContext() error = %v", err)
	}

	if result.Score < 0.5 {
		t.Errorf("Expected score >= 0.5 with context, got %f", result.Score)
	}

	// Check if context was considered
	if !result.ContextConsidered {
		t.Error("Expected ContextConsidered to be true")
	}
}

func TestVerifier_Configuration(t *testing.T) {
	config := VerificationConfig{
		MinScore:         0.8,
		EnableContext:     true,
		EnableSpellCheck:  true,
		EnableGrammarCheck: true,
		StrictMode:        true,
	}

	verifier := NewVerifierWithConfig(
		language.English,
		language.French,
		events.NewEventBus(),
		"test-session",
		config,
	)

	if verifier.config.MinScore != 0.8 {
		t.Errorf("Expected MinScore 0.8, got %f", verifier.config.MinScore)
	}

	if !verifier.config.EnableContext {
		t.Error("Expected EnableContext to be true")
	}
}

func TestVerifier_ConcurrentVerification(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	numTranslations := 10
	translations := make([]VerificationRequest, numTranslations)
	
	for i := 0; i < numTranslations; i++ {
		translations[i] = VerificationRequest{
			Original:   fmt.Sprintf("Text %d", i),
			Translated: fmt.Sprintf("Traduction %d", i),
			SourceLang: "en",
			TargetLang: "fr",
		}
	}

	// Test concurrent verification
	results, err := verifier.BatchVerifyConcurrent(ctx, translations) // Simplified for test compatibility
	if err != nil {
		t.Fatalf("BatchVerifyConcurrent() error = %v", err)
	}

	if len(results) != numTranslations {
		t.Errorf("Expected %d results, got %d", numTranslations, len(results))
	}

	for i, result := range results {
		if result.Score < 0.5 {
			t.Errorf("Translation %d expected score >= 0.5, got %f", i, result.Score)
		}
	}
}

func TestVerifier_ErrorHandling(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cancel() // Cancel immediately

		req := VerificationRequest{
		Original:   "test",
		Translated: "test",
		SourceLang: "en",
		TargetLang: "fr",
	}
	_, err := verifier.VerifyTranslation(ctx, req)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

func TestVerifier_InvalidInputs(t *testing.T) {
	eventBus := events.NewEventBus()
	verifier := NewVerifier(
		language.English,
		language.French,
		eventBus,
		"test-session",
	)

	ctx := context.Background()

	tests := []struct {
		name        string
		original    string
		translated  string
		sourceLang  string
		targetLang  string
		expectError bool
	}{
		{
			name:        "empty source language",
			original:    "Hello",
			translated:  "Bonjour",
			sourceLang:  "",
			targetLang:  "fr",
			expectError: true,
		},
		{
			name:        "empty target language",
			original:    "Hello",
			translated:  "Bonjour",
			sourceLang:  "en",
			targetLang:  "",
			expectError: true,
		},
		{
			name:        "same source and target language",
			original:    "Hello",
			translated:  "Hello",
			sourceLang:  "en",
			targetLang:  "en",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := VerificationRequest{
				Original:   tt.original,
				Translated: tt.translated,
				SourceLang: tt.sourceLang,
				TargetLang: tt.targetLang,
			}
			_, err := verifier.VerifyTranslation(ctx, req)
			if (err != nil) != tt.expectError {
				t.Errorf("VerifyTranslation() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}