# ATM-071 Design Proposal: Reasoning-Model Structured Content Support

**Revision:** 1
**Last modified:** 2026-07-08T14:35:00Z
**Scope:** Investigation + design proposal only (no source edits per §11.4.122)

## Summary

OpenAI-compatible providers' `Message.Content` is typed as `string`. Reasoning models (Mistral `magistral-medium-latest`, likely deepseek-reasoner, glm-5) return `content` as a structured JSON array `[{type: "text", text: "..."}]`. `json.Unmarshal` fails with a type mismatch — translation breaks silently.

## Evidence

**Response struct:** `pkg/translator/llm/openai.go:30-33`
```go
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`  // ← typed as string
}
```

**Unmarshal site:** `openai.go:212`
```go
var response OpenAIResponse
if err := json.Unmarshal(body, &response); err != nil {
    return "", fmt.Errorf("failed to unmarshal response: %w", err)
}
```

**Failure mode:** When a reasoning model returns `"content": [{"type": "text", "text": "translated text"}]`, Go's `json.Unmarshal` cannot decode a JSON array into a `string` field → returns an error → translation fails with "failed to unmarshal response".

**Affected providers (OpenAI-compatible):** All providers using the `OpenAIClient` struct: openai, deepseek, qwen, groq, mistral, xai, togetherai, sambanova, cerebras, novita, nlpcloud, upstage, sarvam, hyperbolic, siliconflow, cloudflare, replicate. Each sends requests to an OpenAI-compatible `/v1/chat/completions` endpoint and parses the response into `OpenAIResponse`.

**Not affected:** Anthropic (already handles structured content as `[]ContentBlock` at `anthropic.go:171-184`), Gemini (uses its own response format at `gemini.go:270`), Zhipu (separate response struct).

## Design options

### Option A — Change `Message.Content` to `json.RawMessage`

**Approach:** Change the struct field to `json.RawMessage` (raw bytes), then extract text with a helper:

```go
type Message struct {
    Role    string          `json:"role"`
    Content json.RawMessage `json:"content"`
}

// extractText handles both string and structured content.
func extractText(content json.RawMessage) string {
    if len(content) == 0 {
        return ""
    }
    // Try as string first (common case)
    var s string
    if err := json.Unmarshal(content, &s); err == nil {
        return s
    }
    // Try as structured array [{type: "text", text: "..."}]
    var blocks []struct {
        Type string `json:"type"`
        Text string `json:"text"`
    }
    if err := json.Unmarshal(content, &blocks); err == nil {
        var sb strings.Builder
        for _, b := range blocks {
            if b.Type == "text" {
                sb.WriteString(b.Text)
            }
        }
        return sb.String()
    }
    return ""
}
```

**Pros:** Handles both string and array content. `json.RawMessage` defers parsing until we know the shape. Backward-compatible (string content still works).

**Cons:** Every site that reads `Message.Content` must call `extractText()` instead of using the field directly. Touches all provider files that share the struct.

### Option B — Custom `UnmarshalJSON` on `Message`

**Approach:** Keep `Content string` but add a custom unmarshaler that handles both shapes:

```go
func (m *Message) UnmarshalJSON(data []byte) error {
    type Alias Message
    aux := &struct{ Content json.RawMessage `json:"content"` }{}
    if err := json.Unmarshal(data, aux); err == nil {
        m.Content = extractText(aux.Content)
    }
    // unmarshal rest into Alias
    ...
}
```

**Pros:** Transparent to all callers — `Content` remains a `string`. Zero changes at call sites.

**Cons:** More complex unmarshal logic. Must handle the `Role` field too (needs a full custom unmarshal or an alias trick).

### Option C — `interface{}` with type switch

**Approach:** Change `Content` to `interface{}`, type-switch at the call site.

**Pros:** Simple struct change.
**Cons:** Every call site needs a type switch. Fragile — easy to miss one. Not idiomatic Go.

## Recommendation

**Option A (json.RawMessage)** — it's the most robust and idiomatic. The `extractText` helper is simple, testable, and handles both shapes. Call sites change from `msg.Content` to `extractText(msg.Content)` — mechanical, grep-able, hard to miss.

**Implementation plan:**
1. Change `Message.Content` from `string` to `json.RawMessage` in `openai.go`
2. Add `extractText()` helper in `openai.go`
3. Update `openai.go:220` to use `extractText(response.Choices[0].Message.Content)`
4. Update all other OpenAI-compatible providers that share the struct (grep for `Message` usage)
5. Add test: mock server returns `"content": [{"type":"text","text":"translated"}]` → assert correct extraction
6. Add test: mock server returns `"content": "translated"` → assert still works (backward compat)
7. RED→GREEN per §11.4.43

**Risk:** LOW — the change is backward-compatible (string content still works). The only risk is missing a call site, which grep catches.

## Git history

The `Message` struct with `Content string` has been in place since the initial OpenAI provider implementation. No prior attempt to handle structured content.
