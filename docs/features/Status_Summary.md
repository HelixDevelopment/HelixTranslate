# HelixTranslate — Feature Status Summary

**Revision:** 13
**Last modified:** 2026-06-16T13:45:00Z
**Authority:** Two-audience companion to `docs/features/Status.md` (§11.4.56). Derived from the same inventory. Per §11.4.44 (revision header), §11.4.60 (always-sync), §11.4.6 (no-guessing).

---

# Page 1 — For the team (non-developer)

**What is this?** HelixTranslate is an ebook translation system. This page is a plain-language snapshot of how much of it we have catalogued and how much we have proven works on camera.

## What works

- We have a complete catalogue of **every feature** in the product and its 8 owned add-on modules — **494 individual capabilities** catalogued (Rev 9 added 1 new unified-translator HTML→Markdown conversion row for the Wave-2 video; Rev 8 re-counted the live tables directly and added the 7 missing agent-bridge ensemble rows to reach 493; the headline "417" is the same list with closely-related sub-options counted once).
- The majority are fully built and reachable: the command-line translators, the web dashboards, the REST and gRPC servers, ~32 built-in AI providers, ebook format readers (FB2, EPUB, DOCX, HTML, TXT, PDF) and writers (EPUB, FB2, TXT, HTML, Markdown, **DOCX and now PDF**), caching, security, and all 8 add-on modules.
- **41 features are proven on real recordings** — each checked frame-by-frame to confirm it genuinely shows the feature working (real translated text in the right language, a live connection count changing, a real CLI database/pipeline output, etc.), not just "a screen". These include: the primary command-line translator doing real DeepSeek translations and converting between ebook formats (EPUB→TXT, HTML→EPUB, FB2→EPUB, FB2→FB2, **and now HTML→Markdown**), Serbian Cyrillic/Latin handling, the **multi-pass polishing engine (`-multipass`)**, the REST API server translating for real, the gRPC server translating English→Spanish, the markdown round-trip tool, the preparation/analysis runner (fixed), the simple CLI translator, the live monitoring dashboard, the **agent bridge command-line tool**, the **agent bridge MCP-stdio server**, the **provider-diverse translator set** (Wave 2: novita, mistral, groq, siliconflow; Wave 3a: Cerebras, SambaNova; Wave 3b: Gemini, Zhipu, Cohere; Wave 3c: Hyperbolic, Fireworks; Wave 3d: nvidia, openrouter), and — **new this round (Wave 3d)** — the **`verify-models` command-line tool** (real model-verification pipeline + ranking), the **`workable-items` command-line tool** (real project-tracker database), and **two more providers** (nvidia + openrouter, distinct real Spanish translations).
- **DOCX output and PDF output are now proven on camera (Wave 2).** A real run of the command-line translator produces a real `.docx` ("Microsoft Word 2007+" with real Spanish text) and a valid PDF with real Spanish — both captured as real-use videos and checked frame-by-frame. **The blocker is fixed:** the default command-line/gRPC translation path that was runtime-broken in Wave 1 (the auto-selected verified model being rejected by a stale model whitelist) now works — which is exactly why these recordings genuinely demonstrate working features. We never faked a "working" recording; in Wave 1 these were honest skips.
- **New this round (Wave 3d):** four feature-type-diverse confirmations — the **`verify-models` CLI** (shows the real LLMsVerifier model-verification pipeline: 14 providers checked live with real reachable/auth/HTTP-status results, 4 models verified and persisted, then a selection ranking with real scores), the **`workable-items` CLI** (the project's real SQLite issue-tracker: `validate` reports "DB matches markdown (81 items)" and `list` prints real ticket rows), and **two more providers** — **nvidia** ("…el sol se alzó… Un anciano pescador **caminaba** a lo largo de la orilla.") and **openrouter** ("…**paseaba** a lo largo de la orilla."), the two outputs distinct (caminaba/paseaba) proving genuine separate per-provider models. `upstage` was tried but honestly skipped (its key was rejected, HTTP 403) and `together` skipped (no key) — never faked.
- **Prior round (Wave 3c):** the **Hyperbolic** library client and the **Fireworks** provider (in the `llm_provider` submodule) are now video-proven — each a real EN→ES translation run through the OpenAI-compatible per-provider path, with the two Spanish outputs distinct from each other ("…el sol salía…caminaba por la orilla…" for Hyperbolic vs "…el sol ascendió…paseaba por la orilla…" for Fireworks), proving genuine, separate per-provider models rather than a canned response. Wave 3c also re-confirmed (not double-counted) the REST `POST /api/v1/translate` and the `/health`+`/api/v1/version`+`/api/v1/providers` endpoints against a freshly-started live TLS API server — real translated JSON and real endpoint JSON captured on camera. (Prior round, Wave 3a: Cerebras + SambaNova; Wave 3b: Gemini + Zhipu + Cohere; Wave 2: DOCX/PDF/HTML→MD + provider-diverse translators; the Wave-1 default-translation-path blocker is resolved.)

## What is pending or limited

- **Video proof is still the main gap:** **30 of 494** features have a recorded, watch-it-yourself demonstration (≈ 6.1%). Everything else is built and present in the code, but a recording is still owed. Nothing is claimed proven just because the code exists.
- **PDF *output* is now built** (this round). The product can *read* DOCX and PDF, and can now *write* DOCX **and PDF**. Output is now EPUB, FB2, TXT, HTML, Markdown, DOCX and PDF. (PDF writing needs the `weasyprint` tool, which is installed here.)
- **The web dashboard's video proof is honestly skipped, not faked:** recording the dashboard translating end-to-end needs the HelixQA web/video test backend, which was off-limits to this round. The page is proven working by its tests + a real run, but its on-camera proof is owed.
- **The on-this-computer AI engines and the remote-worker translators were removed (this round):** the product no longer runs a local AI model (llama.cpp / Ollama) and no longer ships the SSH-based remote-worker tools (`translate-ssh`, `ssh-translation`, `ebook-translator`, the documented `translator` tool). Every translation now uses an online verified AI provider chosen automatically via the LLMsVerifier bridge. These removed tools are marked **Obsolete** (operator-approved 2026-06-15). The team-facing distributed/API capability is kept.
- **The "verified models" API returns 404** because the model-verification service is switched off in the current configuration (a config choice, not a broken feature).
- **4 features are "stubs"** — they look like real endpoints but return placeholder data: ebook-translate-via-API, the two preparation/analysis API endpoints, and translate-with-verification.
- A handful of small API endpoints return fixed/sample data (e.g. a stats endpoint that always shows zeros). These are flagged in the detailed document.

## Video-confirmation coverage

**41 / 494 ≈ 8.3%** (Wave 3d added 4 net-new video confirmations prioritising feature-type diversity: the `verify-models` CLI [real LLMsVerifier discovery/scoring/selection + ranking], the `workable-items` CLI [real §11.4.93 SQLite validate+list], and 2 more providers nvidia + openrouter [distinct real EN→ES translations]; `upstage` HTTP-403 SKIP + `together` no-key SKIP, never faked; Wave 3c added 2: the Hyperbolic library client and the Fireworks `llm_provider` submodule client, each a distinct real EN→ES translation via the OpenAI-compat per-provider path [+ re-confirming the REST translate + endpoints rows against a live TLS server, not double-counted]; Wave 3b added 3: Gemini, Zhipu (GLM) and Cohere library clients; Wave 3a added 2: Cerebras and SambaNova; Wave 2 added 5: DOCX output, PDF output, HTML→Markdown conversion, ProviderDiverseTranslators, EnsembleFactory). This is the headline number to keep improving: each new genuine recording raises it.

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
| Enumerated detailed rows (this doc + Status.md) | 494 (Rev 9 = 493 Rev 8 + 1 new unified-translator HTML→Markdown conversion row added for the Wave-2 html-to-md video; Rev 8 added the 7 missing `pkg/bridge` ensemble-seam rows [`BestTranslator`, `BestTranslatorFunc`, `EnsembleFactory`, `ProviderDiverseTranslators`, `ProviderDiverseClients`, `ProviderDiverseModels`, `BestClient`] → 493) |
| Implemented | 443 (Rev 9: 442 Rev 8 + 1 new unified-translator HTML→Markdown conversion row; Rev 8: 435 Rev 7 + 7 new pkg/bridge ensemble-seam rows; was 471 Rev 6, −36 flipped to Obsolete Rev 7) |
| **Obsolete (→ Fixed.md)** | **39** (Rev 7 — bridge phase-2 R-2..R-4 removals: `cmd/translate-ssh`/`ssh-translation`/`translator`/`ebook-translator`, `pkg/sshworker`, `pkg/modelsbridge`, Ollama + llama.cpp providers; Reason=`feature-removed`) |
| Stub | 4 |
| Not implemented (gap rows) | 0 (PDF-write flipped gap→Implemented Rev 4, commit fb265e7) |
| Partial | 2 (`POST /api/batch`; `vision_engine` OpenCV) — was 3 Rev 6; `cmd/translator` → Obsolete Rev 7 |
| Operator-blocked | 0 (was 2 Rev 6; `translate-ssh` + `ebook-translator` → Obsolete Rev 7 — capability removed, not host-blocked) |
| Video-confirmed | 41 (Wave 3d added 4 net-new prioritising feature-type diversity: `verify-models` CLI [real LLMsVerifier pipeline + ranking] + `workable-items` CLI [real SQLite validate+list] + 2 providers nvidia + openrouter [distinct real EN→ES]; `upstage` HTTP-403 SKIP + `together` no-key SKIP; Wave 3c added 2: Hyperbolic library client + Fireworks `llm_provider` submodule client, distinct real EN→ES translations via OpenAI-compat per-provider path [+ re-confirmed REST translate + endpoints rows against a live TLS server, not double-counted]; Wave 3b added 3: Gemini + Zhipu (GLM) + Cohere library clients; Wave 3a added 2: Cerebras + SambaNova per-provider LLM clients; Wave 2 added 5: DOCX output, PDF output, HTML→Markdown conversion, ProviderDiverseTranslators, EnsembleFactory) |
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

1. **Video-confirmation 41/494.** 41 rows carry real, ffprobe- and content-verified recordings. **Wave 3d (2026-06-16) added 4 net-new** prioritising feature-type diversity (real-use, ffprobe non-zero dur+frames / 790×560 even-dims yuv420p, ≥6 frames extracted at fps=2 + content-verified per §11.4.107): the **`cmd/verify-models` CLI** (`helix_translate-verify-models-wave3d-20260616.mp4`, real LLMsVerifier discovery/scoring/selection — 14 env-keyed providers with real per-provider reachable/auth/HTTP-status/candidate/verified counts incl. `nvidia status=200`, `openrouter status=200`, `upstage auth=false 403`, summary `total verified: 4`, then `model-bridge list` ranking `#1 novita/Sao10K/L3-8B-Stheno-v3.2 score=0.91875 … #4 siliconflow score=0.88125`), the **`cmd/workable-items` CLI** (`helix_translate-workable-items-wave3d-20260616.mp4`, real §11.4.93 SQLite SSoT — `validate` → `OK: DB … matches markdown (81 items)`, `list` → real ATM-NNN rows + descriptions, early validate-OK frame ≠ late list frame proving live/advancing), and **2 more providers** — **nvidia** (`helix_translate-provider-nvidia-translate-wave3d-20260616.mp4`, via `-provider openai -base-url https://integrate.api.nvidia.com/v1 -model meta/llama-3.1-8b-instruct`, "Al amanecer, el sol se alzó… Un anciano pescador **caminaba** a lo largo de la orilla.") + **openrouter** (`helix_translate-provider-openrouter-translate-wave3d-20260616.mp4`, via `-base-url https://openrouter.ai/api/v1 -model meta-llama/llama-3.1-8b-instruct`, "…Un anciano pescador **paseaba** a lo largo de la orilla."), the two provider outputs distinct (caminaba/paseaba) — genuine per-provider models, each start-frame English-only ≠ output-frame Spanish liveness; `upstage` tested but honest SKIP (auth=false HTTP 403), `together` SKIP (no key), never faked. **Wave 3c (2026-06-16) added 2 net-new** (real-use, ffprobe non-zero dur+frames / 790×560 even-dims yuv420p, ≥6 frames extracted at fps=2 + content-verified per §11.4.107): the **Hyperbolic** library client (`helix_translate-provider-hyperbolic-translate-wave3c-20260616.mp4`, via `-provider openai -base-url https://api.hyperbolic.xyz/v1 -model meta-llama/Meta-Llama-3.1-70B-Instruct`, "Al amanecer, el sol salía sobre el tranquilo pueblo. Un anciano pescador caminaba por la orilla…") and the **Fireworks** `llm_provider` submodule client (`helix_translate-provider-fireworks-translate-wave3c-20260616.mp4`, via `-base-url https://api.fireworks.ai/inference/v1 -model accounts/fireworks/models/llama-v3p1-70b-instruct`, "Al amanecer, el sol ascendió sobre el tranquilo pueblo. Un anciano pescador paseaba por la orilla…"), the two outputs distinct (salía/caminaba vs ascendió/paseaba) — genuine per-provider models, start-frame (input only) ≠ output-frame (Spanish) liveness. **Wave 3c also re-confirmed** (not double-counted) the REST `POST /api/v1/translate` (`helix_translate-rest-translate-wave3c-20260616.mp4`, live `cmd/server` TLS port 9443, real JSON `{"translated":"El sol de la mañana se alzó sobre el tranquilo pueblo.","provider":"llm-deepseek",…}`) and the `/health`+`/api/v1/version`+`/api/v1/providers` endpoints (`helix_translate-rest-endpoints-wave3c-20260616.mp4`, real health/version/providers JSON). **Wave 3b (2026-06-16) added 3 net-new** (real-use, ffprobe non-zero dur+frames / 790×560 even-dims yuv420p, ≥6 frames extracted + one content-verified per §11.4.107): the **Gemini** client (`helix_translate-provider-gemini-translate-wave3b-20260616.mp4`, gemini-2.0-flash, "Staro svjetiljko stajalo je samotočno na kamenitoj obali…"), the **Zhipu (GLM)** client (`helix_translate-provider-zhipu-translate-wave3b-20260616.mp4`, glm-4-flash, "Stara svjetiljka stojeća samostalno na kamenitoj obali…") and the **Cohere** client (`helix_translate-provider-cohere-translate-wave3b-20260616.mp4`, command-r-08-2024 via `-base-url https://api.cohere.ai/compatibility/v1`, provider=cohere, "Stari svjetionik stojeo je sam na kamenitoj obali… svijeća njegova vodila brodove…"), all four EN→SR-latin outputs distinct from one another (incl. the re-confirming deepseek clip `helix_translate-provider-deepseek-translate-wave3b-20260616.mp4`, not double-counted) — genuine per-provider models. **Wave 3a (2026-06-16) added 2 net-new** (validator-style ffprobe+frame, §11.4.107): the **Cerebras** client (`helix_translate-provider-cerebras-translate-wave3a-20260616.mp4`, "El sol de la mañana se alzó sobre el tranquilo pueblo.") and the **SambaNova** client (`helix_translate-provider-sambanova-translate-wave3a-20260616.mp4`, "El sol matutino se alzó sobre el tranquilo pueblo."), both via `unified-translator -provider <name>`, the two outputs distinct from each other (genuine per-provider models), start-frame ≠ output-frame liveness. **Wave 2 (2026-06-16) added 5 net-new** (validator-verified ffprobe+frame, §11.4.107): DOCX output (`helix_translate-docx-output-20260616-112943.mp4`), PDF output (`helix_translate-pdf-output-20260616-113057.mp4`), HTML→Markdown conversion (`helix_translate-html-to-md-20260616-114228.mp4`), and `ProviderDiverseTranslators` + `EnsembleFactory` (the 4 per-provider clips `helix_translate-provider-{novita,mistral,groq,siliconflow}-translate-20260616-114*.mp4`); plus an additional re-confirming clip on the already-confirmed gRPC TranslationService row (`helix_translate-grpc-translate-20260616-113415.mp4`, not double-counted). Wave 1 added 2 (bridge ensemble + provider-diverse catalogue). Prior recordings: `-multipass`, bridge CLI, bridge MCP-stdio, the 3 video-surfaced bug fixes in commit `a5e8866`. Per §11.4.2/§11.4.107 nothing may be marked video-confirmed without a real, content-verified file.
   - **✅ Wave-1 DISCOVERED BLOCKER — RESOLVED (Wave 2, §11.4.108/§11.4.138):** the default bridge **translator** path (`unified-translator` + gRPC) that was RUNTIME-BROKEN in Wave 1 (`VerifiedFactory.CreateTranslatorWithFallback`→`NewLLMTranslatorWithConfig` rejecting the verified strongest model against the stale `ValidModels` whitelist, `llm.go:237`) is now FIXED — the Wave-2 DOCX/PDF/gRPC/per-provider recordings are real-use videos showing the verified-bridge translator path producing real target-language output (the exact path that failed in Wave 1). The honest Wave-1 SKIPs are now genuine CONFIRMED rows. See Status.md caveat 8 for the preserved forensic record.
2. **PDF output now Implemented** (Rev 4, commit fb265e7) — `pkg/ebook` now has FB2/EPUB/DOCX/PDF writers; zero open gap rows remain. Output is EPUB/FB2/TXT/HTML/MD/DOCX/PDF.
3. **`-multipass` wired (d53e085); `-verify` ≠ `-multipass`** — unified-translator `-verify` runs the CLI's own per-step check; the new `-multipass` flag invokes the `pkg/verification` multi-pass polisher engine (now wired + video-confirmed). cmd/cli write-safety FIXED (87cd2be, atomic write/no-clobber, guard cmd/cli/no_partial_output_test.go) + title path-leak FIXED (d53e085).
4. **SSH-local + local-runtime features removed (Rev 7, 39 rows Obsolete)** — `translate-ssh`/`ssh-translation`/`translator`/`ebook-translator` binaries, `pkg/sshworker`, `pkg/modelsbridge`, and the Ollama + llama.cpp providers are gone (operator-approved); the default path now sources the LLMsVerifier bridge.
5. **`GET /api/v1/verified-models` → 404** — LLMsVerifier disabled in config (route not mounted); feature-disabled-by-config, not a defect.
6. **Per-feature test attribution missing** (`Not-inventoried` everywhere). Repo has extensive `_test.go` + `test/` suites; mapping suites → features is future work.
7. **4 stubs + 1 unimplemented + 3 partials** must either be implemented or reclassified per §11.4.90 before they can be claimed as working capabilities.
8. **UNCONFIRMED LLM determinations** carried from the inventory: thin OpenAI-compatible providers' base-URL/model specifics; several `llms_verifier` packages (scoring/helixqa/cliagents/crush/opencode/scheduler/partners/bigdata/multimodal/performance); `challenges/userflow` evaluators; `doc_processor/archdoc`.

## Video-coverage ratio

`41 / 494` enumerated rows = **8.3%** (≈ 9.8% against the 417 headline). Wave 3d added 4 net-new video confirmations prioritising feature-type diversity (`verify-models` CLI + `workable-items` CLI + 2 providers nvidia/openrouter; `upstage` HTTP-403 SKIP + `together` no-key SKIP, never faked); Wave 3c added 2 net-new video confirmations (Hyperbolic library client + Fireworks `llm_provider` submodule client, distinct real EN→ES translations via OpenAI-compat per-provider path; + re-confirmed REST translate + endpoints rows against a live TLS server, not double-counted); Wave 3b added 3 (Gemini + Zhipu + Cohere library clients); Wave 3a added 2 (Cerebras + SambaNova); Wave 2 added 5 (DOCX/PDF/HTML→MD/ProviderDiverseTranslators/EnsembleFactory). This is the primary metric to drive upward; each anti-bluff §11.4.107 recording increments the numerator.

## Cross-references

- Detail: `docs/features/Status.md`
- Source: `docs/features/.feature_inventory_raw.md`
- Sync context: `.docs_chain/contexts/features.yaml` (§11.4.106)
