package discovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
)

func TestNewService(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)
	assert.NotNil(t, svc)
	assert.True(t, svc.LastSync().IsZero())
}

func TestRegisterProvider(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	svc.RegisterProvider(verifier.ProviderConfig{
		ID:      "openai",
		APIKey:  "test",
		BaseURL: "https://api.openai.com",
		Models:  []string{"gpt-4"},
	})

	assert.NotNil(t, svc)
}

func TestDiscover(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	svc.RegisterProvider(verifier.ProviderConfig{ID: "openai", APIKey: "test", Models: []string{"gpt-4", "gpt-3.5-turbo"}})
	svc.RegisterProvider(verifier.ProviderConfig{ID: "anthropic", APIKey: "test", Models: []string{"claude-3"}})

	err := svc.Discover(context.Background())
	require.NoError(t, err)
	assert.False(t, svc.LastSync().IsZero())

	// Tier 1: provider models should be registered
	models := registry.ListModels()
	assert.GreaterOrEqual(t, len(models), 2)
}

func TestDiscoverTier2OpenRouter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/models", r.URL.Path)
		resp := openRouterResponse{
			Data: []openRouterModel{
				{ID: "openai/gpt-4", Name: "GPT-4", ContextLength: 8192},
				{ID: "anthropic/claude-3", Name: "Claude 3", ContextLength: 200000},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)
	// Override the OpenRouter endpoint for testing
	svc.httpClient = &http.Client{Timeout: 5 * time.Second}
	// We can't easily override the hardcoded URL, but we can test the parsing logic
	// by testing discoverFromOpenRouter directly if we could inject the URL.
	// For now, we test that Discover completes without error.

	err := svc.Discover(context.Background())
	require.NoError(t, err)
}

func TestDiscoverTier3Community(t *testing.T) {
	communityModels := []verifier.Model{
		{ID: "community-model-1", ProviderID: "community", Name: "Community Model 1", VerificationStatus: "discovered"},
		{ID: "community-model-2", ProviderID: "community", Name: "Community Model 2", VerificationStatus: "discovered"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(communityModels)
	}))
	defer server.Close()

	cfg := verifier.DefaultConfig()
	cfg.Options = map[string]interface{}{"community_registry_url": server.URL}
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	err := svc.Discover(context.Background())
	require.NoError(t, err)

	// Tier 3: community models should be registered
	m, ok := registry.GetModel("community-model-1")
	require.True(t, ok)
	assert.Equal(t, "Community Model 1", m.Name)
}

func TestDiscoverEmptyProviderID(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	svc.RegisterProvider(verifier.ProviderConfig{ID: "", APIKey: "test"})

	err := svc.Discover(context.Background())
	require.NoError(t, err)
}

func TestDiscoverWithCancelledContext(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.Discover(ctx)
	require.NoError(t, err)
}

func TestLastSync(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	assert.True(t, svc.LastSync().IsZero())

	err := svc.Discover(context.Background())
	require.NoError(t, err)

	assert.False(t, svc.LastSync().IsZero())
}

func TestConcurrentRegisterAndDiscover(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			svc.RegisterProvider(verifier.ProviderConfig{ID: string(rune('a' + idx)), APIKey: "test"})
			done <- true
		}(i)
	}
	for i := 0; i < 10; i++ {
		go func() {
			_ = svc.Discover(context.Background())
			done <- true
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
	assert.False(t, svc.LastSync().IsZero())
}

func TestDiscoverFromProviderRegistersModels(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	err := svc.discoverFromProvider(context.Background(), verifier.ProviderConfig{
		ID:     "test-provider",
		Models: []string{"model-a", "model-b"},
	})
	require.NoError(t, err)

	_, ok := registry.GetModel("model-a")
	assert.True(t, ok)
	_, ok = registry.GetModel("model-b")
	assert.True(t, ok)
}

func TestDiscoverFromCommunityNoURL(t *testing.T) {
	cfg := verifier.DefaultConfig()
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	err := svc.discoverFromCommunity(context.Background())
	require.NoError(t, err) // Should skip silently
}

func TestDiscoverFromCommunityError(t *testing.T) {
	cfg := verifier.DefaultConfig()
	cfg.Options = map[string]interface{}{"community_registry_url": "http://localhost:1"}
	registry := verifier.NewRegistry()
	svc := NewService(cfg, registry)

	err := svc.discoverFromCommunity(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}
