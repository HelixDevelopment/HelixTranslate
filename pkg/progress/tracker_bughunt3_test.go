package progress

import (
	"testing"
)

// Third-pass bug-hunt wave (pkg/progress). Reproduce-first (§11.4.115): the test
// fails on the current code, the source fix flips it green, and reverting the
// fix makes it FAIL again (mutation-proven).

// Bug E — item-based progress reports 0% forever when TotalChapters==0.
//
// updateProgress() only derives PercentComplete from the chapter counters; the
// whole percentage block is guarded by `if t.progress.TotalChapters > 0`. For an
// items-only workflow (a tracker created with totalChapters==0, which is a
// supported construction — GetProgress() even falls back to the items-based ETA
// projection in exactly this case), PercentComplete is therefore NEVER computed
// and stays 0.0 even when every item is completed.
//
// The dashboard binds PercentComplete directly, so an items-only run shows a 0%
// progress bar through to the very end despite ItemsCompleted == ItemsTotal —
// the user sees "no progress" on a run that is actually finished.
//
// FACT (captured on pre-fix code):
//
//	ItemsCompleted=10 ItemsTotal=10 TotalChapters=0 => PercentComplete=0  (want 100)
func TestUpdateProgress_ItemsOnly_DrivesPercent(t *testing.T) {
	tr := NewTracker("s", "B", 0, "ru", "sr", "p", "m") // 0 chapters: items-only mode
	tr.SetTotal(10)
	for i := 0; i < 5; i++ {
		tr.IncrementCompleted()
	}

	// Halfway through the items, the bar must reflect ~50%, not 0%.
	if p := tr.GetProgress(); p.PercentComplete < 49.9 || p.PercentComplete > 50.1 {
		t.Fatalf("items-only halfway: PercentComplete=%v, want ~50", p.PercentComplete)
	}

	for i := 0; i < 5; i++ {
		tr.IncrementCompleted()
	}
	// All items done: must read 100%, not 0%.
	if p := tr.GetProgress(); p.PercentComplete != 100.0 {
		t.Fatalf("items-only complete: PercentComplete=%v, want 100", p.PercentComplete)
	}
}

// Companion (no-regression): the chapter-driven path must be UNCHANGED. When
// TotalChapters>0 the percentage comes from chapters/sections, NOT from items —
// proving the items fallback only kicks in when there are no chapters.
func TestUpdateProgress_ChapterPath_UnaffectedByItems(t *testing.T) {
	tr := NewTracker("s", "B", 10, "ru", "sr", "p", "m") // chapter-driven
	tr.SetTotal(100)
	tr.IncrementCompleted() // 1/100 items, but percent must follow chapters
	tr.UpdateChapter(5, "c", 0)
	// (5-1)/10 = 40% from chapters; items (1/100=1%) must NOT override it.
	if p := tr.GetProgress(); p.PercentComplete < 39.9 || p.PercentComplete > 40.1 {
		t.Fatalf("chapter path: PercentComplete=%v, want ~40 (items must not drive it)", p.PercentComplete)
	}
}
