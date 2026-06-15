package verifier

import (
	"os"
	"sort"
)

// envProviderSpec maps an environment-variable name to a provider ID and its
// OpenAI-compatible base URL. The verifier Pipeline issues requests against
// "<BaseURL>/models" and "<BaseURL>/chat/completions", so every BaseURL here is
// the OpenAI-compatible v1 root for that provider.
type envProviderSpec struct {
	EnvVar  string
	ID      string
	BaseURL string
}

// envProviderSpecs is the canonical, deterministic list of OpenAI-compatible
// providers HelixTranslate can verify directly from environment-provided API
// keys. Base URLs are the providers' documented OpenAI-compatible endpoints.
// New providers are appended here; the list is the single source of truth for
// ProvidersFromEnv.
var envProviderSpecs = []envProviderSpec{
	{EnvVar: "OPENAI_API_KEY", ID: "openai", BaseURL: "https://api.openai.com/v1"},
	{EnvVar: "DEEPSEEK_API_KEY", ID: "deepseek", BaseURL: "https://api.deepseek.com/v1"},
	{EnvVar: "GROQ_API_KEY", ID: "groq", BaseURL: "https://api.groq.com/openai/v1"},
	{EnvVar: "OPENROUTER_API_KEY", ID: "openrouter", BaseURL: "https://openrouter.ai/api/v1"},
	{EnvVar: "CEREBRAS_API_KEY", ID: "cerebras", BaseURL: "https://api.cerebras.ai/v1"},
	{EnvVar: "MISTRAL_API_KEY", ID: "mistral", BaseURL: "https://api.mistral.ai/v1"},
	{EnvVar: "TOGETHER_API_KEY", ID: "together", BaseURL: "https://api.together.xyz/v1"},
	{EnvVar: "FIREWORKS_API_KEY", ID: "fireworks", BaseURL: "https://api.fireworks.ai/inference/v1"},
	{EnvVar: "NVIDIA_API_KEY", ID: "nvidia", BaseURL: "https://integrate.api.nvidia.com/v1"},
	{EnvVar: "SAMBANOVA_API_KEY", ID: "sambanova", BaseURL: "https://api.sambanova.ai/v1"},
	{EnvVar: "HYPERBOLIC_API_KEY", ID: "hyperbolic", BaseURL: "https://api.hyperbolic.xyz/v1"},
	{EnvVar: "NOVITA_API_KEY", ID: "novita", BaseURL: "https://api.novita.ai/v3/openai"},
	{EnvVar: "SILICONFLOW_API_KEY", ID: "siliconflow", BaseURL: "https://api.siliconflow.com/v1"},
	{EnvVar: "ZHIPU_API_KEY", ID: "zhipu", BaseURL: "https://open.bigmodel.cn/api/paas/v4"},
	{EnvVar: "UPSTAGE_API_KEY", ID: "upstage", BaseURL: "https://api.upstage.ai/v1"},
	// GEMINI exposes an OpenAI-compatible surface at the openai/ sub-path.
	{EnvVar: "GEMINI_API_KEY", ID: "gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai"},
}

// ProvidersFromEnv builds a ProviderConfig for every OpenAI-compatible provider
// in envProviderSpecs whose API-key environment variable is set and non-empty.
// A provider with a missing or empty key is omitted entirely (never returned
// with a blank APIKey), so a caller only ever receives configs it can actually
// authenticate. The result is sorted by provider ID for deterministic ordering.
//
// API-key VALUES are read from the environment and stored in the returned
// configs in memory only — they are never logged, printed, or persisted by this
// function (§11.4.10).
func ProvidersFromEnv() []ProviderConfig {
	return ProvidersFromGetenv(os.Getenv)
}

// ProvidersFromGetenv is ProvidersFromEnv with an injectable key source. The
// bridge passes its own getenv so its bootstrap honours the same environment
// view as its resolver (deterministic + unit-testable without mutating the
// process environment, §11.4.10). getenv nil → os.Getenv.
func ProvidersFromGetenv(getenv func(string) string) []ProviderConfig {
	if getenv == nil {
		getenv = os.Getenv
	}
	var out []ProviderConfig
	for _, spec := range envProviderSpecs {
		key := getenv(spec.EnvVar)
		if key == "" {
			continue
		}
		out = append(out, ProviderConfig{
			ID:      spec.ID,
			APIKey:  key,
			BaseURL: spec.BaseURL,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// EnvProviderIDs returns the provider IDs ProvidersFromEnv would emit for the
// CURRENT environment, sorted. Useful for logging which providers are
// configured WITHOUT exposing any key value (§11.4.10).
func EnvProviderIDs() []string {
	cfgs := ProvidersFromEnv()
	ids := make([]string, 0, len(cfgs))
	for _, c := range cfgs {
		ids = append(ids, c.ID)
	}
	return ids
}
