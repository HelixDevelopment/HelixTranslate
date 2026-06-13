package verifier

import (
	"sort"
	"sync"
)

// sortModels orders models deterministically: highest OverallScore first, with
// model ID as the ascending tie-breaker. The registry's backing store is a Go
// map whose iteration order is randomized per run, so callers (and tests) that
// rely on order must receive a stable, useful ordering rather than map order.
func sortModels(models []Model) {
	sort.Slice(models, func(i, j int) bool {
		if models[i].OverallScore != models[j].OverallScore {
			return models[i].OverallScore > models[j].OverallScore
		}
		return models[i].ID < models[j].ID
	})
}

// Registry holds discovered and verified models.
type Registry struct {
	mu        sync.RWMutex
	models    map[string]Model
	providers map[string]ProviderConfig
}

// NewRegistry creates an empty model registry.
func NewRegistry() *Registry {
	return &Registry{
		models:    make(map[string]Model),
		providers: make(map[string]ProviderConfig),
	}
}

// RegisterProvider registers a provider in the registry.
func (r *Registry) RegisterProvider(cfg ProviderConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[cfg.ID] = cfg
}

// AddModel adds a verified model to the registry.
func (r *Registry) AddModel(model Model) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[model.ID] = model
}

// GetModel retrieves a model by ID.
func (r *Registry) GetModel(modelID string) (Model, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[modelID]
	return m, ok
}

// ListModels returns all registered models.
func (r *Registry) ListModels() []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Model, 0, len(r.models))
	for _, m := range r.models {
		result = append(result, m)
	}
	sortModels(result)
	return result
}

// FilterVerified returns models that have passed verification.
func (r *Registry) FilterVerified(minScore float64) []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Model, 0, len(r.models))
	for _, m := range r.models {
		if m.VerificationStatus == "verified" && m.CanSeeCode && m.AffirmativeResponse && m.OverallScore > minScore {
			result = append(result, m)
		}
	}
	sortModels(result)
	return result
}
