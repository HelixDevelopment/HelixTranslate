# §11.4.153 Feature-Ledger Reconciliation — wave-7..9 planning

**Revision:** 1
**Last modified:** 2026-06-16T15:04:12Z
**Run-id:** feature_ledger_recon_20260616T150412Z
**Authority:** §11.4.153 (per-feature video ledger), §11.4.118 (code-vs-ledger reconciliation / discovery-coverage), §11.4.6 (no-guessing — every line below is a captured fact from the actual files).
**Stream:** BACKGROUND operator-authorized, READ-ONLY, NON-COMMITTING. This agent did NOT edit/stage/commit/`git add` (a separate committer agent owns the main checkout, §11.4.84). This file is a file-write only.
**Main checkout HEAD at start:** `0726c20`.
**Provider keys observed (§11.4.6, values never printed):** `DEEPSEEK_API_KEY` SET, `GROQ_API_KEY` SET. No other provider keys present in this shell.

---

## 1. What waves 1–6 ALREADY confirmed (the baseline — do NOT re-record as net-new)

Established from `docs/features/Status.md` (Rev 14) Anti-bluff note + `docs/qa/video_wave*/EVIDENCE.md` + `docs/qa/wave3e-multipass-regression/FINDING.md`. Per the doc's own §11.4.6 re-derivable count, **48 video-confirmed** rows after wave 5 (43 at Rev 14 + 5 wave-5), with wave 6 adding 5 more (HTML→HTML, PDF-input, DOCX-input, EN→ZH, EN→RU) → **~53** confirmed; the Status.md summary cell has not yet absorbed waves 4–6 (committer lag, NOT a code gap).

**CONFIRMED — do not repeat:**

- **Core translate (CLI):** DeepSeek EN→ES translate; `-script` Cyrillic↔Latin; Serbian-Cyrillic; format-detection; help/version; `-verify` flag.
- **Format conversions (CLI):** EPUB→TXT, HTML→EPUB, FB2→EPUB, FB2→FB2, EPUB→EPUB, TXT→TXT, HTML→HTML; HTML→Markdown; markdown EPUB↔MD round-trip.
- **Format input (CLI):** PDF input accepted, DOCX input accepted.
- **Format output (CLI):** DOCX output, PDF output (+ EPUB/FB2/TXT/HTML/MD).
- **Language pairs (CLI):** EN→FR, EN→DE, EN→IT, EN→ES, EN→PT, EN→JA, EN→ZH, EN→RU, RU→SR(cyrillic).
- **Providers (CLI per-provider path):** deepseek, cerebras, sambanova, gemini, zhipu, cohere, hyperbolic, nvidia, openrouter, fireworks, novita, mistral, groq, siliconflow.
- **REST API (live `cmd/server`):** `POST /api/v1/translate`, `GET /health · /version · /providers`, `POST /api/v1/convert/script`, `POST /api/v1/translate/fb2` (→ real Serbian EPUB, hardcoded ru→sr target — honest limitation noted).
- **gRPC:** TranslationService EN→ES (CLI clip) AND **full async file-job round-trip on the LIVE nezha stack** (`docs/qa/nezha_grpc_roundtrip_20260616T143649Z/RESULT.md` — StartTranslation→poll GetTranslationStatus→completed→valid translated EPUB with real Serbian; PASS).
- **Bridge / LLMsVerifier CLI:** `cmd/model-bridge` best-model translate + MCP-stdio surface; `cmd/verify-models` discovery/scoring/selection; `cmd/workable-items` SQLite SSoT validate+list.
- **Other binaries:** markdown-translator, preparation-translator (FIXED), cmd/cli (DeepSeek), monitor-server (live WS hub), `-multipass` (confirmed ONLY with explicit valid `-model deepseek-chat`; see §5 known bug).

---

## 2. Classification of remaining PENDING features (CLOSED set, §11.4.6)

The 186 `PENDING` ledger rows + Web/Library/Submodule rows were classified into the closed set the operator specified. Counts are of distinct ledger rows (flags + internal types that carry `N/A` are excluded from PENDING).

| Classification | Count (approx, distinct PENDING rows) | Meaning |
|---|---|---|
| **[CLI-LOCAL-CONFIRMABLE]** | **~38** | exercisable via `./build/unified-translator` or local `./build/server` with the keys present (deepseek/groq), no live nezha/LLMsVerifier server needed |
| **[LIVE-BRIDGE-NEEDED]** | **~30** | needs a reachable LLMsVerifier server (localhost:8080) and/or the additional provider keys that are EMPTY in the env — blocked by the operator-review item; DEFER |
| **[WEB-DASHBOARD]** | **~37** | needs a browser/HelixQA driving monitor.html / enhanced-monitor.html / dashboard.html — operator-attended |
| **[GRPC]** | **0 net-new** | the 7 RPCs class is already LIVE-confirmed via the nezha async round-trip; remaining per-RPC rows are sub-features of that confirmed flow |
| **[DESKTOP/MOBILE]** | **0** | no desktop/mobile surface exists in this codebase |
| **[NON-FEATURE / N/A]** | 258 (the `N/A` rows) | bare CLI flags, internal library types, infra middleware — no standalone user-visible video applies |

**LIVE-BRIDGE-NEEDED detail (DEFER, §11.4.6):** the `-use-verifier` path requires a live LLMsVerifier at `http://localhost:8080` which is unreachable in this environment (every wave EVIDENCE.md records this honest constraint). Additionally, per `docs/qa/operator_review_20260616T142856Z/PLANS.md`, 9 provider keys are EMPTY in the nezha container (`OPENAI`, `ANTHROPIC`, `QWEN`, `XAI`, `TOGETHER`, `NLP_CLOUD`, `SARVAM`, `LLMSVERIFIER`, `SSH_WORKER_PASSWORD`) — so OpenAI/Anthropic/Qwen/xAI/Together/NLPCloud/Sarvam providers CANNOT be confirmed until the operator populates those keys. These belong to LIVE-BRIDGE-NEEDED, not CLI-LOCAL.

**WEB-DASHBOARD detail:** monitor.html (9 rows), enhanced-monitor.html (8 rows), dashboard.html (7 rows), monitor-server `/ws`,`/monitor`,`/health` (4 rows) + version-monitoring dashboard.html. All require browser/HelixQA per §11.4.143/§11.4.117 — operator-attended; not autonomously CLI-confirmable.

---

## 3. CLI-LOCAL-CONFIRMABLE inventory (the actionable pool for waves 7–9)

These ARE exercisable now with `DEEPSEEK_API_KEY`/`GROQ_API_KEY` and `./build/unified-translator` / `./build/server`. Distinct from waves 1–6.

**A. Native-factory providers via `unified-translator -provider <name>` (key permitting).** `pkg/translator/llm/llm.go` switches 26 native providers directly. Confirmed already: deepseek, cerebras, sambanova, gemini, zhipu, cohere, hyperbolic, novita, mistral, groq, siliconflow (+ nvidia/openrouter/fireworks via submodule path). **Remaining native providers with code BUT no key in this env** → these are CLI-shaped but key-blocked, so effectively LIVE-BRIDGE/key-blocked: openai, anthropic, qwen, xai, togetherai, nlpcloud, sarvam, upstage (403). **Remaining native providers possibly key-available depends on env:** kimi, modal, publicai, nia, vulavula, replicate, cloudflare — confirm `<NAME>_API_KEY` presence before recording; only those with a real key are genuine CLI-LOCAL.

**B. NOT-yet-confirmed CLI-local FEATURES that work on deepseek/groq (no extra key):**

1. **GROQ provider** EN→ES via `-provider groq` (key is SET; only deepseek-family confirmed for many feature rows) — a second-provider liveness confirmation distinct from the deepseek baseline.
2. **More language pairs** through the confirmed deepseek path that the matrix has not yet covered, e.g. **EN→SR(latin)**, **EN→AR**, **EN→KO**, **DE→EN** (reverse direction), **FR→ES** (non-English source→non-English target) — each a distinct user-visible capability the `-source-lang`/`-target-lang` rows enable (Status.md lines 271–272 are `N/A` as bare flags; the CAPABILITY deserves §11.4.153 confirmation per the wave-4 precedent).
3. **TXT→EPUB / TXT→DOCX / TXT→PDF / TXT→FB2** output-format cross-products from a plain TXT source (writers all confirmed present in code; the specific source→target cross-products are net-new rows).
4. **EPUB→DOCX / EPUB→PDF / EPUB→HTML / EPUB→MD** conversions (EPUB parser + each writer confirmed in code; these specific conversions not yet recorded).
5. **`-chunk-size` / `-concurrency` / `-workers`** real effect on a multi-section book (a session-report-backed confirmation that chunking actually parallelises — distinct from bare-flag N/A).
6. **`-temperature` / `-max-tokens`** producing a genuinely different output on the same input (liveness via output-diff, deepseek).
7. **REST `POST /api/v1/translate/string`** and **`POST /api/v1/translate/directory`** (`pkg/api/batch_handlers.go`) against the local `cmd/server` — distinct from the confirmed `/translate` JSON endpoint.
8. **REST `POST /api/translate`** + **`POST /api/upload`** + **`GET /api/languages`** on the standalone `pkg/api/server.go` gin server (a different server impl than `cmd/server`'s `handler.go`).
9. **`cmd/preparation-translator` analysis JSON** consumed by a follow-on translate (the two-stage pipeline end-to-end, deepseek) — preparation row is confirmed in isolation but the consume-the-analysis flow is net-new.
10. **`cmd/markdown-translator` MD→EPUB direction** specifically (round-trip confirmed; the MD-as-source→EPUB direction as a standalone is distinct).

---

## 4. RECOMMENDED waves 7–9 (CLI-LOCAL, no overlap with waves 1–6, deepseek/groq only)

Ordered by value × distinctness; each batch = 3–5 net-new, no cross-batch overlap. All confirmable with the present keys + local binaries.

### Wave 7 — output-format cross-products (highest distinctness, all writers code-confirmed)
1. **TXT→EPUB** (EN .txt → real Spanish EPUB, open artifact + read chapter)
2. **TXT→PDF** (EN .txt → valid Spanish PDF)
3. **EPUB→DOCX** (EN .epub → real Spanish .docx, open OOXML)
4. **EPUB→HTML** (EN .epub → Spanish .html)
5. **TXT→FB2** (EN .txt → Spanish FB2 with namespace preserved)

### Wave 8 — language-matrix breadth + second provider (capability rows the flags enable)
1. **GROQ provider** EN→ES (second live provider, key SET — distinct from all deepseek rows)
2. **EN→SR(latin)** translate (latin Serbian, distinct from confirmed cyrillic)
3. **EN→KO (Korean)** non-Latin output (distinct from JA/ZH)
4. **DE→EN** reverse-direction translate (source≠English)
5. **FR→ES** non-English-source→non-English-target (distinct pair class)

### Wave 9 — REST surface breadth + pipeline depth (local `cmd/server` / `pkg/api/server.go`)
1. **REST `POST /api/v1/translate/string`** real JSON translated string (batch_handlers.go)
2. **REST `POST /api/v1/translate/directory`** real directory batch (batch_handlers.go)
3. **REST `POST /api/translate`** on standalone `pkg/api/server.go` gin server (distinct impl)
4. **REST `POST /api/upload`** file-upload→translate on `pkg/api/server.go`
5. **preparation→translate two-stage pipeline** (preparation-translator analysis JSON consumed by unified-translator, deepseek)

> §11.4.6 caveat on Wave 8.1/native providers: only record a provider whose `<NAME>_API_KEY` is actually present at record time. groq is SET. For any other provider, verify the key first or it is an honest SKIP, never a faked PASS (the wave-3d `upstage`-403 / `together`-no-key precedent).

---

## 5. §11.4.118 code-vs-ledger reconciliation findings (mismatches / honest notes)

Verified against actual code (`pkg/translator/llm/*.go`, `llm_provider/pkg/providers/*`, `pkg/ebook/*`, `pkg/grpc/server.go`, `pkg/api/verifier_handlers.go`, `cmd/`).

- **[MATCH]** All 26 native factory providers in `pkg/translator/llm/llm.go` (lines 264–315) have real client `.go` files. No claimed native provider lacks code.
- **[MATCH]** All 7 output writers present: `epub_writer.go`, `fb2_writer.go`, `docx_writer.go`, `pdf_writer.go`; TXT/HTML/MD via `cmd/unified-translator/main.go` output switch. All 6 input parsers present: `fb2/epub/docx/html/pdf/txt_parser.go`.
- **[MATCH]** All 7 gRPC RPC handlers present in `pkg/grpc/server.go`.
- **[MATCH]** `GET /api/v1/verified-models` handler EXISTS (`pkg/api/verifier_handlers.go:100/109`); the runtime 404 is config-driven (LLMsVerifier disabled), NOT missing code — Status.md states this honestly.
- **[MATCH]** All 4 Obsolete cmd dirs (`translator`, `translate-ssh`, `ebook-translator`, `ssh-translation`) are genuinely ABSENT from disk — Obsolete classification is accurate.
- **[MISMATCH-no-ledger / minor]** `cmd/grpc-translate-probe` EXISTS on disk (used as the nezha gRPC wire-client in `docs/qa/nezha_grpc_roundtrip_*`) but is NOT enumerated as a section in `docs/features/Status.md`. It is a test/probe client, not a shipped user binary, so the omission is minor — recommend adding a one-line ledger row classified `N/A` (internal probe) or `Confirmed` (it IS the tool that produced the gRPC round-trip evidence). Honest §11.4.118 gap, not a code-absence.
- **[NOTE]** The `llm_provider` submodule carries ~44 provider dirs; only the subset reachable through the unified-translator/bridge path is user-exercisable. The extra submodule providers (ai21, chutes, codestral, githubmodels, huggingface, junie, kilo, ollama, perplexity, venice, zai, zen, generic, claude, ai21, …) are NOT enumerated as confirmable HelixTranslate features and are LIVE-BRIDGE-only — correctly out of the CLI-LOCAL pool.
- **[NOTE — real bug, already tracked]** `-multipass` default-model path is BROKEN (`docs/qa/wave3e-multipass-regression/FINDING.md` + wave-5 EVIDENCE): default `gpt-4` is rejected by the `ValidModels` whitelist for deepseek; the polish path also has no guard against the LLM returning a meta-response (writes garbage as the translation). Confirmed working ONLY with explicit `-model deepseek-chat`. The ledger correctly demotes it to PENDING_FORENSICS. Do NOT re-record multipass as a clean confirmation until the source fix lands.
- **[NOTE — real bug, already tracked]** `POST /api/v1/translate/fb2` hardcodes ru→sr (`pkg/api/handler.go:84-85`), ignoring requested target. Feature confirmed working, hardcoded-target is an honest limitation.

---

## 6. Counts summary (for the report)

- **Video-confirmed baseline (waves 1–6):** ~53 distinct rows (Status.md re-derivable 48 post-wave-5 + 5 wave-6; summary cell lags at 43 — committer catch-up owed, not a code gap).
- **PENDING classification:** CLI-LOCAL-CONFIRMABLE ~38 · LIVE-BRIDGE-NEEDED ~30 · WEB-DASHBOARD ~37 · GRPC 0 net-new (class already live-confirmed) · DESKTOP/MOBILE 0 · NON-FEATURE/N/A 258.
- **Code-vs-ledger mismatches:** 1 minor ledger gap (`cmd/grpc-translate-probe` unlisted); 0 claim-without-code. Ledger is materially accurate.
- **Recommended next 3 waves:** 15 CLI-local net-new confirmations (Wave 7 output cross-products, Wave 8 language-matrix + groq, Wave 9 REST breadth + pipeline depth) — all autonomously recordable with deepseek/groq, no live nezha bridge required.
