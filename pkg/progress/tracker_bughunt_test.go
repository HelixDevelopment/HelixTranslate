package progress

import (
	"testing"
	"time"
)

// Bug-hunt wave (pkg/progress). Reproduce-first (§11.4.115): each test fails on
// the current code, the source fix flips it green, and reverting the fix makes
// it FAIL again (mutation-proven).

// Bug C — GetProgress() clobbers the "Completed" ETA when items are tracked.
// After Complete() the tracker holds Status="completed", PercentComplete=100,
// EstimatedETA="Completed". But GetProgress() unconditionally recomputes the
// items-based ETA whenever ItemsCompleted>0 && ItemsTotal>0, overwriting the
// "Completed" marker with an items projection (empty/garbage), so a finished
// translation reports a non-"Completed" ETA to the dashboard.
//
// FACT (captured on pre-fix code):
//
//	Status="completed" PercentComplete=100 EstimatedETA=""   (want "Completed")
func TestGetProgress_CompletedKeepsCompletedETA(t *testing.T) {
	tr := NewTracker("s", "B", 10, "ru", "sr", "p", "m")
	tr.SetTotal(5)
	tr.IncrementCompleted()
	tr.IncrementCompleted() // completed=2, total=5  -> items ETA path is active
	tr.Complete()

	p := tr.GetProgress()
	if p.PercentComplete != 100.0 {
		t.Fatalf("setup: expected 100%%, got %v", p.PercentComplete)
	}
	if p.EstimatedETA != "Completed" {
		t.Fatalf("after Complete(), GetProgress ETA=%q, want %q (items path clobbered it)",
			p.EstimatedETA, "Completed")
	}
}

// Companion: a genuinely IN-PROGRESS run (not complete, <100%) MUST still get
// the items-based ETA from GetProgress — proving the fix only suppresses the
// override at completion, not in the normal case.
func TestGetProgress_InProgress_StillComputesItemsETA(t *testing.T) {
	tr := NewTracker("s", "B", 0, "ru", "sr", "p", "m") // 0 chapters -> chapter path off
	tr.mu.Lock()
	tr.progress.StartTime = time.Now().Add(-100 * time.Second)
	tr.mu.Unlock()
	tr.SetTotal(10)
	tr.IncrementCompleted()
	tr.IncrementCompleted() // 2/10 done, elapsed 100s -> avg 50s, remaining 8 -> 400s
	p := tr.GetProgress()
	if p.EstimatedETA == "" || p.EstimatedETA == "Completed" {
		t.Fatalf("in-progress run must compute an items ETA, got %q", p.EstimatedETA)
	}
}
