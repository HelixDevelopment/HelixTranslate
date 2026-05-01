package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"digital.vasic.translator/internal/config"
)

// TestDefaultConfig_HasLLMsVerifier verifies default config includes verifier.
func TestDefaultConfig_HasLLMsVerifier(t *testing.T) {
	cfg := config.DefaultConfig()
	require.NotNil(t, cfg)

	assert.False(t, cfg.LLMsVerifier.Enabled)
	assert.Equal(t, "http://localhost:8080", cfg.LLMsVerifier.APIURL)
	assert.Equal(t, "./data/verifier.db", cfg.LLMsVerifier.DBPath)
	assert.Equal(t, time.Hour, cfg.LLMsVerifier.CacheTTL)
	assert.True(t, cfg.LLMsVerifier.VerificationEnabled)
	assert.Equal(t, 25, cfg.LLMsVerifier.MaxProviders)
}

// TestConfig_Validate_LLMVerifierEnabled verifies validation with verifier enabled.
func TestConfig_Validate_LLMVerifierEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Security.EnableAuth = false
	cfg.LLMsVerifier.Enabled = true
	cfg.LLMsVerifier.APIURL = ""

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API URL is required")
}

// TestConfig_Validate_LLMVerifierAPIKey verifies API key requirement.
func TestConfig_Validate_LLMVerifierAPIKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Security.EnableAuth = false
	cfg.LLMsVerifier.Enabled = true
	cfg.LLMsVerifier.APIURL = "http://localhost:8080"
	cfg.LLMsVerifier.APIKey = ""

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key is required")
}

// TestConfig_Validate_ScoringWeights verifies weight sum validation.
func TestConfig_Validate_ScoringWeights(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Security.EnableAuth = false
	cfg.LLMsVerifier.Enabled = true
	cfg.LLMsVerifier.APIURL = "http://localhost:8080"
	cfg.LLMsVerifier.APIKey = "test"
	cfg.LLMsVerifier.ScoringWeights = config.ScoreWeights{
		ResponseSpeed:     0.5,
		CostEffectiveness: 0.5,
		ModelEfficiency:   0.5,
		Capability:        0.5,
		Recency:           0.5,
	}

	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "weights must sum to 1.0")
}

// TestConfig_Validate_ValidWeights verifies acceptance of valid weights.
func TestConfig_Validate_ValidWeights(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Security.EnableAuth = false
	cfg.LLMsVerifier.Enabled = true
	cfg.LLMsVerifier.APIURL = "http://localhost:8080"
	cfg.LLMsVerifier.APIKey = "test"
	cfg.LLMsVerifier.ScoringWeights = config.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	}

	// Need valid server port and other defaults
	err := cfg.Validate()
	require.NoError(t, err)
}

// TestConfig_LoadConfig_LoadsAPIKeys verifies environment variable loading.
func TestConfig_LoadConfig_LoadsAPIKeys(t *testing.T) {
	// Set env vars
	_ = os.Setenv("OPENAI_API_KEY", "openai-test")
	_ = os.Setenv("ANTHROPIC_API_KEY", "anthropic-test")
	_ = os.Setenv("GROQ_API_KEY", "groq-test")
	defer func() {
		_ = os.Unsetenv("OPENAI_API_KEY")
		_ = os.Unsetenv("ANTHROPIC_API_KEY")
		_ = os.Unsetenv("GROQ_API_KEY")
	}()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	data := []byte(`{
		"server": {"host": "0.0.0.0", "port": 8080},
		"security": {"enable_auth": false},
		"translation": {"providers": {}},
		"preparation": {"enabled": false},
		"distributed": {"enabled": false},
		"logging": {"level": "info"},
		"llmsverifier": {"enabled": false}
	}`)
	require.NoError(t, os.WriteFile(configPath, data, 0600))

	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "openai-test", cfg.Translation.Providers["openai"].APIKey)
	assert.Equal(t, "anthropic-test", cfg.Translation.Providers["anthropic"].APIKey)
	assert.Equal(t, "groq-test", cfg.Translation.Providers["groq"].APIKey)
}

// TestConfig_LoadConfig_LLMVerifierEnv verifies LLMsVerifier env loading.
func TestConfig_LoadConfig_LLMVerifierEnv(t *testing.T) {
	_ = os.Setenv("LLMSVERIFIER_ENABLED", "true")
	_ = os.Setenv("LLMSVERIFIER_API_URL", "https://verifier.example.com")
	_ = os.Setenv("LLMSVERIFIER_API_KEY", "verifier-key")
	_ = os.Setenv("LLMSVERIFIER_CACHE_TTL", "30m")
	defer func() {
		_ = os.Unsetenv("LLMSVERIFIER_ENABLED")
		_ = os.Unsetenv("LLMSVERIFIER_API_URL")
		_ = os.Unsetenv("LLMSVERIFIER_API_KEY")
		_ = os.Unsetenv("LLMSVERIFIER_CACHE_TTL")
	}()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	data := []byte(`{
		"server": {"host": "0.0.0.0", "port": 8080},
		"security": {"enable_auth": false},
		"translation": {"providers": {}},
		"preparation": {"enabled": false},
		"distributed": {"enabled": false},
		"logging": {"level": "info"},
		"llmsverifier": {"enabled": false, "api_url": "", "api_key": ""}
	}`)
	require.NoError(t, os.WriteFile(configPath, data, 0600))

	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)

	assert.True(t, cfg.LLMsVerifier.Enabled)
	assert.Equal(t, "https://verifier.example.com", cfg.LLMsVerifier.APIURL)
	assert.Equal(t, "verifier-key", cfg.LLMsVerifier.APIKey)
	assert.Equal(t, 30*time.Minute, cfg.LLMsVerifier.CacheTTL)
}
