package distributed

import (
	"context"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestVersionManager_AlertsSliceRace proves a data race on vm.alerts.
// CheckVersionDrift writes the slice header at version_manager.go:1445
// (`vm.alerts = alerts`) with NO lock, while GetAlerts (:1206 `return vm.alerts`)
// and GetHealthStatus (`len(vm.alerts)`) read it with NO lock. CheckVersionDrift
// is invoked periodically / from a background drift loop while API handlers call
// GetAlerts / GetHealthStatus — concurrent unguarded reader+writer on the slice
// header is a data race flagged by `go test -race`.
//
// Hermetic: CheckVersionDrift with an EMPTY services slice performs the
// production `vm.alerts = alerts` write but skips the per-worker network calls,
// so this exercises the real write site with no SSH / HTTP.
//
// RED on pre-fix code: WARNING: DATA RACE on vm.alerts.
func TestVersionManager_AlertsSliceRace(t *testing.T) {
	vm := NewVersionManager(events.NewEventBus())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer: drift checks reassign vm.alerts (production write site).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			_ = vm.CheckVersionDrift(ctx, nil) // empty -> no network, still writes vm.alerts
		}
	}()

	// Readers: API surface reads vm.alerts concurrently.
	for r := 0; r < 3; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = vm.GetAlerts()
				_ = vm.GetHealthStatus()
			}
		}()
	}

	time.Sleep(250 * time.Millisecond)
	close(stop)
	wg.Wait()
}
