// Package modelsbridge wires the shared digital.vasic.models LLM abstraction
// (the rich LLMRequest/LLMResponse/Usage type set) into the translator system.
//
// The system's provider layer speaks a deliberately tiny interface —
// llm.LLMClient.Translate(ctx, text, prompt) (string, error). digital.vasic.models
// instead models a full request/response contract (model, messages, temperature,
// usage, finish reason, metadata). Until now those two never met:
// digital.vasic.models was a standalone, zero-importer module (W1 finding).
//
// Bridge is that missing seam — it lets callers express an interaction in the
// shared models vocabulary and run it through ANY concrete llm.LLMClient, and it
// maps the string result back into a models.LLMResponse. This makes
// digital.vasic.models a first-class CONSUMED dependency of the system rather
// than an orphan, without forcing a system-wide retype of the provider layer.
package modelsbridge

import (
	"context"
	"strings"

	models "digital.vasic.models"

	"digital.vasic.translator/pkg/translator/llm"
)

// Bridge adapts the rich digital.vasic.models request/response contract onto the
// system's string-based llm.LLMClient.
type Bridge struct {
	client llm.LLMClient
}

// New wraps a concrete llm.LLMClient as a models-aware Bridge.
// It panics on a nil client — a Bridge with no backing client is a programming
// error, never a runtime condition to paper over.
func New(client llm.LLMClient) *Bridge {
	if client == nil {
		panic("modelsbridge: New requires a non-nil llm.LLMClient")
	}
	return &Bridge{client: client}
}

// Provider returns the underlying provider's name (passthrough to the client).
func (b *Bridge) Provider() string { return b.client.GetProviderName() }

// RequestToTextPrompt deterministically maps a models.LLMRequest onto the
// (text, prompt) pair llm.LLMClient.Translate expects:
//
//   - Messages present  -> text = the LAST user message; prompt = all system
//     messages joined by "\n" (the instruction/context the model is given).
//   - No messages       -> text = Prompt; prompt = Options["system_prompt"] if a
//     string is provided there, else "".
//
// A nil request maps to ("", "").
func RequestToTextPrompt(req *models.LLMRequest) (text, prompt string) {
	if req == nil {
		return "", ""
	}
	if len(req.Messages) > 0 {
		sys := make([]string, 0, len(req.Messages))
		for _, m := range req.Messages {
			switch strings.ToLower(strings.TrimSpace(m.Role)) {
			case "system":
				sys = append(sys, m.Content)
			case "user":
				text = m.Content // last user message wins
			}
		}
		return text, strings.Join(sys, "\n")
	}
	if req.Options != nil {
		if sp, ok := req.Options["system_prompt"].(string); ok {
			prompt = sp
		}
	}
	return req.Prompt, prompt
}

// Complete runs a rich models.LLMRequest through the underlying llm.LLMClient and
// returns a models.LLMResponse. Errors from the client propagate unchanged.
func (b *Bridge) Complete(ctx context.Context, req *models.LLMRequest) (*models.LLMResponse, error) {
	text, prompt := RequestToTextPrompt(req)

	out, err := b.client.Translate(ctx, text, prompt)
	if err != nil {
		return nil, err
	}

	model := ""
	if req != nil {
		model = req.Model
	}
	promptTokens := estimateTokens(text) + estimateTokens(prompt)
	completionTokens := estimateTokens(out)
	return &models.LLMResponse{
		Text:         out,
		Model:        model,
		FinishReason: "stop",
		Usage: models.Usage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
		Metadata: map[string]interface{}{"provider": b.client.GetProviderName()},
	}, nil
}

// estimateTokens is a deterministic whitespace-word token estimate. It avoids a
// tokenizer dependency; it is an accounting approximation for the Usage fields,
// not a billing-grade count (documented honestly per §11.4.6).
func estimateTokens(s string) int {
	return len(strings.Fields(s))
}
