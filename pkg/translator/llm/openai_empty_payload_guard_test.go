package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureUserMessage stands up an httptest server that records the single user
// message content the OpenAI-compatible client actually sends, and simulates a
// DeepSeek-style provider: an empty user message is answered with boilerplate
// (mirroring the real "you may have accidentally sent an empty message" reply
// that downstream code silently stored as a translation), a non-empty message is
// echoed back prefixed so the test can assert the real content reached the model.
func captureUserMessage(t *testing.T) (*httptest.Server, *string) {
	t.Helper()
	captured := new(string)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req OpenAIRequest
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &req)
		if len(req.Messages) > 0 {
			*captured = req.Messages[0].Content
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.TrimSpace(*captured) == "" {
			// The exact failure mode: provider boilerplate for an empty message.
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"It seems you may have accidentally sent an empty message."}}]}`))
			return
		}
		out, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "OK::" + *captured}}},
		})
		_, _ = w.Write(out)
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

// TestOpenAITranslate_EmptyPayloadGuard is the §11.4.115 reproduce-first /
// §11.4.135 standing regression guard for the empty-payload data-loss class
// (audit sites #1 markdown-translator:244 and #2/#3/#4 preparation ensemble),
// which all reach a raw OpenAI-compatible client via Translate(ctx, content, "").
//
// Pre-fix (RED_MODE=1): OpenAIClient.Translate sent ONLY the empty 2nd arg as the
// user message → provider boilerplate returned and stored as the translation.
// Post-fix (RED_MODE=0, default standing guard): the empty 2nd arg falls back to
// the content-bearing 1st arg, so the real content reaches the model.
//
// Evidence path: docs/qa/translate_arg_audit_completion_20260616-154307/COMPLETION.md
func TestOpenAITranslate_EmptyPayloadGuard(t *testing.T) {
	const realContent = "The old man walked along the shore at dawn."
	red := os.Getenv("RED_MODE") == "1"

	srv, captured := captureUserMessage(t)
	client, err := NewOpenAIClient(TranslationConfig{
		Provider: "openai", APIKey: "k", Model: "gpt-4", BaseURL: srv.URL,
	})
	require.NoError(t, err)

	// Exactly the markdown-translator:244 / preparation call shape: content in the
	// 1st arg, empty 2nd arg.
	out, err := client.Translate(context.Background(), realContent, "")

	if red {
		// Pre-fix artifact reproduction: the user message was EMPTY and the model
		// returned boilerplate.
		require.NoError(t, err)
		assert.Equal(t, "", strings.TrimSpace(*captured),
			"PRE-FIX: an empty user message was sent (the defect)")
		assert.Contains(t, out, "accidentally sent an empty message",
			"PRE-FIX: provider boilerplate stored as the translation")
		return
	}

	// Standing GREEN guard (post-fix): the real content reached the model.
	require.NoError(t, err)
	assert.Equal(t, realContent, *captured,
		"the content-bearing 1st arg must reach the model when the 2nd arg is empty")
	assert.Contains(t, out, realContent)
	assert.NotContains(t, out, "accidentally sent an empty message")
}

// TestOpenAITranslate_RefuseAllEmpty proves §11.4.69: when BOTH text and prompt
// are empty/whitespace the client returns a loud error instead of letting the
// provider emit boilerplate that downstream code would store as a translation.
func TestOpenAITranslate_RefuseAllEmpty(t *testing.T) {
	srv, captured := captureUserMessage(t)
	client, err := NewOpenAIClient(TranslationConfig{
		Provider: "openai", APIKey: "k", Model: "gpt-4", BaseURL: srv.URL,
	})
	require.NoError(t, err)

	out, err := client.Translate(context.Background(), "   ", "\t\n ")
	require.Error(t, err, "an all-whitespace request must fail loudly, never silently")
	assert.Empty(t, out)
	assert.Contains(t, err.Error(), "empty user message")
	assert.Equal(t, "", *captured, "no request should have been sent to the provider")
}
