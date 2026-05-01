package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digital.vasic.translator/internal/verifier"
)

// Service performs three-tier model discovery.
type Service struct {
	config       *verifier.Config
	registry     *verifier.Registry
	mu           sync.RWMutex
	providers    []verifier.ProviderConfig
	lastSync     time.Time
	syncInterval time.Duration
}

// NewService creates a discovery service.
func NewService(cfg *verifier.Config, registry *verifier.Registry) *Service {
	return &Service{
		config:       cfg,
		registry:     registry,
		syncInterval: cfg.CacheTTL,
	}
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

	// Tier 2 & 3 would query public registries and community endpoints
	// These are stubbed for Phase 1 and will be implemented in Phase 4.

	s.lastSync = time.Now()
	return nil
}

// discoverFromProvider queries a single provider for available models.
func (s *Service) discoverFromProvider(ctx context.Context, provider verifier.ProviderConfig) error {
	// Stub: in Phase 4 this will perform actual HTTP discovery against provider APIs.
	// For now, register the provider so the registry knows it exists.
	if provider.ID == "" {
		return fmt.Errorf("provider ID cannot be empty")
	}
	return nil
}

// LastSync returns the timestamp of the last successful discovery.
func (s *Service) LastSync() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSync
}
