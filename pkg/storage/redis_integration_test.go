//go:build integration

// Package storage integration tests for the Redis backend.
//
// These exercise the REAL RedisStorage against a REAL ephemeral Redis booted on
// demand via the containers submodule's brokertest helper
// (digital.vasic.containers/pkg/brokertest) — no mocks, no fakes (§11.4.27,
// §11.4.76 on-demand-infra invariant). The container is memory-limited (§12.6),
// bound to 127.0.0.1, and torn down on every exit path (§11.4.14).
//
// Run:  go test -tags=integration -run TestRedisIntegration ./pkg/storage/
// Requires: a working container runtime (podman/docker). If absent/unreachable
// the test SKIPs with reason (§11.4.3) rather than failing.
package storage

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"digital.vasic.containers/pkg/brokertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configFromRedisAddr parses a "host:port" address (the shape
// brokertest.StartRedis returns) into a storage.Config for the Redis backend.
func configFromRedisAddr(t *testing.T, addr string) *Config {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err, "split brokertest redis addr %q", addr)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "parse port from redis addr %q", addr)
	return &Config{
		Type: "redis",
		Host: host,
		Port: port,
	}
}

func TestRedisIntegration_RealCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// On-demand boot of a REAL Redis (§11.4.76). Memory-limited (§12.6),
	// ephemeral, 127.0.0.1-only. If the runtime is unavailable, SKIP (§11.4.3).
	addr, stop, err := brokertest.StartRedis(ctx, brokertest.WithMemoryLimit("128m"))
	if err != nil {
		t.Skipf("SKIP-OK: container runtime unavailable for real Redis — %v (§11.4.3 topology absent)", err)
	}
	defer stop() // §11.4.14 cleanup on every exit path

	// 1h TTL so nothing expires under the test wall-clock window.
	st, err := NewRedisStorage(configFromRedisAddr(t, addr), time.Hour)
	require.NoError(t, err, "NewRedisStorage against the booted Redis (PING on connect must succeed)")
	defer func() { _ = st.Close() }()

	require.NoError(t, st.Ping(ctx), "Ping the real Redis")

	// --- Session save/get round-trip (real SET/GET against Redis) ---
	now := time.Now().UTC().Truncate(time.Second)
	sess := &TranslationSession{
		ID:             "redis-sess-1",
		BookTitle:      "Гавран и бокал",
		InputFile:      "in.fb2",
		OutputFile:     "out.epub",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         "translating",
		TotalChapters:  3,
		ItemsTotal:     10,
		StartTime:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, st.CreateSession(ctx, sess), "CreateSession (real SET)")

	got, err := st.GetSession(ctx, "redis-sess-1")
	require.NoError(t, err, "GetSession (real GET)")
	require.NotNil(t, got)
	assert.Equal(t, "Гавран и бокал", got.BookTitle, "book title round-trips incl. Cyrillic")
	assert.Equal(t, "translating", got.Status)
	assert.Equal(t, 3, got.TotalChapters)
	assert.Equal(t, "sr", got.TargetLanguage)

	// Session miss: a never-stored ID returns an explicit not-found error.
	_, err = st.GetSession(ctx, "no-such-session")
	assert.Error(t, err, "GetSession on a missing id returns a not-found error (real redis.Nil path)")

	// Update + read back the change (real overwrite SET via UpdateSession).
	got.Status = "completed"
	got.PercentComplete = 100
	got.ItemsCompleted = 10
	end := now.Add(2 * time.Minute)
	got.EndTime = &end
	require.NoError(t, st.UpdateSession(ctx, got), "UpdateSession (real overwrite SET)")
	after, err := st.GetSession(ctx, "redis-sess-1")
	require.NoError(t, err)
	assert.Equal(t, "completed", after.Status, "status update persisted")
	assert.InDelta(t, 100.0, after.PercentComplete, 0.001)

	// List (real SCAN over session:* keys).
	list, err := st.ListSessions(ctx, 10, 0)
	require.NoError(t, err, "ListSessions (real SCAN)")
	assert.GreaterOrEqual(t, len(list), 1, "the created session is listed")

	// --- Cache set/get hit + miss, exercising the cache-key helpers end-to-end ---
	cache := &TranslationCache{
		ID:             "redis-cache-1",
		SourceText:     "The crow was thirsty.",
		TargetText:     "Гавран је био жедан.",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		LastAccessedAt: now,
	}
	require.NoError(t, st.CacheTranslation(ctx, cache), "CacheTranslation (real SET under makeCacheKey)")

	// HIT: identical (text, langs, provider, model) tuple resolves to the same
	// cache key and returns the stored entry.
	hit, err := st.GetCachedTranslation(ctx, "The crow was thirsty.", "en", "sr", "deepseek", "deepseek-chat")
	require.NoError(t, err, "GetCachedTranslation hit")
	require.NotNil(t, hit, "cache hit returns the stored entry")
	assert.Equal(t, "Гавран је био жедан.", hit.TargetText, "cached target round-trips incl. Cyrillic")

	// MISS by source text — proves the key incorporates the source-text hash.
	missText, err := st.GetCachedTranslation(ctx, "different text entirely", "en", "sr", "deepseek", "deepseek-chat")
	require.NoError(t, err)
	assert.Nil(t, missText, "cache miss on differing source text returns nil,nil")

	// MISS by a key dimension (target language) — proves the key incorporates
	// the language pair, not just the text. Same text, different target lang.
	missLang, err := st.GetCachedTranslation(ctx, "The crow was thirsty.", "en", "de", "deepseek", "deepseek-chat")
	require.NoError(t, err)
	assert.Nil(t, missLang, "cache miss on differing target language returns nil,nil")

	// MISS by provider/model dimension — proves both are part of the key.
	missModel, err := st.GetCachedTranslation(ctx, "The crow was thirsty.", "en", "sr", "openai", "gpt-4o")
	require.NoError(t, err)
	assert.Nil(t, missModel, "cache miss on differing provider/model returns nil,nil")

	// --- Statistics over real keys (SCAN of session:* and cache:*) ---
	stats, err := st.GetStatistics(ctx)
	require.NoError(t, err, "GetStatistics (real SCAN census)")
	require.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalSessions, int64(1), "the session is counted")
	assert.GreaterOrEqual(t, stats.CompletedSessions, int64(1), "the completed session is counted")
	assert.GreaterOrEqual(t, stats.TotalTranslations, int64(1), "the cached translation is counted")

	// --- Delete (real DEL) ---
	require.NoError(t, st.DeleteSession(ctx, "redis-sess-1"), "DeleteSession (real DEL)")
	_, err = st.GetSession(ctx, "redis-sess-1")
	assert.Error(t, err, "GetSession after delete returns a not-found error")
}
