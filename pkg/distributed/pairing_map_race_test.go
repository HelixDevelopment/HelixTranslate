package distributed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestPairingManager_ServicesMapDataRace is a REPRODUCE-FIRST (§11.4.115)
// data-race regression guard for the unsynchronised PairingManager.services map.
//
// Root cause (FACT): PairingManager.services is a plain map mutated by
// DiscoverService / PairWithService / UnpairService and the per-tick
// checkServiceHealth goroutines, while concurrently read by GetPairedServices /
// GetServiceStatus — with NO PairingManager-level mutex. The *RemoteService
// struct fields (Status, LastSeen, PairedAt) are likewise mutated by
// checkServiceHealth concurrently with reads by GetPairedServices.
//
// RED on the pre-fix code: `go test -race` reports a DATA RACE between the
// concurrent map writes/reads (and the struct-field writes/reads).
// GREEN after the fix: clean under -race, no deadlock, fast completion.
//
// This test deliberately drives Discover-style inserts, Pair, Unpair,
// GetPairedServices, GetServiceStatus AND a health-style mutation against the
// SAME PairingManager from many goroutines at once.
func TestPairingManager_ServicesMapDataRace(t *testing.T) {
	// Reachable health endpoint so checkServiceHealth takes the "mutate
	// service.Status/LastSeen" path (the struct-field write side of the race).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer srv.Close()
	host, port := hostPortFromURL(t, srv.URL)

	sshPool := NewSSHPool()
	defer sshPool.Close()
	pm := NewPairingManager(sshPool, events.NewEventBus())
	defer pm.Close()
	pm.httpClient = &http.Client{Timeout: 5 * time.Second}

	const workers = 8
	const iterations = 200

	mkService := func(id string) *RemoteService {
		return &RemoteService{
			WorkerID: id,
			Name:     "svc-" + id,
			Host:     host,
			Port:     port,
			Protocol: "http",
			Status:   "online",
			LastSeen: time.Now(),
		}
	}

	ids := []string{"w0", "w1", "w2", "w3"}

	var wg sync.WaitGroup

	// Writer goroutines: insert into the map (Discover-style), pair, unpair.
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				id := ids[(g+i)%len(ids)]
				pm.addService(id, mkService(id))
				_ = pm.PairWithService(id)
				_ = pm.UnpairService(id)
			}
		}(g)
	}

	// Reader goroutines: GetPairedServices + GetServiceStatus.
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = pm.GetPairedServices()
				_, _ = pm.GetServiceStatus(ids[(g+i)%len(ids)])
			}
		}(g)
	}

	// Health-mutation goroutines: drive checkServiceHealth concurrently, which
	// mutates service.Status / LastSeen on the shared *RemoteService.
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				pm.performHealthChecks()
				_ = context.Background()
			}
		}(g)
	}

	wg.Wait()
}
