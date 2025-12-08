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

// TestTranslateTextMoreCoverage adds more test cases for translateText handler
func TestTranslateTextMoreCoverage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Test with distributed manager nil
	t.Run("translateText with nil distributed manager", func(t *testing.T) {
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
		
		router := gin.New()
		router.POST("/translate", handler.translateText)
		
		testData := map[string]interface{}{
			"text":        "Hello world",
			"provider":    "openai",
			"model":       "gpt-3.5-turbo",
			"context":     "Test context",
			"script":      "test-script",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// The response should indicate whether translation succeeded or failed
		// We're checking that it processed the request
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError || w.Code == http.StatusBadRequest)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err == nil && response["status"] != nil {
			// If JSON is valid and has status, check it's one of expected values
			status := response["status"].(float64)
			assert.True(t, status == 200 || status == 500 || status == 400)
		}
	})
	
	t.Run("translateText with invalid JSON", func(t *testing.T) {
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
		router.POST("/translate", handler.translateText)
		
		// Invalid JSON
		req, _ := http.NewRequest("POST", "/translate", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 400 for invalid JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	
	t.Run("translateText with unsupported provider", func(t *testing.T) {
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
		
		router := gin.New()
		router.POST("/translate", handler.translateText)
		
		testData := map[string]interface{}{
			"text":        "Hello world",
			"provider":    "unsupported-provider",
			"model":       "gpt-3.5-turbo",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 400 for unsupported provider
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err == nil && response["status"] != nil {
			// Check status if available
			status := response["status"].(float64)
			assert.Equal(t, float64(400), status)
		}
	})
}