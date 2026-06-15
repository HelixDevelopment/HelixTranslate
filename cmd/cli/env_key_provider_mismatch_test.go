package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/language"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvAPIKeyMatchesResolvedProvider reproduces a real CLI orchestration
// defect: main() loads the env API key for the *pre-config* CLI provider
// (default "openai") BEFORE config resolution picks the actual provider, and
// translateEbook then treats that already-populated (wrong-provider) key as
// "user supplied" — so the per-provider config key is never applied.
//
// Scenario mirrors main.go exactly:
//   - User did NOT pass -provider, so the CLI provider value is the "openai"
//     default; main.go line 156-157 does apiKey = getAPIKeyFromEnv("openai").
//   - The config selects DefaultProvider "deepseek" with its OWN correct API
//     key (and a mock base URL).
//   - OPENAI_API_KEY is present in the environment (very common).
//
// Correct behaviour: the deepseek provider client must authenticate with the
// deepseek key from config. Buggy behaviour: the leaked OPENAI_API_KEY is sent
// as the Bearer token to the deepseek endpoint — wrong credential for the
// resolved provider.
func TestEnvAPIKeyMatchesResolvedProvider(t *testing.T) {
	const (
		leakedOpenAIKey   = "LEAKED-OPENAI-KEY"
		correctDeepSeekKey = "CORRECT-DEEPSEEK-KEY"
		mockTranslation    = "prevod"
	)

	// Simulate the common environment: OPENAI_API_KEY is set.
	t.Setenv("OPENAI_API_KEY", leakedOpenAIKey)
	// Make sure no real deepseek env key interferes with the assertion.
	t.Setenv("DEEPSEEK_API_KEY", "")

	var hits int32
	var gotAuth atomic.Value
	gotAuth.Store("")
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		atomic.AddInt32(&hits, 1)
		gotAuth.Store(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"` + mockTranslation + `"}}]}`))
	}))
	defer mock.Close()

	// Mimic main.go's flag-default + env-key load sequence: the user left
	// -provider at its "openai" default, so the env key loaded is OPENAI's.
	cliProvider := "openai"
	apiKey := getAPIKeyFromEnv(cliProvider)
	require.Equal(t, leakedOpenAIKey, apiKey, "precondition: openai env key loaded")

	appConfig := &config.Config{
		Translation: config.TranslationConfig{
			DefaultProvider: "deepseek",
			DefaultModel:    "deepseek-chat",
			Providers: map[string]config.ProviderConfig{
				"deepseek": {
					APIKey:  correctDeepSeekKey,
					BaseURL: mock.URL,
				},
			},
		},
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
		cliProvider, // provider value as main() passes it (default "openai")
		"",          // model — resolved from config
		apiKey,      // env key already loaded for the *CLI* provider
		"",          // baseURL — from config
		"default",
		appConfig,
		language.English,
		language.Spanish,
		nil,
		false,
		false,
		false, // provider NOT explicitly set -> config DefaultProvider applies
		false, // api-key NOT explicitly set (came from env) -> must re-resolve
	)
	require.NoError(t, err)
	require.Greater(t, atomic.LoadInt32(&hits), int32(0), "deepseek mock must be hit")

	auth := gotAuth.Load().(string)
	assert.Equal(t, "Bearer "+correctDeepSeekKey, auth,
		"resolved deepseek provider must authenticate with its config key, not the leaked OPENAI_API_KEY")
	assert.NotEqual(t, "Bearer "+leakedOpenAIKey, auth,
		"the pre-config OPENAI_API_KEY must not leak as the deepseek bearer token")
}
