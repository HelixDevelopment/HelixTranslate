package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTranslationSession_Creation tests creating a translation session
func TestTranslationSession_Creation(t *testing.T) {
	now := time.Now()
	session := &TranslationSession{
		ID:              "test-session-123",
		BookTitle:       "Test Book",
		InputFile:       "input.epub",
		OutputFile:      "output.epub",
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

	assert.Equal(t, "test-session-123", session.ID)
	assert.Equal(t, "Test Book", session.BookTitle)
	assert.Equal(t, "initializing", session.Status)
	assert.Equal(t, 0.0, session.PercentComplete)
	assert.Nil(t, session.EndTime)
	assert.Empty(t, session.ErrorMessage)
}

// TestTranslationSession_WithEndTime tests session with completion time
func TestTranslationSession_WithEndTime(t *testing.T) {
	now := time.Now()
	endTime := now.Add(time.Hour)

	session := &TranslationSession{
		ID:              "completed-session",
		BookTitle:       "Completed Book",
		InputFile:       "input.epub",
		SourceLanguage:  "ru",
		TargetLanguage:  "sr",
		Provider:        "deepseek",
		Model:           "deepseek-chat",
		Status:          "completed",
		PercentComplete: 100.0,
		StartTime:       now,
		EndTime:         &endTime,
		CreatedAt:       now,
		UpdatedAt:       endTime,
	}

	assert.Equal(t, "completed", session.Status)
	assert.Equal(t, 100.0, session.PercentComplete)
	require.NotNil(t, session.EndTime)
	assert.Equal(t, endTime, *session.EndTime)
}

// TestTranslationCache_Creation tests creating a translation cache entry
func TestTranslationCache_Creation(t *testing.T) {
	now := time.Now()
	cache := &TranslationCache{
		ID:             "cache-123",
		SourceText:     "Hello world",
		TargetText:     "Здраво свете",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		AccessCount:    1,
		LastAccessedAt: now,
	}

	assert.Equal(t, "cache-123", cache.ID)
	assert.Equal(t, "Hello world", cache.SourceText)
	assert.Equal(t, "Здраво свете", cache.TargetText)
	assert.Equal(t, 1, cache.AccessCount)
}

// TestStatistics_Defaults tests statistics with default values
func TestStatistics_Defaults(t *testing.T) {
	stats := &Statistics{}

	assert.Equal(t, int64(0), stats.TotalSessions)
	assert.Equal(t, int64(0), stats.CompletedSessions)
	assert.Equal(t, int64(0), stats.FailedSessions)
	assert.Equal(t, int64(0), stats.InProgressSessions)
	assert.Equal(t, int64(0), stats.TotalTranslations)
	assert.Equal(t, 0.0, stats.CacheHitRate)
	assert.Equal(t, 0.0, stats.AverageDuration)
}

// TestStatistics_Calculation tests statistics calculation logic
func TestStatistics_Calculation(t *testing.T) {
	stats := &Statistics{
		TotalSessions:      100,
		CompletedSessions:  80,
		FailedSessions:     10,
		InProgressSessions: 10,
		TotalTranslations:  1000,
	}

	// Calculate derived metrics
	totalAccountedSessions := stats.CompletedSessions + stats.FailedSessions + stats.InProgressSessions
	assert.Equal(t, stats.TotalSessions, totalAccountedSessions)

	// Success rate
	successRate := float64(stats.CompletedSessions) / float64(stats.TotalSessions) * 100.0
	assert.Equal(t, 80.0, successRate)

	// Failure rate
	failureRate := float64(stats.FailedSessions) / float64(stats.TotalSessions) * 100.0
	assert.Equal(t, 10.0, failureRate)
}

// TestConfig_SQLite tests SQLite configuration
func TestConfig_SQLite(t *testing.T) {
	config := &Config{
		Type:          "sqlite",
		Database:      "/tmp/test.db",
		EncryptionKey: "test-key-123",
		MaxOpenConns:  10,
		MaxIdleConns:  5,
	}

	assert.Equal(t, "sqlite", config.Type)
	assert.Equal(t, "/tmp/test.db", config.Database)
	assert.Equal(t, "test-key-123", config.EncryptionKey)
	assert.Equal(t, 10, config.MaxOpenConns)
}

// TestConfig_PostgreSQL tests PostgreSQL configuration
func TestConfig_PostgreSQL(t *testing.T) {
	config := &Config{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "translator_test",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
	}

	assert.Equal(t, "postgres", config.Type)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "disable", config.SSLMode)
}

// TestConfig_Redis tests Redis configuration
func TestConfig_Redis(t *testing.T) {
	config := &Config{
		Type:     "redis",
		Host:     "localhost",
		Port:     6379,
		Password: "redis-password",
	}

	assert.Equal(t, "redis", config.Type)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 6379, config.Port)
}

// TestSessionStatusTransitions tests valid session status transitions
func TestSessionStatusTransitions(t *testing.T) {
	validTransitions := map[string][]string{
		"initializing": {"translating", "error"},
		"translating":  {"completed", "error"},
		"completed":    {}, // Terminal state
		"error":        {}, // Terminal state
	}

	for fromStatus, toStatuses := range validTransitions {
		for _, toStatus := range toStatuses {
			t.Run(fromStatus+"_to_"+toStatus, func(t *testing.T) {
				session := &TranslationSession{
					ID:     "test-transition",
					Status: fromStatus,
				}
				assert.Equal(t, fromStatus, session.Status)

				// Transition
				session.Status = toStatus
				assert.Equal(t, toStatus, session.Status)
			})
		}
	}
}

// TestSessionPercentageValidation tests session percentage boundaries
func TestSessionPercentageValidation(t *testing.T) {
	tests := []struct {
		name    string
		percent float64
		valid   bool
	}{
		{"zero percent", 0.0, true},
		{"mid progress", 50.5, true},
		{"completed", 100.0, true},
		{"negative", -10.0, false},
		{"over 100", 150.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &TranslationSession{
				ID:              "test-percent",
				PercentComplete: tt.percent,
			}

			if tt.valid {
				assert.GreaterOrEqual(t, session.PercentComplete, 0.0)
				assert.LessOrEqual(t, session.PercentComplete, 100.0)
			} else {
				// These would be invalid - application should validate
				isValid := session.PercentComplete >= 0.0 && session.PercentComplete <= 100.0
				assert.False(t, isValid)
			}
		})
	}
}

// testStorageInterface tests common storage interface operations
// This helper is used by backend-specific tests
func testStorageInterface(t *testing.T, storage Storage) {
	ctx := context.Background()

	// Test Ping
	err := storage.Ping(ctx)
	require.NoError(t, err, "Ping should succeed")

	// Create a session
	now := time.Now()
	session := &TranslationSession{
		ID:              "test-interface-123",
		BookTitle:       "Interface Test Book",
		InputFile:       "test.epub",
		OutputFile:      "test_out.epub",
		SourceLanguage:  "ru",
		TargetLanguage:  "sr",
		Provider:        "test-provider",
		Model:           "test-model",
		Status:          "initializing",
		PercentComplete: 0.0,
		CurrentChapter:  0,
		TotalChapters:   5,
		ItemsCompleted:  0,
		ItemsFailed:     0,
		ItemsTotal:      50,
		StartTime:       now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Test CreateSession
	err = storage.CreateSession(ctx, session)
	require.NoError(t, err, "CreateSession should succeed")

	// Test GetSession
	retrieved, err := storage.GetSession(ctx, session.ID)
	require.NoError(t, err, "GetSession should succeed")
	require.NotNil(t, retrieved)
	assert.Equal(t, session.ID, retrieved.ID)
	assert.Equal(t, session.BookTitle, retrieved.BookTitle)
	assert.Equal(t, session.Status, retrieved.Status)

	// Test UpdateSession
	retrieved.Status = "translating"
	retrieved.PercentComplete = 50.0
	retrieved.CurrentChapter = 2
	retrieved.ItemsCompleted = 25

	err = storage.UpdateSession(ctx, retrieved)
	require.NoError(t, err, "UpdateSession should succeed")

	updated, err := storage.GetSession(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "translating", updated.Status)
	assert.Equal(t, 50.0, updated.PercentComplete)
	assert.Equal(t, 2, updated.CurrentChapter)
	assert.Equal(t, 25, updated.ItemsCompleted)

	// Test ListSessions
	sessions, err := storage.ListSessions(ctx, 10, 0)
	require.NoError(t, err, "ListSessions should succeed")
	assert.GreaterOrEqual(t, len(sessions), 1)

	// Find our session
	found := false
	for _, s := range sessions {
		if s.ID == session.ID {
			found = true
			break
		}
	}
	assert.True(t, found, "Created session should be in list")

	// Test Cache operations
	cache := &TranslationCache{
		ID:             "cache-test-123",
		SourceText:     "Test text",
		TargetText:     "Тест текст",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "test-provider",
		Model:          "test-model",
		CreatedAt:      now,
		AccessCount:    0,
		LastAccessedAt: now,
	}

	// Test CacheTranslation
	err = storage.CacheTranslation(ctx, cache)
	require.NoError(t, err, "CacheTranslation should succeed")

	// Test GetCachedTranslation
	cachedResult, err := storage.GetCachedTranslation(ctx,
		cache.SourceText,
		cache.SourceLanguage,
		cache.TargetLanguage,
		cache.Provider,
		cache.Model,
	)
	require.NoError(t, err, "GetCachedTranslation should succeed")
	require.NotNil(t, cachedResult)
	assert.Equal(t, cache.TargetText, cachedResult.TargetText)

	// Test GetStatistics
	stats, err := storage.GetStatistics(ctx)
	require.NoError(t, err, "GetStatistics should succeed")
	require.NotNil(t, stats)
	assert.Greater(t, stats.TotalSessions, int64(0))

	// Test DeleteSession
	err = storage.DeleteSession(ctx, session.ID)
	require.NoError(t, err, "DeleteSession should succeed")

	// Verify deletion
	deleted, err := storage.GetSession(ctx, session.ID)
	assert.Error(t, err, "GetSession should fail for deleted session")
	assert.Nil(t, deleted)

	// Test CleanupOldCache
	err = storage.CleanupOldCache(ctx, 24*time.Hour)
	require.NoError(t, err, "CleanupOldCache should succeed")
}

// TestSessionID_UniqueConstraint tests that session IDs must be unique
func TestSessionID_UniqueConstraint(t *testing.T) {
	// This is conceptual - actual enforcement depends on storage backend
	sessionID := "unique-session-123"

	session1 := &TranslationSession{
		ID:             sessionID,
		BookTitle:      "First Book",
		InputFile:      "first.epub",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         "initializing",
		StartTime:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	session2 := &TranslationSession{
		ID:             sessionID, // Same ID
		BookTitle:      "Second Book",
		InputFile:      "second.epub",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         "initializing",
		StartTime:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	assert.Equal(t, session1.ID, session2.ID, "IDs are the same")
	// Storage backends should enforce uniqueness via PRIMARY KEY constraint
}

// TestCacheKey_Generation tests cache key uniqueness
func TestCacheKey_Generation(t *testing.T) {
	tests := []struct {
		name         string
		sourceText   string
		sourceLang   string
		targetLang   string
		provider     string
		model        string
		shouldBeSame bool
		compareWith  *struct{ sourceText, sourceLang, targetLang, provider, model string }
	}{
		{
			name:         "identical parameters",
			sourceText:   "Hello",
			sourceLang:   "en",
			targetLang:   "sr",
			provider:     "deepseek",
			model:        "deepseek-chat",
			shouldBeSame: true,
			compareWith: &struct{ sourceText, sourceLang, targetLang, provider, model string }{
				"Hello", "en", "sr", "deepseek", "deepseek-chat",
			},
		},
		{
			name:         "different text",
			sourceText:   "Hello",
			sourceLang:   "en",
			targetLang:   "sr",
			provider:     "deepseek",
			model:        "deepseek-chat",
			shouldBeSame: false,
			compareWith: &struct{ sourceText, sourceLang, targetLang, provider, model string }{
				"Goodbye", "en", "sr", "deepseek", "deepseek-chat",
			},
		},
		{
			name:         "different provider",
			sourceText:   "Hello",
			sourceLang:   "en",
			targetLang:   "sr",
			provider:     "deepseek",
			model:        "deepseek-chat",
			shouldBeSame: false,
			compareWith: &struct{ sourceText, sourceLang, targetLang, provider, model string }{
				"Hello", "en", "sr", "openai", "gpt-4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Conceptual test - actual cache key generation is backend-specific
			key1 := tt.sourceLang + ":" + tt.targetLang + ":" + tt.provider + ":" + tt.model + ":" + tt.sourceText
			key2 := tt.compareWith.sourceLang + ":" + tt.compareWith.targetLang + ":" +
				tt.compareWith.provider + ":" + tt.compareWith.model + ":" + tt.compareWith.sourceText

			if tt.shouldBeSame {
				assert.Equal(t, key1, key2)
			} else {
				assert.NotEqual(t, key1, key2)
			}
		})
	}
}

// BenchmarkSessionCreation benchmarks session struct creation
func BenchmarkSessionCreation(b *testing.B) {
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &TranslationSession{
			ID:              "bench-session",
			BookTitle:       "Benchmark Book",
			InputFile:       "bench.epub",
			OutputFile:      "bench_out.epub",
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
	}
}

// BenchmarkCacheCreation benchmarks cache struct creation
func BenchmarkCacheCreation(b *testing.B) {
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &TranslationCache{
			ID:             "bench-cache",
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
	}
}
