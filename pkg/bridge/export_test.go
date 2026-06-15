package bridge

import (
	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/pkg/translator/llm"
)

// newTestBridge builds a Bridge backed by a factory pre-populated with the given
// verified models, WITHOUT opening any network source or reading real API keys.
// It is the in-package test seam for exercising selection/ranking/fallback logic
// deterministically (§11.4.27 unit; no fakes leak to production — this file is
// _test.go only). getenv supplies provider keys for the resolver.
func newTestBridge(getenv func(string) string, models ...verifier.Model) *Bridge {
	cfg := verifier.DefaultConfig()
	cfg.MinScoreThreshold = 0.0
	factory := llm.NewVerifiedFactory(cfg)
	resolver := verifier.NewProviderResolverWithEnv(getenv)
	factory.SetKeyResolver(func(providerID string) string {
		rp, err := resolver.Resolve(providerID)
		if err != nil {
			return ""
		}
		return rp.APIKey
	})
	for _, m := range models {
		factory.RegisterModel(m)
	}
	return &Bridge{factory: factory, resolver: resolver, cfg: cfg, source: "in-process"}
}
