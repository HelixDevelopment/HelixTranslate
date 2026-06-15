package verification

import (
	"context"
	"os"
	"testing"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// stubFactoryTranslator is a zero-network stub implementing the 4-method
// translator.Translator interface. Translate returns a canned
// verification-shaped response so it can drive verifyWithLLM end-to-end without
// any LLM call.
type stubFactoryTranslator struct {
	name     string
	response string
}

func (s *stubFactoryTranslator) Translate(_ context.Context, _ string, _ string) (string, error) {
	return s.response, nil
}

func (s *stubFactoryTranslator) TranslateWithProgress(
	_ context.Context, _ string, _ string, _ *events.EventBus, _ string,
) (string, error) {
	return s.response, nil
}

func (s *stubFactoryTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (s *stubFactoryTranslator) GetName() string { return s.name }

// cannedVerification is a minimal valid verification-format response (the
// polisher's parseVerificationResponse parses it). The actual content is
// irrelevant for the routing/ensemble assertions.
const cannedVerification = `SPIRIT_SCORE: 0.90
LANGUAGE_SCORE: 0.90
CONTEXT_SCORE: 0.90
VOCABULARY_SCORE: 0.90

ISSUES:

POLISHED_TEXT:
UNCHANGED

EXPLANATION:
stub`

// TestNewBookPolisherWithFactory_InjectedEnsemble is the §11.4.115 polarity
// guard. With an injected factory returning translators for 3 distinct
// providers, the polisher MUST:
//
//   - build the interface-typed ensemble map keyed by GetName() with EXACTLY
//     those 3 providers, NOT the concrete `translators` map; and
//   - route verifyWithLLM through the injected ensemble (proving the read site
//     uses the interface map, not the concrete one).
//
// Polarity switch RED_MODE=1: emulate the §1.1 mutation where the read site
// IGNORES the injected ensemble and falls back to the concrete `translators`
// map (which is nil/empty when a factory is injected). Under that mutation the
// guard MUST FAIL (no translator resolves → verifyWithLLM errors and the
// ensemble keys are absent), proving the test genuinely catches the regression.
func TestNewBookPolisherWithFactory_InjectedEnsemble(t *testing.T) {
	redMode := os.Getenv("RED_MODE") == "1"

	wantProviders := []string{"alpha", "beta", "gamma"}

	factory := func(_ context.Context) ([]translator.Translator, error) {
		return []translator.Translator{
			&stubFactoryTranslator{name: "alpha", response: cannedVerification},
			&stubFactoryTranslator{name: "beta", response: cannedVerification},
			&stubFactoryTranslator{name: "gamma", response: cannedVerification},
		}, nil
	}

	config := PolishingConfig{
		// Deliberately empty Providers/TranslationConfigs: the bridge case
		// supplies the provider set via the factory, not config.
		MinConsensus:     1,
		VerifySpirit:     true,
		VerifyLanguage:   true,
		VerifyContext:    true,
		VerifyVocabulary: true,
	}

	bp, err := NewBookPolisherWithFactory(context.Background(), config, events.NewEventBus(), "sess", factory)
	if err != nil {
		t.Fatalf("NewBookPolisherWithFactory returned error: %v", err)
	}

	// RED_MODE mutation: blank the ensemble so resolveTranslator falls back to
	// the (empty) concrete map — the exact behaviour a "ignore injected
	// ensemble" regression would exhibit.
	if redMode {
		bp.ensemble = nil
	}

	// (1) Ensemble map shape: exactly the 3 injected providers, keyed by GetName().
	if len(bp.ensemble) != len(wantProviders) {
		t.Fatalf("ensemble must hold exactly %d injected translators, got %d (RED_MODE=%v)",
			len(wantProviders), len(bp.ensemble), redMode)
	}
	for _, p := range wantProviders {
		if _, ok := bp.ensemble[p]; !ok {
			t.Fatalf("ensemble missing injected provider %q (RED_MODE=%v)", p, redMode)
		}
	}

	// providerSet() must reflect the injected providers, not config.Providers.
	got := bp.providerSet()
	if len(got) != len(wantProviders) {
		t.Fatalf("providerSet() = %v, want the %d injected providers", got, len(wantProviders))
	}

	// (2) Routing proof: verifyWithLLM resolves the INJECTED translator (interface
	// map) rather than the concrete map. The stub returns a parseable response,
	// so a successful, non-nil verification stamped with the provider proves the
	// route went through the ensemble.
	for _, p := range wantProviders {
		v, vErr := bp.verifyWithLLM(context.Background(), p, "orig", "trans", "loc")
		if vErr != nil {
			t.Fatalf("verifyWithLLM(%q) errored — injected ensemble was not used: %v (RED_MODE=%v)",
				p, vErr, redMode)
		}
		if v == nil || v.Provider != p {
			t.Fatalf("verifyWithLLM(%q) did not route to injected translator (RED_MODE=%v)", p, redMode)
		}
	}
}

// TestNewBookPolisher_DefaultPathUnchanged proves the nil-factory path is
// behaviour-preserving: NewBookPolisher (which delegates to the WithFactory
// variant with a nil factory) still builds the concrete `translators` map from
// config.Providers/TranslationConfigs via llm.NewLLMTranslator, and leaves the
// ensemble map nil. The "mock" provider is a real, network-free LLM provider.
func TestNewBookPolisher_DefaultPathUnchanged(t *testing.T) {
	config := PolishingConfig{
		Providers:    []string{"mock"},
		MinConsensus: 1,
		TranslationConfigs: map[string]translator.TranslationConfig{
			"mock": {
				Provider:   "mock",
				SourceLang: "ru",
				TargetLang: "sr",
			},
		},
	}

	bp, err := NewBookPolisher(config, events.NewEventBus(), "sess")
	if err != nil {
		t.Fatalf("NewBookPolisher (default path) returned error: %v", err)
	}

	// ensemble must be nil on the default path — no factory injected.
	if bp.ensemble != nil {
		t.Fatalf("default path must leave ensemble nil, got %v", bp.ensemble)
	}

	// concrete map built exactly as before, keyed on config.Providers.
	if len(bp.translators) != 1 {
		t.Fatalf("default path must build concrete translators map for config.Providers, got %d entries",
			len(bp.translators))
	}
	if _, ok := bp.translators["mock"]; !ok {
		t.Fatalf("default path concrete map missing 'mock' provider")
	}

	// providerSet() falls back to config.Providers on the default path.
	got := bp.providerSet()
	if len(got) != 1 || got[0] != "mock" {
		t.Fatalf("providerSet() on default path = %v, want [mock]", got)
	}

	// resolveTranslator routes through the concrete map on the default path.
	if _, ok := bp.resolveTranslator("mock"); !ok {
		t.Fatalf("resolveTranslator('mock') must resolve via concrete map on default path")
	}

	// Missing-config error path unchanged.
	badConfig := PolishingConfig{
		Providers:          []string{"absent"},
		TranslationConfigs: map[string]translator.TranslationConfig{},
	}
	if _, err := NewBookPolisher(badConfig, events.NewEventBus(), "sess"); err == nil {
		t.Fatalf("default path must still error on missing translation config")
	}
}
