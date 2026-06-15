package progress

import "testing"

// TestTracker_ItemsOnly_FailedItemsCountTowardCompletion is a REPRODUCE-FIRST
// (§11.4.115) test for an items-only run where every item is accounted for
// (some completed, some failed) but the run is genuinely DONE. The progress
// bar must reach 100% and the ETA must resolve to "Completed" — a run where
// all work is finished must not report a partial percentage forever.
func TestTracker_ItemsOnly_FailedItemsCountTowardCompletion(t *testing.T) {
	// Items-only mode: TotalChapters=0 so the chapter branch is skipped.
	tracker := NewTracker("s", "Book", 0, "ru", "sr", "deepseek", "m")
	tracker.SetTotal(10)

	for i := 0; i < 6; i++ {
		tracker.IncrementCompleted()
	}
	for i := 0; i < 4; i++ {
		tracker.IncrementFailed()
	}

	p := tracker.GetProgress()
	if p.PercentComplete != 100.0 {
		t.Fatalf("all 10 items accounted for (6 ok + 4 failed) => run is done; "+
			"want PercentComplete=100, got %.2f", p.PercentComplete)
	}
	if p.EstimatedETA != "Completed" {
		t.Fatalf("finished run must report ETA \"Completed\", got %q", p.EstimatedETA)
	}
}
