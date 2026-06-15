# HelixTranslate — Feature Status Summary

**Revision:** 3
**Last modified:** 2026-06-15T18:45:00Z
**Authority:** Two-audience companion to `docs/features/Status.md` (§11.4.56). Derived from the same inventory. Per §11.4.44 (revision header), §11.4.60 (always-sync), §11.4.6 (no-guessing).

---

# Page 1 — For the team (non-developer)

**What is this?** HelixTranslate is an ebook translation system. This page is a plain-language snapshot of how much of it we have catalogued and how much we have proven works on camera.

## What works

- We have a complete catalogue of **every feature** in the product and its 8 owned add-on modules — **472 individual capabilities** catalogued (470 from the source inventory + 2 output gap rows; one of those — DOCX output — is now built this round; the headline "417" is the same list with closely-related sub-options counted once).
- The majority are fully built and reachable: the command-line translators, the web dashboards, the REST and gRPC servers, ~32 built-in AI providers, ebook format readers (FB2, EPUB, DOCX, HTML, TXT, PDF) and writers (EPUB, FB2, TXT, HTML, Markdown, **and now DOCX**), caching, security, and all 8 add-on modules.
- **21 features are now proven on real recordings** this round — and each recording was checked frame-by-frame to confirm it genuinely shows the feature working (real translated text in the right language, a live connection count changing, etc.), not just "a screen". These include: the primary command-line translator doing real DeepSeek translations and converting between ebook formats (EPUB→TXT, HTML→EPUB, FB2→EPUB, FB2→FB2), Serbian Cyrillic/Latin handling, the **multi-pass polishing engine (`-multipass`, newly wired)**, the REST API server translating for real, the gRPC server translating English→Spanish, the markdown round-trip tool, the preparation/analysis runner (now fixed), the simple CLI translator, and the live monitoring dashboard.
- **DOCX output is newly built and proven by the produced file itself:** a real DeepSeek translation produced a `.docx` that opens as a genuine "Microsoft Word 2007+" document with the real translated text inside. (A screen recording of it is still owed, so it counts as artifact-proven rather than video-proven for now.)

## What is pending or limited

- **Video proof is still the main gap, but moving:** **21 of 472** features now have a recorded, watch-it-yourself demonstration (≈ 4.5%, up from 20). Everything else is built and present in the code, but a recording is still owed. Nothing is claimed proven just because the code exists.
- **PDF *output* is not built yet.** The product can *read* DOCX and PDF, and can now *write* DOCX; it still cannot *write* PDF — there is no PDF writer. Output is now EPUB, FB2, TXT, HTML, Markdown and DOCX.
- **Two remote-worker tools are blocked on infrastructure:** the SSH-based translators (`translate-ssh`, `ebook-translator`) need a remote helper machine running a local AI engine, which we don't have available. They build and start, then stop at the missing machine.
- **The documented `translator` tool's "translate on this computer" mode is a placeholder** ("local translation not yet implemented"); its remote mode is blocked on the same missing machine.
- **The "verified models" API returns 404** because the model-verification service is switched off in the current configuration (a config choice, not a broken feature).
- **4 features are "stubs"** — they look like real endpoints but return placeholder data: ebook-translate-via-API, the two preparation/analysis API endpoints, and translate-with-verification.
- A handful of small API endpoints return fixed/sample data (e.g. a stats endpoint that always shows zeros). These are flagged in the detailed document.

## Video-confirmation coverage

**21 / 472 ≈ 4.5%** (up from 20), plus **1 real-artifact-confirmed** (DOCX output, video PENDING). This is the headline number to keep improving: each new genuine recording raises it.

## Team actions

- **Operator decision owed for 2 blocked tools:** to confirm `translate-ssh` and `ebook-translator`, provide a remote SSH host running the local AI engine (llama.cpp). Until then they stay marked OPERATOR-BLOCKED.
- Priority ask: produce real recordings for the next-most-important runtime features (web dashboard translation, the other AI providers) to keep lifting video-confirmation coverage.

---

# Page 2 — For software engineers

## Source & method

Derived 1:1 from `docs/features/.feature_inventory_raw.md` (Rev 1). Every detailed inventory row → one Status row. Statuses set per §11.4.6 from what the inventory actually found in source; no test files were attributed per-feature (recorded `Not-inventoried`, an honest unknown, never guessed).

## Counts

| Dimension | Value |
|---|---|
| Headline total (dedup per-category tally) | 417 |
| Enumerated detailed rows (this doc + Status.md) | 472 (470 inventory + 2 gap rows: DOCX-write [Implemented Rev 3], PDF-write) |
| Implemented | 464 (DOCX-write flipped gap→Implemented Rev 3, commit 87cd2be) |
| Stub | 4 |
| Not implemented (gap rows) | 1 (`pkg/ebook` PDF-write) |
| Partial | 3 (`POST /api/batch`; `vision_engine` OpenCV; `cmd/translator` local STUB + remote OPERATOR-BLOCKED) |
| Operator-blocked | 2 (`translate-ssh`, `ebook-translator`) |
| Video-confirmed | 21 (+ 1 real-artifact-confirmed: DOCX output) |
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
3. `cmd/translator` — LOCAL path is a STUB ("local translation not yet implemented"); REMOTE path OPERATOR-BLOCKED (no SSH/llama.cpp worker host).

## Not-implemented gap rows (§11.4.6)

1. `pkg/ebook` PDF-write — no PDF writer; PDF is extract-only; unified-translator rejects `.pdf` output.

> Resolved Rev 3: `pkg/ebook` DOCX-write — now Implemented (pure-Go OOXML writer `pkg/ebook/docx_writer.go`, commit 87cd2be; real DeepSeek run → `garden_es.docx` = `Microsoft Word 2007+`). Output formats are now EPUB / FB2 / TXT / HTML / MD / DOCX.

## Operator-blocked list (§11.4.45)

1. `cmd/translate-ssh` — SSH worker FB2→EPUB; needs remote SSH host + llama.cpp.
2. `cmd/ebook-translator` — FB2 remote-translate workflow; same remote SSH/llama.cpp requirement.

> Evidence (builds + runs to the blocked point): `helixtranslate-cmd-blocked-binaries_20260615-172456.mp4`.

## Static / hardcoded-return endpoints (Implemented, but not live)

- `GET /api/stats` (`pkg/api/server.go`) — hardcoded zeros.
- `GET /api/v1/status/:session_id` (`pkg/api/handler.go`) — hardcoded `completed`.
- `GET /api/v1/metrics` (`cmd/api-server/main.go`) — static/zero.
- `GET /api/v1/status/:session_id` (`cmd/monitor-server/main.go`) — static `monitoring_active`.
- `GET /api/v1/providers` (gRPC `GetProviders`) — static ProviderRegistry (openai, anthropic, ssh).
- Static language/provider lists in several `GET /api/.../languages` routes.

## Top gaps (engineering)

1. **Video-confirmation 21/472** (+ 1 real-artifact-confirmed: DOCX output). 21 rows carry real, ffprobe- and content-verified recordings (Status.md cites each `.mp4`; the new `-multipass` video is `helixtranslate-cli-multipass-verify-20260615.mp4`, ffprobe 888x630/93fr/9.3s; the 3 video-surfaced bug fixes — prep-translator dead, server TLS startup, REST hardcoded target-lang — are in commit `a5e8866`). All other runtime features carry `PENDING`. Per §11.4.2/§11.4.107 nothing else may be marked video-confirmed without a real, content-verified file.
2. **PDF output unimplemented** — `pkg/ebook` now has FB2/EPUB/DOCX writers; PDF-write remains the one open gap row. Output is EPUB/FB2/TXT/HTML/MD/DOCX.
3. **`-multipass` wired (d53e085); `-verify` ≠ `-multipass`** — unified-translator `-verify` runs the CLI's own per-step check; the new `-multipass` flag invokes the `pkg/verification` multi-pass polisher engine (now wired + video-confirmed). cmd/cli write-safety FIXED (87cd2be, atomic write/no-clobber, guard cmd/cli/no_partial_output_test.go) + title path-leak FIXED (d53e085).
4. **2 binaries OPERATOR-BLOCKED + 1 local STUB** — `translate-ssh`/`ebook-translator` need a remote SSH/llama.cpp host; `cmd/translator` local path is a STUB.
5. **`GET /api/v1/verified-models` → 404** — LLMsVerifier disabled in config (route not mounted); feature-disabled-by-config, not a defect.
6. **Per-feature test attribution missing** (`Not-inventoried` everywhere). Repo has extensive `_test.go` + `test/` suites; mapping suites → features is future work.
7. **4 stubs + 1 unimplemented + 3 partials** must either be implemented or reclassified per §11.4.90 before they can be claimed as working capabilities.
8. **UNCONFIRMED LLM determinations** carried from the inventory: thin OpenAI-compatible providers' base-URL/model specifics; several `llms_verifier` packages (scoring/helixqa/cliagents/crush/opencode/scheduler/partners/bigdata/multimodal/performance); `challenges/userflow` evaluators; `doc_processor/archdoc`.

## Video-coverage ratio

`21 / 472` enumerated rows = **4.5%** (≈ 5.0% against the 417 headline), up from `20 / 472`. Plus 1 real-artifact-confirmed (DOCX output, video PENDING). This is the primary metric to drive upward; each anti-bluff §11.4.107 recording increments the numerator.

## Cross-references

- Detail: `docs/features/Status.md`
- Source: `docs/features/.feature_inventory_raw.md`
- Sync context: `.docs_chain/contexts/features.yaml` (§11.4.106)
