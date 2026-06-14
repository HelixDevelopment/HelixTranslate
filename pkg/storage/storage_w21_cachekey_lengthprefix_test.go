package storage

import "testing"

// W21 — REAL bug: cache-key serialisation collision via a delimiter byte inside
// a field.
//
// cacheLookupHash and Redis makeCacheKey both serialised the lookup tuple by
// joining the five components with a single NUL ('\x00') separator and hashing.
// The code claimed this was "injection-proof" because NUL "cannot appear in
// normal text". That assumption is false: sourceText is free-form user content
// and provider/model are externally supplied, so a field can carry the delimiter
// and shift a neighbouring field's boundary. Two DISTINCT tuples then serialise
// to the SAME string -> same hash -> same cache row/key -> the WRONG cached
// translation is served (same defect class as the §W3 32-bit-hash collision and
// the §W20 ':'-delimiter injection, but via the NUL delimiter).
//
// Fixed by length-prefixing each field (encodeCacheTuple), which is unambiguous
// for ANY byte content. RED on the old NUL-join, GREEN on the length-prefix.
func TestW21_CacheLookupHash_NoDelimiterCollision(t *testing.T) {
	// Distinct (provider, model) tuples where the delimiter byte sits inside a
	// field so a bare-delimiter join would merge their boundaries.
	a := cacheLookupHash("x", "en", "fr", "p\x00q", "m")
	b := cacheLookupHash("x", "en", "fr", "p", "q\x00m")
	if a == b {
		t.Fatalf("NUL-delimiter collision in cacheLookupHash: distinct tuples share hash %s\n"+
			"  A provider=%q model=%q\n  B provider=%q model=%q", a, "p\x00q", "m", "p", "q\x00m")
	}

	// Realistic field: free-form sourceText carrying the delimiter, splitting
	// against a different (model, sourceText) boundary.
	c := cacheLookupHash("y\x00z", "en", "fr", "p", "m")
	d := cacheLookupHash("z", "en", "fr", "p", "m\x00y")
	if c == d {
		t.Fatalf("NUL-delimiter collision via sourceText boundary: %s", c)
	}
}

func TestW21_MakeCacheKey_NoDelimiterCollision(t *testing.T) {
	r := &RedisStorage{} // makeCacheKey needs no client.

	keyA := r.makeCacheKey("x", "en", "fr", "p\x00q", "m")
	keyB := r.makeCacheKey("x", "en", "fr", "p", "q\x00m")
	if keyA == keyB {
		t.Fatalf("NUL-delimiter collision in makeCacheKey: distinct tuples share key %q", keyA)
	}

	// Length-prefix encoding must also keep the §W20 ':'-injection class collision-free.
	keyC := r.makeCacheKey("Дом", "ru", "en", "ollama", "llama3:8b")
	keyD := r.makeCacheKey("Дом", "ru", "en", "ollama:llama3", "8b")
	if keyC == keyD {
		t.Fatalf("':'-injection collision regressed in makeCacheKey: %q", keyC)
	}
}

// Sanity: equal tuples still map to the SAME key (idempotency preserved) and
// length-prefix encoding is deterministic.
func TestW21_EncodeCacheTuple_StableAndDistinct(t *testing.T) {
	if encodeCacheTuple("a", "b", "c") != encodeCacheTuple("a", "b", "c") {
		t.Fatal("encodeCacheTuple not deterministic")
	}
	// ("a","bc") vs ("ab","c") must differ.
	if encodeCacheTuple("a", "bc") == encodeCacheTuple("ab", "c") {
		t.Fatal("encodeCacheTuple ambiguous: (a,bc) collides with (ab,c)")
	}
	// Same tuple via cacheLookupHash is idempotent (the cache-hit contract).
	if cacheLookupHash("t", "en", "fr", "openai", "gpt-4") !=
		cacheLookupHash("t", "en", "fr", "openai", "gpt-4") {
		t.Fatal("cacheLookupHash not idempotent for equal tuples")
	}
}
