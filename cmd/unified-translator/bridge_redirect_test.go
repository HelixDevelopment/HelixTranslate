package main

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/bridge"
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
// the mock test/demo seam must keep working offline. The assertion is made
// env-independent by installing a sentinel bridgeOpener that t.Fatal's if it is
// ever called: a correct mock path never opens the bridge, so the sentinel stays
// untouched; a regression that routed "mock" through the bridge would invoke the
// opener and fail loudly — a POSITIVE assertion that does NOT depend on the host
// carrying (or lacking) provider keys / network (§11.4.3).
func TestExecuteAPITranslation_MockSeamBypassesBridge(t *testing.T) {
	// Sentinel opener: the mock path MUST NOT reach the bridge. If it does, this
	// fails regardless of the host's key/network state.
	orig := bridgeOpener
	t.Cleanup(func() { bridgeOpener = orig })
	bridgeOpener = func(context.Context) (*bridge.Bridge, error) {
		t.Fatal("mock-provider path opened the LLMsVerifier bridge — the mock seam must bypass the bridge entirely (§11.4.69/R-1)")
		return nil, nil
	}

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
