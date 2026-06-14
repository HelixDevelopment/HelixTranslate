package llm

import (
	"context"
	"strings"
	"testing"
)

// markerClient is a deterministic LLMClient stub whose output reveals exactly
// which input text it was asked to translate, so a test can detect when the
// cache serves the WRONG translation.
type markerClient struct{ calls int }

func (m *markerClient) Translate(_ context.Context, text string, _ string) (string, error) {
	m.calls++
	return "xlate(" + text + ")", nil
}
func (m *markerClient) GetProviderName() string { return "marker" }

// TestTranslate_CacheKeyCollision_ServesWrongTranslation is a reproduce-first
// guard (§11.4.115) for a real wrong-result bug: the in-memory translation cache
// key was fmt.Sprintf("%s:%s", text, contextStr). Because ":" can appear inside
// the text, two DIFFERENT (text, context) pairs can map to the SAME key —
// ("a:b","c") and ("a","b:c") both produce "a:b:c" — so the second request gets
// the first request's translation served from cache. Same collision class as the
// storage Redis cache-key bug fixed earlier in this codebase.
func TestTranslate_CacheKeyCollision_ServesWrongTranslation(t *testing.T) {
	stub := &markerClient{}
	tr := &LLMTranslator{
		BaseTranslator: NewBaseTranslator(TranslationConfig{}),
		provider:       "marker",
		client:         stub,
	}
	ctx := context.Background()

	// First pair: text "a:b" with context "c".
	r1, err := tr.Translate(ctx, "a:b", "c")
	if err != nil {
		t.Fatalf("translate pair1: %v", err)
	}
	if !strings.Contains(r1, "(a:b)") {
		t.Fatalf("pair1 should translate %q, got %q", "a:b", r1)
	}

	// Second pair: DIFFERENT text "a" with context "b:c". This is a distinct
	// input that must get its own translation ("xlate(a)"), NOT pair1's.
	r2, err := tr.Translate(ctx, "a", "b:c")
	if err != nil {
		t.Fatalf("translate pair2: %v", err)
	}

	if r2 == r1 || strings.Contains(r2, "(a:b)") {
		t.Errorf("CACHE-KEY COLLISION: input (text=%q,ctx=%q) was served pair1's translation %q "+
			"(a different (text,ctx) pair) — the cache key is not injective; a reader would get the "+
			"wrong translated text", "a", "b:c", r2)
	}
	if !strings.Contains(r2, "(a)") {
		t.Errorf("pair2 should translate %q to contain %q, got %q", "a", "(a)", r2)
	}
}

// TestMakeLLMCacheKey_Injective directly asserts the key builder is collision-free
// for the canonical ambiguous pairs (the mutation unit for the fix).
func TestMakeLLMCacheKey_Injective(t *testing.T) {
	pairs := []struct{ text, ctx string }{
		{"a:b", "c"},
		{"a", "b:c"},
		{"10:30", "foo"},
		{"10", "30:foo"},
		{"", "x:y"},
		{"x:y", ""},
	}
	seen := map[string][2]string{}
	for _, p := range pairs {
		k := makeLLMCacheKey(p.text, p.ctx)
		if prev, dup := seen[k]; dup && (prev[0] != p.text || prev[1] != p.ctx) {
			t.Errorf("collision: (%q,%q) and (%q,%q) both map to key %q",
				prev[0], prev[1], p.text, p.ctx, k)
		}
		seen[k] = [2]string{p.text, p.ctx}
	}
}
