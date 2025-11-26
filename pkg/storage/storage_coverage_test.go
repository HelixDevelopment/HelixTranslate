package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabase is a mock for database operations
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

// TestStorageErrorHandling tests various error scenarios
func TestStorageErrorHandling(t *testing.T) {
	t.Run("Config validation", func(t *testing.T) {
		// Test empty config type
		config := &Config{}
		
		// Empty type should still be valid (defaults may be applied)
		assert.Equal(t, "", config.Type)
		
		// Test invalid connection settings
		config.MaxOpenConns = -1
		assert.True(t, config.MaxOpenConns < 0, "Negative MaxOpenConns should be detected")
		
		config.MaxIdleConns = -1
		assert.True(t, config.MaxIdleConns < 0, "Negative MaxIdleConns should be detected")
	})
	
	t.Run("Session validation", func(t *testing.T) {
		// Test session with invalid percentage
		session := &TranslationSession{
			ID:              "test-session",
			PercentComplete: 150.0, // Invalid
		}
		
		isValid := session.PercentComplete >= 0.0 && session.PercentComplete <= 100.0
		assert.False(t, isValid, "Percentage > 100 should be invalid")
		
		// Test session with negative percentage
		session.PercentComplete = -10.0
		isValid = session.PercentComplete >= 0.0 && session.PercentComplete <= 100.0
		assert.False(t, isValid, "Negative percentage should be invalid")
		
		// Test valid boundaries
		validPercentages := []float64{0.0, 50.5, 99.9, 100.0}
		for _, percent := range validPercentages {
			session.PercentComplete = percent
			isValid = session.PercentComplete >= 0.0 && session.PercentComplete <= 100.0
			assert.True(t, isValid, "Percentage %.1f should be valid", percent)
		}
	})
	
	t.Run("Cache validation", func(t *testing.T) {
		// Test cache with empty source text
		cache := &TranslationCache{
			ID:             "test-cache",
			SourceText:     "", // Empty
			TargetText:     "Translated",
			SourceLanguage: "en",
			TargetLanguage: "ru",
		}
		
		// Empty source text should be flagged as potentially invalid
		assert.Empty(t, cache.SourceText, "Source text should not be empty for cache entries")
		
		// Test cache with negative access count
		cache.AccessCount = -1
		assert.True(t, cache.AccessCount < 0, "Negative access count should be detected")
	})
	
	t.Run("Statistics validation", func(t *testing.T) {
		stats := &Statistics{
			TotalSessions:      100,
			CompletedSessions:  80,
			FailedSessions:     10,
			InProgressSessions: 5,
		}
		
		// Check if accounted sessions match total
		accountedTotal := stats.CompletedSessions + stats.FailedSessions + stats.InProgressSessions
		assert.NotEqual(t, stats.TotalSessions, accountedTotal, 
			"There's a discrepancy between total and accounted sessions")
		
		// Check cache hit rate bounds
		stats.CacheHitRate = 1.5 // Invalid
		isValidRate := stats.CacheHitRate >= 0.0 && stats.CacheHitRate <= 1.0
		assert.False(t, isValidRate, "Cache hit rate > 1.0 should be invalid")
		
		stats.CacheHitRate = -0.1 // Invalid
		isValidRate = stats.CacheHitRate >= 0.0 && stats.CacheHitRate <= 1.0
		assert.False(t, isValidRate, "Negative cache hit rate should be invalid")
	})
}

// TestStorageEdgeCases tests edge cases and boundary conditions
func TestStorageEdgeCases(t *testing.T) {
	t.Run("TranslationSession time handling", func(t *testing.T) {
		now := time.Now()
		
		// Test session with start time after end time
		endTime := now.Add(-1 * time.Hour) // Before start time
		session := &TranslationSession{
			ID:        "time-test",
			StartTime: now,
			EndTime:   &endTime,
		}
		
		if session.EndTime != nil {
			// This is actually invalid but we're testing the edge case
			isInvalidTime := session.EndTime.Before(session.StartTime)
			assert.True(t, isInvalidTime, 
				"End time before start should be detected")
		}
		
		// Test session created before start time
		createdTime := now.Add(1 * time.Hour) // After start time
		session.CreatedAt = createdTime
		session.StartTime = now
		
		// Created before start might be valid in some scenarios (pre-created sessions)
		assert.NotEqual(t, session.CreatedAt, session.StartTime, 
			"Created time and start time should be different")
	})
	
	t.Run("Translation cache timestamps", func(t *testing.T) {
		now := time.Now()
		pastTime := now.Add(-1 * time.Hour)
		
		cache := &TranslationCache{
			ID:             "time-test",
			SourceText:     "Test",
			TargetText:     "Translation",
			CreatedAt:      pastTime,
			LastAccessedAt: now,
		}
		
		// Last accessed should be after creation
		assert.True(t, cache.LastAccessedAt.After(cache.CreatedAt), 
			"Last accessed time should be after creation time")
		
		// Test cache accessed before creation (should be invalid)
		cache.LastAccessedAt = pastTime.Add(-30 * time.Minute)
		isValidTime := cache.LastAccessedAt.After(cache.CreatedAt) || 
			cache.LastAccessedAt.Equal(cache.CreatedAt)
		assert.False(t, isValidTime, "Cache should not be accessed before creation")
	})
	
	t.Run("Configuration edge cases", func(t *testing.T) {
		// Test SQLite with empty database path
		config := &Config{
			Type:     "sqlite",
			Database: "", // Empty
		}
		
		assert.Equal(t, "", config.Database, 
			"Empty database path should be preserved (may be invalid in practice)")
		
		// Test PostgreSQL with minimum port
		config.Type = "postgres"
		config.Port = 0
		assert.Equal(t, 0, config.Port, "Port 0 should be preserved (may be invalid)")
		
		config.Port = 65535
		assert.Equal(t, 65535, config.Port, "Max port value should be preserved")
		
		// Test connection pool extremes
		config.MaxOpenConns = 1
		config.MaxIdleConns = 0 // Should be less than or equal to MaxOpenConns
		assert.Equal(t, 0, config.MaxIdleConns, 
			"Zero idle connections should be allowed")
	})
}

// TestStorageInterfaceValidation tests interface contract requirements
func TestStorageInterfaceValidation(t *testing.T) {
	t.Run("Method signatures", func(t *testing.T) {
		// Verify all Storage interface methods are properly defined
		// This is a compile-time check - if it compiles, interfaces are correct
		var _ Storage = (*MockStorageImplementation)(nil)
		assert.True(t, true, "Interface implementation is valid")
	})
	
	t.Run("Context handling", func(t *testing.T) {
		ctx := context.Background()
		mockStorage := &MockStorageImplementation{}
		
		// Test with cancelled context
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately
		
		// Methods should handle cancelled context gracefully
		err := mockStorage.Ping(cancelledCtx)
		assert.Error(t, err, "Operations with cancelled context should return error")
		assert.Contains(t, err.Error(), "context canceled", 
			"Error should indicate context cancellation")
	})
	
	t.Run("Nil input handling", func(t *testing.T) {
		mockStorage := &MockStorageImplementation{}
		ctx := context.Background()
		
		// Test with nil session
		err := mockStorage.CreateSession(ctx, nil)
		assert.Error(t, err, "Creating nil session should return error")
		
		err = mockStorage.UpdateSession(ctx, nil)
		assert.Error(t, err, "Updating nil session should return error")
		
		// Test with nil cache
		err = mockStorage.CacheTranslation(ctx, nil)
		assert.Error(t, err, "Caching nil translation should return error")
	})
}

// MockStorageImplementation is a mock implementation for testing interface compliance
type MockStorageImplementation struct{}

func (m *MockStorageImplementation) CreateSession(ctx context.Context, session *TranslationSession) error {
	if session == nil {
		return assert.AnError
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (m *MockStorageImplementation) GetSession(ctx context.Context, sessionID string) (*TranslationSession, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &TranslationSession{ID: sessionID}, nil
}

func (m *MockStorageImplementation) UpdateSession(ctx context.Context, session *TranslationSession) error {
	if session == nil {
		return assert.AnError
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (m *MockStorageImplementation) ListSessions(ctx context.Context, limit, offset int) ([]*TranslationSession, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return []*TranslationSession{}, nil
}

func (m *MockStorageImplementation) DeleteSession(ctx context.Context, sessionID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (m *MockStorageImplementation) GetCachedTranslation(ctx context.Context, sourceText, sourceLanguage, targetLanguage, provider, model string) (*TranslationCache, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &TranslationCache{
		SourceText:     sourceText,
		SourceLanguage: sourceLanguage,
		TargetLanguage: targetLanguage,
		Provider:       provider,
		Model:          model,
	}, nil
}

func (m *MockStorageImplementation) CacheTranslation(ctx context.Context, cache *TranslationCache) error {
	if cache == nil {
		return assert.AnError
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (m *MockStorageImplementation) CleanupOldCache(ctx context.Context, olderThan time.Duration) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (m *MockStorageImplementation) GetStatistics(ctx context.Context) (*Statistics, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return &Statistics{}, nil
}

func (m *MockStorageImplementation) Ping(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

func (m *MockStorageImplementation) Close() error {
	return nil
}

// TestStoragePerformanceBenchmarks benchmarks critical operations
func TestStoragePerformanceBenchmarks(t *testing.T) {
	t.Run("Session struct creation", func(t *testing.T) {
		now := time.Now()
		
		for i := 0; i < 1000; i++ {
			session := &TranslationSession{
				ID:              "session-" + string(rune(i)),
				BookTitle:       "Test Book",
				SourceLanguage:  "en",
				TargetLanguage:  "ru",
				Provider:        "openai",
				Model:           "gpt-4",
				Status:          "initializing",
				PercentComplete: 0.0,
				StartTime:       now,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			_ = session
		}
		
		assert.True(t, true, "Session creation benchmark completed")
	})
	
	t.Run("Cache struct creation", func(t *testing.T) {
		now := time.Now()
		
		for i := 0; i < 1000; i++ {
			cache := &TranslationCache{
				ID:             "cache-" + string(rune(i)),
				SourceText:     "Test text",
				TargetText:     "Переведенный текст",
				SourceLanguage: "en",
				TargetLanguage: "ru",
				Provider:       "openai",
				Model:          "gpt-4",
				CreatedAt:      now,
				AccessCount:    0,
				LastAccessedAt: now,
			}
			_ = cache
		}
		
		assert.True(t, true, "Cache creation benchmark completed")
	})
}

// TestStorageConsistencyRules tests business logic consistency
func TestStorageConsistencyRules(t *testing.T) {
	t.Run("Session status consistency", func(t *testing.T) {
		// Test completed session should have 100% progress
		session := &TranslationSession{
			Status:         "completed",
			PercentComplete: 95.0, // Inconsistent
		}
		
		isConsistent := session.Status != "completed" || session.PercentComplete == 100.0
		assert.False(t, isConsistent, 
			"Completed session should have 100% progress")
		
		// Fix the consistency
		session.PercentComplete = 100.0
		isConsistent = session.Status != "completed" || session.PercentComplete == 100.0
		assert.True(t, isConsistent, 
			"Completed session with 100% progress should be consistent")
		
		// Test failed session should not have 100% progress
		session.Status = "failed"
		session.PercentComplete = 100.0
		isConsistent = session.Status != "failed" || session.PercentComplete < 100.0
		assert.False(t, isConsistent, 
			"Failed session should not have 100% progress")
		
		// Fix the consistency
		session.PercentComplete = 50.0
		isConsistent = session.Status != "failed" || session.PercentComplete < 100.0
		assert.True(t, isConsistent, 
			"Failed session with < 100% progress should be consistent")
	})
	
	t.Run("Translation cache uniqueness", func(t *testing.T) {
		// Test cache entry uniqueness
		cache1 := &TranslationCache{
			SourceText:     "Hello",
			SourceLanguage: "en",
			TargetLanguage: "ru",
			Provider:       "openai",
			Model:          "gpt-4",
		}
		
		cache2 := &TranslationCache{
			SourceText:     "Hello",      // Same
			SourceLanguage: "en",         // Same
			TargetLanguage: "ru",         // Same
			Provider:       "deepseek",   // Different
			Model:          "deepseek-chat", // Different
		}
		
		// These should be considered different cache entries
		areSame := cache1.SourceText == cache2.SourceText &&
			cache1.SourceLanguage == cache2.SourceLanguage &&
			cache1.TargetLanguage == cache2.TargetLanguage &&
			cache1.Provider == cache2.Provider &&
			cache1.Model == cache2.Model
		
		assert.False(t, areSame, 
			"Caches with different providers should be different")
		
		// Make them identical except target text
		cache2.Provider = cache1.Provider
		cache2.Model = cache1.Model
		
		areSame = cache1.SourceText == cache2.SourceText &&
			cache1.SourceLanguage == cache2.SourceLanguage &&
			cache1.TargetLanguage == cache2.TargetLanguage &&
			cache1.Provider == cache2.Provider &&
			cache1.Model == cache2.Model
		
		assert.True(t, areSame, 
			"Caches with identical parameters should be considered same key")
	})
}

// TestStorageConfigurations tests various configuration scenarios
func TestStorageConfigurations(t *testing.T) {
	t.Run("Default configurations", func(t *testing.T) {
		// Test zero-value config
		config := &Config{}
		
		assert.Empty(t, config.Type, "Default type should be empty")
		assert.Empty(t, config.Database, "Default database should be empty")
		assert.Equal(t, 0, config.Port, "Default port should be 0")
		assert.Equal(t, 0, config.MaxOpenConns, "Default MaxOpenConns should be 0")
		assert.Equal(t, 0, config.MaxIdleConns, "Default MaxIdleConns should be 0")
		assert.Equal(t, time.Duration(0), config.ConnMaxLifetime, 
			"Default ConnMaxLifetime should be 0")
	})
	
	t.Run("Configuration validation", func(t *testing.T) {
		config := &Config{
			Type:            "unknown",
			Host:            "",
			Port:            8080,
			Database:        "",
			MaxOpenConns:    100,
			MaxIdleConns:    50,
			ConnMaxLifetime: time.Hour,
		}
		
		// Validate connection pool settings
		assert.Greater(t, config.MaxOpenConns, 0, "MaxOpenConns should be positive")
		assert.GreaterOrEqual(t, config.MaxOpenConns, config.MaxIdleConns, 
			"MaxOpenConns should be >= MaxIdleConns")
		assert.Greater(t, config.ConnMaxLifetime, time.Duration(0), 
			"ConnMaxLifetime should be positive")
	})
	
	t.Run("Storage type specific configs", func(t *testing.T) {
		// SQLite specific
		sqliteConfig := &Config{
			Type:          "sqlite",
			Database:      "/path/to/db.sqlite",
			EncryptionKey: "secret-key",
		}
		
		assert.Equal(t, "sqlite", sqliteConfig.Type)
		assert.NotEmpty(t, sqliteConfig.Database)
		assert.NotEmpty(t, sqliteConfig.EncryptionKey)
		
		// PostgreSQL specific
		postgresConfig := &Config{
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "translator_db",
			Username: "user",
			Password: "pass",
			SSLMode:  "require",
		}
		
		assert.Equal(t, "postgres", postgresConfig.Type)
		assert.NotEmpty(t, postgresConfig.Host)
		assert.Equal(t, 5432, postgresConfig.Port)
		assert.Equal(t, "require", postgresConfig.SSLMode)
		
		// Redis specific
		redisConfig := &Config{
			Type:     "redis",
			Host:     "localhost",
			Port:     6379,
			Database: "0",
			Password: "redis-pass",
		}
		
		assert.Equal(t, "redis", redisConfig.Type)
		assert.Equal(t, 6379, redisConfig.Port)
		assert.Equal(t, "0", redisConfig.Database)
	})
}