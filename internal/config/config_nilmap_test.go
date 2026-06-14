package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig_NilProvidersMapWithEnvKey is a REPRODUCE-FIRST RED test for a
// crash bug in loadAPIKeysFromEnv (config.go else-branch on the providers map).
//
// LoadConfig unmarshals config.json into a ZERO-VALUE Config, NOT DefaultConfig().
// When the file omits the "translation.providers" section (a perfectly valid,
// minimal config), c.Translation.Providers stays nil. loadAPIKeysFromEnv then
// writes a new provider entry into that nil map whenever any provider API-key
// env var is set (e.g. OPENAI_API_KEY) -> "panic: assignment to entry in nil
// map", crashing LoadConfig for a real-world minimal-config + exported-key setup.
//
// Before the fix this test PANICS (an unrecovered panic FAILs the test). After
// the fix, LoadConfig must succeed and the env key must land in a freshly
// created provider entry.
func TestLoadConfig_NilProvidersMapWithEnvKey(t *testing.T) {
	// Minimal config with NO translation.providers section.
	configJSON := `{
  "server": {"port": 9000, "enable_http3": false},
  "security": {"enable_auth": false}
}`

	tmpFile, err := os.CreateTemp("", "config-nilmap-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString(configJSON)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())

	t.Setenv("OPENAI_API_KEY", "env-openai-key")

	// Must NOT panic, must NOT error.
	config, err := LoadConfig(tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, config)

	// The env key must be applied into a newly-created provider entry.
	require.NotNil(t, config.Translation.Providers)
	assert.Equal(t, "env-openai-key", config.Translation.Providers["openai"].APIKey)
}
