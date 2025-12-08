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

// TestLowestCoverageFunctions adds basic coverage for functions with lowest coverage
func TestLowestCoverageFunctions(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	// Test translateDistributed handler (14.3% coverage)
	t.Run("translateDistributed with nil distributed manager", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/distributed/translate", handler.translateDistributed)
		
		testData := map[string]interface{}{
			"text":        "Test text",
			"source_lang": "en",
			"target_lang": "es",
			"worker_id":   "test-worker",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/translate", bytes.NewBuffer(jsonData))
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
	
	// Test acknowledgeAlert handler (15.8% coverage)
	t.Run("acknowledgeAlert with nil distributed manager", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/:alert_id/acknowledge", handler.acknowledgeAlert)
		
		testData := map[string]interface{}{
			"comment": "Acknowledged in test",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/test-alert/acknowledge", bytes.NewBuffer(jsonData))
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
	
	// Test getVersionAlerts handler (18.8% coverage)
	t.Run("getVersionAlerts with nil distributed manager", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts", handler.getVersionAlerts)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 503 when distributedManager is nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Distributed work not available", response["error"])
	})
	
	// Test getVersionDashboard handler (11.1% coverage)
	t.Run("getVersionDashboard with nil distributed manager", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/monitoring/version/dashboard", handler.getVersionDashboard)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 503 when distributedManager is nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Distributed work not available", response["error"])
	})
	
	// Test addWebhookAlertChannel handler (18.8% coverage)
	t.Run("addWebhookAlertChannel with nil distributed manager", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/webhook", handler.addWebhookAlertChannel)
		
		testData := map[string]interface{}{
			"url":         "https://example.com/webhook",
			"enabled":     true,
			"min_severity": "error",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/webhook", bytes.NewBuffer(jsonData))
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
	
	// Test addSlackAlertChannel handler (18.8% coverage)
	t.Run("addSlackAlertChannel with nil distributed manager", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/slack", handler.addSlackAlertChannel)
		
		testData := map[string]interface{}{
			"webhook_url": "https://hooks.slack.com/test",
			"channel":     "#alerts",
			"enabled":     true,
			"min_severity": "warning",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/slack", bytes.NewBuffer(jsonData))
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
}