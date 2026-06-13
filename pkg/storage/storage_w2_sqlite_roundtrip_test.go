package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// W2 slice: REAL SQLite round-trip tests. SQLite (file / driver) is a REAL
// backend per §11.4.27 — every assertion below reads back exact persisted
// values, not nil-error. These target branches the existing suite leaves
// uncovered: cache hit access-count increment, INSERT-OR-REPLACE overwrite,
// statistics over real rows, list ordering/pagination, nullable EndTime /
// ErrorMessage round-trip, and the NewSQLiteStorage pool/error paths.

func newW2SQLite(t *testing.T) *SQLiteStorage {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "w2.db")
	st, err := NewSQLiteStorage(&Config{
		Type:         "sqlite",
		Database:     dbPath,
		MaxOpenConns: 4,
		MaxIdleConns: 2,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func mkSession(id, status string) *TranslationSession {
	now := time.Now()
	return &TranslationSession{
		ID:             id,
		BookTitle:      "Book " + id,
		InputFile:      "/in/" + id + ".fb2",
		OutputFile:     "/out/" + id + ".epub",
		SourceLanguage: "ru",
		TargetLanguage: "en",
		Provider:       "openai",
		Model:          "gpt-4",
		Status:         status,
		StartTime:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewSQLiteStorage pool settings + a deliberately broken DSN error path.
func TestW2_NewSQLiteStorage_PoolAndError(t *testing.T) {
	st := newW2SQLite(t) // exercises MaxOpenConns/MaxIdleConns branches
	if st == nil || st.db == nil {
		t.Fatal("expected initialized storage with non-nil *sql.DB")
	}
	if err := st.Ping(context.Background()); err != nil {
		t.Fatalf("Ping on fresh DB: %v", err)
	}

	// Unwritable directory path => schema init / open should error, not panic.
	_, err := NewSQLiteStorage(&Config{
		Type:     "sqlite",
		Database: "/this/path/does/not/exist/nope.db",
	})
	if err == nil {
		t.Fatal("expected error opening sqlite at non-existent directory, got nil")
	}
}

// Cache hit MUST increment access_count and persist the new value — the
// branch in GetCachedTranslation that runs the UPDATE on a hit.
func TestW2_SQLite_CacheHitIncrementsAccessCount(t *testing.T) {
	st := newW2SQLite(t)
	ctx := context.Background()

	c := &TranslationCache{
		ID:             "w2-c1",
		SourceText:     "Привет",
		TargetText:     "Hello",
		SourceLanguage: "ru",
		TargetLanguage: "en",
		Provider:       "openai",
		Model:          "gpt-4",
		CreatedAt:      time.Now(),
		AccessCount:    0,
		LastAccessedAt: time.Now(),
	}
	if err := st.CacheTranslation(ctx, c); err != nil {
		t.Fatalf("CacheTranslation: %v", err)
	}

	// First hit -> exact value returned.
	got1, err := st.GetCachedTranslation(ctx, "Привет", "ru", "en", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("GetCachedTranslation #1: %v", err)
	}
	if got1 == nil || got1.TargetText != "Hello" {
		t.Fatalf("hit #1 wrong: %+v", got1)
	}

	// Second hit must observe access_count incremented by the prior hit's UPDATE.
	got2, err := st.GetCachedTranslation(ctx, "Привет", "ru", "en", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("GetCachedTranslation #2: %v", err)
	}
	if got2.AccessCount < 1 {
		t.Fatalf("expected access_count incremented after a hit, got %d", got2.AccessCount)
	}
}

// INSERT OR REPLACE: caching the same ID with a new target MUST overwrite.
func TestW2_SQLite_CacheUpsertOverwrites(t *testing.T) {
	st := newW2SQLite(t)
	ctx := context.Background()

	first := &TranslationCache{
		ID: "w2-up", SourceText: "Дом", TargetText: "Houze (typo)",
		SourceLanguage: "ru", TargetLanguage: "en", Provider: "openai", Model: "gpt-4",
		CreatedAt: time.Now(), LastAccessedAt: time.Now(),
	}
	if err := st.CacheTranslation(ctx, first); err != nil {
		t.Fatalf("cache first: %v", err)
	}
	corrected := *first
	corrected.TargetText = "House"
	if err := st.CacheTranslation(ctx, &corrected); err != nil {
		t.Fatalf("cache corrected: %v", err)
	}

	got, err := st.GetCachedTranslation(ctx, "Дом", "ru", "en", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("get after upsert: %v", err)
	}
	if got == nil || got.TargetText != "House" {
		t.Fatalf("upsert did not overwrite: %+v", got)
	}
}

// Missing key MUST return (nil, nil) — the cache-miss branch.
func TestW2_SQLite_CacheMissReturnsNil(t *testing.T) {
	st := newW2SQLite(t)
	got, err := st.GetCachedTranslation(context.Background(), "absent", "ru", "en", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("unexpected error on miss: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil on cache miss, got %+v", got)
	}
}

// Update round-trip including nullable EndTime + ErrorMessage transitions.
func TestW2_SQLite_UpdateSession_NullableFields(t *testing.T) {
	st := newW2SQLite(t)
	ctx := context.Background()

	s := mkSession("w2-upd", "translating")
	if err := st.CreateSession(ctx, s); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Initially no EndTime / no error -> reads back as zero/empty.
	got, err := st.GetSession(ctx, "w2-upd")
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.EndTime != nil {
		t.Fatalf("expected nil EndTime initially, got %v", got.EndTime)
	}
	if got.ErrorMessage != "" {
		t.Fatalf("expected empty ErrorMessage initially, got %q", got.ErrorMessage)
	}

	// Complete it: set EndTime + ErrorMessage; assert they persist.
	end := time.Now().Add(2 * time.Minute)
	s.Status = "completed"
	s.EndTime = &end
	s.ErrorMessage = "recovered after retry"
	s.PercentComplete = 100
	s.ItemsCompleted = 42
	if err := st.UpdateSession(ctx, s); err != nil {
		t.Fatalf("UpdateSession: %v", err)
	}

	got2, err := st.GetSession(ctx, "w2-upd")
	if err != nil {
		t.Fatalf("GetSession after update: %v", err)
	}
	if got2.Status != "completed" || got2.PercentComplete != 100 || got2.ItemsCompleted != 42 {
		t.Fatalf("update not persisted: %+v", got2)
	}
	if got2.EndTime == nil {
		t.Fatal("expected EndTime persisted, got nil")
	}
	if got2.ErrorMessage != "recovered after retry" {
		t.Fatalf("ErrorMessage round-trip mismatch: %q", got2.ErrorMessage)
	}
}

// ListSessions ordering (created_at DESC) + pagination via LIMIT/OFFSET.
func TestW2_SQLite_ListSessions_OrderAndPagination(t *testing.T) {
	st := newW2SQLite(t)
	ctx := context.Background()

	base := time.Now().Add(-time.Hour)
	for i := 0; i < 5; i++ {
		s := mkSession(
			"w2-list-"+string(rune('a'+i)),
			"completed",
		)
		// Stagger created_at so DESC ordering is deterministic.
		s.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		if err := st.CreateSession(ctx, s); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}

	// Page 1: limit 2 -> the two most recent (i=4, i=3).
	page1, err := st.ListSessions(ctx, 2, 0)
	if err != nil {
		t.Fatalf("ListSessions page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("expected 2 rows on page1, got %d", len(page1))
	}
	if page1[0].ID != "w2-list-e" || page1[1].ID != "w2-list-d" {
		t.Fatalf("DESC order wrong: %s, %s", page1[0].ID, page1[1].ID)
	}

	// Page 2: offset 2, limit 2 -> i=2, i=1.
	page2, err := st.ListSessions(ctx, 2, 2)
	if err != nil {
		t.Fatalf("ListSessions page2: %v", err)
	}
	if len(page2) != 2 || page2[0].ID != "w2-list-c" {
		t.Fatalf("pagination offset wrong: got %d rows, first=%v", len(page2), page2)
	}
}

// GetStatistics over REAL rows: counts by status, total cache, avg duration,
// and cache-hit-rate branch (totalAccess > totalTranslations).
func TestW2_SQLite_GetStatistics_RealRows(t *testing.T) {
	st := newW2SQLite(t)
	ctx := context.Background()

	// 2 completed (with end_time for avg duration), 1 error, 1 translating.
	start := time.Now().Add(-10 * time.Minute)
	for i, st2 := range []string{"completed", "completed"} {
		s := mkSession("w2-st-c"+string(rune('0'+i)), st2)
		s.StartTime = start
		end := start.Add(time.Duration(60+i*60) * time.Second)
		s.EndTime = &end
		if err := st.CreateSession(ctx, s); err != nil {
			t.Fatalf("create completed %d: %v", i, err)
		}
		if err := st.UpdateSession(ctx, s); err != nil { // persists end_time
			t.Fatalf("update completed %d: %v", i, err)
		}
	}
	if err := st.CreateSession(ctx, mkSession("w2-st-err", "error")); err != nil {
		t.Fatalf("create error: %v", err)
	}
	if err := st.CreateSession(ctx, mkSession("w2-st-ip", "translating")); err != nil {
		t.Fatalf("create in-progress: %v", err)
	}

	// One cache entry, hit twice so access_count rises above translation count.
	c := &TranslationCache{
		ID: "w2-st-cache", SourceText: "x", TargetText: "y",
		SourceLanguage: "ru", TargetLanguage: "en", Provider: "openai", Model: "gpt-4",
		CreatedAt: time.Now(), LastAccessedAt: time.Now(),
	}
	if err := st.CacheTranslation(ctx, c); err != nil {
		t.Fatalf("cache: %v", err)
	}
	_, _ = st.GetCachedTranslation(ctx, "x", "ru", "en", "openai", "gpt-4")
	_, _ = st.GetCachedTranslation(ctx, "x", "ru", "en", "openai", "gpt-4")

	stats, err := st.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("GetStatistics: %v", err)
	}
	if stats.TotalSessions != 4 {
		t.Fatalf("TotalSessions = %d, want 4", stats.TotalSessions)
	}
	if stats.CompletedSessions != 2 {
		t.Fatalf("CompletedSessions = %d, want 2", stats.CompletedSessions)
	}
	if stats.FailedSessions != 1 {
		t.Fatalf("FailedSessions = %d, want 1", stats.FailedSessions)
	}
	if stats.InProgressSessions != 1 {
		t.Fatalf("InProgressSessions = %d, want 1", stats.InProgressSessions)
	}
	if stats.TotalTranslations != 1 {
		t.Fatalf("TotalTranslations = %d, want 1", stats.TotalTranslations)
	}
	if stats.AverageDuration <= 0 {
		t.Fatalf("expected positive AverageDuration for completed sessions, got %v", stats.AverageDuration)
	}
	if stats.CacheHitRate <= 0 {
		t.Fatalf("expected positive CacheHitRate after repeated hits, got %v", stats.CacheHitRate)
	}
}

// CleanupOldCache MUST delete entries older than the cutoff and keep fresh ones.
func TestW2_SQLite_CleanupOldCache_DeletesStale(t *testing.T) {
	st := newW2SQLite(t)
	ctx := context.Background()

	old := &TranslationCache{
		ID: "w2-old", SourceText: "old", TargetText: "stary",
		SourceLanguage: "ru", TargetLanguage: "en", Provider: "openai", Model: "gpt-4",
		CreatedAt:      time.Now().Add(-48 * time.Hour),
		LastAccessedAt: time.Now().Add(-48 * time.Hour),
	}
	fresh := &TranslationCache{
		ID: "w2-fresh", SourceText: "new", TargetText: "novy",
		SourceLanguage: "ru", TargetLanguage: "en", Provider: "openai", Model: "gpt-4",
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}
	if err := st.CacheTranslation(ctx, old); err != nil {
		t.Fatalf("cache old: %v", err)
	}
	if err := st.CacheTranslation(ctx, fresh); err != nil {
		t.Fatalf("cache fresh: %v", err)
	}

	if err := st.CleanupOldCache(ctx, 24*time.Hour); err != nil {
		t.Fatalf("CleanupOldCache: %v", err)
	}

	// Old gone, fresh stays.
	if got, _ := st.GetCachedTranslation(ctx, "old", "ru", "en", "openai", "gpt-4"); got != nil {
		t.Fatalf("expected stale entry removed, still present: %+v", got)
	}
	if got, _ := st.GetCachedTranslation(ctx, "new", "ru", "en", "openai", "gpt-4"); got == nil {
		t.Fatal("expected fresh entry retained, but it was deleted")
	}
}

// DeleteSession removes the row; subsequent GetSession returns a not-found error.
func TestW2_SQLite_DeleteSession_RemovesRow(t *testing.T) {
	st := newW2SQLite(t)
	ctx := context.Background()

	if err := st.CreateSession(ctx, mkSession("w2-del", "translating")); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.DeleteSession(ctx, "w2-del"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := st.GetSession(ctx, "w2-del"); err == nil {
		t.Fatal("expected not-found error after delete, got nil")
	}
}
