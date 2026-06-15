// Package bridge is the single facade between the HelixTranslate System (every
// translating component) AND a Claude Code agent session and the LLMsVerifier-
// obtained STRONGEST verified models. It wraps the EXISTING internal/verifier
// pipeline + selection.Engine + pkg/translator/llm.VerifiedFactory (§11.4.74 —
// no reimplementation of discovery/scoring/selection) behind:
//
//   - BestTranslator(task) — strongest verified model + deterministic fallback
//     chain (try the next verified model if the top one fails).
//   - Invoke(ctx, system, prompt) — raw chat capability of the strongest model,
//     for direct agent access.
//   - BestModel / ListVerified — the strongest model + the verified set.
//
// Out-of-the-box bootstrap (§6 of the design, §11.4.6 — no guessing):
//   - LLMSVERIFIER_API_URL set  → HTTP verifier.Client mode (operator-provided
//     running service).
//   - else (canonical OOTB)     → in-process: ProvidersFromEnv() →
//     RunVerification (real provider APIs) → SQLite store → select.
//   - NO *_API_KEY present      → honest hard error (NEVER a local/llama.cpp
//     fallback — local inference is forbidden by the mandate).
//
// API-key values are read from the environment IN MEMORY ONLY and are never
// logged, printed, or persisted (§11.4.10).
package bridge

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"digital.vasic.translator/internal/services"
	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/persistence"
	"digital.vasic.translator/internal/verifier/selection"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// invokeMaxTokens bounds a raw Invoke completion. Kept below the smallest common
// per-model cap among verified providers (e.g. Groq allam-2-7b = 4096) so Invoke
// never trips a provider's max_tokens ceiling.
const invokeMaxTokens = 2048

// ModelInfo is the bridge's public, key-free view of a verified model. It never
// carries an API key — only metadata safe to print (§11.4.10).
type ModelInfo struct {
	ProviderID    string          `json:"provider_id"`
	FactoryName   string          `json:"factory_provider"`
	ModelID       string          `json:"model_id"`
	ModelName     string          `json:"model_name"`
	Score         float64         `json:"score"`
	FallbackOrder int             `json:"fallback_order"`
	Capabilities  map[string]bool `json:"capabilities,omitempty"`
	BaseURL       string          `json:"base_url,omitempty"`
}

// Options tunes Open. Zero value is valid: in-process mode, default DB path,
// bounded verification.
type Options struct {
	// APIURL, when non-empty (or when LLMSVERIFIER_API_URL is set), selects the
	// HTTP verifier.Client source instead of the in-process pipeline.
	APIURL string
	// APIKey is the Bearer token for the HTTP service (ignored in in-process mode).
	APIKey string
	// DBPath is the SQLite store path for the in-process pipeline.
	// Defaults to "./data/verified_models.db".
	DBPath string
	// MaxModelsPerProvider caps how many discovered models per provider are run
	// through the full pipeline when a fresh verification is needed (0 = no cap).
	// Defaults to 3 to bound wall-clock + token spend (§11.4.82).
	MaxModelsPerProvider int
	// MinScoreThreshold is the floor a model's overall score must exceed to be
	// considered verified-and-selectable.
	MinScoreThreshold float64
	// VerifyTimeout bounds an in-process RunVerification pass. Defaults to 5m.
	VerifyTimeout time.Duration
	// ForceVerify, when true, always runs a fresh in-process verification pass
	// even if the SQLite store already holds models.
	ForceVerify bool
	// Getenv overrides the environment source (tests). Defaults to os.Getenv.
	Getenv func(string) string
}

// Bridge is the facade. Construct with Open; it is safe for concurrent use.
type Bridge struct {
	factory  *llm.VerifiedFactory
	resolver *verifier.ProviderResolver
	cfg      *verifier.Config
	source   string // "http" or "in-process" — for honest reporting
}

// Open bootstraps the bridge out-of-the-box. See package doc for the resolution
// order. Returns an honest hard error if no source can be provisioned (e.g. no
// API keys present) — never a silent fallback.
func Open(ctx context.Context, opts Options) (*Bridge, error) {
	getenv := opts.Getenv
	if getenv == nil {
		getenv = osGetenv
	}

	cfg := verifier.DefaultConfig()
	cfg.MinScoreThreshold = opts.MinScoreThreshold

	apiURL := opts.APIURL
	if apiURL == "" {
		apiURL = getenv("LLMSVERIFIER_API_URL")
	}

	factory := llm.NewVerifiedFactory(cfg)
	resolver := verifier.NewProviderResolverWithEnv(getenv)

	// The factory needs the concrete API key for a selected model's provider.
	// Resolve it from the environment via the ProviderResolver so BOTH the
	// in-process (named id) and HTTP (numeric id) paths yield a real key
	// (§3.3). Never logged (§11.4.10).
	factory.SetKeyResolver(func(providerID string) string {
		rp, err := resolver.Resolve(providerID)
		if err != nil {
			return ""
		}
		return rp.APIKey
	})

	b := &Bridge{factory: factory, resolver: resolver, cfg: cfg}

	if apiURL != "" {
		// HTTP mode: operator-provided running LLMsVerifier service.
		vc := verifier.NewClient(&verifier.Config{
			APIURL:   apiURL,
			APIKey:   opts.APIKey,
			CacheTTL: cfg.CacheTTL,
		})
		if err := vc.Ping(ctx); err != nil {
			return nil, fmt.Errorf("LLMSVERIFIER_API_URL=%s is set but the service is unreachable: %w", apiURL, err)
		}
		factory.SetClient(vc)
		b.source = "http"
		return b, nil
	}

	// In-process mode (canonical OOTB): provision from env keys → SQLite store.
	if err := b.bootstrapInProcess(ctx, opts, getenv); err != nil {
		return nil, err
	}
	b.source = "in-process"
	return b, nil
}

// Source reports which provisioning source the bridge resolved ("http" or
// "in-process"). Safe to log.
func (b *Bridge) Source() string { return b.source }

// bootstrapInProcess provisions verified models from environment API keys into
// the factory's registry via the in-process pipeline + SQLite store.
func (b *Bridge) bootstrapInProcess(ctx context.Context, opts Options, getenv func(string) string) error {
	providers := verifier.ProvidersFromGetenv(getenv)
	if len(providers) == 0 {
		return fmt.Errorf(
			"no provider API keys set in the environment — set at least one of e.g. " +
				"DEEPSEEK_API_KEY / OPENAI_API_KEY / GROQ_API_KEY and re-run " +
				"(local llama.cpp fallback is not permitted)")
	}

	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = "./data/verified_models.db"
	}
	if err := ensureDir(dbPath); err != nil {
		return fmt.Errorf("prepare verified-models db dir: %w", err)
	}

	store, err := persistence.NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("open verified-models sqlite store %s: %w", dbPath, err)
	}
	defer store.Close()

	// Load any already-verified models from the store into the factory registry.
	persisted, err := store.LoadModels()
	if err != nil {
		return fmt.Errorf("load persisted verified models: %w", err)
	}
	for _, m := range persisted {
		b.factory.RegisterModel(m)
	}

	needVerify := opts.ForceVerify || len(b.selectable()) == 0
	if !needVerify {
		return nil
	}

	// Run a bounded fresh verification against the REAL provider APIs.
	maxModels := opts.MaxModelsPerProvider
	if maxModels == 0 {
		maxModels = 3
	}
	timeout := opts.VerifyTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	vctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	reg := verifier.NewRegistry()
	pipe := verifier.NewPipeline()
	if _, err := verifier.RunVerification(vctx, reg, pipe, store, providers, verifier.RunOptions{
		MaxModelsPerProvider: maxModels,
		MinScoreToPersist:    b.cfg.MinScoreThreshold,
	}); err != nil {
		return fmt.Errorf("in-process verification pass failed: %w", err)
	}

	// Reload from the store (now populated) into the factory registry.
	verified, err := store.LoadModels()
	if err != nil {
		return fmt.Errorf("reload verified models after verification: %w", err)
	}
	for _, m := range verified {
		b.factory.RegisterModel(m)
	}
	if len(b.selectable()) == 0 {
		return fmt.Errorf(
			"verification completed but no models met the verification threshold "+
				"(>%.2f score, verified+can_see_code+affirmative); check provider keys/quotas",
			b.cfg.MinScoreThreshold)
	}
	return nil
}

// selectable returns the models the factory considers verified-and-usable.
func (b *Bridge) selectable() []verifier.Model {
	return b.factory.ListVerifiedModels()
}

// BestTranslator returns the strongest verified model as a translator.Translator
// PLUS a deterministic, score-descending fallback chain of model IDs (try the
// next verified model if the top one is unreachable). It reuses
// VerifiedFactory.CreateTranslatorWithFallback verbatim (§11.4.74).
func (b *Bridge) BestTranslator(ctx context.Context, task selection.TaskRequirements) (translator.Translator, []string, error) {
	tr, fallback, err := b.factory.CreateTranslatorWithFallback(ctx, task)
	if err != nil {
		return nil, nil, fmt.Errorf("bridge: no verified translator available: %w", err)
	}
	return tr, fallback, nil
}

// BestModel returns metadata for the strongest verified model for the task. The
// returned ModelInfo carries no API key (§11.4.10).
//
// It ranks directly from the verified registry rather than routing through
// CreateTranslator: the factory's translator builder validates the model name
// against a provider's hardcoded whitelist (rejecting e.g. a verified DeepSeek
// model under the generic "openai" client), but selecting the STRONGEST verified
// model is a pure metadata operation that must not be gated on that whitelist —
// otherwise a genuinely-verified model would be reported as "none available".
func (b *Bridge) BestModel(ctx context.Context, task selection.TaskRequirements) (ModelInfo, error) {
	// In HTTP mode, refresh the registry from the SSOT first (a no-op in-process).
	if b.source == "http" {
		if _, _, err := b.factory.CreateTranslatorWithFallback(ctx, task); err != nil {
			// If the registry is empty AND the SSOT is unreachable this is fatal;
			// otherwise fall through to whatever is registered.
			if len(b.rankedModels()) == 0 {
				return ModelInfo{}, fmt.Errorf("bridge: no strongest verified model: %w", err)
			}
		}
	}
	models := b.rankedModels()
	if len(models) == 0 {
		return ModelInfo{}, fmt.Errorf("bridge: no strongest verified model: registry holds no verified models")
	}
	return models[0], nil
}

// ListVerified returns all verified models ranked strongest-first. No API keys.
func (b *Bridge) ListVerified(ctx context.Context) ([]ModelInfo, error) {
	// In HTTP mode, ensure the registry is refreshed from the SSOT first.
	if b.source == "http" {
		if _, err := b.factory.CreateTranslator(ctx, selection.TaskRequirements{}); err != nil {
			// A pure list should not hard-fail if selection's capability filter
			// rejects all; fall through to whatever is registered.
			_ = err
		}
	}
	models := b.rankedModels()
	if len(models) == 0 {
		return nil, fmt.Errorf("bridge: no verified models available")
	}
	return models, nil
}

// llmClient is the bridge's alias for the llm.LLMClient interface — the contract
// BestClient returns (Translate + GetProviderName). Aliased so the in-package
// test (and the clientBuild seam) can name it without re-importing.
type llmClient = llm.LLMClient

// clientBuild is the SINGLE raw-client construction seam: it builds the
// OpenAI-compatible chat client (wrapped so GetProviderName reflects the selected
// provider, see markedClient) from (providerMarker, modelID, rp). BOTH BestClient
// and realInvokeDispatch route through it, so there is exactly ONE construction
// path. It is a package var (not a method) ONLY so an in-package test can
// intercept the (providerMarker, modelID) a build is actually asked for WITHOUT a
// live network call, proving the adapter/route guards catch a mis-build (§1.1
// paired mutation). Production behaviour is unchanged.
var clientBuild = realClientBuild

// realClientBuild constructs the raw OpenAI-compatible chat client for the given
// (providerMarker, modelID) and resolved provider, then wraps it so its
// GetProviderName() reports the selected provider id rather than the underlying
// *OpenAIClient hardcoded "openai". No API key is logged (§11.4.10).
func realClientBuild(providerMarker, modelID string, rp *verifier.ResolvedProvider) (llmClient, error) {
	oc, err := llm.NewOpenAIClient(translator.TranslationConfig{
		Provider: providerMarker,
		Model:    modelID,
		APIKey:   rp.APIKey,
		BaseURL:  rp.BaseURL,
		// Cap max_tokens conservatively: the OpenAI client defaults to 8192 (sized
		// for book chapters), but many verified models cap lower (e.g. Groq's
		// allam-2-7b at 4096) and return HTTP 400 above their limit. Invoke is a
		// raw agent-capability call, not a long-form translation, so a modest cap
		// is both safe across providers and sufficient. Callers needing more use
		// BestTranslator + the component path.
		MaxTokens: invokeMaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("bridge: build raw client for %s: %w", modelID, err)
	}
	// Report the SELECTED model's provider id (rp.ProviderID) from GetProviderName,
	// per the plan's prescription, instead of the bare *OpenAIClient literal
	// "openai" — so a consumer depending on llm.LLMClient.GetProviderName() sees
	// the genuine provider of the strongest verified model.
	return &markedClient{LLMClient: oc, provider: rp.ProviderID}, nil
}

// markedClient wraps an llm.LLMClient (the raw *OpenAIClient) delegating Translate
// verbatim while overriding GetProviderName to report the selected provider id.
type markedClient struct {
	llm.LLMClient
	provider string
}

// GetProviderName reports the selected verified model's provider id (e.g.
// "deepseek", "groq", "openai"), not the underlying client's hardcoded "openai".
func (m *markedClient) GetProviderName() string { return m.provider }

// invokeDispatch is the seam that turns the strongest-model selection into a real
// completion. Production assigns it realInvokeDispatch (below) — building the raw
// client via clientBuild and calling Translate. It is a package var (not a method)
// ONLY so an in-package test can intercept the (providerMarker, modelID) the
// dispatch is actually asked to route to WITHOUT a live network call, proving the
// routing guard catches a mis-route (§1.1 paired mutation). Production behaviour
// is unchanged.
var invokeDispatch = realInvokeDispatch

// realInvokeDispatch builds the raw OpenAI-compatible chat client (via the shared
// clientBuild seam — ONE construction path) and calls it. providerMarker is the
// delegation marker (concrete provider id, or "openai" for the genuine OpenAI
// provider); modelID is the strongest verified model's id. The resolved provider
// rp carries the concrete key/base_url. No API key is logged.
func realInvokeDispatch(ctx context.Context, providerMarker, modelID string, rp *verifier.ResolvedProvider, full string) (string, error) {
	client, err := clientBuild(providerMarker, modelID, rp)
	if err != nil {
		return "", err
	}
	// OpenAIClient.Translate sends `prompt` (2nd arg) as the single user message
	// and ignores `text` (1st arg); pass the composed prompt there.
	return client.Translate(ctx, "", full)
}

// resolveBest selects the strongest verified model for the task, resolves its
// provider, and computes the delegation provider marker. It is the SINGLE
// selection→resolution→marker path shared by Invoke and BestClient.
//
// Provider marker MUST be the concrete provider id (e.g. "groq"), NOT the literal
// "openai": NewOpenAIClient only SKIPS its hardcoded OpenAI model-name whitelist
// when Provider is non-empty AND != "openai" (its "delegated" branch). A verified
// non-OpenAI model (allam-2-7b, deepseek-chat, …) is not in that whitelist, so
// passing "openai" would reject it. The provider id is purely a delegation marker
// here; the actual endpoint comes from BaseURL.
func (b *Bridge) resolveBest(ctx context.Context, task selection.TaskRequirements) (ModelInfo, verifier.ResolvedProvider, string, error) {
	best, err := b.BestModel(ctx, task)
	if err != nil {
		return ModelInfo{}, verifier.ResolvedProvider{}, "", err
	}
	rp, err := b.resolver.Resolve(best.ProviderID)
	if err != nil {
		return ModelInfo{}, verifier.ResolvedProvider{}, "",
			fmt.Errorf("bridge: cannot materialize strongest model %s/%s: %w", best.ProviderID, best.ModelID, err)
	}
	providerMarker := rp.ProviderID
	if providerMarker == "" || providerMarker == "openai" {
		// Genuine OpenAI provider (or empty): keep "openai" so its real model
		// whitelist applies; for every other provider the concrete id delegates.
		providerMarker = rp.FactoryProvider
	}
	return best, rp, providerMarker, nil
}

// BestClient returns the strongest verified model for the task as an
// llm.LLMClient — the no-local-runtime redirect target for consumers that depend
// on the llm.LLMClient contract (Translate + GetProviderName). The returned
// client is the IDENTICALLY-configured raw OpenAI-compatible client Invoke builds
// (same Model + provider delegation marker + key + base_url), wrapped so its
// GetProviderName() reports the selected model's provider id (BestModel.ProviderID)
// rather than the underlying *OpenAIClient hardcoded "openai".
//
// It honours the no-key hard error (Open already refuses to construct a bridge
// without provider keys) — BestClient NEVER silently falls back to a local
// runtime; an empty registry / unresolvable provider yields an honest error.
// No API key is ever returned or logged (§11.4.10).
func (b *Bridge) BestClient(ctx context.Context, task selection.TaskRequirements) (llmClient, error) {
	best, rp, providerMarker, err := b.resolveBest(ctx, task)
	if err != nil {
		return nil, err
	}
	return clientBuild(providerMarker, best.ModelID, &rp)
}

// Invoke calls the strongest verified model with a raw prompt — the direct
// agent-access path. system is prepended to prompt (the underlying OpenAI-
// compatible client sends a single user message, so we compose them). Returns
// the model's raw completion. No API key is ever returned or logged.
func (b *Bridge) Invoke(ctx context.Context, system, prompt string) (string, error) {
	best, rp, providerMarker, err := b.resolveBest(ctx, selection.TaskRequirements{})
	if err != nil {
		return "", err
	}

	full := strings.TrimSpace(prompt)
	if s := strings.TrimSpace(system); s != "" {
		full = s + "\n\n" + full
	}

	out, err := invokeDispatch(ctx, providerMarker, best.ModelID, &rp, full)
	if err != nil {
		return "", fmt.Errorf("bridge: invoke %s/%s: %w", rp.ProviderID, best.ModelID, err)
	}
	return out, nil
}

// rankedModels builds a strongest-first ModelInfo list from the factory's
// verified set, attaching FallbackOrder + resolved base_url. Reuses the
// services adapter's ranking semantics (score desc, model-id tie-break).
func (b *Bridge) rankedModels() []ModelInfo {
	verified := b.factory.ListVerifiedModels() // already verified-filtered
	infos := make([]ModelInfo, 0, len(verified))
	for _, m := range verified {
		info := ModelInfo{
			ProviderID:   m.ProviderID,
			ModelID:      m.ID,
			ModelName:    m.Name,
			Score:        m.OverallScore,
			Capabilities: m.Capabilities,
		}
		if rp, err := b.resolver.Resolve(m.ProviderID); err == nil {
			info.FactoryName = rp.FactoryProvider
			info.BaseURL = rp.BaseURL
		}
		infos = append(infos, info)
	}
	sort.SliceStable(infos, func(i, j int) bool {
		if infos[i].Score != infos[j].Score {
			return infos[i].Score > infos[j].Score
		}
		return infos[i].ModelID < infos[j].ModelID
	})
	for i := range infos {
		infos[i].FallbackOrder = i + 1
	}
	return infos
}

// compile-time assurance the services adapter type stays referenced so a future
// refactor of ProviderPreference is caught here too.
var _ = services.ProviderPreference{}
