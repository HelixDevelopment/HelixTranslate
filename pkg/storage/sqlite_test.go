package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSQLiteStorage tests SQLite storage creation
func TestNewSQLiteStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := &Config{
		Type:         "sqlite",
		Database:     dbPath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}

	storage, err := NewSQLiteStorage(config)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Database file should exist")
}

// TestSQLiteStorage_Encryption tests SQLite with encryption key
func TestSQLiteStorage_Encryption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping encryption test in short mode")  // SKIP-OK: #short-mode
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "encrypted.db")

	config := &Config{
		Type:          "sqlite",
		Database:      dbPath,
		EncryptionKey: "test-encryption-key-123",
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		// SQLCipher may not be available - skip test
		t.Skip("SQLCipher not available, skipping encryption test")  // SKIP-OK: #legacy-untriaged
		return
	}
	require.NotNil(t, storage)
	defer storage.Close()

	ctx := context.Background()

	// Test that we can perform operations on encrypted database
	err = storage.Ping(ctx)
	assert.NoError(t, err)
}

// TestSQLiteStorage_CreateSession tests creating a session
func TestSQLiteStorage_CreateSession(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:              "sqlite-test-123",
		BookTitle:       "SQLite Test Book",
		InputFile:       "test.epub",
		OutputFile:      "test_out.epub",
		SourceLanguage:  "ru",
		TargetLanguage:  "sr",
		Provider:        "deepseek",
		Model:           "deepseek-chat",
		Status:          "initializing",
		PercentComplete: 0.0,
		CurrentChapter:  0,
		TotalChapters:   10,
		ItemsCompleted:  0,
		ItemsFailed:     0,
		ItemsTotal:      100,
		StartTime:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := storage.CreateSession(ctx, session)
	require.NoError(t, err)

	// Retrieve and verify
	retrieved, err := storage.GetSession(ctx, session.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.BookTitle, retrieved.BookTitle)
	assert.Equal(t, session.Provider, retrieved.Provider)
}

// TestSQLiteStorage_GetSessionNotFound tests retrieving non-existent session
func TestSQLiteStorage_GetSessionNotFound(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()

	session, err := storage.GetSession(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "not found")
}

// TestSQLiteStorage_UpdateSession tests updating a session
func TestSQLiteStorage_UpdateSession(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create initial session
	session := &TranslationSession{
		ID:              "update-test-123",
		BookTitle:       "Update Test",
		InputFile:       "test.epub",
		SourceLanguage:  "ru",
		TargetLanguage:  "sr",
		Provider:        "deepseek",
		Model:           "deepseek-chat",
		Status:          "initializing",
		PercentComplete: 0.0,
		StartTime:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	err := storage.CreateSession(ctx, session)
	require.NoError(t, err)

	// Update session
	session.Status = "completed"
	session.PercentComplete = 100.0
	endTime := now.Add(time.Hour)
	session.EndTime = &endTime

	err = storage.UpdateSession(ctx, session)
	require.NoError(t, err)

	// Verify update
	updated, err := storage.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, 100.0, updated.PercentComplete)
	require.NotNil(t, updated.EndTime)
}

// TestSQLiteStorage_ListSessions tests listing sessions with pagination
func TestSQLiteStorage_ListSessions(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := &TranslationSession{
			ID:             "list-test-" + string(rune('1'+i)),
			BookTitle:      "List Test " + string(rune('1'+i)),
			InputFile:      "test.epub",
			SourceLanguage: "ru",
			TargetLanguage: "sr",
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			Status:         "initializing",
			StartTime:      now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		err := storage.CreateSession(ctx, session)
		require.NoError(t, err)
	}

	// Test listing
	sessions, err := storage.ListSessions(ctx, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sessions), 5)

	// Test pagination
	page1, err := storage.ListSessions(ctx, 2, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := storage.ListSessions(ctx, 2, 2)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page2), 1)

	// Verify different sessions in each page
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

// TestSQLiteStorage_DeleteSession tests deleting a session
func TestSQLiteStorage_DeleteSession(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "delete-test-123",
		BookTitle:      "Delete Test",
		InputFile:      "test.epub",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         "initializing",
		StartTime:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := storage.CreateSession(ctx, session)
	require.NoError(t, err)

	// Delete session
	err = storage.DeleteSession(ctx, session.ID)
	require.NoError(t, err)

	// Verify deletion
	deleted, err := storage.GetSession(ctx, session.ID)
	assert.Error(t, err)
	assert.Nil(t, deleted)
}

// TestSQLiteStorage_CacheTranslation tests caching translations
func TestSQLiteStorage_CacheTranslation(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	cache := &TranslationCache{
		ID:             "cache-sqlite-123",
		SourceText:     "Test translation",
		TargetText:     "Тест превод",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		AccessCount:    1,
		LastAccessedAt: now,
	}

	err := storage.CacheTranslation(ctx, cache)
	require.NoError(t, err)

	// Retrieve cached translation
	result, err := storage.GetCachedTranslation(ctx,
		cache.SourceText,
		cache.SourceLanguage,
		cache.TargetLanguage,
		cache.Provider,
		cache.Model,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, cache.TargetText, result.TargetText)
	assert.Equal(t, cache.SourceText, result.SourceText)
}

// TestSQLiteStorage_CacheMiss tests cache miss behavior
func TestSQLiteStorage_CacheMiss(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()

	result, err := storage.GetCachedTranslation(ctx,
		"non-existent text",
		"en",
		"sr",
		"deepseek",
		"deepseek-chat",
	)
	require.NoError(t, err)
	assert.Nil(t, result, "Cache miss should return nil")
}

// TestSQLiteStorage_CleanupOldCache tests cache cleanup
func TestSQLiteStorage_CleanupOldCache(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()
	oldTime := time.Now().Add(-48 * time.Hour)
	recentTime := time.Now()

	// Create old cache entry
	oldCache := &TranslationCache{
		ID:             "old-cache-123",
		SourceText:     "Old text",
		TargetText:     "Стари текст",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      oldTime,
		AccessCount:    1,
		LastAccessedAt: oldTime,
	}
	err := storage.CacheTranslation(ctx, oldCache)
	require.NoError(t, err)

	// Create recent cache entry
	recentCache := &TranslationCache{
		ID:             "recent-cache-123",
		SourceText:     "Recent text",
		TargetText:     "Скоро текст",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      recentTime,
		AccessCount:    1,
		LastAccessedAt: recentTime,
	}
	err = storage.CacheTranslation(ctx, recentCache)
	require.NoError(t, err)

	// Cleanup entries older than 24 hours
	err = storage.CleanupOldCache(ctx, 24*time.Hour)
	require.NoError(t, err)

	// Old cache should be deleted
	oldResult, err := storage.GetCachedTranslation(ctx,
		oldCache.SourceText,
		oldCache.SourceLanguage,
		oldCache.TargetLanguage,
		oldCache.Provider,
		oldCache.Model,
	)
	require.NoError(t, err)
	assert.Nil(t, oldResult, "Old cache should be cleaned up")

	// Recent cache should still exist
	recentResult, err := storage.GetCachedTranslation(ctx,
		recentCache.SourceText,
		recentCache.SourceLanguage,
		recentCache.TargetLanguage,
		recentCache.Provider,
		recentCache.Model,
	)
	require.NoError(t, err)
	assert.NotNil(t, recentResult, "Recent cache should remain")
}

// TestSQLiteStorage_GetStatistics tests statistics retrieval
func TestSQLiteStorage_GetStatistics(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create sessions with different statuses
	statuses := []string{"completed", "error", "translating"}
	for i, status := range statuses {
		session := &TranslationSession{
			ID:             "stats-test-" + string(rune('1'+i)),
			BookTitle:      "Stats Test",
			InputFile:      "test.epub",
			SourceLanguage: "ru",
			TargetLanguage: "sr",
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			Status:         status,
			StartTime:      now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if status == "completed" {
			endTime := now.Add(time.Hour)
			session.EndTime = &endTime
		}

		err := storage.CreateSession(ctx, session)
		require.NoError(t, err)
	}

	// Get statistics
	stats, err := storage.GetStatistics(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.GreaterOrEqual(t, stats.TotalSessions, int64(3))
	assert.GreaterOrEqual(t, stats.CompletedSessions, int64(1))
	assert.GreaterOrEqual(t, stats.FailedSessions, int64(1))
	assert.GreaterOrEqual(t, stats.InProgressSessions, int64(1))
}

// TestSQLiteStorage_Ping tests database connection check
func TestSQLiteStorage_Ping(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	ctx := context.Background()

	err := storage.Ping(ctx)
	assert.NoError(t, err)
}

// TestSQLiteStorage_Close tests closing the database
func TestSQLiteStorage_Close(t *testing.T) {
	storage := setupSQLiteTest(t)

	err := storage.Close()
	assert.NoError(t, err)

	// Operations after close should fail
	ctx := context.Background()
	err = storage.Ping(ctx)
	assert.Error(t, err)
}

// TestSQLiteStorage_InterfaceCompliance tests full storage interface
func TestSQLiteStorage_InterfaceCompliance(t *testing.T) {
	storage := setupSQLiteTest(t)
	defer storage.Close()

	// Run common interface tests
	testStorageInterface(t, storage)
}

// setupSQLiteTest creates a temporary SQLite database for testing
func setupSQLiteTest(t *testing.T) *SQLiteStorage {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := &Config{
		Type:         "sqlite",
		Database:     dbPath,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}

	storage, err := NewSQLiteStorage(config)
	require.NoError(t, err)
	require.NotNil(t, storage)

	return storage
}

// BenchmarkSQLiteStorage_CreateSession benchmarks session creation
func BenchmarkSQLiteStorage_CreateSession(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	config := &Config{
		Type:     "sqlite",
		Database: dbPath,
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		b.Fatal(err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := &TranslationSession{
			ID:             "bench-" + string(rune('0'+i%10)),
			BookTitle:      "Bench Book",
			InputFile:      "bench.epub",
			SourceLanguage: "ru",
			TargetLanguage: "sr",
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			Status:         "initializing",
			StartTime:      now,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		_ = storage.CreateSession(ctx, session)
	}
}

// BenchmarkSQLiteStorage_GetSession benchmarks session retrieval
func BenchmarkSQLiteStorage_GetSession(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	config := &Config{
		Type:     "sqlite",
		Database: dbPath,
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		b.Fatal(err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create test session
	session := &TranslationSession{
		ID:             "bench-session",
		BookTitle:      "Bench Book",
		InputFile:      "bench.epub",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         "initializing",
		StartTime:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_ = storage.CreateSession(ctx, session)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetSession(ctx, "bench-session")
	}
}

// BenchmarkSQLiteStorage_CacheTranslation benchmarks cache operations
func BenchmarkSQLiteStorage_CacheTranslation(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	config := &Config{
		Type:     "sqlite",
		Database: dbPath,
	}

	storage, err := NewSQLiteStorage(config)
	if err != nil {
		b.Fatal(err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache := &TranslationCache{
			ID:             "bench-cache-" + string(rune('0'+i%10)),
			SourceText:     "Benchmark text",
			TargetText:     "Бенчмарк текст",
			SourceLanguage: "en",
			TargetLanguage: "sr",
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			CreatedAt:      now,
			AccessCount:    1,
			LastAccessedAt: now,
		}
		_ = storage.CacheTranslation(ctx, cache)
	}
}
