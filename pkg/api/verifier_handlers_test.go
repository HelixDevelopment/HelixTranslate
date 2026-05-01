package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/verifier"
)

func setupVerifierRouter(t *testing.T) (*gin.Engine, *VerifierHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	cfg := &verifier.Config{
		APIURL:            "",
		APIKey:            "",
		CacheTTL:          time.Hour,
		MinScoreThreshold: 0.0,
		ScoringWeights: verifier.ScoreWeights{
			ResponseSpeed:     0.20,
			CostEffectiveness: 0.30,
			ModelEfficiency:   0.25,
			Capability:        0.20,
			Recency:           0.05,
		},
	}

	handler := NewVerifierHandler(cfg)
	handler.RegisterVerifierRoutes(router.Group("/api/v1"))
	return router, handler
}

func TestVerifierHandler_listVerifiedModels(t *testing.T) {
	router, _ := setupVerifierRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models", nil)
	router.ServeHTTP(w, req)

	// Without a mock server, this will return service unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestVerifierHandler_getVerifiedModel_NotFound(t *testing.T) {
	router, _ := setupVerifierRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verified-models/nonexistent-model", nil)
	router.ServeHTTP(w, req)

	// Model not found returns 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVerifierHandler_getVerificationStatus(t *testing.T) {
	router, _ := setupVerifierRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/verification-status", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["llmsverifier_connected"])
	assert.Equal(t, 0.0, resp["min_score_threshold"])
}

func TestVerifierHandler_refreshVerification(t *testing.T) {
	router, _ := setupVerifierRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/verification/refresh", nil)
	router.ServeHTTP(w, req)

	// Without mock server, refresh will fail
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestVerifierHandler_translateWithVerification_Validation(t *testing.T) {
	router, _ := setupVerifierRouter(t)

	// Missing text
	w := httptest.NewRecorder()
	body := `{"source_lang":"en","target_lang":"sr"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/translate-with-verification", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifierHandler_translateWithVerification_MissingModel(t *testing.T) {
	router, _ := setupVerifierRouter(t)

	w := httptest.NewRecorder()
	body := `{"text":"hello","source_lang":"en","target_lang":"sr","model_id":"nonexistent"}`
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/translate-with-verification", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Model not found in empty cache
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInitVerifierFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LLMsVerifier.APIURL = "http://test:8080"
	cfg.LLMsVerifier.APIKey = "test-key"
	cfg.LLMsVerifier.MinScoreThreshold = 0.5

	handler := InitVerifierFromConfig(cfg)
	require.NotNil(t, handler)
	assert.Equal(t, "http://test:8080", handler.config.APIURL)
	assert.Equal(t, "test-key", handler.config.APIKey)
	assert.Equal(t, 0.5, handler.config.MinScoreThreshold)
}

func TestVerifierHandler_RegisterVerifierRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	cfg := &verifier.Config{
		CacheTTL: time.Hour,
	}
	handler := NewVerifierHandler(cfg)
	handler.RegisterVerifierRoutes(router.Group("/api/v1"))

	// Verify all routes are registered by making requests
	routes := router.Routes()
	require.NotEmpty(t, routes)

	expectedPaths := map[string]bool{
		"/api/v1/verified-models":             false,
		"/api/v1/verified-models/:id":         false,
		"/api/v1/verification-status":         false,
		"/api/v1/verification/refresh":        false,
		"/api/v1/providers/verified":          false,
		"/api/v1/translate-with-verification": false,
	}

	for _, r := range routes {
		if _, exists := expectedPaths[r.Path]; exists {
			expectedPaths[r.Path] = true
		}
	}

	for path, found := range expectedPaths {
		assert.True(t, found, "route %s not registered", path)
	}
}
