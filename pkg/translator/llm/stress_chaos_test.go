package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/pkg/translator"
)

// newMockTranslator builds an LLMTranslator backed by the in-package MockLLMClient
// and returns both so the test can inject faults and assert on the real pipeline.
// Mock-from-unit-test usage is permitted per §11.4.27 (unit tier).
func newMockTranslator(t *testing.T) (*LLMTranslator, *MockLLMClient) {
	t.Helper()
	lt, err := NewLLMTranslatorWithConfig(translator.TranslationConfig{Provider: "mock"})
	if err != nil {
		t.Fatalf("failed to construct mock translator: %v", err)
	}
	mock, ok := lt.client.(*MockLLMClient)
	if !ok {
		t.Fatalf("expected *MockLLMClient, got %T", lt.client)
	}
	return lt, mock
}

// ---------------------------------------------------------------------------
// STRESS: sustained load on a single translator (cache hit/miss path).
// Asserts every call returns the correct deterministic translation, the cache
// short-circuits repeats, and stats are internally consistent.
// ---------------------------------------------------------------------------

func TestStress_SustainedLoad_SingleText(t *testing.T) {
	lt, mock := newMockTranslator(t)
	const iterations = 500
	const text = "Привет мир"
	// Case-insensitive: enhanceTranslation correctly capitalizes the first letter
	// when the source ("Привет…") starts uppercase, so the mock's lowercase
	// "translated: …" becomes "Translated: …". This sanity guard only proves the
	// translation is not stubbed/echoed, so case is irrelevant here.
	const wantPrefix = "translated: "

	for i := 0; i < iterations; i++ {
		got, err := lt.Translate(context.Background(), text, "greeting")
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if got == "" {
			t.Fatalf("iteration %d: empty translation (stub/regression)", i)
		}
	}

	// Only the FIRST call should hit the underlying client; the rest are cache hits.
	if cc := mock.CallCount(); cc != 1 {
		t.Fatalf("expected exactly 1 client call across %d iterations (cache must short-circuit), got %d",
			iterations, cc)
	}

	stats := lt.GetStats()
	if stats.Cached != iterations-1 {
		t.Fatalf("expected %d cache hits, got %d", iterations-1, stats.Cached)
	}
	if stats.Translated != 1 {
		t.Fatalf("expected 1 real translation, got %d", stats.Translated)
	}
	// Sanity: the produced text must actually contain the expected mock output,
	// so a stubbed Translate (returning "" or echoing input) fails this test.
	if got, _ := lt.Translate(context.Background(), text, "greeting"); !strings.HasPrefix(strings.ToLower(got), wantPrefix) {
		t.Fatalf("translation does not match mock contract: %q", got)
	}
}

// STRESS: sustained load with many DISTINCT inputs — exercises the cache write
// path under load and verifies one client call per distinct (text,context) key.
func TestStress_SustainedLoad_DistinctTexts(t *testing.T) {
	lt, mock := newMockTranslator(t)
	const iterations = 300

	for i := 0; i < iterations; i++ {
		text := fmt.Sprintf("текст-%d", i)
		got, err := lt.Translate(context.Background(), text, "ctx")
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		want := "translated: " + text
		if got != want {
			t.Fatalf("iteration %d: got %q want %q", i, got, want)
		}
	}

	if cc := mock.CallCount(); cc != iterations {
		t.Fatalf("expected %d client calls for %d distinct texts, got %d", iterations, iterations, cc)
	}
	stats := lt.GetStats()
	if stats.Translated != iterations {
		t.Fatalf("expected %d translations, got %d", iterations, stats.Translated)
	}
	if stats.Cached != 0 {
		t.Fatalf("expected 0 cache hits for distinct texts, got %d", stats.Cached)
	}
}

// ---------------------------------------------------------------------------
// CONCURRENCY/CONTENTION: many goroutines hammering the shared cache + stats.
// Run with -race; asserts no data race, no deadlock, correct results, and that
// the recorded stats account for exactly the work performed.
// ---------------------------------------------------------------------------

func TestStress_Concurrent_SharedCache(t *testing.T) {
	lt, _ := newMockTranslator(t)
	const goroutines = 32
	const perGoroutine = 50
	// Bounded distinct key space so goroutines genuinely contend on the same
	// cache entries (write/read races) rather than each owning a private key.
	const distinctKeys = 8

	var wg sync.WaitGroup
	var failures int64
	done := make(chan struct{})

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				key := fmt.Sprintf("shared-%d", (g+i)%distinctKeys)
				got, err := lt.Translate(context.Background(), key, "concurrent")
				if err != nil {
					atomic.AddInt64(&failures, 1)
					return
				}
				want := "translated: " + key
				if got != want {
					atomic.AddInt64(&failures, 1)
					return
				}
			}
		}(g)
	}

	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("deadlock/timeout: concurrent Translate did not complete within 30s")
	}

	if f := atomic.LoadInt64(&failures); f != 0 {
		t.Fatalf("%d concurrent goroutines observed wrong/failed results", f)
	}

	// Correctness invariant under concurrency: total work recorded must equal
	// the number of cache misses + cache hits == total invocations.
	stats := lt.GetStats()
	total := goroutines * perGoroutine
	if stats.Translated+stats.Cached != total {
		t.Fatalf("stats lost updates under concurrency: translated(%d)+cached(%d)=%d, want %d",
			stats.Translated, stats.Cached, stats.Translated+stats.Cached, total)
	}
	// At least one real translation per distinct key must have happened; cache
	// must have absorbed the remainder.
	if stats.Translated < distinctKeys {
		t.Fatalf("expected >= %d real translations (one per key), got %d", distinctKeys, stats.Translated)
	}
	if stats.Errors != 0 {
		t.Fatalf("expected 0 errors, got %d", stats.Errors)
	}
}

// CONCURRENCY: independent translators sharing one fault-free mock-less path,
// each with its own cache, run in parallel. Guards against any package-level
// shared mutable state leaking across instances.
func TestStress_Concurrent_IndependentTranslators(t *testing.T) {
	const translators = 12
	var wg sync.WaitGroup
	errs := make([]error, translators)

	for i := 0; i < translators; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lt, err := NewLLMTranslatorWithConfig(translator.TranslationConfig{Provider: "mock"})
			if err != nil {
				errs[i] = err
				return
			}
			for j := 0; j < 40; j++ {
				if _, e := lt.Translate(context.Background(), fmt.Sprintf("t%d-%d", i, j), "c"); e != nil {
					errs[i] = e
					return
				}
			}
		}(i)
	}
	wg.Wait()
	for i, e := range errs {
		if e != nil {
			t.Fatalf("translator %d failed: %v", i, e)
		}
	}
}

// ---------------------------------------------------------------------------
// BOUNDARY: degenerate / extreme inputs must produce a categorised result and
// never panic.
// ---------------------------------------------------------------------------

func TestBoundary_Inputs(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		context   string
		wantEmpty bool // empty/whitespace short-circuits to the input unchanged
	}{
		{"empty text", "", "ctx", true},
		{"whitespace only", "   \t\n  ", "ctx", true},
		{"empty context", "hello", "", false},
		{"unicode multibyte", "Здравствуйте, мир! 你好 🌍", "ctx", false},
		{"single char", "a", "ctx", false},
		{"newlines", "line1\n\nline2\n", "ctx", false},
		{"large text 100KB", strings.Repeat("Большой текст. ", 7000), "ctx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lt, _ := newMockTranslator(t)
			var got string
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("panic on input %q: %v", tt.name, r)
					}
				}()
				got, err = lt.Translate(context.Background(), tt.text, tt.context)
			}()
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.name, err)
			}
			if tt.wantEmpty {
				// Whitespace/empty input is returned unchanged (documented short-circuit).
				if got != tt.text {
					t.Fatalf("expected empty/whitespace input echoed unchanged, got %q", got)
				}
				return
			}
			if got == "" {
				t.Fatalf("non-empty input produced empty translation (stub/regression) for %q", tt.name)
			}
		})
	}
}

// BOUNDARY: a very large text that exceeds the internal split threshold must be
// chunked, translated, and re-joined without loss or panic.
func TestBoundary_LargeText_TriggersSplit(t *testing.T) {
	lt, mock := newMockTranslator(t)
	// >20KB forces splitText() into multiple chunks. Make the mock fail the
	// first whole-text attempt with a size error so the retry/split path runs.
	mock.SetFailure(true, 1)
	mock.SetSizeError(true)

	large := strings.Repeat("Это предложение. ", 3000) // ~51KB, well over 20KB
	got, err := lt.Translate(context.Background(), large, "big")
	if err != nil {
		t.Fatalf("large-text split path failed: %v", err)
	}
	if got == "" {
		t.Fatalf("split path produced empty result")
	}
	// More than one client call proves the split path actually executed
	// (1 failed whole-text attempt + N chunk calls).
	if cc := mock.CallCount(); cc < 2 {
		t.Fatalf("expected split into multiple client calls, got %d", cc)
	}
}

// ---------------------------------------------------------------------------
// CHAOS / FAULT INJECTION: errors, transient flapping, slow responses, ctx
// cancellation. Pipeline must degrade gracefully (categorised error, no crash,
// cache not corrupted).
// ---------------------------------------------------------------------------

// Hard failure: a persistent upstream error must surface as a wrapped error and
// be recorded in stats.Errors, with NO entry cached (cache-not-corrupted).
func TestChaos_PersistentError_NoCachePoisoning(t *testing.T) {
	lt, mock := newMockTranslator(t)
	mock.SetFailure(true, 1<<30) // effectively always fail

	_, err := lt.Translate(context.Background(), "обреченный", "ctx")
	if err == nil {
		t.Fatal("expected error from failing client, got nil")
	}
	if !strings.Contains(err.Error(), "LLM translation failed") {
		t.Fatalf("error not categorised/wrapped by pipeline: %v", err)
	}
	stats := lt.GetStats()
	if stats.Errors != 1 {
		t.Fatalf("expected 1 recorded error, got %d", stats.Errors)
	}
	// Now stop failing: the same key must translate freshly (proving the failed
	// attempt did NOT poison the cache with a bad/empty entry).
	mock.SetFailure(false, 0)
	got, err2 := lt.Translate(context.Background(), "обреченный", "ctx")
	if err2 != nil {
		t.Fatalf("recovery translation failed: %v", err2)
	}
	if got != "translated: обреченный" {
		t.Fatalf("cache poisoned by prior failure: got %q", got)
	}
}

// Transient flapping under concurrency: every Nth call fails. Pipeline must
// neither deadlock nor crash; successes are correct, failures are accounted.
func TestChaos_TransientFlapping_Concurrent(t *testing.T) {
	lt, mock := newMockTranslator(t)
	mock.SetFailEveryNth(3) // ~1/3 of calls fail transiently

	const goroutines = 16
	const perGoroutine = 30
	var wg sync.WaitGroup
	var ok, bad int64
	done := make(chan struct{})

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				// Distinct keys so failures don't get masked by cache hits.
				_, err := lt.Translate(context.Background(), fmt.Sprintf("flap-%d-%d", g, i), "ctx")
				if err != nil {
					atomic.AddInt64(&bad, 1)
				} else {
					atomic.AddInt64(&ok, 1)
				}
			}
		}(g)
	}
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("deadlock/timeout under transient-failure chaos")
	}

	total := int64(goroutines * perGoroutine)
	if ok+bad != total {
		t.Fatalf("lost invocations: ok(%d)+bad(%d)=%d, want %d", ok, bad, ok+bad, total)
	}
	if bad == 0 {
		t.Fatal("fault injection produced no failures — chaos not exercised")
	}
	if ok == 0 {
		t.Fatal("every call failed — fault injection mis-wired")
	}
	// Stats must reconcile: recorded errors == observed failures, and the
	// stats structure must remain internally consistent (no torn counters).
	stats := lt.GetStats()
	if int64(stats.Errors) != bad {
		t.Fatalf("stats.Errors(%d) != observed failures(%d) — lost/torn updates under concurrency",
			stats.Errors, bad)
	}
	if int64(stats.Translated) != ok {
		t.Fatalf("stats.Translated(%d) != observed successes(%d)", stats.Translated, ok)
	}
}

// Slow response + context cancellation: a deadline must abort the call promptly
// with a context error rather than hanging.
func TestChaos_SlowResponse_ContextCancel(t *testing.T) {
	lt, mock := newMockTranslator(t)
	mock.SetDelay(5*time.Second, true) // honor ctx during the delay

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := lt.Translate(ctx, "медленный", "ctx")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context-deadline error from slow response, got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("call did not abort promptly on ctx cancel: took %v", elapsed)
	}
	stats := lt.GetStats()
	if stats.Errors != 1 {
		t.Fatalf("expected 1 recorded error from cancelled call, got %d", stats.Errors)
	}
}

// Concurrent slow responses with a shared bounded key space: verifies the
// pipeline holds no lock across the blocking client call (otherwise N slow
// calls would serialize and blow the timeout).
func TestChaos_ConcurrentSlowResponses_NoLockHeldAcrossCall(t *testing.T) {
	lt, mock := newMockTranslator(t)
	const delay = 200 * time.Millisecond
	mock.SetDelay(delay, false)

	const goroutines = 20
	var wg sync.WaitGroup
	done := make(chan struct{})
	start := time.Now()

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			// Distinct keys => every call goes through the (slow) client.
			_, _ = lt.Translate(context.Background(), fmt.Sprintf("slow-%d", g), "ctx")
		}(g)
	}
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("concurrent slow calls timed out — pipeline likely serializes under a held lock")
	}
	elapsed := time.Since(start)

	// If the cache lock were held across the blocking client call, 20 calls at
	// 200ms each would take >=4s. Concurrency should keep it far below that.
	if elapsed >= time.Duration(goroutines)*delay {
		t.Fatalf("calls serialized (%v for %d x %v) — blocking call under held lock",
			elapsed, goroutines, delay)
	}
}
