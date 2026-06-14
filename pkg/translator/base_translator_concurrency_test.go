package translator

import (
	"sync"
	"testing"
)

// TestBaseTranslator_ConcurrentCacheAndStatsAreRaceFree reproduces a concurrency
// defect in the exported BaseTranslator: its cache (a plain map[string]string)
// and stats counters are mutated from CheckCache / AddToCache / UpdateStats with
// NO synchronization. The Translator base is the shared building block for
// concurrent translation workers (the CLI runs with -concurrency / -workers), so
// concurrent callers either trigger Go's runtime "fatal error: concurrent map
// writes" (a hard crash that aborts the whole run) or a stats data race.
//
// The sibling pkg/translator/llm copy of BaseTranslator already carries a
// sync.RWMutex "guards cache and stats for concurrent Translate callers" — this
// root-level copy was left unguarded, a latent crash for any concurrent user of
// the public API.
//
// Run with -race to surface the data race deterministically; without -race the
// concurrent map writes still fatal-crash the test binary under enough load.
func TestBaseTranslator_ConcurrentCacheAndStatsAreRaceFree(t *testing.T) {
	bt := NewBaseTranslator(TranslationConfig{Provider: "mock"})

	const workers = 16
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := string(rune('a'+id)) + string(rune('0'+(i%10)))
				if _, found := bt.CheckCache(key); !found {
					bt.AddToCache(key, "translated-"+key)
					bt.UpdateStats(true)
				}
			}
		}(w)
	}
	wg.Wait()

	// If we reach here without a fatal "concurrent map writes" crash and the
	// race detector reports nothing, the base translator is concurrency-safe.
	stats := bt.GetStats()
	if stats.Total < 0 {
		t.Fatalf("impossible negative total: %d", stats.Total)
	}
}
