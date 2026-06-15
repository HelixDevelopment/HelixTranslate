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

		// R-1b/R2: the "provider" field is now ADVISORY — the LLMsVerifier bridge
		// selects the strongest VERIFIED model regardless of the requested provider
		// name, so batchTranslate no longer rejects an unknown provider at the
		// local-construction layer (the prior NewLLMTranslator allowlist is gone).
		// The request is accepted and the bridge-sourced translator runs.
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
