package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/language"

	"github.com/stretchr/testify/require"
)

// hasAnyProviderKey reports whether any provider API key is present in the
// environment. Under R2 (LLMsVerifier bridge) translateEbook obtains its
// translator from the bridge, which needs at least one provider key to provision
// a verified model; with none present bridge.Open returns an honest hard error.
func hasAnyProviderKey() bool {
	for _, k := range []string{
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY",
		"ZHIPU_API_KEY", "QWEN_API_KEY", "GEMINI_API_KEY",
		"GROQ_API_KEY", "MISTRAL_API_KEY", "XAI_API_KEY",
		"COHERE_API_KEY", "TOGETHER_API_KEY",
	} {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			return true
		}
	}
	return false
}

// TestTranslateEbookNoKeysHonestError is the R2 contract guard (replaces the
// pre-R2 TestEnvAPIKeyMatchesResolvedProvider, which asserted the now-removed
// per-provider config-key → provider-Bearer routing inside translateEbook).
//
// Under R2 translateEbook opens the LLMsVerifier bridge and sources its
// translator from the strongest verified model — it NO LONGER threads a
// config/CLI api-key into an llm.NewLLMTranslator construction. The credential-
// routing that the old test guarded now lives in the bridge's ProviderResolver
// (covered by pkg/bridge tests). What translateEbook MUST guarantee at the CLI
// seam is the §11.4.69 invariant: with NO provider API keys present it returns an
// honest hard error and NEVER silently falls back to a local runtime.
func TestTranslateEbookNoKeysHonestError(t *testing.T) {
	if hasAnyProviderKey() {
		t.Skip("SKIP-OK (§11.4.3): a provider API key is present; the no-key honest-error contract is only assertable with no keys in the environment")
	}

	book := &ebook.Book{
		Metadata: ebook.Metadata{Title: "T", Language: "en"},
		Chapters: []ebook.Chapter{{
			Title:    "C",
			Sections: []ebook.Section{{Content: "hello world"}},
		}},
	}
	outFile := filepath.Join(t.TempDir(), "out.epub")

	err := translateEbook(
		book,
		outFile,
		"epub",
		"openai",
		"",
		"",
		"",
		"default",
		nil,
		language.English,
		language.Spanish,
		nil,
		false,
		false,
		false,
		false,
	)
	require.Error(t, err, "with no provider API keys translateEbook MUST fail loudly via the bridge, never fall back to a local runtime")
	require.NoFileExists(t, outFile, "a failed (no-key) run must not leave an output file")
}

// keep config import referenced (config-driven paths remain exercised by other
// tests); avoids churn if this file later regrows config-based cases.
var _ = config.Config{}
