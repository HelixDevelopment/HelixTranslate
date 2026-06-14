package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/script"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newScriptTestHandler builds a Handler wired to the mock translator with no
// distributed manager — usable entirely under httptest with zero external deps.
func newScriptTestHandler() (*Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Translation: config.TranslationConfig{
			DefaultProvider: "mock",
			Providers: map[string]config.ProviderConfig{
				"mock": {APIKey: "test-key", Model: "mock"},
			},
		},
	}
	eventBus := events.NewEventBus()
	wsHub := websocket.NewHub(eventBus)
	h := &Handler{config: cfg, eventBus: eventBus, wsHub: wsHub, distributedManager: nil}
	router := gin.New()
	router.POST("/translate", h.translateText)
	return h, router
}

func doTranslate(t *testing.T, router *gin.Engine, body map[string]interface{}) (int, map[string]interface{}) {
	t.Helper()
	jsonData, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/translate", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

// TestTranslateText_ScriptCyrillic_Honored is the §11.4.115 reproduce-first
// guard for the silent-script-drop bug: translateText only honored
// script=="latin" and silently ignored script=="cyrillic", returning Latin
// text to a client that explicitly asked for Cyrillic — a response-correctness
// defect (convertScript handles both, so the capability exists). On the
// pre-fix code this FAILs (translated is still raw Latin).
func TestTranslateText_ScriptCyrillic_Honored(t *testing.T) {
	_, router := newScriptTestHandler()

	mockOutput := "Translated: Hello world" // mock translator's deterministic output
	wantCyrillic := script.NewConverter().ToCyrillic(mockOutput)

	// Sanity (FACT): cyrillic conversion is non-trivial for this Latin string.
	assert.NotEqual(t, mockOutput, wantCyrillic,
		"precondition: ToCyrillic must transform the Latin mock output")

	code, resp := doTranslate(t, router, map[string]interface{}{
		"text":     "Hello world",
		"provider": "mock",
		"model":    "mock",
		"script":   "cyrillic",
	})

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, wantCyrillic, resp["translated"],
		"script=cyrillic MUST be honored, not silently dropped")
}

// TestTranslateText_ScriptInvalid_Rejected guards that an unknown script value
// is rejected with 400 rather than silently no-op'd (a client typo for
// "cyrillic"/"latin" must not silently return un-converted text). Pre-fix the
// handler accepts ANY script value and returns 200 with raw text.
func TestTranslateText_ScriptInvalid_Rejected(t *testing.T) {
	_, router := newScriptTestHandler()

	code, resp := doTranslate(t, router, map[string]interface{}{
		"text":     "Hello world",
		"provider": "mock",
		"model":    "mock",
		"script":   "klingon",
	})

	assert.Equal(t, http.StatusBadRequest, code,
		"unknown script value MUST be rejected, not silently ignored")
	assert.Contains(t, resp, "error")
}

// TestTranslateText_ScriptLatin_StillWorks is the regression guard that the
// fix does not break the previously-working latin path.
func TestTranslateText_ScriptLatin_StillWorks(t *testing.T) {
	_, router := newScriptTestHandler()

	mockOutput := "Translated: Hello world"
	wantLatin := script.NewConverter().ToLatin(mockOutput)

	code, resp := doTranslate(t, router, map[string]interface{}{
		"text":     "Hello world",
		"provider": "mock",
		"model":    "mock",
		"script":   "latin",
	})

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, wantLatin, resp["translated"])
}

// TestTranslateText_ScriptOmitted_NoConversion guards that omitting script
// returns the translation unchanged (default behavior preserved).
func TestTranslateText_ScriptOmitted_NoConversion(t *testing.T) {
	_, router := newScriptTestHandler()

	code, resp := doTranslate(t, router, map[string]interface{}{
		"text":     "Hello world",
		"provider": "mock",
		"model":    "mock",
	})

	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "Translated: Hello world", resp["translated"])
}
