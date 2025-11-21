package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRedisStorage tests Redis storage creation
func TestNewRedisStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis test - requires Redis server")
	}

	config := getRedisTestConfig(t)
	if config == nil {
		t.Skip("Redis not available for testing")
	}

	storage, err := NewRedisStorage(config, 24*time.Hour)
	require.NoError(t, err)
	require.NotNil(t, storage)
	defer storage.Close()

	// Verify connection
	err = storage.Ping(context.Background())
	assert.NoError(t, err)
}

// TestRedisStorage_CreateSession tests creating a session
func TestRedisStorage_CreateSession(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:              "redis-test-123",
		BookTitle:       "Redis Test Book",
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

// TestRedisStorage_UpdateSession tests updating a session
func TestRedisStorage_UpdateSession(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "update-redis-test",
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

	err = storage.UpdateSession(ctx, session)
	require.NoError(t, err)

	// Verify
	updated, err := storage.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, 100.0, updated.PercentComplete)
}

// TestRedisStorage_GetSessionNotFound tests non-existent session
func TestRedisStorage_GetSessionNotFound(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()

	session, err := storage.GetSession(ctx, "non-existent-redis")
	assert.Error(t, err)
	assert.Nil(t, session)
}

// TestRedisStorage_ListSessions tests listing sessions
func TestRedisStorage_ListSessions(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := &TranslationSession{
			ID:             "list-redis-" + string(rune('A'+i)),
			BookTitle:      "List Test",
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

	// List sessions
	sessions, err := storage.ListSessions(ctx, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sessions), 5)
}

// TestRedisStorage_DeleteSession tests session deletion
func TestRedisStorage_DeleteSession(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "delete-redis-test",
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

// TestRedisStorage_CacheTranslation tests caching translations
func TestRedisStorage_CacheTranslation(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	cache := &TranslationCache{
		ID:             "cache-redis-123",
		SourceText:     "Redis test",
		TargetText:     "Redis тест",
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

// TestRedisStorage_CacheMiss tests cache miss behavior
func TestRedisStorage_CacheMiss(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
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

// TestRedisStorage_TTL tests automatic expiration
func TestRedisStorage_TTL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping TTL test in short mode")
	}

	config := getRedisTestConfig(t)
	if config == nil {
		t.Skip("Redis not available")
	}

	// Create storage with short TTL
	storage, err := NewRedisStorage(config, 2*time.Second)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "ttl-test-redis",
		BookTitle:      "TTL Test",
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

	err = storage.CreateSession(ctx, session)
	require.NoError(t, err)

	// Verify exists
	retrieved, err := storage.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Wait for TTL expiration
	time.Sleep(3 * time.Second)

	// Should be expired
	expired, err := storage.GetSession(ctx, session.ID)
	assert.Error(t, err)
	assert.Nil(t, expired)
}

// TestRedisStorage_CleanupOldCache tests cleanup (no-op for Redis)
func TestRedisStorage_CleanupOldCache(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()

	// Cleanup should succeed but is a no-op (Redis handles TTL)
	err := storage.CleanupOldCache(ctx, 24*time.Hour)
	assert.NoError(t, err)
}

// TestRedisStorage_GetStatistics tests statistics retrieval
func TestRedisStorage_GetStatistics(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Create sessions with different statuses
	statuses := []string{"completed", "error", "translating"}
	for i, status := range statuses {
		session := &TranslationSession{
			ID:             "stats-redis-" + string(rune('1'+i)),
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
}

// TestRedisStorage_Ping tests connection check
func TestRedisStorage_Ping(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	err := storage.Ping(context.Background())
	assert.NoError(t, err)
}

// TestRedisStorage_Close tests closing connection
func TestRedisStorage_Close(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}

	err := storage.Close()
	assert.NoError(t, err)

	// Operations after close should fail
	ctx := context.Background()
	err = storage.Ping(ctx)
	assert.Error(t, err)
}

// TestRedisStorage_InterfaceCompliance tests full interface
func TestRedisStorage_InterfaceCompliance(t *testing.T) {
	storage := setupRedisTest(t)
	if storage == nil {
		return
	}
	defer storage.Close()

	testStorageInterface(t, storage)
}

// setupRedisTest creates Redis storage for testing
func setupRedisTest(t *testing.T) *RedisStorage {
	if testing.Short() {
		t.Skip("Skipping Redis test - requires Redis server")
		return nil
	}

	config := getRedisTestConfig(t)
	if config == nil {
		t.Skip("Redis not available for testing")
		return nil
	}

	storage, err := NewRedisStorage(config, 24*time.Hour)
	if err != nil {
		t.Skipf("Failed to connect to Redis: %v", err)
		return nil
	}

	// Clean up test data
	ctx := context.Background()
	keys, _ := storage.client.Keys(ctx, "session:*").Result()
	if len(keys) > 0 {
		storage.client.Del(ctx, keys...)
	}
	keys, _ = storage.client.Keys(ctx, "cache:*").Result()
	if len(keys) > 0 {
		storage.client.Del(ctx, keys...)
	}

	return storage
}

// getRedisTestConfig returns test configuration
func getRedisTestConfig(t *testing.T) *Config {
	host := os.Getenv("REDIS_TEST_HOST")
	if host == "" {
		host = "localhost"
	}

	// Check if Redis is disabled for tests
	if os.Getenv("SKIP_REDIS_TESTS") == "1" {
		return nil
	}

	return &Config{
		Type:     "redis",
		Host:     host,
		Port:     6379,
		Password: os.Getenv("REDIS_TEST_PASSWORD"),
	}
}

// BenchmarkRedisStorage_CreateSession benchmarks session creation
func BenchmarkRedisStorage_CreateSession(b *testing.B) {
	config := getRedisTestConfig(nil)
	if config == nil {
		b.Skip("Redis not available")
	}

	storage, err := NewRedisStorage(config, 24*time.Hour)
	if err != nil {
		b.Skip("Failed to connect to Redis")
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := &TranslationSession{
			ID:             "bench-redis-" + string(rune('0'+i%10)),
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

// BenchmarkRedisStorage_GetSession benchmarks retrieval
func BenchmarkRedisStorage_GetSession(b *testing.B) {
	config := getRedisTestConfig(nil)
	if config == nil {
		b.Skip("Redis not available")
	}

	storage, err := NewRedisStorage(config, 24*time.Hour)
	if err != nil {
		b.Skip("Failed to connect to Redis")
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	session := &TranslationSession{
		ID:             "bench-redis-get",
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
		_, _ = storage.GetSession(ctx, "bench-redis-get")
	}
}

// BenchmarkRedisStorage_CacheTranslation benchmarks cache operations
func BenchmarkRedisStorage_CacheTranslation(b *testing.B) {
	config := getRedisTestConfig(nil)
	if config == nil {
		b.Skip("Redis not available")
	}

	storage, err := NewRedisStorage(config, 24*time.Hour)
	if err != nil {
		b.Skip("Failed to connect to Redis")
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
