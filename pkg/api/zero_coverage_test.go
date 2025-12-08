package api

import (
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/websocket"
	"github.com/stretchr/testify/assert"
)

// TestZeroCoverageFunctions tests functions with 0% coverage
func TestZeroCoverageFunctions(t *testing.T) {
	// Test applyUpdatePackage function (0% coverage)
	t.Run("applyUpdatePackage", func(t *testing.T) {
		// This should fail for non-existent package
		err := applyUpdatePackage("/non/existent/package.tar.gz")
		assert.Error(t, err)
	})
	
	// Test createTranslator function (42.9% coverage)
	t.Run("createTranslator", func(t *testing.T) {
		// Test with valid config
		eventBus := events.NewEventBus()
		wsHub := websocket.NewHub(eventBus)
		
		validCfg := &config.Config{
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
		
		validHandler := &Handler{
			config:             validCfg,
			eventBus:           eventBus,
			wsHub:              wsHub,
			distributedManager: nil,
		}
		
		trans, err := validHandler.createTranslator("openai", "gpt-3.5-turbo")
		// Should either succeed or fail with expected error
		if err != nil {
			assert.Contains(t, err.Error(), "provider") // Might fail if provider is not supported
		} else {
			assert.NotNil(t, trans)
		}
	})
}