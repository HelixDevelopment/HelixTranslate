package grpc

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/bridge"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/translator"
)

// fakeGRPCBridgeTranslator is a minimal translator.Translator used to prove the
// gRPC API-translation path routes through the injected bridge factory rather
// than constructing a local llm.NewLLMTranslator. It records the task.
type fakeGRPCBridgeTranslator struct {
	task selection.TaskRequirements
}

func (f *fakeGRPCBridgeTranslator) Translate(_ context.Context, text, _ string) (string, error) {
	return "BRIDGE:" + text, nil
}

func (f *fakeGRPCBridgeTranslator) TranslateWithProgress(_ context.Context, text, _ string, _ *events.EventBus, _ string) (string, error) {
	return "BRIDGE:" + text, nil
}

func (f *fakeGRPCBridgeTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}
func (f *fakeGRPCBridgeTranslator) GetName() string { return "bridge-verified" }

func newJob(req *proto.TranslationRequest) *TranslationJob {
	return &TranslationJob{ID: req.SessionId, Request: req, Context: context.Background()}
}

// installMockBridge wires a deterministic in-memory bridge translator factory onto
// the core translator so the API-translation path exercises the bridge seam
// WITHOUT real provider keys or a network call (§11.4.27 unit-test fake). The
// returned translator reproduces the historical mock provider's observable
// transform ("translated: <text>", preserving the source) so existing
// content-preservation assertions still hold after the R-1b/R2 redirect.
func installMockBridge(ct *CoreTranslatorImpl) {
	ct.bridgeTranslatorFactory = func(_ context.Context, _ selection.TaskRequirements) (translator.Translator, error) {
		return &mockBridgeTranslator{}, nil
	}
}

type mockBridgeTranslator struct{}

func (m *mockBridgeTranslator) Translate(_ context.Context, text, _ string) (string, error) {
	return "translated: " + text, nil
}

func (m *mockBridgeTranslator) TranslateWithProgress(_ context.Context, text, _ string, _ *events.EventBus, _ string) (string, error) {
	return "translated: " + text, nil
}

func (m *mockBridgeTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}
func (m *mockBridgeTranslator) GetName() string { return "llm-mock" }

// TestExecuteAPITranslation_RoutesThroughBridge proves the R-1b/R2 redirect: the
// default/API arm sources its translator from the LLMsVerifier bridge (the
// strongest verified model for the requested language pair) instead of a local
// llm.NewLLMTranslator. This is the §11.4.108 runtime-signature for the gRPC site.
func TestExecuteAPITranslation_RoutesThroughBridge(t *testing.T) {
	ct := NewCoreTranslator(logger.NewNoOpLogger()).(*CoreTranslatorImpl)

	var captured selection.TaskRequirements
	ct.bridgeTranslatorFactory = func(_ context.Context, task selection.TaskRequirements) (translator.Translator, error) {
		captured = task
		return &fakeGRPCBridgeTranslator{task: task}, nil
	}

	req := &proto.TranslationRequest{
		SessionId:  "grpc-bridge-1",
		SourceLang: "en",
		TargetLang: "es",
		ProviderConfig: &proto.ProviderConfig{
			Type:  "openai",
			Model: "gpt-3.5-turbo",
		},
	}

	out, err := ct.executeAPITranslation(newJob(req), "hello", nil)
	if err != nil {
		t.Fatalf("expected bridge-sourced translation, got error: %v", err)
	}
	if out != "BRIDGE:hello" {
		t.Fatalf("expected the bridge translator to be used (BRIDGE:hello), got %q", out)
	}
	if captured.SourceLang != "en" || captured.TargetLang != "es" {
		t.Fatalf("language pair must be threaded into the bridge task; got %+v", captured)
	}
}

// TestExecuteAPITranslation_NoKey_HonestHardError proves the R2 honest no-key
// path: with NO provider API keys and NO LLMSVERIFIER_API_URL, the API arm must
// return a hard error — never a silent local-runtime fallback (§11.4.69). We
// drive the real bridge.Open via an empty Getenv so the result is deterministic.
func TestExecuteAPITranslation_NoKey_HonestHardError(t *testing.T) {
	ct := NewCoreTranslator(logger.NewNoOpLogger()).(*CoreTranslatorImpl)
	ct.bridgeOpener = func(ctx context.Context) (*bridge.Bridge, error) {
		return bridge.Open(ctx, bridge.Options{Getenv: func(string) string { return "" }})
	}

	req := &proto.TranslationRequest{
		SessionId:      "grpc-bridge-nokey",
		SourceLang:     "en",
		TargetLang:     "es",
		ProviderConfig: &proto.ProviderConfig{Type: "openai", Model: "gpt-3.5-turbo"},
	}

	_, err := ct.executeAPITranslation(newJob(req), "hello", nil)
	if err == nil {
		t.Fatal("no-key path MUST be a hard error, never a silent local fallback")
	}
	if !strings.Contains(err.Error(), "no provider API keys") {
		t.Fatalf("error must honestly explain that no provider keys are set; got: %v", err)
	}
}
