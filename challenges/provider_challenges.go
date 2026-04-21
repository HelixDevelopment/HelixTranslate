package challenges

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"digital.vasic.challenges/pkg/challenge"
)

// NewLLMProviderChallenge verifies LLM provider factory and interface.
type LLMProviderChallenge struct {
	challenge.BaseChallenge
}

// NewLLMProviderChallenge creates the LLM provider challenge.
func NewLLMProviderChallenge() *LLMProviderChallenge {
	bc := challenge.NewBaseChallenge(
		"llm-providers",
		"LLM Providers",
		"Verifies all LLM provider implementations compile and load",
		"unit",
		nil,
	)
	return &LLMProviderChallenge{BaseChallenge: bc}
}

// Execute runs the LLM provider challenge.
func (c *LLMProviderChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	providers := []string{"openai", "anthropic", "deepseek", "zhipu", "qwen", "gemini", "ollama", "llamacpp"}
	assertions := []challenge.AssertionResult{}
	allPass := true

	for _, p := range providers {
		cmd := exec.CommandContext(ctx, "go", "build", fmt.Sprintf("./pkg/translator/llm/%s/...", p))
		out, err := cmd.CombinedOutput()
		pass := err == nil
		if !pass {
			allPass = false
		}
		assertions = append(assertions, challenge.AssertionResult{
			Type:    "compilation",
			Target:  fmt.Sprintf("provider_%s", p),
			Passed:  pass,
			Message: string(out),
		})
	}

	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestNewLLMTranslator", "./pkg/translator/llm/...")
	out, err := cmd.CombinedOutput()
	pass := err == nil
	if !pass {
		allPass = false
	}
	assertions = append(assertions, challenge.AssertionResult{
		Type:    "test_pass",
		Target:  "provider_factory",
		Passed:  pass,
		Message: string(out),
	})

	status := challenge.StatusPassed
	if !allPass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewOpenAIProviderChallenge verifies OpenAI provider specifically.
type OpenAIProviderChallenge struct {
	challenge.BaseChallenge
}

// NewOpenAIProviderChallenge creates the OpenAI provider challenge.
func NewOpenAIProviderChallenge() *OpenAIProviderChallenge {
	bc := challenge.NewBaseChallenge(
		"openai-provider",
		"OpenAI Provider",
		"Verifies OpenAI provider configuration validation and mock translation",
		"unit",
		nil,
	)
	return &OpenAIProviderChallenge{BaseChallenge: bc}
}

// Execute runs the OpenAI provider challenge.
func (c *OpenAIProviderChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestOpenAI", "./pkg/translator/llm/openai/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "openai_provider",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}

// NewAnthropicProviderChallenge verifies Anthropic provider.
type AnthropicProviderChallenge struct {
	challenge.BaseChallenge
}

// NewAnthropicProviderChallenge creates the Anthropic provider challenge.
func NewAnthropicProviderChallenge() *AnthropicProviderChallenge {
	bc := challenge.NewBaseChallenge(
		"anthropic-provider",
		"Anthropic Provider",
		"Verifies Anthropic provider configuration validation and mock translation",
		"unit",
		nil,
	)
	return &AnthropicProviderChallenge{BaseChallenge: bc}
}

// Execute runs the Anthropic provider challenge.
func (c *AnthropicProviderChallenge) Execute(ctx context.Context) (*challenge.Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-run", "TestAnthropic", "./pkg/translator/llm/anthropic/...")
	out, err := cmd.CombinedOutput()

	pass := err == nil
	assertions := []challenge.AssertionResult{
		{
			Type:    "test_pass",
			Target:  "anthropic_provider",
			Passed:  pass,
			Message: string(out),
		},
	}

	status := challenge.StatusPassed
	if !pass {
		status = challenge.StatusFailed
	}

	return c.CreateResult(status, start, assertions, nil, nil, ""), nil
}
