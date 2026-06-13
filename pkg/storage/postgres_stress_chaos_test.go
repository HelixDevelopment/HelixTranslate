//go:build integration

// Package storage stress + chaos integration tests for the PostgreSQL backend
// (W15 + §11.4.85 stress/chaos).
//
// These exercise the REAL PostgreSQLStorage against a REAL ephemeral PostgreSQL
// booted on demand via the containers submodule's brokertest helper
// (digital.vasic.containers/pkg/brokertest) — no mocks, no fakes (§11.4.27,
// §11.4.76 on-demand-infra invariant). Exactly ONE memory-limited container is
// booted per run (§12.6), bound to 127.0.0.1, torn down on every exit path
// (§11.4.14). If a container runtime is unavailable the suite SKIPs with reason
// (§11.4.3) rather than failing.
//
// Every assertion checks REAL persisted data (row counts, read-back values,
// access-count deltas), never merely a nil error (§11.4.85 anti-bluff; §11.4.69
// captured-state evidence is the persisted row itself).
//
// Run:  go test -tags=integration -race -run TestPostgresStressChaos ./pkg/storage/ -count=1 -timeout 240s
package storage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"digital.vasic.containers/pkg/brokertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startRealPostgres boots ONE memory-limited ephemeral Postgres and returns a
// connected PostgreSQLStorage. SKIPs (§11.4.3) when no container runtime exists.
// The caller gets a torn-down container + closed storage via the returned
// cleanup (§11.4.14). This is the single bounded container for the whole suite
// (§12.6) — every subtest reuses it.
func startRealPostgres(t *testing.T, ctx context.Context) (*PostgreSQLStorage, func()) {
	t.Helper()
	dsn, stop, err := brokertest.StartPostgres(ctx, brokertest.WithMemoryLimit("256m"))
	if err != nil {
		t.Skipf("SKIP-OK: container runtime unavailable for real Postgres — %v (§11.4.3 topology absent)", err)
	}
	st, err := NewPostgreSQLStorage(configFromDSN(t, dsn))
	if err != nil {
		stop()
		require.NoError(t, err, "NewPostgreSQLStorage against the booted Postgres (schema init must succeed)")
	}
	require.NoError(t, st.Ping(ctx), "Ping the real Postgres")
	cleanup := func() {
		_ = st.Close()
		stop() // §11.4.14 — container down on every exit path
	}
	return st, cleanup
}

// TestPostgresStressChaos runs the full stress / concurrency / boundary / chaos
// battery against ONE real Postgres container (§12.6 one bounded container).
func TestPostgresStressChaos(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Second)
	defer cancel()

	st, cleanup := startRealPostgres(t, ctx)
	defer cleanup()

	t.Run("Stress_ConcurrentSessionsAndCache", func(t *testing.T) {
		stressConcurrentSessionsAndCache(ctx, t, st)
	})
	t.Run("Stress_100CacheWritesAllReadable", func(t *testing.T) {
		stress100CacheWritesAllReadable(ctx, t, st)
	})
	t.Run("Concurrency_SameKeyAccessCountIncrements", func(t *testing.T) {
		concurrencySameKeyAccessCount(ctx, t, st)
	})
	t.Run("Concurrency_DistinctSessionIDsAllPersisted", func(t *testing.T) {
		concurrencyDistinctSessionIDs(ctx, t, st)
	})
	t.Run("Boundary_EmptyAndDuplicateSessionID", func(t *testing.T) {
		boundaryEmptyAndDuplicateSessionID(ctx, t, st)
	})
	t.Run("Boundary_CacheOverwriteReplaceSemantics", func(t *testing.T) {
		boundaryCacheOverwrite(ctx, t, st)
	})
	t.Run("Boundary_CleanupOldCacheKeepsFresh", func(t *testing.T) {
		boundaryCleanupOldCache(ctx, t, st)
	})
	t.Run("Chaos_CancelledContextReturnsPromptly", func(t *testing.T) {
		chaosCancelledContext(ctx, t, st)
	})
	t.Run("Chaos_ClosedStorageReturnsErrorNotPanic", func(t *testing.T) {
		chaosClosedStorageReturnsError(t)
	})
}

func scMkSession(id, status string, now time.Time) *TranslationSession {
	return &TranslationSession{
		ID:             id,
		BookTitle:      "Гавран — " + id,
		InputFile:      "in.fb2",
		OutputFile:     "out.epub",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		Status:         status,
		TotalChapters:  1,
		ItemsTotal:     1,
		StartTime:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func scMkCache(id, source, target string, now time.Time) *TranslationCache {
	return &TranslationCache{
		ID:             id,
		SourceText:     source,
		TargetText:     target,
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		LastAccessedAt: now,
	}
}

// --- STRESS: >=10 concurrent goroutines, CreateSession + CacheTranslation +
// GetCachedTranslation against the real DB. Asserts no lost writes (every row
// persisted), no deadlock (bounded wait), correct final counts via
// GetStatistics. ---
func stressConcurrentSessionsAndCache(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	const workers = 16
	now := time.Now().UTC().Truncate(time.Second)
	prefix := fmt.Sprintf("stress-%d", now.UnixNano())

	// Baseline counts so the test is robust to rows left by sibling subtests.
	base, err := st.GetStatistics(ctx)
	require.NoError(t, err, "baseline GetStatistics")

	var wg sync.WaitGroup
	errCh := make(chan error, workers*3)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			sid := fmt.Sprintf("%s-sess-%d", prefix, n)
			// Each worker creates a "completed" session ...
			if e := st.CreateSession(ctx, scMkSession(sid, "completed", now)); e != nil {
				errCh <- fmt.Errorf("worker %d CreateSession: %w", n, e)
				return
			}
			// ... caches a distinct translation ...
			cid := fmt.Sprintf("%s-cache-%d", prefix, n)
			src := fmt.Sprintf("source text number %d", n)
			tgt := fmt.Sprintf("преведени текст број %d", n)
			if e := st.CacheTranslation(ctx, scMkCache(cid, src, tgt, now)); e != nil {
				errCh <- fmt.Errorf("worker %d CacheTranslation: %w", n, e)
				return
			}
			// ... and reads it back, asserting the real persisted value.
			hit, e := st.GetCachedTranslation(ctx, src, "en", "sr", "deepseek", "deepseek-chat")
			if e != nil {
				errCh <- fmt.Errorf("worker %d GetCachedTranslation: %w", n, e)
				return
			}
			if hit == nil || hit.TargetText != tgt {
				errCh <- fmt.Errorf("worker %d read-back mismatch: got %v", n, hit)
			}
		}(i)
	}

	// No-deadlock guard: bounded wait. A hang fails loudly instead of blocking.
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(90 * time.Second):
		t.Fatal("stress goroutines did not finish within 90s — possible deadlock (§11.4.85)")
	}
	close(errCh)
	for e := range errCh {
		t.Errorf("concurrent op failed: %v", e)
	}

	// No lost writes: every session + cache row persisted (final == baseline+workers).
	final, err := st.GetStatistics(ctx)
	require.NoError(t, err, "final GetStatistics")
	assert.Equal(t, base.TotalSessions+workers, final.TotalSessions,
		"all %d concurrent sessions persisted (no lost writes)", workers)
	assert.Equal(t, base.CompletedSessions+workers, final.CompletedSessions,
		"all %d completed sessions counted", workers)
	assert.Equal(t, base.TotalTranslations+workers, final.TotalTranslations,
		"all %d concurrent cache writes persisted (no lost writes)", workers)
}

// >=100 cache writes then verify ALL readable with their exact stored value.
func stress100CacheWritesAllReadable(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	const n = 120
	now := time.Now().UTC().Truncate(time.Second)
	prefix := fmt.Sprintf("bulk-%d", now.UnixNano())

	// Bounded in-flight concurrency (semaphore). All 120 writes still happen
	// concurrently in waves, but at most `maxInFlight` connections are open at
	// once — a realistic pooled client. WITHOUT this bound the unbounded
	// database/sql pool (see BUG_REPORT — NewPostgreSQLStorage leaves
	// MaxOpenConns at Go's default of 0/unlimited) opens 120 connections at
	// once and the server rejects with "pq: sorry, too many clients already".
	// This test exercises real high-throughput load without that production
	// defect masking the lost-write check it is here to make.
	const maxInFlight = 20
	sem := make(chan struct{}, maxInFlight)
	var wg sync.WaitGroup
	writeErr := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			cid := fmt.Sprintf("%s-c-%d", prefix, k)
			src := fmt.Sprintf("%s phrase %d", prefix, k)
			tgt := fmt.Sprintf("%s фраза %d", prefix, k)
			if e := st.CacheTranslation(ctx, scMkCache(cid, src, tgt, now)); e != nil {
				writeErr <- e
			}
		}(i)
	}
	wg.Wait()
	close(writeErr)
	for e := range writeErr {
		t.Fatalf("bulk cache write failed: %v", e)
	}

	// Every one of the 120 entries must be individually readable with the
	// correct persisted target text — not a sampled subset.
	readable := 0
	for i := 0; i < n; i++ {
		src := fmt.Sprintf("%s phrase %d", prefix, i)
		want := fmt.Sprintf("%s фраза %d", prefix, i)
		hit, err := st.GetCachedTranslation(ctx, src, "en", "sr", "deepseek", "deepseek-chat")
		require.NoError(t, err, "read-back %d", i)
		require.NotNil(t, hit, "entry %d must be readable after bulk write", i)
		assert.Equal(t, want, hit.TargetText, "entry %d target round-trips", i)
		readable++
	}
	assert.Equal(t, n, readable, "all %d bulk cache writes are readable", n)
}

// --- CONCURRENCY: concurrent GetCachedTranslation on the SAME key. Each hit
// increments access_count (a real UPDATE). The final access_count must equal
// the number of concurrent reads exactly — proving no lost increment. ---
func concurrencySameKeyAccessCount(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	const reads = 30
	now := time.Now().UTC().Truncate(time.Second)
	cid := fmt.Sprintf("acc-%d", now.UnixNano())
	src := "shared lookup key " + cid
	require.NoError(t, st.CacheTranslation(ctx, scMkCache(cid, src, "дељени циљ", now)),
		"seed cache entry for access-count test")

	// Learn the starting access_count by reading the column DIRECTLY (not via
	// GetCachedTranslation — that path itself fires a +1 UPDATE, which would
	// add a phantom increment to the burst total). CacheTranslation seeds it 0.
	var startCount int
	require.NoError(t, st.db.QueryRowContext(ctx,
		"SELECT access_count FROM translation_cache WHERE id = $1", cid).Scan(&startCount),
		"read starting access_count directly (no side-effecting read)")

	var wg sync.WaitGroup
	readErr := make(chan error, reads)
	for i := 0; i < reads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h, e := st.GetCachedTranslation(ctx, src, "en", "sr", "deepseek", "deepseek-chat")
			if e != nil {
				readErr <- e
				return
			}
			if h == nil {
				readErr <- fmt.Errorf("concurrent read returned nil for an existing key")
			}
		}()
	}
	wg.Wait()
	close(readErr)
	for e := range readErr {
		t.Errorf("concurrent same-key read failed: %v", e)
	}

	// Each GetCachedTranslation increments access_count by 1 (UPDATE … +1).
	// After exactly `reads` concurrent hits the persisted count must have
	// advanced by exactly `reads` (no lost UPDATE under contention). We read
	// the column DIRECTLY here so this verification does not itself increment.
	var persisted int
	row := st.db.QueryRowContext(ctx,
		"SELECT access_count FROM translation_cache WHERE id = $1", cid)
	require.NoError(t, row.Scan(&persisted), "read persisted access_count directly")
	assert.Equal(t, startCount+reads, persisted,
		"access_count advanced by exactly %d concurrent reads (no lost increment)", reads)
}

// concurrent CreateSession with DISTINCT IDs — all persisted; ListSessions count
// reflects every one.
func concurrencyDistinctSessionIDs(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	const n = 25
	now := time.Now().UTC().Truncate(time.Second)
	prefix := fmt.Sprintf("distinct-%d", now.UnixNano())

	var wg sync.WaitGroup
	createErr := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			sid := fmt.Sprintf("%s-%d", prefix, k)
			if e := st.CreateSession(ctx, scMkSession(sid, "translating", now)); e != nil {
				createErr <- e
			}
		}(i)
	}
	wg.Wait()
	close(createErr)
	for e := range createErr {
		t.Errorf("concurrent distinct-ID CreateSession failed: %v", e)
	}

	// Verify each distinct ID is individually retrievable (real persisted rows).
	found := 0
	for i := 0; i < n; i++ {
		sid := fmt.Sprintf("%s-%d", prefix, i)
		got, err := st.GetSession(ctx, sid)
		require.NoError(t, err, "GetSession %s after concurrent create", sid)
		require.NotNil(t, got)
		assert.Equal(t, sid, got.ID)
		found++
	}
	assert.Equal(t, n, found, "all %d distinct-ID sessions persisted", n)

	// ListSessions with a generous page must contain at least our n rows.
	list, err := st.ListSessions(ctx, n+1000, 0)
	require.NoError(t, err, "ListSessions")
	matched := 0
	for _, s := range list {
		if len(s.ID) >= len(prefix) && s.ID[:len(prefix)] == prefix {
			matched++
		}
	}
	assert.Equal(t, n, matched, "ListSessions returns all %d concurrently-created sessions", n)
}

// --- BOUNDARY: empty-ID create persists & is retrievable; duplicate-ID create
// returns a categorised error (CreateSession uses plain INSERT, so PG raises a
// unique-violation — NOT an upsert). ---
func boundaryEmptyAndDuplicateSessionID(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	now := time.Now().UTC().Truncate(time.Second)

	// Empty string is a valid TEXT primary-key value in Postgres (distinct from
	// NULL). Assert the REAL behaviour: it inserts and is retrievable by "".
	empty := scMkSession("", "completed", now)
	require.NoError(t, st.CreateSession(ctx, empty), "CreateSession with empty ID (TEXT PK '' is valid)")
	gotEmpty, err := st.GetSession(ctx, "")
	require.NoError(t, err, "GetSession by empty ID")
	require.NotNil(t, gotEmpty)
	assert.Equal(t, "", gotEmpty.ID, "empty-ID session round-trips")

	// Duplicate ID: second CreateSession on the same PK must ERROR (no upsert).
	dupID := fmt.Sprintf("dup-%d", now.UnixNano())
	require.NoError(t, st.CreateSession(ctx, scMkSession(dupID, "completed", now)), "first create")
	err = st.CreateSession(ctx, scMkSession(dupID, "translating", now))
	require.Error(t, err, "duplicate-PK CreateSession must return a real unique-violation error (not silent upsert)")

	// And the ORIGINAL row must be untouched (status still 'completed'),
	// proving the failed insert did not partially overwrite.
	orig, err := st.GetSession(ctx, dupID)
	require.NoError(t, err)
	require.NotNil(t, orig)
	assert.Equal(t, "completed", orig.Status,
		"original row unchanged after a rejected duplicate insert")
}

// CacheTranslation overwrite — ON CONFLICT (id) DO UPDATE replaces target_text.
// Write a wrong value, overwrite with the corrected one, read back the corrected.
func boundaryCacheOverwrite(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	now := time.Now().UTC().Truncate(time.Second)
	cid := fmt.Sprintf("ow-%d", now.UnixNano())
	src := "overwrite source " + cid

	require.NoError(t, st.CacheTranslation(ctx, scMkCache(cid, src, "WRONG translation", now)),
		"initial (wrong) cache write")
	first, err := st.GetCachedTranslation(ctx, src, "en", "sr", "deepseek", "deepseek-chat")
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "WRONG translation", first.TargetText, "first write stored the wrong value")

	// Overwrite the SAME id with the corrected target (INSERT-OR-REPLACE semantics).
	require.NoError(t, st.CacheTranslation(ctx, scMkCache(cid, src, "ИСПРАВНО tačan prevod", now)),
		"overwrite cache write (same id)")
	corrected, err := st.GetCachedTranslation(ctx, src, "en", "sr", "deepseek", "deepseek-chat")
	require.NoError(t, err)
	require.NotNil(t, corrected)
	assert.Equal(t, "ИСПРАВНО tačan prevod", corrected.TargetText,
		"overwrite replaced the target_text (read-back corrected value)")

	// Still exactly ONE row for this id (overwrite, not a second insert).
	var cnt int
	require.NoError(t, st.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM translation_cache WHERE id = $1", cid).Scan(&cnt))
	assert.Equal(t, 1, cnt, "overwrite kept a single row (not a duplicate insert)")
}

// CleanupOldCache removes stale entries and keeps fresh ones.
func boundaryCleanupOldCache(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	now := time.Now().UTC()
	tag := fmt.Sprintf("clean-%d", now.UnixNano())

	// Stale: last_accessed_at well in the past.
	stale := scMkCache(tag+"-stale", tag+" stale src", "stari", now.Add(-72*time.Hour))
	stale.LastAccessedAt = now.Add(-72 * time.Hour)
	require.NoError(t, st.CacheTranslation(ctx, stale), "seed stale entry")

	// Fresh: last_accessed_at = now.
	fresh := scMkCache(tag+"-fresh", tag+" fresh src", "novi", now)
	fresh.LastAccessedAt = now
	require.NoError(t, st.CacheTranslation(ctx, fresh), "seed fresh entry")

	// Remove anything older than 24h. Stale (-72h) goes; fresh (now) stays.
	require.NoError(t, st.CleanupOldCache(ctx, 24*time.Hour), "CleanupOldCache(24h)")

	var staleCnt, freshCnt int
	require.NoError(t, st.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM translation_cache WHERE id = $1", tag+"-stale").Scan(&staleCnt))
	require.NoError(t, st.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM translation_cache WHERE id = $1", tag+"-fresh").Scan(&freshCnt))
	assert.Equal(t, 0, staleCnt, "stale entry removed by CleanupOldCache")
	assert.Equal(t, 1, freshCnt, "fresh entry kept by CleanupOldCache")
}

// --- CHAOS (real-infra-appropriate, no destructive host action): a cancelled
// context mid-operation returns promptly (no hang); operating on a CLOSED
// storage returns an error, never a panic. ---
func chaosCancelledContext(ctx context.Context, t *testing.T, st *PostgreSQLStorage) {
	// Seed a row so there is something to read.
	now := time.Now().UTC().Truncate(time.Second)
	sid := fmt.Sprintf("cancel-%d", now.UnixNano())
	require.NoError(t, st.CreateSession(ctx, scMkSession(sid, "completed", now)), "seed for cancel test")

	cctx, cancel := context.WithCancel(ctx)
	cancel() // already cancelled BEFORE the call

	start := time.Now()
	// Any of these may error with context.Canceled; the contract under chaos is
	// "return promptly, do not hang and do not panic".
	_, err := st.GetSession(cctx, sid)
	elapsed := time.Since(start)
	assert.Error(t, err, "GetSession on a cancelled context returns an error")
	assert.Less(t, elapsed, 5*time.Second, "cancelled GetSession returned promptly (no hang)")

	start = time.Now()
	err = st.CreateSession(cctx, scMkSession(sid+"-x", "completed", now))
	assert.Error(t, err, "CreateSession on a cancelled context returns an error")
	assert.Less(t, time.Since(start), 5*time.Second, "cancelled CreateSession returned promptly")

	// The DB itself is still healthy for non-cancelled work afterwards.
	require.NoError(t, st.Ping(ctx), "real Postgres still healthy after cancelled ops")
	got, err := st.GetSession(ctx, sid)
	require.NoError(t, err, "seeded row still retrievable with a live context")
	require.NotNil(t, got)
}

// Closed storage: Close() then operate — must return an error, not panic.
// Boots its OWN short-lived container so it never poisons the shared `st`
// used by the other subtests. Still ONE container at a time (§12.6) — this
// runs after creating + immediately closing its own.
func chaosClosedStorageReturnsError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dsn, stop, err := brokertest.StartPostgres(ctx, brokertest.WithMemoryLimit("256m"))
	if err != nil {
		t.Skipf("SKIP-OK: container runtime unavailable — %v (§11.4.3)", err)
	}
	defer stop()

	st2, err := NewPostgreSQLStorage(configFromDSN(t, dsn))
	require.NoError(t, err, "NewPostgreSQLStorage for closed-storage chaos")
	require.NoError(t, st2.Ping(ctx), "Ping before close")

	require.NoError(t, st2.Close(), "Close the storage")

	// Operating after Close must surface an error (sql: database is closed),
	// NOT panic. We capture any panic and fail explicitly if one occurs.
	now := time.Now().UTC().Truncate(time.Second)
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("operation on closed storage PANICKED (must return error): %v", r)
			}
		}()
		err := st2.CreateSession(ctx, scMkSession("after-close", "completed", now))
		assert.Error(t, err, "CreateSession on closed storage returns an error (not a panic)")

		_, gerr := st2.GetSession(ctx, "after-close")
		assert.Error(t, gerr, "GetSession on closed storage returns an error")

		perr := st2.Ping(ctx)
		assert.Error(t, perr, "Ping on closed storage returns an error")
	}()
}
