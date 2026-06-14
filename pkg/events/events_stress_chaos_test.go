package events

// §11.4.85 STRESS + CHAOS suite for the EventBus pub/sub core. This is the
// permanent regression guard (§11.4.135) for the documented livelock/leak class
// in events.go: per-handler fan-out, writer starvation, handler-slice races,
// and panic isolation. The existing events_concurrency_guard_test.go proves the
// goroutine-explosion + writer-starvation FIXES; this file is ADDITIVE and asserts
// the END-USER-VISIBLE invariants under sustained load + failure injection:
//
//   STRESS (sustained load / concurrent contention):
//     - every event published to a live subscriber IS delivered (no drops) under
//       N>=10 concurrent publishers
//     - event IDs stay unique under a high-frequency burst (the documented
//       same-microsecond collision class)
//     - HandlerCount returns to baseline after a churn of Subscribe/Unsubscribe
//       (no handler leak)
//
//   CHAOS (failure injection):
//     - a panicking handler does NOT crash the publisher and does NOT prevent
//       later handlers from running (panic isolation)
//     - concurrent Unsubscribe DURING Publish is race-free and an unsubscribed
//       handler stops receiving events
//
// Anti-bluff (§11.4 / §11.4.1): every PASS asserts a concrete delivered-count /
// uniqueness / handler-count / survivor-count. If Publish were stubbed to a
// no-op, the delivery assertions FAIL; if invokeHandler dropped its recover(),
// TestChaos_EventBus_PanicHandlerIsolation FAILs (panic propagates / later
// handler not run). Cleanup of sampler/worker goroutines is in defer.

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// drainGoroutines waits (bounded) for transient goroutines to retire so a
// before/after NumGoroutine comparison is not poisoned by stragglers. Returns
// the observed count after settling.
func drainGoroutines(target int, within time.Duration) int {
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

// TestStress_EventBus_ConcurrentPublishDeliveryComplete asserts that under N>=10
// concurrent publishers hammering the bus, EVERY event is delivered to the live
// SubscribeAll handler exactly the expected number of times. Delivery
// completeness is the user-visible contract the dashboard depends on. Synchronous
// fan-out (events.go) makes the delivered count deterministic: publishers*each.
func TestStress_EventBus_ConcurrentPublishDeliveryComplete(t *testing.T) {
	bus := NewEventBus()

	const publishers = 16 // >= 10 concurrent (FD-aware: pure goroutines, no FDs)
	const eachPublishes = 300

	var delivered int64
	bus.SubscribeAll(func(Event) { atomic.AddInt64(&delivered, 1) })
	// A second, type-specific subscriber: must receive ONLY its own type.
	var progressDelivered int64
	bus.Subscribe(EventTranslationProgress, func(Event) { atomic.AddInt64(&progressDelivered, 1) })

	var wg sync.WaitGroup
	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < eachPublishes; i++ {
				bus.Publish(NewEvent(EventTranslationProgress, "p", nil))
			}
		}()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("concurrent Publish deadlocked / timed out")
	}

	wantAll := int64(publishers * eachPublishes)
	if got := atomic.LoadInt64(&delivered); got != wantAll {
		t.Fatalf("SubscribeAll dropped events: delivered %d, want %d", got, wantAll)
	}
	// Type-specific handler must get exactly the progress events (all of them here).
	if got := atomic.LoadInt64(&progressDelivered); got != wantAll {
		t.Fatalf("type handler delivery mismatch: delivered %d, want %d", got, wantAll)
	}
}

// TestStress_EventBus_TypeFilteringUnderLoad asserts that under concurrent
// publishes of MIXED event types, each type-specific subscriber receives ONLY
// its own type — never another type's events. A bug that fanned every event to
// every type-handler would over-count here.
func TestStress_EventBus_TypeFilteringUnderLoad(t *testing.T) {
	bus := NewEventBus()

	const publishers = 12
	const eachPerType = 200

	var startedCnt, completedCnt, errorCnt int64
	bus.Subscribe(EventTranslationStarted, func(e Event) {
		if e.Type != EventTranslationStarted {
			t.Errorf("started-handler got wrong type %q", e.Type)
		}
		atomic.AddInt64(&startedCnt, 1)
	})
	bus.Subscribe(EventTranslationCompleted, func(e Event) {
		if e.Type != EventTranslationCompleted {
			t.Errorf("completed-handler got wrong type %q", e.Type)
		}
		atomic.AddInt64(&completedCnt, 1)
	})
	bus.Subscribe(EventTranslationError, func(e Event) {
		if e.Type != EventTranslationError {
			t.Errorf("error-handler got wrong type %q", e.Type)
		}
		atomic.AddInt64(&errorCnt, 1)
	})

	types := []EventType{EventTranslationStarted, EventTranslationCompleted, EventTranslationError}
	var wg sync.WaitGroup
	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, et := range types {
				for i := 0; i < eachPerType; i++ {
					bus.Publish(NewEvent(et, "m", nil))
				}
			}
		}()
	}
	wg.Wait()

	want := int64(publishers * eachPerType)
	if startedCnt != want || completedCnt != want || errorCnt != want {
		t.Fatalf("type filtering wrong under load: started=%d completed=%d error=%d want each %d",
			startedCnt, completedCnt, errorCnt, want)
	}
}

// TestStress_EventBus_EventIDUniquenessUnderBurst asserts generateEventID stays
// unique under a high-frequency concurrent burst — the documented
// same-microsecond collision class (events.go:189). Many IDs land in the same
// microsecond; the atomic suffix MUST keep them distinct.
func TestStress_EventBus_EventIDUniquenessUnderBurst(t *testing.T) {
	const goroutines = 16
	const perGoroutine = 2000
	total := goroutines * perGoroutine

	idCh := make(chan string, total)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				idCh <- NewEvent(EventTranslationProgress, "p", nil).ID
			}
		}()
	}
	wg.Wait()
	close(idCh)

	seen := make(map[string]struct{}, total)
	for id := range idCh {
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate event ID generated under burst: %q (%d/%d unique so far)",
				id, len(seen), total)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != total {
		t.Fatalf("expected %d unique IDs, got %d", total, len(seen))
	}
}

// TestStress_EventBus_NoHandlerLeakAfterUnsubscribeChurn asserts HandlerCount
// returns to baseline after a sustained Subscribe/Unsubscribe churn interleaved
// with Publish — proving finite-lifetime subscribers (e.g. a gRPC stream) do not
// leak handlers (events.go:92 contract). A leak would leave HandlerCount > 0.
func TestStress_EventBus_NoHandlerLeakAfterUnsubscribeChurn(t *testing.T) {
	bus := NewEventBus()
	if bus.HandlerCount() != 0 {
		t.Fatalf("fresh bus should have 0 handlers, got %d", bus.HandlerCount())
	}

	const churners = 10
	const cyclesEach = 500

	// A constant publisher running alongside the churn.
	stop := make(chan struct{})
	var pubWG sync.WaitGroup
	pubWG.Add(1)
	go func() {
		defer pubWG.Done()
		for {
			select {
			case <-stop:
				return
			default:
				bus.Publish(NewEvent(EventTranslationProgress, "p", nil))
			}
		}
	}()
	defer func() { close(stop); pubWG.Wait() }()

	var wg sync.WaitGroup
	for c := 0; c < churners; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < cyclesEach; i++ {
				id := bus.SubscribeAll(func(Event) {})
				id2 := bus.Subscribe(EventTranslationProgress, func(Event) {})
				bus.Unsubscribe(id)
				bus.Unsubscribe(id2)
			}
		}()
	}
	wg.Wait()

	if got := bus.HandlerCount(); got != 0 {
		t.Fatalf("handler leak: %d handlers remain after balanced Subscribe/Unsubscribe churn", got)
	}
}

// TestChaos_EventBus_PanicHandlerIsolation injects handlers that PANIC and
// asserts (a) the publisher survives (no crash), and (b) handlers registered
// AFTER the panicking one still run. This is the invokeHandler recover() contract
// (events.go:171). If recover() were removed, the panic would propagate out of
// Publish (test crash) and the survivor handler would be skipped.
func TestChaos_EventBus_PanicHandlerIsolation(t *testing.T) {
	bus := NewEventBus()

	var beforeRan, afterRan int64
	// Handler order within a type is subscription order (events.go:163).
	bus.Subscribe(EventTranslationProgress, func(Event) { atomic.AddInt64(&beforeRan, 1) })
	bus.Subscribe(EventTranslationProgress, func(Event) { panic("chaos: handler exploded") })
	bus.Subscribe(EventTranslationProgress, func(Event) { atomic.AddInt64(&afterRan, 1) })
	// An all-events panicking handler too — must not block the publisher.
	bus.SubscribeAll(func(Event) { panic("chaos: all-handler exploded") })

	const publishes = 500
	const publishers = 8
	var wg sync.WaitGroup
	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < publishes; i++ {
				// If the panic escaped, this goroutine would crash the test binary.
				bus.Publish(NewEvent(EventTranslationProgress, "p", nil))
			}
		}()
	}
	wg.Wait()

	want := int64(publishers * publishes)
	if got := atomic.LoadInt64(&beforeRan); got != want {
		t.Fatalf("handler before the panicking one ran %d times, want %d", got, want)
	}
	if got := atomic.LoadInt64(&afterRan); got != want {
		t.Fatalf("panic isolation broken: handler AFTER the panicking one ran %d times, want %d "+
			"(panic must not skip later handlers)", got, want)
	}
}

// TestChaos_EventBus_ConcurrentUnsubscribeDuringPublish drives Unsubscribe
// concurrently with a heavy Publish stream and asserts (a) no race/panic, and
// (b) once a handler is unsubscribed it stops receiving NEW events (after a
// settle). The fix's snapshot-under-RLock semantics make this safe: a handler
// removed mid-flight is either fully included or fully excluded for a given
// Publish (events.go:149), never observed torn.
func TestChaos_EventBus_ConcurrentUnsubscribeDuringPublish(t *testing.T) {
	bus := NewEventBus()

	// A permanent handler so the bus always has work.
	var permanent int64
	bus.SubscribeAll(func(Event) { atomic.AddInt64(&permanent, 1) })

	const transient = 200
	ids := make([]SubscriptionID, transient)
	counters := make([]int64, transient)
	for i := 0; i < transient; i++ {
		idx := i
		ids[i] = bus.Subscribe(EventTranslationProgress, func(Event) {
			atomic.AddInt64(&counters[idx], 1)
		})
	}

	stop := make(chan struct{})
	var pubWG sync.WaitGroup
	for p := 0; p < 6; p++ {
		pubWG.Add(1)
		go func() {
			defer pubWG.Done()
			for {
				select {
				case <-stop:
					return
				default:
					bus.Publish(NewEvent(EventTranslationProgress, "p", nil))
				}
			}
		}()
	}

	// Unsubscribe all transient handlers concurrently with the publish storm.
	var unWG sync.WaitGroup
	for i := 0; i < transient; i++ {
		unWG.Add(1)
		go func(idx int) {
			defer unWG.Done()
			bus.Unsubscribe(ids[idx])
		}(i)
	}
	unWG.Wait()

	// Let the publish storm run a bit longer AFTER all unsubscribes are done,
	// then snapshot each transient counter and confirm it is frozen.
	time.Sleep(50 * time.Millisecond)
	snapshot := make([]int64, transient)
	for i := range counters {
		snapshot[i] = atomic.LoadInt64(&counters[i])
	}
	time.Sleep(100 * time.Millisecond)
	close(stop)
	pubWG.Wait()

	// After unsubscribe + settle, NO transient handler may have received further
	// events. (permanent must still be alive — proves the bus kept publishing.)
	if atomic.LoadInt64(&permanent) == 0 {
		t.Fatal("permanent handler never ran — publish storm did not exercise the bus")
	}
	for i := range counters {
		final := atomic.LoadInt64(&counters[i])
		if final != snapshot[i] {
			t.Fatalf("unsubscribed handler %d kept receiving events: %d -> %d "+
				"(Unsubscribe did not remove it)", i, snapshot[i], final)
		}
	}
	// Only the permanent handler must remain.
	if got := bus.HandlerCount(); got != 1 {
		t.Fatalf("after unsubscribing all %d transient handlers, HandlerCount=%d, want 1", transient, got)
	}
}

// TestChaos_EventBus_PublishWhileNoSubscribers asserts the boundary chaos case:
// publishing to a type with zero handlers (and an empty all-events set) is a
// safe no-op under load — exercises the append(nil, ...) empty-slice path
// (events.go:157) which must yield zero iterations, not a nil-deref/panic.
func TestChaos_EventBus_PublishWhileNoSubscribers(t *testing.T) {
	bus := NewEventBus()
	const publishers = 8
	var wg sync.WaitGroup
	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				bus.Publish(NewEvent(EventConversionStarted, fmt.Sprintf("p%d-%d", p, i), nil))
			}
		}(p)
	}
	wg.Wait()
	// Surviving here (no panic) is the assertion; confirm bus still functional.
	var got int64
	bus.SubscribeAll(func(Event) { atomic.AddInt64(&got, 1) })
	bus.Publish(NewEvent(EventConversionStarted, "after", nil))
	if got != 1 {
		t.Fatalf("bus not functional after no-subscriber storm: delivered %d, want 1", got)
	}
}

// TestStress_EventBus_GoroutineBoundedUnderSustainedPublish is the explicit
// before/after goroutine-count guard the §11.4.85 methodology asks for: under a
// sustained concurrent Publish load the live goroutine count must stay bounded
// near baseline (synchronous fan-out — no per-handler-per-event spawn) and must
// settle back after the load stops (no leak).
func TestStress_EventBus_GoroutineBoundedUnderSustainedPublish(t *testing.T) {
	bus := NewEventBus()
	const handlers = 20
	for i := 0; i < handlers; i++ {
		bus.SubscribeAll(func(Event) {})
	}

	baseline := runtime.NumGoroutine()

	var peak int64
	stopSampler := make(chan struct{})
	var samplerWG sync.WaitGroup
	samplerWG.Add(1)
	go func() {
		defer samplerWG.Done()
		for {
			select {
			case <-stopSampler:
				return
			default:
				n := int64(runtime.NumGoroutine())
				for {
					p := atomic.LoadInt64(&peak)
					if n <= p || atomic.CompareAndSwapInt64(&peak, p, n) {
						break
					}
				}
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()

	const publishers = 8
	const eachPublishes = 1000
	var wg sync.WaitGroup
	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < eachPublishes; i++ {
				bus.Publish(NewEvent(EventTranslationProgress, "p", nil))
			}
		}()
	}
	wg.Wait()
	close(stopSampler)
	samplerWG.Wait()

	observedPeak := atomic.LoadInt64(&peak)
	peakDelta := observedPeak - int64(baseline)
	// publishers + sampler + scheduling margin. A per-handler-per-event spawn
	// would blow far past this with 20 handlers x 8000 publishes.
	ceiling := int64(publishers + 50)
	if peakDelta > ceiling {
		t.Fatalf("goroutine growth under sustained publish: peak delta %d > ceiling %d", peakDelta, ceiling)
	}

	// After the load, goroutines must settle back near baseline (no leak).
	settled := drainGoroutines(baseline+5, 3*time.Second)
	if settled > baseline+10 {
		t.Fatalf("goroutine leak after publish load: settled %d, baseline %d", settled, baseline)
	}
}
