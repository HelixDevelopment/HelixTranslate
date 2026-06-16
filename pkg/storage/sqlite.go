package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteStorage implements Storage using SQLite with SQLCipher encryption
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(config *Config) (*SQLiteStorage, error) {
	// Build the DSN with a busy_timeout so a writer waits out the single-file
	// write lock instead of failing immediately with SQLITE_BUSY ("database is
	// locked") under concurrency. Combined with the single-connection default
	// below, concurrent writers serialize cleanly (§11.4.85 stress contract:
	// "Single SQLite file connection must serialize writers cleanly").
	dsn := config.Database
	params := []string{"_busy_timeout=5000"}
	// Add SQLCipher encryption key if provided
	if config.EncryptionKey != "" {
		params = append(params, "_pragma_key="+config.EncryptionKey, "_pragma_cipher_page_size=4096")
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	dsn += sep + strings.Join(params, "&")

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings. SQLite is a single-writer engine on one
	// file: with an unbounded pool, database/sql opens multiple connections and
	// concurrent writers collide (SQLITE_BUSY). Default to a single connection
	// so all access serializes cleanly (sqlite.go has no nested transactions,
	// so a 1-conn pool cannot self-deadlock). An explicit config value still
	// wins; the busy_timeout above then absorbs any residual contention.
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(1)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	storage := &SQLiteStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the necessary tables
func (s *SQLiteStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS translation_sessions (
		id TEXT PRIMARY KEY,
		book_title TEXT NOT NULL,
		input_file TEXT NOT NULL,
		output_file TEXT,
		source_language TEXT NOT NULL,
		target_language TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		status TEXT NOT NULL,
		percent_complete REAL DEFAULT 0,
		current_chapter INTEGER DEFAULT 0,
		total_chapters INTEGER DEFAULT 0,
		items_completed INTEGER DEFAULT 0,
		items_failed INTEGER DEFAULT 0,
		items_total INTEGER DEFAULT 0,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		error_message TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_status ON translation_sessions(status);
	CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON translation_sessions(created_at DESC);

	CREATE TABLE IF NOT EXISTS translation_cache (
		id TEXT PRIMARY KEY,
		source_text TEXT NOT NULL,
		target_text TEXT NOT NULL,
		source_language TEXT NOT NULL,
		target_language TEXT NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		lookup_hash TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		access_count INTEGER DEFAULT 0,
		last_accessed_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_cache_lookup ON translation_cache(source_text, source_language, target_language, provider, model);
	CREATE INDEX IF NOT EXISTS idx_cache_last_accessed ON translation_cache(last_accessed_at);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Migrate legacy DBs to the lookup_hash + UNIQUE-tuple model. Idempotent and
	// safe to run on every open (CREATE TABLE above only fires on a brand-new DB).
	return s.migrateCacheUniqueTuple()
}

// migrateCacheUniqueTuple ensures the translation_cache table carries a
// lookup_hash column and a UNIQUE index on it, deduplicating any pre-existing
// duplicate-tuple rows (keeping the freshest) BEFORE the index is added so the
// index creation never fails on a populated legacy DB. Idempotent: every step is
// a no-op once already applied.
//
// §11.4.124/§9 migration safety: dedup keeps the row with the greatest
// (created_at, last_accessed_at) per tuple and deletes the older shadow rows
// (the stale rows the old lookup could serve). No backup is needed beyond the
// caller's normal DB file — only redundant duplicate rows are removed, and the
// kept row is the one the corrected lookup would return anyway.
func (s *SQLiteStorage) migrateCacheUniqueTuple() error {
	// 1) Add the lookup_hash column if a legacy schema lacks it.
	hasCol, err := s.cacheHasColumn("lookup_hash")
	if err != nil {
		return fmt.Errorf("inspect translation_cache columns: %w", err)
	}
	if !hasCol {
		if _, err := s.db.Exec(
			`ALTER TABLE translation_cache ADD COLUMN lookup_hash TEXT NOT NULL DEFAULT ''`,
		); err != nil {
			return fmt.Errorf("add lookup_hash column: %w", err)
		}
	}

	// 2) Backfill lookup_hash for any rows missing it (empty default). Done in Go
	//    so the hash matches cacheLookupHash exactly (sha256 over NUL-joined tuple).
	if err := s.backfillCacheLookupHashes(); err != nil {
		return fmt.Errorf("backfill lookup_hash: %w", err)
	}

	// 3) Deduplicate by lookup_hash, keeping the freshest row per tuple. rowid is
	//    the stable per-row key; we keep the rowid whose (created_at,
	//    last_accessed_at) is greatest and delete the rest.
	if _, err := s.db.Exec(`
		DELETE FROM translation_cache
		WHERE rowid NOT IN (
			SELECT keep_rowid FROM (
				SELECT rowid AS keep_rowid,
				       ROW_NUMBER() OVER (
				           PARTITION BY lookup_hash
				           ORDER BY created_at DESC, last_accessed_at DESC, rowid DESC
				       ) AS rn
				FROM translation_cache
			)
			WHERE rn = 1
		)
	`); err != nil {
		return fmt.Errorf("dedup duplicate-tuple cache rows: %w", err)
	}

	// 4) Add the UNIQUE index (now safe — no duplicate lookup_hash values remain).
	if _, err := s.db.Exec(
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_cache_unique_tuple ON translation_cache(lookup_hash)`,
	); err != nil {
		return fmt.Errorf("create unique tuple index: %w", err)
	}

	return nil
}

// cacheHasColumn reports whether translation_cache has the named column.
func (s *SQLiteStorage) cacheHasColumn(name string) (bool, error) {
	rows, err := s.db.Query(`PRAGMA table_info(translation_cache)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &colName, &colType, &notNull, &dflt, &pk); err != nil {
			return false, err
		}
		if colName == name {
			return true, rows.Err()
		}
	}
	return false, rows.Err()
}

// backfillCacheLookupHashes computes lookup_hash for any row whose value is empty
// (legacy rows, or rows added before the column existed).
func (s *SQLiteStorage) backfillCacheLookupHashes() error {
	rows, err := s.db.Query(
		`SELECT id, source_text, source_language, target_language, provider, model
		 FROM translation_cache WHERE lookup_hash = ''`,
	)
	if err != nil {
		return err
	}
	type rowKey struct {
		id, sourceText, srcLang, tgtLang, provider, model string
	}
	var pending []rowKey
	for rows.Next() {
		var rk rowKey
		if err := rows.Scan(&rk.id, &rk.sourceText, &rk.srcLang, &rk.tgtLang, &rk.provider, &rk.model); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, rk)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, rk := range pending {
		h := cacheLookupHash(rk.sourceText, rk.srcLang, rk.tgtLang, rk.provider, rk.model)
		if _, err := s.db.Exec(
			`UPDATE translation_cache SET lookup_hash = ? WHERE id = ?`, h, rk.id,
		); err != nil {
			return err
		}
	}
	return nil
}

// CreateSession creates a new translation session
func (s *SQLiteStorage) CreateSession(ctx context.Context, session *TranslationSession) error {
	// end_time + error_message MUST be in the INSERT: a session can be created
	// already-completed (EndTime set), e.g. when a finished run is persisted in one
	// shot rather than created-then-updated. Omitting end_time silently dropped it
	// to NULL, which (a) loses the EndTime on GetSession/ListSessions and (b) made
	// GetStatistics.AverageDuration compute over zero rows (WHERE end_time IS NOT
	// NULL) and report 0 while sessions actually took real wall-clock time.
	query := `
		INSERT INTO translation_sessions (
			id, book_title, input_file, output_file, source_language, target_language,
			provider, model, status, percent_complete, current_chapter, total_chapters,
			items_completed, items_failed, items_total, start_time, end_time, error_message,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		session.ID, session.BookTitle, session.InputFile, session.OutputFile,
		session.SourceLanguage, session.TargetLanguage, session.Provider, session.Model,
		session.Status, session.PercentComplete, session.CurrentChapter, session.TotalChapters,
		session.ItemsCompleted, session.ItemsFailed, session.ItemsTotal,
		session.StartTime, session.EndTime, session.ErrorMessage, session.CreatedAt, session.UpdatedAt,
	)

	return err
}

// GetSession retrieves a session by ID
func (s *SQLiteStorage) GetSession(ctx context.Context, sessionID string) (*TranslationSession, error) {
	query := `
		SELECT id, book_title, input_file, output_file, source_language, target_language,
			provider, model, status, percent_complete, current_chapter, total_chapters,
			items_completed, items_failed, items_total, start_time, end_time, error_message,
			created_at, updated_at
		FROM translation_sessions
		WHERE id = ?
	`

	session := &TranslationSession{}
	var endTime sql.NullTime
	var errorMessage sql.NullString

	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID, &session.BookTitle, &session.InputFile, &session.OutputFile,
		&session.SourceLanguage, &session.TargetLanguage, &session.Provider, &session.Model,
		&session.Status, &session.PercentComplete, &session.CurrentChapter, &session.TotalChapters,
		&session.ItemsCompleted, &session.ItemsFailed, &session.ItemsTotal,
		&session.StartTime, &endTime, &errorMessage, &session.CreatedAt, &session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, err
	}

	if endTime.Valid {
		session.EndTime = &endTime.Time
	}
	if errorMessage.Valid {
		session.ErrorMessage = errorMessage.String
	}

	return session, nil
}

// UpdateSession updates an existing session
func (s *SQLiteStorage) UpdateSession(ctx context.Context, session *TranslationSession) error {
	query := `
		UPDATE translation_sessions
		SET book_title = ?, output_file = ?, status = ?, percent_complete = ?,
			current_chapter = ?, total_chapters = ?, items_completed = ?, items_failed = ?,
			items_total = ?, end_time = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		session.BookTitle, session.OutputFile, session.Status, session.PercentComplete,
		session.CurrentChapter, session.TotalChapters, session.ItemsCompleted, session.ItemsFailed,
		session.ItemsTotal, session.EndTime, session.ErrorMessage, time.Now(), session.ID,
	)

	return err
}

// ListSessions lists translation sessions with pagination
func (s *SQLiteStorage) ListSessions(ctx context.Context, limit, offset int) ([]*TranslationSession, error) {
	query := `
		SELECT id, book_title, input_file, output_file, source_language, target_language,
			provider, model, status, percent_complete, current_chapter, total_chapters,
			items_completed, items_failed, items_total, start_time, end_time, error_message,
			created_at, updated_at
		FROM translation_sessions
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*TranslationSession
	for rows.Next() {
		session := &TranslationSession{}
		var endTime sql.NullTime
		var errorMessage sql.NullString

		err := rows.Scan(
			&session.ID, &session.BookTitle, &session.InputFile, &session.OutputFile,
			&session.SourceLanguage, &session.TargetLanguage, &session.Provider, &session.Model,
			&session.Status, &session.PercentComplete, &session.CurrentChapter, &session.TotalChapters,
			&session.ItemsCompleted, &session.ItemsFailed, &session.ItemsTotal,
			&session.StartTime, &endTime, &errorMessage, &session.CreatedAt, &session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if endTime.Valid {
			session.EndTime = &endTime.Time
		}
		if errorMessage.Valid {
			session.ErrorMessage = errorMessage.String
		}

		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// DeleteSession deletes a session
func (s *SQLiteStorage) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM translation_sessions WHERE id = ?", sessionID)
	return err
}

// GetCachedTranslation retrieves a cached translation
func (s *SQLiteStorage) GetCachedTranslation(ctx context.Context, sourceText, sourceLanguage, targetLanguage, provider, model string) (*TranslationCache, error) {
	query := `
		SELECT id, source_text, target_text, source_language, target_language, provider, model,
			created_at, access_count, last_accessed_at
		FROM translation_cache
		WHERE source_text = ? AND source_language = ? AND target_language = ? AND provider = ? AND model = ?
	`

	cache := &TranslationCache{}
	err := s.db.QueryRowContext(ctx, query, sourceText, sourceLanguage, targetLanguage, provider, model).Scan(
		&cache.ID, &cache.SourceText, &cache.TargetText, &cache.SourceLanguage, &cache.TargetLanguage,
		&cache.Provider, &cache.Model, &cache.CreatedAt, &cache.AccessCount, &cache.LastAccessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Update access count and last accessed time
	_, _ = s.db.ExecContext(ctx,
		"UPDATE translation_cache SET access_count = access_count + 1, last_accessed_at = ? WHERE id = ?",
		time.Now(), cache.ID,
	)

	return cache, nil
}

// CacheTranslation caches a translation, idempotent on the lookup tuple
// (source_text, source_language, target_language, provider, model).
//
// Upsert semantics: ON CONFLICT(lookup_hash) DO UPDATE — NOT INSERT OR REPLACE.
// INSERT OR REPLACE deletes the conflicting row and re-inserts, which would lose
// the existing access_count and change the row's identity. ON CONFLICT...DO
// UPDATE keeps the original row (and its accumulated access_count) and overwrites
// only the translation payload + freshness columns, so re-caching the same tuple
// with a corrected/fresher translation updates in place and GetCachedTranslation
// returns the FRESHEST target_text. The conflict is detected on lookup_hash (the
// UNIQUE-tuple key), so two different ids carrying the same tuple collapse to one
// row instead of leaving a stale shadow.
func (s *SQLiteStorage) CacheTranslation(ctx context.Context, cache *TranslationCache) error {
	lookupHash := cacheLookupHash(
		cache.SourceText, cache.SourceLanguage, cache.TargetLanguage, cache.Provider, cache.Model,
	)

	query := `
		INSERT INTO translation_cache (
			id, source_text, target_text, source_language, target_language, provider, model,
			lookup_hash, created_at, access_count, last_accessed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(lookup_hash) DO UPDATE SET
			target_text = excluded.target_text,
			created_at = excluded.created_at,
			last_accessed_at = excluded.last_accessed_at
	`

	_, err := s.db.ExecContext(ctx, query,
		cache.ID, cache.SourceText, cache.TargetText, cache.SourceLanguage, cache.TargetLanguage,
		cache.Provider, cache.Model, lookupHash, cache.CreatedAt, cache.AccessCount, cache.LastAccessedAt,
	)

	return err
}

// CleanupOldCache removes cache entries older than the specified duration
func (s *SQLiteStorage) CleanupOldCache(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.ExecContext(ctx, "DELETE FROM translation_cache WHERE last_accessed_at < ?", cutoff)
	return err
}

// GetStatistics returns translation statistics
func (s *SQLiteStorage) GetStatistics(ctx context.Context) (*Statistics, error) {
	stats := &Statistics{}

	// Total sessions
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM translation_sessions").Scan(&stats.TotalSessions)
	if err != nil {
		return nil, err
	}

	// Completed sessions
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM translation_sessions WHERE status = 'completed'").Scan(&stats.CompletedSessions)
	if err != nil {
		return nil, err
	}

	// Failed sessions
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM translation_sessions WHERE status = 'error'").Scan(&stats.FailedSessions)
	if err != nil {
		return nil, err
	}

	// In progress sessions
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM translation_sessions WHERE status IN ('initializing', 'translating')").Scan(&stats.InProgressSessions)
	if err != nil {
		return nil, err
	}

	// Total translations (cache entries)
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM translation_cache").Scan(&stats.TotalTranslations)
	if err != nil {
		return nil, err
	}

	// Average duration for completed sessions
	var avgDuration sql.NullFloat64
	err = s.db.QueryRowContext(ctx, `
		SELECT AVG(CAST((julianday(end_time) - julianday(start_time)) * 86400 AS REAL))
		FROM translation_sessions
		WHERE status = 'completed' AND end_time IS NOT NULL
	`).Scan(&avgDuration)
	if err != nil {
		return nil, err
	}
	if avgDuration.Valid {
		stats.AverageDuration = avgDuration.Float64
	}

	// Cache hit rate (approximate based on access count).
	//
	// access_count counts re-reads (HITS) of an entry AFTER its initial insert;
	// each distinct entry represents one MISS (the lookup that caused the insert).
	// The hit rate is therefore hits / (hits + misses) = totalAccess /
	// (totalAccess + totalTranslations). The previous formula,
	// (totalAccess - totalTranslations) / totalAccess, went NEGATIVE whenever
	// entries were inserted but rarely re-read (e.g. 3 entries, 1 hit => -200%),
	// reporting a nonsensical cache-hit-rate. The corrected formula is always in
	// [0, 100).
	var totalAccess sql.NullInt64
	err = s.db.QueryRowContext(ctx, "SELECT SUM(access_count) FROM translation_cache").Scan(&totalAccess)
	if err == nil && totalAccess.Valid && totalAccess.Int64 > 0 && stats.TotalTranslations > 0 {
		denom := float64(totalAccess.Int64 + stats.TotalTranslations)
		stats.CacheHitRate = float64(totalAccess.Int64) / denom * 100.0
	}

	return stats, nil
}

// Ping checks the database connection
func (s *SQLiteStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
