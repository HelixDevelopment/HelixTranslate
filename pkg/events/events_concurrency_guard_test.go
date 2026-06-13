package events

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRepro_PublishGoroutineExplosion is the RED reproduction for the real
// defect: Publish spawns one detached goroutine PER HANDLER PER EVENT. With a
// modest handler count and a burst of publishes, the live-goroutine count
// explodes far beyond a sane ceiling — unbounded fan-out with no backpressure.
//
// RED expectation on the pre-fix code: peak goroutine count climbs into the
// thousands+ for a 50-handler x 2000-publish burst (100k transient goroutines).
// Post-fix (synchronous fan-out) the peak stays bounded near the baseline.
func TestRepro_PublishGoroutineExplosion(t *testing.T) {
	bus := NewEventBus()

	const handlers = 50
	const publishes = 2000

	// Slow-ish handlers so spawned goroutines stay alive long enough to pile up.
	for i := 0; i < handlers; i++ {
		bus.SubscribeAll(func(Event) {
			time.Sleep(200 * time.Microsecond)
		})
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
				time.Sleep(50 * time.Microsecond)
			}
		}
	}()

	for i := 0; i < publishes; i++ {
		bus.Publish(NewEvent(EventTranslationProgress, "p", nil))
	}

	close(stopSampler)
	samplerWG.Wait()

	observedPeak := atomic.LoadInt64(&peak)
	delta := observedPeak - int64(baseline)
	t.Logf("baseline goroutines=%d peak=%d delta=%d (handlers=%d publishes=%d)",
		baseline, observedPeak, delta, handlers, publishes)

	// A sane bus fans out without leaking unbounded goroutines. Allow generous
	// slack (handlers + sampler + a small worker margin), but a per-handler
	// per-event spawn blows WAY past this on the pre-fix code.
	const ceiling = int64(handlers + 50)
	if delta > ceiling {
		t.Fatalf("goroutine explosion: peak delta %d exceeds ceiling %d "+
			"(unbounded per-handler-per-event goroutine spawn)", delta, ceiling)
	}
}

// TestRepro_PublishSubscribeAllRace hammers Publish concurrently with a
// BOUNDED stream of SubscribeAll/Subscribe to surface any -race fault on the
// handler slice/map. The subscriber adds a capped number of handlers (not an
// unbounded flood) so the test exercises the concurrent read/write of the
// handler containers deterministically rather than degenerating into an
// O(N^2) synchronous-invocation pathology.
func TestRepro_PublishSubscribeAllRace(t *testing.T) {
	bus := NewEventBus()
	var wg sync.WaitGroup

	const subscriberAdds = 500
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < subscriberAdds; i++ {
			bus.SubscribeAll(func(Event) {})
			bus.Subscribe(EventTranslationProgress, func(Event) {})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			bus.Publish(NewEvent(EventTranslationProgress, "p", nil))
		}
	}()

	wg.Wait()
}

// TestRepro_PublishNoWriterStarvation proves the fix removes writer starvation:
// a continuous Publish stream MUST NOT prevent a Subscribe from acquiring the
// write lock in a bounded time. Pre-fix, Publish held RLock across the entire
// goroutine-spawning fan-out, starving the writer under sustained load.
func TestRepro_PublishNoWriterStarvation(t *testing.T) {
	bus := NewEventBus()
	bus.SubscribeAll(func(Event) {})

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

	// Under continuous publish pressure, a Subscribe must still complete fast.
	done := make(chan struct{})
	go func() {
		bus.Subscribe(EventTranslationStarted, func(Event) {})
		close(done)
	}()

	select {
	case <-done:
		// Writer acquired the lock despite publish pressure — fix works.
	case <-time.After(3 * time.Second):
		close(stop)
		pubWG.Wait()
		t.Fatal("writer starvation: Subscribe could not acquire the write lock " +
			"within 3s under continuous Publish load")
	}

	close(stop)
	pubWG.Wait()
}
