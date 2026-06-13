package websocket

import (
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestGuard_PublishDoesNotBlockOnFullClientChannel is the cross-package guard
// for the events.Publish -> hub.handleEvent seam. Publish now invokes handlers
// synchronously, so a hub handler that blocked would stall the publisher. This
// asserts that a client whose Send channel is FULL does not block Publish:
// handleEvent's non-blocking select{default} must drop, not wedge.
func TestGuard_PublishDoesNotBlockOnFullClientChannel(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	// Buffer of 1, never drained -> it fills immediately and stays full.
	client := &Client{ID: "c", SessionID: "s", Send: make(chan []byte, 1), Hub: hub}
	hub.Register(client)
	time.Sleep(20 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		// Many publishes to a full, undrained client. If handleEvent blocked,
		// the FIRST or SECOND publish would never return.
		for i := 0; i < 100; i++ {
			ev := events.NewEvent(events.EventTranslationProgress, "p", nil)
			ev.SessionID = "s"
			eventBus.Publish(ev)
		}
		close(done)
	}()

	select {
	case <-done:
		// Publisher completed despite the full client channel — non-blocking.
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a full client Send channel " +
			"(synchronous handler dispatch must not stall the publisher)")
	}
}

// reproSendOnClosed hammers Broadcast/handleEvent concurrently with
// register/unregister churn. If a sender can send on client.Send after Run()
// closed it (send-on-closed-channel race / panic), -race or a panic catches it.
func TestRepro_SendOnClosedChannel(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Constant broadcast pressure (sender side).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				hub.Broadcast([]byte("x"))
				event := events.NewEvent(events.EventTranslationProgress, "p", nil)
				eventBus.Publish(event)
			}
		}
	}()

	// Rapid register/unregister churn (closes Send via Run()).
	for i := 0; i < 200; i++ {
		c := &Client{ID: "c", SessionID: "s", Send: make(chan []byte, 4), Hub: hub}
		hub.Register(c)
		// Do NOT drain; let Send fill, then unregister (closes it).
		hub.Unregister(c)
	}

	close(stop)
	wg.Wait()
	time.Sleep(20 * time.Millisecond)
}

// reproDoubleUnregister fires two concurrent Unregister for the same client.
// If Run()'s guard is wrong, this double-closes Send (panic) or races.
func TestRepro_ConcurrentDoubleUnregister(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	for i := 0; i < 100; i++ {
		c := &Client{ID: "c", SessionID: "s", Send: make(chan []byte, 4), Hub: hub}
		hub.Register(c)
		time.Sleep(time.Millisecond)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); hub.Unregister(c) }()
		go func() { defer wg.Done(); hub.Unregister(c) }()
		wg.Wait()
	}
}
