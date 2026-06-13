package events

import (
	"sync"
	"sync/atomic"
	"testing"
)

// TestEventBus_Unsubscribe_RemovesAllEventsHandler proves the leak fix: a
// handler registered via SubscribeAll is invoked while subscribed, and is NOT
// invoked (and is fully removed from the bus) after Unsubscribe. Before this
// fix the EventBus had no Unsubscribe, so a finite-lifetime subscriber such as
// the gRPC SubscribeEvents stream stayed registered forever and was invoked on
// every future Publish — a handler leak.
func TestEventBus_Unsubscribe_RemovesAllEventsHandler(t *testing.T) {
	bus := NewEventBus()

	var calls int32
	id := bus.SubscribeAll(func(Event) { atomic.AddInt32(&calls, 1) })

	if id == 0 {
		t.Fatalf("SubscribeAll returned zero SubscriptionID")
	}
	if got := bus.HandlerCount(); got != 1 {
		t.Fatalf("after SubscribeAll: HandlerCount = %d, want 1", got)
	}

	bus.Publish(NewEvent(EventTranslationProgress, "x", nil))
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("after first publish: calls = %d, want 1", got)
	}

	bus.Unsubscribe(id)
	if got := bus.HandlerCount(); got != 0 {
		t.Fatalf("after Unsubscribe: HandlerCount = %d, want 0 (handler leaked)", got)
	}

	bus.Publish(NewEvent(EventTranslationProgress, "y", nil))
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("after Unsubscribe + publish: calls = %d, want 1 (handler still invoked = leak)", got)
	}
}

// TestEventBus_Unsubscribe_RemovesTypedHandler is the same proof for the
// type-specific Subscribe path.
func TestEventBus_Unsubscribe_RemovesTypedHandler(t *testing.T) {
	bus := NewEventBus()

	var calls int32
	id := bus.Subscribe(EventTranslationCompleted, func(Event) { atomic.AddInt32(&calls, 1) })

	bus.Publish(NewEvent(EventTranslationCompleted, "x", nil))
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("after first publish: calls = %d, want 1", got)
	}

	bus.Unsubscribe(id)
	if got := bus.HandlerCount(); got != 0 {
		t.Fatalf("after Unsubscribe: HandlerCount = %d, want 0", got)
	}

	bus.Publish(NewEvent(EventTranslationCompleted, "y", nil))
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("after Unsubscribe + publish: calls = %d, want 1", got)
	}
}

// TestEventBus_NoLeakAcrossManySubscribers simulates the real defect's blast
// radius: many finite-lifetime subscribers (as gRPC streams open and close).
// With Unsubscribe wired the handler set returns to empty; without it the bus
// would grow unbounded (the leak).
func TestEventBus_NoLeakAcrossManySubscribers(t *testing.T) {
	bus := NewEventBus()

	const n = 1000
	for i := 0; i < n; i++ {
		id := bus.SubscribeAll(func(Event) {})
		bus.Publish(NewEvent(EventTranslationProgress, "x", nil))
		bus.Unsubscribe(id)
	}

	if got := bus.HandlerCount(); got != 0 {
		t.Fatalf("after %d subscribe/unsubscribe cycles: HandlerCount = %d, want 0 (leak)", n, got)
	}
}

// TestEventBus_Unsubscribe_UnknownIDIsNoOp guards the documented contract that
// Unsubscribe with an unknown / already-removed / zero id is harmless.
func TestEventBus_Unsubscribe_UnknownIDIsNoOp(t *testing.T) {
	bus := NewEventBus()
	id := bus.SubscribeAll(func(Event) {})

	bus.Unsubscribe(0)          // zero id
	bus.Unsubscribe(id + 12345) // never-issued id
	if got := bus.HandlerCount(); got != 1 {
		t.Fatalf("unknown-id Unsubscribe removed a live handler: HandlerCount = %d, want 1", got)
	}

	bus.Unsubscribe(id)
	bus.Unsubscribe(id) // double-unsubscribe
	if got := bus.HandlerCount(); got != 0 {
		t.Fatalf("HandlerCount = %d, want 0", got)
	}
}

// TestEventBus_Unsubscribe_Concurrent exercises Subscribe/Unsubscribe/Publish
// under -race to prove the new id bookkeeping is race-free.
func TestEventBus_Unsubscribe_Concurrent(t *testing.T) {
	bus := NewEventBus()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := bus.SubscribeAll(func(Event) {})
			bus.Publish(NewEvent(EventTranslationProgress, "x", nil))
			bus.Unsubscribe(id)
		}()
	}
	// concurrent publishers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				bus.Publish(NewEvent(EventTranslationProgress, "y", nil))
			}
		}()
	}
	wg.Wait()

	if got := bus.HandlerCount(); got != 0 {
		t.Fatalf("after concurrent churn: HandlerCount = %d, want 0", got)
	}
}
