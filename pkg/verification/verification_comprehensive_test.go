package verification

import (
	"context"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerificationConfig tests the VerificationConfig structure
func TestVerificationConfig(t *testing.T) {
	config := VerificationConfig{
		StrictMode:          true,
		EnableQualityCheck:   true,
		EnableIssueDetection: true,
		EnableContext:        true,
		EnableSpellCheck:     true,
		EnableGrammarCheck:   true,
		MinQualityScore:     0.8,
		MinScore:           0.8, // Alias
		AllowedLanguages:    []string{"en", "es", "fr"},
	}

	assert.True(t, config.StrictMode)
	assert.True(t, config.EnableQualityCheck)
	assert.Equal(t, 0.8, config.MinQualityScore)
	assert.Equal(t, 0.8, config.MinScore)
	assert.Equal(t, []string{"en", "es", "fr"}, config.AllowedLanguages)
}

// TestVerificationRequest tests the VerificationRequest structure
func TestVerificationRequest(t *testing.T) {
	req := VerificationRequest{
		Original:   "Hello world",
		Translated: "Hola mundo",
		SourceLang: "en",
		TargetLang: "es",
		Context:    "greeting",
		Metadata: map[string]string{
			"chapter": "1",
			"section": "2",
		},
	}

	assert.Equal(t, "Hello world", req.Original)
	assert.Equal(t, "Hola mundo", req.Translated)
	assert.Equal(t, "en", req.SourceLang)
	assert.Equal(t, "es", req.TargetLang)
	assert.Equal(t, "greeting", req.Context)
	assert.Equal(t, "1", req.Metadata["chapter"])
}

// TestQualityMetrics tests the QualityMetrics structure
func TestQualityMetrics(t *testing.T) {
	metrics := QualityMetrics{
		LengthRatio:        1.2,
		WordCountRatio:     1.1,
		VocabularyDiversity: 0.8,
		Accuracy:           0.9,
		Fluency:            0.85,
		Consistency:        0.95,
		Completeness:       0.92,
		Overall:            0.905,
	}

	assert.Equal(t, 1.2, metrics.LengthRatio)
	assert.Equal(t, 1.1, metrics.WordCountRatio)
	assert.Equal(t, 0.8, metrics.VocabularyDiversity)
	assert.Equal(t, 0.9, metrics.Accuracy)
	assert.Equal(t, 0.905, metrics.Overall)
}

// TestUntranslatedBlock tests the UntranslatedBlock structure
func TestUntranslatedBlock(t *testing.T) {
	block := UntranslatedBlock{
		Location:    "Chapter 1, Section 2",
		OriginalText: "Hello world",
		Language:     "en",
		Length:       11,
	}

	assert.Equal(t, "Chapter 1, Section 2", block.Location)
	assert.Equal(t, "Hello world", block.OriginalText)
	assert.Equal(t, "en", block.Language)
	assert.Equal(t, 11, block.Length)
}

// TestHTMLArtifact tests the HTMLArtifact structure
func TestHTMLArtifact(t *testing.T) {
	artifact := HTMLArtifact{
		Location: "Chapter 1, Section 1",
		Content:  "<p>Hello</p>",
		Type:     "tag",
	}

	assert.Equal(t, "Chapter 1, Section 1", artifact.Location)
	assert.Equal(t, "<p>Hello</p>", artifact.Content)
	assert.Equal(t, "tag", artifact.Type)
}

// TestVerificationIssue tests the VerificationIssue structure
func TestVerificationIssue(t *testing.T) {
	issue := VerificationIssue{
		Type:        "no_translation",
		Description: "Content appears untranslated",
		Location:    "Chapter 1",
		Severity:    "medium",
	}

	assert.Equal(t, "no_translation", issue.Type)
	assert.Equal(t, "Content appears untranslated", issue.Description)
	assert.Equal(t, "Chapter 1", issue.Location)
	assert.Equal(t, "medium", issue.Severity)
}

// TestVerificationResult tests the VerificationResult structure
func TestVerificationResult(t *testing.T) {
	result := VerificationResult{
		IsValid:            false,
		QualityScore:       0.7,
		Score:              0.7, // Alias
		ContextConsidered:  true,
		UntranslatedBlocks: []UntranslatedBlock{
			{
				Location:    "Chapter 1",
				OriginalText: "Hello",
				Language:     "en",
				Length:       5,
			},
		},
		HTMLArtifacts: []HTMLArtifact{
			{
				Content: "<b>Hello</b>",
				Type:    "tag",
			},
		},
		Warnings:     []string{"Test warning"},
		Errors:       []string{"Test error"},
		Issues: []VerificationIssue{
			{
				Type:        "test_issue",
				Description: "Test issue",
			},
		},
		StringIssues: []string{"Test string issue"},
	}

	assert.False(t, result.IsValid)
	assert.Equal(t, 0.7, result.QualityScore)
	assert.Equal(t, 0.7, result.Score)
	assert.True(t, result.ContextConsidered)
	assert.Len(t, result.UntranslatedBlocks, 1)
	assert.Len(t, result.HTMLArtifacts, 1)
	assert.Len(t, result.Warnings, 1)
	assert.Len(t, result.Errors, 1)
	assert.Len(t, result.Issues, 1)
	assert.Len(t, result.StringIssues, 1)
}

// TestNewVerifierWithConfig tests verifier creation with custom config
func TestNewVerifierWithConfig(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	eventBus := events.NewEventBus()
	sessionID := "test-session"
	config := VerificationConfig{
		StrictMode:        true,
		MinQualityScore:   0.9,
		EnableSpellCheck:   true,
		EnableGrammarCheck: true,
	}

	verifier := NewVerifierWithConfig(sourceLang, targetLang, eventBus, sessionID, config)

	assert.NotNil(t, verifier)
	assert.Equal(t, sourceLang, verifier.sourceLanguage)
	assert.Equal(t, targetLang, verifier.targetLanguage)
	assert.Equal(t, eventBus, verifier.eventBus)
	assert.Equal(t, sessionID, verifier.sessionID)
	assert.True(t, verifier.config.StrictMode)
	assert.Equal(t, 0.9, verifier.config.MinQualityScore)
	assert.True(t, verifier.config.EnableSpellCheck)
}

// TestVerifier_CalculateQualityMetrics tests quality metrics calculation
func TestVerifier_CalculateQualityMetrics(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	original := "Hello world, this is a test"
	translated := "Hola mundo, esto es una prueba"

	metrics := verifier.calculateQualityMetrics(original, translated)

	assert.Greater(t, metrics.LengthRatio, 0.0)
	assert.Greater(t, metrics.WordCountRatio, 0.0)
	assert.Greater(t, metrics.VocabularyDiversity, 0.0)
	assert.Greater(t, metrics.Accuracy, 0.0)
	assert.Greater(t, metrics.Fluency, 0.0)
	assert.Greater(t, metrics.Consistency, 0.0)
	assert.Greater(t, metrics.Completeness, 0.0)
	assert.Greater(t, metrics.Overall, 0.0)
}

// TestVerifier_DetectIssues tests issue detection
func TestVerifier_DetectIssues(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	// Test empty translation
	issues := verifier.detectIssues("Hello world", "")
	require.Len(t, issues, 1)
	assert.Equal(t, "empty_translation", issues[0].Type)
	assert.Equal(t, "high", issues[0].Severity)

	// Test untranslated content
	issues = verifier.detectIssues("Hello world", "Hello world")
	require.Len(t, issues, 1)
	assert.Equal(t, "no_translation", issues[0].Type)
	assert.Equal(t, "medium", issues[0].Severity)

	// Test specific length mismatch case
	issues = verifier.detectIssues("This is a very long sentence with many words", "Court")
	require.Len(t, issues, 1)
	assert.Equal(t, "length_mismatch", issues[0].Type)
	assert.Equal(t, "high", issues[0].Severity)

	// Test repetition case
	issues = verifier.detectIssues("Hello", "Hello Hello Hello Hello Hello")
	require.Len(t, issues, 1)
	assert.Equal(t, "repetition", issues[0].Type)
	assert.Equal(t, "low", issues[0].Severity)

	// Test normal translation
	issues = verifier.detectIssues("Hello world", "Hola mundo")
	assert.Empty(t, issues)
}

// TestVerifier_BatchVerify tests batch verification
func TestVerifier_BatchVerify(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	requests := []VerificationRequest{
		{
			Original:   "Hello",
			Translated: "Hola",
			SourceLang: "en",
			TargetLang: "es",
		},
		{
			Original:   "Goodbye",
			Translated: "Adiós",
			SourceLang: "en",
			TargetLang: "es",
		},
	}

	results, err := verifier.BatchVerify(context.Background(), requests)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotNil(t, results[0])
	assert.NotNil(t, results[1])
}

// TestVerifier_BatchVerifyConcurrent tests concurrent batch verification
func TestVerifier_BatchVerifyConcurrent(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	requests := []VerificationRequest{
		{
			Original:   "Hello",
			Translated: "Hola",
			SourceLang: "en",
			TargetLang: "es",
		},
		{
			Original:   "Goodbye",
			Translated: "Adiós",
			SourceLang: "en",
			TargetLang: "es",
		},
	}

	results, err := verifier.BatchVerifyConcurrent(context.Background(), requests)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.NotNil(t, results[0])
	assert.NotNil(t, results[1])
}

// TestVerifier_VerifyWithContext tests verification with context
func TestVerifier_VerifyWithContext(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	original := "Hello world"
	translated := "Hola mundo"
	sourceLangCode := "en"
	targetLangCode := "es"
	contextText := "greeting"

	result, err := verifier.VerifyWithContext(context.Background(), original, translated, sourceLangCode, targetLangCode, contextText)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.ContextConsidered)
}

// TestVerifier_VerifyTranslation_ErrorCases tests verification error cases
func TestVerifier_VerifyTranslation_ErrorCases(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	// Test empty source language
	req := VerificationRequest{
		Original:   "Hello",
		Translated: "Hola",
		SourceLang: "",
		TargetLang: "es",
	}
	_, err := verifier.VerifyTranslation(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source language cannot be empty")

	// Test empty target language
	req = VerificationRequest{
		Original:   "Hello",
		Translated: "Hola",
		SourceLang: "en",
		TargetLang: "",
	}
	_, err = verifier.VerifyTranslation(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target language cannot be empty")

	// Test same source and target language
	req = VerificationRequest{
		Original:   "Hello",
		Translated: "Hola",
		SourceLang: "en",
		TargetLang: "en",
	}
	_, err = verifier.VerifyTranslation(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "same")
}

// TestVerifier_VerifyBook_EmptyBook tests verification of empty book
func TestVerifier_VerifyBook_EmptyBook(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	eventBus := events.NewEventBus()
	verifier := NewVerifier(sourceLang, targetLang, eventBus, "test")

	// Test with empty book
	book := &ebook.Book{}
	result, err := verifier.VerifyBook(context.Background(), book)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestVerifier_VerifyBook_ContentTranslation tests book content verification
func TestVerifier_VerifyBook_ContentTranslation(t *testing.T) {
	// Test with Russian to Serbian which has specific character detection
	sourceLang := language.Language{Code: "ru", Name: "Russian"}
	targetLang := language.Language{Code: "sr", Name: "Serbian"}
	eventBus := events.NewEventBus()
	verifier := NewVerifier(sourceLang, targetLang, eventBus, "test")

	// Create a book with Russian content that should be detected as untranslated
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       "Тестовая книга", // Russian title
			Description: "Тестовое описание",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Глава 1", // Russian title
				Sections: []ebook.Section{
					{
						Title:   "Раздел 1", // Russian title
						Content: "Это тестовый контент на русском языке, который должен быть обнаружен как непереведенный. Содержит русские буквы ъ, ы, э.", // Russian content with specific letters
					},
				},
			},
		},
	}

	result, err := verifier.VerifyBook(context.Background(), book)
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Print result details for debugging
	t.Logf("Errors: %v", result.Errors)
	t.Logf("UntranslatedBlocks: %d", len(result.UntranslatedBlocks))
	t.Logf("Warnings: %d", len(result.Warnings))
	
	// Check that verification actually detected untranslated content
	detectedIssues := len(result.Errors) > 0 || len(result.UntranslatedBlocks) > 0 || len(result.Warnings) > 0
	assert.True(t, detectedIssues, "Should detect untranslated Russian content when translating to Serbian")
}

// TestVerifier_DetectHTMLArtifacts tests HTML artifact detection
func TestVerifier_DetectHTMLArtifacts(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	// Test content with HTML tags
	content := "<p>This is <b>bold</b> content with <a href=\"#\">link</a>.</p>"
	artifacts := verifier.detectHTMLArtifacts(content)

	assert.Greater(t, len(artifacts), 0)
	
	// Check for different types of artifacts
	hasTags := false
	hasEntities := false
	for _, artifact := range artifacts {
		if artifact.Type == "tag" {
			hasTags = true
		}
		if artifact.Type == "entity" {
			hasEntities = true
		}
	}
	assert.True(t, hasTags, "Should detect HTML tags")

	// Test content with HTML entities
	content = "This &amp; that &lt; these &gt; those"
	artifacts = verifier.detectHTMLArtifacts(content)
	assert.Greater(t, len(artifacts), 0)
	
	for _, artifact := range artifacts {
		if artifact.Type == "entity" {
			hasEntities = true
		}
	}
	assert.True(t, hasEntities, "Should detect HTML entities")
}

// TestVerifier_SplitIntoParagraphs tests paragraph splitting
func TestVerifier_SplitIntoParagraphs(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	// Test normal paragraph splitting
	content := "First paragraph.\n\nSecond paragraph.\n\n\nThird paragraph."
	paragraphs := verifier.splitIntoParagraphs(content)
	
	assert.Len(t, paragraphs, 3)
	assert.Equal(t, "First paragraph.", paragraphs[0])
	assert.Equal(t, "Second paragraph.", paragraphs[1])
	assert.Equal(t, "Third paragraph.", paragraphs[2])

	// Test content with no paragraphs
	content = "Single paragraph content."
	paragraphs = verifier.splitIntoParagraphs(content)
	assert.Len(t, paragraphs, 1)

	// Test empty content
	content = ""
	paragraphs = verifier.splitIntoParagraphs(content)
	assert.Len(t, paragraphs, 0)

	// Test content with only whitespace
	content = "\n\n\n   \n\n"
	paragraphs = verifier.splitIntoParagraphs(content)
	assert.Len(t, paragraphs, 0)
}

// TestTruncate tests text truncation
func TestTruncate(t *testing.T) {
	// Test short text (no truncation)
	text := "Short"
	result := truncate(text, 10)
	assert.Equal(t, "Short", result)

	// Test exact length text
	text = "Exactly10"
	result = truncate(text, 10)
	assert.Equal(t, "Exactly10", result)

	// Test long text (with truncation)
	text = "This is a very long text that should be truncated"
	result = truncate(text, 20)
	expected := text[:20] + "..."
	assert.Equal(t, expected, result)
	assert.Len(t, result, 23) // 20 chars + "..."
}

// TestVerifier_CalculateQualityScore tests quality score calculation
func TestVerifier_CalculateQualityScore(t *testing.T) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	// Test with nil book
	result := &VerificationResult{
		IsValid:            true,
		UntranslatedBlocks: []UntranslatedBlock{},
		HTMLArtifacts:      []HTMLArtifact{},
		Warnings:           []string{},
		Errors:             []string{},
	}

	score := verifier.calculateQualityScore(result, nil)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)

	// Test with invalid result
	result.IsValid = false
	score = verifier.calculateQualityScore(result, nil)
	assert.Equal(t, 0.0, score)

	// Test with real book
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       "Test Book",
			Description: "Test Description",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{
						Title:   "Section 1",
						Content: "This is test content.",
					},
				},
			},
		},
	}

	result.IsValid = true
	score = verifier.calculateQualityScore(result, book)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

// Benchmark tests
func BenchmarkVerifier_VerifyTranslation(b *testing.B) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	req := VerificationRequest{
		Original:   "Hello world, this is a test",
		Translated: "Hola mundo, esto es una prueba",
		SourceLang: "en",
		TargetLang: "es",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.VerifyTranslation(context.Background(), req)
	}
}

func BenchmarkVerifier_DetectIssues(b *testing.B) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	original := "This is a long test sentence with many words for benchmarking"
	translated := "Esta es una larga frase de prueba con muchas palabras para benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifier.detectIssues(original, translated)
	}
}

func BenchmarkVerifier_DetectHTMLArtifacts(b *testing.B) {
	sourceLang := language.Language{Code: "en", Name: "English"}
	targetLang := language.Language{Code: "es", Name: "Spanish"}
	verifier := NewVerifier(sourceLang, targetLang, nil, "test")

	content := "<p>This is <b>test content</b> with <a href=\"#\">links</a> and &amp; entities.</p>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifier.detectHTMLArtifacts(content)
	}
}