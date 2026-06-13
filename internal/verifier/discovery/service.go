package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"digital.vasic.translator/internal/verifier"
)

// Default Tier-2 public registry endpoints. Exported as defaults so that
// production uses the real registries while tests (and operators via
// Config.Options) can point them at a local/alternate endpoint.
const (
	defaultOpenRouterURL  = "https://openrouter.ai/api/v1/models"
	defaultHuggingFaceURL = "https://huggingface.co/api/models?filter=text-generation&limit=50"
)

// Service performs three-tier model discovery.
type Service struct {
	config       *verifier.Config
	registry     *verifier.Registry
	mu           sync.RWMutex
	providers    []verifier.ProviderConfig
	lastSync     time.Time
	syncInterval time.Duration
	httpClient   *http.Client

	// Tier-2 registry endpoints. Defaulted to the public registries in
	// NewService; overridable via Config.Options ("openrouter_url",
	// "huggingface_url") for operators, or directly in tests, so the suite
	// never depends on live internet reachability (§11.4.3 topology-aware).
	openRouterURL  string
	huggingFaceURL string
}

// openRouterModel represents a model from OpenRouter API.
type openRouterModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Pricing  struct {
		Prompt float64 `json:"prompt"`
		Completion float64 `json:"completion"`
	} `json:"pricing"`
	ContextLength int `json:"context_length"`
}

// openRouterResponse represents the OpenRouter models API response.
type openRouterResponse struct {
	Data []openRouterModel `json:"data"`
}

// huggingFaceModel represents a model from HuggingFace API.
type huggingFaceModel struct {
	ID        string `json:"id"`
	ModelID   string `json:"modelId"`
	Downloads int    `json:"downloads"`
	Tags      []string `json:"tags"`
}

// NewService creates a discovery service.
func NewService(cfg *verifier.Config, registry *verifier.Registry) *Service {
	s := &Service{
		config:         cfg,
		registry:       registry,
		syncInterval:   cfg.CacheTTL,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		openRouterURL:  defaultOpenRouterURL,
		huggingFaceURL: defaultHuggingFaceURL,
	}
	// Allow operators (and tests) to override the Tier-2 registry endpoints
	// without code changes; falls back to the public registries otherwise.
	if cfg.Options != nil {
		if url, ok := cfg.Options["openrouter_url"].(string); ok && url != "" {
			s.openRouterURL = url
		}
		if url, ok := cfg.Options["huggingface_url"].(string); ok && url != "" {
			s.huggingFaceURL = url
		}
	}
	return s
}

// RegisterProvider adds a provider for discovery.
func (s *Service) RegisterProvider(cfg verifier.ProviderConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers = append(s.providers, cfg)
	s.registry.RegisterProvider(cfg)
}

// Discover runs the three-tier discovery pipeline.
func (s *Service) Discover(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Tier 1: Provider APIs
	for _, provider := range s.providers {
		if err := s.discoverFromProvider(ctx, provider); err != nil {
			// Log but continue — partial discovery is acceptable
			_ = err
		}
	}

	// Tier 2: Public registries
	if err := s.discoverFromOpenRouter(ctx); err != nil {
		_ = err
	}
	if err := s.discoverFromHuggingFace(ctx); err != nil {
		_ = err
	}

	// Tier 3: Community endpoints
	if err := s.discoverFromCommunity(ctx); err != nil {
		_ = err
	}

	s.lastSync = time.Now()
	return nil
}

// discoverFromProvider queries a single provider for available models.
func (s *Service) discoverFromProvider(ctx context.Context, provider verifier.ProviderConfig) error {
	if provider.ID == "" {
		return fmt.Errorf("provider ID cannot be empty")
	}

	// Register models declared in provider config
	for _, modelID := range provider.Models {
		s.registry.AddModel(verifier.Model{
			ID:                 modelID,
			ProviderID:         provider.ID,
			Name:               modelID,
			VerificationStatus: "discovered",
			Capabilities:       map[string]bool{},
		})
	}

	return nil
}

// discoverFromOpenRouter queries OpenRouter's public model registry (Tier 2).
func (s *Service) discoverFromOpenRouter(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.openRouterURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create OpenRouter request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("OpenRouter unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenRouter returned status %d", resp.StatusCode)
	}

	var result openRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode OpenRouter response: %w", err)
	}

	for _, m := range result.Data {
		s.registry.AddModel(verifier.Model{
			ID:                 m.ID,
			ProviderID:         "openrouter",
			Name:               m.Name,
			VerificationStatus: "discovered",
			Capabilities:       map[string]bool{"streaming": true},
			Pricing: verifier.PricingInfo{
				InputTokenCost:  m.Pricing.Prompt,
				OutputTokenCost: m.Pricing.Completion,
				Currency:        "USD",
			},
		})
	}

	return nil
}

// discoverFromHuggingFace queries HuggingFace's model registry (Tier 2).
func (s *Service) discoverFromHuggingFace(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.huggingFaceURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HuggingFace request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HuggingFace unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HuggingFace returned status %d", resp.StatusCode)
	}

	var models []huggingFaceModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return fmt.Errorf("failed to decode HuggingFace response: %w", err)
	}

	for _, m := range models {
		caps := map[string]bool{}
		for _, tag := range m.Tags {
			if tag == "transformers" || tag == "pytorch" {
				caps["local"] = true
			}
		}
		// "id" is the HuggingFace API's canonical, contract-guaranteed model
		// identifier; "modelId" is a legacy alias that may be absent. Prefer the
		// canonical field and only fall back to the alias, otherwise a valid
		// payload supplying only "id" registered every model under the empty
		// string and silently collapsed to zero usable models.
		modelID := m.ID
		if modelID == "" {
			modelID = m.ModelID
		}
		if modelID == "" {
			// No usable identifier at all — skip rather than collide on "".
			continue
		}
		s.registry.AddModel(verifier.Model{
			ID:                 modelID,
			ProviderID:         "huggingface",
			Name:               modelID,
			VerificationStatus: "discovered",
			Capabilities:       caps,
		})
	}

	return nil
}

// discoverFromCommunity queries configured community endpoints (Tier 3).
func (s *Service) discoverFromCommunity(ctx context.Context) error {
	// Check if a community endpoint is configured in options
	communityURL := ""
	if url, ok := s.config.Options["community_registry_url"].(string); ok && url != "" {
		communityURL = url
	}

	if communityURL == "" {
		// No community endpoint configured; skip Tier 3 silently
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, communityURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create community request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("community registry unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("community registry returned status %d", resp.StatusCode)
	}

	var models []verifier.Model
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return fmt.Errorf("failed to decode community response: %w", err)
	}

	for _, m := range models {
		s.registry.AddModel(m)
	}

	return nil
}

// LastSync returns the timestamp of the last successful discovery.
func (s *Service) LastSync() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync
}
