package distributed

import (
	"context"
	"sync"
	"testing"

	"digital.vasic.translator/pkg/events"
)

// TestVersionManager_BackupsMapConcurrentWrite proves that the vm.backups map is
// written without synchronization. BatchUpdateWorkers spawns one goroutine per
// service and calls vm.UpdateWorker concurrently; UpdateWorker -> createWorkerBackup
// does `vm.backups[service.WorkerID] = backup` (version_manager.go:1006) with NO
// lock. Concurrent writes to a Go map for distinct keys are a FATAL runtime error
// ("concurrent map writes"), not merely a race — it crashes the whole process.
//
// This drives createWorkerBackup directly (the exact unsynchronized map write
// BatchUpdateWorkers reaches via UpdateWorker) so the test stays hermetic: it
// touches only a temp filesystem dir + the in-memory map, no SSH, no network.
//
// RED on pre-fix code: `fatal error: concurrent map writes` OR a -race report on
// vm.backups. GREEN after the map write is guarded by a mutex.
func TestVersionManager_BackupsMapConcurrentWrite(t *testing.T) {
	vm := NewVersionManager(events.NewEventBus())
	vm.backupDir = t.TempDir() // hermetic: no /tmp pollution, filesystem only

	const goroutines = 24
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			svc := &RemoteService{
				WorkerID: workerKey(n),
				Version:  VersionInfo{CodebaseVersion: "1.0.0"},
			}
			<-start
			// Mirrors UpdateWorker's backup step exactly.
			_, _ = vm.createWorkerBackup(context.Background(), svc)
		}(i)
	}

	close(start) // release all goroutines to hammer vm.backups simultaneously
	wg.Wait()

	// Sanity: every distinct worker's backup must have survived (no lost write).
	// Safe to read unlocked here — all writer goroutines have joined via wg.Wait.
	got := len(vm.backups)
	if got != goroutines {
		t.Fatalf("expected %d backups stored, got %d (lost writes under concurrency)", goroutines, got)
	}
}

func workerKey(n int) string {
	return "worker-" + string(rune('A'+n%26)) + string(rune('0'+n/26))
}
