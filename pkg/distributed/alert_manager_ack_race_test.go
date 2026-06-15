package distributed

import (
	"sync"
	"testing"
)

// TestAlertManager_AcknowledgeFieldRace probes whether AcknowledgeAlert's writes
// to the shared *DriftAlert fields (Acknowledged / AcknowledgedAt / AcknowledgedBy,
// done under am.mu) race against a reader that obtained the SAME *DriftAlert via
// GetAlertHistory and reads those fields without am.mu.
//
// GetAlertHistory copies the SLICE of pointers but the pointed-to DriftAlert is
// shared with the AlertManager, so reading alert.Acknowledged after the manager
// flips it under the lock is an unsynchronized reader vs locked writer on the
// same memory — a data race iff the read site holds no lock.
//
// Run with `go test -race`. If this is a genuine race the detector fires; if the
// API is in fact safe (e.g. GetAlertHistory deep-copies), it stays green.
func TestAlertManager_AcknowledgeFieldRace(t *testing.T) {
	am := NewAlertManager(100)
	_ = am.SendAlert(&DriftAlert{WorkerID: "w1", Severity: "high", AlertID: "a1"})

	var wg sync.WaitGroup
	wg.Add(2)

	// Writer: acknowledge (mutates alert fields under am.mu).
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			am.AcknowledgeAlert("a1", "ops")
			// Reset so the loop keeps writing (AcknowledgeAlert only writes when !Acknowledged).
			am.mu.Lock()
			for _, al := range am.alertHistory {
				al.Acknowledged = false
				al.AcknowledgedAt = nil
			}
			am.mu.Unlock()
		}
	}()

	// Reader: read the acknowledged fields off the pointer returned by the public API.
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			h := am.GetAlertHistory(0)
			for _, al := range h {
				_ = al.Acknowledged
				_ = al.AcknowledgedBy
				if al.AcknowledgedAt != nil {
					_ = *al.AcknowledgedAt
				}
			}
		}
	}()

	wg.Wait()
}
