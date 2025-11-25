package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestTranslationSessionStructure tests the TranslationSession struct
func TestTranslationSessionStructure(t *testing.T) {
	endTime := time.Now()
	session := TranslationSession{
		ID:              "test-session-123",
		BookTitle:       "Test Book",
		InputFile:       "/input/book.epub",
		OutputFile:      "/output/book_translated.epub",
		SourceLanguage:  "en",
		TargetLanguage:  "ru",
		Provider:        "openai",
		Model:           "gpt-4",
		Status:          "completed",
		PercentComplete: 100.0,
		CurrentChapter:  10,
		TotalChapters:   10,
		ItemsCompleted:  10,
		ItemsFailed:     0,
		ItemsTotal:      10,
		StartTime:       time.Now().Add(-1 * time.Hour),
		EndTime:         &endTime,
		ErrorMessage:    "",
		CreatedAt:       time.Now().Add(-2 * time.Hour),
		UpdatedAt:       time.Now(),
	}

	assert.Equal(t, "test-session-123", session.ID)
	assert.Equal(t, "Test Book", session.BookTitle)
	assert.Equal(t, "/input/book.epub", session.InputFile)
	assert.Equal(t, "/output/book_translated.epub", session.OutputFile)
	assert.Equal(t, "en", session.SourceLanguage)
	assert.Equal(t, "ru", session.TargetLanguage)
	assert.Equal(t, "openai", session.Provider)
	assert.Equal(t, "gpt-4", session.Model)
	assert.Equal(t, "completed", session.Status)
	assert.Equal(t, 100.0, session.PercentComplete)
	assert.Equal(t, 10, session.CurrentChapter)
	assert.Equal(t, 10, session.TotalChapters)
	assert.Equal(t, 10, session.ItemsCompleted)
	assert.Equal(t, 0, session.ItemsFailed)
	assert.Equal(t, 10, session.ItemsTotal)
	assert.NotNil(t, session.EndTime)
	assert.Equal(t, "", session.ErrorMessage)
}

// TestTranslationSessionWithNilEndTime tests session with nil end time
func TestTranslationSessionWithNilEndTime(t *testing.T) {
	session := TranslationSession{
		ID:              "incomplete-session",
		Status:          "in_progress",
		PercentComplete:  50.0,
		CurrentChapter:  5,
		TotalChapters:   10,
		ItemsCompleted:  5,
		ItemsFailed:     0,
		ItemsTotal:      10,
		StartTime:       time.Now().Add(-30 * time.Minute),
		EndTime:         nil, // Still in progress
		CreatedAt:       time.Now().Add(-1 * time.Hour),
		UpdatedAt:       time.Now(),
	}

	assert.Equal(t, "incomplete-session", session.ID)
	assert.Equal(t, "in_progress", session.Status)
	assert.Equal(t, 50.0, session.PercentComplete)
	assert.Nil(t, session.EndTime)
}

// TestTranslationSessionWithError tests session with error
func TestTranslationSessionWithError(t *testing.T) {
	errorMsg := "Translation failed: API timeout"
	session := TranslationSession{
		ID:              "failed-session",
		Status:          "failed",
		PercentComplete:  25.0,
		CurrentChapter:  2,
		TotalChapters:   10,
		ItemsCompleted:  2,
		ItemsFailed:     1,
		ItemsTotal:      10,
		StartTime:       time.Now().Add(-45 * time.Minute),
		EndTime:         nil, // Failed during processing
		ErrorMessage:    errorMsg,
		CreatedAt:       time.Now().Add(-1 * time.Hour),
		UpdatedAt:       time.Now(),
	}

	assert.Equal(t, "failed-session", session.ID)
	assert.Equal(t, "failed", session.Status)
	assert.Equal(t, 25.0, session.PercentComplete)
	assert.Equal(t, errorMsg, session.ErrorMessage)
}

// TestTranslationCacheStructure tests the TranslationCache struct
func TestTranslationCacheStructure(t *testing.T) {
	cache := TranslationCache{
		ID:              "cache-123",
		SourceText:      "Hello world",
		TargetText:      "Привет мир",
		SourceLanguage:  "en",
		TargetLanguage:  "ru",
		Provider:        "openai",
		Model:           "gpt-4",
		CreatedAt:       time.Now().Add(-1 * time.Hour),
		AccessCount:     5,
		LastAccessedAt:  time.Now().Add(-10 * time.Minute),
	}

	assert.Equal(t, "cache-123", cache.ID)
	assert.Equal(t, "Hello world", cache.SourceText)
	assert.Equal(t, "Привет мир", cache.TargetText)
	assert.Equal(t, "en", cache.SourceLanguage)
	assert.Equal(t, "ru", cache.TargetLanguage)
	assert.Equal(t, "openai", cache.Provider)
	assert.Equal(t, "gpt-4", cache.Model)
	assert.Equal(t, 5, cache.AccessCount)
	assert.True(t, cache.LastAccessedAt.After(cache.CreatedAt))
}

// TestTranslationCacheZeroAccess tests cache with zero access count
func TestTranslationCacheZeroAccess(t *testing.T) {
	now := time.Now()
	cache := TranslationCache{
		ID:              "new-cache",
		SourceText:      "Test text",
		TargetText:      "Текст теста",
		SourceLanguage:  "en",
		TargetLanguage:  "ru",
		Provider:        "openai",
		Model:           "gpt-4",
		CreatedAt:       now,
		AccessCount:     0,
		LastAccessedAt:  now,
	}

	assert.Equal(t, 0, cache.AccessCount)
	assert.Equal(t, cache.CreatedAt, cache.LastAccessedAt)
}

// TestStatisticsStructure tests the Statistics struct
func TestStatisticsStructure(t *testing.T) {
	stats := Statistics{
		TotalSessions:      1000,
		CompletedSessions:  850,
		FailedSessions:     100,
		InProgressSessions: 50,
		TotalTranslations:  10000,
		CacheHitRate:       0.75,
		AverageDuration:    120.5,
	}

	assert.Equal(t, int64(1000), stats.TotalSessions)
	assert.Equal(t, int64(850), stats.CompletedSessions)
	assert.Equal(t, int64(100), stats.FailedSessions)
	assert.Equal(t, int64(50), stats.InProgressSessions)
	assert.Equal(t, int64(10000), stats.TotalTranslations)
	assert.Equal(t, 0.75, stats.CacheHitRate)
	assert.Equal(t, 120.5, stats.AverageDuration)
	
	// Test that completed + failed + in_progress equals total
	total := stats.CompletedSessions + stats.FailedSessions + stats.InProgressSessions
	assert.Equal(t, stats.TotalSessions, total)
}

// TestStatisticsEmpty tests empty statistics
func TestStatisticsEmpty(t *testing.T) {
	stats := Statistics{
		TotalSessions:      0,
		CompletedSessions:  0,
		FailedSessions:     0,
		InProgressSessions:  0,
		TotalTranslations:  0,
		CacheHitRate:       0.0,
		AverageDuration:    0.0,
	}

	assert.Equal(t, int64(0), stats.TotalSessions)
	assert.Equal(t, int64(0), stats.CompletedSessions)
	assert.Equal(t, int64(0), stats.FailedSessions)
	assert.Equal(t, int64(0), stats.InProgressSessions)
	assert.Equal(t, int64(0), stats.TotalTranslations)
	assert.Equal(t, 0.0, stats.CacheHitRate)
	assert.Equal(t, 0.0, stats.AverageDuration)
}

// TestConfigStructure tests the Config struct
func TestConfigStructure(t *testing.T) {
	config := Config{
		Type:            "sqlite",
		Host:            "localhost",
		Port:            5432,
		Database:        "translations.db",
		Username:        "user",
		Password:        "password",
		SSLMode:         "disable",
		EncryptionKey:    "encryption-key-123",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime:  5 * time.Minute,
	}

	assert.Equal(t, "sqlite", config.Type)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "translations.db", config.Database)
	assert.Equal(t, "user", config.Username)
	assert.Equal(t, "password", config.Password)
	assert.Equal(t, "disable", config.SSLMode)
	assert.Equal(t, "encryption-key-123", config.EncryptionKey)
	assert.Equal(t, 25, config.MaxOpenConns)
	assert.Equal(t, 5, config.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, config.ConnMaxLifetime)
}

// TestConfigPostgres tests PostgreSQL configuration
func TestConfigPostgres(t *testing.T) {
	config := Config{
		Type:            "postgres",
		Host:            "db.example.com",
		Port:            5432,
		Database:        "translate_db",
		Username:        "admin",
		Password:        "secure_password",
		SSLMode:         "require",
		EncryptionKey:    "", // Not used for PostgreSQL
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime:  10 * time.Minute,
	}

	assert.Equal(t, "postgres", config.Type)
	assert.Equal(t, "db.example.com", config.Host)
	assert.Equal(t, "require", config.SSLMode)
	assert.Equal(t, "", config.EncryptionKey)
	assert.Equal(t, 50, config.MaxOpenConns)
	assert.Equal(t, 10, config.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, config.ConnMaxLifetime)
}

// TestConfigRedis tests Redis configuration
func TestConfigRedis(t *testing.T) {
	config := Config{
		Type:            "redis",
		Host:            "redis.example.com",
		Port:            6379,
		Database:        "0", // Redis database number
		Username:        "",
		Password:        "redis_password",
		SSLMode:         "", // Not used for Redis
		EncryptionKey:    "", // Not used for Redis
		MaxOpenConns:    100,
		MaxIdleConns:    20,
		ConnMaxLifetime:  30 * time.Second,
	}

	assert.Equal(t, "redis", config.Type)
	assert.Equal(t, "redis.example.com", config.Host)
	assert.Equal(t, 6379, config.Port)
	assert.Equal(t, "0", config.Database)
	assert.Equal(t, "", config.Username)
	assert.Equal(t, "redis_password", config.Password)
	assert.Equal(t, 100, config.MaxOpenConns)
	assert.Equal(t, 20, config.MaxIdleConns)
	assert.Equal(t, 30*time.Second, config.ConnMaxLifetime)
}

// TestStorageInterfaceMethods tests that all interface methods are defined
func TestStorageInterfaceMethods(t *testing.T) {
	// Create a real mock storage instance
	storage := &mockStorage{}

	assert.NotNil(t, storage)
	
	// Verify all methods exist by calling them with context
	ctx := context.Background()
	
	// Test that all methods exist and can be called (should not panic)
	assert.NotPanics(t, func() {
		storage.CreateSession(ctx, &TranslationSession{ID: "test"})
	})
	assert.NotPanics(t, func() {
		storage.GetSession(ctx, "test")
	})
	assert.NotPanics(t, func() {
		storage.UpdateSession(ctx, &TranslationSession{ID: "test"})
	})
	assert.NotPanics(t, func() {
		storage.ListSessions(ctx, 10, 0)
	})
	assert.NotPanics(t, func() {
		storage.DeleteSession(ctx, "test")
	})
	assert.NotPanics(t, func() {
		storage.GetCachedTranslation(ctx, "test", "en", "ru", "openai", "gpt-4")
	})
	assert.NotPanics(t, func() {
		storage.CacheTranslation(ctx, &TranslationCache{ID: "test"})
	})
	assert.NotPanics(t, func() {
		storage.CleanupOldCache(ctx, time.Hour)
	})
	assert.NotPanics(t, func() {
		storage.GetStatistics(ctx)
	})
	assert.NotPanics(t, func() {
		storage.Ping(ctx)
	})
	assert.NotPanics(t, func() {
		storage.Close()
	})
}

// mockStorage is a minimal implementation of Storage interface for testing
type mockStorage struct{}

func (m *mockStorage) CreateSession(ctx context.Context, session *TranslationSession) error {
	return nil
}

func (m *mockStorage) GetSession(ctx context.Context, sessionID string) (*TranslationSession, error) {
	return nil, nil
}

func (m *mockStorage) UpdateSession(ctx context.Context, session *TranslationSession) error {
	return nil
}

func (m *mockStorage) ListSessions(ctx context.Context, limit, offset int) ([]*TranslationSession, error) {
	return nil, nil
}

func (m *mockStorage) DeleteSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockStorage) GetCachedTranslation(ctx context.Context, sourceText, sourceLanguage, targetLanguage, provider, model string) (*TranslationCache, error) {
	return nil, nil
}

func (m *mockStorage) CacheTranslation(ctx context.Context, cache *TranslationCache) error {
	return nil
}

func (m *mockStorage) CleanupOldCache(ctx context.Context, olderThan time.Duration) error {
	return nil
}

func (m *mockStorage) GetStatistics(ctx context.Context) (*Statistics, error) {
	return nil, nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}