package distributed

import (
	"testing"
	"time"
)

// TestResultCache_UpdateExistingKeyWhenFull_NoSpuriousEviction proves that
// updating an ALREADY-PRESENT key in a full cache must NOT evict an unrelated
// entry. Updating an existing key does not grow the map, so the size guard in
// Set() must treat it as an in-place overwrite, not as an insertion that needs
// to make room.
//
// RED on the pre-fix code: Set() unconditionally runs removeOldest() when
// len(cache) >= maxSize, dropping an innocent entry even though the write only
// overwrites an existing slot — silently shrinking the effective cache and
// throwing away a valid cached translation.
func TestResultCache_UpdateExistingKeyWhenFull_NoSpuriousEviction(t *testing.T) {
	cfg := DefaultPerformanceConfig()
	cfg.MaxCacheSize = 3
	cfg.CacheTTL = time.Hour              // long, so nothing is "expired"
	cfg.CacheCleanupInterval = time.Hour // keep the cleanup goroutine quiet

	rc := NewResultCache(cfg)

	// Fill cache to capacity with three distinct, non-expired entries.
	rc.Set("k0", "v0")
	rc.Set("k1", "v1")
	rc.Set("k2", "v2")

	if got := cacheLen(rc); got != 3 {
		t.Fatalf("precondition: expected 3 entries, got %d", got)
	}

	// Update an EXISTING key. This must not require evicting anything because
	// the map size does not change.
	rc.Set("k1", "v1-updated")

	// All three original keys MUST still be retrievable.
	for _, k := range []string{"k0", "k1", "k2"} {
		if _, ok := rc.Get(k); !ok {
			t.Errorf("key %q was spuriously evicted when updating an existing key in a full cache", k)
		}
	}

	if got, ok := rc.Get("k1"); !ok || got != "v1-updated" {
		t.Errorf("k1 value: got (%q, %v), want (\"v1-updated\", true)", got, ok)
	}

	if got := cacheLen(rc); got != 3 {
		t.Errorf("cache size after in-place update: got %d, want 3", got)
	}
}

func cacheLen(rc *ResultCache) int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.cache)
}
