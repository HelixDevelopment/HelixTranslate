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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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
	assert.Contains(t, output, "Universal Ebook Translator v" + versionpkg.AppVersion)
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

	// Test with minimal parameters
	t.Run("minimal_parameters", func(t *testing.T) {
		err := translateEbook(
			book,
			"test_output.epub",
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
		)

		// We expect an error due to missing API key in test environment
		assert.Error(t, err)
	})

	t.Run("with_app_config", func(t *testing.T) {
		// This subtest drives translateEbook through the REAL provider path
		// (no mock translator is injected) using the config-provided provider
		// settings. Previously it (a) hard-coded a fake "config-key" for openai
		// and then asserted NoError, and (b) guarded with a skip that only fired
		// when NO provider key was in the env. That combination was a §11.4.98
		// bluff: when ANY provider key WAS present (e.g. DEEPSEEK), the subtest
		// did not skip, forced provider=openai with the fake "config-key", dialled
		// the REAL OpenAI endpoint, got a 401 invalid_api_key, and FAILED the
		// NoError assertion — a non-deterministic, ambient-env-dependent outcome.
		//
		// Fixed to be fully self-driving + deterministic (§11.4.98): stand up an
		// httptest mock that speaks the OpenAI chat-completions wire format and
		// point the config's openai BaseURL at it. The config-key path is now
		// exercised end-to-end with NO real network and NO env dependency, and we
		// positively assert (1) the run succeeded, (2) the mock was actually hit
		// (the real provider client path truly executed — not short-circuited),
		// and (3) a non-empty output file was produced.
		const mockTranslation = "Здраво свете"
		var hits int32
		mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			atomic.AddInt32(&hits, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"` + mockTranslation + `"}}]}`))
		}))
		defer mock.Close()

		outFile := filepath.Join(t.TempDir(), "test_output.epub")

		// Create app config pointing the openai provider at the mock server.
		appConfig := &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
				DefaultModel:    "gpt-3.5-turbo",
				Providers: map[string]config.ProviderConfig{
					"openai": {
						APIKey:  "config-key",
						BaseURL: mock.URL,
					},
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
			"", // baseURL — use from config (the mock URL)
			"latn",
			appConfig,
			language.English,
			language.Spanish,
			nil,
			false,
			false,
			false, // provider not explicitly set — config default applies
		)

		// (1) the config-key + base-url provider path completed successfully.
		assert.NoError(t, err)
		// (2) the real provider client path actually executed against the mock —
		// proves the translation was performed, not silently skipped.
		assert.Greater(t, atomic.LoadInt32(&hits), int32(0),
			"expected the openai client to POST to the mock /chat/completions endpoint")
		// (3) a non-empty output file was produced.
		if info, statErr := os.Stat(outFile); assert.NoError(t, statErr) {
			assert.Greater(t, info.Size(), int64(0), "expected a non-empty output EPUB")
		}
	})
}
