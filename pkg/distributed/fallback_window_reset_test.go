package distributed

import (
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
)

// TestFallbackManager_getFailureRate_WindowExpiry_NoFabricatedFailures proves that
// when a component's failure-tracking window expires, getFailureRate MUST reset the
// window to an empty (0 failures / 0 requests) state and report a 0% failure rate —
// NOT fabricate a 100% failure rate by resetting to 1/1.
//
// Root cause (fallback.go getFailureRate window-reset branch): on window expiry the
// counters were set to Failures=1, TotalRequests=1 and 1.0 was returned. getFailureRate
// is a read-side query (called from monitorFailures with NO new failure) — fabricating a
// 100% rate for a component with a perfectly healthy history traps the distributed
// coordinator in degraded mode and fires false high-failure alerts.
func TestFallbackManager_getFailureRate_WindowExpiry_NoFabricatedFailures(t *testing.T) {
	eventBus := events.NewEventBus()
	logger := &mockLogger{}
	config := DefaultFallbackConfig()
	// Tiny window so it expires deterministically; positive RecoveryCheckInterval not
	// needed for this direct getFailureRate test, set to 0 to avoid monitor goroutines.
	config.FailureTrackingWindow = 30 * time.Millisecond
	config.RecoveryCheckInterval = 0
	fm := NewFallbackManager(config, DefaultPerformanceConfig(), eventBus, logger)

	// Healthy history: 10 successful requests, zero failures -> 0% failure rate.
	for i := 0; i < 10; i++ {
		fm.recordSuccess("healthy-component")
	}

	if rate := fm.getFailureRate("healthy-component"); rate != 0.0 {
		t.Fatalf("precondition: expected 0%% failure rate before window expiry, got %f", rate)
	}

	// Let the tracking window expire with NO new activity.
	time.Sleep(50 * time.Millisecond)

	// A read of the failure rate after expiry must report 0% (window reset to empty),
	// not a fabricated 100% from resetting counters to 1/1.
	rate := fm.getFailureRate("healthy-component")
	if rate != 0.0 {
		t.Fatalf("window expiry fabricated a failure rate: got %f, want 0.0 "+
			"(healthy component with no new failures must not appear as failing)", rate)
	}

	// And the reset must NOT have invented a failure: post-reset counters must be empty.
	fm.mu.RLock()
	tracker := fm.failureCounts["healthy-component"]
	fm.mu.RUnlock()
	tracker.mu.Lock()
	failures, total := tracker.Failures, tracker.TotalRequests
	tracker.mu.Unlock()
	if failures != 0 {
		t.Fatalf("window reset fabricated %d failure(s); fresh window must start at 0 failures", failures)
	}
	if total != 0 {
		t.Fatalf("window reset fabricated %d request(s); fresh window must start at 0 requests", total)
	}
}
