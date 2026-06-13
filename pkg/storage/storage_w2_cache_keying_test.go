package storage

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// W2 slice: REAL, anti-bluff tests raising pkg/storage coverage.
//
// These tests deliberately avoid mocks-as-real (per §11.4.27): the SQLite
// backend is a REAL backend (in-memory / temp-file sqlite3 driver), and the
// Redis cache-key construction is PURE deterministic logic that needs no
// daemon. Real Redis round-trips are gated behind a reachability SKIP
// (§11.4.3) — never a fake pass.

// ---------------------------------------------------------------------------
// Redis cache-key construction — PURE, deterministic, no daemon required.
// makeCacheKey + hashString were 0% covered. These assert the exact
// (text, context) keying contract: identical inputs => identical key;
// any differing component => different key. A stubbed makeCacheKey returning
// a constant would fail TestRedisMakeCacheKey_DistinctPerComponent.
// ---------------------------------------------------------------------------

func TestRedisHashString_Deterministic(t *testing.T) {
	// Same input MUST hash identically across calls.
	a := hashString("Здравствуйте, мир")
	b := hashString("Здравствуйте, мир")
	if a != b {
		t.Fatalf("hashString not deterministic: %q != %q", a, b)
	}
	// Output is the documented 8-hex-digit form.
	if len(a) != 8 {
		t.Fatalf("expected 8-char hex hash, got %q (len %d)", a, len(a))
	}
	for _, c := range a {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("hash %q contains non-hex char %q", a, c)
		}
	}
}

func TestRedisHashString_DistinctInputs(t *testing.T) {
	// Different source texts MUST (in the common case) hash differently.
	// These two short strings are known-distinct under h = h*31 + c.
	h1 := hashString("hello")
	h2 := hashString("world")
	if h1 == h2 {
		t.Fatalf("distinct inputs collided: hashString(hello)=%s hashString(world)=%s", h1, h2)
	}
	// Empty string has a well-defined, stable hash.
	if got := hashString(""); got != "00000000" {
		t.Fatalf("empty-string hash changed: got %q want 00000000", got)
	}
}

func TestRedisHashString_KnownVector(t *testing.T) {
	// Lock the algorithm (h = h*31 + rune) against silent change.
	// "ab": ('a'=97) -> 97; 97*31 + 98 = 3105 = 0x00000c21
	if got := hashString("ab"); got != "00000c21" {
		t.Fatalf("hashString(\"ab\") = %q, want 00000c21 (algorithm changed?)", got)
	}
}

func TestRedisMakeCacheKey_DistinctPerComponent(t *testing.T) {
	r := &RedisStorage{} // makeCacheKey needs no client.

	base := r.makeCacheKey("text", "en", "sr", "openai", "gpt-4")

	// Documented format: cache:<srcLang>:<tgtLang>:<provider>:<model>:<hash(text)>
	wantPrefix := fmt.Sprintf("cache:en:sr:openai:gpt-4:%s", hashString("text"))
	if base != wantPrefix {
		t.Fatalf("cache key format mismatch:\n got  %q\n want %q", base, wantPrefix)
	}

	// Each component must influence the key — a stubbed key generator
	// returning a constant would fail every one of these.
	cases := []struct {
		name string
		key  string
	}{
		{"diff text", r.makeCacheKey("other", "en", "sr", "openai", "gpt-4")},
		{"diff srcLang", r.makeCacheKey("text", "ru", "sr", "openai", "gpt-4")},
		{"diff tgtLang", r.makeCacheKey("text", "en", "de", "openai", "gpt-4")},
		{"diff provider", r.makeCacheKey("text", "en", "sr", "anthropic", "gpt-4")},
		{"diff model", r.makeCacheKey("text", "en", "sr", "openai", "claude-3")},
	}
	for _, c := range cases {
		if c.key == base {
			t.Errorf("changing %s did not change cache key (both %q)", c.name, base)
		}
	}
}

func TestRedisMakeCacheKey_StableForSameInputs(t *testing.T) {
	r := &RedisStorage{}
	k1 := r.makeCacheKey("Война и мир", "ru", "en", "deepseek", "deepseek-chat")
	k2 := r.makeCacheKey("Война и мир", "ru", "en", "deepseek", "deepseek-chat")
	if k1 != k2 {
		t.Fatalf("identical inputs produced different keys: %q vs %q", k1, k2)
	}
}

// ---------------------------------------------------------------------------
// Redis real round-trips — gated behind reachability SKIP (§11.4.3).
// Runs ONLY when a real Redis daemon is reachable via REDIS_TEST_HOST.
// Never a mock; honest SKIP when the daemon is absent.
// ---------------------------------------------------------------------------

func redisReachableOrSkip(t *testing.T) *RedisStorage {
	t.Helper()
	cfg := getRedisTestConfig(t)
	if cfg == nil {
		t.Skip("REAL Redis daemon not reachable (set REDIS_TEST_HOST) — skipping daemon-backed round-trip") // SKIP-OK: #requires-infra-port
		return nil
	}
	st, err := NewRedisStorage(cfg, time.Minute)
	if err != nil {
		t.Skipf("Redis configured but connect failed: %v — skipping", err) // SKIP-OK: #requires-infra-port
		return nil
	}
	return st
}

func TestRedisCacheRoundTrip_RealDaemon(t *testing.T) {
	st := redisReachableOrSkip(t)
	if st == nil {
		return
	}
	defer st.Close()
	ctx := context.Background()

	c := &TranslationCache{
		ID:             "w2-redis-rt",
		SourceText:     "Good morning",
		TargetText:     "Добро јутро",
		SourceLanguage: "en",
		TargetLanguage: "sr",
		Provider:       "openai",
		Model:          "gpt-4",
		CreatedAt:      time.Now(),
	}
	if err := st.CacheTranslation(ctx, c); err != nil {
		t.Fatalf("CacheTranslation: %v", err)
	}

	got, err := st.GetCachedTranslation(ctx, c.SourceText, c.SourceLanguage, c.TargetLanguage, c.Provider, c.Model)
	if err != nil {
		t.Fatalf("GetCachedTranslation: %v", err)
	}
	if got == nil {
		t.Fatal("expected cache hit, got nil")
	}
	if got.TargetText != "Добро јутро" {
		t.Fatalf("round-trip target mismatch: got %q want %q", got.TargetText, "Добро јутро")
	}
	// Different (text,context) must miss.
	miss, err := st.GetCachedTranslation(ctx, "Different text", c.SourceLanguage, c.TargetLanguage, c.Provider, c.Model)
	if err != nil {
		t.Fatalf("GetCachedTranslation(miss): %v", err)
	}
	if miss != nil {
		t.Fatalf("expected miss for unknown text, got %+v", miss)
	}
}
