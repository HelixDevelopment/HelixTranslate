package main

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/logger"
)

// testSession builds a TranslationSession with a real (text) logger + event bus
// so executeAPITranslation's session.Logger.Info calls do not nil-panic.
func testSession(id string) *TranslationSession {
	return &TranslationSession{
		ID:       id,
		EventBus: events.NewEventBus(),
		Logger:   logger.NewLogger(logger.LoggerConfig{Level: logger.INFO, Format: logger.FORMAT_TEXT}),
	}
}

// TestExecuteAPITranslation_MockSeamBypassesBridge proves the R-1/R2 redirect
// preserved the in-process "mock" provider seam: a Provider:"mock" run MUST NOT
// route through the LLMsVerifier bridge (which requires real verified models /
// network), so it produces deterministic output with NO API key and NO network.
//
// This is the load-bearing R-1 contract for unified-translator: the legacy
// default branch was redirected to bridgeTranslator (bridge.BestTranslator), but
// the mock test/demo seam must keep working offline. If a future change routed
// "mock" through the bridge, this test FAILs (the bridge would hard-error with no
// keys, or attempt a live verification pass), proving the seam guard is real
// (§11.4.115 polarity — the failure mode it catches is "mock collapsed into the
// bridge path").
func TestExecuteAPITranslation_MockSeamBypassesBridge(t *testing.T) {
	cfg := &UnifiedConfig{
		SourceLang: "en",
		TargetLang: "es",
		Provider:   "mock",
	}
	session := testSession("bridge-redirect-mock-seam")

	out, err := executeAPITranslation(context.Background(), cfg, session, "Hello, world.")
	if err != nil {
		t.Fatalf("mock-provider executeAPITranslation returned error (mock must work offline, no bridge): %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("mock-provider executeAPITranslation produced empty output — the mock seam did not run")
	}
}

// TestBridgeTranslator_HonestErrorOnNoSource proves bridgeTranslator surfaces the
// bridge's honest hard error rather than ever silently falling back to a local
// runtime (§11.4.69). We force the in-process no-source condition by pointing the
// bridge at an HTTP URL that is unreachable — bridge.Open then returns an honest
// "service is unreachable" error, which bridgeTranslator must propagate.
//
// (A pure no-key assertion is not deterministic here because the host CI/dev
// environment legitimately carries provider keys; an unreachable explicit source
// is the deterministic, network-independent way to exercise the honest-error path
// without performing a live verification pass with real credentials, §11.4.3.)
func TestBridgeTranslator_HonestErrorOnNoSource(t *testing.T) {
	// LLMSVERIFIER_API_URL set to an unroutable address → bridge.Open pings it,
	// fails, and returns an honest error (never a local fallback).
	t.Setenv("LLMSVERIFIER_API_URL", "http://127.0.0.1:1/never-listens")

	cfg := &UnifiedConfig{SourceLang: "en", TargetLang: "es", Provider: "deepseek"}
	_, err := bridgeTranslator(context.Background(), cfg)
	if err == nil {
		t.Fatal("bridgeTranslator returned nil error with an unreachable source — a silent fallback is forbidden (§11.4.69)")
	}
}
