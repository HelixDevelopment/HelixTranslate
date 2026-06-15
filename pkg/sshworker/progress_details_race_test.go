package sshworker

import (
	"sync"
	"testing"
	"time"
)

// TestProgressTracker_GetProgress_DetailsRace is the §11.4.115 reproduce-first
// RED test for a data race in ProgressTracker.GetProgress().
//
// Defect: GetProgress() takes pt.mu.RLock(), but places the LIVE internal
// pt.Details map (not a copy) into the returned snapshot. The caller then reads
// that map AFTER GetProgress has released the lock, while another goroutine
// legitimately mutates pt.Details under pt.mu.Lock() (exactly as
// ExecuteCommandWithProgress / MonitorLongRunningCommand do via
// tracker.Details["..."] = ...). Concurrent map read + write on the same Go map
// is a data race AND can panic with "concurrent map read and map write".
//
// GetCopy() already deep-copies Details for precisely this reason; GetProgress()
// does not — the asymmetry is the bug.
//
// Anti-bluff: this test does NOT assert on values; it drives the real concurrent
// access pattern and relies on `-race` to flag the unsynchronised map access.
// On the buggy code, `go test -race` reports a DATA RACE between the map write
// (under lock) and the lock-free map read of the returned snapshot. The fix
// (deep-copy Details inside GetProgress, like GetCopy) removes the race.
func TestProgressTracker_GetProgress_DetailsRace(t *testing.T) {
	pt := &ProgressTracker{
		Operation: "race-op",
		Total:     100,
		Details:   map[string]interface{}{"seed": 0},
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	// Writer: mutates Details under the tracker's lock, the same way the
	// production wrappers do (tracker.mu.Lock(); tracker.Details[...] = ...).
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
			}
			pt.mu.Lock()
			pt.Details["k"] = i
			i++
			pt.mu.Unlock()
		}
	}()

	// Reader: takes a snapshot via the public GetProgress() and then READS the
	// returned details map — outside any lock, because GetProgress is supposed
	// to hand back a safe snapshot. With the live-map bug this read races the
	// writer above.
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			snap := pt.GetProgress()
			details, _ := snap["details"].(map[string]interface{})
			// Range over the returned map => concurrent map read vs the writer's
			// concurrent map write on the buggy (shared-reference) code path.
			for range details {
			}
		}
	}()

	time.Sleep(150 * time.Millisecond)
	close(stop)
	wg.Wait()
}
