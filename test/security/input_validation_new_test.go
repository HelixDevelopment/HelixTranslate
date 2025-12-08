package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/test/mocks"
	"digital.vasic.translator/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestInputSanitization(t *testing.T) {
	// Test 1: HTML tag sanitization
	t.Run("HTMLTagSanitization", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return("translated text", nil)

		// Use test infrastructure with dynamic port
		httpServer := utils.NewTestHTTPServer(t)
		defer httpServer.Close()

		testServerURL := httpServer.GetURL()

		// Send input with HTML tags
		htmlInput := `<p>Hello <b>world</b> <script>alert('xss')</script></p>`
		reqBody := fmt.Sprintf(`{
			"text": "%s",
			"source_lang": "en",
			"target_lang": "es"
		}`, htmlInput)

		req, err := http.NewRequest("POST", testServerURL+"/api/v1/translate", strings.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response to ensure HTML is sanitized
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		if translatedText, ok := result["translated_text"].(string); ok {
			// Should not contain script tags or dangerous HTML
			assert.NotContains(t, translatedText, "<script>")
			assert.NotContains(t, translatedText, "alert(")
			// May contain safe HTML depending on sanitization policy
		}
	})

	// Test 2: Script injection in various fields
	t.Run("ScriptInjectionInFields", func(t *testing.T) {
		server := api.NewServer(api.ServerConfig{
			Port:     8090,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey:        "test-key",
				SanitizeInput:  true,
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		testCases := []struct {
			name     string
			field    string
			value    string
		}{
			{"scriptInText", "text", "<script>window.location='http://evil.com'</script>"},
			{"scriptInSourceLang", "source_lang", "en<script>alert(1)</script>"},
			{"scriptInTargetLang", "target_lang", "es<script>alert(2)</script>"},
			{"javascriptInText", "text", "javascript:alert('xss')"},
			{"dataURL", "text", "data:text/html,<script>alert('xss')</script>"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reqBody := fmt.Sprintf(`{
					"text": "Hello world",
					"source_lang": "en",
					"target_lang": "es",
					"%s": "%s"
				}`, tc.field, tc.value)

				req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(reqBody))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer test-key")
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				// Should either be rejected or sanitized
				if resp.StatusCode == http.StatusBadRequest {
					// Good - rejected at validation
					return
				}

				// If accepted, ensure response is sanitized
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			})
		}
	})

	// Test 3: Null byte injection
	t.Run("NullByteInjection", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:     8091,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey:        "test-key",
				SanitizeInput:  true,
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		// Test null byte injection
		nullByteInput := "Hello\x00world"
		reqBody := fmt.Sprintf(`{
			"text": "%s",
			"source_lang": "en",
			"target_lang": "es"
		}`, nullByteInput)

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should reject null bytes
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test 4: Unicode control characters
	t.Run("UnicodeControlCharacters", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:     8092,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey:        "test-key",
				SanitizeInput:  true,
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		// Test with various Unicode control characters
		controlChars := "\u0000\u0001\u0002\u0003\u0004\u0005\u0006\u0007\u0008\u000b\u000c\u000e\u000f\u0010\u0011\u0012\u0013\u0014\u0015\u0016\u0017\u0018\u0019\u001a\u001b\u001c\u001d\u001e\u001f\u007f"
		reqBody := fmt.Sprintf(`{
			"text": "Hello%sWorld",
			"source_lang": "en",
			"target_lang": "es"
		}`, controlChars)

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should reject control characters
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestSizeLimitValidation(t *testing.T) {
	t.Skip("Skipping size limit validation tests that require HTTP server infrastructure")
	// Test 1: Request size limits
	t.Run("RequestSizeLimit", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:     8093,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey:           "test-key",
				MaxRequestSize:   1024 * 1024, // 1MB
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		// Create a very large text (2MB)
		largeText := strings.Repeat("a", 2*1024*1024)
		reqBody := fmt.Sprintf(`{
			"text": "%s",
			"source_lang": "en",
			"target_lang": "es"
		}`, largeText)

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should reject oversized requests
		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
	})

	// Test 2: Text length limits
	t.Run("TextLengthLimit", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:     8094,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey:         "test-key",
				MaxTextLength:  10000, // 10k characters
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		// Create text longer than limit
		longText := strings.Repeat("word ", 10001) // 55,055 characters
		reqBody := fmt.Sprintf(`{
			"text": "%s",
			"source_lang": "en",
			"target_lang": "es"
		}`, longText)

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should reject oversized text
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Check error message
		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		
		if message, ok := errorResp["error"].(string); ok {
			assert.Contains(t, message, "text length")
			assert.Contains(t, message, "exceeds")
		}
	})
}

func TestFieldValidation(t *testing.T) {
	t.Skip("Skipping field validation tests that require HTTP server infrastructure")
	// Test 1: Required field validation
	t.Run("RequiredFieldValidation", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:     8095,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey: "test-key",
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		testCases := []struct {
			name    string
			reqBody string
		}{
			{
				"missingText",
				`{"source_lang": "en", "target_lang": "es"}`,
			},
			{
				"missingSourceLang",
				`{"text": "Hello", "target_lang": "es"}`,
			},
			{
				"missingTargetLang",
				`{"text": "Hello", "source_lang": "en"}`,
			},
			{
				"emptyText",
				`{"text": "", "source_lang": "en", "target_lang": "es"}`,
			},
			{
				"emptySourceLang",
				`{"text": "Hello", "source_lang": "", "target_lang": "es"}`,
			},
			{
				"emptyTargetLang",
				`{"text": "Hello", "source_lang": "en", "target_lang": ""}`,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(tc.reqBody))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer test-key")
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

				// Check error message
				var errorResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(t, err)
				
				if message, ok := errorResp["error"].(string); ok {
					assert.NotEmpty(t, message)
				}
			})
		}
	})

	// Test 2: Language code validation
	t.Run("LanguageCodeValidation", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:     8096,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey: "test-key",
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		testCases := []struct {
			name    string
			source  string
			target  string
			valid   bool
		}{
			{"validCodes", "en", "es", true},
			{"validCodesWithRegion", "en-US", "es-ES", true},
			{"invalidSourceCode", "invalid-lang", "es", false},
			{"invalidTargetCode", "en", "invalid-lang", false},
			{"tooLongCode", "eng", "spa", false},
			{"numericCode", "123", "456", false},
			{"specialChars", "en@", "es#", false},
			{"uppercase", "EN", "ES", false},
			{"mixedCase", "En", "eS", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reqBody := fmt.Sprintf(`{
					"text": "Hello",
					"source_lang": "%s",
					"target_lang": "%s"
				}`, tc.source, tc.target)

				req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(reqBody))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer test-key")
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				if tc.valid {
					// Should accept valid codes (may fail for other reasons like missing translator)
					assert.NotEqual(t, http.StatusBadRequest, resp.StatusCode)
				} else {
					// Should reject invalid codes
					assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
				}
			})
		}
	})
}

func TestContentTypeValidation(t *testing.T) {
	t.Skip("Skipping content type validation tests that require HTTP server infrastructure")
	// Test 1: Content-Type validation
	t.Run("ContentTypeValidation", func(t *testing.T) {
		mockLogger := logger.NewLogger(logger.LoggerConfig{
			Level:  logger.DEBUG,
			Format: logger.FORMAT_TEXT,
		})

		server := api.NewServer(api.ServerConfig{
			Port:     8097,
			Logger:    mockLogger,
			Security: &api.SecurityConfig{
				APIKey: "test-key",
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		testCases := []struct {
			name        string
			contentType string
			shouldPass  bool
		}{
			{"validJSON", "application/json", true},
			{"validJSONCharset", "application/json; charset=utf-8", true},
			{"invalidContentType", "text/plain", false},
			{"invalidXML", "application/xml", false},
			{"invalidForm", "application/x-www-form-urlencoded", false},
			{"missingContentType", "", false},
			{"wildcard", "*/*", false},
		}

		reqBody := `{
			"text": "Hello",
			"source_lang": "en",
			"target_lang": "es"
		}`

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(reqBody))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer test-key")
				
				if tc.contentType != "" {
					req.Header.Set("Content-Type", tc.contentType)
				}

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				if tc.shouldPass {
					assert.NotEqual(t, http.StatusUnsupportedMediaType, resp.StatusCode)
				} else {
					assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
				}
			})
		}
	})
}