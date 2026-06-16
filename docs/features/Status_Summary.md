# HelixTranslate — Feature Status Summary

**Revision:** 8
**Last modified:** 2026-06-16T00:00:00Z
**Authority:** Two-audience companion to `docs/features/Status.md` (§11.4.56). Derived from the same inventory. Per §11.4.44 (revision header), §11.4.60 (always-sync), §11.4.6 (no-guessing).

---

# Page 1 — For the team (non-developer)

**What is this?** HelixTranslate is an ebook translation system. This page is a plain-language snapshot of how much of it we have catalogued and how much we have proven works on camera.

## What works

- We have a complete catalogue of **every feature** in the product and its 8 owned add-on modules — **493 individual capabilities** catalogued (Rev 8 re-counted the live tables directly: the earlier "478" was a hand-tracked tally that had drifted; the real live count was 486, and this round added the 7 missing agent-bridge ensemble rows to reach 493; the headline "417" is the same list with closely-related sub-options counted once).
- The majority are fully built and reachable: the command-line translators, the web dashboards, the REST and gRPC servers, ~32 built-in AI providers, ebook format readers (FB2, EPUB, DOCX, HTML, TXT, PDF) and writers (EPUB, FB2, TXT, HTML, Markdown, **DOCX and now PDF**), caching, security, and all 8 add-on modules.
- **23 features are proven on real recordings** — each checked frame-by-frame to confirm it genuinely shows the feature working (real translated text in the right language, a live connection count changing, etc.), not just "a screen". These include: the primary command-line translator doing real DeepSeek translations and converting between ebook formats (EPUB→TXT, HTML→EPUB, FB2→EPUB, FB2→FB2), Serbian Cyrillic/Latin handling, the **multi-pass polishing engine (`-multipass`)**, the REST API server translating for real, the gRPC server translating English→Spanish, the markdown round-trip tool, the preparation/analysis runner (fixed), the simple CLI translator, the live monitoring dashboard, **the agent bridge command-line tool** (picks the best verifier-scored model and translates through it — recording shows it choosing `novita/Sao10K/L3-8B-Stheno-v3.2` and translating "Good morning, friend." → "Buenos días, amigo."), and **the agent bridge MCP-stdio server** (a real JSON-RPC MCP session — recording shows `tools/list`, `bridge_best_model`, and the `bridge_invoke` tool returning a live model translation "Le pont relie deux rives.").
- **DOCX output** is built and proven by the produced file itself (a real `.docx` = "Microsoft Word 2007+" with real translated text). **PDF output is newly built this round** and proven by passing tests (it renders the translated book to a real, valid PDF that keeps Serbian Cyrillic readable). Screen recordings of both are still owed, so they count as file/test-proven rather than video-proven for now.
- **New this round:** the **agent bridge MCP-stdio server** is now **video-proven** (`helixtranslate-bridge-mcp-stdio-20260615.mp4` — a real JSON-RPC MCP session over stdio: initialize→tools/list→`bridge_best_model`→`bridge_invoke`, with the `bridge_invoke` tool returning a live verified-model translation "Le pont relie deux rives."), joining the **agent bridge command-line tool** which was video-proven the previous round (`helixtranslate-bridge-bestmodel-translate-20260615.mp4`); a **correctness fix in the model-verification gate** (a model that fails the basic "do you respond?" check is now properly rejected); and the **web dashboard's translation page** which used to return "page not found" and now genuinely translates.

## What is pending or limited

- **Video proof is still the main gap:** **23 of 493** features have a recorded, watch-it-yourself demonstration (≈ 4.7%). Everything else is built and present in the code, but a recording is still owed. Nothing is claimed proven just because the code exists.
- **PDF *output* is now built** (this round). The product can *read* DOCX and PDF, and can now *write* DOCX **and PDF**. Output is now EPUB, FB2, TXT, HTML, Markdown, DOCX and PDF. (PDF writing needs the `weasyprint` tool, which is installed here.)
- **The web dashboard's video proof is honestly skipped, not faked:** recording the dashboard translating end-to-end needs the HelixQA web/video test backend, which was off-limits to this round. The page is proven working by its tests + a real run, but its on-camera proof is owed.
- **The on-this-computer AI engines and the remote-worker translators were removed (this round):** the product no longer runs a local AI model (llama.cpp / Ollama) and no longer ships the SSH-based remote-worker tools (`translate-ssh`, `ssh-translation`, `ebook-translator`, the documented `translator` tool). Every translation now uses an online verified AI provider chosen automatically via the LLMsVerifier bridge. These removed tools are marked **Obsolete** (operator-approved 2026-06-15). The team-facing distributed/API capability is kept.
- **The "verified models" API returns 404** because the model-verification service is switched off in the current configuration (a config choice, not a broken feature).
- **4 features are "stubs"** — they look like real endpoints but return placeholder data: ebook-translate-via-API, the two preparation/analysis API endpoints, and translate-with-verification.
- A handful of small API endpoints return fixed/sample data (e.g. a stats endpoint that always shows zeros). These are flagged in the detailed document.

## Video-confirmation coverage

**23 / 493 ≈ 4.7%**, plus **2 real-artifact/test-confirmed** (DOCX + PDF output, video PENDING). This is the headline number to keep improving: each new genuine recording raises it.

## Team actions

- **No operator decision owed for the former SSH/local tools:** the remote-worker + local-engine tools were removed (operator-approved 2026-06-15), so they no longer wait on a remote host — they are Obsolete, not blocked.
- Priority ask: produce real recordings for the next-most-important runtime features (web dashboard translation, the other AI providers) to keep lifting video-confirmation coverage.

---

# Page 2 — For software engineers

## Source & method

Derived 1:1 from `docs/features/.feature_inventory_raw.md` (Rev 1). Every detailed inventory row → one Status row. Statuses set per §11.4.6 from what the inventory actually found in source; no test files were attributed per-feature (recorded `Not-inventoried`, an honest unknown, never guessed).

## Counts

| Dimension | Value |
|---|---|
| Headline total (dedup per-category tally) | 417 |
| Enumerated detailed rows (this doc + Status.md) | 493 (Rev 8 — re-derived mechanically from the live 60 feature tables via `awk` over 9-column rows; the earlier "478" hand-tally had drifted from the true live count of 486; Rev 8 added the 7 missing `pkg/bridge` ensemble-seam rows [`BestTranslator`, `BestTranslatorFunc`, `EnsembleFactory`, `ProviderDiverseTranslators`, `ProviderDiverseClients`, `ProviderDiverseModels`, `BestClient`] → 493) |
| Implemented | 442 (Rev 8: 435 Rev 7 + 7 new pkg/bridge ensemble-seam rows; was 471 Rev 6, −36 flipped to Obsolete Rev 7) |
| **Obsolete (→ Fixed.md)** | **39** (Rev 7 — bridge phase-2 R-2..R-4 removals: `cmd/translate-ssh`/`ssh-translation`/`translator`/`ebook-translator`, `pkg/sshworker`, `pkg/modelsbridge`, Ollama + llama.cpp providers; Reason=`feature-removed`) |
| Stub | 4 |
| Not implemented (gap rows) | 0 (PDF-write flipped gap→Implemented Rev 4, commit fb265e7) |
| Partial | 2 (`POST /api/batch`; `vision_engine` OpenCV) — was 3 Rev 6; `cmd/translator` → Obsolete Rev 7 |
| Operator-blocked | 0 (was 2 Rev 6; `translate-ssh` + `ebook-translator` → Obsolete Rev 7 — capability removed, not host-blocked) |
| Video-confirmed | 23 (+ 2 real-artifact/test-confirmed: DOCX + PDF output) |
| Video PENDING | runtime/user-visible, no recording yet |
| Video N/A | flags / internal types / infra middleware |

### Per-category (inventory headline)

| Category | Count |
|---|---|
| CLI | 130 |
| API | 70 |
| Web | 23 |
| gRPC | 7 |
| Library | 81 |
| Submodule | 95 |
| Infra | 11 |
| **TOTAL** | **417** |

## Stub list (Implementation=Stub — returns mock, no real work)

1. `POST /api/v1/translate/ebook` — `pkg/api/handler.go:1954` (emits start event, returns mock; does NOT translate).
2. `POST /api/v1/preparation/analyze` — `pkg/api/handler.go:1832` (file stat/count + events; stub analysis, no LLM).
3. `GET /api/v1/preparation/result/:session_id` — `pkg/api/handler.go:1927` (returns mock result).
4. `POST /api/v1/translate-with-verification` — `pkg/api/verifier_handlers.go` (selects/verifies a model, returns metadata + preview; does NOT translate).

## Partial list

1. `POST /api/batch` (standalone `pkg/api/server.go`) — returns `queued` batch_id only, no translation executed.
2. `vision_engine` OpenCV — stub by default; real CV only behind a build tag.

> `cmd/translator` was Partial in Rev 6 (local STUB + remote OPERATOR-BLOCKED) — flipped to **Obsolete** Rev 7 (the whole binary was removed in bridge phase-2 R-4).

## Not-implemented gap rows (§11.4.6)

None — both former gap rows are now Implemented.

> Resolved Rev 3: `pkg/ebook` DOCX-write — Implemented (pure-Go OOXML writer `pkg/ebook/docx_writer.go`, commit 87cd2be; real DeepSeek run → `garden_es.docx` = `Microsoft Word 2007+`).
> Resolved Rev 4: `pkg/ebook` PDF-write — Implemented (`pkg/ebook/pdf_writer.go`, commit fb265e7; Book→HTML5→weasyprint→Cyrillic-faithful PDF, wired into unified-translator `.pdf` output, main.go:1160; `pkg/ebook` PDF tests green). Output formats are now EPUB / FB2 / TXT / HTML / MD / DOCX / PDF.

## New Rev 4 features (built + tested this round, recordings owed/skipped)

1. **cmd/model-bridge + pkg/bridge** — LLMsVerifier→component+agent bridge (no local models): selects the best verifier-scored model (top-1 + fallback) and exposes it via a CLI and an MCP stdio server (`.mcp.json`). Commits ab1bed3 + a5860b2. PASS (real run: best-model `novita/Sao10K/L3-8B-Stheno-v3.2` score 0.919, `Invoke`→"Buenos días, amigo."; MCP nonce-echo `HELIX-PROOF-9b21x`; `go test -race ./pkg/bridge` green; §1.1 routing-guard `TestBridge_Invoke_RoutesToBestModel`). **CLI video CONFIRMED** (`helixtranslate-bridge-bestmodel-translate-20260615.mp4`, ffprobe 8.0s/80fr, §11.4.107 content-verified); **MCP-stdio surface video CONFIRMED** (`helixtranslate-bridge-mcp-stdio-20260615.mp4` — real JSON-RPC MCP session: initialize→tools/list→`bridge_best_model`→`bridge_invoke`, tool `bridge_invoke` real output "Le pont relie deux rives." isError:None; ffprobe 11.88s/10fr, ≥6 frames §11.4.107 content-verified).
2. **internal/verifier affirmative-response hard-gate** — commit 97a8afd: a model with `affirmative_response=0` is now a hard disqualifier. PASS (§11.4.115 polarity guard `pipeline_affirmative_gate_test.go`: GREEN-default — the standing suite asserts the defect ABSENT; RED reproduction is OPT-IN only under `RED_MODE=1`, green this session). Video N/A (internal gate, no user-visible surface).
3. **pkg/api/dashboard.go web-dashboard backend** — commit f6ba5cc: dashboard page + translations endpoints, previously 404, now genuinely translates. PASS unit/integration (`dashboard_test.go` §11.4.115 RED→GREEN, `go test -race ./pkg/api` green; live run → "Dobro jutro, prijatelju."). Video **SKIP** (§11.4.3/§11.4.52 — autonomous browser+video capture needs the HelixQA web/video backend, off-limits this round; migration item owed; NOT a faked confirmation).

## Operator-blocked list (§11.4.45)

None — the Rev 6 operator-blocked binaries (`cmd/translate-ssh`, `cmd/ebook-translator`) were **removed** in bridge phase-2 R-4 and are now Obsolete (see the Obsolete list), not host-blocked.

## Obsolete list (§11.4.90 — Rev 7, operator-approved 2026-06-15 D1/D2 + R-1 + R-4)

1. `cmd/translate-ssh`, `cmd/ssh-translation`, `cmd/translator`, `cmd/ebook-translator` — SSH-local / remote-worker translators (removed; FACT: `git ls-files` empty for these cmd dirs).
2. `pkg/sshworker`, `pkg/modelsbridge` — SSH worker library + models-bridge adapter (removed).
3. Ollama provider, llama.cpp provider + multi-worker coordinator (`pkg/translator/llm/{ollama.go,llamacpp.go,llamacpp_provider.go}` removed).

> Reason=`feature-removed`; Superseding-item=`pkg/bridge` (LLMsVerifier→agent bridge); the KEPT distributed-API path is NOT Obsolete. Triple-check: source files gone + `CM-NO-LOCAL-RUNTIME` gate PASS.

## Static / hardcoded-return endpoints (Implemented, but not live)

- `GET /api/stats` (`pkg/api/server.go`) — hardcoded zeros.
- `GET /api/v1/status/:session_id` (`pkg/api/handler.go`) — hardcoded `completed`.
- `GET /api/v1/metrics` (`cmd/api-server/main.go`) — static/zero.
- `GET /api/v1/status/:session_id` (`cmd/monitor-server/main.go`) — static `monitoring_active`.
- `GET /api/v1/providers` (gRPC `GetProviders`) — static ProviderRegistry (API providers only; the `ssh` local provider was removed in bridge phase-2 R-4).
- Static language/provider lists in several `GET /api/.../languages` routes.

## Top gaps (engineering)

1. **Video-confirmation 23/493** (+ 2 real-artifact/test-confirmed: DOCX + PDF output). 23 rows carry real, ffprobe- and content-verified recordings (Status.md cites each `.mp4`; the `-multipass` video is `helixtranslate-cli-multipass-verify-20260615.mp4`, ffprobe 888x630/93fr/9.3s; the bridge CLI video is `helixtranslate-bridge-bestmodel-translate-20260615.mp4`, ffprobe 790×560/80fr/8.0s; the bridge MCP-stdio video is `helixtranslate-bridge-mcp-stdio-20260615.mp4`, ffprobe 790×560/10fr/11.88s; the 3 video-surfaced bug fixes — prep-translator dead, server TLS startup, REST hardcoded target-lang — are in commit `a5e8866`). The remaining Rev-4 additions carry `PENDING` (PDF), `SKIP` (web dashboard — needs HelixQA web/video backend) or `N/A` (internal verifier gate). Per §11.4.2/§11.4.107 nothing may be marked video-confirmed without a real, content-verified file.
2. **PDF output now Implemented** (Rev 4, commit fb265e7) — `pkg/ebook` now has FB2/EPUB/DOCX/PDF writers; zero open gap rows remain. Output is EPUB/FB2/TXT/HTML/MD/DOCX/PDF.
3. **`-multipass` wired (d53e085); `-verify` ≠ `-multipass`** — unified-translator `-verify` runs the CLI's own per-step check; the new `-multipass` flag invokes the `pkg/verification` multi-pass polisher engine (now wired + video-confirmed). cmd/cli write-safety FIXED (87cd2be, atomic write/no-clobber, guard cmd/cli/no_partial_output_test.go) + title path-leak FIXED (d53e085).
4. **SSH-local + local-runtime features removed (Rev 7, 39 rows Obsolete)** — `translate-ssh`/`ssh-translation`/`translator`/`ebook-translator` binaries, `pkg/sshworker`, `pkg/modelsbridge`, and the Ollama + llama.cpp providers are gone (operator-approved); the default path now sources the LLMsVerifier bridge.
5. **`GET /api/v1/verified-models` → 404** — LLMsVerifier disabled in config (route not mounted); feature-disabled-by-config, not a defect.
6. **Per-feature test attribution missing** (`Not-inventoried` everywhere). Repo has extensive `_test.go` + `test/` suites; mapping suites → features is future work.
7. **4 stubs + 1 unimplemented + 3 partials** must either be implemented or reclassified per §11.4.90 before they can be claimed as working capabilities.
8. **UNCONFIRMED LLM determinations** carried from the inventory: thin OpenAI-compatible providers' base-URL/model specifics; several `llms_verifier` packages (scoring/helixqa/cliagents/crush/opencode/scheduler/partners/bigdata/multimodal/performance); `challenges/userflow` evaluators; `doc_processor/archdoc`.

## Video-coverage ratio

`23 / 493` enumerated rows = **4.7%** (≈ 5.5% against the 417 headline). Plus 2 real-artifact/test-confirmed (DOCX + PDF output, video PENDING). This is the primary metric to drive upward; each anti-bluff §11.4.107 recording increments the numerator.

## Cross-references

- Detail: `docs/features/Status.md`
- Source: `docs/features/.feature_inventory_raw.md`
- Sync context: `.docs_chain/contexts/features.yaml` (§11.4.106)
