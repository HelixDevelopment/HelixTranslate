package storage

import (
	"math"
	"testing"
	"time"
)

// W3 slice — REAL bug: Redis GetStatistics averaged completed-session
// duration over the count of ALL completed sessions, but only sessions that
// actually carry an EndTime contribute a duration. A completed session with
// no EndTime inflated the divisor, dragging the reported average duration
// below the true value (e.g. one 120s session reported as 60s). SQLite and
// PostgreSQL correctly compute AVG(...) WHERE end_time IS NOT NULL; Redis was
// inconsistent and wrong.
//
// The fix extracts a pure incremental accumulator that divides only by the
// number of duration-bearing sessions. RED on the old inline logic; GREEN on
// the corrected helper. Offline, deterministic — no daemon required.

func TestAccumulateAvgDuration_IgnoresSessionsWithoutDuration(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Session A: completed, NO EndTime (no duration).
	// Session B: completed, EndTime = start+120s (duration 120s).
	sessions := []*TranslationSession{
		{Status: "completed", StartTime: start, EndTime: nil},
		{Status: "completed", StartTime: start, EndTime: timePtr(start.Add(120 * time.Second))},
	}

	var avg float64
	var withDuration int64
	for _, s := range sessions {
		if s.Status == "completed" && s.EndTime != nil {
			d := s.EndTime.Sub(s.StartTime).Seconds()
			avg, withDuration = accumulateAvgDuration(avg, withDuration, d)
		}
	}

	if withDuration != 1 {
		t.Fatalf("expected 1 duration-bearing session, got %d", withDuration)
	}
	if math.Abs(avg-120.0) > 1e-9 {
		t.Fatalf("average duration = %.4f, want 120.0000 — a completed session "+
			"without an EndTime must NOT dilute the average", avg)
	}
}

func TestAccumulateAvgDuration_MeanOfMultiple(t *testing.T) {
	// Two real durations: 60s and 180s -> mean 120s.
	var avg float64
	var n int64
	avg, n = accumulateAvgDuration(avg, n, 60)
	avg, n = accumulateAvgDuration(avg, n, 180)
	if n != 2 {
		t.Fatalf("n = %d, want 2", n)
	}
	if math.Abs(avg-120.0) > 1e-9 {
		t.Fatalf("avg = %.4f, want 120.0000", avg)
	}
}

func timePtr(t time.Time) *time.Time { return &t }
