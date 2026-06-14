package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// P6 coverage (§11.4.27 / §11.4.85): genuinely-missing performance/benchmark +
// stress/chaos coverage for the storage cache hot paths.
//
// Additive-only: these exercise the REAL functions (cacheLookupHash, makeCacheKey,
// hashString, accumulateAvgDuration, SQLite CacheTranslation/GetCachedTranslation),
// never mocks/stubs, so a regression in the function under test changes the
// measured numbers / breaks the asserted correctness. SQLite is an embedded
// backend (no live Postgres/Redis required), so these run fully offline.

// ---------------------------------------------------------------------------
// Benchmarks — pure cache-key hot paths (no I/O).
// Run: go test -bench=. -benchmem -run=^$ ./pkg/storage/
// ---------------------------------------------------------------------------

// benchSourceText is a realistic paragraph-sized source text — cache keys are
// computed over full paragraphs, so a tiny string would understate the cost.
const benchSourceText = "The quick brown fox jumps over the lazy dog. " +
	"Pack my box with five dozen liquor jugs. " +
	"How vexingly quick daft zebras jump! " +
	"Sphinx of black quartz, judge my vow. " +
	"The five boxing wizards jump quickly."

func BenchmarkCacheLookupHash(b *testing.B) {
	b.ReportAllocs()
	var sink string
	for i := 0; i < b.N; i++ {
		sink = cacheLookupHash(benchSourceText, "en", "sr", "deepseek", "deepseek-chat")
	}
	_ = sink
}

func BenchmarkRedisMakeCacheKey(b *testing.B) {
	r := &RedisStorage{}
	b.ReportAllocs()
	var sink string
	for i := 0; i < b.N; i++ {
		sink = r.makeCacheKey(benchSourceText, "en", "sr", "deepseek", "deepseek-chat")
	}
	_ = sink
}

func BenchmarkHashString(b *testing.B) {
	b.ReportAllocs()
	var sink string
	for i := 0; i < b.N; i++ {
		sink = hashString(benchSourceText)
	}
	_ = sink
}

func BenchmarkAccumulateAvgDuration(b *testing.B) {
	b.ReportAllocs()
	avg := 0.0
	var n int64
	for i := 0; i < b.N; i++ {
		avg, n = accumulateAvgDuration(avg, n, float64(i%1000))
	}
	_ = avg
	_ = n
}

// ---------------------------------------------------------------------------
// Benchmark — SQLite cache GET hot path (the read side; only CacheTranslation
// had a benchmark before). Pre-seeds a row, then measures repeated lookups.
// ---------------------------------------------------------------------------

func BenchmarkSQLiteStorage_GetCachedTranslation(b *testing.B) {
	storage := newBenchSQLite(b)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()
	seed := &TranslationCache{
		ID:             "bench-get-seed",
		SourceText:     benchSourceText,
		TargetText:     "Бенчмарк превод",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		AccessCount:    0,
		LastAccessedAt: now,
	}
	if err := storage.CacheTranslation(ctx, seed); err != nil {
		b.Fatalf("seed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		got, err := storage.GetCachedTranslation(ctx, benchSourceText, "en", "sr", "deepseek", "deepseek-chat")
		if err != nil {
			b.Fatalf("get: %v", err)
		}
		if got == nil {
			b.Fatal("expected cache hit, got miss")
		}
	}
}

func newBenchSQLite(tb testing.TB) *SQLiteStorage {
	tb.Helper()
	dbPath := filepath.Join(tb.TempDir(), "p6bench.db")
	storage, err := NewSQLiteStorage(&Config{Type: "sqlite", Database: dbPath})
	if err != nil {
		tb.Fatalf("NewSQLiteStorage: %v", err)
	}
	return storage
}

// ---------------------------------------------------------------------------
// Stress (§11.4.85) — concurrent contention on the SQLite cache. N goroutines
// concurrently put+get distinct tuples; asserts every write is readable with the
// correct target text and no panic/corruption. Run with -race to prove the
// concurrent path is data-race clean.
// ---------------------------------------------------------------------------

func TestStress_SQLiteCache_ConcurrentPutGet(t *testing.T) {
	storage := newBenchSQLite(t)
	defer storage.Close()
	// Single SQLite file connection must serialize writers cleanly.
	ctx := context.Background()

	const (
		goroutines  = 16 // ≥10 per §11.4.85 concurrent-contention floor
		perGoroutine = 40
	)

	var (
		wg        sync.WaitGroup
		writeErrs int64
		readErrs  int64
		mismatch  int64
		hits      int64
	)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			now := time.Now()
			for i := 0; i < perGoroutine; i++ {
				src := fmt.Sprintf("stress source g%d i%d", gid, i)
				want := fmt.Sprintf("превод g%d i%d", gid, i)
				c := &TranslationCache{
					ID:             fmt.Sprintf("stress-%d-%d", gid, i),
					SourceText:     src,
					TargetText:     want,
					SourceLanguage: "en",
					TargetLanguage: "sr",
					Provider:       "deepseek",
					Model:          "deepseek-chat",
					CreatedAt:      now,
					AccessCount:    0,
					LastAccessedAt: now,
				}
				if err := storage.CacheTranslation(ctx, c); err != nil {
					atomic.AddInt64(&writeErrs, 1)
					continue
				}
				got, err := storage.GetCachedTranslation(ctx, src, "en", "sr", "deepseek", "deepseek-chat")
				if err != nil {
					atomic.AddInt64(&readErrs, 1)
					continue
				}
				if got == nil {
					atomic.AddInt64(&mismatch, 1)
					continue
				}
				atomic.AddInt64(&hits, 1)
				if got.TargetText != want {
					atomic.AddInt64(&mismatch, 1)
				}
			}
		}(g)
	}
	wg.Wait()

	if writeErrs != 0 {
		t.Fatalf("concurrent cache writes failed: %d errors", writeErrs)
	}
	if readErrs != 0 {
		t.Fatalf("concurrent cache reads failed: %d errors", readErrs)
	}
	if mismatch != 0 {
		t.Fatalf("concurrent cache returned wrong/missing translations: %d mismatches", mismatch)
	}
	wantHits := int64(goroutines * perGoroutine)
	if hits != wantHits {
		t.Fatalf("expected %d successful round-trips, got %d", wantHits, hits)
	}
}

// ---------------------------------------------------------------------------
// Chaos (§11.4.85) — backend made unavailable (Close()d) mid-flight. Cache
// operations against a closed SQLite backend MUST surface an error gracefully
// (no panic, no silent success), proving the storage layer degrades cleanly
// rather than crashing the translator when the DB handle is gone.
// ---------------------------------------------------------------------------

func TestChaos_SQLiteCache_ClosedBackendGraceful(t *testing.T) {
	storage := newBenchSQLite(t)
	ctx := context.Background()
	now := time.Now()

	// Establish a healthy baseline write so the path is known-good first.
	healthy := &TranslationCache{
		ID:             "chaos-baseline",
		SourceText:     "chaos baseline",
		TargetText:     "хаос основа",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		CreatedAt:      now,
		AccessCount:    0,
		LastAccessedAt: now,
	}
	if err := storage.CacheTranslation(ctx, healthy); err != nil {
		t.Fatalf("baseline write should succeed: %v", err)
	}

	// Inject the fault: backend goes away.
	if err := storage.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Writes against the dead backend must error, never panic, never succeed.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("CacheTranslation panicked on closed backend: %v", r)
			}
		}()
		err := storage.CacheTranslation(ctx, healthy)
		if err == nil {
			t.Fatal("expected error caching to a closed backend, got nil (silent success)")
		}
	}()

	// Reads against the dead backend must error, never panic.
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("GetCachedTranslation panicked on closed backend: %v", r)
			}
		}()
		_, err := storage.GetCachedTranslation(ctx, "chaos baseline", "en", "sr", "deepseek", "deepseek-chat")
		if err == nil {
			t.Fatal("expected error reading from a closed backend, got nil")
		}
	}()
}
