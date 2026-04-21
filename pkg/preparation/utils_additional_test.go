package preparation

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavePreparationResult(t *testing.T) {
	t.Run("Save valid result", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "preparation.json")

		result := &PreparationResult{
			SourceLanguage: "en",
			TargetLanguage: "es",
			PassCount:      2,
			TotalTokens:    1000,
			TotalDuration:  5 * time.Minute,
			StartedAt:      time.Now(),
			CompletedAt:    time.Now(),
			Passes: []PreparationPass{
				{
					PassNumber: 1,
					Provider:   "openai",
					Analysis: ContentAnalysis{
						ContentType: "fiction",
						Genre:       "science_fiction",
					},
				},
			},
			FinalAnalysis: ContentAnalysis{
				ContentType: "fiction",
				Genre:       "science_fiction",
				Characters: []Character{
					{Name: "John", Role: "protagonist"},
				},
			},
		}

		err := SavePreparationResult(result, outputPath)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(outputPath)
		require.NoError(t, err)

		// Verify it can be loaded back
		loaded, err := LoadPreparationResult(outputPath)
		require.NoError(t, err)
		assert.Equal(t, result.SourceLanguage, loaded.SourceLanguage)
		assert.Equal(t, result.TargetLanguage, loaded.TargetLanguage)
		assert.Equal(t, result.PassCount, loaded.PassCount)
		assert.Equal(t, result.FinalAnalysis.ContentType, loaded.FinalAnalysis.ContentType)
		assert.Equal(t, result.FinalAnalysis.Genre, loaded.FinalAnalysis.Genre)
		assert.Len(t, loaded.FinalAnalysis.Characters, 1)
	})

	t.Run("Save creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		deepPath := filepath.Join(tmpDir, "a", "b", "c", "preparation.json")

		result := &PreparationResult{
			SourceLanguage: "en",
			TargetLanguage: "es",
		}

		err := SavePreparationResult(result, deepPath)
		require.NoError(t, err)

		_, err = os.Stat(deepPath)
		require.NoError(t, err)
	})

	t.Run("Save to invalid path returns error", func(t *testing.T) {
		result := &PreparationResult{
			SourceLanguage: "en",
			TargetLanguage: "es",
		}

		err := SavePreparationResult(result, "/invalid/path/that/does/not/exist/preparation.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create directory")
	})
}

func TestLoadPreparationResult(t *testing.T) {
	t.Run("Load existing valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputPath := filepath.Join(tmpDir, "preparation.json")

		// Create a valid JSON file
		jsonData := `{
			"source_language": "ru",
			"target_language": "sr",
			"pass_count": 2,
			"total_tokens": 5000,
			"passes": [
				{
					"pass_number": 1,
					"provider": "openai",
					"analysis": {
						"content_type": "novel",
						"genre": "literary_fiction"
					}
				}
			],
			"final_analysis": {
				"content_type": "novel",
				"genre": "literary_fiction",
				"untranslatable_terms": [
					{"term": "Borscht", "reason": "Traditional food"}
				]
			}
		}`
		err := os.WriteFile(inputPath, []byte(jsonData), 0644)
		require.NoError(t, err)

		result, err := LoadPreparationResult(inputPath)
		require.NoError(t, err)
		assert.Equal(t, "ru", result.SourceLanguage)
		assert.Equal(t, "sr", result.TargetLanguage)
		assert.Equal(t, 2, result.PassCount)
		assert.Equal(t, 5000, result.TotalTokens)
		assert.Equal(t, "novel", result.FinalAnalysis.ContentType)
		assert.Len(t, result.FinalAnalysis.UntranslatableTerms, 1)
		assert.Equal(t, "Borscht", result.FinalAnalysis.UntranslatableTerms[0].Term)
	})

	t.Run("Load non-existent file returns error", func(t *testing.T) {
		_, err := LoadPreparationResult("/nonexistent/path/file.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("Load invalid JSON returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputPath := filepath.Join(tmpDir, "invalid.json")

		err := os.WriteFile(inputPath, []byte("not valid json"), 0644)
		require.NoError(t, err)

		_, err = LoadPreparationResult(inputPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal JSON")
	})

	t.Run("Load empty JSON object", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputPath := filepath.Join(tmpDir, "empty.json")

		err := os.WriteFile(inputPath, []byte("{}"), 0644)
		require.NoError(t, err)

		result, err := LoadPreparationResult(inputPath)
		require.NoError(t, err)
		assert.Equal(t, "", result.SourceLanguage)
		assert.Equal(t, "", result.TargetLanguage)
		assert.Empty(t, result.Passes)
	})
}

func TestFormatPreparationSummary(t *testing.T) {
	t.Run("Full summary with all fields", func(t *testing.T) {
		result := &PreparationResult{
			SourceLanguage: "ru",
			TargetLanguage: "sr",
			TotalDuration:  10 * time.Minute,
			PassCount:      2,
			TotalTokens:    5000,
			FinalAnalysis: ContentAnalysis{
				ContentType:    "Novel",
				Genre:          "Literary Fiction",
				Subgenres:      []string{"Psychological", "Historical"},
				Tone:           "Melancholic",
				TargetAudience: "Adult readers",
				UntranslatableTerms: []UntranslatableTerm{
					{Term: "Borscht", Reason: "Traditional soup"},
					{Term: "Matryoshka", Reason: "Cultural icon"},
					{Term: "Dacha", Reason: "Unique concept"},
					{Term: "Samovar", Reason: "Tea tradition"},
					{Term: "Balalaika", Reason: "Musical instrument"},
					{Term: "Babushka", Reason: "Family role"},
					{Term: "Tovarisch", Reason: "Historical term"},
					{Term: "Peristroika", Reason: "Historical event"},
					{Term: "Glasnost", Reason: "Policy"},
					{Term: "Troika", Reason: "Vehicle/Concept"},
					{Term: "Kremlin", Reason: "Landmark"},
					{Term: "Soviet", Reason: "Historical"},
				},
				FootnoteGuidance: []FootnoteGuidance{
					{Term: "Dacha", Explanation: "Country house", Priority: "high"},
					{Term: "Soviet", Explanation: "Council", Priority: "medium"},
				},
				Characters: []Character{
					{Name: "Ivan", Role: "Protagonist", SpeechPattern: "Formal"},
					{Name: "Maria", Role: "Antagonist", SpeechPattern: "Casual"},
				},
				KeyThemes: []string{"Redemption", "Love", "War", "Peace"},
				CulturalReferences: []CulturalReference{
					{Reference: "Red Square", Origin: "Russian"},
				},
				ChapterAnalyses: []ChapterAnalysis{
					{ChapterNum: 1, Summary: "Introduction"},
					{ChapterNum: 2, Summary: "Conflict"},
				},
			},
		}

		summary := FormatPreparationSummary(result)

		assert.Contains(t, summary, "PREPARATION ANALYSIS SUMMARY")
		assert.Contains(t, summary, "ru → sr")
		assert.Contains(t, summary, "Novel")
		assert.Contains(t, summary, "Literary Fiction")
		assert.Contains(t, summary, "Melancholic")
		assert.Contains(t, summary, "Borscht")
		assert.Contains(t, summary, "Ivan")
		assert.Contains(t, summary, "Maria")
		assert.Contains(t, summary, "Redemption")
		assert.Contains(t, summary, "Cultural References: 1")
		assert.Contains(t, summary, "Dacha")
		assert.Contains(t, summary, "... and 2 more")
	})

	t.Run("Minimal summary", func(t *testing.T) {
		result := &PreparationResult{
			SourceLanguage: "en",
			TargetLanguage: "es",
			TotalDuration:  0,
			PassCount:      0,
			TotalTokens:    0,
			FinalAnalysis:  ContentAnalysis{},
		}

		summary := FormatPreparationSummary(result)

		assert.Contains(t, summary, "en → es")
		assert.Contains(t, summary, "Duration: 0.00 seconds")
		assert.Contains(t, summary, "Passes: 0")
	})

	t.Run("Summary with empty analysis fields", func(t *testing.T) {
		result := &PreparationResult{
			SourceLanguage: "en",
			TargetLanguage: "fr",
			TotalDuration:  5 * time.Minute,
			PassCount:      1,
			FinalAnalysis: ContentAnalysis{
				ContentType: "Unknown",
				Genre:       "Unknown",
			},
		}

		summary := FormatPreparationSummary(result)

		assert.Contains(t, summary, "en → fr")
		assert.Contains(t, summary, "Type: Unknown")
		assert.NotContains(t, summary, "--- KEY THEMES")
		assert.NotContains(t, summary, "--- CHARACTERS")
	})
}

func TestGetTranslationContext(t *testing.T) {
	t.Run("Full context with chapter analysis", func(t *testing.T) {
		analysis := &ContentAnalysis{
			ContentType: "Novel",
			Genre:       "Fiction",
			Tone:        "Formal",
			UntranslatableTerms: []UntranslatableTerm{
				{Term: "Borscht", Reason: "Food name"},
			},
			Characters: []Character{
				{Name: "Ivan", SpeechPattern: "Formal Russian"},
			},
			ChapterAnalyses: []ChapterAnalysis{
				{ChapterNum: 1, Summary: "Introduction", Caveats: []string{"Complex metaphor"}},
				{ChapterNum: 2, Summary: "Rising action", Caveats: []string{"Historical reference"}},
			},
		}

		context := GetTranslationContext(analysis, 1)

		assert.Contains(t, context, "Novel")
		assert.Contains(t, context, "Fiction")
		assert.Contains(t, context, "Formal")
		assert.Contains(t, context, "Borscht")
		assert.Contains(t, context, "Ivan")
		assert.Contains(t, context, "Chapter 1 Context")
		assert.Contains(t, context, "Introduction")
		assert.Contains(t, context, "Complex metaphor")
	})

	t.Run("Context without chapter analysis", func(t *testing.T) {
		analysis := &ContentAnalysis{
			ContentType: "Technical",
			Genre:       "Documentation",
			Tone:        "Technical",
		}

		context := GetTranslationContext(analysis, 1)

		assert.Contains(t, context, "Technical")
		assert.NotContains(t, context, "Chapter 1 Context")
	})

	t.Run("Context with chapter number out of range", func(t *testing.T) {
		analysis := &ContentAnalysis{
			ContentType: "Novel",
			Genre:       "Fiction",
			ChapterAnalyses: []ChapterAnalysis{
				{ChapterNum: 1, Summary: "Chapter 1"},
			},
		}

		context := GetTranslationContext(analysis, 5)

		assert.Contains(t, context, "Novel")
		assert.NotContains(t, context, "Chapter 5 Context")
	})

	t.Run("Context with zero chapter number", func(t *testing.T) {
		analysis := &ContentAnalysis{
			ContentType: "Poetry",
			Genre:       "Epic",
			ChapterAnalyses: []ChapterAnalysis{
				{ChapterNum: 1, Summary: "Canto 1"},
			},
		}

		context := GetTranslationContext(analysis, 0)

		assert.Contains(t, context, "Poetry")
		assert.NotContains(t, context, "Chapter 0 Context")
	})

	t.Run("Context with characters but no speech patterns", func(t *testing.T) {
		analysis := &ContentAnalysis{
			ContentType: "Novel",
			Characters: []Character{
				{Name: "John", SpeechPattern: ""},
				{Name: "Jane", SpeechPattern: "Casual"},
			},
		}

		context := GetTranslationContext(analysis, 1)

		assert.Contains(t, context, "Jane")
		assert.Contains(t, context, "Casual")
		// John has empty speech pattern, so should not appear
		assert.NotContains(t, context, "John")
	})
}
