package websocket

import (
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestCombinedStress_AllOpsConcurrent drives every hub concurrency surface at
// once under -race: register/unregister churn, Broadcast + Publish fan-out,
// real WritePump-style drainers consuming Send, and GetClientCount readers.
// Any send-on-closed / double-close / map race surfaces as a -race WARNING or
// a panic. A clean run is the regression guard for the combined interleaving
// the per-op guard tests exercise only in isolation.
func TestCombinedStress_AllOpsConcurrent(t *testing.T) {
	eventBus := events.NewEventBus()
	hub := NewHub(eventBus)
	go hub.Run()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Two senders: Broadcast + Publish.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				hub.Broadcast([]byte("b"))
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				ev := events.NewEvent(events.EventTranslationProgress, "p", nil)
				ev.SessionID = "s"
				eventBus.Publish(ev)
			}
		}
	}()

	// Reader of client count.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				_ = hub.GetClientCount()
			}
		}
	}()

	// Register/unregister churn with a real drainer goroutine per client that
	// receives from Send until it is closed (mirrors WritePump's receive loop,
	// which races Run()'s close).
	for i := 0; i < 400; i++ {
		c := &Client{ID: "c", SessionID: "s", Send: make(chan []byte, 8), Hub: hub}
		hub.Register(c)
		drained := make(chan struct{})
		go func(cl *Client) {
			defer close(drained)
			for range cl.Send { // exits when Run() closes Send
			}
		}(c)
		// Let it live briefly, take some events, then unregister (closes Send).
		time.Sleep(time.Millisecond)
		hub.Unregister(c)
		<-drained
	}

	close(stop)
	wg.Wait()
	time.Sleep(20 * time.Millisecond)
}
