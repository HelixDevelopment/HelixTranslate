package distributed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"digital.vasic.translator/pkg/events"
)

// TestVersionManager_CheckWorkerVersion_ConcurrentCacheNoRace is a REPRODUCE-FIRST
// (§11.4.115) guard for the unsynchronized versionCache map.
//
// Root cause (FACT): VersionManager.versionCache is a plain map with NO mutex.
// CheckWorkerVersion reads it (cache hit branch) and writes it (cache fill) with
// no lock. BatchUpdateWorkers fans CheckWorkerVersion out across N goroutines, so
// concurrent distinct-key writes hit the same map header → data race under -race
// and "fatal error: concurrent map writes" without it.
//
// RED on the pre-fix code: `go test -race` reports a DATA RACE on
// distributed.versionCache (and the run may panic with concurrent map writes).
// GREEN after wrapping cache access in a mutex.
func TestVersionManager_CheckWorkerVersion_ConcurrentCacheNoRace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(VersionInfo{
			CodebaseVersion: "v1.0.0",
			Components:      map[string]string{"translator": "1", "api": "1", "distributed": "1"},
		})
	}))
	defer srv.Close()

	vm := NewVersionManager(events.NewEventBus())
	vm.SetBaseURL(srv.URL)

	ctx := context.Background()

	// Each goroutine targets a DISTINCT workerID so they write different cache
	// keys concurrently — the exact shape that corrupts an unguarded map.
	const workers = 32
	const iters = 30

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc := &RemoteService{
				WorkerID: "w-" + string(rune('A'+id%26)) + string(rune('0'+id/26)),
				Host:     "127.0.0.1",
				Port:     1,
				Protocol: "http",
			}
			for j := 0; j < iters; j++ {
				if _, err := vm.CheckWorkerVersion(ctx, svc); err != nil {
					t.Errorf("CheckWorkerVersion: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

// TestVersionManager_CacheAndClearConcurrent stresses the read/write/clear/stat
// paths together (CheckWorkerVersion + ClearCache + GetCacheStats), all of which
// touch versionCache. Under -race any unsynchronized access is flagged.
func TestVersionManager_CacheAndClearConcurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(VersionInfo{CodebaseVersion: "v1.0.0"})
	}))
	defer srv.Close()

	vm := NewVersionManager(events.NewEventBus())
	vm.SetBaseURL(srv.URL)
	ctx := context.Background()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Readers/writers via CheckWorkerVersion.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc := &RemoteService{WorkerID: "x" + string(rune('A'+id)), Host: "127.0.0.1", Port: 1, Protocol: "http"}
			for {
				select {
				case <-stop:
					return
				default:
					_, _ = vm.CheckWorkerVersion(ctx, svc)
				}
			}
		}(i)
	}

	// Concurrent ClearCache + GetCacheStats.
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					vm.ClearCache()
					_ = vm.GetCacheStats()
				}
			}
		}()
	}

	// Let them race briefly.
	for i := 0; i < 2000; i++ {
		_ = vm.GetCacheStats()
	}
	close(stop)
	wg.Wait()
}
