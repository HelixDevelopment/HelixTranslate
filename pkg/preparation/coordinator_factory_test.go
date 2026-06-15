package preparation

import (
	"context"
	"os"
	"testing"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// §11.4.115 polarity switch: RED_MODE=1 reproduces the defect (the injected
// EnsembleTranslatorFactory is IGNORED and the built-in loop runs) so the guard
// FAILs on the broken behaviour; RED_MODE=0 (default) is the standing GREEN
// regression guard asserting the injected factory's providers are used verbatim.
//
// To capture the RED proof on the pre-fix-equivalent behaviour, run:
//
//	RED_MODE=1 go test -run TestNewPreparationCoordinatorWithFactory_InjectedProvidersUsed ./pkg/preparation/
//
// which builds the coordinator via the built-in path (factory ignored) and the
// assertion that len(providers)==3 FAILs (built-in path yields 0 valid
// providers in a no-network test environment → constructor errors).
func injectedFactoryRedModeOn() bool { return os.Getenv("RED_MODE") == "1" }

// stubFactoryTranslator is a zero-network translator.Translator used purely to
// count how many providers the coordinator received from the injected factory.
// It reuses the same interface surface MockTranslator implements; a dedicated
// tiny stub keeps this file self-contained.
type stubFactoryTranslator struct{ name string }

func (s *stubFactoryTranslator) Translate(_ context.Context, _ string, _ string) (string, error) {
	return "{}", nil
}

func (s *stubFactoryTranslator) TranslateWithProgress(
	ctx context.Context, text string, c string, _ *events.EventBus, _ string,
) (string, error) {
	return s.Translate(ctx, text, c)
}

func (s *stubFactoryTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}

func (s *stubFactoryTranslator) GetName() string { return s.name }

// TestNewPreparationCoordinatorWithFactory_InjectedProvidersUsed proves the
// injectable seam: an injected factory returning 3 stub translators makes the
// coordinator's providers EXACTLY those 3.
func TestNewPreparationCoordinatorWithFactory_InjectedProvidersUsed(t *testing.T) {
	stubs := []translator.Translator{
		&stubFactoryTranslator{name: "stub-a"},
		&stubFactoryTranslator{name: "stub-b"},
		&stubFactoryTranslator{name: "stub-c"},
	}

	var factory EnsembleTranslatorFactory = func(_ context.Context) ([]translator.Translator, error) {
		return stubs, nil
	}

	// RED_MODE=1 mutation: ignore the injected factory and use the built-in loop
	// (config has no providers + no keys → 0 valid providers → error). The
	// assertions below then FAIL, proving the guard catches a seam that does not
	// actually consult the injected factory.
	if injectedFactoryRedModeOn() {
		factory = nil
	}

	coordinator, err := NewPreparationCoordinatorWithFactory(
		context.Background(),
		PreparationConfig{SourceLanguage: "en", TargetLanguage: "es"},
		factory,
	)
	if err != nil {
		t.Fatalf("unexpected error building coordinator with injected factory: %v", err)
	}
	if coordinator == nil {
		t.Fatal("expected coordinator, got nil")
	}

	if got := len(coordinator.providers); got != 3 {
		t.Fatalf("expected 3 injected providers, got %d", got)
	}

	for i, want := range []string{"stub-a", "stub-b", "stub-c"} {
		if got := coordinator.providers[i].GetName(); got != want {
			t.Errorf("provider[%d]: expected %q, got %q", i, want, got)
		}
	}
}

// TestNewPreparationCoordinatorWithFactory_EmptyFactoryHonestError proves a
// factory returning 0 translators yields the SAME honest error as the built-in
// path's empty-providers case.
func TestNewPreparationCoordinatorWithFactory_EmptyFactoryHonestError(t *testing.T) {
	var factory EnsembleTranslatorFactory = func(_ context.Context) ([]translator.Translator, error) {
		return nil, nil
	}

	coordinator, err := NewPreparationCoordinatorWithFactory(
		context.Background(),
		PreparationConfig{SourceLanguage: "en", TargetLanguage: "es"},
		factory,
	)
	if err == nil {
		t.Fatal("expected 'no valid LLM providers available' error, got nil")
	}
	if coordinator != nil {
		t.Fatalf("expected nil coordinator on empty factory, got %#v", coordinator)
	}
	if err.Error() != "no valid LLM providers available" {
		t.Errorf("expected honest error %q, got %q", "no valid LLM providers available", err.Error())
	}
}

// TestNewPreparationCoordinator_DefaultPathUnchanged proves the nil-factory
// default path reproduces today's built-in construction exactly: with a valid
// provider + key the constructor succeeds and PassCount defaults to 2.
func TestNewPreparationCoordinator_DefaultPathUnchanged(t *testing.T) {
	os.Setenv("OPENAI_API_KEY", "test-key-for-testing")
	defer os.Unsetenv("OPENAI_API_KEY")

	coordinator, err := NewPreparationCoordinator(PreparationConfig{
		SourceLanguage: "en",
		TargetLanguage: "es",
		Providers:      []string{"openai"},
		APIKey:         "test-key-for-testing",
	})
	if err != nil {
		t.Fatalf("unexpected error on default (nil-factory) path: %v", err)
	}
	if coordinator == nil {
		t.Fatal("expected coordinator, got nil")
	}
	if coordinator.config.PassCount != 2 {
		t.Errorf("expected default PassCount 2, got %d", coordinator.config.PassCount)
	}
	if len(coordinator.providers) == 0 {
		t.Error("expected built-in path to produce at least one provider")
	}
}
