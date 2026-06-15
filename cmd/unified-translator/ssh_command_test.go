package main

import (
	"context"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// SSH-local translation path removal — bridge phase-2 R-4 (operator-confirmed).
//
// The remote-SSH-worker-running-llama.cpp path was the project's only
// local-runtime translation path. Operator scope decision (R-1 + R-4): "Keep
// distributed API, remove only SSH-local." The provider=ssh arm of
// executeProviderTranslation is removed.
//
// CONTRACT (§11.4.69 — never a silent fallback): with the SSH executor gone,
// provider=ssh MUST NOT silently route to the API/bridge path (which would
// translate via a DIFFERENT provider than the user asked for — a silent
// wrong-provider defect). It MUST hard-error honestly.
//
// §11.4.115 polarity: this assertion FAILs against a `default:`-only router
// (where ssh falls through to executeAPITranslation and returns no error), and
// PASSes once executeProviderTranslation hard-errors for provider=ssh.
// ---------------------------------------------------------------------------

func TestExecuteProviderTranslation_SSHRemovedHonestError(t *testing.T) {
	cfg := &UnifiedConfig{
		SourceLang: "en",
		TargetLang: "fr",
		Provider:   "ssh",
	}
	session := testSession("ssh-removed-honest-error")

	out, err := executeProviderTranslation(context.Background(), cfg, session, "hello")
	if err == nil {
		t.Fatalf("provider=ssh MUST hard-error after SSH-local removal (no silent API fallback); got nil error, output %q", out)
	}
	if out != "" {
		t.Fatalf("provider=ssh error path must return empty output, got %q", out)
	}
	// The error must name the removed SSH path so the operator understands why.
	if !strings.Contains(strings.ToLower(err.Error()), "ssh") {
		t.Fatalf("ssh-removal error should mention ssh; got %q", err.Error())
	}
}
