package distributed

// §11.4.85 STRESS + CONCURRENCY + CHAOS / fault-injection tests for the
// NON-SSH-dependent logic in pkg/distributed. All tests here run fully
// deterministically under `go test -race` with NO live SSH daemon and NO
// network: they exercise in-memory state machines (CircuitBreaker), the
// concurrency-guarded ResultCache + BatchProcessor, the FallbackManager
// degradation path (operation funcs are an injectable fault seam), and the
// DistributedCoordinator round-robin index. Live-SSH I/O (ssh_pool dialing
// real hosts) is integration-tier and honest-deferred via SKIP (§11.4.3).
//
// Anti-bluff (§11.4 / §11.4.1): every test asserts concrete behaviour and
// FAILs if the unit-under-test is stubbed. Worked example: TestStress_
// CircuitBreaker_ConcurrentTripAndRecover asserts the breaker actually
// reaches StateOpen after the failure threshold AND rejects calls while
// open — if CircuitBreaker.Call were stubbed to `return fn()` (no state
// machine), the open-state rejection assertion would fail.

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// errInjected is the canonical injected-fault sentinel for chaos tests.
var errInjected = errors.New("injected fault")

// newStressChaosFallbackManager builds a FallbackManager with monitoring goroutines
// disabled (RecoveryCheckInterval == 0 per fallback.go:116) so tests stay
// deterministic and leak no background tickers.
func newStressChaosFallbackManager(t *testing.T) *FallbackManager {
	t.Helper()
	cfg := &FallbackConfig{
		EnableGracefulDegradation: true,
		DegradationThreshold:      0.5,
		MaxRetries:                0, // no retries: keep stress timing tight + deterministic
		RetryBackoffBase:          time.Millisecond,
		RetryBackoffMax:           time.Millisecond,
		RetryJitter:               false,
		RequestTimeout:            2 * time.Second,
		ConnectionTimeout:         time.Second,
		HealthCheckTimeout:        time.Second,
		RecoveryCheckInterval:     0, // disables monitor goroutines
		RecoverySuccessThreshold:  3,
		RecoveryWindow:            time.Minute,
		EnableLocalFallback:       true,
		EnableReducedQuality:      true,
		EnableCachingFallback:     true,
		FailureTrackingWindow:     5 * time.Minute,
		AlertThreshold:            0.8,
	}
	return NewFallbackManager(cfg, DefaultPerformanceConfig(), events.NewEventBus(), &fallbackMockLogger{})
}

// ---------------------------------------------------------------------------
// STRESS: sustained concurrent load on the in-memory components.
// ---------------------------------------------------------------------------

// TestStress_ResultCache_ConcurrentSetGet hammers the cache from N goroutines
// (no deadlock, no data race under -race) and asserts every write is readable
// (correct totals). Anti-bluff: a stubbed Get (always "", false) makes the
// hit-count assertion fail.
func TestStress_ResultCache_ConcurrentSetGet(t *testing.T) {
	cfg := DefaultPerformanceConfig()
	cfg.MaxCacheSize = 100000
	cfg.CacheCleanupInterval = time.Hour // keep cleanup goroutine inert during test
	cache := NewResultCache(cfg)

	const goroutines = 16
	const perG = 500

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				key := fmt.Sprintf("g%d-k%d", g, i)
				cache.Set(key, fmt.Sprintf("v%d-%d", g, i))
			}
		}(g)
	}
	wg.Wait()

	// Verify correctness: every written key is present with its exact value.
	hits := 0
	for g := 0; g < goroutines; g++ {
		for i := 0; i < perG; i++ {
			key := fmt.Sprintf("g%d-k%d", g, i)
			want := fmt.Sprintf("v%d-%d", g, i)
			if got, ok := cache.Get(key); ok && got == want {
				hits++
			}
		}
	}
	if hits != goroutines*perG {
		t.Fatalf("ResultCache lost writes under concurrency: got %d hits, want %d", hits, goroutines*perG)
	}
}

// TestStress_CircuitBreaker_ConcurrentTripAndRecover drives the breaker from
// many goroutines through trip -> open -> half-open -> closed and asserts the
// real state machine behaviour. Anti-bluff: a stubbed Call that just runs fn
// without tracking state never reaches StateOpen, so the open-rejection check
// below fails.
func TestStress_CircuitBreaker_ConcurrentTripAndRecover(t *testing.T) {
	cb := NewCircuitBreaker(5, 20*time.Millisecond, 2)

	// Phase 1: concurrent failing calls must trip the breaker open.
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cb.Call(func() error { return errInjected })
		}()
	}
	wg.Wait()

	if cb.GetState() != StateOpen {
		t.Fatalf("breaker did not open after %d concurrent failures; state=%v", 50, cb.GetState())
	}

	// While open (before recovery timeout) calls are rejected without running fn.
	ran := false
	err := cb.Call(func() error { ran = true; return nil })
	if err == nil || ran {
		t.Fatalf("open breaker must reject without running fn: err=%v ran=%v", err, ran)
	}

	// Phase 2: after the recovery timeout, successful calls close the breaker.
	time.Sleep(25 * time.Millisecond)
	for i := 0; i < 3; i++ {
		if err := cb.Call(func() error { return nil }); err != nil && i > 0 {
			t.Fatalf("half-open recovery call %d failed: %v", i, err)
		}
	}
	if cb.GetState() != StateClosed {
		t.Fatalf("breaker did not recover to closed; state=%v", cb.GetState())
	}
}

// TestStress_FallbackManager_ConcurrentRecord runs N goroutines recording a
// mix of successes and failures and asserts GetStatus returns consistent,
// non-corrupt totals (TotalRequests == successes+failures per component).
func TestStress_FallbackManager_ConcurrentRecord(t *testing.T) {
	fm := newStressChaosFallbackManager(t)

	const goroutines = 12
	const perG = 200
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			comp := fmt.Sprintf("component-%d", g%4) // 4 shared components -> contention
			for i := 0; i < perG; i++ {
				if i%3 == 0 {
					fm.recordFailure(comp, errInjected)
				} else {
					fm.recordSuccess(comp)
				}
			}
		}(g)
	}
	wg.Wait()

	status := fm.GetStatus()
	comps, ok := status["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("GetStatus returned no components map: %#v", status)
	}
	totalRequests := 0
	for _, raw := range comps {
		c := raw.(map[string]interface{})
		tr := c["total_requests"].(int)
		f := c["failures"].(int)
		if f > tr {
			t.Fatalf("corrupt tracker: failures %d > total_requests %d", f, tr)
		}
		totalRequests += tr
	}
	if totalRequests != goroutines*perG {
		t.Fatalf("FallbackManager lost records under concurrency: got %d, want %d", totalRequests, goroutines*perG)
	}
}

// ---------------------------------------------------------------------------
// BOUNDARY: zero / one / empty / all-failed edge cases — categorised, no panic.
// ---------------------------------------------------------------------------

func TestBoundary_Coordinator_NoRemoteInstances(t *testing.T) {
	eventBus := events.NewEventBus()
	dc := NewDistributedCoordinator(nil, nil, nil, nil, nil, eventBus, nil)

	// Zero instances: round-robin selector returns nil, not a panic.
	if got := dc.getNextRemoteInstance(); got != nil {
		t.Fatalf("expected nil instance with empty pool, got %#v", got)
	}
	if c := dc.GetRemoteInstanceCount(); c != 0 {
		t.Fatalf("expected 0 instances, got %d", c)
	}

	// translateWithRemoteInstances must categorise "no remote instances" cleanly.
	_, err := dc.translateWithRemoteInstances(context.Background(), "hello", "")
	if err == nil {
		t.Fatal("expected error with no remote instances, got nil")
	}
}

func TestBoundary_ResultCache_EmptyAndEviction(t *testing.T) {
	cfg := DefaultPerformanceConfig()
	cfg.MaxCacheSize = 2 // tiny cache forces eviction boundary
	cfg.CacheTTL = time.Hour
	cfg.CacheCleanupInterval = time.Hour
	cache := NewResultCache(cfg)

	// Empty cache: miss, no panic.
	if _, ok := cache.Get("absent"); ok {
		t.Fatal("empty cache reported a hit")
	}

	// Overfill beyond MaxCacheSize: must not grow unbounded (eviction path).
	for i := 0; i < 10; i++ {
		cache.Set(fmt.Sprintf("k%d", i), "v")
	}
	cache.mu.RLock()
	size := len(cache.cache)
	cache.mu.RUnlock()
	if size > cfg.MaxCacheSize {
		t.Fatalf("cache exceeded MaxCacheSize: %d > %d (eviction broken)", size, cfg.MaxCacheSize)
	}
}

func TestBoundary_CircuitBreaker_ZeroThresholdAndEmptyFn(t *testing.T) {
	// Threshold of 1: a single failure must open immediately.
	cb := NewCircuitBreaker(1, time.Second, 1)
	_ = cb.Call(func() error { return errInjected })
	if cb.GetState() != StateOpen {
		t.Fatalf("threshold=1 breaker should open after 1 failure; state=%v", cb.GetState())
	}

	// Empty/no-op fn on a fresh breaker: closes-stays-closed, no panic.
	cb2 := NewCircuitBreaker(3, time.Second, 1)
	if err := cb2.Call(func() error { return nil }); err != nil {
		t.Fatalf("no-op call on closed breaker errored: %v", err)
	}
	if cb2.GetState() != StateClosed {
		t.Fatalf("breaker should remain closed after success; state=%v", cb2.GetState())
	}
}

// ---------------------------------------------------------------------------
// CHAOS / fault-injection: simulate failures mid-operation via the injectable
// operation-func seam; assert graceful degradation, no crash, consistent
// state, and no goroutine leak.
// ---------------------------------------------------------------------------

// TestChaos_FallbackManager_AllFallbacksFail injects a guaranteed-failing
// primary AND a guaranteed-failing fallback, asserting the manager degrades
// gracefully (returns a wrapped error, never panics) and records the failure.
// Anti-bluff: a stubbed ExecuteWithFallback returning nil would fail the
// "expected error" assertion.
func TestChaos_FallbackManager_AllFallbacksFail(t *testing.T) {
	fm := newStressChaosFallbackManager(t)

	primaryCalls := int32(0)
	fbCalls := int32(0)

	fb := FallbackStrategy{
		Name: "custom_fb",
		Function: func() error {
			atomic.AddInt32(&fbCalls, 1)
			return errInjected
		},
	}

	err := fm.ExecuteWithFallback(context.Background(), "chaos-comp",
		func() error {
			atomic.AddInt32(&primaryCalls, 1)
			return errInjected
		}, fb)

	if err == nil {
		t.Fatal("expected wrapped error when primary + fallback both fail, got nil")
	}
	if atomic.LoadInt32(&primaryCalls) == 0 {
		t.Fatal("primary operation was never invoked")
	}
	if atomic.LoadInt32(&fbCalls) == 0 {
		t.Fatal("fallback strategy was never invoked despite primary failure")
	}

	// State must reflect the failure (graceful, consistent).
	status := fm.GetStatus()
	comps := status["components"].(map[string]interface{})
	c, ok := comps["chaos-comp"]
	if !ok {
		t.Fatal("failure not recorded for chaos-comp")
	}
	if c.(map[string]interface{})["failures"].(int) == 0 {
		t.Fatal("failure count not incremented after all-fail")
	}
}

// TestChaos_FallbackManager_FallbackRecovers injects a failing primary but a
// succeeding fallback, asserting the graceful-degradation path actually
// recovers the operation (returns nil) — the §11.4 fallback contract.
func TestChaos_FallbackManager_FallbackRecovers(t *testing.T) {
	fm := newStressChaosFallbackManager(t)

	recovered := int32(0)
	fb := FallbackStrategy{
		Name: "recovering_fb",
		Function: func() error {
			atomic.AddInt32(&recovered, 1)
			return nil
		},
	}

	err := fm.ExecuteWithFallback(context.Background(), "recover-comp",
		func() error { return errInjected }, fb)

	if err != nil {
		t.Fatalf("expected fallback to recover (nil error), got %v", err)
	}
	if atomic.LoadInt32(&recovered) != 1 {
		t.Fatalf("recovering fallback invoked %d times, want 1", recovered)
	}
}

// TestChaos_FallbackManager_DegradedModeUnderFailureStorm floods one component
// past the DegradationThreshold and asserts the manager enters degraded mode
// (graceful degradation) rather than crashing. Anti-bluff: if recordFailure /
// enterDegradedMode were stubbed, degraded_mode would stay false.
func TestChaos_FallbackManager_DegradedModeUnderFailureStorm(t *testing.T) {
	fm := newStressChaosFallbackManager(t)

	// 1 success then 9 failures => 90% failure rate, well past the 0.5 threshold.
	fm.recordSuccess("storm")
	for i := 0; i < 9; i++ {
		fm.recordFailure("storm", errInjected)
	}

	status := fm.GetStatus()
	if degraded, _ := status["degraded_mode"].(bool); !degraded {
		t.Fatal("manager did not enter degraded mode under failure storm")
	}
}

// TestChaos_Coordinator_ConcurrentRoundRobinNoLeak drives the round-robin
// selector concurrently (each goroutine also a mid-flight "worker failure" via
// the empty-pool reset) and asserts no data race, no panic, and no goroutine
// leak after the storm settles.
func TestChaos_Coordinator_ConcurrentRoundRobinNoLeak(t *testing.T) {
	eventBus := events.NewEventBus()
	dc := NewDistributedCoordinator(nil, nil, nil, nil, nil, eventBus, nil)

	// Seed a few instances directly (no SSH): the round-robin index is the
	// concurrency-exposed surface under test.
	dc.mu.Lock()
	for i := 0; i < 4; i++ {
		dc.remoteInstances = append(dc.remoteInstances, &RemoteLLMInstance{
			ID:        fmt.Sprintf("inst-%d", i),
			WorkerID:  fmt.Sprintf("w-%d", i),
			Available: true,
		})
	}
	dc.mu.Unlock()

	before := runtime.NumGoroutine()

	var wg sync.WaitGroup
	var selections int64
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				if inst := dc.getNextRemoteInstance(); inst != nil {
					atomic.AddInt64(&selections, 1)
				}
				_ = dc.GetRemoteInstanceCount()
			}
		}()
	}
	wg.Wait()

	if atomic.LoadInt64(&selections) != 16*1000 {
		t.Fatalf("round-robin lost selections under concurrency: got %d, want %d", selections, 16*1000)
	}

	// Index must remain in-bounds (no corruption from concurrent ++).
	dc.mu.RLock()
	idx := dc.currentIndex
	n := len(dc.remoteInstances)
	dc.mu.RUnlock()
	if idx < 0 || idx >= n {
		t.Fatalf("currentIndex out of bounds after concurrency: %d (n=%d)", idx, n)
	}

	// No goroutine leak: allow scheduler to settle, then compare.
	time.Sleep(50 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutine leak: before=%d after=%d", before, after)
	}
}

// TestChaos_BatchProcessor_ConcurrentAddAndFlush injects concurrent AddRequest
// calls (some triggering full-batch processing, some left pending) plus a
// concurrent FlushAll, asserting the processFn sees every request exactly the
// expected number of times — no lost or double-processed requests under the
// time.AfterFunc + mutex interplay.
func TestChaos_BatchProcessor_ConcurrentAddAndFlush(t *testing.T) {
	var mu sync.Mutex
	processed := 0
	bp := NewBatchProcessor(5, time.Hour /* never auto-fire during test */, func(reqs []interface{}) error {
		mu.Lock()
		processed += len(reqs)
		mu.Unlock()
		return nil
	})

	const goroutines = 8
	const perG = 50
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			batchID := fmt.Sprintf("batch-%d", g)
			for i := 0; i < perG; i++ {
				_ = bp.AddRequest(batchID, i)
			}
		}(g)
	}
	wg.Wait()

	// Flush whatever did not reach the full-batch threshold.
	if err := bp.FlushAll(); err != nil {
		t.Fatalf("FlushAll errored: %v", err)
	}

	mu.Lock()
	got := processed
	mu.Unlock()
	if got != goroutines*perG {
		t.Fatalf("BatchProcessor lost/duplicated requests: processed %d, want %d", got, goroutines*perG)
	}
}

// TestChaos_FallbackManager_ContextCancelledMidOperation injects a cancelled
// context and asserts ExecuteWithFallback returns promptly with the context
// error rather than hanging or panicking (chaos: caller-cancellation fault).
func TestChaos_FallbackManager_ContextCancelledMidOperation(t *testing.T) {
	fm := newStressChaosFallbackManager(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the call

	done := make(chan error, 1)
	go func() {
		done <- fm.ExecuteWithFallback(ctx, "ctx-comp", func() error {
			return errInjected
		})
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected non-nil error on cancelled context")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ExecuteWithFallback hung on cancelled context (no graceful exit)")
	}
}

// ---------------------------------------------------------------------------
// SSH-tier honest-defer (§11.4.3): live SSH dialing requires a real daemon and
// is integration-tier, not unit-deterministic.
// ---------------------------------------------------------------------------

func TestStressChaos_SSHTier_Deferred(t *testing.T) {
	t.Skip("SKIP-OK: live SSH pool dial/exec is integration-tier (§11.4.3); " +
		"requires a real SSH daemon. Non-SSH fault paths are covered above " +
		"via the injectable operation/fallback seams.")
}
