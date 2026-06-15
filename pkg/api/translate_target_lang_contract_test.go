package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator/llm"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// §11.4.115 RED-baseline polarity switch. With RED_MODE=1 the test reproduces
// the defect on the pre-fix behaviour expectation (target language hardcoded to
// Serbian); with RED_MODE=0 (default) it is the standing GREEN regression guard
// asserting the request's target_lang actually configures the translator.
func redMode() bool { return os.Getenv("RED_MODE") == "1" }

func newContractHandler() *Handler {
	cfg := &config.Config{
		Translation: config.TranslationConfig{
			DefaultProvider: "openai",
			Providers: map[string]config.ProviderConfig{
				"openai": {APIKey: "test-key", Model: "gpt-3.5-turbo"},
			},
		},
	}
	eventBus := events.NewEventBus()
	return &Handler{
		config:             cfg,
		eventBus:           eventBus,
		wsHub:              websocket.NewHub(eventBus),
		distributedManager: nil,
	}
}

// TestCreateTranslator_TargetLangReachesTranslator proves the seam: the language
// pair passed to createTranslator is the pair the constructed translator will
// translate into. Before the fix createTranslator hardcoded ru→sr, so a Spanish
// request silently produced a Serbian translator. This is the contract.
func TestCreateTranslator_TargetLangReachesTranslator(t *testing.T) {
	h := newContractHandler()

	t.Run("explicit Spanish target is honoured (not Serbian)", func(t *testing.T) {
		trans, err := h.createTranslator("openai", "gpt-3.5-turbo", "en", "es")
		require.NoError(t, err)

		lt, ok := trans.(*llm.LLMTranslator)
		require.True(t, ok, "expected an *llm.LLMTranslator")

		cfg := lt.Config()

		if redMode() {
			// RED: reproduce the pre-fix defect — target was hardcoded to "sr".
			assert.Equal(t, "sr", cfg.TargetLang,
				"RED_MODE expects the historical hardcoded Serbian target")
			return
		}

		// GREEN guard: the request's target_lang must reach the translator.
		assert.Equal(t, "es", cfg.TargetLang, "target_lang=es must configure a Spanish translator")
		assert.Equal(t, "en", cfg.SourceLang, "source_lang=en must reach the translator")
		assert.NotEqual(t, "sr", cfg.TargetLang, "Spanish request must NOT yield the hardcoded Serbian target")
	})

	t.Run("omitted langs preserve legacy Russian->Serbian default (backward compat)", func(t *testing.T) {
		trans, err := h.createTranslator("openai", "gpt-3.5-turbo", "", "")
		require.NoError(t, err)
		lt, ok := trans.(*llm.LLMTranslator)
		require.True(t, ok)
		cfg := lt.Config()
		assert.Equal(t, "ru", cfg.SourceLang)
		assert.Equal(t, "sr", cfg.TargetLang)
	})
}

// TestTranslateRequest_JSONShape_BothSides asserts the request JSON wire format
// on both sides (§11.4.5 contract): a client sending target_lang/source_lang is
// bound into the handler's request struct, and that value drives the translator
// the handler builds. We use the real factory + the documented Spanish prompt as
// the user-visible evidence that the Spanish target actually took effect.
func TestTranslateRequest_JSONShape_BothSides(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newContractHandler()

	// Sending side: the JSON shape the client uses.
	body := map[string]interface{}{
		"text":        "Hello world",
		"provider":    "openai",
		"model":       "gpt-3.5-turbo",
		"source_lang": "en",
		"target_lang": "spanish",
	}
	jsonData, _ := json.Marshal(body)

	// Receiving side: bind into the SAME anonymous struct shape the handler uses
	// and assert the fields land — this is the wire-format contract on the seam.
	var bound struct {
		Text       string `json:"text" binding:"required"`
		Provider   string `json:"provider"`
		Model      string `json:"model"`
		Context    string `json:"context"`
		Script     string `json:"script"`
		SourceLang string `json:"source_lang"`
		TargetLang string `json:"target_lang"`
	}
	require.NoError(t, json.Unmarshal(jsonData, &bound))
	require.Equal(t, "spanish", bound.TargetLang, "target_lang must bind from JSON")
	require.Equal(t, "en", bound.SourceLang, "source_lang must bind from JSON")

	// And the bound values must produce a Spanish-configured translator whose
	// generated prompt names Spanish, not Serbian.
	trans, err := h.createTranslator(bound.Provider, bound.Model, bound.SourceLang, bound.TargetLang)
	require.NoError(t, err)
	lt := trans.(*llm.LLMTranslator)
	prompt := lt.BuildContractPrompt("Hello world", "")

	if redMode() {
		assert.Contains(t, prompt, "Serbian", "RED_MODE expects the hardcoded Serbian prompt")
		return
	}
	assert.True(t, strings.Contains(prompt, "Spanish"),
		"prompt for target_lang=spanish must mention Spanish; got:\n%s", prompt)
	assert.False(t, strings.Contains(prompt, "Russian to Serbian"),
		"Spanish request must NOT produce the Russian→Serbian prompt")

	// Also drive the full HTTP path to confirm the request is accepted and routed
	// (no API key for a real call here, so we accept OK or upstream-error, but a
	// 400 would mean the new fields broke binding — that must NOT happen).
	router := gin.New()
	router.POST("/translate", h.translateText)
	req, _ := http.NewRequest("POST", "/translate", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusBadRequest, w.Code,
		"adding source_lang/target_lang must not break request binding")
}
