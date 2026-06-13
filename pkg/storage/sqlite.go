package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteStorage implements Storage using SQLite with SQLCipher encryption
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(config *Config) (*SQLiteStorage, error) {
	dsn := config.Database

	// Add SQLCipher encryption key if provided
	if config.EncryptionKey != "" {
		dsn += fmt.Sprintf("?_pragma_key=%s&_pragma_cipher_page_size=4096", config.EncryptionKey)
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
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
		created_at DATETIME NOT NULL,
		access_count INTEGER DEFAULT 0,
		last_accessed_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_cache_lookup ON translation_cache(source_text, source_language, target_language, provider, model);
	CREATE INDEX IF NOT EXISTS idx_cache_last_accessed ON translation_cache(last_accessed_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateSession creates a new translation session
func (s *SQLiteStorage) CreateSession(ctx context.Context, session *TranslationSession) error {
	query := `
		INSERT INTO translation_sessions (
			id, book_title, input_file, output_file, source_language, target_language,
			provider, model, status, percent_complete, current_chapter, total_chapters,
			items_completed, items_failed, items_total, start_time, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		session.ID, session.BookTitle, session.InputFile, session.OutputFile,
		session.SourceLanguage, session.TargetLanguage, session.Provider, session.Model,
		session.Status, session.PercentComplete, session.CurrentChapter, session.TotalChapters,
		session.ItemsCompleted, session.ItemsFailed, session.ItemsTotal,
		session.StartTime, session.CreatedAt, session.UpdatedAt,
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

// CacheTranslation caches a translation
func (s *SQLiteStorage) CacheTranslation(ctx context.Context, cache *TranslationCache) error {
	query := `
		INSERT OR REPLACE INTO translation_cache (
			id, source_text, target_text, source_language, target_language, provider, model,
			created_at, access_count, last_accessed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		cache.ID, cache.SourceText, cache.TargetText, cache.SourceLanguage, cache.TargetLanguage,
		cache.Provider, cache.Model, cache.CreatedAt, cache.AccessCount, cache.LastAccessedAt,
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
