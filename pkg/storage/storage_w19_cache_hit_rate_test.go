package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// W19 bug-hunt: GetStatistics cache-hit-rate must NEVER be negative.
//
// Root cause (FACT): the original formula was
//
//	CacheHitRate = (totalAccess - totalTranslations) / totalAccess * 100
//
// where totalAccess = SUM(access_count) (re-reads/HITS) and totalTranslations =
// COUNT(*) cache rows. It assumed every cache entry is re-read at least once
// (totalAccess >= totalTranslations). That is false: entries are routinely
// inserted and never re-read, so totalAccess < totalTranslations, driving the
// reported rate NEGATIVE (e.g. 3 entries, 1 hit => (1-3)/1*100 = -200%).
//
// REPRODUCE-FIRST against the REAL SQLite backend (in-process driver, a real
// backend per §11.4.27): three distinct entries, only one re-read once. The
// corrected formula hits/(hits+misses) is always in [0, 100).
//
// Polarity per §11.4.115: this is the RED_MODE=0 GREEN guard; reverting the
// source formula to the old (totalAccess-totalTranslations)/totalAccess makes
// the assertion FAIL (mutation-proven).

func newW19SQLite(t *testing.T) *SQLiteStorage {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "w19.db")
	st, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func w19Cache(id, text string) *TranslationCache {
	return &TranslationCache{
		ID: id, SourceText: text, TargetText: "t-" + text,
		SourceLanguage: "en", TargetLanguage: "sr", Provider: "openai", Model: "gpt-4",
		CreatedAt: time.Now(), LastAccessedAt: time.Now(),
	}
}

func TestW19_CacheHitRate_NeverNegative_RarelyReadEntries(t *testing.T) {
	st := newW19SQLite(t)
	ctx := context.Background()

	// 3 distinct entries (3 misses-on-insert), only ONE re-read once.
	for _, txt := range []string{"a", "b", "c"} {
		if err := st.CacheTranslation(ctx, w19Cache("id-"+txt, txt)); err != nil {
			t.Fatalf("CacheTranslation(%s): %v", txt, err)
		}
	}
	if _, err := st.GetCachedTranslation(ctx, "a", "en", "sr", "openai", "gpt-4"); err != nil {
		t.Fatalf("GetCachedTranslation: %v", err)
	}

	stats, err := st.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("GetStatistics: %v", err)
	}

	// The old formula produced -200% here. Assert sane bounds.
	if stats.CacheHitRate < 0 {
		t.Fatalf("BUG: negative CacheHitRate=%.2f (totalTranslations=%d) — hit rate must be >= 0",
			stats.CacheHitRate, stats.TotalTranslations)
	}
	if stats.CacheHitRate > 100 {
		t.Fatalf("BUG: CacheHitRate=%.2f exceeds 100%%", stats.CacheHitRate)
	}
	// 1 hit, 3 misses => 1/(1+3)*100 = 25%.
	if got, want := stats.CacheHitRate, 25.0; got < want-0.01 || got > want+0.01 {
		t.Fatalf("CacheHitRate=%.4f, want %.1f (hits/(hits+misses)=1/4)", got, want)
	}
}

// A second independent point: with MANY re-reads on a single entry the rate
// rises toward but never reaches 100% — and stays positive (the existing W2
// "> 0 after repeated hits" contract is preserved by the new formula).
func TestW19_CacheHitRate_PositiveOnRepeatedHits(t *testing.T) {
	st := newW19SQLite(t)
	ctx := context.Background()

	if err := st.CacheTranslation(ctx, w19Cache("solo", "x")); err != nil {
		t.Fatalf("CacheTranslation: %v", err)
	}
	for i := 0; i < 9; i++ {
		if _, err := st.GetCachedTranslation(ctx, "x", "en", "sr", "openai", "gpt-4"); err != nil {
			t.Fatalf("GetCachedTranslation #%d: %v", i, err)
		}
	}

	stats, err := st.GetStatistics(ctx)
	if err != nil {
		t.Fatalf("GetStatistics: %v", err)
	}
	// 9 hits, 1 miss => 9/(9+1)*100 = 90%.
	if got, want := stats.CacheHitRate, 90.0; got < want-0.01 || got > want+0.01 {
		t.Fatalf("CacheHitRate=%.4f, want %.1f", got, want)
	}
}
