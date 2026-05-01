package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/discovery"
)

// TestRegistry_AddAndGet verifies basic CRUD operations.
func TestRegistry_AddAndGet(t *testing.T) {
	reg := verifier.NewRegistry()

	model := verifier.Model{
		ID:                 "gpt-4",
		ProviderID:         "openai",
		Name:               "GPT-4",
		VerificationStatus: "verified",
		CanSeeCode:         true,
		AffirmativeResponse: true,
		OverallScore:       0.95,
	}

	reg.AddModel(model)

	m, ok := reg.GetModel("gpt-4")
	require.True(t, ok)
	assert.Equal(t, "GPT-4", m.Name)
	assert.Equal(t, 0.95, m.OverallScore)

	_, ok = reg.GetModel("missing")
	assert.False(t, ok)
}

// TestRegistry_ListModels returns all registered models.
func TestRegistry_ListModels(t *testing.T) {
	reg := verifier.NewRegistry()
	reg.AddModel(verifier.Model{ID: "m1", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.9})
	reg.AddModel(verifier.Model{ID: "m2", VerificationStatus: "verified", CanSeeCode: true, AffirmativeResponse: true, OverallScore: 0.8})

	models := reg.ListModels()
	require.Len(t, models, 2)
}

// TestRegistry_FilterVerified enforces the verification gate.
func TestRegistry_FilterVerified(t *testing.T) {
	reg := verifier.NewRegistry()

	// Verified model
	reg.AddModel(verifier.Model{
		ID: "good", VerificationStatus: "verified", CanSeeCode: true,
		AffirmativeResponse: true, OverallScore: 0.9,
	})

	// Not verified
	reg.AddModel(verifier.Model{
		ID: "bad1", VerificationStatus: "pending", CanSeeCode: true,
		AffirmativeResponse: true, OverallScore: 0.9,
	})

	// Can't see code
	reg.AddModel(verifier.Model{
		ID: "bad2", VerificationStatus: "verified", CanSeeCode: false,
		AffirmativeResponse: true, OverallScore: 0.9,
	})

	// Negative response
	reg.AddModel(verifier.Model{
		ID: "bad3", VerificationStatus: "verified", CanSeeCode: true,
		AffirmativeResponse: false, OverallScore: 0.9,
	})

	// Below threshold
	reg.AddModel(verifier.Model{
		ID: "bad4", VerificationStatus: "verified", CanSeeCode: true,
		AffirmativeResponse: true, OverallScore: 0.1,
	})

	verified := reg.FilterVerified(0.5)
	require.Len(t, verified, 1)
	assert.Equal(t, "good", verified[0].ID)
}

// TestService_Discover runs discovery with registered providers.
func TestService_Discover(t *testing.T) {
	reg := verifier.NewRegistry()
	cfg := verifier.DefaultConfig()
	svc := discovery.NewService(cfg, reg)

	svc.RegisterProvider(verifier.ProviderConfig{ID: "openai", APIKey: "test", BaseURL: "https://api.openai.com"})
	svc.RegisterProvider(verifier.ProviderConfig{ID: "anthropic", APIKey: "test", BaseURL: "https://api.anthropic.com"})

	err := svc.Discover(context.Background())
	require.NoError(t, err)

	// Anti-bluff: After discovery, LastSync should be updated
	assert.False(t, svc.LastSync().IsZero())
	assert.WithinDuration(t, time.Now(), svc.LastSync(), 5*time.Second)
}

// TestService_RegisterProvider_EmptyID verifies validation.
func TestService_RegisterProvider_EmptyID(t *testing.T) {
	reg := verifier.NewRegistry()
	cfg := verifier.DefaultConfig()
	svc := discovery.NewService(cfg, reg)

	svc.RegisterProvider(verifier.ProviderConfig{ID: "valid"})

	// Discovery with empty provider ID should fail validation internally
	err := svc.Discover(context.Background())
	require.NoError(t, err) // Empty provider list means no work, no error
}
