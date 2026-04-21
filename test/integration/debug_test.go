//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDebugStringTranslation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a custom gin handler that uses our mock translator
	router := gin.New()

	// Register our custom translation endpoint
	router.POST("/api/v1/translate/string", func(c *gin.Context) {
		mockTrans := &MockTranslator{}

		var req struct {
			Text           string `json:"text" binding:"required"`
			TargetLanguage string `json:"target_language" binding:"required"`
			Provider       string `json:"provider"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		translated, err := mockTrans.Translate(c.Request.Context(), req.Text, "")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"translated_text": translated,
			"source_language": "en",
			"target_language": req.TargetLanguage,
			"provider":        "mock",
		})
	})

	reqBody := map[string]interface{}{
		"text":            "Hello, world!",
		"target_language": "sr",
		"provider":        "mock",
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/translate/string", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	t.Logf("Status code: %d", w.Code)
	t.Logf("Response body: %s", w.Body.String())
}
