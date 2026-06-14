package progress

import (
	"strings"
	"testing"
	"time"
)

// Second-pass bug-hunt wave (pkg/progress). Reproduce-first (§11.4.115): the
// test fails on the current code, the source fix flips it green, and reverting
// the fix makes it FAIL again (mutation-proven).

// Bug D — formatDuration emits a TRAILING SPACE when the larger unit is
// non-zero but the smaller unit is exactly zero. formatTime(0, ...) returns ""
// (the empty smaller-unit), yet formatDuration unconditionally joins the two
// parts with " ", so an exact 5-hour duration renders as "5 hours " and an
// exact 2-minute duration as "2 minutes ".
//
// These strings are surfaced verbatim to the dashboard via the
// TranslationProgress.EstimatedETA / .ElapsedTime JSON fields, so the user sees
// "Time remaining: 5 hours " with stray trailing whitespace.
//
// FACT (captured on pre-fix code via a probe test):
//
//	d=5h0m0s  => "5 hours "
//	d=2m0s    => "2 minutes "
//	d=1h0m0s  => "1 hour "
//
// Correct behavior: no leading/trailing/double whitespace in any rendering.
func TestFormatDuration_NoTrailingSpace(t *testing.T) {
	cases := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"exact 5 hours", 5 * time.Hour, "5 hours"},
		{"exact 1 hour", 1 * time.Hour, "1 hour"},
		{"exact 2 minutes", 2 * time.Minute, "2 minutes"},
		{"exact 1 minute", 1 * time.Minute, "1 minute"},
		// Mixed (smaller unit non-zero) must be unchanged — no regression.
		{"5h30m", 5*time.Hour + 30*time.Minute, "5 hours 30 minutes"},
		{"1h30m", 1*time.Hour + 30*time.Minute, "1 hour 30 minutes"},
		{"2m15s", 2*time.Minute + 15*time.Second, "2 minutes 15 seconds"},
		{"45s", 45 * time.Second, "45 seconds"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDuration(tc.d)
			if got != tc.want {
				t.Fatalf("formatDuration(%v)=%q, want %q", tc.d, got, tc.want)
			}
			// Strong anti-bluff guard: never any edge/double whitespace.
			if got != strings.TrimSpace(got) {
				t.Fatalf("formatDuration(%v)=%q has leading/trailing whitespace", tc.d, got)
			}
			if strings.Contains(got, "  ") {
				t.Fatalf("formatDuration(%v)=%q has a double space", tc.d, got)
			}
		})
	}
}
