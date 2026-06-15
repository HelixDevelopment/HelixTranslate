package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgreSQLStorage implements Storage using PostgreSQL
type PostgreSQLStorage struct {
	db *sql.DB
}

// NewPostgreSQLStorage creates a new PostgreSQL storage
func NewPostgreSQLStorage(config *Config) (*PostgreSQLStorage, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.Username, config.Password, config.Database, config.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings. Bound the pool by DEFAULT (W15): with an
	// unconfigured Config{} (MaxOpenConns==0) Go's database/sql pool is UNLIMITED,
	// so concurrent callers each open a connection and exhaust the server's
	// max_connections (~100) — `pq: sorry, too many clients already`, a real
	// connection-exhaustion / DoS risk surfaced by the W15 real-Postgres chaos test.
	maxOpen := config.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	db.SetMaxOpenConns(maxOpen)

	maxIdle := config.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}
	db.SetMaxIdleConns(maxIdle)

	if config.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	storage := &PostgreSQLStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the necessary tables
func (s *PostgreSQLStorage) initSchema() error {
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
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP,
		error_message TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
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
		created_at TIMESTAMP NOT NULL,
		access_count INTEGER DEFAULT 0,
		last_accessed_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_cache_lookup ON translation_cache(source_text, source_language, target_language, provider, model);
	CREATE INDEX IF NOT EXISTS idx_cache_last_accessed ON translation_cache(last_accessed_at);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Migrate legacy DBs to the lookup_hash + UNIQUE-tuple model. Idempotent and
	// safe to run on every open.
	return s.migrateCacheUniqueTuple()
}

// migrateCacheUniqueTuple ensures the translation_cache table carries a
// lookup_hash column and a UNIQUE index on it, deduplicating any pre-existing
// duplicate-tuple rows (keeping the freshest) BEFORE the index is added so the
// index creation never fails on a populated legacy DB. Idempotent. §11.4.124/§9:
// only redundant duplicate rows are removed; the kept row is the freshest per
// tuple — exactly the row the corrected lookup would serve.
func (s *PostgreSQLStorage) migrateCacheUniqueTuple() error {
	// 1) Add the lookup_hash column if a legacy schema lacks it (IF NOT EXISTS is
	//    supported by PostgreSQL 9.6+ and is idempotent).
	if _, err := s.db.Exec(
		`ALTER TABLE translation_cache ADD COLUMN IF NOT EXISTS lookup_hash TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("add lookup_hash column: %w", err)
	}

	// 2) Backfill lookup_hash for rows missing it, computed in Go so it matches
	//    cacheLookupHash exactly (sha256 over the NUL-joined tuple).
	if err := s.backfillCacheLookupHashes(); err != nil {
		return fmt.Errorf("backfill lookup_hash: %w", err)
	}

	// 3) Deduplicate by lookup_hash, keeping the freshest row per tuple. ctid is
	//    PostgreSQL's stable per-row physical identifier.
	if _, err := s.db.Exec(`
		DELETE FROM translation_cache
		WHERE ctid NOT IN (
			SELECT keep_ctid FROM (
				SELECT ctid AS keep_ctid,
				       ROW_NUMBER() OVER (
				           PARTITION BY lookup_hash
				           ORDER BY created_at DESC, last_accessed_at DESC
				       ) AS rn
				FROM translation_cache
			) ranked
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

// backfillCacheLookupHashes computes lookup_hash for any row whose value is empty
// (legacy rows added before the column existed).
func (s *PostgreSQLStorage) backfillCacheLookupHashes() error {
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
			`UPDATE translation_cache SET lookup_hash = $1 WHERE id = $2`, h, rk.id,
		); err != nil {
			return err
		}
	}
	return nil
}

// CreateSession creates a new translation session
func (s *PostgreSQLStorage) CreateSession(ctx context.Context, session *TranslationSession) error {
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
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
func (s *PostgreSQLStorage) GetSession(ctx context.Context, sessionID string) (*TranslationSession, error) {
	query := `
		SELECT id, book_title, input_file, output_file, source_language, target_language,
			provider, model, status, percent_complete, current_chapter, total_chapters,
			items_completed, items_failed, items_total, start_time, end_time, error_message,
			created_at, updated_at
		FROM translation_sessions
		WHERE id = $1
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
func (s *PostgreSQLStorage) UpdateSession(ctx context.Context, session *TranslationSession) error {
	query := `
		UPDATE translation_sessions
		SET book_title = $1, output_file = $2, status = $3, percent_complete = $4,
			current_chapter = $5, total_chapters = $6, items_completed = $7, items_failed = $8,
			items_total = $9, end_time = $10, error_message = $11, updated_at = $12
		WHERE id = $13
	`

	_, err := s.db.ExecContext(ctx, query,
		session.BookTitle, session.OutputFile, session.Status, session.PercentComplete,
		session.CurrentChapter, session.TotalChapters, session.ItemsCompleted, session.ItemsFailed,
		session.ItemsTotal, session.EndTime, session.ErrorMessage, time.Now(), session.ID,
	)

	return err
}

// ListSessions lists translation sessions with pagination
func (s *PostgreSQLStorage) ListSessions(ctx context.Context, limit, offset int) ([]*TranslationSession, error) {
	query := `
		SELECT id, book_title, input_file, output_file, source_language, target_language,
			provider, model, status, percent_complete, current_chapter, total_chapters,
			items_completed, items_failed, items_total, start_time, end_time, error_message,
			created_at, updated_at
		FROM translation_sessions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
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
func (s *PostgreSQLStorage) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM translation_sessions WHERE id = $1", sessionID)
	return err
}

// GetCachedTranslation retrieves a cached translation
func (s *PostgreSQLStorage) GetCachedTranslation(ctx context.Context, sourceText, sourceLanguage, targetLanguage, provider, model string) (*TranslationCache, error) {
	query := `
		SELECT id, source_text, target_text, source_language, target_language, provider, model,
			created_at, access_count, last_accessed_at
		FROM translation_cache
		WHERE source_text = $1 AND source_language = $2 AND target_language = $3 AND provider = $4 AND model = $5
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
		"UPDATE translation_cache SET access_count = access_count + 1, last_accessed_at = $1 WHERE id = $2",
		time.Now(), cache.ID,
	)

	return cache, nil
}

// CacheTranslation caches a translation, idempotent on the lookup tuple
// (source_text, source_language, target_language, provider, model).
//
// Upsert semantics: ON CONFLICT(lookup_hash) — the UNIQUE-tuple key — DO UPDATE,
// NOT ON CONFLICT(id). Keying on lookup_hash means two different ids carrying the
// same tuple collapse into one row (the freshest translation wins) instead of
// leaving a stale shadow row the lookup could serve. access_count is preserved
// (only the payload + freshness columns are overwritten).
func (s *PostgreSQLStorage) CacheTranslation(ctx context.Context, cache *TranslationCache) error {
	lookupHash := cacheLookupHash(
		cache.SourceText, cache.SourceLanguage, cache.TargetLanguage, cache.Provider, cache.Model,
	)

	query := `
		INSERT INTO translation_cache (
			id, source_text, target_text, source_language, target_language, provider, model,
			lookup_hash, created_at, access_count, last_accessed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (lookup_hash) DO UPDATE SET
			target_text = EXCLUDED.target_text,
			created_at = EXCLUDED.created_at,
			last_accessed_at = EXCLUDED.last_accessed_at
	`

	_, err := s.db.ExecContext(ctx, query,
		cache.ID, cache.SourceText, cache.TargetText, cache.SourceLanguage, cache.TargetLanguage,
		cache.Provider, cache.Model, lookupHash, cache.CreatedAt, cache.AccessCount, cache.LastAccessedAt,
	)

	return err
}

// CleanupOldCache removes cache entries older than the specified duration
func (s *PostgreSQLStorage) CleanupOldCache(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	_, err := s.db.ExecContext(ctx, "DELETE FROM translation_cache WHERE last_accessed_at < $1", cutoff)
	return err
}

// GetStatistics returns translation statistics
func (s *PostgreSQLStorage) GetStatistics(ctx context.Context) (*Statistics, error) {
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
		SELECT AVG(EXTRACT(EPOCH FROM (end_time - start_time)))
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
	// hit rate = hits / (hits + misses) = totalAccess / (totalAccess +
	// totalTranslations). The previous (totalAccess - totalTranslations) /
	// totalAccess formula went NEGATIVE when entries were rarely re-read,
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
func (s *PostgreSQLStorage) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the database connection
func (s *PostgreSQLStorage) Close() error {
	return s.db.Close()
}
