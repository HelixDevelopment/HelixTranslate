package preparation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SavePreparationResult saves the preparation result to a JSON file
func SavePreparationResult(result *PreparationResult, outputPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadPreparationResult loads a preparation result from a JSON file
func LoadPreparationResult(inputPath string) (*PreparationResult, error) {
	// Read file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON
	var result PreparationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &result, nil
}

// FormatPreparationSummary creates a human-readable summary of the preparation
func FormatPreparationSummary(result *PreparationResult) string {
	var summary string

	summary += "=== PREPARATION ANALYSIS SUMMARY ===\n\n"
	summary += fmt.Sprintf("Languages: %s → %s\n", result.SourceLanguage, result.TargetLanguage)
	summary += fmt.Sprintf("Duration: %.2f seconds\n", result.TotalDuration.Seconds())
	summary += fmt.Sprintf("Passes: %d\n", result.PassCount)
	summary += fmt.Sprintf("Total Tokens: %d\n\n", result.TotalTokens)

	analysis := result.FinalAnalysis

	summary += "--- CONTENT CLASSIFICATION ---\n"
	summary += fmt.Sprintf("Type: %s\n", analysis.ContentType)
	summary += fmt.Sprintf("Genre: %s\n", analysis.Genre)
	if len(analysis.Subgenres) > 0 {
		summary += fmt.Sprintf("Subgenres: %v\n", analysis.Subgenres)
	}
	summary += fmt.Sprintf("Tone: %s\n", analysis.Tone)
	summary += fmt.Sprintf("Target Audience: %s\n\n", analysis.TargetAudience)

	summary += "--- KEY FINDINGS ---\n"
	summary += fmt.Sprintf("Untranslatable Terms: %d\n", len(analysis.UntranslatableTerms))
	summary += fmt.Sprintf("Footnotes Needed: %d\n", len(analysis.FootnoteGuidance))
	summary += fmt.Sprintf("Characters: %d\n", len(analysis.Characters))
	summary += fmt.Sprintf("Cultural References: %d\n", len(analysis.CulturalReferences))
	summary += fmt.Sprintf("Key Themes: %d\n", len(analysis.KeyThemes))
	summary += fmt.Sprintf("Chapters Analyzed: %d\n\n", len(analysis.ChapterAnalyses))

	if len(analysis.KeyThemes) > 0 {
		summary += "--- KEY THEMES ---\n"
		for _, theme := range analysis.KeyThemes {
			summary += fmt.Sprintf("• %s\n", theme)
		}
		summary += "\n"
	}

	if len(analysis.UntranslatableTerms) > 0 {
		summary += "--- UNTRANSLATABLE TERMS (showing first 10) ---\n"
		count := len(analysis.UntranslatableTerms)
		if count > 10 {
			count = 10
		}
		for i := 0; i < count; i++ {
			term := analysis.UntranslatableTerms[i]
			summary += fmt.Sprintf("• %s: %s\n", term.Term, term.Reason)
		}
		if len(analysis.UntranslatableTerms) > 10 {
			summary += fmt.Sprintf("  ... and %d more\n", len(analysis.UntranslatableTerms)-10)
		}
		summary += "\n"
	}

	if len(analysis.Characters) > 0 {
		summary += "--- CHARACTERS ---\n"
		for _, char := range analysis.Characters {
			summary += fmt.Sprintf("• %s (%s)\n", char.Name, char.Role)
			if char.SpeechPattern != "" {
				summary += fmt.Sprintf("  Speech: %s\n", char.SpeechPattern)
			}
		}
		summary += "\n"
	}

	if len(analysis.FootnoteGuidance) > 0 {
		summary += "--- HIGH PRIORITY FOOTNOTES ---\n"
		for _, footnote := range analysis.FootnoteGuidance {
			if footnote.Priority == "high" {
				summary += fmt.Sprintf("• %s\n", footnote.Term)
				summary += fmt.Sprintf("  %s\n", footnote.Explanation)
			}
		}
		summary += "\n"
	}

	summary += "=== END SUMMARY ===\n"

	return summary
}

// GetTranslationContext creates a formatted context string for translators
// This can be passed to the translation LLM as additional context
func GetTranslationContext(analysis *ContentAnalysis, chapterNum int) string {
	var context string

	context += "## TRANSLATION CONTEXT\n\n"

	// Content type and style
	context += fmt.Sprintf("**Content Type**: %s\n", analysis.ContentType)
	context += fmt.Sprintf("**Genre**: %s\n", analysis.Genre)
	context += fmt.Sprintf("**Tone**: %s\n\n", analysis.Tone)

	// Untranslatable terms
	if len(analysis.UntranslatableTerms) > 0 {
		context += "**Terms to Keep in Original**:\n"
		for _, term := range analysis.UntranslatableTerms {
			context += fmt.Sprintf("- %s: %s\n", term.Term, term.Reason)
		}
		context += "\n"
	}

	// Characters for this chapter
	if len(analysis.Characters) > 0 {
		context += "**Character Speech Patterns**:\n"
		for _, char := range analysis.Characters {
			if char.SpeechPattern != "" {
				context += fmt.Sprintf("- %s: %s\n", char.Name, char.SpeechPattern)
			}
		}
		context += "\n"
	}

	// Chapter-specific context
	if chapterNum > 0 && chapterNum <= len(analysis.ChapterAnalyses) {
		chapterAnalysis := analysis.ChapterAnalyses[chapterNum-1]
		context += fmt.Sprintf("**Chapter %d Context**:\n", chapterNum)
		context += fmt.Sprintf("Summary: %s\n", chapterAnalysis.Summary)
		if len(chapterAnalysis.Caveats) > 0 {
			context += "Translation Caveats:\n"
			for _, caveat := range chapterAnalysis.Caveats {
				context += fmt.Sprintf("- %s\n", caveat)
			}
		}
		context += "\n"
	}

	return context
}
