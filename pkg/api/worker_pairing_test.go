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

// TestWorkerPairingMoreCoverage adds more test cases for worker pairing handlers
func TestWorkerPairingMoreCoverage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	t.Run("pairWorker with nil distributed manager", func(t *testing.T) {
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
		router.POST("/api/v1/distributed/workers/:worker_id/pair", handler.pairWorker)
		
		testData := map[string]interface{}{
			"auth_token": "test-token",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/workers/test-worker/pair", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 503 when distributedManager is nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Distributed work not available", response["error"])
	})
	
	t.Run("pairWorker with empty worker ID", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)
		
		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: &struct{}{}, // Not nil but not a valid DistributedManager
		}
		
		router := gin.New()
		router.POST("/api/v1/distributed/workers/:worker_id/pair", handler.pairWorker)
		
		testData := map[string]interface{}{
			"auth_token": "test-token",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/workers//pair", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 400 when worker ID is empty
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Worker ID is required", response["error"])
	})
	
	t.Run("pairWorker with invalid distributed manager", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)
		
		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: &struct{}{}, // Not nil but not a valid DistributedManager
		}
		
		router := gin.New()
		router.POST("/api/v1/distributed/workers/:worker_id/pair", handler.pairWorker)
		
		testData := map[string]interface{}{
			"auth_token": "test-token",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/workers/test-worker/pair", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 500 when distributedManager is invalid
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})
	
	t.Run("unpairWorker with nil distributed manager", func(t *testing.T) {
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
		router.DELETE("/api/v1/distributed/workers/:worker_id/pair", handler.unpairWorker)
		
		req, _ := http.NewRequest("DELETE", "/api/v1/distributed/workers/test-worker/pair", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 503 when distributedManager is nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Distributed work not available", response["error"])
	})
	
	t.Run("unpairWorker with empty worker ID", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)
		
		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: &struct{}{}, // Not nil but not a valid DistributedManager
		}
		
		router := gin.New()
		router.DELETE("/api/v1/distributed/workers/:worker_id/pair", handler.unpairWorker)
		
		req, _ := http.NewRequest("DELETE", "/api/v1/distributed/workers//pair", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 400 when worker ID is empty
		assert.Equal(t, http.StatusBadRequest, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Worker ID is required", response["error"])
	})
	
	t.Run("unpairWorker with invalid distributed manager", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)
		
		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: &struct{}{}, // Not nil but not a valid DistributedManager
		}
		
		router := gin.New()
		router.DELETE("/api/v1/distributed/workers/:worker_id/pair", handler.unpairWorker)
		
		req, _ := http.NewRequest("DELETE", "/api/v1/distributed/workers/test-worker/pair", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 500 when distributedManager is invalid
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})
}