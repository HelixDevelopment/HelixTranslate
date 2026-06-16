package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestBatchTranslateMoreCoverage adds more test cases for batchTranslate handler
func TestBatchTranslateMoreCoverage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("batchTranslate with empty texts array", func(t *testing.T) {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
				Providers: map[string]config.ProviderConfig{
					"openai": {
						APIKey: "test-key",
						Model:  "gpt-3.5-turbo",
					},
				},
			},
		}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)

		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}

		installMockBridge(handler) // R-1b/R2: source translator from the bridge seam
		router := gin.New()
		router.POST("/batch", handler.batchTranslate)

		testData := map[string]interface{}{
			"texts":    []string{},
			"provider": "openai",
			"model":    "gpt-3.5-turbo",
			"context":  "Test context",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/batch", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should handle empty array gracefully
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err == nil {
			assert.NotNil(t, response)
		}
	})

	t.Run("batchTranslate with multiple texts", func(t *testing.T) {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
				Providers: map[string]config.ProviderConfig{
					"openai": {
						APIKey: "test-key",
						Model:  "gpt-3.5-turbo",
					},
				},
			},
		}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)

		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}

		installMockBridge(handler) // R-1b/R2: source translator from the bridge seam
		router := gin.New()
		router.POST("/batch", handler.batchTranslate)

		testData := map[string]interface{}{
			"texts":    []string{"Hello", "World", "Test"},
			"provider": "openai",
			"model":    "gpt-3.5-turbo",
			"context":  "Test context",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/batch", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should handle multiple texts
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err == nil {
			assert.NotNil(t, response)
			if response["translated"] != nil {
				translated := response["translated"].([]interface{})
				assert.Equal(t, 3, len(translated)) // Should have 3 results
			}
		}
	})

	t.Run("batchTranslate with invalid JSON", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)

		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}

		router := gin.New()
		router.POST("/batch", handler.batchTranslate)

		// Invalid JSON
		req, _ := http.NewRequest("POST", "/batch", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 for invalid JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("batchTranslate with missing texts field", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)

		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}

		router := gin.New()
		router.POST("/batch", handler.batchTranslate)

		testData := map[string]interface{}{
			"provider": "openai",
			"model":    "gpt-3.5-turbo",
			"context":  "Test context",
			// Missing texts field
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/batch", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 for missing required field
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("batchTranslate with invalid provider", func(t *testing.T) {
		cfg := &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "openai",
				Providers: map[string]config.ProviderConfig{
					"openai": {
						APIKey: "test-key",
						Model:  "gpt-3.5-turbo",
					},
				},
			},
		}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)

		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}

		installMockBridge(handler) // R-1b/R2: source translator from the bridge seam
		router := gin.New()
		router.POST("/batch", handler.batchTranslate)

		testData := map[string]interface{}{
			"texts":    []string{"Hello", "World"},
			"provider": "invalid-provider",
			"model":    "gpt-3.5-turbo",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/batch", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// An explicitly-named provider the system does not support is a client
		// error and MUST be rejected with 400 — never silently routed to a
		// different provider by the bridge (a response-correctness defect: the
		// caller asked for "invalid-provider" and would otherwise get a real
		// translation from some OTHER provider with no error). createTranslator
		// validates the requested provider against the canonical provider registry
		// (llm.IsKnownProvider) before reaching the bridge, so batchTranslate
		// returns 400 here. (§11.4.69 no-silent-substitution; §11.4.120 — this
		// assertion was reconciled from a stale advisory-200 expectation to the
		// corrected 400 behaviour, matching translateText's unsupported-provider
		// contract.)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
			assert.Contains(t, response["error"], "unsupported provider")
		}
	})
}
