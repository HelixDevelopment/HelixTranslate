package bridge

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.translator/internal/verifier/selection"
)

// TestBridge_EnsembleFactory_YieldsProviderDiverse proves the EnsembleFactory
// closure sources the provider-diverse verified translators (one per distinct
// provider, strongest-first) — the R-1 ensemble redirect contract. With two
// providers each holding two verified models, the factory MUST yield exactly two
// translators (the strongest of each provider), proving diversity is preserved
// and weaker same-provider models are dropped.
func TestBridge_EnsembleFactory_YieldsProviderDiverse(t *testing.T) {
	// Model IDs MUST be members of each provider client's static whitelist
	// (the provider-diverse path builds real clients via NewOpenAIClient): use
	// gpt-4o (valid OpenAI) and deepseek-chat (valid DeepSeek).
	b := newTestBridge(keyForAll(),
		verifiedModel("deepseek-chat", "deepseek", "DeepSeek Chat", 0.95),
		verifiedModel("deepseek-coder", "deepseek", "DeepSeek Coder", 0.50),
		verifiedModel("gpt-4o", "openai", "GPT-4o", 0.90),
		verifiedModel("gpt-4", "openai", "GPT-4", 0.40),
	)

	factory := b.EnsembleFactory(selection.TaskRequirements{TargetLang: "es"})
	trs, err := factory(context.Background())
	if err != nil {
		t.Fatalf("EnsembleFactory closure returned error: %v", err)
	}
	if len(trs) != 2 {
		t.Fatalf("expected 2 provider-diverse translators (one per distinct provider), got %d", len(trs))
	}

	// The two translators MUST be the two distinct providers (order = score desc:
	// deepseek 0.95 then openai 0.90), each reporting its provider via GetName().
	var names []string
	for _, tr := range trs {
		if tr == nil {
			t.Fatal("EnsembleFactory yielded a nil translator")
		}
		names = append(names, tr.GetName())
	}
	joined := strings.Join(names, ",")
	for _, want := range []string{"deepseek", "openai"} {
		if !strings.Contains(joined, want) {
			t.Errorf("provider-diverse set missing strongest model of provider %q (got %v)", want, names)
		}
	}
}

// TestBridge_BestTranslatorFunc_YieldsStrongest proves the BestTranslatorFunc
// closure yields the single strongest verified translator for the task (the
// non-ensemble single-translator redirect contract).
func TestBridge_BestTranslatorFunc_YieldsStrongest(t *testing.T) {
	b := newTestBridge(keyForAll(),
		verifiedModel("deepseek-chat", "deepseek", "DeepSeek Chat", 0.95),
		verifiedModel("gpt-4o", "openai", "GPT-4o", 0.90),
	)

	fn := b.BestTranslatorFunc(selection.TaskRequirements{TargetLang: "es"})
	tr, err := fn(context.Background())
	if err != nil {
		t.Fatalf("BestTranslatorFunc closure returned error: %v", err)
	}
	if tr == nil {
		t.Fatal("BestTranslatorFunc yielded a nil translator")
	}
	if got := tr.GetName(); !strings.Contains(got, "deepseek") {
		t.Errorf("expected strongest translator to be deepseek-backed, got %q", got)
	}
}

// TestBridge_EnsembleFactory_EmptyRegistryHonestError proves the closure
// propagates the bridge's honest no-models error (R2 — never a silent local
// fallback).
func TestBridge_EnsembleFactory_EmptyRegistryHonestError(t *testing.T) {
	b := newTestBridge(keyForAll()) // no models registered
	factory := b.EnsembleFactory(selection.TaskRequirements{})
	if _, err := factory(context.Background()); err == nil {
		t.Fatal("expected honest error from EnsembleFactory on empty registry, got nil (silent fallback is forbidden)")
	}
}
