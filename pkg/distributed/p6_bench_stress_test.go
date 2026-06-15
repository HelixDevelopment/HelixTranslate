package distributed

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"digital.vasic.translator/pkg/events"
)

// P6 coverage (§11.4.27 / §11.4.85): genuinely-missing performance/benchmark +
// stress coverage for pure / thread-safe distributed hot paths that need NO real
// SSH or network. These exercise the REAL functions, so a regression breaks the
// asserted correctness or shifts the measured numbers.

func newP6Coordinator() *DistributedCoordinator {
	// The pure priority/count helpers do not dereference any of these fields, so
	// nils are safe here — the construction matches the public constructor.
	return NewDistributedCoordinator(nil, nil, nil, nil, nil, events.NewEventBus(), nil)
}

func newP6Fallback() *FallbackManager {
	return NewFallbackManager(DefaultFallbackConfig(), DefaultPerformanceConfig(),
		events.NewEventBus(), &fallbackMockLogger{})
}

// ---------------------------------------------------------------------------
// Benchmarks — pure coordinator routing-decision hot paths (no SSH/network).
// Run: go test -bench=. -benchmem -run=^$ ./pkg/distributed/
// ---------------------------------------------------------------------------

func BenchmarkGetPriorityForProvider(b *testing.B) {
	dc := newP6Coordinator()
	providers := []string{"openai", "anthropic", "llamacpp", "qwen", "unknown"}
	b.ResetTimer()
	b.ReportAllocs()
	var sink int
	for i := 0; i < b.N; i++ {
		sink = dc.getPriorityForProvider(providers[i%len(providers)])
	}
	_ = sink
}

func BenchmarkGetInstanceCountForPriority(b *testing.B) {
	dc := newP6Coordinator()
	b.ResetTimer()
	b.ReportAllocs()
	var sink int
	for i := 0; i < b.N; i++ {
		sink = dc.getInstanceCountForPriority(i%12, 4)
	}
	_ = sink
}

// ---------------------------------------------------------------------------
// Benchmarks — fallback failure-tracking hot paths (thread-safe, no network).
// ---------------------------------------------------------------------------

func BenchmarkFallbackCalculateBackoff(b *testing.B) {
	fm := newP6Fallback()
	b.ResetTimer()
	b.ReportAllocs()
	var sink int64
	for i := 0; i < b.N; i++ {
		sink += int64(fm.calculateBackoff(i % 8))
	}
	_ = sink
}

func BenchmarkFallbackRecordSuccess(b *testing.B) {
	fm := newP6Fallback()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		fm.recordSuccess("bench-component")
	}
}

func BenchmarkFallbackGetFailureRate(b *testing.B) {
	fm := newP6Fallback()
	// Pre-seed some traffic so the rate computation runs the real division path.
	for i := 0; i < 100; i++ {
		fm.recordSuccess("bench-component")
	}
	fm.recordFailure("bench-component", errors.New("seed failure"))
	b.ResetTimer()
	b.ReportAllocs()
	var sink float64
	for i := 0; i < b.N; i++ {
		fm.mu.RLock()
		sink += fm.getFailureRate("bench-component")
		fm.mu.RUnlock()
	}
	_ = sink
}

// ---------------------------------------------------------------------------
// Stress (§11.4.85) — concurrent contention on the failure-tracking path. N
// goroutines concurrently record successes/failures and read the failure rate;
// asserts the accounting is correct (no lost updates) and the path is data-race
// clean under -race.
// ---------------------------------------------------------------------------

func TestStress_Fallback_ConcurrentRecord(t *testing.T) {
	fm := newP6Fallback()
	// Disable degradation/alert side effects from skewing the test: a high config
	// threshold keeps the component out of degraded mode regardless of rate so we
	// measure raw accounting. (DefaultFallbackConfig keeps monitoring goroutines
	// unstarted in this construction.)
	fm.config.EnableGracefulDegradation = false
	fm.config.AlertThreshold = 2.0 // unreachable; suppress alert emission

	const (
		goroutines        = 16 // ≥10 per §11.4.85
		successPerRoutine = 50
		failurePerRoutine = 25
	)
	component := "stress-component"

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < successPerRoutine; i++ {
				fm.recordSuccess(component)
			}
			for i := 0; i < failurePerRoutine; i++ {
				fm.recordFailure(component, errors.New("stress failure"))
			}
			// Concurrent reader exercising the read path under contention.
			fm.mu.RLock()
			_ = fm.getFailureRate(component)
			fm.mu.RUnlock()
		}()
	}
	wg.Wait()

	wantTotal := goroutines * (successPerRoutine + failurePerRoutine)
	wantFailures := goroutines * failurePerRoutine

	fm.mu.RLock()
	tracker := fm.failureCounts[component]
	fm.mu.RUnlock()
	if tracker == nil {
		t.Fatal("expected a failure tracker for the stressed component")
	}

	tracker.mu.Lock()
	gotTotal := tracker.TotalRequests
	gotFailures := tracker.Failures
	tracker.mu.Unlock()

	if gotTotal != wantTotal {
		t.Fatalf("lost updates under contention: TotalRequests=%d, want %d", gotTotal, wantTotal)
	}
	if gotFailures != wantFailures {
		t.Fatalf("lost updates under contention: Failures=%d, want %d", gotFailures, wantFailures)
	}

	// The default window is large, so the rate must equal failures/total exactly.
	wantRate := float64(wantFailures) / float64(wantTotal)
	fm.mu.RLock()
	gotRate := fm.getFailureRate(component)
	fm.mu.RUnlock()
	if gotRate != wantRate {
		t.Fatalf("failure rate wrong after contention: got %v, want %v", gotRate, wantRate)
	}
}

// TestStress_Coordinator_ConcurrentPriorityDecisions hammers the pure routing
// helpers from many goroutines; asserts deterministic correct results (the
// decision table is read-only, so concurrent callers must always agree).
func TestStress_Coordinator_ConcurrentPriorityDecisions(t *testing.T) {
	dc := newP6Coordinator()
	const goroutines = 16

	type want struct {
		provider string
		priority int
	}
	cases := []want{
		{"openai", 10}, {"anthropic", 10}, {"zhipu", 10}, {"deepseek", 10},
		{"llamacpp", 5},
		{"ollama", 1}, // R-2: ollama removed — now default priority
		{"qwen", 1}, {"", 1}, {"unknown-provider", 1},
	}

	var wg sync.WaitGroup
	errCh := make(chan string, goroutines*len(cases))
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 100; n++ {
				for _, c := range cases {
					if got := dc.getPriorityForProvider(c.provider); got != c.priority {
						errCh <- fmt.Sprintf("getPriorityForProvider(%q)=%d want %d", c.provider, got, c.priority)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	if msg, ok := <-errCh; ok {
		t.Fatalf("concurrent priority decision incorrect: %s", msg)
	}
}
