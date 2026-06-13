package storage

import "testing"

// W3 slice — REAL bug: the Redis cache-key hash (hashString) is a 32-bit
// polynomial hash h = h*31 + c. Polynomial-31 hashes have trivial collisions
// (e.g. "Aa" vs "BB", "AaAa" vs "BBBB") AND overflow at 2^32. A collision
// means makeCacheKey() produces the SAME Redis key for two DISTINCT source
// texts (same lang/provider/model) — so caching one text's translation and
// then looking up the colliding text returns the WRONG translation to the end
// user. This is a correctness defect, not merely a theoretical hash weakness.
//
// RED on the pre-fix 31-polynomial hash; GREEN after the hash is made
// collision-resistant. Behavioral, anti-bluff: asserts a real wrong-key /
// wrong-translation outcome, not just hash internals.

// TestHashString_NoKnownPolynomialCollision proves the hash does not collide
// on the canonical polynomial-31 collision pairs.
func TestHashString_NoKnownPolynomialCollision(t *testing.T) {
	collisionPairs := [][2]string{
		{"Aa", "BB"},
		{"AaAa", "BBBB"},
		{"AaBB", "BBAa"},
	}
	for _, p := range collisionPairs {
		if hashString(p[0]) == hashString(p[1]) {
			t.Errorf("hash collision: hashString(%q)==hashString(%q)==%s — distinct source texts share a cache key",
				p[0], p[1], hashString(p[0]))
		}
	}
}

// TestMakeCacheKey_NoCollisionServesWrongTranslation proves the END-USER
// consequence: two distinct source texts must NOT map to the same cache key,
// because that would serve text A's translation when text B is requested.
func TestMakeCacheKey_NoCollisionServesWrongTranslation(t *testing.T) {
	r := &RedisStorage{}
	keyA := r.makeCacheKey("Aa", "en", "sr", "openai", "gpt-4")
	keyB := r.makeCacheKey("BB", "en", "sr", "openai", "gpt-4")
	if keyA == keyB {
		t.Fatalf("distinct source texts %q and %q collide on cache key %q — "+
			"looking up %q would return %q's cached translation (WRONG translation served)",
			"Aa", "BB", keyA, "BB", "Aa")
	}
}
