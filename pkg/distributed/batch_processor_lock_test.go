package distributed

import (
	"sync"
	"testing"
	"time"
)

// TestBatchProcessor_ProcessFnUnderLockDeadlock proves that BatchProcessor
// invokes the user-supplied processFn while holding bp.mu (performance.go:
// processBatch is called from AddRequest/FlushAll/timer, all holding the lock).
//
// This violates the project hard rule "No blocking operations inside
// synchronized regions": processFn is arbitrary, potentially long-running work
// (the real one performs the batched remote translation network call). The
// concrete, reproducible failure is a self-deadlock — any processFn that needs
// to touch the BatchProcessor again (a natural pattern: flush a sibling batch,
// or enqueue a follow-up request) re-enters the non-reentrant mutex and hangs
// forever.
//
// RED on the pre-fix code: the AddRequest that triggers a full batch never
// returns because processFn (run under bp.mu) calls FlushAll, which blocks on
// the same lock -> deadlock -> watchdog fires.
func TestBatchProcessor_ProcessFnUnderLockDeadlock(t *testing.T) {
	var bp *BatchProcessor

	processed := make(chan struct{}, 1)
	// processFn does a perfectly reasonable thing: after handling this batch it
	// flushes any other pending batches. With processFn running under bp.mu this
	// re-enters the lock and deadlocks.
	processFn := func(requests []interface{}) error {
		_ = bp.FlushAll() // sibling-flush; must not deadlock
		processed <- struct{}{}
		return nil
	}

	bp = NewBatchProcessor(2, time.Hour, processFn) // batchSize 2, long timer

	done := make(chan struct{})
	go func() {
		_ = bp.AddRequest("b", "r1")
		_ = bp.AddRequest("b", "r2") // fills batch -> triggers processBatch -> processFn
		close(done)
	}()

	select {
	case <-done:
		// AddRequest returned; ensure processFn actually ran.
		select {
		case <-processed:
		case <-time.After(time.Second):
			t.Fatal("processFn did not complete")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock: processFn ran under bp.mu and re-entered the lock via FlushAll")
	}
}

// TestBatchProcessor_UnrelatedBatchesSerializeOnProcessFn proves a second facet:
// while processFn runs for batch A, an AddRequest for an UNRELATED batch B is
// blocked for the entire processFn duration, because processFn holds bp.mu.
// Independent batches must not serialize on each other's processing.
//
// RED on pre-fix: the batch-B AddRequest cannot start until processFn for A
// releases the lock, so it observes a long delay.
func TestBatchProcessor_UnrelatedBatchesSerializeOnProcessFn(t *testing.T) {
	release := make(chan struct{})
	inProcess := make(chan struct{}, 1)

	// Only batch A's processFn blocks (simulating a slow network call); batch B
	// returns immediately. If B is delayed it can only be because it is blocked
	// on bp.mu held by A's in-flight processFn — the bug under test.
	processFn := func(requests []interface{}) error {
		if len(requests) > 0 && requests[0] == "ra" {
			select {
			case inProcess <- struct{}{}:
			default:
			}
			<-release // hold until the test lets go
		}
		return nil
	}

	bp := NewBatchProcessor(1, time.Hour, processFn) // batchSize 1 -> process immediately

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = bp.AddRequest("A", "ra") // triggers processFn for A, which blocks on release
	}()

	<-inProcess // processFn for A is now running (under the lock on pre-fix code)

	// An unrelated batch B must be enqueueable while A is processing.
	bDone := make(chan struct{})
	go func() {
		_ = bp.AddRequest("B", "rb")
		close(bDone)
	}()

	select {
	case <-bDone:
		// good: B proceeded independently of A's in-flight processFn
	case <-time.After(2 * time.Second):
		close(release)
		wg.Wait()
		t.Fatal("unrelated batch B blocked behind batch A's processFn held under bp.mu")
	}

	close(release)
	wg.Wait()
}
