package verification

import (
	"fmt"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/translator"
)

func TestNewPolishingReport(t *testing.T) {
	config := PolishingConfig{
		Providers: []string{"openai", "zhipu", "deepseek"},
		MinConsensus: 2,
		VerifySpirit: true,
		VerifyLanguage: true,
		VerifyContext: true,
		VerifyVocabulary: true,
		TranslationConfigs: map[string]translator.TranslationConfig{},
	}

	report := NewPolishingReport(config)

	// Verify basic report structure
	if len(report.Config.Providers) != len(config.Providers) {
		t.Error("Config providers not properly set")
	}

	for i, provider := range report.Config.Providers {
		if config.Providers[i] != provider {
			t.Errorf("Provider mismatch: expected %s, got %s", config.Providers[i], provider)
		}
	}

	if report.StartTime.IsZero() {
		t.Error("StartTime not set")
	}

	if len(report.SectionResults) != 0 {
		t.Errorf("Expected 0 section results, got %d", len(report.SectionResults))
	}

	if report.TotalSections != 0 {
		t.Errorf("Expected TotalSections=0, got %d", report.TotalSections)
	}

	if report.TotalChanges != 0 {
		t.Errorf("Expected TotalChanges=0, got %d", report.TotalChanges)
	}

	if report.TotalIssues != 0 {
		t.Errorf("Expected TotalIssues=0, got %d", report.TotalIssues)
	}

	// Verify maps are initialized
	if report.IssuesByType == nil {
		t.Error("IssuesByType map not initialized")
	}

	if report.IssuesBySeverity == nil {
		t.Error("IssuesBySeverity map not initialized")
	}

	if report.ProviderAgreements == nil {
		t.Error("ProviderAgreements map not initialized")
	}

	if report.ProviderScores == nil {
		t.Error("ProviderScores map not initialized")
	}
}

func TestPolishingReport_AddSectionResult(t *testing.T) {
	config := PolishingConfig{
		Providers: []string{"openai", "zhipu"},
		MinConsensus: 2,
	}

	report := NewPolishingReport(config)

	// Create test result
	result := &PolishingResult{
		SectionID:       "section-1",
		Location:        "Chapter 1, Page 5",
		OriginalText:    "Original text 1",
		TranslatedText:  "Translated text 1",
		PolishedText:    "Polished text 1",
		SpiritScore:     8.5,
		LanguageScore:   7.8,
		ContextScore:    9.0,
		VocabularyScore: 8.2,
		OverallScore:    8.4,
		Consensus:       2,
		Confidence:      0.85,
		Changes: []Change{
			{
				Location:   "Line 5",
				Original:   "Original text",
				Polished:   "Polished text",
				Reason:     "Grammar correction",
				Agreement:  2,
				Confidence: 0.9,
			},
		},
		Issues: []Issue{
			{
				Type:        "spirit",
				Severity:    "major",
				Description: "Spirit issue detected",
				Location:    "Chapter 1",
				Suggestion:  "Adjust tone",
			},
		},
		Suggestions: []Suggestion{
			{
				Type:        "style",
				Description: "Style improvement suggestion",
				Location:    "Paragraph 2",
				Example:     "Example of improvement",
			},
		},
	}

	// Add result to report
	report.AddSectionResult(result)

	// Verify statistics updated
	if report.TotalSections != 1 {
		t.Errorf("Expected TotalSections=1, got %d", report.TotalSections)
	}

	if report.TotalChanges != 1 {
		t.Errorf("Expected TotalChanges=1, got %d", report.TotalChanges)
	}

	if report.TotalIssues != 1 {
		t.Errorf("Expected TotalIssues=1, got %d", report.TotalIssues)
	}

	if report.TotalSuggestions != 1 {
		t.Errorf("Expected TotalSuggestions=1, got %d", report.TotalSuggestions)
	}

	// Verify section results
	if len(report.SectionResults) != 1 {
		t.Errorf("Expected 1 section result, got %d", len(report.SectionResults))
	}

	if report.SectionResults[0] != result {
		t.Error("Section result not properly added")
	}

	// Verify issue tracking
	if report.IssuesByType["spirit"] != 1 {
		t.Errorf("Expected spirit issues=1, got %d", report.IssuesByType["spirit"])
	}

	if report.IssuesBySeverity["major"] != 1 {
		t.Errorf("Expected major severity issues=1, got %d", report.IssuesBySeverity["major"])
	}

	// Verify significant changes tracking (high confidence)
	if len(report.SignificantChanges) != 1 {
		t.Errorf("Expected 1 significant change, got %d", len(report.SignificantChanges))
	}

	// Verify top issues tracking (critical/major severity)
	if len(report.TopIssues) != 1 {
		t.Errorf("Expected 1 top issue, got %d", len(report.TopIssues))
	}
}

func TestPolishingReport_Finalize(t *testing.T) {
	config := PolishingConfig{
		Providers: []string{"openai", "zhipu", "deepseek"},
		MinConsensus: 2,
	}

	report := NewPolishingReport(config)

	// Add test results
	results := []*PolishingResult{
		{
			SectionID:       "section-1",
			SpiritScore:     8.5,
			LanguageScore:   7.8,
			ContextScore:    9.0,
			VocabularyScore: 8.2,
			OverallScore:    8.4,
			Consensus:       2,
			Confidence:      0.85,
		},
		{
			SectionID:       "section-2",
			SpiritScore:     7.2,
			LanguageScore:   6.8,
			ContextScore:    7.5,
			VocabularyScore: 7.0,
			OverallScore:    7.1,
			Consensus:       1,
			Confidence:      0.72,
		},
		{
			SectionID:       "section-3",
			SpiritScore:     5.5,
			LanguageScore:   6.0,
			ContextScore:    5.8,
			VocabularyScore: 5.2,
			OverallScore:    5.6,
			Consensus:       1,
			Confidence:      0.56,
		},
	}

	for _, result := range results {
		report.AddSectionResult(result)
	}

	// Finalize report
	report.Finalize()

	// Verify timing
	if report.EndTime.IsZero() {
		t.Error("EndTime not set")
	}

	if report.Duration <= 0 {
		t.Error("Duration should be positive")
	}

	// Verify score calculations
	expectedAvgSpirit := (8.5 + 7.2 + 5.5) / 3
	if report.AverageSpiritScore != expectedAvgSpirit {
		t.Errorf("Expected AverageSpiritScore=%.1f, got %.1f", expectedAvgSpirit, report.AverageSpiritScore)
	}

	expectedAvgLanguage := (7.8 + 6.8 + 6.0) / 3
	if diff := report.AverageLanguageScore - expectedAvgLanguage; diff < -0.1 || diff > 0.1 {
		t.Errorf("Expected AverageLanguageScore=%.1f, got %.1f", expectedAvgLanguage, report.AverageLanguageScore)
	}

	expectedAvgContext := (9.0 + 7.5 + 5.8) / 3
	if report.AverageContextScore != expectedAvgContext {
		t.Errorf("Expected AverageContextScore=%.1f, got %.1f", expectedAvgContext, report.AverageContextScore)
	}

	expectedAvgVocabulary := (8.2 + 7.0 + 5.2) / 3
	if report.AverageVocabularyScore != expectedAvgVocabulary {
		t.Errorf("Expected AverageVocabularyScore=%.1f, got %.1f", expectedAvgVocabulary, report.AverageVocabularyScore)
	}

	expectedAvgOverall := (8.4 + 7.1 + 5.6) / 3
	if diff := report.OverallScore - expectedAvgOverall; diff < -0.1 || diff > 0.1 {
		t.Errorf("Expected OverallScore=%.1f, got %.1f", expectedAvgOverall, report.OverallScore)
	}

	// Verify confidence calculations
	expectedAvgConfidence := (0.85 + 0.72 + 0.56) / 3
	if diff := report.AverageConfidence - expectedAvgConfidence; diff < -0.01 || diff > 0.01 {
		t.Errorf("Expected AverageConfidence=%.2f, got %.2f", expectedAvgConfidence, report.AverageConfidence)
	}

	// Verify consensus calculations
	// 1 out of 3 sections meet consensus >= 2 (only the first section has consensus 2)
	expectedConsensusRate := 1.0 / 3.0 * 100.0
	
	if diff := report.ConsensusRate - expectedConsensusRate; diff < -1.0 || diff > 1.0 {
		t.Errorf("Expected ConsensusRate=%.1f, got %.1f", expectedConsensusRate, report.ConsensusRate)
	}
}

func TestPolishingReport_GenerateMarkdownReport(t *testing.T) {
	config := PolishingConfig{
		Providers: []string{"openai", "zhipu", "deepseek"},
		MinConsensus: 2,
		VerifySpirit: true,
		VerifyLanguage: true,
		VerifyContext: false,
		VerifyVocabulary: true,
	}

	report := NewPolishingReport(config)

	// Add test result
	result := &PolishingResult{
		SectionID:       "section-1",
		Location:        "Chapter 1",
		SpiritScore:     8.5,
		LanguageScore:   7.8,
		ContextScore:    9.0,
		VocabularyScore: 8.2,
		OverallScore:    8.4,
		Consensus:       2,
		Confidence:      0.85,
		Changes: []Change{
			{
				Location:   "Line 5",
				Original:   "Original text",
				Polished:   "Polished text",
				Reason:     "Grammar correction",
				Agreement:  2,
				Confidence: 0.9,
			},
		},
		Issues: []Issue{
			{
				Type:        "spirit",
				Severity:    "major",
				Description: "Spirit issue detected",
				Location:    "Chapter 1",
				Suggestion:  "Adjust tone",
			},
		},
	}

	report.AddSectionResult(result)
	report.Finalize()

	// Generate markdown report
	markdown := report.GenerateMarkdownReport()

	// Verify markdown structure
	if len(markdown) == 0 {
		t.Error("Markdown report is empty")
	}

	// Should contain header
	if !strings.Contains(markdown, "# Translation Polishing Report") {
		t.Error("Missing report header")
	}

	// Should contain configuration section
	if !strings.Contains(markdown, "## Configuration") {
		t.Error("Missing configuration section")
	}

	// Should contain providers
	if !strings.Contains(markdown, "openai, zhipu, deepseek") {
		t.Error("Missing provider information")
	}

	// Should contain verification dimensions
	if !strings.Contains(markdown, "Spirit & Tone") {
		t.Error("Missing spirit verification info")
	}

	if !strings.Contains(markdown, "Language Quality") {
		t.Error("Missing language verification info")
	}

	if !strings.Contains(markdown, "Vocabulary Richness") {
		t.Error("Missing vocabulary verification info")
	}

	// Context verification should not be explicitly listed when disabled
	// but may still appear in other sections

	// Should contain summary section
	if !strings.Contains(markdown, "## Executive Summary") {
		t.Error("Missing executive summary section")
	}

	// Should contain statistics
	if !strings.Contains(markdown, "Total Sections Verified") {
		t.Error("Missing total sections")
	}

	if !strings.Contains(markdown, "Consensus Rate") {
		t.Error("Missing consensus rate")
	}

	if !strings.Contains(markdown, "Overall Quality Score") {
		t.Error("Missing overall quality score")
	}
}

func TestPolishingReport_EmptyResults(t *testing.T) {
	config := PolishingConfig{
		Providers: []string{"openai"},
		MinConsensus: 1,
	}

	report := NewPolishingReport(config)

	// Finalize with no results
	report.Finalize()

	// Verify zero averages
	if report.OverallScore != 0 {
		t.Errorf("Expected OverallScore=0, got %f", report.OverallScore)
	}

	if report.AverageConfidence != 0 {
		t.Errorf("Expected AverageConfidence=0, got %f", report.AverageConfidence)
	}

	if report.ConsensusRate != 0 {
		t.Errorf("Expected ConsensusRate=0, got %f", report.ConsensusRate)
	}

	// Should still generate markdown
	markdown := report.GenerateMarkdownReport()
	if len(markdown) == 0 {
		t.Error("Empty results should still generate markdown report")
	}

	if !strings.Contains(markdown, "# Translation Polishing Report") {
		t.Error("Empty report should still have header")
	}
}

func TestPolishingReport_IssueSorting(t *testing.T) {
	config := PolishingConfig{
		Providers: []string{"openai"},
		MinConsensus: 1,
	}

	report := NewPolishingReport(config)

	// Add result with mixed issue severities
	result := &PolishingResult{
		SectionID: "section-1",
		Issues: []Issue{
			{Type: "language", Severity: "minor"},
			{Type: "spirit", Severity: "critical"},
			{Type: "context", Severity: "major"},
			{Type: "vocabulary", Severity: "minor"},
		},
		Changes: []Change{
			{Confidence: 0.9}, // High confidence
			{Confidence: 0.7}, // Medium confidence
			{Confidence: 0.85}, // High confidence
		},
	}

	report.AddSectionResult(result)
	report.Finalize()

	// Verify issue sorting (critical -> major -> minor)
	if len(report.TopIssues) != 2 { // critical and major
		t.Errorf("Expected 2 top issues (critical + major), got %d", len(report.TopIssues))
	}

	if report.TopIssues[0].Severity != "critical" {
		t.Errorf("Expected first issue to be critical, got %s", report.TopIssues[0].Severity)
	}

	if report.TopIssues[1].Severity != "major" {
		t.Errorf("Expected second issue to be major, got %s", report.TopIssues[1].Severity)
	}

	// Verify change sorting by confidence (high to low)
	if len(report.SignificantChanges) != 2 { // >= 0.8 confidence
		t.Errorf("Expected 2 significant changes, got %d", len(report.SignificantChanges))
	}

	if report.SignificantChanges[0].Confidence < report.SignificantChanges[1].Confidence {
		t.Error("Significant changes should be sorted by confidence (high to low)")
	}
}

func TestPolishingReport_Limits(t *testing.T) {
	config := PolishingConfig{
		Providers: []string{"openai"},
		MinConsensus: 1,
	}

	report := NewPolishingReport(config)

	// Add result with many issues and changes
	result := &PolishingResult{
		SectionID: "section-1",
		Issues: make([]Issue, 60), // More than limit of 50
		Changes: make([]Change, 150), // More than limit of 100
	}

	// Make all issues critical/major so they go to top issues
	for i := range result.Issues {
		if i%2 == 0 {
			result.Issues[i].Severity = "critical"
		} else {
			result.Issues[i].Severity = "major"
		}
	}

	// Make all changes high confidence
	for i := range result.Changes {
		result.Changes[i].Confidence = 0.9
	}

	report.AddSectionResult(result)
	report.Finalize()

	// Verify limits applied
	if len(report.TopIssues) > 50 {
		t.Errorf("Top issues limited to 50, got %d", len(report.TopIssues))
	}

	if len(report.SignificantChanges) > 100 {
		t.Errorf("Significant changes limited to 100, got %d", len(report.SignificantChanges))
	}
}

// Benchmark tests
func BenchmarkNewPolishingReport(b *testing.B) {
	config := PolishingConfig{
		Providers: []string{"openai", "zhipu", "deepseek"},
		MinConsensus: 2,
		VerifySpirit: true,
		VerifyLanguage: true,
		VerifyContext: true,
		VerifyVocabulary: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPolishingReport(config)
	}
}

func BenchmarkPolishingReport_AddSectionResult(b *testing.B) {
	config := PolishingConfig{
		Providers: []string{"openai"},
		MinConsensus: 1,
	}

	report := NewPolishingReport(config)

	result := &PolishingResult{
		SectionID:       "section-1",
		SpiritScore:     8.5,
		LanguageScore:   7.8,
		ContextScore:    9.0,
		VocabularyScore: 8.2,
		OverallScore:    8.4,
		Consensus:       1,
		Confidence:      0.85,
		Changes: []Change{
			{Location: "Line 1", Original: "Orig", Polished: "Pol", Reason: "Test", Agreement: 1, Confidence: 0.9},
		},
		Issues: []Issue{
			{Type: "language", Severity: "minor", Description: "Test issue", Location: "Para 1", Suggestion: "Fix"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report.AddSectionResult(result)
	}
}

func BenchmarkPolishingReport_Finalize(b *testing.B) {
	config := PolishingConfig{
		Providers: []string{"openai", "zhipu"},
		MinConsensus: 2,
	}

	// Create report with many results
	report := NewPolishingReport(config)
	
	for i := 0; i < 100; i++ {
		result := &PolishingResult{
			SectionID:       fmt.Sprintf("section-%d", i),
			SpiritScore:     8.5,
			LanguageScore:   7.8,
			ContextScore:    9.0,
			VocabularyScore: 8.2,
			OverallScore:    8.4,
			Consensus:       2,
			Confidence:      0.85,
			Changes: []Change{
				{Location: "Line 1", Original: "Orig", Polished: "Pol", Reason: "Test", Agreement: 2, Confidence: 0.9},
			},
			Issues: []Issue{
				{Type: "language", Severity: "minor", Description: "Test issue", Location: "Para 1", Suggestion: "Fix"},
			},
		}
		report.AddSectionResult(result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid modifying the original
		testReport := *report
		testReport.Finalize()
	}
}

func BenchmarkPolishingReport_GenerateMarkdownReport(b *testing.B) {
	config := PolishingConfig{
		Providers: []string{"openai", "zhipu", "deepseek"},
		MinConsensus: 2,
		VerifySpirit: true,
		VerifyLanguage: true,
		VerifyContext: true,
		VerifyVocabulary: true,
	}

	report := NewPolishingReport(config)

	// Add multiple results
	for i := 0; i < 10; i++ {
		result := &PolishingResult{
			SectionID:       fmt.Sprintf("section-%d", i),
			SpiritScore:     8.5,
			LanguageScore:   7.8,
			ContextScore:    9.0,
			VocabularyScore: 8.2,
			OverallScore:    8.4,
			Consensus:       2,
			Confidence:      0.85,
			Changes: []Change{
				{Location: "Line 1", Original: "Orig", Polished: "Pol", Reason: "Test", Agreement: 2, Confidence: 0.9},
			},
			Issues: []Issue{
				{Type: "language", Severity: "minor", Description: "Test issue", Location: "Para 1", Suggestion: "Fix"},
			},
		}
		report.AddSectionResult(result)
	}

	report.Finalize()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report.GenerateMarkdownReport()
	}
}