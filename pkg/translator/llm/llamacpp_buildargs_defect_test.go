package llm

import (
	"testing"

	"digital.vasic.translator/pkg/hardware"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// argValue returns the argument immediately following flag in args, or "".
func argValue(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

// TestLlamaCppBuildArgs_TypedConfigFieldsHonored is the reproduce-first
// (§11.4.146) RED for the llama.cpp CLI plumbing defect: NewLLMTranslator wires
// the factory's llamacpp provider to NewLlamaCppClient, whose Translate hardcoded
// -n 4096 and --temp 0.3, ignoring the operator's -temperature / -max-tokens CLI
// flags (TranslationConfig.Temperature / .MaxTokens). Sibling clients honor them;
// llama.cpp silently capped output at 4096 tokens and forced temp 0.3.
func TestLlamaCppBuildArgs_TypedConfigFieldsHonored(t *testing.T) {
	c := &LlamaCppClient{
		config:       TranslationConfig{Provider: "llamacpp", Temperature: 0.7, MaxTokens: 8000},
		modelPath:    "/tmp/model.gguf",
		hardwareCaps: &hardware.Capabilities{HasGPU: false},
		threads:      4,
		contextSize:  8192,
	}

	args := c.buildArgs("translate this")

	assert.Equal(t, "8000", argValue(args, "-n"),
		"llama.cpp must honor typed -max-tokens, not hardcode 4096")
	assert.Equal(t, "0.7", argValue(args, "--temp"),
		"llama.cpp must honor typed -temperature, not hardcode 0.3")
}

// TestLlamaCppBuildArgs_OptionsOverrideTypedFields proves Options still wins.
func TestLlamaCppBuildArgs_OptionsOverrideTypedFields(t *testing.T) {
	c := &LlamaCppClient{
		config: TranslationConfig{
			Provider: "llamacpp", Temperature: 0.7, MaxTokens: 8000,
			Options: map[string]interface{}{"temperature": 0.1, "max_tokens": 123},
		},
		modelPath:    "/tmp/model.gguf",
		hardwareCaps: &hardware.Capabilities{HasGPU: false},
		threads:      4,
		contextSize:  8192,
	}

	args := c.buildArgs("x")

	assert.Equal(t, "123", argValue(args, "-n"), "Options max_tokens overrides typed field")
	assert.Equal(t, "0.1", argValue(args, "--temp"), "Options temperature overrides typed field")
}

// TestLlamaCppBuildArgs_DefaultsWhenNothingSet keeps the documented defaults.
func TestLlamaCppBuildArgs_DefaultsWhenNothingSet(t *testing.T) {
	c := &LlamaCppClient{
		config:       TranslationConfig{Provider: "llamacpp"},
		modelPath:    "/tmp/model.gguf",
		hardwareCaps: &hardware.Capabilities{HasGPU: false},
		threads:      4,
		contextSize:  8192,
	}

	args := c.buildArgs("x")
	require.Equal(t, "4096", argValue(args, "-n"))
	require.Equal(t, "0.3", argValue(args, "--temp"))
}
