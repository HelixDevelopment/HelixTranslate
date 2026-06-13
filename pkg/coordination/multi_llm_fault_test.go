package coordination

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// faultMockTranslator is a fault-injecting mock LLMClient used for unit-tier
// fault injection only (permitted by §11.4.27 — unit tests). It can return a
// fixed error, a fixed translation, a per-call script of (translation, error)
// pairs, and optionally sleep to simulate a slow provider. It counts calls
// atomically so it is safe to share across the goroutine fan-out in
// TranslateWithConsensus.
type faultMockTranslator struct {
	// script is consumed call-by-call; the last entry repeats once exhausted.
	script    []faultStep
	calls     int64
	delay     time.Duration
	mu        sync.Mutex // guards script index advancement
	scriptIdx int
}

type faultStep struct {
	translation string
	err         error
}

func (m *faultMockTranslator) Translate(_ context.Context, _, _ string) (string, error) {
	atomic.AddInt64(&m.calls, 1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.script) == 0 {
		return "default", nil
	}
	step := m.script[m.scriptIdx]
	if m.scriptIdx < len(m.script)-1 {
		m.scriptIdx++
	}
	return step.translation, step.err
}

func (m *faultMockTranslator) TranslateWithProgress(ctx context.Context, text, c string, _ *events.EventBus, _ string) (string, error) {
	return m.Translate(ctx, text, c)
}

func (m *faultMockTranslator) GetStats() translator.TranslationStats { return translator.TranslationStats{} }
func (m *faultMockTranslator) GetName() string                       { return "fault-mock" }
func (m *faultMockTranslator) callCount() int64                      { return atomic.LoadInt64(&m.calls) }

// newCoordinator builds a coordinator directly with the given instances,
// bypassing env discovery so tests are deterministic (§11.4.50).
func newCoordinator(maxRetries int, retryDelay time.Duration, insts ...*LLMInstance) *MultiLLMCoordinator {
	return &MultiLLMCoordinator{
		instances:  insts,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// TestTranslateWithRetry_SuccessFirstInstance asserts a real translated value
// is returned (not an empty string / session id) — anti-bluff §11.4.
func TestTranslateWithRetry_SuccessFirstInstance(t *testing.T) {
	m := &faultMockTranslator{script: []faultStep{{translation: "Hello", err: nil}}}
	c := newCoordinator(3, 0, &LLMInstance{ID: "i1", Translator: m, Available: true})

	got, err := c.TranslateWithRetry(context.Background(), "Привет", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Hello" {
		t.Fatalf("expected real translation %q, got %q", "Hello", got)
	}
	if m.callCount() != 1 {
		t.Fatalf("expected exactly 1 translate call, got %d", m.callCount())
	}
}

// TestTranslateWithRetry_RotatesAfterError covers the failure->next-instance
// rotation branch: instance1 errors, instance2 succeeds.
func TestTranslateWithRetry_RotatesAfterError(t *testing.T) {
	bad := &faultMockTranslator{script: []faultStep{{translation: "", err: errors.New("boom")}}}
	good := &faultMockTranslator{script: []faultStep{{translation: "Hello", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "bad", Translator: bad, Available: true},
		&LLMInstance{ID: "good", Translator: good, Available: true},
	)

	got, err := c.TranslateWithRetry(context.Background(), "Привет", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Hello" {
		t.Fatalf("expected fallback to good instance %q, got %q", "Hello", got)
	}
	if bad.callCount() == 0 || good.callCount() == 0 {
		t.Fatalf("expected both instances tried: bad=%d good=%d", bad.callCount(), good.callCount())
	}
}

// TestTranslateWithRetry_EmptyResultTreatedAsFailure covers the
// `err == nil && translated != ""` guard: a nil-error-but-empty result must NOT
// be accepted as success — it must fall through to the next instance.
func TestTranslateWithRetry_EmptyResultTreatedAsFailure(t *testing.T) {
	emptyOK := &faultMockTranslator{script: []faultStep{{translation: "", err: nil}}}
	good := &faultMockTranslator{script: []faultStep{{translation: "Real", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "empty", Translator: emptyOK, Available: true},
		&LLMInstance{ID: "good", Translator: good, Available: true},
	)

	got, err := c.TranslateWithRetry(context.Background(), "x", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Real" {
		t.Fatalf("empty nil-error result must be rejected; expected %q got %q", "Real", got)
	}
}

// TestTranslateWithRetry_ExhaustionReturnsWrappedError covers the post-loop
// failure path (all instances permanently error) — returns a wrapped lastErr.
func TestTranslateWithRetry_ExhaustionReturnsWrappedError(t *testing.T) {
	sentinel := errors.New("provider-down")
	bad := &faultMockTranslator{script: []faultStep{{translation: "", err: sentinel}}}
	c := newCoordinator(2, 0, &LLMInstance{ID: "bad", Translator: bad, Available: true})

	_, err := c.TranslateWithRetry(context.Background(), "x", "")
	if err == nil {
		t.Fatal("expected error after exhaustion")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "translation failed after") {
		t.Fatalf("expected exhaustion message, got: %v", err)
	}
}

// TestTranslateWithRetry_RateLimitDisablesInstance covers the rate-limit branch:
// a "rate limit" error must mark the instance Available=false and spawn the
// re-enable goroutine. We use a long-but-bounded path and assert the instance
// was disabled. Run under -race to catch the Available write race.
func TestTranslateWithRetry_RateLimitDisablesInstance(t *testing.T) {
	rl := &faultMockTranslator{script: []faultStep{{translation: "", err: errors.New("HTTP 429 rate limit exceeded")}}}
	inst := &LLMInstance{ID: "rl", Translator: rl, Available: true}
	c := newCoordinator(1, 0, inst)

	_, err := c.TranslateWithRetry(context.Background(), "x", "")
	if err == nil {
		t.Fatal("expected error from rate-limited instance")
	}
	// After a rate-limit error the instance must have been disabled.
	c.mu.RLock()
	available := inst.Available
	c.mu.RUnlock()
	if available {
		t.Fatal("rate-limited instance must be marked Available=false")
	}
}

// TestTranslateWithRetry_429Variant covers the "429" substring branch
// distinctly from the textual "rate limit" branch.
func TestTranslateWithRetry_429Variant(t *testing.T) {
	rl := &faultMockTranslator{script: []faultStep{{translation: "", err: errors.New("server returned 429")}}}
	inst := &LLMInstance{ID: "rl429", Translator: rl, Available: true}
	c := newCoordinator(1, 0, inst)

	_, _ = c.TranslateWithRetry(context.Background(), "x", "")
	if inst.Available {
		t.Fatal("429 error must disable the instance")
	}
}

// TestTranslateWithRetry_RetryDelayApplied covers the `time.Sleep(retryDelay)`
// branch by using a small non-zero delay and a multi-attempt scenario.
func TestTranslateWithRetry_RetryDelayApplied(t *testing.T) {
	bad := &faultMockTranslator{script: []faultStep{{translation: "", err: errors.New("boom")}}}
	c := newCoordinator(2, 5*time.Millisecond, &LLMInstance{ID: "b", Translator: bad, Available: true})

	start := time.Now()
	_, err := c.TranslateWithRetry(context.Background(), "x", "")
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error")
	}
	// At least one retry delay should have elapsed.
	if elapsed < 5*time.Millisecond {
		t.Fatalf("expected retry delay to be applied, elapsed=%v", elapsed)
	}
}

// TestTranslateWithRetry_AllInstancesUnavailable covers getNextInstance
// returning nil (all Available=false) -> "All LLM instances exhausted" break.
func TestTranslateWithRetry_AllInstancesUnavailable(t *testing.T) {
	m := &faultMockTranslator{script: []faultStep{{translation: "x", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "i1", Translator: m, Available: false},
		&LLMInstance{ID: "i2", Translator: m, Available: false},
	)
	_, err := c.TranslateWithRetry(context.Background(), "x", "")
	if err == nil {
		t.Fatal("expected error when every instance is unavailable")
	}
	if m.callCount() != 0 {
		t.Fatalf("no translate call should occur when all unavailable, got %d", m.callCount())
	}
}

// TestGetNextInstance_AllUnavailableReturnsNil covers the getNextInstance
// branch where it wraps all the way around without finding an available one.
func TestGetNextInstance_AllUnavailableReturnsNil(t *testing.T) {
	c := &MultiLLMCoordinator{
		instances: []*LLMInstance{
			{ID: "a", Available: false},
			{ID: "b", Available: false},
		},
	}
	if got := c.getNextInstance(); got != nil {
		t.Fatalf("expected nil when no instance available, got %v", got.ID)
	}
}

// TestGetNextInstance_EmptyReturnsNil covers the len(instances)==0 early return.
func TestGetNextInstance_EmptyReturnsNil(t *testing.T) {
	c := &MultiLLMCoordinator{instances: nil}
	if got := c.getNextInstance(); got != nil {
		t.Fatalf("expected nil for empty instance list, got %v", got)
	}
}

// TestConsensus_MajorityWins asserts the voting logic actually selects the
// majority translation, not merely the first success.
func TestConsensus_MajorityWins(t *testing.T) {
	a := &faultMockTranslator{script: []faultStep{{translation: "minority", err: nil}}}
	b := &faultMockTranslator{script: []faultStep{{translation: "MAJORITY", err: nil}}}
	d := &faultMockTranslator{script: []faultStep{{translation: "MAJORITY", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "a", Translator: a, Available: true},
		&LLMInstance{ID: "b", Translator: b, Available: true},
		&LLMInstance{ID: "d", Translator: d, Available: true},
	)

	got, err := c.TranslateWithConsensus(context.Background(), "x", "", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "MAJORITY" {
		t.Fatalf("voting must pick the majority result; expected %q got %q", "MAJORITY", got)
	}
}

// TestConsensus_SkipsUnavailableInstance covers the `if !instance.Available`
// continue branch in TranslateWithConsensus.
func TestConsensus_SkipsUnavailableInstance(t *testing.T) {
	down := &faultMockTranslator{script: []faultStep{{translation: "should-not-run", err: nil}}}
	up := &faultMockTranslator{script: []faultStep{{translation: "live", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "down", Translator: down, Available: false},
		&LLMInstance{ID: "up", Translator: up, Available: true},
	)

	got, err := c.TranslateWithConsensus(context.Background(), "x", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "live" {
		t.Fatalf("expected available instance result %q, got %q", "live", got)
	}
	if down.callCount() != 0 {
		t.Fatalf("unavailable instance must not be invoked, got %d calls", down.callCount())
	}
}

// TestConsensus_AllErrorsFallsBackToRetry covers the consensus fallback path:
// when no consensus translation succeeds, it falls back to TranslateWithRetry.
func TestConsensus_AllErrorsFallsBackToRetry(t *testing.T) {
	// Both consensus participants error; the retry fallback then succeeds
	// because the same instances are retried via round-robin and the script's
	// last (repeating) step is still an error -> overall failure expected,
	// proving the fallback path executed (returns the retry error, not a panic).
	bad := &faultMockTranslator{script: []faultStep{{translation: "", err: errors.New("down")}}}
	c := newCoordinator(1, 0,
		&LLMInstance{ID: "x", Translator: bad, Available: true},
		&LLMInstance{ID: "y", Translator: bad, Available: true},
	)
	_, err := c.TranslateWithConsensus(context.Background(), "t", "", 2)
	if err == nil {
		t.Fatal("expected error after consensus + retry fallback both fail")
	}
	if !strings.Contains(err.Error(), "translation failed after") {
		t.Fatalf("expected fallback retry error, got: %v", err)
	}
}

// TestReenableInstanceAfterDelay_SetsAvailable verifies the cooldown goroutine
// actually flips Available back to true (real behavior, not just no-panic).
func TestReenableInstanceAfterDelay_SetsAvailable(t *testing.T) {
	inst := &LLMInstance{ID: "cooldown", Available: false}
	c := &MultiLLMCoordinator{}
	c.reenableInstanceAfterDelay(inst, time.Millisecond)
	inst.mu.Lock()
	available := inst.Available
	inst.mu.Unlock()
	if !available {
		t.Fatal("instance must be re-enabled after the cooldown delay")
	}
}

// --- §11.4.85 concurrency / stress + chaos tests (run under -race) ---

// TestStress_GetNextInstanceConcurrent hammers the round-robin selector from
// many goroutines to expose data races on currentIndex / instances under -race.
func TestStress_GetNextInstanceConcurrent(t *testing.T) {
	insts := make([]*LLMInstance, 8)
	for i := range insts {
		insts[i] = &LLMInstance{ID: fmt.Sprintf("i%d", i), Available: true}
	}
	c := &MultiLLMCoordinator{instances: insts}

	const goroutines = 32
	const itersEach = 500
	var wg sync.WaitGroup
	var nilCount int64
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < itersEach; i++ {
				if c.getNextInstance() == nil {
					atomic.AddInt64(&nilCount, 1)
				}
			}
		}()
	}
	wg.Wait()
	if nilCount != 0 {
		t.Fatalf("getNextInstance returned nil %d times with all-available instances", nilCount)
	}
}

// TestStress_ConcurrentTranslateNoDeadlock runs many concurrent
// TranslateWithRetry calls; asserts they all complete (no deadlock/hang) and
// every result is a real translation. Bounded by the outer test timeout.
func TestStress_ConcurrentTranslateNoDeadlock(t *testing.T) {
	insts := make([]*LLMInstance, 4)
	for i := range insts {
		insts[i] = &LLMInstance{
			ID:         fmt.Sprintf("i%d", i),
			Translator: &faultMockTranslator{script: []faultStep{{translation: "ok", err: nil}}},
			Available:  true,
		}
	}
	c := newCoordinator(3, 0, insts...)

	const callers = 50
	var wg sync.WaitGroup
	errCh := make(chan error, callers)
	for k := 0; k < callers; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := c.TranslateWithRetry(context.Background(), "x", "")
			if err != nil {
				errCh <- err
				return
			}
			if got != "ok" {
				errCh <- fmt.Errorf("bad result %q", got)
			}
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent translations deadlocked / timed out")
	}
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent translate failed: %v", err)
	}
}

// TestChaos_OneProviderSlowOneFastConsensus injects a slow provider alongside
// a fast one in the consensus fan-out: the coordinator must still produce a
// correct result without deadlock or data race (graceful degradation).
func TestChaos_OneProviderSlowOneFastConsensus(t *testing.T) {
	slow := &faultMockTranslator{
		script: []faultStep{{translation: "consensus", err: nil}},
		delay:  150 * time.Millisecond,
	}
	fast := &faultMockTranslator{script: []faultStep{{translation: "consensus", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "fast", Translator: fast, Available: true},
		&LLMInstance{ID: "slow", Translator: slow, Available: true},
	)

	resCh := make(chan string, 1)
	errCh := make(chan error, 1)
	go func() {
		got, err := c.TranslateWithConsensus(context.Background(), "x", "", 2)
		if err != nil {
			errCh <- err
			return
		}
		resCh <- got
	}()

	select {
	case got := <-resCh:
		if got != "consensus" {
			t.Fatalf("expected %q, got %q", "consensus", got)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("consensus with a slow provider deadlocked / timed out")
	}
}

// TestChaos_OneProviderErrorsConsensusDegrades injects one erroring provider
// into the consensus fan-out; the surviving provider's result must win.
func TestChaos_OneProviderErrorsConsensusDegrades(t *testing.T) {
	broken := &faultMockTranslator{script: []faultStep{{translation: "", err: errors.New("provider exploded")}}}
	healthy := &faultMockTranslator{script: []faultStep{{translation: "survivor", err: nil}}}
	c := newCoordinator(3, 0,
		&LLMInstance{ID: "broken", Translator: broken, Available: true},
		&LLMInstance{ID: "healthy", Translator: healthy, Available: true},
	)

	got, err := c.TranslateWithConsensus(context.Background(), "x", "", 2)
	if err != nil {
		t.Fatalf("consensus must degrade gracefully when one provider errors: %v", err)
	}
	if got != "survivor" {
		t.Fatalf("expected surviving provider result %q, got %q", "survivor", got)
	}
}

// TestWrapper_NoInstancesReturnsErr covers the wrapper fallback when no
// instances initialize (env cleared) -> translator.ErrNoLLMInstances.
func TestWrapper_NoInstancesReturnsErr(t *testing.T) {
	clearLLMEnv(t)
	_, err := NewMultiLLMTranslatorWrapper(translator.TranslationConfig{}, nil, "s")
	if err == nil {
		t.Fatal("expected ErrNoLLMInstances when no providers configured")
	}
	if !errors.Is(err, translator.ErrNoLLMInstances) {
		t.Fatalf("expected ErrNoLLMInstances, got: %v", err)
	}
}

func clearLLMEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "ZHIPU_API_KEY",
		"DEEPSEEK_API_KEY", "QWEN_API_KEY", "OLLAMA_ENABLED",
	} {
		t.Setenv(k, "")
	}
	t.Setenv("SKIP_QWEN_OAUTH", "1")
}
