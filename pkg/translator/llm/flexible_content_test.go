package llm

import (
	"encoding/json"
	"testing"
)

// TestFlexibleContent_StringInput verifies FlexibleContent handles plain string content.
func TestFlexibleContent_StringInput(t *testing.T) {
	var fc FlexibleContent
	err := json.Unmarshal([]byte(`"hello world"`), &fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(fc) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(fc))
	}
}

// TestFlexibleContent_ArrayOfObjects verifies FlexibleContent handles reasoning-model
// structured content (array of objects with "text" fields). ATM-071.
func TestFlexibleContent_ArrayOfObjects(t *testing.T) {
	input := `[{"type":"text","text":"Hello "},{"type":"text","text":"world"}]`
	var fc FlexibleContent
	err := json.Unmarshal([]byte(input), &fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(fc) != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", string(fc))
	}
}

// TestFlexibleContent_ArrayOfStrings verifies FlexibleContent handles array of strings.
func TestFlexibleContent_ArrayOfStrings(t *testing.T) {
	input := `["Hello ","world","!"]`
	var fc FlexibleContent
	err := json.Unmarshal([]byte(input), &fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(fc) != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", string(fc))
	}
}

// TestFlexibleContent_EmptyArray verifies FlexibleContent handles empty array.
func TestFlexibleContent_EmptyArray(t *testing.T) {
	var fc FlexibleContent
	err := json.Unmarshal([]byte(`[]`), &fc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(fc) != "" {
		t.Errorf("expected empty string, got %q", string(fc))
	}
}

// TestFlexibleContent_InOpenAIResponse verifies FlexibleContent works in a full
// OpenAI response unmarshaling context.
func TestFlexibleContent_InOpenAIResponse(t *testing.T) {
	// Structured content response (reasoning model format)
	responseJSON := `{
		"id": "chatcmpl-test",
		"object": "chat.completion",
		"model": "magistral-medium-latest",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": [{"type":"text","text":"Translated text here"}]
			},
			"finish_reason": "stop"
		}]
	}`

	var resp OpenAIResponse
	err := json.Unmarshal([]byte(responseJSON), &resp)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("no choices in response")
	}

	content := string(resp.Choices[0].Message.Content)
	if content != "Translated text here" {
		t.Errorf("expected 'Translated text here', got %q", content)
	}
}
