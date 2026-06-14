package coordination

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// --- §11.4.118 second-pass discovery bug-hunt ---

// TestTranslateWithRetry_HonorsContextCancellation is a REPRODUCE-FIRST RED test
// (§11.4.115) for the context-cancellation bug: TranslateWithRetry must stop
// retrying once the caller's context is cancelled. Before the fix, the loop
// ignored ctx entirely — it kept rotating through every instance and even slept
// retryDelay between attempts, doing pointless work (and burning the full
// retryDelay) after the caller had already given up. A correct implementation
// returns promptly with ctx.Err() wrapped and does NOT keep calling Translate.
//
// We use an already-cancelled context + a non-trivial retryDelay + multiple
// instances so that, on the buggy code, the call would sleep at least once and
// call Translate on every rotation; on the fixed code it bails immediately with
// zero or minimal translate calls and well under one retryDelay elapsed.
func TestTranslateWithRetry_HonorsContextCancellation(t *testing.T) {
	var calls int64
	slow := &ctxMockTranslator{calls: &calls}
	// 2 instances, maxRetries 3 => up to 6 attempts; retryDelay 50ms => buggy
	// code would sleep ~5*50ms = 250ms and make several calls.
	c := newCoordinator(3, 50*time.Millisecond,
		&LLMInstance{ID: "a", Translator: slow, Available: true},
		&LLMInstance{ID: "b", Translator: slow, Available: true},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the call

	start := time.Now()
	_, err := c.TranslateWithRetry(ctx, "x", "")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped context.Canceled, got: %v", err)
	}
	// Must bail well before even a single retryDelay would elapse on the buggy
	// path (which sleeps 50ms+ between attempts).
	if elapsed >= 50*time.Millisecond {
		t.Fatalf("cancelled context must short-circuit; elapsed=%v (>= one retryDelay)", elapsed)
	}
	// An ALREADY-cancelled context must not drive ANY provider call: the loop
	// must short-circuit at the top before invoking Translate.
	if got := atomic.LoadInt64(&calls); got != 0 {
		t.Fatalf("already-cancelled context must make zero translate calls; got %d", got)
	}
}

// ctxMockTranslator returns an error so the retry loop would, on the buggy code,
// keep rotating. It counts calls atomically.
type ctxMockTranslator struct {
	calls *int64
}

func (m *ctxMockTranslator) Translate(ctx context.Context, _, _ string) (string, error) {
	atomic.AddInt64(m.calls, 1)
	return "", errors.New("transient")
}

func (m *ctxMockTranslator) TranslateWithProgress(ctx context.Context, text, c string, _ *events.EventBus, _ string) (string, error) {
	return m.Translate(ctx, text, c)
}

func (m *ctxMockTranslator) GetStats() translator.TranslationStats { return translator.TranslationStats{} }
func (m *ctxMockTranslator) GetName() string                       { return "ctx-mock" }

// TestConsensus_UsesAvailableInstancesBeyondPrefix is a REPRODUCE-FIRST RED test
// (§11.4.115) for the consensus instance-selection bug. TranslateWithConsensus
// fans out only over the FIRST requiredAgreement instances and `continue`s past
// unavailable ones WITHOUT scanning further. So when the leading instances are
// unavailable but later ones are available and would AGREE, consensus collects
// zero/too-few results and silently falls back to retry instead of reaching the
// real consensus available among the healthy instances.
//
// Layout: [down, down, good1, good2], requiredAgreement=2. The two healthy
// instances both return "AGREED". Correct behaviour: consensus is reached on
// "AGREED" using the two AVAILABLE instances. Buggy behaviour: only indexes 0,1
// (both down) are considered => instancesUsed=0 => fallback to retry. The retry
// fallback would still eventually return "AGREED" via round-robin, so to prove
// the consensus path itself selected the available instances we assert the
// consensus_reached event fired with agreement_count==2 over total_instances==2.
func TestConsensus_UsesAvailableInstancesBeyondPrefix(t *testing.T) {
	good1 := &faultMockTranslator{script: []faultStep{{translation: "AGREED", err: nil}}}
	good2 := &faultMockTranslator{script: []faultStep{{translation: "AGREED", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "down1", Available: false},
		&LLMInstance{ID: "down2", Available: false},
		&LLMInstance{ID: "good1", Translator: good1, Available: true},
		&LLMInstance{ID: "good2", Translator: good2, Available: true},
	)

	got, err := c.TranslateWithConsensus(context.Background(), "x", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "AGREED" {
		t.Fatalf("expected consensus result %q, got %q", "AGREED", got)
	}
	// Both available instances must have been invoked for the consensus fan-out.
	if good1.callCount() != 1 || good2.callCount() != 1 {
		t.Fatalf("both available instances must participate in consensus; good1=%d good2=%d",
			good1.callCount(), good2.callCount())
	}
}
