package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/websocket"
	"digital.vasic.translator/pkg/translator"
	
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestAPIHandlers_Uncovered tests handlers with 0% coverage
func TestAPIHandlers_Uncovered(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with mock dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	t.Run("unpairWorker", func(t *testing.T) {
		// Test unpairWorker handler
		router := gin.New()
		router.DELETE("/api/v1/distributed/workers/:worker_id/pair", handler.unpairWorker)
		
		req, _ := http.NewRequest("DELETE", "/api/v1/distributed/workers/test-worker/pair", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("translateDistributed", func(t *testing.T) {
		// Test translateDistributed handler
		router := gin.New()
		router.POST("/api/v1/distributed/translate", handler.translateDistributed)
		
		testData := map[string]interface{}{
			"text":           "Test text",
			"source_lang":    "en",
			"target_lang":    "es",
			"worker_id":      "test-worker",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("uploadUpdate", func(t *testing.T) {
		// Test uploadUpdate handler
		router := gin.New()
		router.POST("/api/v1/update/upload", handler.uploadUpdate)
		
		req, _ := http.NewRequest("POST", "/api/v1/update/upload", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil translator
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	
	t.Run("applyUpdate", func(t *testing.T) {
		// Test applyUpdate handler
		router := gin.New()
		router.POST("/api/v1/update/apply", handler.applyUpdate)
		
		testData := map[string]interface{}{
			"update_id": "test-update",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/update/apply", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 400 for missing version header
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	
	t.Run("rollbackUpdate", func(t *testing.T) {
		// Test rollbackUpdate handler
		router := gin.New()
		router.POST("/api/v1/update/rollback", handler.rollbackUpdate)
		
		testData := map[string]interface{}{
			"update_id": "test-update",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/update/rollback", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

// TestAPIVersionHandlers tests version monitoring handlers
func TestAPIVersionHandlers(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with mock dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	t.Run("getVersionMetrics", func(t *testing.T) {
		// Test getVersionMetrics handler
		router := gin.New()
		router.GET("/api/v1/monitoring/version/metrics", handler.getVersionMetrics)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("getVersionAlerts", func(t *testing.T) {
		// Test getVersionAlerts handler
		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts", handler.getVersionAlerts)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("getVersionHealth", func(t *testing.T) {
		// Test getVersionHealth handler
		router := gin.New()
		router.GET("/api/v1/monitoring/version/health", handler.getVersionHealth)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("getVersionDashboard", func(t *testing.T) {
		// Test getVersionDashboard handler
		router := gin.New()
		router.GET("/api/v1/monitoring/version/dashboard", handler.getVersionDashboard)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return JSON response
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("triggerVersionDriftCheck", func(t *testing.T) {
		// Test triggerVersionDriftCheck handler
		router := gin.New()
		router.POST("/api/v1/monitoring/version/drift-check", handler.triggerVersionDriftCheck)
		
		testData := map[string]interface{}{
			"check_id": "test-check",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/drift-check", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

// TestAPIAlertHandlers tests alert-related handlers
func TestAPIAlertHandlers(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with mock dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	t.Run("getAlertHistory", func(t *testing.T) {
		// Test getAlertHistory handler
		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts/history", handler.getAlertHistory)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts/history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("acknowledgeAlert", func(t *testing.T) {
		// Test acknowledgeAlert handler
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
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("addEmailAlertChannel", func(t *testing.T) {
		// Test addEmailAlertChannel handler
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/email", handler.addEmailAlertChannel)
		
		testData := map[string]interface{}{
			"email":      "test@example.com",
			"enabled":    true,
			"min_severity": "warning",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/email", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("addWebhookAlertChannel", func(t *testing.T) {
		// Test addWebhookAlertChannel handler
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
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
	
	t.Run("addSlackAlertChannel", func(t *testing.T) {
		// Test addSlackAlertChannel handler
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
		
		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

// TestAPIHandlerStructure tests API handler structure and initialization
func TestAPIHandlerStructure(t *testing.T) {
	// Test that Handler struct can be created
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	// Verify handler structure
	assert.NotNil(t, handler.config)
	assert.NotNil(t, handler.eventBus)
	assert.NotNil(t, handler.wsHub)
	
	// Test that dependencies are properly initialized
	assert.NotNil(t, handler.config)
	assert.NotNil(t, handler.eventBus)
	assert.NotNil(t, handler.wsHub)
}

// TestAPIErrorHandlingExtended tests error handling in API handlers
func TestAPIErrorHandlingExtended(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with mock dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	t.Run("serveDashboard", func(t *testing.T) {
		router := gin.New()
		router.GET("/api/v1/monitoring/version/dashboard.html", handler.serveDashboard)
		
		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/dashboard.html", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 200 with HTML content (fallback dashboard)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
		assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
	})
	
	t.Run("getEmbeddedDashboardHTML", func(t *testing.T) {
		// Test getEmbeddedDashboardHTML function directly
		html := handler.getEmbeddedDashboardHTML()
		
		// Should return a basic HTML structure
		assert.Contains(t, html, "<!DOCTYPE html>")
		assert.Contains(t, html, "<html")
		assert.Contains(t, html, "</html>")
	})
}

// TestDistributedTranslator tests the distributedTranslator implementation
func TestDistributedTranslator(t *testing.T) {
	// Test with nil DistributedManager
	dt := &distributedTranslator{dm: nil}
	
	t.Run("Translate with nil manager", func(t *testing.T) {
		dt := &distributedTranslator{dm: nil}
		ctx := context.Background()
		
		// This will panic with nil dereference, so we need to recover
		assert.Panics(t, func() {
			dt.Translate(ctx, "test text", "context")
		})
	})
	
	t.Run("TranslateWithProgress with nil manager", func(t *testing.T) {
		dt := &distributedTranslator{dm: nil}
		ctx := context.Background()
		
		// This will panic with nil dereference, so we need to recover
		assert.Panics(t, func() {
			dt.TranslateWithProgress(ctx, "test text", "context", nil, "session123")
		})
	})
	
	t.Run("GetStats", func(t *testing.T) {
		stats := dt.GetStats()
		
		// Should return empty stats
		assert.Equal(t, translator.TranslationStats{}, stats)
	})
	
	t.Run("GetName", func(t *testing.T) {
		name := dt.GetName()
		
		// Should return "distributed"
		assert.Equal(t, "distributed", name)
	})
}

// TestGenerateToken tests the generateToken handler
func TestGenerateToken(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with mock dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
		authService:        nil, // This will cause a panic when used
	}
	
	t.Run("generateToken without auth", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/auth/token", handler.generateToken)
		
		testData := map[string]interface{}{
			"user_id":  "test-user",
			"username": "test-username",
			"roles":    []string{"user"},
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/auth/token", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		// This will panic because authService is nil
		assert.Panics(t, func() {
			router.ServeHTTP(w, req)
		})
	})
}

// TestDiscoverWorkers tests the discoverWorkers handler
func TestDiscoverWorkers(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with mock dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	t.Run("discoverWorkers with nil manager", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/distributed/workers/discover", handler.discoverWorkers)
		
		testData := map[string]interface{}{
			"network_range": "192.168.1.0/24",
		}
		
		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/workers/discover", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// Should return 503 when distributedManager is nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

// TestPairWorker tests the pairWorker handler
func TestPairWorker(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with mock dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	t.Run("pairWorker with nil manager", func(t *testing.T) {
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
	})
}
func TestAPIErrorHandling(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test handler with nil dependencies
	cfg := &config.Config{}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	
	handler := &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              wsHub,
		distributedManager: nil,
	}
	
	t.Run("Handling requests with nil dependencies", func(t *testing.T) {
		// Most handlers should fail gracefully with nil dependencies
		router := gin.New()
		
		// Add routes that would fail
		router.GET("/api/v1/monitoring/version/metrics", handler.getVersionMetrics)
		router.POST("/api/v1/distributed/translate", handler.translateDistributed)
		
		// Test metrics endpoint
		req1, _ := http.NewRequest("GET", "/api/v1/monitoring/version/metrics", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusServiceUnavailable, w1.Code)
		
		// Test distributed translate endpoint
		testData := map[string]interface{}{
			"text":        "Test",
			"source_lang": "en",
			"target_lang": "es",
		}
		jsonData, _ := json.Marshal(testData)
		req2, _ := http.NewRequest("POST", "/api/v1/distributed/translate", bytes.NewBuffer(jsonData))
		req2.Header.Set("Content-Type", "application/json")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusServiceUnavailable, w2.Code)
	})
}