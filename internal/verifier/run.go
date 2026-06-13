package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// ModelSink is the persistence contract RunVerification writes verified models
// to. The persistence.SQLiteStore satisfies it. Declared as an interface here
// (rather than importing the persistence package) to keep the dependency
// direction persistence -> verifier and avoid an import cycle.
type ModelSink interface {
	SaveModel(m Model) error
}

// ProviderVerification aggregates the per-model verification outcomes for one
// provider during a RunVerification pass.
type ProviderVerification struct {
	ProviderID       string
	BaseURL          string
	ReachabilityPass bool
	AuthPass         bool
	AuthStatusCode   int    // best-effort HTTP status from the auth/models probe
	AuthError        string // populated when AuthPass is false
	CandidateModels  []string
	ModelResults     []*ModelVerification
	VerifiedCount    int // models that passed the full pipeline AND were persisted
}

// RunResult is the full outcome of a RunVerification pass.
type RunResult struct {
	Providers     []ProviderVerification
	TotalVerified int // total models persisted across all providers
	StartedAt     time.Time
	FinishedAt    time.Time
}

// RunOptions tunes a RunVerification pass.
type RunOptions struct {
	// MaxModelsPerProvider caps how many discovered models are run through the
	// full 8-step pipeline per provider (0 = no cap). Real provider catalogs can
	// be large; the cap bounds wall-clock + token spend for a verification pass.
	MaxModelsPerProvider int
	// MinScoreToPersist is the floor a model's overall score must exceed to be
	// persisted as "verified" (mirrors Registry.FilterVerified semantics).
	MinScoreToPersist float64
}

// RunVerification registers the given providers into the registry, derives each
// provider's candidate model list from its live "/models" endpoint, runs the
// 8-step Pipeline against the REAL provider APIs, scores each model, and
// persists the models that pass into the supplied ModelSink. It returns a
// structured RunResult capturing the real per-provider reachability/auth
// outcomes and the verified counts.
//
// Providers with an empty BaseURL are skipped. A provider whose models endpoint
// is unreachable or unauthorized yields zero candidate models (honest empty),
// never fabricated model IDs (§11.4 / §11.4.123). API-key values are never
// logged or persisted (§11.4.10).
func RunVerification(
	ctx context.Context,
	registry *Registry,
	pipeline *Pipeline,
	sink ModelSink,
	providers []ProviderConfig,
	opts RunOptions,
) (*RunResult, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	if pipeline == nil {
		return nil, fmt.Errorf("pipeline is nil")
	}
	if sink == nil {
		return nil, fmt.Errorf("model sink is nil")
	}

	result := &RunResult{StartedAt: time.Now()}

	for _, provider := range providers {
		pv := ProviderVerification{ProviderID: provider.ID, BaseURL: provider.BaseURL}
		if provider.BaseURL == "" {
			pv.AuthError = "empty base URL"
			result.Providers = append(result.Providers, pv)
			continue
		}

		registry.RegisterProvider(provider)

		// Derive real candidate models from the provider's live catalog.
		models, status, err := discoverProviderModels(ctx, pipeline.httpClient, provider)
		pv.AuthStatusCode = status
		if err != nil {
			pv.AuthError = err.Error()
		}
		// Reachability + auth are inferred from the catalog probe: any HTTP
		// status means reachable; a non-401/403 status means the key authed.
		pv.ReachabilityPass = status != 0
		pv.AuthPass = status != 0 && status != http.StatusUnauthorized && status != http.StatusForbidden

		sort.Strings(models)
		if opts.MaxModelsPerProvider > 0 && len(models) > opts.MaxModelsPerProvider {
			models = models[:opts.MaxModelsPerProvider]
		}
		pv.CandidateModels = models

		for _, modelID := range models {
			mv := pipeline.Verify(ctx, provider, modelID)
			pv.ModelResults = append(pv.ModelResults, mv)

			// Register the discovered model regardless of pass, so the registry
			// reflects the real catalog.
			registry.AddModel(Model{
				ID:                 modelID,
				ProviderID:         provider.ID,
				Name:               modelID,
				VerificationStatus: verificationStatusFor(mv),
				OverallScore:       mv.Overall,
				Capabilities:       capsFromVerification(mv),
				LastVerifiedAt:     mv.VerifiedAt,
			})

			if mv.Passed && mv.Overall > opts.MinScoreToPersist {
				m := Model{
					ID:                  modelID,
					ProviderID:          provider.ID,
					Name:                modelID,
					VerificationStatus:  "verified",
					CanSeeCode:          true,
					AffirmativeResponse: responseFormatPassed(mv),
					OverallScore:        mv.Overall,
					Capabilities:        capsFromVerification(mv),
					LastVerifiedAt:      mv.VerifiedAt,
				}
				if err := sink.SaveModel(m); err != nil {
					return result, fmt.Errorf("persist model %s/%s: %w", provider.ID, modelID, err)
				}
				pv.VerifiedCount++
				result.TotalVerified++
			}
		}

		result.Providers = append(result.Providers, pv)
	}

	result.FinishedAt = time.Now()
	return result, nil
}

// discoverProviderModels queries the provider's OpenAI-compatible "/models"
// endpoint and returns the real model IDs it advertises, the HTTP status, and
// any transport error. It NEVER invents model IDs — an unreachable or
// non-200 endpoint yields an empty slice.
func discoverProviderModels(ctx context.Context, client *http.Client, provider ProviderConfig) ([]string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.BaseURL+"/models", nil)
	if err != nil {
		return nil, 0, err
	}
	if provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode models list: %w", err)
	}

	ids := make([]string, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, resp.StatusCode, nil
}

func verificationStatusFor(mv *ModelVerification) string {
	if mv.Passed {
		return "verified"
	}
	return "discovered"
}

func responseFormatPassed(mv *ModelVerification) bool {
	for _, s := range mv.Steps {
		if s.Step == "response_format" {
			return s.Passed
		}
	}
	return false
}

// capsFromVerification extracts the capability map detected by the pipeline's
// capabilities step into the Model.Capabilities shape.
func capsFromVerification(mv *ModelVerification) map[string]bool {
	caps := map[string]bool{}
	for _, s := range mv.Steps {
		if s.Step != "capabilities" || s.Details == nil {
			continue
		}
		raw, ok := s.Details["capabilities"]
		if !ok {
			continue
		}
		if cm, ok := raw.(map[string]bool); ok {
			for k, v := range cm {
				caps[k] = v
			}
		}
	}
	return caps
}
