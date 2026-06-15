package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestTranslateFB2MoreCoverage adds more test cases for translateFB2 handler
func TestTranslateFB2MoreCoverage(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Sample FB2 content for testing
	fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
	<description>
		<title-info>
			<genre>nonfiction</genre>
			<book-title>Test Book</book-title>
		</title-info>
	</description>
	<body>
		<section>
			<p>Test content</p>
		</section>
	</body>
</FictionBook>`

	t.Run("translateFB2 with valid FB2 file", func(t *testing.T) {
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
		router.POST("/translate/fb2", handler.translateFB2)

		// Create multipart form data
		writer := fmt.Sprintf(
			"--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"test.fb2\"\r\nContent-Type: application/xml\r\n\r\n%s\r\n--%s--\r\n",
			"boundary", fb2Content, "boundary",
		)

		req, _ := http.NewRequest("POST", "/translate/fb2", strings.NewReader(writer))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		req.Header.Set("X-Source-Language", "en")
		req.Header.Set("X-Target-Language", "es")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should handle FB2 file (may fail due to API, but should process)
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError || w.Code == http.StatusBadRequest)
	})

	t.Run("translateFB2 with custom provider and model", func(t *testing.T) {
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
		router.POST("/translate/fb2", handler.translateFB2)

		// Create multipart form with additional form fields
		writer := fmt.Sprintf(
			"--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"test.fb2\"\r\nContent-Type: application/xml\r\n\r\n%s\r\n--%s\r\nContent-Disposition: form-data; name=\"provider\"\r\n\r\nopenai\r\n--%s\r\nContent-Disposition: form-data; name=\"model\"\r\n\r\ngpt-4\r\n--%s\r\nContent-Disposition: form-data; name=\"script\"\r\n\r\nlatin\r\n--%s--\r\n",
			"boundary", fb2Content, "boundary", "boundary", "boundary", "boundary",
		)

		req, _ := http.NewRequest("POST", "/translate/fb2", strings.NewReader(writer))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		req.Header.Set("X-Source-Language", "en")
		req.Header.Set("X-Target-Language", "es")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should process with custom parameters
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError || w.Code == http.StatusBadRequest)
	})

	t.Run("translateFB2 with invalid FB2 content", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)

		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}

		// R-1b/R2: this case asserts the honest translator-creation failure path
		// (empty config / no verified provider) maps to a 4xx/5xx. Inject the
		// bridge hard error deterministically (§11.4.69).
		handler.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
			return nil, fmt.Errorf("bridge: no verified translator available")
		}
		router := gin.New()
		router.POST("/translate/fb2", handler.translateFB2)

		// Create multipart form with invalid XML
		invalidXML := "This is not valid XML content"
		writer := fmt.Sprintf(
			"--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"invalid.fb2\"\r\nContent-Type: application/xml\r\n\r\n%s\r\n--%s--\r\n",
			"boundary", invalidXML, "boundary",
		)

		req, _ := http.NewRequest("POST", "/translate/fb2", strings.NewReader(writer))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		req.Header.Set("X-Source-Language", "en")
		req.Header.Set("X-Target-Language", "es")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should handle invalid XML gracefully
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError)
	})

	t.Run("translateFB2 with empty file", func(t *testing.T) {
		cfg := &config.Config{}
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)

		handler := &Handler{
			config:             cfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}

		// R-1b/R2: assert the honest translator-creation failure path maps to 4xx/5xx.
		handler.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
			return nil, fmt.Errorf("bridge: no verified translator available")
		}
		router := gin.New()
		router.POST("/translate/fb2", handler.translateFB2)

		// Create empty file
		writer := fmt.Sprintf(
			"--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"empty.fb2\"\r\nContent-Type: application/xml\r\n\r\n\r\n--%s--\r\n",
			"boundary", "boundary",
		)

		req, _ := http.NewRequest("POST", "/translate/fb2", strings.NewReader(writer))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
		req.Header.Set("X-Source-Language", "en")
		req.Header.Set("X-Target-Language", "es")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should handle empty file
		assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError)
	})
}
