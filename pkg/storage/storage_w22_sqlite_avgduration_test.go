package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// W22 — SQLite GetStatistics.AverageDuration must reflect the real elapsed
// time of completed sessions.
//
// REPRODUCE-FIRST (§11.4.115) against the REAL in-process SQLite backend: create
// a completed session whose EndTime is exactly 3600s after StartTime and assert
// the reported AverageDuration is ~3600. The existing GetStatistics test creates
// such a session but never asserts the duration value, so a silently-zero
// AverageDuration (e.g. if SQLite's julianday() cannot parse the timestamp format
// the go-sqlite3 driver stores) would ship undetected — the dashboard would show
// "0s average" while sessions actually took an hour.
func TestSQLite_GetStatistics_AverageDurationReflectsRealElapsed(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "avgdur.db")
	st, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	start := time.Now().Add(-2 * time.Hour)
	end := start.Add(3600 * time.Second)

	sess := &TranslationSession{
		ID:             "avgdur-1",
		BookTitle:      "B",
		InputFile:      "in.epub",
		SourceLanguage: "ru",
		TargetLanguage: "en",
		Provider:       "openai",
		Model:          "gpt-4",
		Status:         "completed",
		StartTime:      start,
		EndTime:        &end,
		CreatedAt:      start,
		UpdatedAt:      end,
	}
	if err := st.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	stats, err := st.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("GetStatistics: %v", err)
	}

	if stats.CompletedSessions != 1 {
		t.Fatalf("CompletedSessions=%d, want 1", stats.CompletedSessions)
	}

	// EndTime data-loss aspect: a session created already-completed must round-trip
	// its EndTime, not silently drop it to NULL.
	got, err := st.GetSession(ctx, "avgdur-1")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.EndTime == nil {
		t.Fatal("EndTime lost on a session created already-completed (CreateSession dropped end_time)")
	}
	if d := got.EndTime.Sub(got.StartTime).Round(time.Second); d != time.Hour {
		t.Fatalf("round-tripped EndTime-StartTime=%v, want 1h", d)
	}
	// Allow a small tolerance for sub-second rounding in the julianday math.
	if stats.AverageDuration < 3599 || stats.AverageDuration > 3601 {
		t.Fatalf("AverageDuration=%.4f, want ~3600 — a completed 1h session must "+
			"produce a non-zero average (julianday timestamp parse may be failing)",
			stats.AverageDuration)
	}
}
