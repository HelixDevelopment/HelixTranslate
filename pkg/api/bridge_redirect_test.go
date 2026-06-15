package api

import (
	"context"
	"testing"

	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/bridge"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeBridgeTranslator is a minimal translator.Translator used to prove the
// createTranslator seam routes through the injected bridge factory rather than a
// local llm.NewLLMTranslator construction. It records the task it was asked for.
type fakeBridgeTranslator struct {
	name string
	task selection.TaskRequirements
}

func (f *fakeBridgeTranslator) Translate(_ context.Context, text, _ string) (string, error) {
	return "BRIDGE:" + text, nil
}

func (f *fakeBridgeTranslator) TranslateWithProgress(_ context.Context, text, _ string, _ *events.EventBus, _ string) (string, error) {
	return "BRIDGE:" + text, nil
}

func (f *fakeBridgeTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}
func (f *fakeBridgeTranslator) GetName() string { return f.name }

// installMockBridge wires a deterministic in-memory bridge translator factory onto
// the handler so translation-path tests exercise the bridge seam WITHOUT real
// provider keys or a network call (§11.4.27 unit-test fake). The returned
// translator mimics the historical mock provider's user-visible behaviour
// (name "llm-mock", output "Translated: <text>") so existing success-path
// assertions still hold after the R-1b/R2 bridge redirect.
func installMockBridge(h *Handler) {
	h.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
		return &mockBridgeTranslator{}, nil
	}
}

// mockBridgeTranslator reproduces the prior mock provider's observable contract.
type mockBridgeTranslator struct{}

func (m *mockBridgeTranslator) Translate(_ context.Context, text, _ string) (string, error) {
	return "Translated: " + text, nil
}

func (m *mockBridgeTranslator) TranslateWithProgress(_ context.Context, text, _ string, _ *events.EventBus, _ string) (string, error) {
	return "Translated: " + text, nil
}

func (m *mockBridgeTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}
func (m *mockBridgeTranslator) GetName() string { return "llm-mock" }

// TestCreateTranslator_RoutesThroughBridge proves the R2 redirect: when a bridge
// translator factory is injected, createTranslator returns the bridge-sourced
// translator (NOT a locally-constructed llm.NewLLMTranslator), and the requested
// language pair is threaded into the bridge's selection.TaskRequirements. This is
// the §11.4.108 runtime-signature for R-1b: the constructed translator is the one
// the bridge yielded, not a local provider runtime.
func TestCreateTranslator_RoutesThroughBridge(t *testing.T) {
	h := newContractHandler()

	var captured selection.TaskRequirements
	h.bridgeTranslatorFactory = func(_ context.Context, task selection.TaskRequirements) (translator.Translator, error) {
		captured = task
		return &fakeBridgeTranslator{name: "bridge-verified", task: task}, nil
	}

	trans, err := h.createTranslator("openai", "gpt-3.5-turbo", "en", "es")
	require.NoError(t, err)
	require.Equal(t, "bridge-verified", trans.GetName(),
		"createTranslator must return the bridge-sourced translator, not a local NewLLMTranslator")

	out, err := trans.Translate(context.Background(), "hi", "")
	require.NoError(t, err)
	assert.Equal(t, "BRIDGE:hi", out, "the bridge translator must be the one actually used")

	assert.Equal(t, "en", captured.SourceLang, "source_lang must be threaded into the bridge task")
	assert.Equal(t, "es", captured.TargetLang, "target_lang must be threaded into the bridge task")
}

// TestCreateTranslator_NoKey_HonestHardError proves the R2 honest no-key path:
// with NO provider API keys and NO LLMSVERIFIER_API_URL, opening the bridge MUST
// return a hard error — never a silent local-runtime fallback (§11.4.69). We
// exercise the real bridge.Open via an empty Getenv so the result is
// deterministic regardless of the host's actual environment.
func TestCreateTranslator_NoKey_HonestHardError(t *testing.T) {
	h := newContractHandler()
	h.bridgeOpener = func(ctx context.Context) (*bridge.Bridge, error) {
		// Empty environment -> no provider keys -> honest hard error.
		return bridge.Open(ctx, bridge.Options{Getenv: func(string) string { return "" }})
	}

	_, err := h.createTranslator("openai", "gpt-3.5-turbo", "en", "es")
	require.Error(t, err, "no-key path MUST be a hard error, never a silent local fallback")
	assert.Contains(t, err.Error(), "no provider API keys",
		"the error must honestly explain that no provider keys are set")
}

// TestCreateTranslator_Distributed_Unchanged guards that the distributed arm is
// untouched by the redirect: it returns the distributedTranslator wrapper and
// never reaches the bridge.
func TestCreateTranslator_Distributed_Unchanged(t *testing.T) {
	h := newContractHandler()
	h.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
		t.Fatal("distributed arm must not reach the bridge factory")
		return nil, nil
	}
	// distributedManager is nil in the contract handler, so the distributed arm
	// returns its honest "not available" error before any bridge call.
	_, err := h.createTranslator("distributed", "", "en", "es")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "distributed translation not available")
}
