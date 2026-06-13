package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStorage implements caching using Redis
type RedisStorage struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisStorage creates a new Redis storage
func NewRedisStorage(config *Config, ttl time.Duration) (*RedisStorage, error) {
	// Validate configuration
	if config.Host == "" {
		return nil, fmt.Errorf("redis host is required")
	}
	if config.Port <= 0 || config.Port > 65535 {
		return nil, fmt.Errorf("redis port must be between 1 and 65535")
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.Password,
		DB:       0,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStorage{
		client: client,
		ttl:    ttl,
	}, nil
}

// CreateSession creates a new translation session in Redis
func (r *RedisStorage) CreateSession(ctx context.Context, session *TranslationSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", session.ID)
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

// GetSession retrieves a session by ID from Redis
func (r *RedisStorage) GetSession(ctx context.Context, sessionID string) (*TranslationSession, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	if err != nil {
		return nil, err
	}

	session := &TranslationSession{}
	if err := json.Unmarshal(data, session); err != nil {
		return nil, err
	}

	return session, nil
}

// UpdateSession updates an existing session in Redis
func (r *RedisStorage) UpdateSession(ctx context.Context, session *TranslationSession) error {
	session.UpdatedAt = time.Now()
	return r.CreateSession(ctx, session) // Redis SET overwrites
}

// ListSessions lists translation sessions from Redis with pagination
func (r *RedisStorage) ListSessions(ctx context.Context, limit, offset int) ([]*TranslationSession, error) {
	pattern := "session:*"
	var cursor uint64
	var sessions []*TranslationSession
	count := 0

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			if count < offset {
				count++
				continue
			}
			if len(sessions) >= limit {
				return sessions, nil
			}

			data, err := r.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			session := &TranslationSession{}
			if err := json.Unmarshal(data, session); err != nil {
				continue
			}

			sessions = append(sessions, session)
			count++
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return sessions, nil
}

// DeleteSession deletes a session from Redis
func (r *RedisStorage) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return r.client.Del(ctx, key).Err()
}

// GetCachedTranslation retrieves a cached translation from Redis
func (r *RedisStorage) GetCachedTranslation(ctx context.Context, sourceText, sourceLanguage, targetLanguage, provider, model string) (*TranslationCache, error) {
	key := r.makeCacheKey(sourceText, sourceLanguage, targetLanguage, provider, model)
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	cache := &TranslationCache{}
	if err := json.Unmarshal(data, cache); err != nil {
		return nil, err
	}

	// Update access count and last accessed time
	cache.AccessCount++
	cache.LastAccessedAt = time.Now()
	_ = r.CacheTranslation(ctx, cache) // Update in background

	return cache, nil
}

// CacheTranslation caches a translation in Redis
func (r *RedisStorage) CacheTranslation(ctx context.Context, cache *TranslationCache) error {
	key := r.makeCacheKey(cache.SourceText, cache.SourceLanguage, cache.TargetLanguage, cache.Provider, cache.Model)
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}

// CleanupOldCache removes cache entries older than the specified duration
// Note: Redis handles TTL automatically, so this is a no-op
func (r *RedisStorage) CleanupOldCache(ctx context.Context, olderThan time.Duration) error {
	// Redis handles expiration automatically via TTL
	return nil
}

// GetStatistics returns translation statistics from Redis
func (r *RedisStorage) GetStatistics(ctx context.Context) (*Statistics, error) {
	stats := &Statistics{}

	// Count sessions by status
	pattern := "session:*"
	var cursor uint64
	// durationSessions counts ONLY completed sessions that carry an EndTime —
	// the divisor for the average duration. Using stats.CompletedSessions here
	// would wrongly dilute the average with completed-but-unfinished sessions
	// that contribute no duration (matching the SQL "... WHERE end_time IS NOT
	// NULL" semantics of the SQLite/PostgreSQL backends).
	var durationSessions int64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			data, err := r.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			session := &TranslationSession{}
			if err := json.Unmarshal(data, session); err != nil {
				continue
			}

			stats.TotalSessions++
			switch session.Status {
			case "completed":
				stats.CompletedSessions++
			case "error":
				stats.FailedSessions++
			case "initializing", "translating":
				stats.InProgressSessions++
			}

			// Calculate average duration for completed sessions that finished.
			if session.Status == "completed" && session.EndTime != nil {
				duration := session.EndTime.Sub(session.StartTime).Seconds()
				stats.AverageDuration, durationSessions = accumulateAvgDuration(
					stats.AverageDuration, durationSessions, duration)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	// Count cache entries
	cachePattern := "cache:*"
	cursor = 0
	var totalAccess int64

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, cachePattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			data, err := r.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			cache := &TranslationCache{}
			if err := json.Unmarshal(data, cache); err != nil {
				continue
			}

			stats.TotalTranslations++
			totalAccess += int64(cache.AccessCount)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	// Calculate cache hit rate
	if totalAccess > 0 && stats.TotalTranslations > 0 {
		stats.CacheHitRate = float64(totalAccess-stats.TotalTranslations) / float64(totalAccess) * 100.0
	}

	return stats, nil
}

// Ping checks the Redis connection
func (r *RedisStorage) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (r *RedisStorage) Close() error {
	return r.client.Close()
}

// makeCacheKey creates a cache key from translation parameters
func (r *RedisStorage) makeCacheKey(sourceText, sourceLanguage, targetLanguage, provider, model string) string {
	return fmt.Sprintf("cache:%s:%s:%s:%s:%s", sourceLanguage, targetLanguage, provider, model, hashString(sourceText))
}

// accumulateAvgDuration folds one more duration sample into a running mean.
// prevAvg/prevN are the mean and sample count BEFORE this sample; it returns
// the updated mean and count. The divisor is the number of duration samples
// actually folded in — never the count of all completed sessions — so a
// completed session with no recorded duration cannot dilute the average.
func accumulateAvgDuration(prevAvg float64, prevN int64, duration float64) (float64, int64) {
	n := prevN + 1
	avg := (prevAvg*float64(prevN) + duration) / float64(n)
	return avg, n
}

// hashString creates a collision-resistant hash of a string (for cache keys).
//
// A previous implementation used a 32-bit polynomial hash (h = h*31 + c),
// which has trivial collisions ("Aa"/"BB", "AaAa"/"BBBB", ...) and overflows
// at 2^32. Because the cache key embeds this hash, a collision made two
// DISTINCT source texts share a Redis key, so a lookup could return the WRONG
// translation. sha256 (256-bit, cryptographically collision-resistant) removes
// that defect; the full hex digest keeps keys unambiguous.
func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
