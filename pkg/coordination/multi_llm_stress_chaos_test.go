package coordination

// §11.4.85 STRESS + CHAOS suite for MultiLLMCoordinator consensus + retry under
// concurrent load and mid-run failure injection. Permanent regression guard
// (§11.4.135). ADDITIVE to multi_llm_fault_test.go: that file proves retry
// no-deadlock and single-shot consensus degradation; THIS file asserts:
//
//   STRESS:
//     - N>=10 concurrent TranslateWithConsensus callers all get the CORRECT
//       consensus result (no deadlock, no wrong/empty result, no race)
//     - the consensus fan-out leaks NO goroutines (before/after count) — the
//       per-call `go func` fan-out (multi_llm.go:424) must all retire
//
//   CHAOS:
//     - instances toggling Available <-> Unavailable mid-run during concurrent
//       consensus: the coordinator still produces a correct result or a clean
//       error, never panics, never deadlocks
//     - context cancellation mid-flight across many concurrent callers returns
//       promptly with a ctx-derived error (no hang)
//
// Anti-bluff (§11.4 / §11.4.1): assertions check the actual translated value +
// goroutine delta + prompt completion. Uses faultMockTranslator + newCoordinator
// from multi_llm_fault_test.go (same package). All mocks — no real network.

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// settleGoroutines waits (bounded) for the consensus fan-out goroutines to retire
// before snapshotting the post-run count, so a transient straggler is not
// mis-read as a leak.
func settleGoroutines(target int, within time.Duration) int {
	deadline := time.Now().Add(within)
	for {
		runtime.GC()
		n := runtime.NumGoroutine()
		if n <= target || time.Now().After(deadline) {
			return n
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestStress_Consensus_ConcurrentCallersCorrectResult runs N>=10 concurrent
// TranslateWithConsensus callers against a pool where every instance agrees on
// the same answer. Every caller MUST get that answer, with no deadlock/race.
func TestStress_Consensus_ConcurrentCallersCorrectResult(t *testing.T) {
	const instances = 5
	insts := make([]*LLMInstance, instances)
	for i := range insts {
		insts[i] = &LLMInstance{
			ID:         fmt.Sprintf("c%d", i),
			Translator: &faultMockTranslator{script: []faultStep{{translation: "agreed", err: nil}}},
			Available:  true,
		}
	}
	c := newCoordinator(3, 0, insts...)

	const callers = 24 // >= 10 concurrent
	var wg sync.WaitGroup
	errCh := make(chan error, callers)
	for k := 0; k < callers; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := c.TranslateWithConsensus(context.Background(), "x", "", 3)
			if err != nil {
				errCh <- fmt.Errorf("consensus errored: %w", err)
				return
			}
			if got != "agreed" {
				errCh <- fmt.Errorf("wrong consensus result %q (want %q)", got, "agreed")
			}
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("concurrent consensus deadlocked / timed out")
	}
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent consensus failed: %v", err)
	}
}

// TestStress_Consensus_MajorityWinsUnderLoad asserts the consensus VOTE is
// correct under concurrent load: 3 instances vote "majority", 1 votes "minority".
// Every concurrent caller must receive the majority answer (the count>maxCount
// branch in multi_llm.go:452 winning deterministically).
func TestStress_Consensus_MajorityWinsUnderLoad(t *testing.T) {
	mkMaj := func() *LLMInstance {
		return &LLMInstance{Translator: &faultMockTranslator{script: []faultStep{{translation: "majority", err: nil}}}, Available: true}
	}
	min := &LLMInstance{ID: "min", Translator: &faultMockTranslator{script: []faultStep{{translation: "minority", err: nil}}}, Available: true}
	m1, m2, m3 := mkMaj(), mkMaj(), mkMaj()
	m1.ID, m2.ID, m3.ID = "maj1", "maj2", "maj3"
	c := newCoordinator(3, 0, m1, m2, m3, min)

	const callers = 16
	var wg sync.WaitGroup
	bad := make(chan string, callers)
	for k := 0; k < callers; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := c.TranslateWithConsensus(context.Background(), "x", "", 4)
			if err != nil {
				bad <- "err:" + err.Error()
				return
			}
			if got != "majority" {
				bad <- got
			}
		}()
	}
	wg.Wait()
	close(bad)
	for b := range bad {
		t.Fatalf("majority consensus broken under load: caller got %q", b)
	}
}

// TestStress_Consensus_NoGoroutineLeak runs many sequential consensus calls and
// asserts the per-call fan-out goroutines (multi_llm.go:424) all retire — the
// goroutine count settles back to baseline. A fan-out that blocked on an
// unbuffered/short channel (or never drained resultsChan) would leak here.
func TestStress_Consensus_NoGoroutineLeak(t *testing.T) {
	const instances = 4
	insts := make([]*LLMInstance, instances)
	for i := range insts {
		insts[i] = &LLMInstance{
			ID:         fmt.Sprintf("g%d", i),
			Translator: &faultMockTranslator{script: []faultStep{{translation: "ok", err: nil}}},
			Available:  true,
		}
	}
	c := newCoordinator(3, 0, insts...)

	// Warm up so baseline excludes lazy one-time goroutines.
	_, _ = c.TranslateWithConsensus(context.Background(), "warm", "", instances)
	baseline := settleGoroutines(runtime.NumGoroutine(), time.Second)

	const calls = 200
	for i := 0; i < calls; i++ {
		got, err := c.TranslateWithConsensus(context.Background(), "x", "", instances)
		if err != nil || got != "ok" {
			t.Fatalf("call %d: got %q err %v", i, got, err)
		}
	}

	settled := settleGoroutines(baseline+5, 5*time.Second)
	if settled > baseline+10 {
		t.Fatalf("consensus goroutine leak: settled %d, baseline %d (fan-out goroutines did not retire)",
			settled, baseline)
	}
}

// TestChaos_Consensus_AvailabilityToggleMidRun injects availability flips on the
// shared instances WHILE many consensus calls run concurrently. The coordinator
// must never panic/deadlock, and every result is either the agreed answer or a
// clean error (when a run happens to find too few available). Anti-bluff: at
// least some calls must SUCCEED (proves it is not trivially erroring out).
func TestChaos_Consensus_AvailabilityToggleMidRun(t *testing.T) {
	const instances = 6
	insts := make([]*LLMInstance, instances)
	for i := range insts {
		insts[i] = &LLMInstance{
			ID:         fmt.Sprintf("t%d", i),
			Translator: &faultMockTranslator{script: []faultStep{{translation: "ok", err: nil}}},
			Available:  true,
		}
	}
	c := newCoordinator(3, 0, insts...)

	stop := make(chan struct{})
	// Chaos: flip availability of a rotating subset (always keep >=2 available so
	// consensus/retry can still succeed sometimes — categorized recovery).
	var chaosWG sync.WaitGroup
	chaosWG.Add(1)
	go func() {
		defer chaosWG.Done()
		i := 0
		for {
			select {
			case <-stop:
				// Restore all-available at the end.
				for _, in := range insts {
					in.SetAvailable(true)
				}
				return
			default:
				// Toggle indices 2..5, leave 0,1 always available.
				idx := 2 + (i % (instances - 2))
				insts[idx].SetAvailable(i%2 == 0)
				i++
				time.Sleep(time.Millisecond)
			}
		}
	}()

	const callers = 20
	var wg sync.WaitGroup
	var successes, cleanErrors int64
	for k := 0; k < callers; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				got, err := c.TranslateWithConsensus(context.Background(), "x", "", 2)
				switch {
				case err == nil && got == "ok":
					atomic.AddInt64(&successes, 1)
				case err == nil && got != "ok":
					t.Errorf("non-error but wrong result %q under chaos", got)
				default:
					// A clean error is acceptable degradation, not a crash.
					atomic.AddInt64(&cleanErrors, 1)
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		close(stop)
		chaosWG.Wait()
		t.Fatal("consensus under availability chaos deadlocked / timed out")
	}
	close(stop)
	chaosWG.Wait()

	// Anti-bluff: instances 0,1 are always available, so successes MUST be > 0.
	if atomic.LoadInt64(&successes) == 0 {
		t.Fatalf("no successful consensus under chaos (%d clean errors) — coordinator not degrading correctly",
			atomic.LoadInt64(&cleanErrors))
	}
	t.Logf("chaos consensus: successes=%d cleanErrors=%d", successes, cleanErrors)
}

// TestChaos_Retry_ContextCancelMidFlightManyCallers cancels each caller's context
// while a slow provider is mid-translate, across many concurrent callers, and
// asserts every caller returns promptly with a ctx-derived error — no hang. This
// exercises the ctx.Err() / timer-vs-ctx.Done() select in TranslateWithRetry
// (multi_llm.go:308, :371) under concurrency.
func TestChaos_Retry_ContextCancelMidFlightManyCallers(t *testing.T) {
	slow := &faultMockTranslator{
		script: []faultStep{{translation: "", err: errors.New("boom")}},
		delay:  200 * time.Millisecond,
	}
	// retryDelay long enough that the cancel path (not a successful retry) is hit.
	c := newCoordinator(5, 500*time.Millisecond,
		&LLMInstance{ID: "slow1", Translator: slow, Available: true},
		&LLMInstance{ID: "slow2", Translator: slow, Available: true},
	)

	const callers = 16
	var wg sync.WaitGroup
	results := make(chan error, callers)
	for k := 0; k < callers; k++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			// Cancel shortly after start, mid-flight.
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()
			_, err := c.TranslateWithRetry(ctx, "x", "")
			cancel()
			results <- err
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("context-cancelled retries hung — did not honor cancellation promptly")
	}
	close(results)

	for err := range results {
		if err == nil {
			t.Fatal("expected a cancellation/failure error, got nil (slow erroring providers)")
		}
		// Must reflect cancellation (aborted) rather than silently succeeding.
		if !errors.Is(err, context.Canceled) && !contains(err.Error(), "aborted") {
			t.Fatalf("expected ctx-derived/aborted error, got: %v", err)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
