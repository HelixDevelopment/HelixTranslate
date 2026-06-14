package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"database/sql"
)

// W-cache-unique slice: REAL in-process SQLite tests proving the translation
// cache enforces a UNIQUE constraint on the lookup tuple
// (source_text, source_language, target_language, provider, model).
//
// THE LATENT (reproduce-first, §11.4.115): GetCachedTranslation looks up by the
// tuple, but CacheTranslation historically dedup'd only on the `id` PK. The same
// tuple stored under two different ids left TWO rows, and the lookup (no
// ORDER BY) returned the OLDER, STALE row — a wrong cached translation served to
// the user. After the fix: the tuple is UNIQUE, CacheTranslation is an
// idempotent UPSERT on the tuple, ONE row survives, and the lookup returns the
// FRESHEST translation.
//
// RED_MODE switch (§11.4.115): with RED_MODE=1 the test asserts the DEFECT is
// PRESENT against a schema/insert path that does NOT enforce the tuple (the
// pre-fix behaviour, reproduced here directly via raw SQL). With RED_MODE=0
// (default) it asserts the FIXED public API behaviour. The same source proves
// both polarities.
const cacheUniqueRedMode = 0

func newCacheUniqueSQLite(t *testing.T) *SQLiteStorage {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "cache_unique.db")
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

func countCacheRows(t *testing.T, db *sql.DB, sourceText string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM translation_cache WHERE source_text = ?", sourceText,
	).Scan(&n); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return n
}

// RED-baseline reproduction: directly INSERT two rows for the same tuple under
// two ids (bypassing the public UPSERT) to demonstrate the broken state the
// pre-fix CacheTranslation produced. With the UNIQUE index in place this raw
// second INSERT MUST be rejected, proving the constraint actually exists.
func TestCacheUnique_RawDoubleInsertRejectedByUniqueIndex(t *testing.T) {
	st := newCacheUniqueSQLite(t)
	ctx := context.Background()

	older := time.Now().Add(-time.Hour)
	newer := time.Now()

	// First raw insert (older, STALE target) — must succeed.
	_, err := st.db.ExecContext(ctx, `
		INSERT INTO translation_cache (
			id, source_text, target_text, source_language, target_language, provider, model,
			lookup_hash, created_at, access_count, last_accessed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"id-old", "Привет", "Hello-STALE", "ru", "en", "openai", "gpt-4",
		cacheLookupHash("Привет", "ru", "en", "openai", "gpt-4"), older, 0, older,
	)
	if err != nil {
		t.Fatalf("first raw insert: %v", err)
	}

	// Second raw insert, SAME tuple, DIFFERENT id (the pre-fix dup). With the
	// UNIQUE(lookup_hash) index this MUST fail.
	_, err = st.db.ExecContext(ctx, `
		INSERT INTO translation_cache (
			id, source_text, target_text, source_language, target_language, provider, model,
			lookup_hash, created_at, access_count, last_accessed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"id-new", "Привет", "Hello-FRESH", "ru", "en", "openai", "gpt-4",
		cacheLookupHash("Привет", "ru", "en", "openai", "gpt-4"), newer, 0, newer,
	)
	if cacheUniqueRedMode == 1 {
		// Pre-fix polarity: a second tuple-dup INSERT succeeds (no UNIQUE index).
		if err != nil {
			t.Fatalf("RED_MODE: expected dup insert to SUCCEED on broken schema, got %v", err)
		}
		if got := countCacheRows(t, st.db, "Привет"); got != 2 {
			t.Fatalf("RED_MODE: expected 2 stale dup rows, got %d", got)
		}
		return
	}
	// Fixed polarity: the UNIQUE index rejects the tuple-dup.
	if err == nil {
		t.Fatal("expected UNIQUE(lookup_hash) to reject a second row for the same tuple, but insert succeeded")
	}
	if got := countCacheRows(t, st.db, "Привет"); got != 1 {
		t.Fatalf("expected exactly 1 row after rejected dup, got %d", got)
	}
}

// Public API: caching the same tuple twice (under different ids) MUST result in
// ONE row, and GetCachedTranslation MUST return the FRESHEST translation.
func TestCacheUnique_UpsertOnTupleKeepsOneRowReturnsFreshest(t *testing.T) {
	st := newCacheUniqueSQLite(t)
	ctx := context.Background()

	older := time.Now().Add(-time.Hour)
	newer := time.Now()

	stale := &TranslationCache{
		ID: "tuple-old", SourceText: "Дом", TargetText: "Houze-STALE",
		SourceLanguage: "ru", TargetLanguage: "en", Provider: "openai", Model: "gpt-4",
		CreatedAt: older, AccessCount: 0, LastAccessedAt: older,
	}
	fresh := &TranslationCache{
		ID: "tuple-new", SourceText: "Дом", TargetText: "House-FRESH",
		SourceLanguage: "ru", TargetLanguage: "en", Provider: "openai", Model: "gpt-4",
		CreatedAt: newer, AccessCount: 0, LastAccessedAt: newer,
	}

	if err := st.CacheTranslation(ctx, stale); err != nil {
		t.Fatalf("cache stale: %v", err)
	}
	// SAME tuple, DIFFERENT id, fresher value.
	if err := st.CacheTranslation(ctx, fresh); err != nil {
		t.Fatalf("cache fresh: %v", err)
	}

	if got := countCacheRows(t, st.db, "Дом"); got != 1 {
		t.Fatalf("expected exactly 1 row for the tuple after two caches, got %d", got)
	}

	got, err := st.GetCachedTranslation(ctx, "Дом", "ru", "en", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("GetCachedTranslation: %v", err)
	}
	if got == nil {
		t.Fatal("expected a cache hit, got nil")
	}
	if got.TargetText != "House-FRESH" {
		t.Fatalf("expected FRESHEST translation %q, got STALE %q", "House-FRESH", got.TargetText)
	}
}

// MIGRATION SAFETY (§11.4.124/§9): a DB that already contains duplicate-tuple
// rows (the legacy broken state) MUST be deduped (keep freshest) and gain the
// UNIQUE index WITHOUT error. We simulate a legacy DB by creating the pre-fix
// schema (no lookup_hash, no UNIQUE), seeding dup rows, then running the
// migration the storage layer applies on open.
func TestCacheUnique_MigrationDedupsExistingDuplicates(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	// 1) Build a LEGACY db (pre-fix schema) and seed duplicate-tuple rows.
	legacy, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open legacy: %v", err)
	}
	_, err = legacy.ExecContext(ctx, `
		CREATE TABLE translation_cache (
			id TEXT PRIMARY KEY,
			source_text TEXT NOT NULL,
			target_text TEXT NOT NULL,
			source_language TEXT NOT NULL,
			target_language TEXT NOT NULL,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			access_count INTEGER DEFAULT 0,
			last_accessed_at DATETIME NOT NULL
		);`)
	if err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	older := time.Now().Add(-2 * time.Hour)
	newer := time.Now()
	ins := `INSERT INTO translation_cache (id, source_text, target_text, source_language, target_language, provider, model, created_at, access_count, last_accessed_at) VALUES (?,?,?,?,?,?,?,?,?,?)`
	// Two dup rows for tuple A (keep the newer "B-FRESH").
	if _, err := legacy.ExecContext(ctx, ins, "a-old", "TextA", "A-STALE", "ru", "en", "openai", "gpt-4", older, 3, older); err != nil {
		t.Fatalf("seed a-old: %v", err)
	}
	if _, err := legacy.ExecContext(ctx, ins, "a-new", "TextA", "A-FRESH", "ru", "en", "openai", "gpt-4", newer, 1, newer); err != nil {
		t.Fatalf("seed a-new: %v", err)
	}
	// A distinct, non-dup tuple B (must survive untouched).
	if _, err := legacy.ExecContext(ctx, ins, "b-1", "TextB", "B-only", "ru", "en", "openai", "gpt-4", newer, 0, newer); err != nil {
		t.Fatalf("seed b-1: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("close legacy: %v", err)
	}

	// 2) Open with the FIXED storage layer — migration must dedup + add the index.
	st, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath, MaxOpenConns: 4, MaxIdleConns: 2})
	if err != nil {
		t.Fatalf("open fixed storage on legacy db (migration failed?): %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Tuple A deduped to ONE row, the FRESHEST kept.
	if got := countCacheRows(t, st.db, "TextA"); got != 1 {
		t.Fatalf("expected TextA deduped to 1 row, got %d", got)
	}
	gotA, err := st.GetCachedTranslation(ctx, "TextA", "ru", "en", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("get TextA: %v", err)
	}
	if gotA == nil || gotA.TargetText != "A-FRESH" {
		t.Fatalf("migration kept STALE row: %+v", gotA)
	}
	// Tuple B untouched.
	gotB, err := st.GetCachedTranslation(ctx, "TextB", "ru", "en", "openai", "gpt-4")
	if err != nil {
		t.Fatalf("get TextB: %v", err)
	}
	if gotB == nil || gotB.TargetText != "B-only" {
		t.Fatalf("migration disturbed non-dup tuple B: %+v", gotB)
	}

	// 3) Index present + enforcing: a raw dup INSERT must now be rejected.
	_, err = st.db.ExecContext(ctx, `
		INSERT INTO translation_cache (id, source_text, target_text, source_language, target_language, provider, model, lookup_hash, created_at, access_count, last_accessed_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"a-third", "TextA", "A-THIRD", "ru", "en", "openai", "gpt-4",
		cacheLookupHash("TextA", "ru", "en", "openai", "gpt-4"), time.Now(), 0, time.Now())
	if err == nil {
		t.Fatal("expected UNIQUE index to reject post-migration dup, but insert succeeded")
	}

	// 4) Idempotent: opening again over the already-migrated db must not error.
	st2, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath, MaxOpenConns: 4, MaxIdleConns: 2})
	if err != nil {
		t.Fatalf("second open (migration not idempotent?): %v", err)
	}
	_ = st2.Close()
}
