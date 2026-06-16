package bridge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/translator/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureClientUserMessage stands up an httptest server recording the single user
// message an OpenAI-compatible client sends, returning a real OpenAIClient wired
// to it. This is the exact raw client a clientTranslator delegates to in the
// ensemble path (bridge.realClientBuild builds only NewOpenAIClient).
func captureClientUserMessage(t *testing.T) (llmClient, *string) {
	t.Helper()
	captured := new(string)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &req)
		if len(req.Messages) > 0 {
			*captured = req.Messages[0].Content
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"DUMMY_TRANSLATION"}}]}`))
	}))
	t.Cleanup(srv.Close)
	client, err := llm.NewOpenAIClient(llm.TranslationConfig{
		Provider: "openai", APIKey: "k", Model: "gpt-4", BaseURL: srv.URL,
	})
	require.NoError(t, err)
	return client, captured
}

// TestClientTranslator_ComposesContentNotLabel is the §11.4.115 reproduce-first /
// §11.4.135 standing guard for the wrong-content data-loss class at audit sites
// #8 multi_llm.go:445 and #9 multi_llm.go:529 (and the polisher ensemble path #5),
// which all reach a verbatim-delegating clientTranslator via
// Translate(ctx, REAL_CONTENT, LABEL). The ebook pipeline always passes a
// non-empty label ("Section content", "Chapter title", …) as the 2nd arg.
//
// Pre-fix (RED_MODE=1): clientTranslator forwarded (text, contextStr) verbatim, so
// the raw client sent the LABEL as the user message and DROPPED the real content.
// Post-fix (RED_MODE=0, default standing guard): clientTranslator composes the
// content into the body and appends the label as a Context: hint, so the real
// content reaches the model.
//
// Evidence path: docs/qa/translate_arg_audit_completion_20260616-154307/COMPLETION.md
func TestClientTranslator_ComposesContentNotLabel(t *testing.T) {
	const realContent = "REAL_SECTION_CONTENT_THAT_MUST_BE_TRANSLATED"
	const label = "Section content"
	red := os.Getenv("RED_MODE") == "1"

	client, captured := captureClientUserMessage(t)
	ct := &clientTranslator{client: client}

	out, err := ct.Translate(context.Background(), realContent, label)
	require.NoError(t, err)
	assert.Equal(t, "DUMMY_TRANSLATION", out)

	if red {
		// Pre-fix artifact reproduction: the LABEL was sent, the content dropped.
		assert.Equal(t, label, *captured,
			"PRE-FIX: the label was sent as the user message")
		assert.NotContains(t, *captured, realContent,
			"PRE-FIX: the real content was dropped (data loss)")
		return
	}

	// Standing GREEN guard (post-fix): the real content reached the model.
	assert.Contains(t, *captured, realContent,
		"the real content MUST reach the model, not just the label")
	assert.Contains(t, *captured, label,
		"the label should be preserved as a labelled Context: hint")
}

// TestClientTranslator_EmptyContextSendsContentVerbatim guards the
// preparation-ensemble shape (audit #2/#3/#4): Translate(ctx, prompt, "") — a
// complete analysis prompt in the 1st arg, empty 2nd arg. The composed message
// must be exactly that prompt (no double-wrapping, no dropped prompt).
func TestClientTranslator_EmptyContextSendsContentVerbatim(t *testing.T) {
	const analysisPrompt = "Analyze the following chapter and return JSON: {chars:[...]}"

	client, captured := captureClientUserMessage(t)
	ct := &clientTranslator{client: client}

	_, err := ct.Translate(context.Background(), analysisPrompt, "")
	require.NoError(t, err)
	assert.Equal(t, analysisPrompt, *captured,
		"with an empty context the analysis prompt must be sent verbatim")
}

// TestClientTranslator_RefusesEmptyContent proves §11.4.69: label-only / empty
// content fails loudly rather than sending the label as if it were the payload.
func TestClientTranslator_RefusesEmptyContent(t *testing.T) {
	client, captured := captureClientUserMessage(t)
	ct := &clientTranslator{client: client}

	out, err := ct.Translate(context.Background(), "   ", "Section content")
	require.Error(t, err, "empty/whitespace content must fail loudly")
	assert.Empty(t, out)
	assert.Equal(t, "", *captured, "no request should reach the provider")
	assert.True(t, strings.Contains(err.Error(), "empty"))
}
