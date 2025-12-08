package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestUpdatePackageMoreCoverage adds more test cases for update package handlers
func TestUpdatePackageMoreCoverage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	t.Run("applyUpdatePackage with valid update structure", func(t *testing.T) {
		// Create a temporary directory to simulate an update package
		tempDir, err := os.MkdirTemp("", "test-update-*")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a simple tar.gz file (this will fail extraction)
		tarPath := filepath.Join(tempDir, "update.tar.gz")
		err = os.WriteFile(tarPath, []byte("not a real tar file"), 0644)
		assert.NoError(t, err)

		// Test applyUpdatePackage with our fake tar file
		err = applyUpdatePackage(tarPath)
		assert.Error(t, err)
		// The actual error might be "failed to create backup" or "failed to extract update package"
		assert.True(t,
			strings.Contains(err.Error(), "failed to create backup") ||
				strings.Contains(err.Error(), "failed to extract update package"))
	})

	t.Run("applyUpdatePackage with non-existent file", func(t *testing.T) {
		// Test with completely non-existent file
		err := applyUpdatePackage("/path/that/does/not/exist.tar.gz")
		assert.Error(t, err)
	})

	t.Run("rollbackUpdate with nil distributed manager", func(t *testing.T) {
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
		router.POST("/api/v1/update/rollback", handler.rollbackUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/rollback", nil)
		req.Header.Set("X-Worker-ID", "test-worker")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 503 when distributedManager is nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Distributed work not available", response["error"])
	})

	t.Run("rollbackUpdate with invalid distributed manager", func(t *testing.T) {
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
		router.POST("/api/v1/update/rollback", handler.rollbackUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/rollback", nil)
		req.Header.Set("X-Worker-ID", "test-worker")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 500 when distributedManager is invalid
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("rollbackUpdate with missing worker ID", func(t *testing.T) {
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
		router.POST("/api/v1/update/rollback", handler.rollbackUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/rollback", nil)
		// No X-Worker-ID header or worker_id query param
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 when worker ID is missing
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Worker ID is required", response["error"])
	})

	t.Run("rollbackUpdate with worker ID in query param", func(t *testing.T) {
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
		router.POST("/api/v1/update/rollback", handler.rollbackUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/rollback?worker_id=test-worker", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 500 when distributedManager is invalid (worker ID is provided correctly)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("applyUpdate with missing version header", func(t *testing.T) {
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
		router.POST("/api/v1/update/apply", handler.applyUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/apply", nil)
		// No X-Update-Version header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 when version header is missing
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Update version not specified", response["error"])
	})

	t.Run("applyUpdate with nil distributed manager", func(t *testing.T) {
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
		router.POST("/api/v1/update/apply", handler.applyUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/apply", nil)
		req.Header.Set("X-Update-Version", "1.0.0")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 404 when update package doesn't exist
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Update package not found", response["error"])
	})

	t.Run("applyUpdate with non-existent update package", func(t *testing.T) {
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
		router.POST("/api/v1/update/apply", handler.applyUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/apply", nil)
		req.Header.Set("X-Update-Version", "non-existent-version")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 404 when update package doesn't exist
		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Update package not found", response["error"])
	})

	t.Run("uploadUpdate with nil translator", func(t *testing.T) {
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
		router.POST("/api/v1/update/upload", handler.uploadUpdate)

		// Create a simple file
		jsonData := []byte(`{"test": "data"}`)
		req, _ := http.NewRequest("POST", "/api/v1/update/upload", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 when translator is nil
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "No update package provided", response["error"])
	})
}
