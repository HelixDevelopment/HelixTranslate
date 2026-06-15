package deployment

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/internal/config"
)

// TestGetDiscoveredServices_NoConcurrentMapWriteUnderRLock is a reproduce-first
// RED test for a data race / concurrent-map-write in
// NetworkDiscoverer.GetDiscoveredServices (network_discovery.go:279-296).
//
// GetDiscoveredServices takes nd.mu.RLock() (a *read* lock) but then MUTATES the
// shared map with delete(nd.services, id) inside the loop. RLock permits multiple
// concurrent holders, so two concurrent GetDiscoveredServices calls can both run
// delete() on the same map at the same time — a concurrent map write, which the Go
// runtime detects under -race (and can fatally panic in production). A reader that
// mutates shared state under a read lock is the classic broken-locking defect.
//
// To trigger expiry-driven deletion deterministically we seed services whose TTL
// has already elapsed, then call GetDiscoveredServices from many goroutines.
// Run under -race.
func TestGetDiscoveredServices_NoConcurrentMapWriteUnderRLock(t *testing.T) {
	cfg := &config.Config{}
	logger := log.New(os.Stdout, "", 0)
	nd := NewNetworkDiscoverer(cfg, logger)
	defer nd.Close()

	// Seed many already-expired services so every GetDiscoveredServices call takes
	// the delete() path.
	nd.mu.Lock()
	for i := 0; i < 200; i++ {
		id := "svc-" + time.Duration(i).String()
		nd.services[id] = &NetworkService{
			ID:       id,
			LastSeen: time.Now().Add(-10 * time.Minute), // long expired
			TTL:      time.Second,
		}
	}
	nd.mu.Unlock()

	const goroutines = 16
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				// Re-seed under a proper write lock so there is always something to
				// expire/delete, keeping the racing delete path hot.
				nd.mu.Lock()
				nd.services["hot"] = &NetworkService{
					ID:       "hot",
					LastSeen: time.Now().Add(-10 * time.Minute),
					TTL:      time.Second,
				}
				nd.mu.Unlock()
				_ = nd.GetDiscoveredServices()
			}
		}()
	}
	wg.Wait()
}
