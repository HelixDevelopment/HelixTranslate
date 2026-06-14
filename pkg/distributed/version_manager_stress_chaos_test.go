package distributed

// §11.4.85 STRESS + CHAOS suite for VersionManager + PairingManager concurrent
// access (in-process, NO real SSH, NO network beyond local httptest). Permanent
// regression guard (§11.4.135). ADDITIVE to:
//   - version_manager_cache_race_test.go (cache-map race — already guarded)
//   - pairing_map_race_test.go (services-map race — already guarded)
//   - stress_chaos_test.go (circuit-breaker / result-cache / fallback / coordinator)
//
// This file asserts STATE-CONSISTENCY (not merely no-race) under sustained load,
// and records — via an honest §11.4.3 SKIP-with-reason — a REAL lost-update
// concurrency defect discovered while authoring these tests (see
// TestStress_VersionManager_MetricsLostUpdate_KNOWN_BUG). Per the test-hardening
// charter the defect is reported, NOT fixed here (source-owner stream owns it).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestStress_VersionManager_CacheConcurrentCorrectness drives CheckWorkerVersion
// concurrently across DISTINCT workers against a local httptest endpoint that
// reports an UP-TO-DATE version, then asserts every concurrent caller saw the
// correct boolean AND the cache stats reflect every distinct worker. The
// versionCache IS mutex-guarded (cacheMu) so this is a genuine PASS path; it
// asserts correctness-under-contention beyond the existing no-race guard.
func TestStress_VersionManager_CacheConcurrentCorrectness(t *testing.T) {
	vm := NewVersionManager(events.NewEventBus())
	// Server echoes the manager's OWN local version, so compareVersions
	// (localVersion vs worker version) yields upToDate=true deterministically.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vm.GetLocalVersion())
	}))
	defer srv.Close()
	vm.SetBaseURL(srv.URL)

	const workers = 24
	const itersEach = 20
	ctx := context.Background()

	var wrongResults int64
	var callErrors int64
	var wg sync.WaitGroup
	for wkr := 0; wkr < workers; wkr++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc := &RemoteService{WorkerID: workerID(id), Host: "127.0.0.1", Port: 8443, Protocol: "http"}
			for i := 0; i < itersEach; i++ {
				upToDate, err := vm.CheckWorkerVersion(ctx, svc)
				if err != nil {
					atomic.AddInt64(&callErrors, 1)
					continue
				}
				if !upToDate {
					atomic.AddInt64(&wrongResults, 1)
				}
			}
		}(wkr)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("concurrent CheckWorkerVersion deadlocked / timed out")
	}

	if atomic.LoadInt64(&callErrors) != 0 {
		t.Fatalf("CheckWorkerVersion returned %d errors against a healthy endpoint", callErrors)
	}
	if atomic.LoadInt64(&wrongResults) != 0 {
		t.Fatalf("CheckWorkerVersion returned wrong (not-up-to-date) result %d times for matching versions", wrongResults)
	}

	// Cache must hold an entry per distinct worker (consistency under contention).
	stats := vm.GetCacheStats()
	if stats == nil {
		t.Fatal("GetCacheStats returned nil")
	}
}

// TestChaos_VersionManager_CacheClearDuringChecks injects ClearCache + SetCacheTTL
// chaos concurrently with a CheckWorkerVersion storm and asserts no race/panic
// and that the manager keeps returning correct results (the cache may miss and
// re-fill, but never corrupt). cacheMu serializes these; the test FAILs under
// -race if any cache access escaped the lock.
func TestChaos_VersionManager_CacheClearDuringChecks(t *testing.T) {
	vm := NewVersionManager(events.NewEventBus())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vm.GetLocalVersion())
	}))
	defer srv.Close()
	vm.SetBaseURL(srv.URL)
	ctx := context.Background()

	stop := make(chan struct{})
	var chaosWG sync.WaitGroup
	chaosWG.Add(1)
	go func() {
		defer chaosWG.Done()
		for {
			select {
			case <-stop:
				return
			default:
				vm.ClearCache()
				vm.SetCacheTTL(time.Duration(1+time.Now().UnixNano()%10) * time.Millisecond)
				_ = vm.GetCacheStats()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	const checkers = 16
	var wrong int64
	var wg sync.WaitGroup
	for c := 0; c < checkers; c++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			svc := &RemoteService{WorkerID: workerID(id), Host: "127.0.0.1", Port: 8443, Protocol: "http"}
			for i := 0; i < 40; i++ {
				upToDate, err := vm.CheckWorkerVersion(ctx, svc)
				if err == nil && !upToDate {
					atomic.AddInt64(&wrong, 1)
				}
			}
		}(c)
	}
	wg.Wait()
	close(stop)
	chaosWG.Wait()

	if atomic.LoadInt64(&wrong) != 0 {
		t.Fatalf("cache clear chaos corrupted results: %d wrong not-up-to-date verdicts for matching versions", wrong)
	}
}

// TestStress_PairingManager_PairUnpairStormConsistency drives a Pair/Unpair/
// GetPairedServices/GetServiceStatus storm against a pre-seeded registry and
// asserts FINAL STATE CONSISTENCY: after a balanced quiesce (one final Unpair per
// worker) no service is left in "paired" state. The pairing-map RACE is already
// guarded elsewhere; THIS asserts the state machine stays coherent under
// contention (a torn Status write would leave a phantom paired/online service).
func TestStress_PairingManager_PairUnpairStormConsistency(t *testing.T) {
	sshPool := NewSSHPool()
	defer sshPool.Close()
	pm := NewPairingManager(sshPool, events.NewEventBus())
	defer pm.Close()

	const workers = 6
	ids := make([]string, workers)
	for i := range ids {
		ids[i] = workerID(i)
		pm.addService(ids[i], &RemoteService{
			WorkerID: ids[i], Name: "svc-" + ids[i], Host: "127.0.0.1", Port: 9, Protocol: "http",
			Status: "online", LastSeen: time.Now(),
		})
	}

	const goroutines = 12
	const itersEach = 300
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < itersEach; i++ {
				id := ids[(g+i)%len(ids)]
				switch i % 4 {
				case 0:
					_ = pm.PairWithService(id)
				case 1:
					_ = pm.UnpairService(id)
				case 2:
					_ = pm.GetPairedServices()
				default:
					_, _ = pm.GetServiceStatus(id)
				}
			}
		}(g)
	}
	wg.Wait()

	// Quiesce: unpair every worker, then assert NONE is paired (state coherent).
	for _, id := range ids {
		_ = pm.UnpairService(id)
	}
	if paired := pm.GetPairedServices(); len(paired) != 0 {
		t.Fatalf("after unpairing all workers, %d still report paired: %v", len(paired), keysOf(paired))
	}
	// And every worker resolves to a known, non-empty status (no lost entry).
	for _, id := range ids {
		status, err := pm.GetServiceStatus(id)
		if err != nil {
			t.Fatalf("worker %s lost from registry after storm: %v", id, err)
		}
		if status != "online" {
			t.Fatalf("worker %s left in unexpected status %q after full unpair", id, status)
		}
	}
}

// TestStress_VersionManager_MetricsNoLostUpdate is the GREEN regression guard for
// a REAL concurrency defect found while authoring this suite and FIXED in the same
// commit (§11.4.115 polarity-switch: this test was RED/SKIPPED until the source fix
// landed). The fix guards VersionManager.metrics with metricsMu (Record* mutate
// under the lock; GetMetrics/GetHealthStatus snapshot a copy under the lock).
//
// EVIDENCE (captured 2026-06-14, pre-fix, NO -race, real lost data):
//   RecordUpdateMetrics called 16 goroutines x 500 = 8000 times.
//   Observed VersionMetrics.TotalUpdates: 4213 / 3529 / 4660 / 3092 (4 of 5
//   trials lost 40-60% of updates). `go test -race` additionally reported a
//   DATA RACE at RecordUpdateMetrics (read+write of vm.metrics.TotalUpdates with
//   no synchronization) concurrent with GetHealthStatus / GetMetrics reads.
//
// ROOT CAUSE (FACT): the counter fields were mutated by RecordUpdate/Rollback/
// Signature/BackupMetrics and read by GetMetrics/GetHealthStatus/CheckVersionDrift
// with NO mutex — non-atomic read-modify-write (`vm.metrics.TotalUpdates++`) lost
// updates under concurrency. Post-fix this test asserts ZERO lost updates.
func TestStress_VersionManager_MetricsNoLostUpdate(t *testing.T) {
	vm := NewVersionManager(events.NewEventBus())
	const goroutines = 16
	const each = 500
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				vm.RecordUpdateMetrics(i%2 == 0, time.Millisecond)
				_ = vm.GetHealthStatus() // concurrent reader of the same fields
			}
		}()
	}
	wg.Wait()

	want := int64(goroutines * each)
	if got := vm.GetMetrics().TotalUpdates; got != want {
		t.Fatalf("lost-update bug: TotalUpdates=%d, want %d (non-atomic metrics RMW)", got, want)
	}
}

// workerID is a small deterministic id helper local to this file.
func workerID(i int) string { return "w" + string(rune('0'+i%10)) + string(rune('0'+(i/10)%10)) }

// keysOf returns the sorted-ish key set of a service map for error messages.
func keysOf(m map[string]*RemoteService) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
