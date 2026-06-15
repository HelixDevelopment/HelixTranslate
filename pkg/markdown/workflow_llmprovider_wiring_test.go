package markdown

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/translator/llm"
)

// TestWorkflowConfig_LLMProvider_IsConsumedByTranslate is the R-1d wiring guard.
//
// WorkflowConfig.LLMProvider is the markdown package's dependency-injection seam
// for the LLM client (its producer is the LLMsVerifier bridge — bridge.BestClient
// — wired in cmd/markdown-translator). Before R-1d the field had ZERO producers,
// so it was an unwired, nil-panicking seam. This test proves the field is the seam
// SimpleWorkflow.TranslateMarkdown actually routes translation through: the
// injected client MUST be invoked and its output MUST land in the translated file.
//
// §11.4.115 polarity: this FAILs if the translate path stops consuming
// config.LLMProvider (e.g. ignores it / hardcodes a different client / leaves the
// field dead) — the exact pre-wiring failure mode. It PASSes when the seam is live.
func TestWorkflowConfig_LLMProvider_IsConsumedByTranslate(t *testing.T) {
	dir := t.TempDir()
	srcMD := filepath.Join(dir, "in.md")
	outMD := filepath.Join(dir, "out.md")

	const line = "Zdravo svete"
	if err := os.WriteFile(srcMD, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("write src md: %v", err)
	}

	mock := llm.NewMockLLMClient()
	mock.SetResponse(line, "PROVIDER_REACHED::"+line)

	cfg := WorkflowConfig{
		ChunkSize:        2000,
		MaxConcurrency:   1,
		TranslationCache: map[string]string{},
		LLMProvider:      mock, // the seam under test
	}
	sw := NewSimpleWorkflow(cfg, logger.NewNoOpLogger(), nil)

	if err := sw.TranslateMarkdown(context.Background(), srcMD, outMD, "sr", "en"); err != nil {
		t.Fatalf("TranslateMarkdown: %v", err)
	}

	// The injected client MUST have been invoked through the LLMProvider seam.
	if mock.CallCount() == 0 {
		t.Fatalf("WorkflowConfig.LLMProvider was never invoked — the seam is not wired into the translate path")
	}

	out, err := os.ReadFile(outMD)
	if err != nil {
		t.Fatalf("read out md: %v", err)
	}
	if !strings.Contains(string(out), "PROVIDER_REACHED::") {
		t.Fatalf("translated output did not come from the injected LLMProvider; got: %q", string(out))
	}
}
