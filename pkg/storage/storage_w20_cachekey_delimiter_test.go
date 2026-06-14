package storage

import "testing"

// W20 — REAL bug (delimiter-injection collision in Redis makeCacheKey).
//
// makeCacheKey builds "cache:<srcLang>:<tgtLang>:<provider>:<model>:<hash(text)>"
// by joining the metadata fields with a RAW ':' delimiter and NO escaping. When
// any field contains a ':' the field boundaries shift, so two DISTINCT
// (srcLang,tgtLang,provider,model) tuples concatenate to the SAME key — a Redis
// GET on tuple B returns the value cached for tuple A: the WRONG translation.
//
// This is the SAME defect class as the already-fixed source-text hash collision
// (W3), but via the metadata fields rather than the text. It is realistic in
// THIS project because Ollama model identifiers natively contain ':'
// (e.g. "llama3:8b", "qwen2.5:7b") — model is a free-form field placed raw into
// the key.
//
// RED on current code: the two tuples below differ ONLY in the provider/model
// split yet produce an identical key.
func TestW20_MakeCacheKey_NoDelimiterInjectionCollision(t *testing.T) {
	r := &RedisStorage{} // makeCacheKey needs no client.

	// Two genuinely distinct tuples (different provider AND different model).
	keyA := r.makeCacheKey("Дом", "ru", "en", "ollama", "llama3:8b")
	keyB := r.makeCacheKey("Дом", "ru", "en", "ollama:llama3", "8b")

	if keyA == keyB {
		t.Fatalf("delimiter-injection collision: distinct (provider,model) tuples share a cache key\n"+
			"  A provider=%q model=%q\n  B provider=%q model=%q\n  both => %q\n"+
			"a GET on tuple B would return tuple A's cached translation (WRONG result)",
			"ollama", "llama3:8b", "ollama:llama3", "8b", keyA)
	}

	// A second realistic instance: a model whose colon collides with a sibling.
	keyC := r.makeCacheKey("Дом", "ru", "en", "ollama", "qwen2.5:7b")
	keyD := r.makeCacheKey("Дом", "ru", "en", "ollama", "qwen2.5:7b:extra")
	keyE := r.makeCacheKey("Дом:extra", "ru", "en", "ollama", "qwen2.5:7b") // injection via text? text is hashed, so safe — keep as control
	if keyC == keyD {
		t.Fatalf("model colon-suffix collision: %q == %q", keyC, keyD)
	}
	_ = keyE // text is hashed (collision-resistant) so keyE differs from keyC by hash; control only.
}
