package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"digital.vasic.translator/internal/config"
)

// TestListProviders_ServesConfiguredProviders is the §11.4.115 reproduce-first
// polarity test for BUG-PROVIDERS-STATIC-LIST (docs/qa/deadcode_providers_
// investigation_20260616_160012/FINDING.md TASK 2).
//
// Bug: pkg/api/handler.go listProviders returned a hardcoded static list of
// exactly {openai, anthropic, zhipu, deepseek}, IGNORING h.config.Translation.
// Providers — so it mis-represented BOTH capability (16+ supported providers
// omitted) AND availability (claimed openai available with no key; never told a
// client that configured gemini/qwen exist).
//
// Polarity (RED_MODE env): when RED_MODE=1 this test asserts the BROKEN
// behaviour is present (proving the defect on the pre-fix artifact). When
// RED_MODE=0 (default, post-fix) it asserts the FIXED behaviour: the endpoint
// reflects the actually-configured provider set.
func TestListProviders_ServesConfiguredProviders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	redMode := os.Getenv("RED_MODE") == "1"

	// Config declares gemini + qwen as the configured providers, and NO openai.
	h := &Handler{
		config: &config.Config{
			Translation: config.TranslationConfig{
				DefaultProvider: "gemini",
				Providers: map[string]config.ProviderConfig{
					"gemini": {APIKey: "test-gemini-key", Model: "gemini-2.0-flash"},
					"qwen":   {APIKey: "test-qwen-key", Model: "qwen-plus"},
				},
			},
		},
	}

	router := gin.New()
	router.GET("/providers", h.listProviders)
	req, _ := http.NewRequest("GET", "/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	rawProviders, ok := response["providers"].([]interface{})
	assert.True(t, ok)

	names := map[string]bool{}
	for _, p := range rawProviders {
		if m, ok := p.(map[string]interface{}); ok {
			if n, ok := m["name"].(string); ok {
				names[n] = true
			}
		}
	}

	if redMode {
		// BROKEN artifact: static list ignores config -> openai present,
		// configured gemini/qwen absent. This assertion PASSES on the pre-fix
		// build (proving the defect is genuinely present), FAILS post-fix.
		assert.True(t, names["openai"],
			"RED: pre-fix static list wrongly contains unconfigured openai")
		assert.False(t, names["gemini"],
			"RED: pre-fix static list wrongly omits configured gemini")
		assert.False(t, names["qwen"],
			"RED: pre-fix static list wrongly omits configured qwen")
		return
	}

	// FIXED artifact: endpoint reflects the configured set.
	assert.True(t, names["gemini"],
		"configured provider gemini MUST be reported")
	assert.True(t, names["qwen"],
		"configured provider qwen MUST be reported")
	assert.False(t, names["openai"],
		"unconfigured openai MUST NOT be reported as a configured provider")

	// Each configured provider must report honest availability + its real model.
	for _, p := range rawProviders {
		m := p.(map[string]interface{})
		name := m["name"].(string)
		assert.Equal(t, true, m["configured"],
			"provider %s must be flagged configured:true", name)
		// configured providers in this fixture carry keys -> available:true
		assert.Equal(t, true, m["available"],
			"provider %s with a configured key must be available:true", name)
		models, _ := m["models"].([]interface{})
		assert.NotEmpty(t, models,
			"provider %s must report its configured model(s)", name)
	}
}

// TestListProviders_NilConfigFallbackIsCompleteCatalogue guards the defensive
// fallback: when no config is wired (zero-value Handler), the endpoint serves a
// static capability CATALOGUE that is honestly flagged configured:false AND is
// complete (not the old 4-provider lie). This keeps the legacy zero-config
// Handler tests working while never claiming an unconfigured provider is
// "available".
func TestListProviders_NilConfigFallbackIsCompleteCatalogue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if os.Getenv("RED_MODE") == "1" {
		t.Skip("RED_MODE targets the configured-path assertion")
	}

	h := &Handler{} // nil config
	router := gin.New()
	router.GET("/providers", h.listProviders)
	req, _ := http.NewRequest("GET", "/providers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	rawProviders, _ := response["providers"].([]interface{})

	names := map[string]bool{}
	for _, p := range rawProviders {
		m, ok := p.(map[string]interface{})
		if !assert.True(t, ok, "each provider entry must be an object") {
			continue
		}
		name, ok := m["name"].(string)
		if !assert.True(t, ok, "each provider entry must carry a string name") {
			continue
		}
		names[name] = true
		// Fallback catalogue entries are NOT configured.
		assert.Equal(t, false, m["configured"],
			"fallback catalogue entry %s must be configured:false", name)
	}
	// The fallback catalogue must be COMPLETE, not the stale 4-provider lie:
	// it must include providers the old static list omitted.
	for _, must := range []string{"openai", "anthropic", "deepseek", "gemini", "qwen", "groq", "mistral"} {
		assert.True(t, names[must],
			"fallback catalogue must include supported provider %s", must)
	}
}
