# HelixTranslate — Feature Status Summary

**Revision:** 1
**Last modified:** 2026-06-15T13:44:25Z
**Authority:** Two-audience companion to `docs/features/Status.md` (§11.4.56). Derived from the same inventory. Per §11.4.44 (revision header), §11.4.60 (always-sync), §11.4.6 (no-guessing).

---

# Page 1 — For the team (non-developer)

**What is this?** HelixTranslate is an ebook translation system. This page is a plain-language snapshot of how much of it we have catalogued and how much we have proven works on camera.

## What works

- We have a complete catalogue of **every feature** in the product and its 8 owned add-on modules — **470 individual capabilities** (the headline "417" is the same list with closely-related sub-options counted once).
- The vast majority — **464 of 470** — are fully built and reachable in the product: the command-line translators, the web dashboards, the REST and gRPC servers, ~32 built-in AI providers, ebook format readers/writers (FB2, EPUB, DOCX, HTML, TXT, PDF), caching, security, and all 8 add-on modules.
- **One feature is proven on a real recording:** the primary command-line translator (`unified-translator`) doing a real English → Spanish translation with the DeepSeek AI provider. The fix it depended on was verified in that recording.

## What is pending or limited

- **Video proof is the big gap.** Only **1 of 470** features has a recorded, watch-it-yourself demonstration so far (about 0.2%). Everything else is built and present in the code, but a recording proving it end-to-end is still owed. This is shown honestly — we are not claiming a feature is proven just because the code exists.
- **4 features are "stubs"** — they look like real endpoints but currently return placeholder data instead of doing the real work: ebook-translate-via-API, the two preparation/analysis API endpoints, and translate-with-verification.
- **2 features are partial:** one batch API endpoint just queues without translating, and the image-analysis (OpenCV) module ships a placeholder unless a special build option is turned on.
- A handful of small API endpoints return fixed/sample data (e.g. a stats endpoint that always shows zeros). These are flagged in the detailed document.

## Video-confirmation coverage

**1 / 470 ≈ 0.2%.** This is the headline number to improve: each future recording raises it.

## Team actions

- No operator decision is currently blocked.
- Priority ask: produce real recordings for the next-most-important runtime features (web dashboard translation, gRPC translation, the other AI providers) to lift the video-confirmation coverage.

---

# Page 2 — For software engineers

## Source & method

Derived 1:1 from `docs/features/.feature_inventory_raw.md` (Rev 1). Every detailed inventory row → one Status row. Statuses set per §11.4.6 from what the inventory actually found in source; no test files were attributed per-feature (recorded `Not-inventoried`, an honest unknown, never guessed).

## Counts

| Dimension | Value |
|---|---|
| Headline total (dedup per-category tally) | 417 |
| Enumerated detailed rows (this doc + Status.md) | 470 |
| Implemented | 464 |
| Stub | 4 |
| Partial | 2 |
| Video-confirmed | 1 |
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

## Static / hardcoded-return endpoints (Implemented, but not live)

- `GET /api/stats` (`pkg/api/server.go`) — hardcoded zeros.
- `GET /api/v1/status/:session_id` (`pkg/api/handler.go`) — hardcoded `completed`.
- `GET /api/v1/metrics` (`cmd/api-server/main.go`) — static/zero.
- `GET /api/v1/status/:session_id` (`cmd/monitor-server/main.go`) — static `monitoring_active`.
- `GET /api/v1/providers` (gRPC `GetProviders`) — static ProviderRegistry (openai, anthropic, ssh).
- Static language/provider lists in several `GET /api/.../languages` routes.

## Top gaps (engineering)

1. **Video-confirmation 1/470.** The single recorded proof is the `unified-translator` DeepSeek EN→Spanish run: `/Volumes/T7/Downloads/Recordings/helixtranslate-cli-deepseek-translation-FIXED_20260615_163824.mp4`. All other runtime features carry `PENDING`. Per §11.4.2/§11.4.107 nothing else may be marked video-confirmed without a real file.
2. **Per-feature test attribution missing** (`Not-inventoried` everywhere). Repo has extensive `_test.go` + `test/` suites; mapping suites → features is future work.
3. **4 stubs + 2 partials** must either be implemented or reclassified per §11.4.90 before they can be claimed as working capabilities.
4. **UNCONFIRMED LLM determinations** carried from the inventory: thin OpenAI-compatible providers' base-URL/model specifics; several `llms_verifier` packages (scoring/helixqa/cliagents/crush/opencode/scheduler/partners/bigdata/multimodal/performance); `challenges/userflow` evaluators; `doc_processor/archdoc`.

## Video-coverage ratio

`1 / 470` enumerated rows = **0.21%** (≈ 0.24% against the 417 headline). This is the primary metric to drive upward; each anti-bluff §11.4.107 recording increments the numerator.

## Cross-references

- Detail: `docs/features/Status.md`
- Source: `docs/features/.feature_inventory_raw.md`
- Sync context: `.docs_chain/contexts/features.yaml` (§11.4.106)
