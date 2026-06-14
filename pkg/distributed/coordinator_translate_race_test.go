package distributed

import (
	"context"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestTranslateWithRemoteInstances_SliceRace proves a data race on
// dc.remoteInstances: translateWithRemoteInstances reads the slice header
// (len + indexing via getNextRemoteInstance, and the unguarded instance.LastUsed
// write) WITHOUT holding dc.mu, while DiscoverRemoteInstances rebuilds the slice
// under dc.mu.Lock(). A re-discovery (worker drops out / re-pairs) concurrent
// with an in-flight translation is the real-world trigger.
//
// translateWithRemoteInstances at coordinator.go:404 and :411 reads
// len(dc.remoteInstances) with no lock; DiscoverRemoteInstances at :82 reassigns
// the slice under the write lock. That is a read/write race on the slice header,
// flagged by `go test -race`.
//
// To keep this hermetic (no real SSH / network), the remote instances point at a
// worker that is NOT in the paired-services map, so translateWithRemoteInstance
// returns the "service not found" error immediately (coordinator.go:514-516)
// without any HTTP dial. The race is on the slice access, not the translation.
func TestTranslateWithRemoteInstances_SliceRace(t *testing.T) {
	eventBus := events.NewEventBus()
	// pairingManager with an empty services map => GetPairedServices returns {}.
	pm := &PairingManager{
		services: make(map[string]*RemoteService),
		eventBus: eventBus,
		ctx:      context.Background(),
	}
	// versionManager nil => validateWorkerForWork is skipped.
	dc := NewDistributedCoordinator(nil, nil, pm, nil, nil, eventBus, nil)
	dc.maxRetries = 1

	// Seed instances pointing at unknown workers so the translation path fails
	// fast with no network I/O but still walks the slice.
	seed := func() {
		dc.mu.Lock()
		dc.remoteInstances = make([]*RemoteLLMInstance, 0, 4)
		for i := 0; i < 4; i++ {
			dc.remoteInstances = append(dc.remoteInstances, &RemoteLLMInstance{
				ID:        "inst",
				WorkerID:  "missing-worker",
				Provider:  "ollama",
				Model:     "x",
				Available: true,
			})
		}
		dc.mu.Unlock()
	}
	seed()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Reader goroutines: translation hot path (unguarded slice reads).
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_, _ = dc.translateWithRemoteInstances(ctx, "hello", "")
			}
		}()
	}

	// Writer goroutine: concurrent re-discovery rebuilds the slice under lock.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			seed() // mimics DiscoverRemoteInstances replacing dc.remoteInstances
		}
	}()

	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
}
