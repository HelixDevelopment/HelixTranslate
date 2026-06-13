package modelsbridge

import (
	"context"
	"errors"
	"testing"

	models "digital.vasic.models"
)

// captureClient is a fake llm.LLMClient (satisfies the 2-method interface) that
// records exactly what (text, prompt) the bridge mapped a models.LLMRequest into,
// and returns a canned output/err. Unit-tier fake (§11.4.27).
type captureClient struct {
	gotText, gotPrompt string
	out                string
	err                error
}

func (c *captureClient) Translate(_ context.Context, text, prompt string) (string, error) {
	c.gotText, c.gotPrompt = text, prompt
	return c.out, c.err
}
func (c *captureClient) GetProviderName() string { return "capture-fake" }

func TestRequestToTextPrompt_MessagesPath(t *testing.T) {
	req := &models.LLMRequest{
		Model: "deepseek-chat",
		Messages: []models.Message{
			{Role: "system", Content: "Translate EN->SR (Cyrillic)."},
			{Role: "system", Content: "Preserve names."},
			{Role: "user", Content: "The crow was thirsty."},
		},
	}
	text, prompt := RequestToTextPrompt(req)
	if text != "The crow was thirsty." {
		t.Fatalf("text = %q, want the last user message", text)
	}
	if prompt != "Translate EN->SR (Cyrillic).\nPreserve names." {
		t.Fatalf("prompt = %q, want the system messages joined by newline", prompt)
	}
}

func TestRequestToTextPrompt_PromptPath(t *testing.T) {
	req := &models.LLMRequest{
		Prompt:  "Здраво",
		Options: map[string]interface{}{"system_prompt": "Be terse."},
	}
	text, prompt := RequestToTextPrompt(req)
	if text != "Здраво" {
		t.Fatalf("text = %q, want req.Prompt", text)
	}
	if prompt != "Be terse." {
		t.Fatalf("prompt = %q, want Options[system_prompt]", prompt)
	}
	// nil request is mapped to empties (no panic)
	if tx, pr := RequestToTextPrompt(nil); tx != "" || pr != "" {
		t.Fatalf("nil request mapped to (%q,%q), want empties", tx, pr)
	}
}

// TestBridge_Complete_ContractRoundTrip is the seam contract test (DoD): a rich
// models.LLMRequest goes IN, the underlying string-based client receives the
// correctly-mapped (text, prompt), and a models.LLMResponse comes back with the
// client output + request model + computed usage + provider metadata.
func TestBridge_Complete_ContractRoundTrip(t *testing.T) {
	fake := &captureClient{out: "Гавран је био жедан."}
	b := New(fake)

	if b.Provider() != "capture-fake" {
		t.Fatalf("Provider() = %q", b.Provider())
	}

	req := &models.LLMRequest{
		Model: "deepseek-chat",
		Messages: []models.Message{
			{Role: "system", Content: "Translate to Serbian Cyrillic."},
			{Role: "user", Content: "The crow was thirsty."},
		},
	}
	resp, err := b.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	// Wire-format on the CLIENT side: the bridge must have mapped the rich
	// request into the system's (text, prompt) contract.
	if fake.gotText != "The crow was thirsty." {
		t.Fatalf("client received text=%q, want the user message", fake.gotText)
	}
	if fake.gotPrompt != "Translate to Serbian Cyrillic." {
		t.Fatalf("client received prompt=%q, want the system message", fake.gotPrompt)
	}

	// Response side: client output + echoed model + computed usage + metadata.
	if resp.Text != "Гавран је био жедан." {
		t.Fatalf("resp.Text = %q, want the client output", resp.Text)
	}
	if resp.Model != "deepseek-chat" {
		t.Fatalf("resp.Model = %q, want the request model", resp.Model)
	}
	if resp.FinishReason != "stop" {
		t.Fatalf("resp.FinishReason = %q", resp.FinishReason)
	}
	if resp.Usage.PromptTokens == 0 || resp.Usage.CompletionTokens == 0 ||
		resp.Usage.TotalTokens != resp.Usage.PromptTokens+resp.Usage.CompletionTokens {
		t.Fatalf("usage not computed/consistent: %+v", resp.Usage)
	}
	if resp.Metadata["provider"] != "capture-fake" {
		t.Fatalf("resp.Metadata[provider] = %v, want capture-fake", resp.Metadata["provider"])
	}
}

func TestBridge_Complete_PropagatesClientError(t *testing.T) {
	sentinel := errors.New("provider exploded")
	b := New(&captureClient{err: sentinel})
	resp, err := b.Complete(context.Background(), &models.LLMRequest{Prompt: "x"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Complete err = %v, want the client's error propagated", err)
	}
	if resp != nil {
		t.Fatalf("resp = %+v, want nil on error", resp)
	}
}

func TestNew_NilClientPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New(nil) must panic — a Bridge with no client is a programming error")
		}
	}()
	_ = New(nil)
}
