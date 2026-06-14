package distributed

import (
	"testing"

	"digital.vasic.translator/pkg/events"
)

// TestGetNextRemoteInstance_IndexStaleAfterShrink proves that getNextRemoteInstance
// must not index out of range when the remoteInstances slice shrinks after
// currentIndex has advanced past the new length.
//
// Real-world trigger: DiscoverRemoteInstances() rebuilds remoteInstances on every
// call (e.g. a worker drops out / re-pairs). currentIndex is round-robin state that
// is never reset, so if discovery #1 produced N instances and advanced currentIndex
// toward N-1, and discovery #2 produces fewer than currentIndex+1 instances, the
// next getNextRemoteInstance() call does remoteInstances[currentIndex] on a shorter
// slice -> index out of range panic (a crash in the translation hot path).
//
// RED on the pre-fix code: panic: runtime error: index out of range.
func TestGetNextRemoteInstance_IndexStaleAfterShrink(t *testing.T) {
	eventBus := events.NewEventBus()
	dc := NewDistributedCoordinator(nil, nil, nil, nil, nil, eventBus, nil)

	// Simulate discovery #1: 6 instances, round-robin advanced near the end.
	dc.remoteInstances = make([]*RemoteLLMInstance, 6)
	for i := range dc.remoteInstances {
		dc.remoteInstances[i] = &RemoteLLMInstance{ID: "old", Available: true}
	}
	dc.currentIndex = 5 // valid for length 6

	// Simulate discovery #2 shrinking the pool to 2 instances WITHOUT touching
	// currentIndex (exactly what DiscoverRemoteInstances does — it rebuilds the
	// slice but leaves currentIndex alone).
	dc.remoteInstances = []*RemoteLLMInstance{
		{ID: "new-0", Available: true},
		{ID: "new-1", Available: true},
	}
	// currentIndex is still 5 -> out of range for the 2-element slice.

	// This MUST NOT panic and MUST return a valid in-range instance.
	got := dc.getNextRemoteInstance()
	if got == nil {
		t.Fatal("expected a non-nil instance from a non-empty pool")
	}
	if got.ID != "new-0" && got.ID != "new-1" {
		t.Errorf("returned a stale/invalid instance %q; round-robin index was not clamped to the shrunken slice", got.ID)
	}
}
