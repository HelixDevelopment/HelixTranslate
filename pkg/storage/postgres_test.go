package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPostgreSQLStorage tests PostgreSQL storage creation
func TestNewPostgreSQLStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL test - requires database")
	}

	config := getPostgreSQLTestConfig(t)
	if config == nil {
		t.Skip("PostgreSQL not available for testing")
	}

	storage, err := NewPostgreSQLStorage(config)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Verify connection
	err = storage.Ping(context.Background())
	assert.NoError(t, err)
}

// TestPostgreSQLStorage_CreateSession tests creating a session
func TestPostgreSQLStorage_CreateSession(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:              "postgres-test-123",
		BookTitle:       "PostgreSQL Test Book",
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
}

// TestPostgreSQLStorage_UpdateSession tests updating a session
func TestPostgreSQLStorage_UpdateSession(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "update-test-pg",
		BookTitle:      "Update Test",
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

	// Update
	session.Status = "completed"
	session.PercentComplete = 100.0
	endTime := now.Add(time.Hour)
	session.EndTime = &endTime

	err = storage.UpdateSession(ctx, session)
	require.NoError(t, err)

	// Verify
	updated, err := storage.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, 100.0, updated.PercentComplete)
}

// TestPostgreSQLStorage_ListSessions tests listing with pagination
func TestPostgreSQLStorage_ListSessions(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := &TranslationSession{
			ID:             "list-pg-" + string(rune('A'+i)),
			BookTitle:      "List Test",
			InputFile:      "test.epub",
			SourceLanguage: "ru",
			TargetLanguage: "sr",
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			Status:         "initializing",
			StartTime:      now,
			CreatedAt:      now.Add(time.Duration(i) * time.Second),
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
}

// TestPostgreSQLStorage_DeleteSession tests deletion
func TestPostgreSQLStorage_DeleteSession(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "delete-pg-test",
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

	err = storage.DeleteSession(ctx, session.ID)
	require.NoError(t, err)

	deleted, err := storage.GetSession(ctx, session.ID)
	assert.Error(t, err)
	assert.Nil(t, deleted)
}

// TestPostgreSQLStorage_CacheTranslation tests caching
func TestPostgreSQLStorage_CacheTranslation(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	cache := &TranslationCache{
		ID:             "cache-pg-123",
		SourceText:     "PostgreSQL test",
		TargetText:     "PostgreSQL тест",
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
}

// TestPostgreSQLStorage_CacheUpsert tests ON CONFLICT update
func TestPostgreSQLStorage_CacheUpsert(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	cache := &TranslationCache{
		ID:             "upsert-test-pg",
		SourceText:     "Upsert test",
		TargetText:     "First version",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		AccessCount:    1,
		LastAccessedAt: now,
	}

	// First insert
	err := storage.CacheTranslation(ctx, cache)
	require.NoError(t, err)

	// Update with new translation
	cache.TargetText = "Updated version"
	cache.LastAccessedAt = now.Add(time.Minute)

	err = storage.CacheTranslation(ctx, cache)
	require.NoError(t, err)

	// Verify update
	result, err := storage.GetCachedTranslation(ctx,
		cache.SourceText,
		cache.SourceLanguage,
		cache.TargetLanguage,
		cache.Provider,
		cache.Model,
	)
	require.NoError(t, err)
	assert.Equal(t, "Updated version", result.TargetText)
}

// TestPostgreSQLStorage_CleanupOldCache tests cache cleanup
func TestPostgreSQLStorage_CleanupOldCache(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	oldTime := time.Now().Add(-48 * time.Hour)
	recentTime := time.Now()

	oldCache := &TranslationCache{
		ID:             "old-pg-cache",
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

	recentCache := &TranslationCache{
		ID:             "recent-pg-cache",
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

	// Cleanup
	err = storage.CleanupOldCache(ctx, 24*time.Hour)
	require.NoError(t, err)

	// Verify old cache deleted
	oldResult, err := storage.GetCachedTranslation(ctx,
		oldCache.SourceText,
		oldCache.SourceLanguage,
		oldCache.TargetLanguage,
		oldCache.Provider,
		oldCache.Model,
	)
	require.NoError(t, err)
	assert.Nil(t, oldResult)

	// Verify recent cache remains
	recentResult, err := storage.GetCachedTranslation(ctx,
		recentCache.SourceText,
		recentCache.SourceLanguage,
		recentCache.TargetLanguage,
		recentCache.Provider,
		recentCache.Model,
	)
	require.NoError(t, err)
	assert.NotNil(t, recentResult)
}

// TestPostgreSQLStorage_GetStatistics tests statistics
func TestPostgreSQLStorage_GetStatistics(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create sessions
	statuses := []string{"completed", "error", "translating"}
	for i, status := range statuses {
		session := &TranslationSession{
			ID:             "stats-pg-" + string(rune('1'+i)),
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

	stats, err := storage.GetStatistics(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.GreaterOrEqual(t, stats.TotalSessions, int64(3))
	assert.GreaterOrEqual(t, stats.CompletedSessions, int64(1))
	assert.GreaterOrEqual(t, stats.FailedSessions, int64(1))
}

// TestPostgreSQLStorage_Ping tests connection check
func TestPostgreSQLStorage_Ping(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	err := storage.Ping(context.Background())
	assert.NoError(t, err)
}

// TestPostgreSQLStorage_InterfaceCompliance tests full interface
func TestPostgreSQLStorage_InterfaceCompliance(t *testing.T) {
	storage := setupPostgreSQLTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	testStorageInterface(t, storage)
}

// setupPostgreSQLTest creates PostgreSQL storage for testing
func setupPostgreSQLTest(t *testing.T) *PostgreSQLStorage {
	if testing.Short() {
		t.Skip("Skipping PostgreSQL test - requires database")
		return nil
	}

	config := getPostgreSQLTestConfig(t)
	if config == nil {
		t.Skip("PostgreSQL not available for testing")
		return nil
	}

	storage, err := NewPostgreSQLStorage(config)
	if err != nil {
		t.Skipf("Failed to connect to PostgreSQL: %v", err)
		return nil
	}

	// Clean up test data
	ctx := context.Background()
	storage.db.ExecContext(ctx, "DELETE FROM translation_sessions WHERE id LIKE 'postgres-%' OR id LIKE 'update-%' OR id LIKE 'list-pg-%' OR id LIKE 'delete-%' OR id LIKE 'stats-pg-%'")
	storage.db.ExecContext(ctx, "DELETE FROM translation_cache WHERE id LIKE 'cache-pg-%' OR id LIKE 'upsert-%' OR id LIKE 'old-pg-%' OR id LIKE 'recent-pg-%'")

	return storage
}

// getPostgreSQLTestConfig returns test configuration
func getPostgreSQLTestConfig(t *testing.T) *Config {
	// Check for test database environment variables
	host := os.Getenv("POSTGRES_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	database := os.Getenv("POSTGRES_TEST_DB")
	if database == "" {
		// No test database configured
		return nil
	}

	username := os.Getenv("POSTGRES_TEST_USER")
	if username == "" {
		username = "postgres"
	}

	password := os.Getenv("POSTGRES_TEST_PASSWORD")

	return &Config{
		Type:         "postgres",
		Host:         host,
		Port:         5432,
		Database:     database,
		Username:     username,
		Password:     password,
		SSLMode:      "disable",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	}
}

// BenchmarkPostgreSQLStorage_CreateSession benchmarks session creation
func BenchmarkPostgreSQLStorage_CreateSession(b *testing.B) {
	config := getPostgreSQLTestConfig(nil)
	if config == nil {
		b.Skip("PostgreSQL not available")
	}

	storage, err := NewPostgreSQLStorage(config)
	if err != nil {
		b.Skip("Failed to connect to PostgreSQL")
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := &TranslationSession{
			ID:             "bench-pg-" + string(rune('0'+i%10)),
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

// BenchmarkPostgreSQLStorage_GetSession benchmarks retrieval
func BenchmarkPostgreSQLStorage_GetSession(b *testing.B) {
	config := getPostgreSQLTestConfig(nil)
	if config == nil {
		b.Skip("PostgreSQL not available")
	}

	storage, err := NewPostgreSQLStorage(config)
	if err != nil {
		b.Skip("Failed to connect to PostgreSQL")
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "bench-pg-get",
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
		_, _ = storage.GetSession(ctx, "bench-pg-get")
	}
}
