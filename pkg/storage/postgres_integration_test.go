//go:build integration

// Package storage integration tests for the PostgreSQL backend (W15).
//
// These exercise the REAL PostgreSQLStorage against a REAL ephemeral PostgreSQL
// booted on demand via the containers submodule's brokertest helper
// (digital.vasic.containers/pkg/brokertest) — no mocks, no fakes (§11.4.27,
// §11.4.76 on-demand-infra invariant). The container is memory-limited (§12.6),
// bound to 127.0.0.1, and torn down on every exit path (§11.4.14).
//
// Run:  go test -tags=integration -run TestPostgresIntegration ./pkg/storage/
// Requires: a working container runtime (podman/docker). If absent/unreachable
// the test SKIPs with reason (§11.4.3) rather than failing.
package storage

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"digital.vasic.containers/pkg/brokertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configFromDSN parses a postgres://user:pass@host:port/db?sslmode=… DSN
// (the shape brokertest.StartPostgres returns) into a storage.Config.
func configFromDSN(t *testing.T, dsn string) *Config {
	t.Helper()
	u, err := url.Parse(dsn)
	require.NoError(t, err, "parse brokertest DSN")
	require.Equal(t, "postgres", u.Scheme)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err, "parse port from DSN")
	pass, _ := u.User.Password()
	ssl := u.Query().Get("sslmode")
	if ssl == "" {
		ssl = "disable"
	}
	return &Config{
		Type:     "postgres",
		Host:     u.Hostname(),
		Port:     port,
		Database: u.Path[1:], // strip leading "/"
		Username: u.User.Username(),
		Password: pass,
		SSLMode:  ssl,
	}
}

func TestPostgresIntegration_RealCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// On-demand boot of a REAL Postgres (§11.4.76). Memory-limited (§12.6),
	// ephemeral, 127.0.0.1-only. If the runtime is unavailable, SKIP (§11.4.3).
	dsn, stop, err := brokertest.StartPostgres(ctx, brokertest.WithMemoryLimit("256m"))
	if err != nil {
		t.Skipf("SKIP-OK: container runtime unavailable for real Postgres — %v (§11.4.3 topology absent)", err)
	}
	defer stop() // §11.4.14 cleanup on every exit path

	st, err := NewPostgreSQLStorage(configFromDSN(t, dsn))
	require.NoError(t, err, "NewPostgreSQLStorage against the booted Postgres (schema init must succeed)")
	defer func() { _ = st.Close() }()

	require.NoError(t, st.Ping(ctx), "Ping the real Postgres")

	// --- Session CRUD round-trip (real INSERT/SELECT/UPDATE/DELETE) ---
	now := time.Now().UTC().Truncate(time.Second)
	sess := &TranslationSession{
		ID:             "w15-sess-1",
		BookTitle:      "Гавран и бокал",
		InputFile:      "in.fb2",
		OutputFile:     "out.epub",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         "in_progress",
		TotalChapters:  3,
		ItemsTotal:     10,
		StartTime:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, st.CreateSession(ctx, sess), "CreateSession")

	got, err := st.GetSession(ctx, "w15-sess-1")
	require.NoError(t, err, "GetSession")
	require.NotNil(t, got)
	assert.Equal(t, "Гавран и бокал", got.BookTitle, "book title round-trips incl. Cyrillic")
	assert.Equal(t, "in_progress", got.Status)
	assert.Equal(t, 3, got.TotalChapters)
	assert.Equal(t, "sr", got.TargetLanguage)

	// Update + read back the change (real UPDATE)
	got.Status = "completed"
	got.PercentComplete = 100
	got.ItemsCompleted = 10
	require.NoError(t, st.UpdateSession(ctx, got), "UpdateSession")
	after, err := st.GetSession(ctx, "w15-sess-1")
	require.NoError(t, err)
	assert.Equal(t, "completed", after.Status, "status update persisted")
	assert.InDelta(t, 100.0, after.PercentComplete, 0.001)

	// List (real SELECT … ORDER BY … LIMIT/OFFSET)
	list, err := st.ListSessions(ctx, 10, 0)
	require.NoError(t, err, "ListSessions")
	assert.GreaterOrEqual(t, len(list), 1, "the created session is listed")

	// --- Translation cache round-trip ---
	cache := &TranslationCache{
		ID:             "w15-cache-1",
		SourceText:     "The crow was thirsty.",
		TargetText:     "Гавран је био жедан.",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		LastAccessedAt: now,
	}
	require.NoError(t, st.CacheTranslation(ctx, cache), "CacheTranslation")
	hit, err := st.GetCachedTranslation(ctx, "The crow was thirsty.", "en", "sr", "deepseek", "deepseek-chat")
	require.NoError(t, err, "GetCachedTranslation")
	require.NotNil(t, hit, "cache hit returns the stored entry")
	assert.Equal(t, "Гавран је био жедан.", hit.TargetText, "cached target round-trips incl. Cyrillic")
	miss, err := st.GetCachedTranslation(ctx, "absent text", "en", "sr", "deepseek", "deepseek-chat")
	require.NoError(t, err)
	assert.Nil(t, miss, "cache miss returns nil,nil")

	// --- Statistics over real rows ---
	stats, err := st.GetStatistics(ctx)
	require.NoError(t, err, "GetStatistics")
	require.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalSessions, int64(1))
	assert.GreaterOrEqual(t, stats.CompletedSessions, int64(1), "the completed session is counted")
	assert.GreaterOrEqual(t, stats.TotalTranslations, int64(1), "the cached translation is counted")

	// --- Delete (real DELETE) ---
	require.NoError(t, st.DeleteSession(ctx, "w15-sess-1"), "DeleteSession")
	_, err = st.GetSession(ctx, "w15-sess-1")
	assert.Error(t, err, "GetSession after delete returns an error / not-found")
}

// TestPostgresIntegration_PoolBoundedNoClientExhaustion is the W15 regression
// guard for the connection-exhaustion DoS fix: with the DEFAULT Config{}
// (MaxOpenConns==0) NewPostgreSQLStorage now bounds the pool (25), so N concurrent
// writers (N > a plausible unbounded burst) complete WITHOUT `pq: sorry, too many
// clients already`. Before the fix the pool was unlimited and this flooded the
// server's max_connections. No semaphore here — that's the whole point.
func TestPostgresIntegration_PoolBoundedNoClientExhaustion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	dsn, stop, err := brokertest.StartPostgres(ctx, brokertest.WithMemoryLimit("256m"))
	if err != nil {
		t.Skipf("SKIP-OK: container runtime unavailable — %v (§11.4.3 topology absent)", err)
	}
	defer stop()

	st, err := NewPostgreSQLStorage(configFromDSN(t, dsn)) // DEFAULT config → pool bounded by the fix
	require.NoError(t, err)
	defer func() { _ = st.Close() }()

	// DETERMINISTIC guard (§11.4.115/§11.4.50): white-box access to the pool's
	// configured ceiling. SetMaxOpenConns(25) → Stats().MaxOpenConnections == 25;
	// the pre-fix unbounded pool → 0. Asserting the bound directly is deterministic
	// — unlike racing actual connection-exhaustion, which is timing-dependent and
	// does NOT reliably reproduce (a blind test). Mutation: revert the default
	// bound → MaxOpenConnections == 0 → this FAILs (the guard genuinely catches it).
	require.Equal(t, 25, st.db.Stats().MaxOpenConnections,
		"default Config{} must bound the connection pool (W15 DoS fix); unbounded (0) exhausts Postgres max_connections")

	// Real-load sanity: a concurrent burst completes WITHOUT connection-exhaustion
	// errors now that the pool is bounded (connections queue through the 25-cap).
	const n = 60
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if e := st.CacheTranslation(ctx, &TranslationCache{
				ID:             fmt.Sprintf("pool-%d", i),
				SourceText:     fmt.Sprintf("src-%d", i),
				TargetText:     fmt.Sprintf("tgt-%d", i),
				SourceLanguage: "en", TargetLanguage: "sr", Provider: "p", Model: "m",
				CreatedAt:      time.Now(), LastAccessedAt: time.Now(),
			}); e != nil {
				errs <- e
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	var got []error
	for e := range errs {
		got = append(got, e)
	}
	require.Empty(t, got, "%d concurrent writers complete without connection-exhaustion (bounded pool); got: %v", n, got)
}
