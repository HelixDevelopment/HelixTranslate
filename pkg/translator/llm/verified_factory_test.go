package llm

import (
	"context"
	"testing"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/selection"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifiedFactoryRegisterAndList(t *testing.T) {
	factory := NewVerifiedFactory(verifier.DefaultConfig())

	require.Empty(t, factory.ListVerifiedModels())

	factory.RegisterModel(verifier.Model{
		ID:                  "gpt-4",
		ProviderID:          "openai",
		Name:                "GPT-4",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        0.95,
		Capabilities:        map[string]bool{"streaming": true},
	})

	models := factory.ListVerifiedModels()
	require.Len(t, models, 1)
	assert.Equal(t, "gpt-4", models[0].ID)
	assert.True(t, factory.IsModelVerified("gpt-4"))
	assert.False(t, factory.IsModelVerified("gpt-5"))
}

func TestVerifiedFactoryKeyResolver(t *testing.T) {
	factory := NewVerifiedFactory(verifier.DefaultConfig())

	factory.RegisterModel(verifier.Model{
		ID:                  "gpt-4",
		ProviderID:          "openai",
		Name:                "GPT-4",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        0.95,
	})

	// Without key resolver, CreateTranslator should fail because APIKey is empty
	_, err := factory.CreateTranslator(context.Background(), selection.TaskRequirements{})
	require.Error(t, err)

	// With key resolver returning a key, it may still fail due to model validation
	// but it proves the resolver is wired
	factory.SetKeyResolver(func(providerID string) string {
		if providerID == "openai" {
			return "test-key"
		}
		return ""
	})

	// The factory will attempt to create a translator. Since "gpt-4" is in ValidModels
	// and we provide an API key, it should succeed if the model is valid.
	trans, err := factory.CreateTranslator(context.Background(), selection.TaskRequirements{})
	require.NoError(t, err)
	require.NotNil(t, trans)
	assert.Equal(t, "llm-openai", trans.GetName())
}

func TestVerifiedFactoryCreateTranslatorNoModels(t *testing.T) {
	factory := NewVerifiedFactory(verifier.DefaultConfig())
	_, err := factory.CreateTranslator(context.Background(), selection.TaskRequirements{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no verified model available")
}

func TestVerifiedFactoryCreateTranslatorWithFallback(t *testing.T) {
	factory := NewVerifiedFactory(verifier.DefaultConfig())

	factory.RegisterModel(verifier.Model{
		ID:                  "gpt-4",
		ProviderID:          "openai",
		Name:                "GPT-4",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        0.95,
	})
	factory.RegisterModel(verifier.Model{
		ID:                  "claude-3-opus-20240229",
		ProviderID:          "anthropic",
		Name:                "Claude 3 Opus",
		VerificationStatus:  "verified",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		OverallScore:        0.92,
	})

	factory.SetKeyResolver(func(providerID string) string {
		return "test-key"
	})

	trans, fallbacks, err := factory.CreateTranslatorWithFallback(context.Background(), selection.TaskRequirements{})
	require.NoError(t, err)
	require.NotNil(t, trans)
	require.NotEmpty(t, fallbacks)
}

func TestVerifiedFactoryRegisterProvider(t *testing.T) {
	factory := NewVerifiedFactory(verifier.DefaultConfig())
	factory.RegisterProvider(verifier.ProviderConfig{
		ID:      "custom",
		BaseURL: "http://localhost:8080",
		APIKey:  "key",
	})
	// Registering a provider alone does not create models
	assert.Empty(t, factory.ListVerifiedModels())
}
