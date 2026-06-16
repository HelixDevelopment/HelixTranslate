# Translate-arg DATA-LOSS bug class — FIX + RED→GREEN evidence (§11.4.115/§11.4.135/§11.4.146)

**Revision:** 1
**Last modified:** 2026-06-16T00:00:00Z
**Fix of:** the 6 confirmed-broken `.Translate(ctx,text,prompt)` call sites in
`docs/qa/translate_arg_audit_20260616-153353/AUDIT.md` +
`docs/qa/translate_arg_audit_completion_20260616-154307/COMPLETION.md` +
`docs/qa/bug_markdown_empty_payload_rootcause_20260616-152640/FINDING.md`.
**Method:** superpowers:systematic-debugging Phase 4 + §11.4.6 FACT-only + §11.4.69 + §11.4.115.

---

## 1. Fix design (two choke-points cover all 6 sites)

### Seam A — `pkg/translator/llm/openai.go` `OpenAIClient.Translate` (defense-in-depth)
Covers the 4 EMPTY-PAYLOAD sites (#1 `cmd/markdown-translator/main.go:244`,
#2/#3/#4 `pkg/preparation/coordinator.go:290/361/424`) that reach a RAW
OpenAI-compatible client with an empty 2nd arg.
- when `prompt` is empty/whitespace → fall back to the content-bearing 1st arg `text`;
- when BOTH are empty/whitespace → return an explicit error (§11.4.69), never send an
  empty user message that the model answers with boilerplate stored as a translation.
- Embedded by DeepSeek/Groq/etc., so the guard covers every OpenAI-compatible provider.

### Seam B — `pkg/bridge/bridge.go` `clientTranslator.Translate` (compose choke-point)
Covers the 3 WRONG-CONTENT sites (#5 `pkg/verification/polisher.go:563`,
#8/#9 `pkg/coordination/multi_llm.go:445/529`) that reach the verbatim-delegating
`clientTranslator` via `Translate(ctx, REAL_CONTENT, LABEL)`.
- the content-bearing 1st arg `text` is always placed in the body the raw client
  actually sends (its 2nd arg), mirroring the `bridge.go:369` `Translate(ctx,"",full)`
  convention; a non-empty `contextStr` is appended as a labelled `Context:` hint, never
  substituted for the content;
- empty `contextStr` → the content is sent verbatim (preserves the preparation-ensemble
  `(prompt, "")` shape #2/#3/#4 reaching `clientTranslator` — no double-wrapping);
- empty/whitespace content → explicit error (§11.4.69).

### Seam C — `cmd/markdown-translator/main.go:244` (belt-and-suspenders caller)
Build a real translation instruction embedding the block text and pass it as the 2nd
arg (mirrors `pkg/markdown/simple_workflow.go`), so the cmd no longer relies solely on
the Seam-A fallback.

### §11.4.6 correction to the dispatch's "fix the 4 empty-payload callers to pass
content in arg-2 too": the preparation callers #2/#3/#4 pass a COMPLETE analysis prompt
in arg-1 with an empty arg-2. Rewriting them to `("", prompt)` would (a) make
`clientTranslator` refuse empty content and (b) make `*LLMTranslator` return early on
empty `text` — BREAKING both paths. They are FIXED by the two choke-points and correctly
LEFT as `(prompt, "")`. Likewise #5 polisher is FIXED by Seam B (clientTranslator now
composes the verification `prompt` + `location`), so polisher.go:563 is unchanged —
editing it to `("", …)` would break the `*LLMTranslator` default path.

### Which of the 6 sites is fixed by what
| # | site | class | fixed by |
|---|------|-------|----------|
| 1 | markdown-translator/main.go:244 | empty-payload | Seam A fallback + Seam C real prompt |
| 2 | preparation/coordinator.go:290 | empty-payload | Seam A fallback (raw) / Seam B verbatim-prompt (ensemble) |
| 3 | preparation/coordinator.go:361 | empty-payload | same as #2 |
| 4 | preparation/coordinator.go:424 | empty-payload | same as #2 |
| 5 | verification/polisher.go:563 | wrong-content | Seam B compose (ensemble path) |
| 8 | coordination/multi_llm.go:445 | wrong-content | Seam B compose |
| 9 | coordination/multi_llm.go:529 | wrong-content | Seam B compose |

(#11 server.go:196 DEAD-CODE, #22 notes.go:166 SAFE — per COMPLETION.md, untouched.)

---

## 2. RED→GREEN evidence (captured this session, httptest, no API key)

### Seam A — `TestOpenAITranslate_EmptyPayloadGuard` (+ `TestOpenAITranslate_RefuseAllEmpty`)
```
GREEN (fixed, default):      ok   pkg/translator/llm
RED_MODE=1 on PRE-FIX openai.go (stashed fix): ok  -> defect reproduced (empty msg sent, boilerplate returned)
RED_MODE=1 on FIXED openai.go:                 FAIL -> content "The old man walked along the shore at dawn." now reaches model
```

### Seam B — `TestClientTranslator_ComposesContentNotLabel` (+ EmptyContext + RefusesEmpty)
```
GREEN (fixed, default):      ok   pkg/bridge
RED_MODE=1 on PRE-FIX bridge.go (stashed fix): ok  -> defect reproduced (label "Section content" sent, real content dropped)
RED_MODE=1 on FIXED bridge.go:                 FAIL -> real content REAL_SECTION_CONTENT_THAT_MUST_BE_TRANSLATED now reaches model
```

The RED tests are registered as permanent §11.4.135 standing guards (default `RED_MODE=0`).

---

## 3. Validation
```
go build ./...                                   -> exit 0
go test ./pkg/translator/llm/ ./pkg/bridge/ \
        ./pkg/coordination/ ./pkg/verification/ \
        ./pkg/preparation/ ./cmd/markdown-translator/ \
        ./pkg/markdown/ -count=1                  -> all ok
```
No SAFE site broke: `TestOpenAITranslate_RequestShapeAndAuth` (asserts the non-empty
2nd-arg user message reaches the wire verbatim) still passes — the Seam-A fallback is
inert for non-empty 2nd args, and the Seam-B compose only fires inside `clientTranslator`.

## 4. §11.4.6 honesty boundary
- PROVEN FACT (wire repro, both polarities, both seams): empty-payload fixed at the raw
  client; wrong-content fixed at the clientTranslator choke-point; no SAFE site regressed.
- The end-to-end #8/#9 production seam is the real `bridge.clientTranslator` (the exact
  type `instance.Translator` holds in the factory ensemble path per COMPLETION.md §1) —
  the Seam-B test drives that real type, not a stub.
