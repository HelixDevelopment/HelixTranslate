package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ctxAwareTranslator blocks inside Translate / TranslateWithProgress until the
// supplied context is cancelled, then returns ctx.Err(). A safety timer bounds
// the wait so the test can never hang the suite even if the fix regresses (the
// safety path records that the context was NOT honoured).
type ctxAwareTranslator struct {
	// honoured is set true iff the blocking call observed ctx cancellation;
	// false means the safety timer fired (the request context was ignored).
	honoured *bool
}

func (c *ctxAwareTranslator) wait(ctx context.Context) (string, error) {
	select {
	case <-ctx.Done():
		*c.honoured = true
		return "", ctx.Err()
	case <-time.After(2 * time.Second):
		// Safety valve: the handler's context never cancelled within a bound
		// far below the 5-minute production deadline, so request-context
		// cancellation was NOT propagated.
		*c.honoured = false
		return "", context.DeadlineExceeded
	}
}

func (c *ctxAwareTranslator) Translate(ctx context.Context, _, _ string) (string, error) {
	return c.wait(ctx)
}

func (c *ctxAwareTranslator) TranslateWithProgress(ctx context.Context, _, _ string, _ *events.EventBus, _ string) (string, error) {
	return c.wait(ctx)
}

func (c *ctxAwareTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}
func (c *ctxAwareTranslator) GetName() string { return "ctx-aware" }

// TestTranslateText_PropagatesRequestContextCancellation is the §11.4.115 /
// §11.4.135 regression guard for the audit's "request-path context misuse"
// finding. It proves the synchronous translateText handler now derives its
// translation context from c.Request.Context() (bounded by a timeout) instead
// of an unbounded context.Background(): when the client's request context is
// already cancelled, the in-flight translation observes the cancellation and
// the handler returns promptly with a 500 — it does NOT ignore the disconnect
// and hang on the provider.
//
// Mutation proof (manual, per §1.1): reverting the handler to
// `ctx := context.Background()` makes the cancelled request context invisible
// to the translator, the 2s safety timer fires, honoured == false, and the
// `require.True(t, honoured, ...)` assertion FAILS — so this test catches the
// negation of the fix.
func TestTranslateText_PropagatesRequestContextCancellation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newContractHandler()

	var honoured bool
	h.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
		return &ctxAwareTranslator{honoured: &honoured}, nil
	}

	router := gin.New()
	router.POST("/translate", h.translateText)

	body, _ := json.Marshal(map[string]any{
		"text":     "Hello world",
		"provider": "openai",
	})

	// Build a request whose context is already cancelled, simulating a client
	// that disconnected before/while the translation ran.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/translate", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	done := make(chan struct{})
	start := time.Now()
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("handler did not return within 5s — request context cancellation was not propagated (context misuse regressed)")
	}

	require.True(t, honoured,
		"translateText must propagate the cancelled request context into the translation; "+
			"with context.Background() the disconnect is ignored and the provider call blocks")
	assert.Equal(t, http.StatusInternalServerError, w.Code,
		"a cancelled request context must surface as a translation error, not a 200 OK")
	assert.Less(t, time.Since(start), 2*time.Second,
		"cancellation must abort the translation promptly, well under the safety timer")
}
