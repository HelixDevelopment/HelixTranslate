package preparation

import (
	"strings"
	"testing"
)

func TestNewPreparationPromptBuilder(t *testing.T) {
	sourceLang := "en"
	targetLang := "es"
	passNumber := 2

	builder := NewPreparationPromptBuilder(sourceLang, targetLang, passNumber)

	if builder.sourceLang != sourceLang {
		t.Errorf("Expected sourceLang '%s', got '%s'", sourceLang, builder.sourceLang)
	}
	if builder.targetLang != targetLang {
		t.Errorf("Expected targetLang '%s', got '%s'", targetLang, builder.targetLang)
	}
	if builder.passNumber != passNumber {
		t.Errorf("Expected passNumber %d, got %d", passNumber, builder.passNumber)
	}
	if builder.previousAnalysis != nil {
		t.Error("Expected previousAnalysis to be nil initially")
	}
}

func TestPreparationPromptBuilder_WithPreviousAnalysis(t *testing.T) {
	builder := NewPreparationPromptBuilder("en", "es", 1)

	analysis := &ContentAnalysis{
		ContentType: "fiction",
		Genre:       "science_fiction",
	}

	result := builder.WithPreviousAnalysis(analysis)

	if result != builder {
		t.Error("WithPreviousAnalysis should return the same builder instance")
	}
	if builder.previousAnalysis != analysis {
		t.Error("Previous analysis not set correctly")
	}
}

func TestPreparationPromptBuilder_BuildInitialAnalysisPrompt(t *testing.T) {
	builder := NewPreparationPromptBuilder("en", "es", 1)
	content := "This is a test book content for analysis."

	prompt := builder.BuildInitialAnalysisPrompt(content)

	// Verify prompt contains key elements
	expectedElements := []string{
		"professional translator and literary analyst",
		"en to es",
		"COMPREHENSIVE CONTENT ANALYSIS",
		"CONTENT TO ANALYZE:",
		content,
		"CONTENT CLASSIFICATION",
		"UNTRANSLATABLE TERMS",
		"FOOTNOTE GUIDANCE",
		"CHARACTERS",
		"CULTURAL REFERENCES",
		"OUTPUT FORMAT:",
		"content_type",
		"genre",
		"untranslatable_terms",
		"footnote_guidance",
		"characters",
		"cultural_references",
		"ONLY the JSON output",
	}

	for _, element := range expectedElements {
		if !containsString(prompt, element) {
			t.Errorf("Expected prompt to contain '%s'", element)
		}
	}
}

func TestPreparationPromptBuilder_BuildRefinementPrompt(t *testing.T) {
	builder := NewPreparationPromptBuilder("en", "es", 2)
	content := "This is test content for refinement."

	previousAnalysis := &ContentAnalysis{
		ContentType: "fiction",
		Genre:       "science_fiction",
	}

	builder.WithPreviousAnalysis(previousAnalysis)
	prompt := builder.BuildRefinementPrompt(content)

	// Verify prompt contains key elements
	expectedElements := []string{
		"Pass #2 of content analysis",
		"PREVIOUS ANALYSIS (Pass #1)",
		"CONTENT TO ANALYZE:",
		content,
		"Review and IMPROVE previous analysis",
		"Validation",
		"Completeness",
		"Refinement",
		"Prioritization",
		"Consolidation",
		"ENHANCED version",
		"ONLY the JSON output",
		"fiction",
		"science_fiction",
	}

	for _, element := range expectedElements {
		if !containsString(prompt, element) {
			t.Errorf("Expected refinement prompt to contain '%s'", element)
		}
	}
}

func TestPreparationPromptBuilder_BuildRefinementPrompt_NoPreviousAnalysis(t *testing.T) {
	builder := NewPreparationPromptBuilder("en", "es", 2)
	content := "Test content"

	prompt := builder.BuildRefinementPrompt(content)

	// Should fall back to initial analysis prompt
	if !containsString(prompt, "COMPREHENSIVE CONTENT ANALYSIS") {
		t.Error("Expected fallback to initial analysis prompt")
	}
}

func TestPreparationPromptBuilder_BuildChapterAnalysisPrompt(t *testing.T) {
	builder := NewPreparationPromptBuilder("en", "es", 1)
	chapterNum := 3
	chapterTitle := "The Mysterious Discovery"
	chapterContent := "In this chapter, the protagonist discovers an ancient artifact..."

	prompt := builder.BuildChapterAnalysisPrompt(chapterNum, chapterTitle, chapterContent)

	// Verify prompt contains key elements
	expectedElements := []string{
		"analyzing Chapter 3",
		"en to es",
		"CHAPTER INFORMATION:",
		"**Number**: 3",
		"**Title**: The Mysterious Discovery",
		chapterContent,
		"SUMMARY",
		"KEY POINTS",
		"TRANSLATION CAVEATS",
		"TONE ANALYSIS",
		"COMPLEXITY ASSESSMENT",
		"SPECIAL NOTES",
		"OUTPUT FORMAT:",
		"chapter_id",
		"chapter_num",
		"title",
		"summary",
		"key_points",
		"caveats",
		"tone",
		"complexity",
		"special_notes",
		"ONLY the JSON output",
	}

	for _, element := range expectedElements {
		if !containsString(prompt, element) {
			t.Errorf("Expected chapter analysis prompt to contain '%s'", element)
		}
	}
}

func TestPreparationPromptBuilder_BuildConsolidationPrompt(t *testing.T) {
	builder := NewPreparationPromptBuilder("en", "es", 3)

	analyses := []ContentAnalysis{
		{
			AnalysisVersion: 1,
			AnalyzedBy:      "provider1",
			ContentType:     "fiction",
			Genre:           "science_fiction",
		},
		{
			AnalysisVersion: 2,
			AnalyzedBy:      "provider2",
			ContentType:     "fiction",
			Genre:           "space_opera",
		},
	}

	prompt := builder.BuildConsolidationPrompt(analyses)

	// Verify prompt contains key elements
	expectedElements := []string{
		"FINAL CONSOLIDATED ANALYSIS",
		"ANALYSES TO CONSOLIDATE:",
		"Analysis from Pass #1 (Provider: provider1)",
		"Analysis from Pass #2 (Provider: provider2)",
		"Merging",
		"Validating",
		"Deduplicating",
		"Prioritizing",
		"Clarifying",
		"Organizing",
		"DEFINITIVE, HIGHEST-QUALITY analysis",
		"ONLY the JSON output",
		"science_fiction",
		"space_opera",
	}

	for _, element := range expectedElements {
		if !containsString(prompt, element) {
			t.Errorf("Expected consolidation prompt to contain '%s'", element)
		}
	}
}

func TestTruncateContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxChars int
		expected string
	}{
		{
			name:     "content shorter than limit",
			content:  "Short content",
			maxChars: 100,
			expected: "Short content",
		},
		{
			name:     "content exactly at limit",
			content:  "Exactly 50 characters long content for testing",
			maxChars: 50,
			expected: "Exactly 50 characters long content for testing",
		},
		{
			name:     "content longer than limit with period",
			content:  "This is a longer piece of content. It has multiple sentences. And should be truncated.",
			maxChars: 40,
			expected: "This is a longer piece of content.\n\n[... content truncated for analysis ...]",
		},
		{
			name:     "content longer than limit with newline",
			content:  "First line\n\nSecond line with more content",
			maxChars: 30,
			expected: "First line\n\n[... content truncated for analysis ...]",
		},
		{
			name:     "content longer than limit no sentence boundary",
			content:  "Thisisaverylongwordwithoutspacesorsentenceboundaries",
			maxChars: 20,
			expected: "Thisisaverylongwordw\n\n[... content truncated for analysis ...]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateContent(tt.content, tt.maxChars)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestPreparationPromptBuilder_PromptConsistency(t *testing.T) {
	// Test that prompts are consistent and contain required elements
	builder := NewPreparationPromptBuilder("en", "fr", 1)

	// Test initial prompt
	initialPrompt := builder.BuildInitialAnalysisPrompt("Test content")
	if !containsString(initialPrompt, "en to fr") {
		t.Error("Initial prompt should contain correct language pair")
	}

	// Test chapter prompt
	chapterPrompt := builder.BuildChapterAnalysisPrompt(1, "Test Chapter", "Test content")
	if !containsString(chapterPrompt, "en to fr") {
		t.Error("Chapter prompt should contain correct language pair")
	}

	// Test that all prompts require JSON output
	prompts := []string{
		initialPrompt,
		chapterPrompt,
		builder.BuildRefinementPrompt("Test content"),
		builder.BuildConsolidationPrompt([]ContentAnalysis{}),
	}

	for i, prompt := range prompts {
		if !containsString(prompt, "ONLY the JSON output") {
			t.Errorf("Prompt %d should require JSON-only output", i)
		}
	}
}

func TestPreparationPromptBuilder_DifferentPassNumbers(t *testing.T) {
	content := "Test content"

	// Test different pass numbers
	pass1Builder := NewPreparationPromptBuilder("en", "es", 1)
	pass2Builder := NewPreparationPromptBuilder("en", "es", 2)
	pass3Builder := NewPreparationPromptBuilder("en", "es", 3)

	_ = pass1Builder.BuildInitialAnalysisPrompt(content)
	
	// Set previous analysis for pass 2
	previousAnalysis := &ContentAnalysis{
		AnalysisVersion: 1,
		AnalyzedBy:      "provider1",
		ContentType:     "fiction",
		Genre:           "science_fiction",
	}
	pass2Builder.WithPreviousAnalysis(previousAnalysis)
	
	pass2Prompt := pass2Builder.BuildRefinementPrompt(content)
	pass3Prompt := pass3Builder.BuildConsolidationPrompt([]ContentAnalysis{})

	// Verify pass numbers are mentioned correctly
	if !containsString(pass2Prompt, "Pass #2") {
		t.Error("Pass 2 prompt should mention pass number")
	}
	if !containsString(pass3Prompt, "FINAL CONSOLIDATED ANALYSIS") {
		t.Error("Pass 3 prompt should be for consolidation")
	}
}

func TestPreparationPromptBuilder_ContentHandling(t *testing.T) {
	builder := NewPreparationPromptBuilder("en", "es", 1)

	// Test with very long content
	longContent := strings.Repeat("This is a very long content that should definitely be truncated. ", 500)

	prompt := builder.BuildInitialAnalysisPrompt(longContent)

	// Verify content is included but truncated
	if !containsString(prompt, "This is a very long content") {
		t.Error("Long content should be included in prompt")
	}
	if !containsString(prompt, "[... content truncated for analysis ...]") {
		t.Error("Truncation indicator should be present")
	}
}
