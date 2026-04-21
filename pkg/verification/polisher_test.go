package verification

import (
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"strings"
	"testing"
)

// TestNewBookPolisher tests creation of book polisher
func TestNewBookPolisher(t *testing.T) {
	config := PolishingConfig{
		Providers:        []string{"mock"},
		MinConsensus:     1,
		VerifySpirit:     true,
		VerifyLanguage:   true,
		VerifyContext:    true,
		VerifyVocabulary: true,
		TranslationConfigs: map[string]translator.TranslationConfig{
			"mock": {
				Provider:   "mock",
				SourceLang: "ru",
				TargetLang: "sr",
			},
		},
	}

	eventBus := events.NewEventBus()
	sessionID := "test-session"

	// Test with an unsupported provider
	config.Providers = []string{"nonexistent-provider"}
	config.TranslationConfigs["nonexistent-provider"] = translator.TranslationConfig{
		Provider:   "nonexistent-provider",
		SourceLang: "ru",
		TargetLang: "sr",
	}
	_, err := NewBookPolisher(config, eventBus, sessionID)

	// Should fail due to unsupported provider
	if err == nil {
		t.Error("Expected error for unsupported provider, got nil")
	}
}

// TestPolishingConfig tests configuration structure
func TestPolishingConfig(t *testing.T) {
	config := PolishingConfig{
		Providers:          []string{"deepseek", "anthropic"},
		MinConsensus:       2,
		VerifySpirit:       true,
		VerifyLanguage:     true,
		VerifyContext:      true,
		VerifyVocabulary:   true,
		TranslationConfigs: make(map[string]translator.TranslationConfig),
	}

	if len(config.Providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(config.Providers))
	}

	if config.MinConsensus != 2 {
		t.Errorf("Expected MinConsensus=2, got %d", config.MinConsensus)
	}

	if !config.VerifySpirit {
		t.Error("Expected VerifySpirit=true")
	}
}

// TestPolishingResult tests result structure
func TestPolishingResult(t *testing.T) {
	result := &PolishingResult{
		SectionID:       "test_section",
		Location:        "Chapter 1",
		OriginalText:    "Привет мир",
		TranslatedText:  "Здраво свет",
		PolishedText:    "Здраво свете",
		Changes:         make([]Change, 0),
		Issues:          make([]Issue, 0),
		Suggestions:     make([]Suggestion, 0),
		Consensus:       2,
		Confidence:      0.95,
		SpiritScore:     0.90,
		LanguageScore:   0.92,
		ContextScore:    0.88,
		VocabularyScore: 0.91,
	}

	// Calculate overall score
	result.OverallScore = (result.SpiritScore + result.LanguageScore +
		result.ContextScore + result.VocabularyScore) / 4.0

	expectedOverall := (0.90 + 0.92 + 0.88 + 0.91) / 4.0
	// Use epsilon for float comparison
	epsilon := 0.0001
	if result.OverallScore < expectedOverall-epsilon || result.OverallScore > expectedOverall+epsilon {
		t.Errorf("Expected overall score %.4f, got %.4f", expectedOverall, result.OverallScore)
	}

	if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %.2f", result.Confidence)
	}
}

// TestChange tests change structure
func TestChange(t *testing.T) {
	change := Change{
		Location:   "Chapter 1, Section 2",
		Original:   "плохой перевод",
		Polished:   "лош превод",
		Reason:     "Better vocabulary choice",
		Agreement:  3,
		Confidence: 0.9,
	}

	if change.Agreement != 3 {
		t.Errorf("Expected agreement=3, got %d", change.Agreement)
	}

	if change.Confidence != 0.9 {
		t.Errorf("Expected confidence=0.9, got %.2f", change.Confidence)
	}
}

// TestIssue tests issue structure
func TestIssue(t *testing.T) {
	issue := Issue{
		Type:        "vocabulary",
		Severity:    "minor",
		Description: "Word choice could be improved",
		Location:    "Chapter 3",
		Suggestion:  "Use more varied vocabulary",
	}

	if issue.Type != "vocabulary" {
		t.Errorf("Expected type=vocabulary, got %s", issue.Type)
	}

	if issue.Severity != "minor" {
		t.Errorf("Expected severity=minor, got %s", issue.Severity)
	}
}

// TestExtractScore tests score extraction from LLM response
func TestExtractScore(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		prefix   string
		expected float64
	}{
		{
			name:     "valid score",
			text:     "SPIRIT_SCORE: 0.95\nOther text",
			prefix:   "SPIRIT_SCORE:",
			expected: 0.95,
		},
		{
			name:     "score with spaces",
			text:     "LANGUAGE_SCORE:  0.88  \nOther text",
			prefix:   "LANGUAGE_SCORE:",
			expected: 0.88,
		},
		{
			name:     "score not found",
			text:     "No score here",
			prefix:   "SPIRIT_SCORE:",
			expected: -1.0,
		},
		{
			name:     "invalid score format",
			text:     "SPIRIT_SCORE: invalid",
			prefix:   "SPIRIT_SCORE:",
			expected: -1.0,
		},
		{
			name:     "score out of range",
			text:     "SPIRIT_SCORE: 1.5",
			prefix:   "SPIRIT_SCORE:",
			expected: -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractScore(tt.text, tt.prefix)
			if result != tt.expected {
				t.Errorf("extractScore() = %.2f, expected %.2f", result, tt.expected)
			}
		})
	}
}

// TestExtractSection tests section extraction from LLM response
func TestExtractSection(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		startMarker string
		endMarker   string
		expected    string
	}{
		{
			name: "valid section",
			text: `Some text
POLISHED_TEXT:
This is the polished version
EXPLANATION:
This is why`,
			startMarker: "POLISHED_TEXT:",
			endMarker:   "EXPLANATION:",
			expected:    "This is the polished version",
		},
		{
			name: "section to end",
			text: `Some text
EXPLANATION:
This is the final section`,
			startMarker: "EXPLANATION:",
			endMarker:   "NOTFOUND:",
			expected:    "This is the final section",
		},
		{
			name:        "marker not found",
			text:        "No markers here",
			startMarker: "POLISHED_TEXT:",
			endMarker:   "EXPLANATION:",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSection(tt.text, tt.startMarker, tt.endMarker)
			if result != tt.expected {
				t.Errorf("extractSection() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestCreateVerificationPrompt tests prompt creation
func TestCreateVerificationPrompt(t *testing.T) {
	bp := &BookPolisher{
		config: PolishingConfig{
			VerifySpirit:     true,
			VerifyLanguage:   true,
			VerifyContext:    true,
			VerifyVocabulary: true,
		},
	}

	originalText := "Привет, мир!"
	translatedText := "Здраво, свете!"

	prompt := bp.createVerificationPrompt(originalText, translatedText)

	// Check that prompt contains key elements
	if !strings.Contains(prompt, originalText) {
		t.Error("Prompt should contain original text")
	}

	if !strings.Contains(prompt, translatedText) {
		t.Error("Prompt should contain translated text")
	}

	if !strings.Contains(prompt, "Spirit") {
		t.Error("Prompt should mention Spirit dimension")
	}

	if !strings.Contains(prompt, "Language") {
		t.Error("Prompt should mention Language dimension")
	}

	if !strings.Contains(prompt, "Context") {
		t.Error("Prompt should mention Context dimension")
	}

	if !strings.Contains(prompt, "Vocabulary") {
		t.Error("Prompt should mention Vocabulary dimension")
	}

	if !strings.Contains(prompt, "SPIRIT_SCORE:") {
		t.Error("Prompt should specify SPIRIT_SCORE format")
	}
}

// TestParseVerificationResponse tests LLM response parsing
func TestParseVerificationResponse(t *testing.T) {
	bp := &BookPolisher{
		config: PolishingConfig{
			Providers: []string{"test"},
		},
	}

	response := `SPIRIT_SCORE: 0.92
LANGUAGE_SCORE: 0.88
CONTEXT_SCORE: 0.90
VOCABULARY_SCORE: 0.85

ISSUES:
vocabulary: Could use richer vocabulary
context: Minor context issue in sentence 3

POLISHED_TEXT:
Ово је побољшана верзија текста

EXPLANATION:
Changed vocabulary for better flow`

	originalTranslation := "Ово је оригинални превод"

	verification := bp.parseVerificationResponse("test", response, originalTranslation)

	if verification.Provider != "test" {
		t.Errorf("Expected provider=test, got %s", verification.Provider)
	}

	if verification.SpiritScore != 0.92 {
		t.Errorf("Expected SpiritScore=0.92, got %.2f", verification.SpiritScore)
	}

	if verification.LanguageScore != 0.88 {
		t.Errorf("Expected LanguageScore=0.88, got %.2f", verification.LanguageScore)
	}

	if verification.ContextScore != 0.90 {
		t.Errorf("Expected ContextScore=0.90, got %.2f", verification.ContextScore)
	}

	if verification.VocabularyScore != 0.85 {
		t.Errorf("Expected VocabularyScore=0.85, got %.2f", verification.VocabularyScore)
	}

	expectedPolished := "Ово је побољшана верзија текста"
	if verification.PolishedText != expectedPolished {
		t.Errorf("Expected polished text=%q, got %q", expectedPolished, verification.PolishedText)
	}

	if len(verification.Issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(verification.Issues))
	}
}

// TestBuildConsensus tests consensus building from multiple verifications
func TestBuildConsensus(t *testing.T) {
	bp := &BookPolisher{
		config: PolishingConfig{
			Providers:    []string{"llm1", "llm2", "llm3"},
			MinConsensus: 2,
		},
	}

	originalText := "Привет"
	translatedText := "Здраво"
	polishedText := "Поздрав" // Better translation

	verifications := []llmVerification{
		{
			Provider:        "llm1",
			SpiritScore:     0.90,
			LanguageScore:   0.88,
			ContextScore:    0.92,
			VocabularyScore: 0.85,
			PolishedText:    polishedText, // Suggests change
		},
		{
			Provider:        "llm2",
			SpiritScore:     0.92,
			LanguageScore:   0.90,
			ContextScore:    0.88,
			VocabularyScore: 0.87,
			PolishedText:    polishedText, // Agrees with llm1
		},
		{
			Provider:        "llm3",
			SpiritScore:     0.88,
			LanguageScore:   0.86,
			ContextScore:    0.90,
			VocabularyScore: 0.84,
			PolishedText:    translatedText, // No change
		},
	}

	result := bp.buildConsensus(
		"test_section",
		"Test Location",
		originalText,
		translatedText,
		verifications,
	)

	// Check average scores
	expectedSpirit := (0.90 + 0.92 + 0.88) / 3.0
	if result.SpiritScore != expectedSpirit {
		t.Errorf("Expected SpiritScore=%.2f, got %.2f", expectedSpirit, result.SpiritScore)
	}

	// Check consensus (2 LLMs agreed on polished version)
	if result.Consensus != 2 {
		t.Errorf("Expected Consensus=2, got %d", result.Consensus)
	}

	// Consensus reached (2 >= MinConsensus=2), so polished version should be applied
	if result.PolishedText != polishedText {
		t.Errorf("Expected polished text=%q, got %q", polishedText, result.PolishedText)
	}

	// Check confidence
	expectedConfidence := 2.0 / 3.0
	if result.Confidence != expectedConfidence {
		t.Errorf("Expected Confidence=%.2f, got %.2f", expectedConfidence, result.Confidence)
	}

	// Check that change was recorded
	if len(result.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(result.Changes))
	}
}

// TestBuildConsensusNoChange tests consensus when no change needed
func TestBuildConsensusNoChange(t *testing.T) {
	bp := &BookPolisher{
		config: PolishingConfig{
			Providers:    []string{"llm1", "llm2"},
			MinConsensus: 2,
		},
	}

	originalText := "Привет"
	translatedText := "Здраво"

	verifications := []llmVerification{
		{
			Provider:        "llm1",
			SpiritScore:     0.95,
			LanguageScore:   0.96,
			ContextScore:    0.94,
			VocabularyScore: 0.95,
			PolishedText:    translatedText, // No change
		},
		{
			Provider:        "llm2",
			SpiritScore:     0.94,
			LanguageScore:   0.95,
			ContextScore:    0.96,
			VocabularyScore: 0.94,
			PolishedText:    translatedText, // No change
		},
	}

	result := bp.buildConsensus(
		"test_section",
		"Test Location",
		originalText,
		translatedText,
		verifications,
	)

	// No change should be made
	if result.PolishedText != translatedText {
		t.Errorf("Expected no change to text, got %q", result.PolishedText)
	}

	// No changes should be recorded
	if len(result.Changes) != 0 {
		t.Errorf("Expected 0 changes, got %d", len(result.Changes))
	}

	// High scores expected
	if result.OverallScore < 0.90 {
		t.Errorf("Expected high overall score, got %.2f", result.OverallScore)
	}
}

// TestBuildConsensusInsufficientAgreement tests insufficient consensus
func TestBuildConsensusInsufficientAgreement(t *testing.T) {
	bp := &BookPolisher{
		config: PolishingConfig{
			Providers:    []string{"llm1", "llm2", "llm3"},
			MinConsensus: 3, // All must agree
		},
	}

	originalText := "Привет"
	translatedText := "Здраво"

	verifications := []llmVerification{
		{
			Provider:     "llm1",
			SpiritScore:  0.90,
			PolishedText: "Поздрав", // Different suggestion
		},
		{
			Provider:     "llm2",
			SpiritScore:  0.92,
			PolishedText: "Хеј", // Different suggestion
		},
		{
			Provider:     "llm3",
			SpiritScore:  0.88,
			PolishedText: translatedText, // Keep original
		},
	}

	result := bp.buildConsensus(
		"test_section",
		"Test Location",
		originalText,
		translatedText,
		verifications,
	)

	// No consensus (max agreement is 1, but need 3)
	if result.Consensus >= 3 {
		t.Errorf("Should not have consensus, got %d", result.Consensus)
	}

	// Original translation should be kept
	if result.PolishedText != translatedText {
		t.Errorf("Expected original text preserved, got %q", result.PolishedText)
	}
}

// TestMetadataStructure tests metadata polishing would work with proper structure
func TestMetadataStructure(t *testing.T) {
	original := ebook.Metadata{
		Title:       "Война и мир",
		Description: "Роман Льва Толстого",
		Authors:     []string{"Лев Толстой"},
		Language:    "ru",
	}

	translated := ebook.Metadata{
		Title:       "Рат и мир",
		Description: "Роман Лава Толстоја",
		Authors:     []string{"Лав Толстој"},
		Language:    "sr",
	}

	// Verify structure is correct
	if original.Title == "" {
		t.Error("Original title should not be empty")
	}

	if translated.Title == "" {
		t.Error("Translated title should not be empty")
	}

	if original.Language != "ru" {
		t.Errorf("Expected original language=ru, got %s", original.Language)
	}

	if translated.Language != "sr" {
		t.Errorf("Expected translated language=sr, got %s", translated.Language)
	}
}

// Benchmark tests

func BenchmarkExtractScore(b *testing.B) {
	text := "SPIRIT_SCORE: 0.95\nLANGUAGE_SCORE: 0.88\nOther text"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractScore(text, "SPIRIT_SCORE:")
	}
}

func BenchmarkExtractSection(b *testing.B) {
	text := `POLISHED_TEXT:
This is a long section of polished text that needs to be extracted
from the LLM response for processing.
EXPLANATION:
More text here`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractSection(text, "POLISHED_TEXT:", "EXPLANATION:")
	}
}

func BenchmarkBuildConsensus(b *testing.B) {
	bp := &BookPolisher{
		config: PolishingConfig{
			Providers:    []string{"llm1", "llm2", "llm3"},
			MinConsensus: 2,
		},
	}

	verifications := []llmVerification{
		{Provider: "llm1", SpiritScore: 0.90, PolishedText: "Здраво"},
		{Provider: "llm2", SpiritScore: 0.92, PolishedText: "Здраво"},
		{Provider: "llm3", SpiritScore: 0.88, PolishedText: "Поздрав"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bp.buildConsensus("test", "Test", "Привет", "Здраво", verifications)
	}
}
