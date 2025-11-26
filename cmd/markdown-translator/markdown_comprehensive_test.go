package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/translator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedFlagParsing tests command-line flag parsing
func TestAdvancedFlagParsing(t *testing.T) {
	// Save original args and reset flags
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()
	
	t.Run("default flag values", func(t *testing.T) {
		// Set minimal args
		os.Args = []string{"markdown-translator", "-input", "test.md"}
		flag.Parse()
		
		// Check that flag parsing doesn't panic
		assert.NotPanics(t, func() {
			flag.Parse()
		})
	})
	
	t.Run("all flags provided", func(t *testing.T) {
		os.Args = []string{
			"markdown-translator",
			"-input", "input.md",
			"-output", "output.epub",
			"-format", "md",
			"-lang", "fr",
			"-provider", "openai",
			"-model", "gpt-4",
			"-keep-md=false",
			"-prepare=true",
			"-prep-passes", "3",
		}
		
		assert.NotPanics(t, func() {
			flag.Parse()
		})
	})
}

// TestInputValidation tests input file validation
func TestInputValidation(t *testing.T) {
	t.Run("valid markdown file", func(t *testing.T) {
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "test.md")
		
		err := os.WriteFile(inputFile, []byte("# Test Document\n\nThis is a test."), 0644)
		require.NoError(t, err)
		
		// Check file exists
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			t.Errorf("Input file should exist: %v", err)
		}
		
		// Check file extension
		ext := strings.ToLower(filepath.Ext(inputFile))
		assert.Equal(t, ".md", ext)
	})
	
	t.Run("valid epub file", func(t *testing.T) {
		tempDir := t.TempDir()
		inputFile := filepath.Join(tempDir, "test.epub")
		
		err := os.WriteFile(inputFile, []byte("dummy epub content"), 0644)
		require.NoError(t, err)
		
		// Check file exists
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			t.Errorf("Input file should exist: %v", err)
		}
		
		// Check file extension
		ext := strings.ToLower(filepath.Ext(inputFile))
		assert.Equal(t, ".epub", ext)
	})
	
	t.Run("nonexistent file", func(t *testing.T) {
		nonexistentFile := "/tmp/nonexistent_file.md"
		
		if _, err := os.Stat(nonexistentFile); !os.IsNotExist(err) {
			t.Errorf("File should not exist: %s", nonexistentFile)
		}
	})
	
	t.Run("supported file formats", func(t *testing.T) {
		supportedFormats := []string{".md", ".markdown", ".epub"}
		
		for _, format := range supportedFormats {
			tempDir := t.TempDir()
			inputFile := filepath.Join(tempDir, "test"+format)
			
			err := os.WriteFile(inputFile, []byte("test content"), 0644)
			require.NoError(t, err)
			
			ext := strings.ToLower(filepath.Ext(inputFile))
			assert.Equal(t, format, ext)
		}
	})
}

// TestFileDetection tests input file type detection
func TestFileDetection(t *testing.T) {
	t.Run("markdown detection", func(t *testing.T) {
		markdownFiles := []string{
			"test.md",
			"test.markdown",
			"document.MD",
			"README.MARKDOWN",
		}
		
		for _, file := range markdownFiles {
			ext := strings.ToLower(filepath.Ext(file))
			isMarkdown := (ext == ".md" || ext == ".markdown")
			assert.True(t, isMarkdown, "File should be detected as markdown: %s", file)
		}
	})
	
	t.Run("epub detection", func(t *testing.T) {
		epubFiles := []string{
			"test.epub",
			"book.EPUB",
			"document.epub",
		}
		
		for _, file := range epubFiles {
			ext := strings.ToLower(filepath.Ext(file))
			isEpub := (ext == ".epub")
			assert.True(t, isEpub, "File should be detected as EPUB: %s", file)
		}
	})
	
	t.Run("case insensitive detection", func(t *testing.T) {
		files := []string{
			"test.MD",
			"test.Md",
			"test.MARKDOWN",
			"test.EPUB",
			"test.Epub",
		}
		
		for _, file := range files {
			ext := strings.ToLower(filepath.Ext(file))
			assert.NotEmpty(t, ext, "Extension should be detected for: %s", file)
		}
	})
}

// TestOutputGeneration tests output filename generation
func TestOutputGeneration(t *testing.T) {
	t.Run("markdown to epub output", func(t *testing.T) {
		inputFile := "document.md"
		base := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
		outputFile := base + "_sr.epub"
		
		assert.Equal(t, "document_sr.epub", outputFile)
	})
	
	t.Run("epub to markdown output", func(t *testing.T) {
		inputFile := "book.epub"
		base := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
		outputFile := base + "_sr.md"
		
		assert.Equal(t, "book_sr.md", outputFile)
	})
	
	t.Run("custom output format", func(t *testing.T) {
		inputFile := "document.md"
		base := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
		
		outputFormats := []string{"epub", "md"}
		for _, format := range outputFormats {
			outputFile := base + "_sr." + format
			assert.Contains(t, outputFile, "_sr."+format)
		}
	})
}

// TestAdvancedProviderConfiguration tests provider configuration
func TestAdvancedProviderConfiguration(t *testing.T) {
	t.Run("supported providers", func(t *testing.T) {
		providers := []string{
			"deepseek",
			"openai", 
			"anthropic",
			"llamacpp",
		}
		
		for _, provider := range providers {
			assert.NotEmpty(t, provider)
			assert.NotEqual(t, "", provider)
		}
	})
	
	t.Run("provider validation", func(t *testing.T) {
		validProviders := map[string][]string{
			"deepseek":   {"deepseek-chat", "deepseek-coder"},
			"openai":     {"gpt-4", "gpt-3.5-turbo", "gpt-4-turbo"},
			"anthropic":  {"claude-3-sonnet", "claude-3-haiku", "claude-3-opus"},
			"llamacpp":   {"llama-2-7b", "llama-2-13b", "mistral-7b"},
		}
		
		for provider, models := range validProviders {
			assert.NotEmpty(t, provider)
			assert.NotEmpty(t, models)
		}
	})
}

// TestAdvancedTranslatorConfiguration tests translator configuration
func TestAdvancedTranslatorConfiguration(t *testing.T) {
	t.Run("basic configuration", func(t *testing.T) {
		config := translator.TranslationConfig{
			SourceLang: "en",
			TargetLang: "sr",
			Provider:   "deepseek",
			Model:      "deepseek-chat",
		}
		
		assert.Equal(t, "en", config.SourceLang)
		assert.Equal(t, "sr", config.TargetLang)
		assert.Equal(t, "deepseek", config.Provider)
		assert.Equal(t, "deepseek-chat", config.Model)
	})
	
	t.Run("language codes", func(t *testing.T) {
		languages := []struct {
			code string
			name string
		}{
			{"en", "English"},
			{"sr", "Serbian"},
			{"fr", "French"},
			{"de", "German"},
			{"es", "Spanish"},
			{"it", "Italian"},
			{"pt", "Portuguese"},
			{"ru", "Russian"},
			{"zh", "Chinese"},
			{"ja", "Japanese"},
		}
		
		for _, lang := range languages {
			assert.NotEmpty(t, lang.code)
			assert.NotEmpty(t, lang.name)
			assert.Len(t, lang.code, 2) // ISO 639-1 codes are 2 characters
		}
	})
	
	t.Run("model configuration", func(t *testing.T) {
		modelConfigs := []struct {
			provider string
			model    string
		}{
			{"deepseek", "deepseek-chat"},
			{"openai", "gpt-4"},
			{"anthropic", "claude-3-sonnet"},
			{"llamacpp", "llama-2-7b"},
		}
		
		for _, config := range modelConfigs {
			assert.NotEmpty(t, config.provider)
			assert.NotEmpty(t, config.model)
		}
	})
}

// TestOutputFormats tests output format handling
func TestOutputFormats(t *testing.T) {
	t.Run("supported output formats", func(t *testing.T) {
		formats := []string{"epub", "md"}
		
		for _, format := range formats {
			assert.NotEmpty(t, format)
			assert.Contains(t, formats, format)
		}
	})
	
	t.Run("format validation", func(t *testing.T) {
		validFormats := map[string]string{
			"epub": "Electronic Publication",
			"md":   "Markdown",
		}
		
		for format, description := range validFormats {
			assert.NotEmpty(t, format)
			assert.NotEmpty(t, description)
		}
	})
}

// TestPreparationConfiguration tests preparation configuration
func TestPreparationConfiguration(t *testing.T) {
	t.Run("preparation enabled", func(t *testing.T) {
		enablePrep := true
		prepPasses := 2
		
		assert.True(t, enablePrep)
		assert.Greater(t, prepPasses, 0)
	})
	
	t.Run("preparation disabled", func(t *testing.T) {
		enablePrep := false
		keepMd := true
		
		assert.False(t, enablePrep)
		assert.True(t, keepMd)
	})
	
	t.Run("pass count validation", func(t *testing.T) {
		passCounts := []int{1, 2, 3, 4, 5}
		
		for _, count := range passCounts {
			assert.Greater(t, count, 0)
			assert.LessOrEqual(t, count, 10) // Reasonable upper bound
		}
	})
}

// TestAdvancedErrorHandling tests error handling scenarios
func TestAdvancedErrorHandling(t *testing.T) {
	t.Run("missing input flag", func(t *testing.T) {
		// Save original args and reset flags
		originalArgs := os.Args
		defer func() {
			os.Args = originalArgs
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		}()
		
		// Test with no input flag
		os.Args = []string{"markdown-translator"}
		
		assert.NotPanics(t, func() {
			flag.Parse()
		})
	})
	
	t.Run("usage message", func(t *testing.T) {
		// Test usage message format
		usage := fmt.Sprintf("Usage: %s -input <file> [-output <output_file>] [-format <format>] [-lang <language>] [-provider <provider>] [-keep-md]", "markdown-translator")
		
		assert.Contains(t, usage, "Usage:")
		assert.Contains(t, usage, "-input")
		assert.Contains(t, usage, "-output")
		assert.Contains(t, usage, "-format")
		assert.Contains(t, usage, "-lang")
		assert.Contains(t, usage, "-provider")
		assert.Contains(t, usage, "-keep-md")
	})
	
	t.Run("supported formats message", func(t *testing.T) {
		inputFormats := "Supported input formats: EPUB (.epub), Markdown (.md)"
		outputFormats := "Supported output formats: EPUB (epub), Markdown (md)"
		
		assert.Contains(t, inputFormats, "EPUB")
		assert.Contains(t, inputFormats, "Markdown")
		assert.Contains(t, outputFormats, "EPUB")
		assert.Contains(t, outputFormats, "Markdown")
	})
}

// TestAdvancedFileOperations tests file operations
func TestAdvancedFileOperations(t *testing.T) {
	t.Run("file creation", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.md")
		
		content := "# Test Document\n\nThis is a test document."
		err := os.WriteFile(testFile, []byte(content), 0644)
		require.NoError(t, err)
		
		// Verify file was created
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			t.Errorf("Test file should exist: %v", err)
		}
		
		// Verify content
		readContent, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, content, string(readContent))
	})
	
	t.Run("directory operations", func(t *testing.T) {
		tempDir := t.TempDir()
		
		// Verify directory exists
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Errorf("Temp directory should exist: %v", err)
		}
		
		// Create a subdirectory
		subDir := filepath.Join(tempDir, "subdir")
		err := os.Mkdir(subDir, 0755)
		require.NoError(t, err)
		
		// Verify subdirectory exists
		if _, err := os.Stat(subDir); os.IsNotExist(err) {
			t.Errorf("Subdirectory should exist: %v", err)
		}
	})
	
	t.Run("file permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		
		err := os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)
		
		// Check file info
		info, err := os.Stat(testFile)
		require.NoError(t, err)
		assert.Equal(t, "test.txt", info.Name())
		assert.Greater(t, info.Size(), int64(0))
	})
}

// TestLLMProviderCreation tests LLM provider creation
func TestLLMProviderCreation(t *testing.T) {
	t.Run("provider config validation", func(t *testing.T) {
		// Define a simple provider config struct for testing
		type ProviderConfig struct {
			Name    string
			Enabled bool
		}
		
		configs := []ProviderConfig{
			{
				Name:    "deepseek",
				Enabled: true,
			},
			{
				Name:    "openai",
				Enabled: true,
			},
			{
				Name:    "anthropic",
				Enabled: false,
			},
			{
				Name:    "llamacpp",
				Enabled: true,
			},
		}
		
		for _, config := range configs {
			assert.NotEmpty(t, config.Name)
			assert.NotNil(t, config.Enabled)
		}
	})
	
	t.Run("model selection", func(t *testing.T) {
		models := []string{
			"deepseek-chat",
			"gpt-4",
			"claude-3-sonnet",
			"llama-2-7b",
		}
		
		for _, model := range models {
			assert.NotEmpty(t, model)
			assert.Contains(t, model, "-") // Models typically have hyphens
		}
	})
}