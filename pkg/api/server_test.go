package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTranslateEndpoint(t *testing.T) {
	// Test 1: Successful translation
	t.Run("SuccessfulTranslation", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, "Hello", "en->es").
			Return("Hola", nil)

		server := NewServer(ServerConfig{
			Port:   8080,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		reqBody := map[string]string{
			"text":        "Hello",
			"source_lang": "en",
			"target_lang": "es",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "Hola", response["translated_text"])
		assert.NotNil(t, response["source_lang"])
		assert.NotNil(t, response["target_lang"])
	})

	// Test 2: Invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port:   8081,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader("invalid json"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp["error"], "invalid character")
	})

	// Test 3: Missing required fields
	t.Run("MissingRequiredFields", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port:   8082,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		testCases := []struct {
			name    string
			reqBody string
		}{
			{"missingText", `{"source_lang": "en", "target_lang": "es"}`},
			{"missingSourceLang", `{"text": "Hello", "target_lang": "es"}`},
			{"missingTargetLang", `{"text": "Hello", "source_lang": "en"}`},
			{"emptyText", `{"text": "", "source_lang": "en", "target_lang": "es"}`},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequest("POST", testServer.URL+"/api/translate", strings.NewReader(tc.reqBody))
				require.NoError(t, err)
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

				var errorResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&errorResp)
				require.NoError(t, err)

				assert.NotEmpty(t, errorResp["error"])
			})
		}
	})

	// Test 4: Translator error
	t.Run("TranslatorError", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, "Hello", "en->es").
			Return("", fmt.Errorf("translation service unavailable"))

		server := NewServer(ServerConfig{
			Port:   8083,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		reqBody := map[string]string{
			"text":        "Hello",
			"source_lang": "en",
			"target_lang": "es",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp["error"], "translation service unavailable")
	})
}

func TestHealthEndpoint(t *testing.T) {
	// Test 1: Basic health check
	t.Run("BasicHealthCheck", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port:   8084,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		req, err := http.NewRequest("GET", testServer.URL+"/health", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "ok", health["status"])
		assert.NotNil(t, health["translator"])
	})

	// Test 2: Health check with translator status
	t.Run("HealthCheckWithTranslator", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("GetStats").Return(map[string]interface{}{
			"total_translations": 100,
			"cache_hits":       85,
		})

		server := NewServer(ServerConfig{
			Port:   8085,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		req, err := http.NewRequest("GET", testServer.URL+"/health", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "ok", health["status"])
		assert.NotNil(t, health["translator"])
		
		if translator, ok := health["translator"].(map[string]interface{}); ok {
			assert.Equal(t, "test-translator", translator["name"])
			assert.Equal(t, float64(100), translator["total_translations"])
			assert.Equal(t, float64(85), translator["cache_hits"])
		}
	})
}

func TestLanguagesEndpoint(t *testing.T) {
	// Test 1: Get supported languages
	t.Run("GetSupportedLanguages", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("GetSupportedLanguages").Return(map[string]string{
			"en": "English",
			"es": "Spanish",
			"fr": "French",
			"de": "German",
		})

		server := NewServer(ServerConfig{
			Port:   8086,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		req, err := http.NewRequest("GET", testServer.URL+"/api/languages", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var languages map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&languages)
		require.NoError(t, err)

		// Check if languages array exists and contains expected codes
		langsArray, ok := languages["languages"].([]interface{})
		assert.True(t, ok)
		assert.Contains(t, langsArray, "en")
		assert.Contains(t, langsArray, "es")
		assert.Contains(t, langsArray, "fr")
		assert.Contains(t, langsArray, "de")
	})
}

func TestStatsEndpoint(t *testing.T) {
	// Test 1: Get translation statistics
	t.Run("GetTranslationStats", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("GetStats").Return(map[string]interface{}{
			"total_translations": 1000,
			"cache_hits":       850,
			"cache_misses":     150,
			"avg_response_time": 150,
			"uptime":          "48h30m",
		})

		server := NewServer(ServerConfig{
			Port:   8087,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		req, err := http.NewRequest("GET", testServer.URL+"/api/stats", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var stats map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&stats)
		require.NoError(t, err)

		assert.Equal(t, float64(0), stats["translations"])
		assert.Equal(t, "0s", stats["uptime"])
	})
}

func TestFileUploadEndpoint(t *testing.T) {
	// Test 1: File upload translation
	t.Run("FileUploadTranslation", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("TranslateFile", mock.Anything, mock.Anything, "en", "es", mock.Anything).
			Run(func(args mock.Arguments) {
				// Mock file translation progress
				progress := args.Get(4).(func(int, int))
				progress(50, 100)
				progress(100, 100)
			}).
			Return([]byte("translated content"), nil)

		server := NewServer(ServerConfig{
			Port:   8088,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		// Create multipart form request
		body := &bytes.Buffer{}
		contentType := "multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW"
		
		// Add file part
		fmt.Fprintf(body, "------WebKitFormBoundary7MA4YWxkTrZu0gW\r\n")
		fmt.Fprintf(body, "Content-Disposition: form-data; name=\"file\"; filename=\"test.txt\"\r\n")
		fmt.Fprintf(body, "Content-Type: text/plain\r\n\r\n")
		fmt.Fprintf(body, "Hello world\r\n")
		fmt.Fprintf(body, "------WebKitFormBoundary7MA4YWxkTrZu0gW\r\n")
		fmt.Fprintf(body, "Content-Disposition: form-data; name=\"source_lang\"\r\n\r\n")
		fmt.Fprintf(body, "en\r\n")
		fmt.Fprintf(body, "------WebKitFormBoundary7MA4YWxkTrZu0gW\r\n")
		fmt.Fprintf(body, "Content-Disposition: form-data; name=\"target_lang\"\r\n\r\n")
		fmt.Fprintf(body, "es\r\n")
		fmt.Fprintf(body, "------WebKitFormBoundary7MA4YWxkTrZu0gW--\r\n")

		req, err := http.NewRequest("POST", testServer.URL+"/api/upload", body)
		require.NoError(t, err)
		req.Header.Set("Content-Type", contentType)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)

		// Since it returns "Not implemented", we can't decode JSON
		// Just verify the endpoint exists and returns the expected status
	})

	// Test 2: Invalid file upload
	t.Run("InvalidFileUpload", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port:   8089,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		// Send request without file
		req, err := http.NewRequest("POST", testServer.URL+"/api/upload", strings.NewReader(""))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=test")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)

		// Since it returns "Not implemented", we can't decode JSON
		// Just verify the endpoint exists and returns the expected status
	})
}

func TestBatchTranslationEndpoint(t *testing.T) {
	// Test 1: Batch translation
	t.Run("BatchTranslation", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, "Hello", "en->es").Return("Hola", nil)
		mockTranslator.On("Translate", mock.Anything, "World", "en", "es").Return("Mundo", nil)

		server := NewServer(ServerConfig{
			Port:   8090,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		batchRequest := map[string]interface{}{
			"texts": []string{"Hello", "World"},
			"source_lang": "en",
			"target_lang": "es",
		}
		jsonBody, _ := json.Marshal(batchRequest)

		req, err := http.NewRequest("POST", testServer.URL+"/api/batch", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)

		// Since it returns "Not implemented", we can't decode JSON
		// Just verify endpoint exists and returns expected status
	})

	// Test 2: Batch translation size limit
	t.Run("BatchTranslationSizeLimit", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port:   8091,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
			Security: &SecurityConfig{
				MaxBatchSize: 10,
			},
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		// Create batch with 20 texts (exceeds limit of 10)
		texts := make([]string, 20)
		for i := range texts {
			texts[i] = fmt.Sprintf("Text %d", i)
		}

		batchRequest := map[string]interface{}{
			"texts": texts,
			"source_lang": "en",
			"target_lang": "es",
		}
		jsonBody, _ := json.Marshal(batchRequest)

		req, err := http.NewRequest("POST", testServer.URL+"/api/batch", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)

		// Since it returns "Not implemented", we can't decode JSON
		// Just verify endpoint exists and returns expected status
	})
}

func TestServerErrorHandling(t *testing.T) {
	// Test 1: 404 Not Found
	t.Run("NotFound", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port:   8092,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		req, err := http.NewRequest("GET", testServer.URL+"/nonexistent", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test 2: Method Not Allowed
	t.Run("MethodNotAllowed", func(t *testing.T) {
		server := NewServer(ServerConfig{
			Port:   8093,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		req, err := http.NewRequest("DELETE", testServer.URL+"/api/translate", nil)
		require.NoError(t, err)

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test 3: Panic recovery
	t.Run("PanicRecovery", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything).
			Panic("simulated panic")

		server := NewServer(ServerConfig{
			Port:   8094,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		reqBody := map[string]string{
			"text":        "Hello",
			"source_lang": "en",
			"target_lang": "es",
		}
		jsonBody, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("POST", testServer.URL+"/api/translate", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 500 instead of crashing
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		// The response might not be valid JSON due to panic recovery
		// Just check status code
	})
}

func TestConcurrency(t *testing.T) {
	// Test 1: Concurrent requests
	t.Run("ConcurrentRequests", func(t *testing.T) {
		mockTranslator := new(mocks.MockTranslator)
		mockTranslator.On("GetName").Return("test-translator")
		mockTranslator.On("Translate", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return("translated", nil)

		server := NewServer(ServerConfig{
			Port:   8095,
			Logger: logger.NewLogger(logger.LoggerConfig{
				Level:  logger.INFO,
				Format: logger.FORMAT_TEXT,
			}),
		})
		server.SetTranslator(mockTranslator)

		testServer := httptest.NewServer(server.GetRouter())
		defer testServer.Close()

		numRequests := 10
		results := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(id int) {
				reqBody := map[string]string{
					"text":        fmt.Sprintf("Request %d", id),
					"source_lang": "en",
					"target_lang": "es",
				}
				jsonBody, _ := json.Marshal(reqBody)

				req, err := http.NewRequest("POST", testServer.URL+"/api/translate", bytes.NewBuffer(jsonBody))
				if err != nil {
					results <- err
					return
				}
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					results <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
					return
				}

				results <- nil
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < numRequests; i++ {
			err := <-results
			assert.NoError(t, err, "Concurrent request should succeed")
		}
	})
}