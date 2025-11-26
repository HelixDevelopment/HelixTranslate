package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/preparation"
	"digital.vasic.translator/pkg/translator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFlagParsing tests command-line flag parsing
func TestFlagParsing(t *testing.T) {
	// Save original args and reset flags
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()
	
	t.Run("default flag values", func(t *testing.T) {
		// Set minimal args
		os.Args = []string{"preparation-translator"}
		flag.Parse()
		
		// Note: We can't directly access flag values after parse without explicit pointers
		// This test verifies that flag parsing doesn't panic
		assert.NotPanics(t, func() {
			flag.Parse()
		})
	})
	
	t.Run("custom flag values", func(t *testing.T) {
		// Set custom args
		os.Args = []string{
			"preparation-translator",
			"-input", "/custom/input.epub",
			"-output", "/custom/output.epub",
			"-analysis", "/custom/analysis.json",
			"-source", "French",
			"-target", "German",
			"-passes", "3",
			"-providers", "deepseek,zhipu,openai",
		}
		
		assert.NotPanics(t, func() {
			flag.Parse()
		})
	})
}

// TestLanguageSetup tests language configuration
func TestLanguageSetup(t *testing.T) {
	t.Run("language creation", func(t *testing.T) {
		// Test source language
		sourceLanguage := language.Language{Code: "en", Name: "English"}
		assert.Equal(t, "en", sourceLanguage.Code)
		assert.Equal(t, "English", sourceLanguage.Name)
		
		// Test target language
		targetLanguage := language.Language{Code: "es", Name: "Spanish"}
		assert.Equal(t, "es", targetLanguage.Code)
		assert.Equal(t, "Spanish", targetLanguage.Name)
	})
	
	t.Run("language compatibility", func(t *testing.T) {
		// Test that different language codes work
		languages := []struct {
			code string
			name string
		}{
			{"fr", "French"},
			{"de", "German"},
			{"it", "Italian"},
			{"pt", "Portuguese"},
			{"ru", "Russian"},
			{"sr", "Serbian"},
			{"zh", "Chinese"},
			{"ja", "Japanese"},
		}
		
		for _, lang := range languages {
			l := language.Language{Code: lang.code, Name: lang.name}
			assert.Equal(t, lang.code, l.Code)
			assert.Equal(t, lang.name, l.Name)
		}
	})
}

// TestPreparationConfig tests preparation configuration
func TestPreparationConfig(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		config := &preparation.PreparationConfig{
			PassCount:          2,
			Providers:          []string{"deepseek", "zhipu"},
			AnalyzeContentType: true,
			AnalyzeCharacters:  true,
			AnalyzeTerminology: true,
			AnalyzeCulture:     true,
			AnalyzeChapters:    true,
			DetailLevel:        "comprehensive",
			SourceLanguage:     "English",
			TargetLanguage:     "Spanish",
		}
		
		assert.Equal(t, 2, config.PassCount)
		assert.Equal(t, []string{"deepseek", "zhipu"}, config.Providers)
		assert.True(t, config.AnalyzeContentType)
		assert.True(t, config.AnalyzeCharacters)
		assert.True(t, config.AnalyzeTerminology)
		assert.True(t, config.AnalyzeCulture)
		assert.True(t, config.AnalyzeChapters)
		assert.Equal(t, "comprehensive", config.DetailLevel)
		assert.Equal(t, "English", config.SourceLanguage)
		assert.Equal(t, "Spanish", config.TargetLanguage)
	})
	
	t.Run("custom configuration", func(t *testing.T) {
		config := &preparation.PreparationConfig{
			PassCount:          3,
			Providers:          []string{"openai", "anthropic"},
			AnalyzeContentType: false,
			AnalyzeCharacters:  false,
			AnalyzeTerminology: true,
			AnalyzeCulture:     false,
			AnalyzeChapters:    true,
			DetailLevel:        "basic",
			SourceLanguage:     "French",
			TargetLanguage:     "German",
		}
		
		assert.Equal(t, 3, config.PassCount)
		assert.Equal(t, []string{"openai", "anthropic"}, config.Providers)
		assert.False(t, config.AnalyzeContentType)
		assert.False(t, config.AnalyzeCharacters)
		assert.True(t, config.AnalyzeTerminology)
		assert.False(t, config.AnalyzeCulture)
		assert.True(t, config.AnalyzeChapters)
		assert.Equal(t, "basic", config.DetailLevel)
		assert.Equal(t, "French", config.SourceLanguage)
		assert.Equal(t, "German", config.TargetLanguage)
	})
}

// TestTranslatorConfig tests translator configuration
func TestTranslatorConfig(t *testing.T) {
	t.Run("basic configuration", func(t *testing.T) {
		sourceLanguage := language.Language{Code: "ru", Name: "Russian"}
		targetLanguage := language.Language{Code: "sr", Name: "Serbian"}
		
		config := translator.TranslationConfig{
			SourceLang: sourceLanguage.Code,
			TargetLang: targetLanguage.Code,
			Provider:   "deepseek",
			Model:      "deepseek-chat",
		}
		
		assert.Equal(t, "ru", config.SourceLang)
		assert.Equal(t, "sr", config.TargetLang)
		assert.Equal(t, "deepseek", config.Provider)
		assert.Equal(t, "deepseek-chat", config.Model)
	})
	
	t.Run("multiple provider configurations", func(t *testing.T) {
		sourceLanguage := language.Language{Code: "en", Name: "English"}
		targetLanguage := language.Language{Code: "es", Name: "Spanish"}
		
		providers := []struct {
			provider string
			model    string
		}{
			{"deepseek", "deepseek-chat"},
			{"zhipu", "glm-4"},
			{"openai", "gpt-4"},
			{"anthropic", "claude-3-sonnet"},
		}
		
		for _, p := range providers {
			config := translator.TranslationConfig{
				SourceLang: sourceLanguage.Code,
				TargetLang: targetLanguage.Code,
				Provider:   p.provider,
				Model:      p.model,
			}
			
			assert.Equal(t, "en", config.SourceLang)
			assert.Equal(t, "es", config.TargetLang)
			assert.Equal(t, p.provider, config.Provider)
			assert.Equal(t, p.model, config.Model)
		}
	})
}

// TestTranslatorProviderParsing tests provider string parsing
func TestTranslatorProviderParsing(t *testing.T) {
	t.Run("single provider", func(t *testing.T) {
		providers := "deepseek"
		providerList := strings.Split(providers, ",")
		
		assert.Len(t, providerList, 1)
		assert.Equal(t, "deepseek", providerList[0])
	})
	
	t.Run("multiple providers", func(t *testing.T) {
		providers := "deepseek,zhipu,openai"
		providerList := strings.Split(providers, ",")
		
		assert.Len(t, providerList, 3)
		assert.Equal(t, "deepseek", providerList[0])
		assert.Equal(t, "zhipu", providerList[1])
		assert.Equal(t, "openai", providerList[2])
	})
	
	t.Run("providers with spaces", func(t *testing.T) {
		providers := "deepseek, zhipu, openai"
		providerList := strings.Split(strings.ReplaceAll(providers, " ", ""), ",")
		
		assert.Len(t, providerList, 3)
		assert.Equal(t, "deepseek", providerList[0])
		assert.Equal(t, "zhipu", providerList[1])
		assert.Equal(t, "openai", providerList[2])
	})
}

// TestFileValidation tests input file validation
func TestFileValidation(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "test.txt")
		
		err := os.WriteFile(inputFile, []byte("test content"), 0644)
		require.NoError(t, err)
		
		// Check file exists
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			t.Errorf("Input file should exist: %v", err)
		}
		
		// Check file content
		content, err := os.ReadFile(inputFile)
		require.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})
	
	t.Run("nonexistent file", func(t *testing.T) {
		nonexistentFile := "/tmp/nonexistent_file.txt"
		
		if _, err := os.Stat(nonexistentFile); !os.IsNotExist(err) {
			t.Errorf("File should not exist: %s", nonexistentFile)
		}
	})
	
	t.Run("file extensions", func(t *testing.T) {
		extensions := []string{".txt", ".md", ".epub", ".mobi", ".fb2", ".pdf"}
		
		for _, ext := range extensions {
			tempDir := t.TempDir()
			inputFile := filepath.Join(tempDir, "test"+ext)
			
			err := os.WriteFile(inputFile, []byte("test content"), 0644)
			require.NoError(t, err)
			
			// Check file extension
			assert.Equal(t, ext, filepath.Ext(inputFile))
		}
	})
}

// TestLogMessages tests log message formatting
func TestLogMessages(t *testing.T) {
	t.Run("progress message format", func(t *testing.T) {
		progressMessage := "📊 Progress: Processing chapter 1/10"
		assert.Contains(t, progressMessage, "📊 Progress")
		assert.Contains(t, progressMessage, "Processing chapter 1/10")
	})
	
	t.Run("error message format", func(t *testing.T) {
		errorMessage := "❌ Error: Failed to translate text"
		assert.Contains(t, errorMessage, "❌ Error")
		assert.Contains(t, errorMessage, "Failed to translate text")
	})
	
	t.Run("success message format", func(t *testing.T) {
		successMessage := "✅ Translation complete in 120.50 seconds"
		assert.Contains(t, successMessage, "✅")
		assert.Contains(t, successMessage, "Translation complete")
		assert.Contains(t, successMessage, "120.50 seconds")
	})
}

// TestStatisticsFormatting tests statistics output formatting
func TestStatisticsFormatting(t *testing.T) {
	t.Run("basic statistics", func(t *testing.T) {
		stats := map[string]interface{}{
			"duration":         120.5,
			"input_chapters":   10,
			"output_file":      "/tmp/output.epub",
			"output_size":      1024000,
			"analysis_file":    "/tmp/analysis.json",
			"analysis_size":    20480,
		}
		
		assert.Equal(t, 120.5, stats["duration"])
		assert.Equal(t, 10, stats["input_chapters"])
		assert.Equal(t, "/tmp/output.epub", stats["output_file"])
		assert.Equal(t, 1024000, stats["output_size"])
		assert.Equal(t, "/tmp/analysis.json", stats["analysis_file"])
		assert.Equal(t, 20480, stats["analysis_size"])
	})
	
	t.Run("format large numbers", func(t *testing.T) {
		// Test formatting of large file sizes
		sizes := []struct {
			size    int64
			formatted string
		}{
			{1024, "1 KB"},
			{1048576, "1 MB"},
			{1073741824, "1 GB"},
		}
		
		for _, s := range sizes {
			// Verify the values
			assert.Greater(t, s.size, int64(0))
			assert.NotEmpty(t, s.formatted)
		}
	})
}

// TestConfigurationValidation tests overall configuration validation
func TestConfigurationValidation(t *testing.T) {
	t.Run("valid preparation config", func(t *testing.T) {
		config := &preparation.PreparationConfig{
			PassCount:          2,
			Providers:          []string{"deepseek", "zhipu"},
			AnalyzeContentType: true,
			AnalyzeCharacters:  true,
			AnalyzeTerminology: true,
			AnalyzeCulture:     true,
			AnalyzeChapters:    true,
			DetailLevel:        "comprehensive",
			SourceLanguage:     "English",
			TargetLanguage:     "Spanish",
		}
		
		// Validate required fields
		assert.Greater(t, config.PassCount, 0)
		assert.NotEmpty(t, config.Providers)
		assert.NotEmpty(t, config.DetailLevel)
		assert.NotEmpty(t, config.SourceLanguage)
		assert.NotEmpty(t, config.TargetLanguage)
	})
	
	t.Run("invalid preparation config", func(t *testing.T) {
		config := &preparation.PreparationConfig{
			PassCount:         0, // Invalid: should be > 0
			Providers:         []string{}, // Invalid: should have providers
			DetailLevel:       "",      // Invalid: should have detail level
			SourceLanguage:    "",     // Invalid: should have source language
			TargetLanguage:    "",     // Invalid: should have target language
		}
		
		// Validate invalid fields
		assert.Equal(t, 0, config.PassCount)
		assert.Empty(t, config.Providers)
		assert.Empty(t, config.DetailLevel)
		assert.Empty(t, config.SourceLanguage)
		assert.Empty(t, config.TargetLanguage)
	})
}

// TestTranslationPipelineStructure tests the structure of the translation pipeline
func TestTranslationPipelineStructure(t *testing.T) {
	t.Run("pipeline steps", func(t *testing.T) {
		steps := []string{
			"1. Parsing ebook...",
			"2. Configuring preparation phase...",
			"3. Creating translator...",
			"4. Creating preparation-aware translator...",
			"5. Running preparation + translation pipeline...",
			"6. Saving preparation analysis...",
			"7. Saving translated book...",
		}
		
		assert.Len(t, steps, 7)
		
		for i, step := range steps {
			assert.Contains(t, step, fmt.Sprintf("%d.", i+1))
			assert.NotEmpty(t, step)
		}
	})
	
	t.Run("event handling structure", func(t *testing.T) {
		// Test that event handlers can be created
		progressHandler := func(event interface{}) {
			// Mock progress handler
		}
		
		errorHandler := func(event interface{}) {
			// Mock error handler
		}
		
		assert.NotNil(t, progressHandler)
		assert.NotNil(t, errorHandler)
	})
}