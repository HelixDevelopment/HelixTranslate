package coordination

import (
	"context"
	"os"
	"testing"

	"digital.vasic.translator/pkg/translator"
)

// R-1c intermediate plumbing guard for the coordination wrapper.
//
// §11.4.115 polarity switch: RED_MODE=1 reproduces the defect (the injected
// EnsembleTranslatorFactory is dropped at the wrapper intermediate and discovery
// runs instead) so the guard FAILs on the broken behaviour; RED_MODE=0 (default)
// is the standing GREEN regression guard asserting the injected factory's
// providers are threaded all the way through NewMultiLLMTranslatorWrapperWithFactory
// to the underlying coordinator.
//
// To capture the RED proof on the threading-removed behaviour, run:
//
//	RED_MODE=1 go test -run TestNewMultiLLMTranslatorWrapperWithFactory_InjectedEnsembleThreaded ./pkg/coordination/
//
// which (a) swaps the factory for nil at the wrapper boundary, so discovery runs
// with no API keys → 0 instances → ErrNoLLMInstances, and the construction
// assertion FAILs — proving the test catches a wrapper that fails to thread the
// factory to the coordinator leaf seam.
func TestNewMultiLLMTranslatorWrapperWithFactory_InjectedEnsembleThreaded(t *testing.T) {
	// Ensure no ambient API keys leak the discovery path into a non-zero count.
	for _, k := range []string{
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "ZHIPU_API_KEY",
		"DEEPSEEK_API_KEY", "QWEN_API_KEY", "OLLAMA_ENABLED",
	} {
		os.Unsetenv(k)
	}
	os.Setenv("SKIP_QWEN_OAUTH", "1")
	defer os.Unsetenv("SKIP_QWEN_OAUTH")

	wantNames := []string{"alpha-llm", "beta-llm", "gamma-llm"}

	factoryCalled := false
	var factory EnsembleTranslatorFactory = func(_ context.Context) ([]translator.Translator, error) {
		factoryCalled = true
		return []translator.Translator{
			&namedStubTranslator{name: wantNames[0]},
			&namedStubTranslator{name: wantNames[1]},
			&namedStubTranslator{name: wantNames[2]},
		}, nil
	}

	// RED_MODE=1: drop the factory at the wrapper boundary, emulating an
	// intermediate that fails to thread it to NewMultiLLMCoordinatorWithFactory.
	if os.Getenv("RED_MODE") == "1" {
		factory = nil
	}

	wrapper, err := NewMultiLLMTranslatorWrapperWithFactory(
		translator.TranslationConfig{SourceLang: "en", TargetLang: "es"},
		nil,
		"wrapper-factory-test",
		false,
		false,
		factory,
	)
	if err != nil {
		t.Fatalf("expected wrapper from injected factory, got error: %v", err)
	}
	if wrapper == nil || wrapper.Coordinator == nil {
		t.Fatal("expected non-nil wrapper with coordinator from injected factory")
	}

	if os.Getenv("RED_MODE") != "1" && !factoryCalled {
		t.Fatal("expected the injected factory to be invoked through the wrapper, but it was not")
	}

	// Exactly the 3 injected translators must become coordinator instances.
	if got := wrapper.Coordinator.GetInstanceCount(); got != len(wantNames) {
		t.Fatalf("expected %d coordinator instances from injected factory, got %d", len(wantNames), got)
	}

	gotProviders := make(map[string]bool)
	for _, inst := range wrapper.Coordinator.instances {
		gotProviders[inst.Provider] = true
	}
	for _, want := range wantNames {
		if !gotProviders[want] {
			t.Errorf("expected coordinator instance with provider identity %q threaded from the factory; providers=%v",
				want, gotProviders)
		}
	}
}

// TestNewMultiLLMTranslatorWrapperWithFactory_NilFactoryUnchanged proves the
// nil-factory wrapper path is byte-for-byte the legacy
// NewMultiLLMTranslatorWrapperWithConfig behaviour: with no API keys discovery
// yields 0 instances and the wrapper returns ErrNoLLMInstances, exactly as before.
func TestNewMultiLLMTranslatorWrapperWithFactory_NilFactoryUnchanged(t *testing.T) {
	for _, k := range []string{
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "ZHIPU_API_KEY",
		"DEEPSEEK_API_KEY", "QWEN_API_KEY", "OLLAMA_ENABLED",
	} {
		os.Unsetenv(k)
	}
	os.Setenv("SKIP_QWEN_OAUTH", "1")
	defer os.Unsetenv("SKIP_QWEN_OAUTH")

	cfg := translator.TranslationConfig{SourceLang: "en", TargetLang: "es"}

	// Nil factory via the new intermediate and the legacy entry point MUST behave
	// identically: discovery, no keys → ErrNoLLMInstances.
	_, errFactory := NewMultiLLMTranslatorWrapperWithFactory(cfg, nil, "nil-factory", false, false, nil)
	_, errLegacy := NewMultiLLMTranslatorWrapperWithConfig(cfg, nil, "nil-legacy", false, false)

	if errFactory != translator.ErrNoLLMInstances {
		t.Errorf("nil-factory wrapper: expected ErrNoLLMInstances, got %v", errFactory)
	}
	if errLegacy != translator.ErrNoLLMInstances {
		t.Errorf("legacy wrapper: expected ErrNoLLMInstances, got %v", errLegacy)
	}
}
