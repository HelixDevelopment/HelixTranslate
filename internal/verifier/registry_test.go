package verifier

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestModel(id string, score float64, status string) Model {
	return Model{
		ID:                  id,
		ProviderID:          "test",
		Name:                id,
		VerificationStatus:  status,
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        score,
		Capabilities:        map[string]bool{},
	}
}

func TestRegistryRegisterProvider(t *testing.T) {
	r := NewRegistry()
	r.RegisterProvider(ProviderConfig{ID: "openai", APIKey: "test", BaseURL: "https://api.openai.com"})

	// Providers are not directly listable, but registration should not panic
	assert.NotNil(t, r)
}

func TestRegistryAddModel(t *testing.T) {
	r := NewRegistry()
	m := makeTestModel("gpt-4", 9.0, "verified")
	r.AddModel(m)

	result, ok := r.GetModel("gpt-4")
	require.True(t, ok)
	assert.Equal(t, "gpt-4", result.ID)
}

func TestRegistryGetModelNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.GetModel("nonexistent")
	assert.False(t, ok)
}

func TestRegistryListModels(t *testing.T) {
	r := NewRegistry()
	r.AddModel(makeTestModel("gpt-4", 9.0, "verified"))
	r.AddModel(makeTestModel("claude-3", 8.5, "verified"))

	models := r.ListModels()
	require.Len(t, models, 2)
}

func TestRegistryFilterVerified(t *testing.T) {
	r := NewRegistry()
	r.AddModel(makeTestModel("gpt-4", 9.0, "verified"))
	r.AddModel(makeTestModel("claude-3", 5.0, "verified"))
	r.AddModel(Model{
		ID:                  "unverified",
		VerificationStatus:  "pending",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        9.0,
	})
	r.AddModel(Model{
		ID:                  "no-code",
		VerificationStatus:  "verified",
		CanSeeCode:          false,
		AffirmativeResponse: true,
		OverallScore:        9.0,
	})

	verified := r.FilterVerified(0)
	require.Len(t, verified, 2)
	assert.Equal(t, "gpt-4", verified[0].ID)
	assert.Equal(t, "claude-3", verified[1].ID)

	verifiedHigh := r.FilterVerified(6.0)
	require.Len(t, verifiedHigh, 1)
	assert.Equal(t, "gpt-4", verifiedHigh[0].ID)
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			r.AddModel(makeTestModel(string(rune('a'+idx%26)), float64(idx), "verified"))
		}(i)
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.ListModels()
		}()
	}

	wg.Wait()
	// Should not deadlock or race
	assert.GreaterOrEqual(t, len(r.ListModels()), 26)
}

func TestRegistryFilterVerifiedMissingFlags(t *testing.T) {
	r := NewRegistry()
	r.AddModel(Model{
		ID:                  "verified-good",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        8.0,
	})
	r.AddModel(Model{
		ID:                  "verified-no-code",
		VerificationStatus:  "verified",
		CanSeeCode:          false,
		AffirmativeResponse: true,
		OverallScore:        8.0,
	})
	r.AddModel(Model{
		ID:                  "verified-no-affirm",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: false,
		OverallScore:        8.0,
	})

	verified := r.FilterVerified(0)
	require.Len(t, verified, 1)
	assert.Equal(t, "verified-good", verified[0].ID)
}
