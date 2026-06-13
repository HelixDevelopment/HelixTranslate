package progress

import (
	"testing"
	"time"
)

// --- Bug 1: GetProgress() must not mutate shared state (data race). ---
// Covered by TestTracker_ThreadSafety under -race; this adds an explicit
// behavioral guard that concurrent GetProgress callers each get a consistent,
// self-contained copy.
func TestGetProgress_ConcurrentReadsConsistent(t *testing.T) {
	tr := NewTracker("s", "B", 10, "ru", "sr", "p", "m")
	tr.SetTotal(100)
	tr.IncrementCompleted()
	tr.IncrementCompleted()
	done := make(chan TranslationProgress, 50)
	for i := 0; i < 50; i++ {
		go func() { done <- tr.GetProgress() }()
	}
	for i := 0; i < 50; i++ {
		p := <-done
		if p.ItemsCompleted != 2 || p.ItemsTotal != 100 {
			t.Fatalf("inconsistent snapshot: completed=%d total=%d", p.ItemsCompleted, p.ItemsTotal)
		}
	}
}

// --- Bug 2: Complete() before any chapter update must report 100%. ---
func TestComplete_BeforeAnyChapter_Is100Percent(t *testing.T) {
	tr := NewTracker("s", "B", 10, "ru", "sr", "p", "m")
	tr.Complete()
	p := tr.GetProgress()
	if p.PercentComplete != 100.0 {
		t.Fatalf("Complete() must be 100%%, got %v", p.PercentComplete)
	}
	if p.EstimatedETA != "Completed" {
		t.Fatalf("Complete() ETA must be 'Completed', got %q", p.EstimatedETA)
	}
}

// --- Bug 3: percentage must never go negative (CurrentChapter==0). ---
func TestUpdateProgress_NeverNegative(t *testing.T) {
	tr := NewTracker("s", "B", 10, "ru", "sr", "p", "m")
	tr.IncrementCompleted() // updateProgress while CurrentChapter==0
	if p := tr.GetProgress(); p.PercentComplete < 0 {
		t.Fatalf("negative PercentComplete %v", p.PercentComplete)
	}
	// Also via section path
	tr2 := NewTracker("s", "B", 10, "ru", "sr", "p", "m")
	tr2.UpdateSection(3) // CurrentChapter still 0
	if p := tr2.GetProgress(); p.PercentComplete < 0 {
		t.Fatalf("negative PercentComplete via section %v", p.PercentComplete)
	}
}

// --- Bug 4: ETA must use float projection, not truncated time.Duration(float). ---
func TestETA_FractionalPercent_NoTruncation(t *testing.T) {
	tr := NewTracker("s", "B", 40, "ru", "sr", "p", "m")
	tr.mu.Lock()
	tr.progress.StartTime = time.Now().Add(-100 * time.Second)
	tr.mu.Unlock()
	tr.UpdateChapter(2, "c", 0) // (2-1)/40*100 = 2.5%
	p := tr.GetProgress()
	if p.PercentComplete < 2.49 || p.PercentComplete > 2.51 {
		t.Fatalf("setup expected 2.5%%, got %v", p.PercentComplete)
	}
	// elapsed=100s @ 2.5% => total=4000s, remaining≈3900s = 1h5m.
	// Truncating time.Duration(2.5)=2 would give remaining=4900s = 1h21m40s.
	if p.EstimatedETA != "1 hour 5 minutes" {
		t.Fatalf("ETA truncation: expected '1 hour 5 minutes' got %q", p.EstimatedETA)
	}
}

// --- Bug 5: a percentage in (0,1) must not divide-by-zero panic. ---
func TestETA_SubOnePercent_NoPanic(t *testing.T) {
	tr := NewTracker("s", "B", 300, "ru", "sr", "p", "m")
	tr.mu.Lock()
	tr.progress.StartTime = time.Now().Add(-50 * time.Second)
	tr.mu.Unlock()
	tr.UpdateChapter(1, "c", 2)
	tr.UpdateSection(1) // 0 < pc < 1, previously panicked at tracker.go:186
	p := tr.GetProgress()
	if p.PercentComplete <= 0 || p.PercentComplete >= 1 {
		t.Fatalf("setup expected 0<pc<1, got %v", p.PercentComplete)
	}
	if p.EstimatedETA == "" || p.EstimatedETA == "Calculating..." {
		t.Fatalf("expected a computed ETA for in-progress sub-1%%, got %q", p.EstimatedETA)
	}
}

// --- ETA via items path must not produce negative remaining when overshooting. ---
func TestGetProgress_ItemsETA_NoNegativeWhenOvershoot(t *testing.T) {
	tr := NewTracker("s", "B", 10, "ru", "sr", "p", "m")
	tr.SetTotal(2)
	tr.IncrementCompleted()
	tr.IncrementCompleted()
	tr.IncrementCompleted() // completed(3) > total(2)
	p := tr.GetProgress()
	// remainingItems clamped to 0 => ETA "" (zero duration), never a wild value.
	if p.ItemsCompleted != 3 {
		t.Fatalf("expected 3 completed, got %d", p.ItemsCompleted)
	}
}

// --- Negative section input must not yield a negative percentage. ---
// UpdateSection does not validate its argument; a negative section would push
// the composite percentage below zero without the low clamp.
func TestUpdateSection_NegativeInput_PercentClampedToZero(t *testing.T) {
	tr := NewTracker("s", "B", 10, "ru", "sr", "p", "m")
	tr.UpdateChapter(1, "c", 10) // base (1-1)/10 = 0%
	tr.UpdateSection(-100)       // section fraction strongly negative
	p := tr.GetProgress()
	if p.PercentComplete < 0 {
		t.Fatalf("negative section produced negative percent: %v", p.PercentComplete)
	}
	if p.PercentComplete != 0.0 {
		t.Fatalf("expected clamp to 0.0, got %v", p.PercentComplete)
	}
}
