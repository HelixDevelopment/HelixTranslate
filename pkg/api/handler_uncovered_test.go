package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/distributed"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// MockDistributedManager is a mock implementation for testing
type MockDistributedManager struct {
	translateResult string
	translateError  error
}

func (m *MockDistributedManager) TranslateDistributed(ctx context.Context, text, contextHint string) (string, error) {
	if m.translateError != nil {
		return "", m.translateError
	}
	return m.translateResult, nil
}

func (m *MockDistributedManager) Initialize(localCoordinator interface{}) error {
	return nil
}

func (m *MockDistributedManager) DiscoverAndPairWorkers(ctx context.Context) error {
	return nil
}

func (m *MockDistributedManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{"status": "ok"}
}

func (m *MockDistributedManager) AddWorker(workerID string, workerCfg *distributed.WorkerConfig) error {
	return nil
}

func (m *MockDistributedManager) RemoveWorker(workerID string) error {
	return nil
}

func (m *MockDistributedManager) PairWorker(workerID string) error {
	return nil
}

func (m *MockDistributedManager) UnpairWorker(workerID string) error {
	return nil
}

func (m *MockDistributedManager) GetWorkerByID(workerID string) *distributed.RemoteService {
	return nil
}

func (m *MockDistributedManager) RollbackWorker(ctx context.Context, service *distributed.RemoteService) error {
	return nil
}

func (m *MockDistributedManager) GetVersionMetrics() *distributed.VersionMetrics {
	return &distributed.VersionMetrics{}
}

func (m *MockDistributedManager) GetVersionAlerts() []*distributed.DriftAlert {
	return []*distributed.DriftAlert{}
}

func (m *MockDistributedManager) GetVersionHealth() map[string]interface{} {
	return map[string]interface{}{"status": "healthy"}
}

func (m *MockDistributedManager) GetPairedServices() map[string]*distributed.RemoteService {
	return make(map[string]*distributed.RemoteService)
}

func (m *MockDistributedManager) CheckVersionDrift(ctx context.Context) []*distributed.DriftAlert {
	return []*distributed.DriftAlert{}
}

func (m *MockDistributedManager) GetAlertHistory(limit int) []*distributed.DriftAlert {
	return []*distributed.DriftAlert{}
}

func (m *MockDistributedManager) AcknowledgeAlert(alertID, acknowledgedBy string) bool {
	return true
}

func (m *MockDistributedManager) AddAlertChannel(channel distributed.AlertChannel) {
	// Do nothing
}

func (m *MockDistributedManager) Close() error {
	return nil
}

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
		// Test unpairWorker handler with nil manager
		router := gin.New()
		router.DELETE("/api/v1/distributed/workers/:worker_id/pair", handler.unpairWorker)

		req, _ := http.NewRequest("DELETE", "/api/v1/distributed/workers/test-worker/pair", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("addSlackAlertChannel_InvalidManagerType", func(t *testing.T) {
		// Test addSlackAlertChannel handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/slack", handlerWithMock.addSlackAlertChannel)

		testData := map[string]interface{}{
			"webhook_url": "https://hooks.slack.com/test",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/slack", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("addSlackAlertChannel_MissingRequiredFields", func(t *testing.T) {
		// Test addSlackAlertChannel handler with missing required fields
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/slack", handlerWithMock.addSlackAlertChannel)

		testData := map[string]interface{}{
			// Missing webhook_url field
			"channel": "#alerts",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/slack", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with binding validation error for missing required field
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("addSlackAlertChannel_InvalidJSON", func(t *testing.T) {
		// Test addSlackAlertChannel handler with invalid JSON
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/slack", handlerWithMock.addSlackAlertChannel)

		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/slack", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with bad request due to invalid JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("translateDistributed_InvalidJSON", func(t *testing.T) {
		// Test translateDistributed handler with invalid JSON
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/distributed/translate", handlerWithMock.translateDistributed)

		req, _ := http.NewRequest("POST", "/api/v1/distributed/translate", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("translateDistributed_MissingText", func(t *testing.T) {
		// Test translateDistributed handler with missing required text field
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/distributed/translate", handlerWithMock.translateDistributed)

		testData := map[string]interface{}{
			"context_hint": "translation context",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("translateDistributed_TranslationError", func(t *testing.T) {
		// Test translateDistributed handler with translation error
		mockDM := &MockDistributedManager{
			translateResult: "",
			translateError:  assert.AnError,
		}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/distributed/translate", handlerWithMock.translateDistributed)

		testData := map[string]interface{}{
			"text": "Test text",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return internal server error
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("translateDistributed_InvalidManagerType_NoSessionID", func(t *testing.T) {
		// Test translateDistributed handler without session ID header (invalid manager type)
		mockDM := &MockDistributedManager{
			translateResult: "Translated text",
			translateError:  nil,
		}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/distributed/translate", handlerWithMock.translateDistributed)

		testData := map[string]interface{}{
			"text": "Test text",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/distributed/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("uploadUpdate", func(t *testing.T) {
		// Test uploadUpdate handler with no file
		router := gin.New()
		router.POST("/api/v1/update/upload", handler.uploadUpdate)

		req, _ := http.NewRequest("POST", "/api/v1/update/upload", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with no update package provided
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "No update package provided", response["error"])
	})

	t.Run("uploadUpdate_NoVersionHeader", func(t *testing.T) {
		// Test uploadUpdate handler with file but no version header
		router := gin.New()
		router.POST("/api/v1/update/upload", handler.uploadUpdate)

		// Create a multipart form with a file
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Create a fake file part
		part, err := writer.CreateFormFile("update_package", "test.tar.gz")
		assert.NoError(t, err)
		part.Write([]byte("fake update content"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/v1/update/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		// Note: No X-Update-Version header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with missing version header
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Update version not specified", response["error"])
	})

	t.Run("applyUpdate", func(t *testing.T) {
		// Test applyUpdate handler with no version header
		router := gin.New()
		router.POST("/api/v1/update/apply", handler.applyUpdate)

		testData := map[string]interface{}{
			"update_id": "test-update",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/update/apply", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		// Note: No X-Update-Version header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 for missing version header
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Update version not specified", response["error"])
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
		// Test getVersionMetrics handler with nil manager
		router := gin.New()
		router.GET("/api/v1/monitoring/version/metrics", handler.getVersionMetrics)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("getVersionMetrics_InvalidManagerType", func(t *testing.T) {
		// Test getVersionMetrics handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.GET("/api/v1/monitoring/version/metrics", handlerWithMock.getVersionMetrics)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("getVersionAlerts", func(t *testing.T) {
		// Test getVersionAlerts handler with nil manager
		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts", handler.getVersionAlerts)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("getVersionAlerts_InvalidManagerType", func(t *testing.T) {
		// Test getVersionAlerts handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts", handlerWithMock.getVersionAlerts)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("getVersionHealth", func(t *testing.T) {
		// Test getVersionHealth handler with nil manager
		router := gin.New()
		router.GET("/api/v1/monitoring/version/health", handler.getVersionHealth)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("getVersionHealth_InvalidManagerType", func(t *testing.T) {
		// Test getVersionHealth handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.GET("/api/v1/monitoring/version/health", handlerWithMock.getVersionHealth)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("getVersionDashboard", func(t *testing.T) {
		// Test getVersionDashboard handler with nil manager
		router := gin.New()
		router.GET("/api/v1/monitoring/version/dashboard", handler.getVersionDashboard)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("getVersionDashboard_InvalidManagerType", func(t *testing.T) {
		// Test getVersionDashboard handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.GET("/api/v1/monitoring/version/dashboard", handlerWithMock.getVersionDashboard)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/dashboard", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("triggerVersionDriftCheck", func(t *testing.T) {
		// Test triggerVersionDriftCheck handler
		router := gin.New()
		router.POST("/api/v1/monitoring/version/drift-check", handler.triggerVersionDriftCheck)

		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/drift-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("triggerVersionDriftCheck_InvalidManagerType", func(t *testing.T) {
		// Test triggerVersionDriftCheck handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/drift-check", handlerWithMock.triggerVersionDriftCheck)

		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/drift-check", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
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

	t.Run("getAlertHistory_InvalidManagerType", func(t *testing.T) {
		// Test getAlertHistory handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts/history", handlerWithMock.getAlertHistory)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts/history", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("getAlertHistory_WithLimit", func(t *testing.T) {
		// Test getAlertHistory handler with limit parameter
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts/history", handlerWithMock.getAlertHistory)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts/history?limit=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type (but would succeed with valid manager)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("getAlertHistory_InvalidLimit", func(t *testing.T) {
		// Test getAlertHistory handler with invalid limit parameter
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.GET("/api/v1/monitoring/version/alerts/history", handlerWithMock.getAlertHistory)

		req, _ := http.NewRequest("GET", "/api/v1/monitoring/version/alerts/history?limit=invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type (but would use default limit with valid manager)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("acknowledgeAlert", func(t *testing.T) {
		// Test acknowledgeAlert handler with nil manager
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/:alert_id/acknowledge", handler.acknowledgeAlert)

		testData := map[string]interface{}{
			"acknowledged_by": "test-user",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/test-alert/acknowledge", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("acknowledgeAlert_InvalidManagerType", func(t *testing.T) {
		// Test acknowledgeAlert handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/:alert_id/acknowledge", handlerWithMock.acknowledgeAlert)

		testData := map[string]interface{}{
			"acknowledged_by": "test-user",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/test-alert/acknowledge", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("acknowledgeAlert_MissingAcknowledgedBy", func(t *testing.T) {
		// Test acknowledgeAlert handler with missing acknowledged_by field
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/:alert_id/acknowledge", handlerWithMock.acknowledgeAlert)

		testData := map[string]interface{}{
			// Missing acknowledged_by field
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/test-alert/acknowledge", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with binding validation error for missing required field
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("addEmailAlertChannel", func(t *testing.T) {
		// Test addEmailAlertChannel handler with nil manager
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/email", handler.addEmailAlertChannel)

		testData := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
			"username":     "test@example.com",
			"password":     "password",
			"from_address": "test@example.com",
			"to_addresses": []string{"alerts@example.com"},
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/email", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail gracefully with nil distributedManager
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("addEmailAlertChannel_InvalidManagerType", func(t *testing.T) {
		// Test addEmailAlertChannel handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/email", handlerWithMock.addEmailAlertChannel)

		testData := map[string]interface{}{
			"smtp_host":    "smtp.example.com",
			"smtp_port":    587,
			"username":     "test@example.com",
			"password":     "password",
			"from_address": "test@example.com",
			"to_addresses": []string{"alerts@example.com"},
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/email", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("addEmailAlertChannel_MissingRequiredFields", func(t *testing.T) {
		// Test addEmailAlertChannel handler with missing required fields
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/email", handlerWithMock.addEmailAlertChannel)

		testData := map[string]interface{}{
			"smtp_host": "smtp.example.com",
			// Missing other required fields
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/email", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with binding validation error for missing required fields
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("addWebhookAlertChannel", func(t *testing.T) {
		// Test addWebhookAlertChannel handler
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/webhook", handler.addWebhookAlertChannel)

		testData := map[string]interface{}{
			"url":          "https://example.com/webhook",
			"enabled":      true,
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

	t.Run("addWebhookAlertChannel_InvalidManagerType", func(t *testing.T) {
		// Test addWebhookAlertChannel handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/webhook", handlerWithMock.addWebhookAlertChannel)

		testData := map[string]interface{}{
			"url": "https://example.com/webhook",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/webhook", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("addWebhookAlertChannel_MissingRequiredFields", func(t *testing.T) {
		// Test addWebhookAlertChannel handler with missing required fields
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/webhook", handlerWithMock.addWebhookAlertChannel)

		testData := map[string]interface{}{
			// Missing url field
			"method": "POST",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/webhook", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with binding validation error for missing required field
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("addWebhookAlertChannel_InvalidJSON", func(t *testing.T) {
		// Test addWebhookAlertChannel handler with invalid JSON
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/webhook", handlerWithMock.addWebhookAlertChannel)

		req, _ := http.NewRequest("POST", "/api/v1/monitoring/version/alerts/channels/webhook", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with bad request due to invalid JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("addSlackAlertChannel", func(t *testing.T) {
		// Test addSlackAlertChannel handler
		router := gin.New()
		router.POST("/api/v1/monitoring/version/alerts/channels/slack", handler.addSlackAlertChannel)

		testData := map[string]interface{}{
			"webhook_url":  "https://hooks.slack.com/test",
			"channel":      "#alerts",
			"enabled":      true,
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

	t.Run("login_InvalidJSON", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/auth/login", handler.login)

		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return bad request for invalid JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("login_MissingFields", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/auth/login", handler.login)

		testData := map[string]interface{}{
			"username": "testuser",
			// Missing password
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return bad request for missing required fields
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("generateToken_InvalidJSON", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/auth/token", handler.generateToken)

		req, _ := http.NewRequest("POST", "/api/v1/auth/token", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Should return bad request for invalid JSON
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("generateToken_MissingFields", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/auth/token", handler.generateToken)

		testData := map[string]interface{}{
			"username": "testuser",
			// Missing user_id
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/auth/token", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Should return bad request for missing required fields
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
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

		req, _ := http.NewRequest("POST", "/api/v1/distributed/workers/discover", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 503 when distributedManager is nil
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})

	t.Run("discoverWorkers_InvalidManagerType", func(t *testing.T) {
		// Test discoverWorkers handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/distributed/workers/discover", handlerWithMock.discoverWorkers)

		req, _ := http.NewRequest("POST", "/api/v1/distributed/workers/discover", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
	})

	t.Run("translateText_InvalidJSON", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/translate", handler.translateText)

		req, _ := http.NewRequest("POST", "/api/v1/translate", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return bad request for invalid JSON
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("translateText_MissingText", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/translate", handler.translateText)

		testData := map[string]interface{}{
			"provider": "dictionary",
			// Missing required text field
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return bad request for missing required field
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("translateText_InvalidProvider", func(t *testing.T) {
		router := gin.New()
		router.POST("/api/v1/translate", handler.translateText)

		testData := map[string]interface{}{
			"text":     "Hello world",
			"provider": "invalid-provider",
		}

		jsonData, _ := json.Marshal(testData)
		req, _ := http.NewRequest("POST", "/api/v1/translate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return bad request for invalid provider
		assert.Equal(t, http.StatusBadRequest, w.Code)
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

	t.Run("pairWorker_InvalidManagerType", func(t *testing.T) {
		// Test pairWorker handler with mock manager (wrong type)
		mockDM := &MockDistributedManager{}

		handlerWithMock := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: mockDM,
		}

		router := gin.New()
		router.POST("/api/v1/distributed/workers/:worker_id/pair", handlerWithMock.pairWorker)

		req, _ := http.NewRequest("POST", "/api/v1/distributed/workers/test-worker/pair", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should fail with invalid manager type
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid distributed manager", response["error"])
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
