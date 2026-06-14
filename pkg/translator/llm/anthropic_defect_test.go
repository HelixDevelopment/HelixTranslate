package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnthropicMultiBlockContent_NotDropped is a §11.4.115 reproduce-first test
// for the data-loss defect where AnthropicClient.Translate returned only
// Content[0].Text, silently discarding every subsequent content block.
//
// Two real failure modes are reproduced:
//  1. A response split into multiple "text" blocks (Anthropic MAY split long
//     output) — only the first block was returned, truncating the translation.
//  2. A response whose FIRST block is a non-text block (e.g. "thinking" /
//     "redacted_thinking", emitted by extended-thinking models BEFORE the text
//     block) — Content[0].Text is empty, so the real translation in Content[1]
//     was discarded and an EMPTY string was returned.
func TestAnthropicMultiBlockContent_NotDropped(t *testing.T) {
	t.Run("multiple_text_blocks_concatenated", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Two text blocks — the full translation is "Привет, мир!"
			w.Write([]byte(`{"content":[{"type":"text","text":"Привет, "},{"type":"text","text":"мир!"}]}`))
		}))
		defer server.Close()

		client := &AnthropicClient{
			config:     TranslationConfig{Provider: "anthropic", APIKey: "k", Model: "claude-3-sonnet-20240229"},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			baseURL:    server.URL,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		got, err := client.Translate(ctx, "Hello, world!", "translate to Russian")
		require.NoError(t, err)
		// Pre-fix: got == "Привет, " (second block dropped).
		assert.Equal(t, "Привет, мир!", got, "all text blocks must be concatenated, not just Content[0]")
	})

	t.Run("thinking_block_first_then_text", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// First block is "thinking" (empty .text), real translation in 2nd block.
			w.Write([]byte(`{"content":[{"type":"thinking","text":""},{"type":"text","text":"Bonjour le monde"}]}`))
		}))
		defer server.Close()

		client := &AnthropicClient{
			config:     TranslationConfig{Provider: "anthropic", APIKey: "k", Model: "claude-3-sonnet-20240229"},
			httpClient: &http.Client{Timeout: 5 * time.Second},
			baseURL:    server.URL,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		got, err := client.Translate(ctx, "Hello world", "translate to French")
		require.NoError(t, err)
		// Pre-fix: got == "" (Content[0] is the empty thinking block).
		assert.Equal(t, "Bonjour le monde", got, "text block after a leading thinking block must not be dropped")
	})
}
