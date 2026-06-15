package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// §11.4.115 RED-baseline polarity switch. With RED_MODE=1 the test reproduces
// the defect on the pre-fix behaviour expectation (target language hardcoded to
// Serbian); with RED_MODE=0 (default) it is the standing GREEN regression guard
// asserting the request's target_lang actually reaches the translator-construction
// seam (now the LLMsVerifier bridge task, per R-1b/R2).
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
// pair passed to createTranslator is the pair threaded into the bridge's
// selection.TaskRequirements (so the strongest verified model is selected FOR that
// pair). Before the fix createTranslator hardcoded ru→sr, so a Spanish request
// silently produced a Serbian translator. R-1b/R2 keeps this contract while
// sourcing the translator from the LLMsVerifier bridge instead of a local runtime.
func TestCreateTranslator_TargetLangReachesTranslator(t *testing.T) {
	h := newContractHandler()

	var captured selection.TaskRequirements
	h.bridgeTranslatorFactory = func(_ context.Context, task selection.TaskRequirements) (translator.Translator, error) {
		captured = task
		return &fakeBridgeTranslator{name: "bridge-verified", task: task}, nil
	}

	t.Run("explicit Spanish target is honoured (not Serbian)", func(t *testing.T) {
		trans, err := h.createTranslator("openai", "gpt-3.5-turbo", "en", "es")
		require.NoError(t, err)
		require.Equal(t, "bridge-verified", trans.GetName(),
			"createTranslator must return the bridge-sourced translator")

		if redMode() {
			// RED: reproduce the pre-fix defect — target was hardcoded to "sr".
			assert.Equal(t, "sr", captured.TargetLang,
				"RED_MODE expects the historical hardcoded Serbian target")
			return
		}

		// GREEN guard: the request's target_lang must reach the bridge task.
		assert.Equal(t, "es", captured.TargetLang, "target_lang=es must reach the bridge task")
		assert.Equal(t, "en", captured.SourceLang, "source_lang=en must reach the bridge task")
		assert.NotEqual(t, "sr", captured.TargetLang, "Spanish request must NOT yield the hardcoded Serbian target")
	})

	t.Run("omitted langs preserve legacy Russian->Serbian default (backward compat)", func(t *testing.T) {
		_, err := h.createTranslator("openai", "gpt-3.5-turbo", "", "")
		require.NoError(t, err)
		assert.Equal(t, "ru", captured.SourceLang)
		assert.Equal(t, "sr", captured.TargetLang)
	})
}

// TestTranslateRequest_JSONShape_BothSides asserts the request JSON wire format
// on both sides (§11.4.5 contract): a client sending target_lang/source_lang is
// bound into the handler's request struct, and that value drives the bridge task
// the handler builds. The injected bridge factory captures the task so we prove
// the Spanish target actually took effect end-to-end through the HTTP path.
func TestTranslateRequest_JSONShape_BothSides(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newContractHandler()

	var captured selection.TaskRequirements
	h.bridgeTranslatorFactory = func(_ context.Context, task selection.TaskRequirements) (translator.Translator, error) {
		captured = task
		return &mockBridgeTranslator{}, nil
	}

	// Sending side: the JSON shape the client uses.
	body := map[string]interface{}{
		"text":        "Hello world",
		"provider":    "openai",
		"model":       "gpt-3.5-turbo",
		"source_lang": "en",
		"target_lang": "es",
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
	require.Equal(t, "es", bound.TargetLang, "target_lang must bind from JSON")
	require.Equal(t, "en", bound.SourceLang, "source_lang must bind from JSON")

	// Drive the full HTTP path; the bridge factory captures the task it was asked
	// to build, proving the bound values reached the translator-construction seam.
	router := gin.New()
	router.POST("/translate", h.translateText)
	req, _ := http.NewRequest("POST", "/translate", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code,
		"adding source_lang/target_lang must not break request binding")

	if redMode() {
		assert.Equal(t, "sr", captured.TargetLang, "RED_MODE expects the hardcoded Serbian target")
		return
	}
	assert.Equal(t, "es", captured.TargetLang, "target_lang=es must reach the bridge task")
	assert.Equal(t, "en", captured.SourceLang, "source_lang=en must reach the bridge task")
}
