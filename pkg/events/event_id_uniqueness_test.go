package events

import (
	"testing"
)

// TestEventID_Unique_BurstSameMicrosecond is a reproduce-first (§11.4.115) test
// for the event-ID collision defect. generateEventID formerly returned
// time.Now().Format("20060102150405.000000") — microsecond-precision wall-clock
// only. Multiple events created within the same microsecond (a tight Publish
// burst — the normal case for progress events) received IDENTICAL IDs, violating
// NewEvent's documented "unique ID" contract. Any consumer keying on Event.ID
// (de-dup, correlation, idempotency) would conflate distinct events.
//
// On the broken code this loop produces many duplicate IDs and FAILs.
func TestEventID_Unique_BurstSameMicrosecond(t *testing.T) {
	const n = 10000
	seen := make(map[string]int, n)
	for i := 0; i < n; i++ {
		ev := NewEvent(EventTranslationProgress, "p", nil)
		if ev.ID == "" {
			t.Fatalf("event %d had empty ID", i)
		}
		seen[ev.ID]++
	}
	dups := 0
	var example string
	for id, c := range seen {
		if c > 1 {
			dups += c - 1
			if example == "" {
				example = id
			}
		}
	}
	if dups != 0 {
		t.Fatalf("event IDs are NOT unique: %d duplicate IDs out of %d events "+
			"(%d distinct), e.g. id=%q seen %d times", dups, n, len(seen), example, seen[example])
	}
}
