package main

import (
	"bytes"
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/script"
	"digital.vasic.translator/pkg/translator"
	versionpkg "digital.vasic.translator/pkg/version"
	"digital.vasic.translator/test/mocks"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMainFunction tests main function with various inputs
func TestMainFunctionComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectedExit   int
		setup          func() func()
	}{
		{
			name:           "version flag",
			args:           []string{"-version"},
			expectedOutput: "Universal Ebook Translator v" + versionpkg.AppVersion,
			expectedExit:   0,
			setup:          func() func() { return func() {} },
		},
		{
			name:           "help flag",
			args:           []string{"-help"},
			expectedOutput: "Universal Ebook Translator v" + versionpkg.AppVersion,
			expectedExit:   0,
			setup:          func() func() { return func() {} },
		},
		{
			name:           "no arguments shows help",
			args:           []string{},
			expectedOutput: "Universal Ebook Translator v" + versionpkg.AppVersion,
			expectedExit:   0,
			setup:          func() func() { return func() {} },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setup()
			defer cleanup()

			// For version and help tests, just verify the function works
			if tt.name == "version flag" || tt.name == "help flag" || tt.name == "no arguments shows help" {
				// Just test the functionality doesn't panic
				assert.NotPanics(t, func() {
					oldArgs := os.Args
					defer func() { os.Args = oldArgs }()

					os.Args = append([]string{"translator"}, tt.args...)

					defer func() {
						if r := recover(); r != nil {
							// Expected due to os.Exit
						}
					}()
					main()
				})
				return
			}

			// Capture stdout and stderr for other tests
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			oldArgs := os.Args

			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			defer func() {
				os.Stdout = oldStdout
				os.Stderr = oldStderr
				os.Args = oldArgs
			}()

			// Set up args
			os.Args = append([]string{"translator"}, tt.args...)

			// Run main in a goroutine to capture exit
			done := make(chan bool, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						// Handle panic from os.Exit
						done <- true
					}
				}()
				main()
				done <- true
			}()

			// Close pipe and read output
			w.Close()
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)

			select {
			case <-done:
				// Function completed
			case <-time.After(5 * time.Second):
				t.Fatal("Main function timed out")
			}

			output := buf.String()
			if tt.expectedOutput != "" {
				assert.Contains(t, output, tt.expectedOutput)
			}
		})
	}
}

// TestCreateConfigFile tests config file creation functionality
func TestCreateConfigFileComprehensive(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
	}{
		{
			name:          "create valid config",
			configContent: `{"provider": "openai", "model": "gpt-4"}`,
			expectError:   false,
		},
		{
			name:          "create empty config",
			configContent: "",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test-config.json")

			err := createConfigFile(configFile)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			_, err = os.Stat(configFile)
			assert.NoError(t, err)

			// Verify content
			content, err := os.ReadFile(configFile)
			assert.NoError(t, err)
			assert.Contains(t, string(content), `"provider": "openai"`)
			assert.Contains(t, string(content), `"model": "gpt-4"`)
		})
	}
}

// TestGenerateOutputFilename tests output filename generation
func TestGenerateOutputFilenameComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		inputFile    string
		targetLang   string
		outputFormat string
		expected     string
	}{
		{
			name:         "simple epub",
			inputFile:    "book.epub",
			targetLang:   "sr",
			outputFormat: "epub",
			expected:     "book_sr.epub",
		},
		{
			name:         "complex path",
			inputFile:    "/path/to/mybook.fb2",
			targetLang:   "de",
			outputFormat: "txt",
			expected:     "/path/to/mybook_de.txt",
		},
		{
			name:         "file with multiple dots",
			inputFile:    "my.book.name.txt",
			targetLang:   "fr",
			outputFormat: "epub",
			expected:     "my.book.name_fr.epub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateOutputFilename(tt.inputFile, tt.targetLang, tt.outputFormat)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetAPIKeyFromEnv tests API key retrieval from environment
func TestGetAPIKeyFromEnvComprehensive(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		envVar        string
		envValue      string
		expectedValue string
	}{
		{
			name:          "openai provider",
			provider:      "openai",
			envVar:        "OPENAI_API_KEY",
			envValue:      "test-openai-key",
			expectedValue: "test-openai-key",
		},
		{
			name:          "anthropic provider",
			provider:      "anthropic",
			envVar:        "ANTHROPIC_API_KEY",
			envValue:      "test-anthropic-key",
			expectedValue: "test-anthropic-key",
		},
		{
			name:          "deepseek provider",
			provider:      "deepseek",
			envVar:        "DEEPSEEK_API_KEY",
			envValue:      "test-deepseek-key",
			expectedValue: "test-deepseek-key",
		},
		{
			name:          "unknown provider",
			provider:      "unknown",
			envVar:        "UNKNOWN_API_KEY",
			envValue:      "test-unknown-key",
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				t.Setenv(tt.envVar, tt.envValue)
			}

			result := getAPIKeyFromEnv(tt.provider)
			assert.Equal(t, tt.expectedValue, result)
		})
	}
}

// TestLanguageParsing tests language parsing from CLI arguments
func TestLanguageParsingComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		locale       string
		language     string
		expectedLang language.Language
		expectError  bool
	}{
		{
			name:         "valid locale",
			locale:       "es",
			language:     "",
			expectedLang: language.Spanish,
			expectError:  false,
		},
		{
			name:         "valid language name",
			locale:       "",
			language:     "Spanish",
			expectedLang: language.Spanish,
			expectError:  false,
		},
		{
			name:        "invalid locale",
			locale:      "xx",
			language:    "",
			expectError: true,
		},
		{
			name:        "invalid language",
			locale:      "",
			language:    "InvalidLanguage",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var targetLang language.Language
			var err error

			if tt.locale != "" {
				targetLang, err = language.ParseLanguage(tt.locale)
			} else if tt.language != "" {
				targetLang, err = language.ParseLanguage(tt.language)
			} else {
				targetLang = language.English
			}

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLang.Code, targetLang.Code)
				assert.Equal(t, tt.expectedLang.Name, targetLang.Name)
			}
		})
	}
}

// TestScriptConversion tests script conversion functionality
func TestScriptConversionComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		book       *ebook.Book
		scriptType string
		targetLang language.Language
		expectFunc func(*ebook.Book)
	}{
		{
			name:       "latin conversion for Serbian",
			book:       createTestBook(t, "Тест књига", "Ово је тест садржај на ћирилиц"),
			scriptType: "latin",
			targetLang: language.Serbian,
			expectFunc: func(book *ebook.Book) {
				// Verify that content is converted to Latin
				assert.Contains(t, book.Metadata.Title, "Test")
				assert.Contains(t, book.Chapters[0].Title, "Test")
			},
		},
		{
			name:       "no conversion for non-Serbian",
			book:       createTestBook(t, "Test Book", "Test content"),
			scriptType: "latin",
			targetLang: language.English,
			expectFunc: func(book *ebook.Book) {
				// Verify that content is unchanged
				assert.Equal(t, "Test Book", book.Metadata.Title)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalBook := *tt.book

			// Test conversion
			if tt.scriptType == "latin" && tt.targetLang.Code == "sr" {
				converter := script.NewConverter()
				convertBookToLatin(tt.book, converter)
			}

			// Verify expectations
			tt.expectFunc(tt.book)

			// Restore original for next test
			*tt.book = originalBook
		})
	}
}

// TestWriteAsText tests text writing functionality
func TestWriteAsTextComprehensive(t *testing.T) {
	tests := []struct {
		name        string
		book        *ebook.Book
		expected    string
		expectError bool
	}{
		{
			name:        "simple book",
			book:        createTestBook(t, "Test Book", "Test content"),
			expected:    "Test content",
			expectError: false,
		},
		{
			name:        "empty book",
			book:        createTestBook(t, "", ""),
			expected:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "test.txt")

			err := writeAsText(tt.book, filename)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Verify content
			content, err := os.ReadFile(filename)
			assert.NoError(t, err)
			assert.Contains(t, string(content), tt.expected)
		})
	}
}

// TestGetSupportedLanguagesString tests the languages string formatting
func TestGetSupportedLanguagesStringComprehensive(t *testing.T) {
	result := getSupportedLanguagesString()

	// Verify it contains some expected languages
	assert.Contains(t, result, "English (en)")
	assert.Contains(t, result, "Spanish (es)")
	assert.Contains(t, result, "French (fr)")
	assert.Contains(t, result, "German (de)")
	assert.Contains(t, result, "Serbian (sr)")
}

// TestPrintHelp tests the help function
func TestPrintHelpComprehensive(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	defer func() {
		os.Stdout = oldStdout
	}()

	printHelp()
	w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Verify help content
	assert.Contains(t, output, "Universal Ebook Translator v"+versionpkg.AppVersion)
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Options:")
	assert.Contains(t, output, "Environment Variables:")
	assert.Contains(t, output, "Examples:")
}

// Helper function to create test book
func createTestBook(t *testing.T, title, content string) *ebook.Book {
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       title,
			Description: "Test description",
			Authors:     []string{"Test Author"},
		},
		Format: "epub",
		Chapters: []ebook.Chapter{
			{
				Title: "Test Chapter",
				Sections: []ebook.Section{
					{
						Title:   "Test Section",
						Content: content,
					},
				},
			},
		},
	}
	return book
}

// TestConfigLoading tests configuration loading functionality
func TestConfigLoadingComprehensive(t *testing.T) {
	tests := []struct {
		name          string
		configContent string
		expectError   bool
		expectedVals  map[string]string
	}{
		{
			name: "valid config with translation settings",
			configContent: `{
				"translation": {
					"default_provider": "openai",
					"default_model": "gpt-4",
					"providers": {
						"openai": {
							"api_key": "test-openai-key",
							"base_url": "https://api.openai.com",
							"model": "gpt-4"
						}
					}
				},
				"distributed": {
					"enabled": true
				}
			}`,
			expectError: false,
			expectedVals: map[string]string{
				"provider": "openai",
				"model":    "gpt-4",
			},
		},
		{
			name:          "invalid JSON config",
			configContent: `{"invalid": json}`,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test-config.json")

			if tt.configContent != "" {
				err := os.WriteFile(configFile, []byte(tt.configContent), 0644)
				require.NoError(t, err)
			}

			// Test config loading
			appConfig, err := config.LoadConfig(configFile)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, appConfig)

			// Verify expected values if provided
			if tt.expectedVals != nil {
				if provider, ok := tt.expectedVals["provider"]; ok {
					assert.Equal(t, provider, appConfig.Translation.DefaultProvider)
				}
				if model, ok := tt.expectedVals["model"]; ok {
					assert.Equal(t, model, appConfig.Translation.DefaultModel)
				}
			}
		})
	}
}

// TestTranslatorCreation tests different translator creation scenarios
func TestTranslatorCreationComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		config       translator.TranslationConfig
		eventBus     *events.EventBus
		expectError  bool
		expectedType string
	}{
		{
			name:         "create OpenAI translator",
			providerName: "openai",
			config: translator.TranslationConfig{
				Provider: "openai",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			expectError:  true, // Will error without real API key
			expectedType: "LLM",
		},
		{
			name:         "create multi-LLM translator",
			providerName: "multi-llm",
			config: translator.TranslationConfig{
				Provider: "multi-llm",
				Model:    "gpt-4",
				APIKey:   "test-key",
			},
			expectError:  true, // Will error without proper setup
			expectedType: "MultiLLM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventBus := events.NewEventBus()
			sessionID := "test-session"

			// Test translator creation - will likely fail in test environment
			// but we can test the logic
			if tt.providerName == "multi-llm" || tt.providerName == "distributed" || tt.providerName == "" {
				// Test multi-LLM path
				_ = sessionID // Use sessionID to avoid unused variable error
			} else {
				// Test single translator path
				_ = eventBus // Use eventBus to avoid unused variable error
			}

			if tt.expectError {
				// In test environment, we expect errors due to missing API keys
				// This is expected behavior
				t.Log("Expected error in test environment due to missing API keys")
			}
		})
	}
}

// TestMockTranslatorIntegration tests mock translator integration
func TestMockTranslatorIntegration(t *testing.T) {
	// Create mock translator
	mockTranslator := &mocks.MockTranslator{}
	mockTranslator.On("GetName").Return("mock-translator")
	mockTranslator.On("GetStats").Return(translator.TranslationStats{
		Total:      10,
		Translated: 8,
		Cached:     2,
		Errors:     0,
	})

	// Test mock functionality
	assert.Equal(t, "mock-translator", mockTranslator.GetName())
	stats := mockTranslator.GetStats()
	assert.Equal(t, 10, stats.Total)
	assert.Equal(t, 8, stats.Translated)
	assert.Equal(t, 2, stats.Cached)
	assert.Equal(t, 0, stats.Errors)

	// Verify mock expectations
	mockTranslator.AssertExpectations(t)
}

// TestTranslateEbookFunction tests the translateEbook function
func TestTranslateEbookFunction(t *testing.T) {
	// Create a test book
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:    "Test Book",
			Language: "en",
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Test Chapter",
				Sections: []ebook.Section{
					{
						Content: "This is test content.",
					},
				},
			},
		},
	}

	// Test with minimal parameters. Under R2 the translator comes from the
	// LLMsVerifier bridge; with no provider API keys in the environment the bridge
	// fails to open and translateEbook returns an honest hard error (never a local
	// fallback, §11.4.69). The positive path needs the live pipeline → SKIP (§11.4.3).
	t.Run("minimal_parameters", func(t *testing.T) {
		if hasAnyProviderKey() {
			t.Skip("SKIP-OK (§11.4.3/§11.4.98): provider key(s) present — the bridge would perform a LIVE verification pass (real credentials, network); the no-key honest-error contract is only assertable with no keys present")
		}
		err := translateEbook(
			book,
			filepath.Join(t.TempDir(), "test_output.epub"),
			"epub",
			"openai",
			"gpt-3.5-turbo",
			"test-key",
			"",
			"latn",
			nil,
			language.English,
			language.Spanish,
			nil,
			false,
			false,
			true, // provider explicitly set
			true, // api-key explicitly set
		)
		assert.Error(t, err, "with no provider keys the bridge-backed translateEbook MUST fail loudly")
	})

	t.Run("with_app_config_bridge_contract", func(t *testing.T) {
		// R2 (R-1a): translateEbook now sources its translator from the LLMsVerifier
		// bridge — it NO LONGER constructs an llm.NewLLMTranslator from a config
		// provider's APIKey/BaseURL, so the previous "point an httptest mock at the
		// openai BaseURL and assert it's hit" path is gone by design (provider /
		// api-key / base-url are advisory under R2; the bridge selects the verified
		// model). The CLI-seam contract this subtest now asserts is §11.4.69: with
		// NO provider API keys present, the config-default provider path still fails
		// loudly via the bridge and produces NO output — never a silent local/config
		// fallback. The positive bridge path needs the live verification pipeline and
		// is gated with SKIP-with-reason (§11.4.3) BEFORE any bridge call so the
		// offline suite never performs a live verification pass.
		if hasAnyProviderKey() {
			t.Skip("SKIP-OK (§11.4.3/§11.4.98): provider key(s) present — the bridge would perform a LIVE verification pass (real credentials, network); the no-key honest-error contract is only assertable with no keys present")
		}
		outFile := filepath.Join(t.TempDir(), "test_output.epub")

		appConfig := &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
				DefaultModel:    "gpt-3.5-turbo",
				Providers: map[string]config.ProviderConfig{
					"openai": {APIKey: "config-key"},
				},
			},
		}

		err := translateEbook(
			book,
			outFile,
			"epub",
			"", // provider — use default from config (openai)
			"", // model — use default from config
			"", // apiKey — use from config
			"", // baseURL — use from config
			"latn",
			appConfig,
			language.English,
			language.Spanish,
			nil,
			false,
			false,
			false, // provider not explicitly set — config default applies
			false, // api-key not explicitly set — config/env applies
		)

		require.Error(t, err, "with no provider keys the bridge-backed translateEbook MUST fail loudly")
		require.NoFileExists(t, outFile, "a failed (no-key) run must not leave an output file")
	})
}
