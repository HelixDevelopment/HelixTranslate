package verifier

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"digital.vasic.llmsverifier/pkg/api"
)

// Client communicates with the LLMsVerifier REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	mu         sync.RWMutex
	models     []Model
	lastFetch  time.Time
	cacheTTL   time.Duration
}

// Model is the canonical verified model type from LLMsVerifier (CONST-034 SSOT).
type Model = api.Model

// PricingInfo is the canonical pricing type from LLMsVerifier.
type PricingInfo = api.PricingInfo

// NewClient creates a new LLMsVerifier client.
func NewClient(cfg *Config) *Client {
	return &Client{
		baseURL: cfg.APIURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		cacheTTL: cfg.CacheTTL,
	}
}

// Ping checks connectivity to the LLMsVerifier API.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrLLMsVerifierUnreachable{URL: c.baseURL}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LLMsVerifier health check returned %d", resp.StatusCode)
	}
	return nil
}

// GetVerifiedModels returns all verified models from LLMsVerifier.
func (c *Client) GetVerifiedModels(ctx context.Context) ([]Model, error) {
	// Cache validity is keyed on a SUCCESSFUL prior fetch (lastFetch != zero),
	// NOT on len(c.models) > 0. A legitimate zero-verified-models response is a
	// cacheable fact; gating on slice length meant the empty result was never
	// cached and every call re-hit the upstream within the TTL window, silently
	// defeating CacheTTL. InvalidateCache resets lastFetch to the zero value to
	// force a refetch.
	c.mu.RLock()
	if !c.lastFetch.IsZero() && time.Since(c.lastFetch) < c.cacheTTL {
		models := make([]Model, len(c.models))
		copy(models, c.models)
		c.mu.RUnlock()
		return models, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if !c.lastFetch.IsZero() && time.Since(c.lastFetch) < c.cacheTTL {
		models := make([]Model, len(c.models))
		copy(models, c.models)
		return models, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create models request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrLLMsVerifierUnreachable{URL: c.baseURL}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLMsVerifier models endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read models response: %w", err)
	}

	models, err := decodeModelsResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	c.models = models
	c.lastFetch = time.Now()
	return models, nil
}

// serverModel mirrors the field shape the LLMsVerifier server's /api/models
// endpoint emits (envelope object + per-model field names that differ from
// api.Model). The server is the decoupled SSOT; the System adapts to it here.
type serverModel struct {
	ID           int64           `json:"id"`
	ModelID      string          `json:"model_id"`
	Name         string          `json:"name"`
	Provider     string          `json:"provider"`
	ProviderID   int64           `json:"provider_id"`
	Status       string          `json:"status"`
	Score        float64         `json:"score"`
	Capabilities []string        `json:"capabilities"`
	Description  string          `json:"description"`
	Version      string          `json:"version"`
	Deprecated   bool            `json:"deprecated"`
}

// decodeModelsResponse accepts BOTH the server's documented envelope
// ({"models":[...],"count":N}) with its server-side field names AND a bare
// []api.Model array (forward-compatible), mapping either onto []Model.
func decodeModelsResponse(body []byte) ([]Model, error) {
	// Preferred: the server envelope with server-side field names. The envelope is
	// a JSON OBJECT, so a successful unmarshal into the struct (err == nil) is the
	// discriminator that the body IS the envelope — a bare array errors here
	// ("cannot unmarshal array into ... struct") and correctly falls through to
	// the array fallback below. A present-but-null models key
	// ({"models":null,"count":0} — what the server emits for a zero-length
	// verified set, since Go marshals a nil slice as JSON null) yields a nil
	// Models slice but a successful struct unmarshal, so it MUST be handled as a
	// valid empty result here rather than wrongly forced down the bare-array
	// fallback (which then fails: "cannot unmarshal object into []"). Gating on
	// `env.Models != nil` conflated "empty envelope" with "bare array" and broke
	// /api/v1/verified-models whenever zero models were verified.
	var env struct {
		Models []serverModel `json:"models"`
		Count  int           `json:"count"`
	}
	if err := json.Unmarshal(body, &env); err == nil {
		out := make([]Model, 0, len(env.Models))
		for _, sm := range env.Models {
			caps := make(map[string]bool, len(sm.Capabilities))
			for _, cap := range sm.Capabilities {
				caps[cap] = true
			}
			id := sm.ModelID
			if id == "" {
				id = fmt.Sprintf("%d", sm.ID)
			}
			out = append(out, Model{
				ID:                 id,
				ProviderID:         fmt.Sprintf("%d", sm.ProviderID),
				Name:               sm.Name,
				VerificationStatus: sm.Status,
				OverallScore:       sm.Score,
				Capabilities:       caps,
			})
		}
		return out, nil
	}

	// Forward-compatible fallback: a bare []api.Model array.
	var models []Model
	if err := json.Unmarshal(body, &models); err != nil {
		return nil, err
	}
	return models, nil
}

// GetModel returns a specific model by ID.
func (c *Client) GetModel(ctx context.Context, modelID string) (*Model, error) {
	models, err := c.GetVerifiedModels(ctx)
	if err != nil {
		return nil, err
	}

	for i := range models {
		if models[i].ID == modelID {
			return &models[i], nil
		}
	}
	return nil, ErrModelNotVerified{ModelID: modelID}
}

// InvalidateCache forces a refresh on the next models call.
func (c *Client) InvalidateCache() {
	c.mu.Lock()
	c.lastFetch = time.Time{}
	c.mu.Unlock()
}
