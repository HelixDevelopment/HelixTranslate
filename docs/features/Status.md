# HelixTranslate — Feature Status

**Revision:** 15
**Last modified:** 2026-06-16T18:30:00Z
**Authority:** Derived from `docs/features/.feature_inventory_raw.md` (Revision 1). Per §11.4.45 (captured-evidence Status doc), §11.4.44 (revision header), §11.4.56 (two-audience summary → `Status_Summary.md`), §11.4.60 (always-sync), §11.4.6 (no-guessing — every status reflects what the inventory actually found in source).
**Scope:** Every one of the 417 inventoried features across CLI, API, Web, gRPC, Library, Submodule and Infra. `helix_qa` deliberately not included.

> **Anti-bluff note (§11.4.6 / §11.4.2 / §11.4.107):** `Video-confirmation` is set to a real file **only** where a recording exists AND that recording genuinely demonstrates the feature working (each cited `.mp4` was ffprobe-verified for non-zero duration+frames and content-verified against its claimed behaviour — e.g. real DeepSeek output in the target language, real gRPC translated text, live WebSocket count change). **54 feature rows are video-confirmed** (Wave 6 of 2026-06-16 added **5 net-new** via the default in-process direct DeepSeek CLI path, each ffprobe+frame-verified per §11.4.107 (790×560 yuv420p, advancing frames, English input-frame ≠ translated output-frame): **HTML→HTML round-trip EN→ES** (`helix_translate-html_html_roundtrip-wave6-20260616-175612.mp4`, real Spanish HTML), **PDF input accepted EN→ES** (`helix_translate-pdf_input-wave6-20260616-175612.mp4`, from a real PDF 1.7), **DOCX input accepted EN→ES** (`helix_translate-docx_input-wave6-20260616-175612.mp4`, from a real Word 2007+ file), **EN→ZH (Chinese)** (`helix_translate-en_zh_translate-wave6-20260616-175612.mp4`) and **EN→RU (Russian)** (`helix_translate-en_ru_translate-wave6-20260616-175612.mp4`); Wave 6 also recorded an honest MINOR finding (HTML/EPUB chapter-title duplication on output, PENDING_FORENSICS, cosmetic — core translation correct). Wave 5 of 2026-06-16 added **4 net-new** (the `-multipass` row was NOT counted — see §11.4.138 demotion below): **EN→PT (Portuguese)** (`helix_translate-en-pt-translate-wave5-20260616-173814.mp4`), **EN→JA (Japanese, non-Latin)** (`helix_translate-en-ja-translate-wave5-20260616-173814.mp4`), **REST `POST /api/v1/convert/script`** (`helix_translate-rest-convert-script-wave5-20260616-173814.mp4`, real Cyrillic↔Latin JSON both directions) and **REST `POST /api/v1/translate/fb2`** (`helix_translate-rest-translate-fb2-wave5-20260616-173814.mp4`, FB2→real Serbian EPUB; note: handler hardcodes ru→sr — tracked fix). Wave 4 of 2026-06-16 added **3 net-new** multi-language-pair translations from the same EN source proving genuine per-language LLM calls (distinct outputs): **EN→FR** (`helix_translate-multilang-en-fr-wave4-20260616-173316.mp4`, "Le soleil matinal se leva…"), **EN→DE** (`helix_translate-multilang-en-de-wave4-20260616-173319.mp4`, "Der morgendliche Sonnenaufgang…"), **EN→IT** (`helix_translate-multilang-en-it-wave4-20260616-173419.mp4`, "Il sole mattutino sorse…"); Wave 4 also re-confirmed (NOT double-counted) RU→SR-Cyrillic (`helix_translate-multilang-ru-sr-cyr-wave4-20260616-173445.mp4`). **§11.4.138 DEMOTION (−1):** the `pkg/verification` **Multi-pass polisher** (`-multipass`) row, previously PASS via `helixtranslate-cli-multipass-verify-20260615.mp4`, is DEMOTED to PENDING_FORENSICS — the wave-3e regression (`docs/qa/wave3e-multipass-regression/FINDING.md`) and wave-5 evidence prove the default-model path (`gpt-4`) fails the `ValidModels` whitelist and silently no-ops, and even with an explicit model the polish path can write an LLM meta-response instead of polished prose; a tracked source fix + re-record is owed before any re-confirmation. Net video-confirmed: 43 (Rev 14) − 1 (multipass demote) + 3 (Wave 4) + 4 (Wave 5) + 5 (Wave 6) = **54**. Wave 3f of 2026-06-16 added **2 net-new** — **EPUB→EPUB conversion** (`helix_translate-epub-output-wave3e-20260616.mp4`: EN .epub → real deepseek run → OUTPUT EPUB carries real Spanish "La Antigua Farola la vieja farola se alzaba sola en la rocosa costa…") and **TXT→TXT translation** (`helix_translate-txt-translation-wave3f-20260616.mp4`: EN .txt → real deepseek run → ES .txt "El río fluía tranquilamente a través del valle verde. Los pájaros cantaban bajo la luz de la mañana."), each ffprobe+frame-verified per §11.4.107 (790×560 yuv420p, ≥6 frames, start-frame English-only ≠ output-frame Spanish); Wave 3f also **re-confirmed** (NOT double-counted) the already-confirmed **FB2→FB2** (`helix_translate-fb2-convert-wave3e-20260616.mp4`, namespace preserved + real Spanish), **format-detection** (`helix_translate-format-detect-wave3e-20260616.mp4`, extensionless `.dat` detected-by-content), **`-script` Cyrillic↔Latin** (`helix_translate-script-conversion-wave3e-20260616.mp4`, RUN A latin "Ovo je testni…" ≠ RUN B cyrillic "Систем треба…") and **markdown-translator EPUB↔MD round-trip** (`helix_translate-markdown-roundtrip-wave3f-20260616.mp4`, "# El Faro Viejo / El viejo faro se alzaba solitario en la costa rocosa.") rows with fresh frame-verified clips; running total 41 (Rev 13) + 2 = **43**. Wave 3d of 2026-06-16 added **4 net-new**, prioritising feature-type diversity — the **`cmd/verify-models` CLI** (real LLMsVerifier discovery/scoring/selection: 14 env-keyed providers with real per-provider reachable/auth/HTTP-status/candidate/verified counts, summary `total verified: 4`, then `model-bridge list` selection ranking `#1 novita/Sao10K/L3-8B-Stheno-v3.2 score=0.91875 … #4 siliconflow score=0.88125` — `helix_translate-verify-models-wave3d-20260616.mp4`), the **`cmd/workable-items` CLI** (real §11.4.93 SQLite SSoT: `validate` → `OK: DB … matches markdown (81 items)`, `list` → real ATM-NNN rows + descriptions — `helix_translate-workable-items-wave3d-20260616.mp4`), and **2 more providers** — **`nvidia`** ("…Un anciano pescador **caminaba** a lo largo de la orilla." via `integrate.api.nvidia.com`, `helix_translate-provider-nvidia-translate-wave3d-20260616.mp4`) and **`openrouter`** ("…Un anciano pescador **paseaba** a lo largo de la orilla." via `openrouter.ai/api`, `helix_translate-provider-openrouter-translate-wave3d-20260616.mp4`), the two provider outputs distinct (caminaba/paseaba) proving genuine per-provider models, all four Wave-3d clips ffprobe+frame-verified per §11.4.107 (790×560 yuv420p, ≥6 frames at fps=2, CLI clips show advancing distinct sections / provider clips show start-frame English-only ≠ output-frame Spanish); the Wave-3d providers reach the `llm_provider`/OpenAI-compat per-provider path; `upstage` was tested but is an honest SKIP (auth=false HTTP 403), `together` SKIPped (no key) — never faked. Wave 3c of 2026-06-16 added **2 net-new** — the **Hyperbolic** library client (`pkg/translator/llm`) and the **Fireworks** provider in the `llm_provider` submodule (`llm_provider/pkg/providers/fireworks/fireworks.go`), each a real-use Wave-3c `.mp4` showing a distinct EN→ES translation through the OpenAI-compat per-provider path — `helix_translate-provider-hyperbolic-translate-wave3c-20260616.mp4` ("Al amanecer, el sol salía sobre el tranquilo pueblo. Un anciano pescador caminaba por la orilla…") and `helix_translate-provider-fireworks-translate-wave3c-20260616.mp4` ("Al amanecer, el sol ascendió sobre el tranquilo pueblo. Un anciano pescador paseaba por la orilla…"), the two outputs distinct from each other (salía/caminaba vs ascendió/paseaba) proving genuine per-provider models, ffprobe+frame-verified per §11.4.107 (790×560 yuv420p, ≥6 frames at fps=2, start-frame input-only ≠ output-frame Spanish). Wave 3c also **re-confirmed** (no double-count) the already-confirmed REST `POST /api/v1/translate` and `GET /health · /api/v1/version · /api/v1/providers` rows against a freshly-started live `cmd/server` TLS instance (port 9443) — `helix_translate-rest-translate-wave3c-20260616.mp4` (real JSON `{"translated":"El sol de la mañana se alzó sobre el tranquilo pueblo.","provider":"llm-deepseek",…}`) and `helix_translate-rest-endpoints-wave3c-20260616.mp4` (real health/version/providers JSON). The `GET /api/v1/verified-models` 404 remains an honest PENDING gap (LLMsVerifier disabled in config — not confirmed). Wave 3b of 2026-06-16 added 3 net-new — the **Gemini**, **Zhipu (GLM)** and **Cohere** library clients, each backed by a real-use Wave-3b `.mp4` showing a distinct EN→SR-latin translation through the per-provider path — `helix_translate-provider-gemini-translate-wave3b-20260616.mp4` ("Staro svjetiljko stajalo je samotočno na kamenitoj obali…"), `helix_translate-provider-zhipu-translate-wave3b-20260616.mp4` ("Stara svjetiljka stojeća samostalno na kamenitoj obali…"), `helix_translate-provider-cohere-translate-wave3b-20260616.mp4` (provider=cohere via `-base-url https://api.cohere.ai/compatibility/v1`, "Stari svjetionik stojeo je sam na kamenitoj obali… svijeća njegova vodila brodove…"), all four outputs (incl. the re-confirmed deepseek clip `helix_translate-provider-deepseek-translate-wave3b-20260616.mp4` "Stari svjetionik stajao je sam na kamenitoj obali…") distinct from one another proving genuine per-provider models, ffprobe-verified (non-zero dur+frames, 790×560 even-dims yuv420p) + ≥6 frames extracted with one content-verified per §11.4.107; the deepseek clip is a re-confirmation, no double-count; Wave 3a of 2026-06-16 added 2 net-new — the **Cerebras** and **SambaNova** per-provider LLM clients, each backed by a real-use Wave-3a `.mp4` showing a distinct EN→ES translation through `unified-translator -provider <name>` — `helix_translate-provider-cerebras-translate-wave3a-20260616.mp4` ("El sol de la mañana se alzó sobre el tranquilo pueblo.") and `helix_translate-provider-sambanova-translate-wave3a-20260616.mp4` ("El sol matutino se alzó sobre el tranquilo pueblo."), the two outputs distinct from each other proving genuine per-provider models, ffprobe+frame-verified per §11.4.107 with start-frame ≠ output-frame liveness; Wave 2 of 2026-06-16 added 5 net-new — DOCX output, PDF output, HTML→Markdown conversion, `ProviderDiverseTranslators`, `EnsembleFactory` — each backed by a real-use Wave-2 `.mp4` independently validator-verified ffprobe+frame; Wave 2 also re-confirmed the already-confirmed gRPC TranslationService row with an additional clip, no double-count; Wave 1 added 2 — bridge-ensemble best-model selection + bridge provider-diverse catalogue) across unified-translator (DeepSeek translate, EPUB→TXT / HTML→EPUB / FB2→EPUB / FB2→FB2 conversions, Serbian-Cyrillic, `-script` Cyrillic↔Latin, format-detection, help/version, `-verify`, **`-multipass` multi-pass polishing engine**), the REST API server (POST /api/v1/translate real DeepSeek + target_lang FIXED, TLS startup FIXED, /health · /api/v1/version · /api/v1/providers), the gRPC server (real EN→ES), markdown-translator (EPUB↔MD round-trip), preparation-translator (FIXED real analysis), cmd/cli (real DeepSeek), monitor-server (live WS hub), the DeepSeek library client, and the **LLMsVerifier→agent bridge CLI** (`cmd/model-bridge` — `helixtranslate-bridge-bestmodel-translate-20260615.mp4`: in-process verify→SQLite store→select strongest `novita/Sao10K/L3-8B-Stheno-v3.2` score 0.919 → live `invoke` EN→ES "Good morning, friend." → "Buenos días, amigo."; ffprobe 8.0s/80 frames, ≥6 frames content-verified) **and the LLMsVerifier→agent bridge MCP-stdio surface** (`cmd/model-bridge mcp` — `helixtranslate-bridge-mcp-stdio-20260615.mp4`: a real JSON-RPC MCP session over stdio — initialize→tools/list→`bridge_best_model`→`bridge_invoke` — where tool `bridge_invoke` returned a live verified-model translation "Le pont relie deux rives." (isError:None); ffprobe 11.88s/10 frames, ≥6 frames content-verified). **Now video-confirmed (Wave 2):** **DOCX output** (Implemented; pure-Go OOXML writer `pkg/ebook/docx_writer.go`, commit 87cd2be) — Wave-2 real-use video `helix_translate-docx-output-20260616-112943.mp4` (unified-translator DOCX output, openai/gpt-4, the produced `.docx` carries real Spanish text; validator ffprobe+frame-verified). **PDF output** (Implemented, commit fb265e7; `pkg/ebook/pdf_writer.go` Book→HTML5→`weasyprint`→Cyrillic-faithful PDF, wired into unified-translator `.pdf` output, main.go:1160) — Wave-2 real-use video `helix_translate-pdf-output-20260616-113057.mp4` (valid PDF with real Spanish; validator ffprobe+frame-verified). **Wave-1 bridge-translator-path blocker RESOLVED (caveat 8):** the `ValidModels`-whitelist runtime failure that blocked the DOCX/PDF/gRPC/per-provider videos in Wave 1 is fixed — the verified-bridge translator path now produces real per-provider translations (Wave-2 provider clips: novita/mistral/groq/siliconflow). **Honest gaps recorded, NOT confirmed:** `cmd/translator` + `translate-ssh` + `ebook-translator` are **Obsolete (→ Fixed.md), Rev 7** (`feature-removed` — the local-runtime + SSH-local paths were removed in bridge phase-2 R-2..R-4; the default path now sources the LLMsVerifier `pkg/bridge`; the earlier STUB/OPERATOR-BLOCKED wording is stale and corrected); `GET /api/v1/verified-models` returns 404 because LLMsVerifier is disabled in config. Everything else is `PENDING` (a recording is owed) or `N/A` (no user-visible video applies — e.g. CLI flags, internal library types). Absence of a runtime recording is stated honestly, never papered over.

## Operator-blocked items (read first, §11.4.45)

Two binaries are OPERATOR-BLOCKED — both require a remote SSH worker host running llama.cpp, which is not available in this environment. The binaries build and run as far as the missing host allows (captured in `helixtranslate-cmd-blocked-binaries_20260615-172456.mp4`):

- **`cmd/translate-ssh`** — SSH worker 4-step FB2→EPUB translator. **Unblock condition:** a reachable SSH host with llama.cpp + a model.
- **`cmd/ebook-translator`** — FB2 remote-translate workflow (syncs binary to remote, runs remote translate, downloads outputs). **Unblock condition:** same remote SSH/llama.cpp host.
- **`cmd/translator` (remote path)** — remote SSH/llama.cpp translate path is OPERATOR-BLOCKED (same host requirement); its LOCAL path is a separate STUB ("local translation not yet implemented").

## Coverage summary

| Metric | Value |
|---|---|
| Headline feature total (inventory's deduplicated per-category tally) | **417** |
| Enumerated feature rows in this doc | **496** (Rev 14 = 494 Rev 13 + 2 new unified-translator rows added Wave 3f for the net-new videos: `EPUB→EPUB conversion` + `TXT→TXT translation`; Rev 9 = 493 Rev 8 + 1 new unified-translator `HTML→Markdown conversion` row added that wave for the Wave-2 html-to-md video; §11.4.6 — re-derive with `awk` over `^\|` 9-column rows. The earlier "478" was NOT re-derivable from the live table — it was a hand-tracked running tally that had drifted; Rev 8 added the 7 missing `pkg/bridge` ensemble-seam rows: `BestTranslator`, `BestTranslatorFunc`, `EnsembleFactory`, `ProviderDiverseTranslators`, `ProviderDiverseClients`, `ProviderDiverseModels`, `BestClient`) |
| Implemented | 445 (Rev 14: 443 Rev 13 + 2 new unified-translator rows EPUB→EPUB conversion + TXT→TXT translation, both Implemented + Wired + Wave-3f video-confirmed; Rev 9: 442 Rev 8 + 1 new unified-translator HTML→Markdown conversion row; Rev 8 was 435 Rev 7 + 7 new pkg/bridge ensemble-seam rows; was 471 Rev 6, −36 flipped to Obsolete Rev 7 — local-runtime + SSH-local features removed in bridge phase-2 R-2..R-4) |
| **Obsolete (→ Fixed.md)** | **39** (Rev 7 — bridge phase-2 R-2..R-4 removals: `cmd/translate-ssh` ×9, `cmd/ssh-translation` ×2, `cmd/translator` ×19, `cmd/ebook-translator` ×3, `pkg/sshworker` ×2, `pkg/modelsbridge` ×1, Ollama provider ×1, llama.cpp provider+coordinator ×2; Reason=`feature-removed`; the KEPT distributed-API path is NOT Obsolete) |
| Stub | 4 (`POST /api/v1/translate/ebook`, `POST /api/v1/preparation/analyze`, `GET /api/v1/preparation/result/:id`, `POST /api/v1/translate-with-verification`) |
| Not implemented (gap rows) | 0 (PDF-write flipped gap→Implemented Rev 4, commit fb265e7; output now EPUB/FB2/TXT/HTML/MD/DOCX/PDF) |
| Partial | 2 (`POST /api/batch` on standalone `pkg/api/server.go` — returns `queued` batch_id only, no translation; `vision_engine` OpenCV — stub default, real impl behind build tag) — was 3 Rev 6; `cmd/translator` flipped to Obsolete Rev 7 |
| Operator-blocked | 0 (was 2 Rev 6 — `translate-ssh` + `ebook-translator` flipped to Obsolete Rev 7: the remote SSH/llama.cpp worker capability was removed, not merely host-blocked) |
| Video-confirmed | **43** (real, content-verified recordings — see Anti-bluff note; Rev-14 / Wave 3f of 2026-06-16 added **2 net-new** video confirmations prioritising feature-type diversity — **EPUB→EPUB conversion** (`helix_translate-epub-output-wave3e-20260616.mp4`, EN .epub → real Spanish EPUB) and **TXT→TXT translation** (`helix_translate-txt-translation-wave3f-20260616.mp4`, EN .txt → real Spanish .txt), each ffprobe+frame-verified per §11.4.107; Wave 3f also re-confirmed (NOT double-counted) FB2→FB2 / format-detection / `-script` Cyrillic↔Latin / markdown EPUB↔MD round-trip with fresh frame-verified clips; running total 41 (Rev 13) + 2 = **43**. Rev-13 / Wave 3d of 2026-06-16 added **4 net-new** video confirmations prioritising feature-type diversity — the **`cmd/verify-models` CLI** (`helix_translate-verify-models-wave3d-20260616.mp4`, real LLMsVerifier discovery/scoring/selection + ranking), the **`cmd/workable-items` CLI** (`helix_translate-workable-items-wave3d-20260616.mp4`, real §11.4.93 SQLite validate+list), and **2 more providers** **nvidia** (`helix_translate-provider-nvidia-translate-wave3d-20260616.mp4`, "…caminaba a lo largo de la orilla.") + **openrouter** (`helix_translate-provider-openrouter-translate-wave3d-20260616.mp4`, "…paseaba a lo largo de la orilla."), each ffprobe+frame-verified per §11.4.107; running total 37 (Rev 12) + 4 = **41**. `upstage` HTTP-403 SKIP + `together` no-key SKIP, never faked. Rev-12 / Wave 3c of 2026-06-16 added **2 net-new** video confirmations — the **Hyperbolic** library client (`helix_translate-provider-hyperbolic-translate-wave3c-20260616.mp4`, "…el sol salía…caminaba por la orilla…") and the **Fireworks** `llm_provider` submodule client (`helix_translate-provider-fireworks-translate-wave3c-20260616.mp4`, "…el sol ascendió…paseaba por la orilla…"), each a distinct real EN→ES translation via the OpenAI-compat per-provider path, ffprobe+frame-verified per §11.4.107 (790×560 yuv420p, ≥6 frames at fps=2, start-frame input-only ≠ output-frame Spanish), the two outputs distinct proving genuine per-provider models; Wave 3c also re-confirmed (NOT double-counted) the already-confirmed REST `POST /api/v1/translate` + `/health · /api/v1/version · /api/v1/providers` rows against a live `cmd/server` TLS instance (`helix_translate-rest-translate-wave3c-20260616.mp4` + `helix_translate-rest-endpoints-wave3c-20260616.mp4`); **count reconciliation (§11.4.6):** this cell read **32** through Rev 10/11 — a sync lag behind the Anti-bluff-note prose; it is corrected to the true running total here — 30 (Rev 9) + 2 (Wave 3a Cerebras/SambaNova) + 3 (Wave 3b Gemini/Zhipu/Cohere) + 2 (Wave 3c Hyperbolic/Fireworks) = **37**. Rev-10 / Wave 3a of 2026-06-16 added **2 net-new** video confirmations — the **Cerebras** and **SambaNova** per-provider LLM clients (`helix_translate-provider-cerebras-translate-wave3a-20260616.mp4` + `helix_translate-provider-sambanova-translate-wave3a-20260616.mp4`), each a distinct real EN→ES translation via `unified-translator -provider <name>`, ffprobe+frame-verified per §11.4.107 with start-frame ≠ output-frame liveness, the two outputs distinct proving genuine per-provider models; Rev-11 / Wave 3b of 2026-06-16 added **3 net-new** — the **Gemini**, **Zhipu (GLM)** and **Cohere** library clients (`helix_translate-provider-{gemini,zhipu,cohere}-translate-wave3b-20260616.mp4`), distinct EN→SR-latin per-provider translations, frame-verified per §11.4.107; Rev-9 / Wave 2 of 2026-06-16 added **5 net-new** video confirmations, each backed by a real-use Wave-2 `.mp4` independently validator-verified ffprobe+frame §11.4.107: **DOCX output** (`helix_translate-docx-output-20260616-112943.mp4`), **PDF output** (`helix_translate-pdf-output-20260616-113057.mp4`), **HTML→Markdown conversion** (`helix_translate-html-to-md-20260616-114228.mp4`), **`ProviderDiverseTranslators`** + **`EnsembleFactory`** (the 4 per-provider clips novita/mistral/groq/siliconflow — `helix_translate-provider-{novita,mistral,groq,siliconflow}-translate-20260616-114*.mp4`). Wave 2 also re-confirmed the already-confirmed **gRPC TranslationService** row with an additional clip `helix_translate-grpc-translate-20260616-113415.mp4` (NOT double-counted — that row was already video-confirmed). **DOCX output and PDF output are no longer "real-artifact-only" — they are now video-confirmed.** §11.4.108 note: the Wave-1 bridge-translator-path runtime blocker (caveat 8) is RESOLVED, which is what made the DOCX/PDF/gRPC/per-provider Wave-2 recordings genuinely demonstrate working features (recording them in Wave 1 would have been a §11.4.107 bluff; they were honest SKIPs then). Wave 1 added 2: bridge-ensemble best-model selection + bridge provider-diverse catalogue. ≈ 8.3% of 494 enumerated rows (≈ 9.8% of the 417 headline) |
| Video-confirmation PENDING | runtime/user-visible features without a recording yet |
| Video-confirmation N/A | flags, internal library types, infra middleware (no standalone user-visible video) |

> **§11.4.6 reconciliation:** The inventory's headline is **417** — its own note states this is an *approximate, deduplicated* per-category tally where "some rows are sub-features... counted once, and a few rows span two categories (counted once, in the first/primary category)." The inventory's *detailed* tables actually enumerate **470** feature rows. This document transcribes **all 470 detailed rows 1:1** so **no feature is dropped** (which would itself be a bluff). The 417 headline and the 470 enumerated rows are both reported honestly; the 53-row gap is dedup, not missing coverage.

> "Static-return" endpoints (e.g. `GET /api/stats` hardcoded zeros, `GET /api/v1/status/:id` hardcoded `completed`, `GET /api/v1/metrics` static, monitor-server `GET /api/v1/status` static `monitoring_active`) are marked **Implemented** at the route level but their `Real-use` cell states the static/hardcoded behaviour honestly so no false "working backend" impression is created.

### Per-category counts

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

---

## Legend

- **Implementation:** `Implemented` (real code path present) / `Stub` (returns mock, no real work) / `Partial` (does some but not the named work).
- **Wiring:** `Wired` (reachable from a binary / route / public API) / `Unwired` (present but not reached).
- **Real-use:** what the code actually does at runtime (honest, §11.4.6).
- **Tests:** `Covered` (unit/integration/challenge present in repo) / `None` (no test found / not asserted in inventory).
- **Validation:** strongest evidence available.
- **Video-confirmation:** `Confirmed:<file>` only with a real recording; else `PENDING` / `N/A`.

> Per §11.4.6, where the inventory did not read test files line-by-line, `Tests` is recorded `Not-inventoried` (honest unknown) rather than guessed as Covered or None. The repo is known to carry extensive `_test.go` suites and a `test/` tree, but per-feature test attribution was not part of the raw inventory.

---

# 1. CLI Binaries — cmd/

## cmd/api-server

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api-server | REST translation server | API/Web | Implemented | Wired | Gin HTTP server (:8080) proxying to gRPC backend; serves dashboard + static files | Not-inventoried | Source-confirmed (cmd/api-server/main.go) | PENDING |
| api-server | `-http-port` flag | API | Implemented | Wired | Override HTTP port (else `HTTP_PORT` env) | Not-inventoried | Source-confirmed | N/A |
| api-server | `-http-address` flag | API | Implemented | Wired | Override HTTP bind address (else `HTTP_ADDRESS` env) | Not-inventoried | Source-confirmed | N/A |
| api-server | `-grpc-address` flag | API | Implemented | Wired | Override gRPC backend address (else `GRPC_ADDRESS` env) | Not-inventoried | Source-confirmed | N/A |
| api-server | `-debug` flag | API | Implemented | Wired | Enable debug / Gin debug mode | Not-inventoried | Source-confirmed | N/A |

## cmd/challenge-runner

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| challenge-runner | Challenge orchestrator | CLI | Implemented | Wired | Registers + runs all anti-bluff challenges, prints PASS/FAIL/SKIP, writes results dir | Not-inventoried | Source-confirmed | PENDING |
| challenge-runner | `-parallel` | CLI | Implemented | Wired | Run challenges in parallel | Not-inventoried | Source-confirmed | N/A |
| challenge-runner | `-max-concurrency` | CLI | Implemented | Wired | Max concurrent challenges | Not-inventoried | Source-confirmed | N/A |
| challenge-runner | `-stop-on-failure` | CLI | Implemented | Wired | Stop on first failure | Not-inventoried | Source-confirmed | N/A |
| challenge-runner | `-verbose` | CLI | Implemented | Wired | Verbose output | Not-inventoried | Source-confirmed | N/A |
| challenge-runner | `-timeout` | CLI | Implemented | Wired | Per-challenge timeout | Not-inventoried | Source-confirmed | N/A |
| challenge-runner | `-filter` | CLI | Implemented | Wired | Comma-separated challenge IDs to run | Not-inventoried | Source-confirmed | N/A |
| challenge-runner | `-category` | CLI | Implemented | Wired | Limit to a category | Not-inventoried | Source-confirmed | N/A |
| challenge-runner | `-results-dir` | CLI | Implemented | Wired | Results output directory | Not-inventoried | Source-confirmed | N/A |

## cmd/cli

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| cli | Universal ebook translator | CLI | Implemented | Wired | Parse ebook → translate → atomic-write EPUB/FB2/TXT; multi-llm/distributed support | Not-inventoried | **PASS — real DeepSeek run captured** (real Serbian TXT output). Title path-leak FIXED (d53e085, sanitizeBookTitle). Write-safety FIXED (87cd2be): format validated before any FS write, written to temp sibling + atomic os.Rename only on full success — never a partial/clobbered output. Guard: cmd/cli/no_partial_output_test.go (5 tests, mutation-proven) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-deepseek-translate_20260615-172211.mp4 |
| cli | `-i/-input` | CLI | Implemented | Wired | Input ebook (FB2/EPUB/TXT/HTML/PDF/DOCX) | Not-inventoried | Source-confirmed | N/A |
| cli | `-o/-output` | CLI | Implemented | Wired | Output file | Not-inventoried | Source-confirmed | N/A |
| cli | `-f/-format` | CLI | Implemented | Wired | Output format (epub, fb2, txt) | Not-inventoried | Source-confirmed | N/A |
| cli | `-p/-provider` | CLI | Implemented | Wired | Translation provider | Not-inventoried | Source-confirmed | N/A |
| cli | `-model` | CLI | Implemented | Wired | LLM model name | Not-inventoried | Source-confirmed | N/A |
| cli | `-api-key` | CLI | Implemented | Wired | Provider API key | Not-inventoried | Source-confirmed | N/A |
| cli | `-base-url` | CLI | Implemented | Wired | Provider base URL | Not-inventoried | Source-confirmed | N/A |
| cli | `-script` | CLI | Implemented | Wired | Output script (default/cyrillic/latin) | Not-inventoried | Source-confirmed | N/A |
| cli | `-locale` | CLI | Implemented | Wired | Target language locale code | Not-inventoried | Source-confirmed | N/A |
| cli | `-language` | CLI | Implemented | Wired | Target language name | Not-inventoried | Source-confirmed | N/A |
| cli | `-source` | CLI | Implemented | Wired | Source language (auto-detect if unset) | Not-inventoried | Source-confirmed | N/A |
| cli | `-detect` | CLI | Implemented | Wired | Detect source language and exit (LLM detector if API key) | Not-inventoried | Source-confirmed | N/A |
| cli | `-v/-version` | CLI | Implemented | Wired | Show version | Not-inventoried | Source-confirmed | N/A |
| cli | `-h/-help` | CLI | Implemented | Wired | Show help | Not-inventoried | Source-confirmed | N/A |
| cli | `-create-config` | CLI | Implemented | Wired | Write a config-file template | Not-inventoried | Source-confirmed | N/A |
| cli | `-disable-local-llms` | CLI | Implemented | Wired | Use only distributed/API providers | Not-inventoried | Source-confirmed | N/A |
| cli | `-prefer-distributed` | CLI | Implemented | Wired | Prefer distributed workers | Not-inventoried | Source-confirmed | N/A |
| cli | `-c/-config` | CLI | Implemented | Wired | Config file path | Not-inventoried | Source-confirmed | N/A |
| cli | `-hash-codebase` | CLI | Implemented | Wired | Compute codebase hash and exit | Not-inventoried | Source-confirmed | N/A |

## cmd/deployment

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| deployment | Distributed deployment manager | CLI | Implemented | Wired | Orchestrates Docker-based main+worker deployment over SSH | Not-inventoried | Source-confirmed | PENDING |
| deployment | `-config` | CLI | Implemented | Wired | Config file (default config.distributed.json) | Not-inventoried | Source-confirmed | N/A |
| deployment | `-action` | CLI | Implemented | Wired | deploy/status/stop/cleanup/update/restart/generate-plan | Not-inventoried | Source-confirmed | N/A |
| deployment | `-service` | CLI | Implemented | Wired | Service name for update/restart | Not-inventoried | Source-confirmed | N/A |
| deployment | `-image` | CLI | Implemented | Wired | New image for update | Not-inventoried | Source-confirmed | N/A |
| deployment | `-plan` | CLI | Implemented | Wired | Deployment plan JSON file | Not-inventoried | Source-confirmed | N/A |
| deployment | `-verbose` | CLI | Implemented | Wired | Verbose logging | Not-inventoried | Source-confirmed | N/A |
| deployment | deploy/status/stop/cleanup/update/restart actions | CLI | Implemented | Wired | Execute deployment lifecycle ops | Not-inventoried | Source-confirmed | PENDING |
| deployment | generate-plan action | CLI | Implemented | Wired | Generate deployment-plan.json (random JWT secret per instance) | Not-inventoried | Source-confirmed | PENDING |

## cmd/ebook-translator

> **Obsolete-Details:** Since=bridge phase-2 R-2/R-4 (2026-06-16); Reason=`feature-removed` (local-runtime + SSH-local translation removed; default path now sources the LLMsVerifier bridge — no local llama.cpp/Ollama, no SSH-local worker); Superseding-item=`pkg/bridge` (LLMsVerifier→agent bridge); Operator-approval=2026-06-15 D1/D2 (forbid all local runtimes) + R-1 redirect + the R-4 “keep distributed API, remove SSH-local” decision (2026-06-16); Triple-check evidence=source files removed (FACT: `git ls-files` empty for the removed cmd/* dirs + pkg/sshworker + pkg/modelsbridge + ollama.go/llamacpp*.go) + `CM-NO-LOCAL-RUNTIME` gate PASS. The KEPT distributed-API path is NOT Obsolete.
| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| ebook-translator | FB2 remote-translate workflow | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH worker syncs binary, runs remote FB2→MD→translate→EPUB, downloads + verifies outputs | Not-inventoried | **OPERATOR-BLOCKED** — needs a remote SSH worker host + remote llama.cpp; none available. Binary builds + prints usage in blocked-binaries demo | OPERATOR-BLOCKED; evidence:/Volumes/T7/Downloads/Recordings/helixtranslate-cmd-blocked-binaries_20260615-172456.mp4 |
| ebook-translator | Positional args (no flags) | CLI | **Obsolete (→ Fixed.md)** | Wired | `<source_fb2> <target_lang> <remote_host> <remote_user> <remote_password>` | Not-inventoried | Source-confirmed | N/A |
| ebook-translator | Output verification | CLI | **Obsolete (→ Fixed.md)** | Wired | Verifies MD/EPUB exist, non-degenerate, target-language (Cyrillic) check | Not-inventoried | Source-confirmed | PENDING |

## cmd/grpc-server

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| grpc-server | gRPC TranslationService | gRPC | Implemented | Wired | gRPC server (:50051): Start/Status/List/Cancel/Stream/GetProviders/SubscribeEvents | Not-inventoried | **PASS — real EN→ES translation over gRPC captured** (StartTranslation → real Spanish DeepSeek output). Wave 2 re-confirms via the verified-bridge path (EN→ES "Hola, mundo. Este es un pequeño test.") | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-grpc-deepseek-20260615-171206-v2.mp4 (additional Wave-2: /Volumes/T7/Downloads/Recordings/helix_translate-grpc-translate-20260616-113415.mp4) |
| grpc-server | `-address` | gRPC | Implemented | Wired | Bind address (or `GRPC_ADDRESS` env) | Not-inventoried | Source-confirmed | N/A |
| grpc-server | `-port` | gRPC | Implemented | Wired | Port (or `GRPC_PORT` env) | Not-inventoried | Source-confirmed | N/A |
| grpc-server | `-max-connections` | gRPC | Implemented | Wired | Max concurrent translations | Not-inventoried | Source-confirmed | N/A |
| grpc-server | `-reflection` | gRPC | Implemented | Wired | Enable gRPC reflection (or `ENABLE_REFLECTION` env) | Not-inventoried | Source-confirmed | N/A |
| grpc-server | `-metrics` | gRPC | Implemented | Wired | Enable metrics (or `ENABLE_METRICS` env) | Not-inventoried | Source-confirmed | N/A |
| grpc-server | `-log-level` | gRPC | Implemented | Wired | Log level (or `LOG_LEVEL` env) | Not-inventoried | Source-confirmed | N/A |
| grpc-server | `-version` | gRPC | Implemented | Wired | Show version | Not-inventoried | Source-confirmed | N/A |
| grpc-server | `-help` | gRPC | Implemented | Wired | Show help | Not-inventoried | Source-confirmed | N/A |

## cmd/markdown-translator

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| markdown-translator | EPUB↔Markdown translate pipeline | CLI | Implemented | Wired | EPUB/MD → markdown → translate → EPUB/MD; optional preparation phase | Not-inventoried | **PASS — EPUB↔MD round-trip captured** (Wave 3f re-confirm: EPUB→source MD "# The Old Lighthouse / The old lighthouse stood alone…" → translated MD "# El Faro Viejo / El viejo faro se alzaba solitario en la costa rocosa.", real deepseek round-trip, frame-verified §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-markdown-roundtrip-20260615.mp4 (Wave-3f re-confirm: /Volumes/T7/Downloads/Recordings/helix_translate-markdown-roundtrip-wave3f-20260616.mp4) |
| markdown-translator | `-input` | CLI | Implemented | Wired | Input file (EPUB or Markdown) | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-output` | CLI | Implemented | Wired | Output file (auto if empty) | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-format` | CLI | Implemented | Wired | Output format (epub, md) | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-lang` | CLI | Implemented | Wired | Target language code | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-provider` | CLI | Implemented | Wired | LLM provider (deepseek/openai/anthropic/llamacpp) | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-model` | CLI | Implemented | Wired | LLM model | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-keep-md` | CLI | Implemented | Wired | Keep intermediate markdown | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-prepare` | CLI | Implemented | Wired | Enable multi-LLM preparation/analysis phase | Not-inventoried | Source-confirmed | N/A |
| markdown-translator | `-prep-passes` | CLI | Implemented | Wired | Number of preparation passes | Not-inventoried | Source-confirmed | N/A |

## cmd/monitor-server

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| monitor-server | WebSocket monitoring hub | Web | Implemented | Wired | Gin server (:8090, `MONITOR_SERVER_PORT`): /ws, /monitor dashboard, status + health | Not-inventoried | **PASS — live WS hub captured** (real client connect→`websockets:1`→disconnect→`websockets:0`; /monitor serves real dashboard HTML) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-monitor-server-websocket_20260615-172343.mp4 |
| monitor-server | GET /ws | Web | Implemented | Wired | WebSocket subscription by session_id | Not-inventoried | Source-confirmed | PENDING |
| monitor-server | /monitor + / | Web | Implemented | Wired | Serve monitor.html (redirect from /) | Not-inventoried | Source-confirmed | PENDING |
| monitor-server | GET /api/v1/status/:session_id | Web/API | Implemented | Wired | Returns static `monitoring_active` JSON (not a live session lookup) | Not-inventoried | Source-confirmed; static return | PENDING |
| monitor-server | GET /health | Web | Implemented | Wired | Health + client count | Not-inventoried | Source-confirmed | PENDING |
| monitor-server | `MONITOR_SERVER_PORT` env | Web | Implemented | Wired | Configure bind port (no CLI flag) | Not-inventoried | Source-confirmed | N/A |

## cmd/preparation-translator

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| preparation-translator | Preparation+translation runner | CLI | Implemented | Wired | Parse ebook → multi-LLM preparation analysis → translate → write EPUB + analysis JSON | Not-inventoried | **PASS — FIXED, real analysis captured** (key-wiring bug a5e8866; real preparation phase: tokens/genre/characters/cultural-refs/key-themes) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-preparation-translator-FIXED-20260615.mp4 |
| preparation-translator | `-input` | CLI | Implemented | Wired | Input ebook path | Not-inventoried | Source-confirmed | N/A |
| preparation-translator | `-output` | CLI | Implemented | Wired | Output EPUB path | Not-inventoried | Source-confirmed | N/A |
| preparation-translator | `-analysis` | CLI | Implemented | Wired | Preparation analysis JSON output path | Not-inventoried | Source-confirmed | N/A |
| preparation-translator | `-source` | CLI | Implemented | Wired | Source language | Not-inventoried | Source-confirmed | N/A |
| preparation-translator | `-target` | CLI | Implemented | Wired | Target language | Not-inventoried | Source-confirmed | N/A |
| preparation-translator | `-passes` | CLI | Implemented | Wired | Number of preparation passes | Not-inventoried | Source-confirmed | N/A |
| preparation-translator | `-providers` | CLI | Implemented | Wired | Comma-separated LLM providers | Not-inventoried | Source-confirmed | N/A |

## cmd/server

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| server | Main TLS HTTP/2+HTTP/3 server | API/Web | Implemented | Wired | TLS REST/WebSocket API (config.json); CORS + rate limit + JWT; optional distributed + LLMsVerifier | Not-inventoried | **PASS — FIXED, real TLS startup captured** (TLS cert backfill bug a5e8866; "Server started successfully" + HTTP/2 TLS on :8443) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-api-translate-target-lang-FIXED-20260615.mp4 |
| server | `-config` | API | Implemented | Wired | Config file path (default config.json) | Not-inventoried | Source-confirmed | N/A |
| server | `-version` | API | Implemented | Wired | Show version | Not-inventoried | Source-confirmed | N/A |
| server | `-generate-certs` | CLI | Implemented | Wired | Generate self-signed TLS certs (0600 key) and exit | Not-inventoried | Source-confirmed | N/A |
| server | HTTP/3 (QUIC) + HTTP/2 fallback serving | API/Web | Implemented | Wired | Serve API over TLS; HTTP/3 if `EnableHTTP3` | Not-inventoried | Source-confirmed | PENDING |
| server | API routes (pkg/api Handler) | API/Web | Implemented | Wired | Registers translation REST + WebSocket routes | Not-inventoried | Source-confirmed | PENDING |
| server | LLMsVerifier routes (if enabled) | API | Implemented | Wired | Registers verifier endpoints | Not-inventoried | Source-confirmed | PENDING |

## cmd/ssh-translation

> **Obsolete-Details:** Since=bridge phase-2 R-2/R-4 (2026-06-16); Reason=`feature-removed` (local-runtime + SSH-local translation removed; default path now sources the LLMsVerifier bridge — no local llama.cpp/Ollama, no SSH-local worker); Superseding-item=`pkg/bridge` (LLMsVerifier→agent bridge); Operator-approval=2026-06-15 D1/D2 (forbid all local runtimes) + R-1 redirect + the R-4 “keep distributed API, remove SSH-local” decision (2026-06-16); Triple-check evidence=source files removed (FACT: `git ls-files` empty for the removed cmd/* dirs + pkg/sshworker + pkg/modelsbridge + ollama.go/llamacpp*.go) + `CM-NO-LOCAL-RUNTIME` gate PASS. The KEPT distributed-API path is NOT Obsolete.
| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| ssh-translation | SSH translation system (fixed-config) | CLI/Web | **Obsolete (→ Fixed.md)** | Wired | Hardcoded Russian→Serbian-Cyrillic workflow over SSH worker w/ llama.cpp; WebSocket monitoring | Not-inventoried | Source-confirmed | PENDING |
| ssh-translation | (no CLI flags) | CLI | **Obsolete (→ Fixed.md)** | Wired | All config hardcoded in `SystemConfig` (host, creds, langs, llama providers) | Not-inventoried | Source-confirmed | N/A |

## cmd/translate-ssh

> **Obsolete-Details:** Since=bridge phase-2 R-2/R-4 (2026-06-16); Reason=`feature-removed` (local-runtime + SSH-local translation removed; default path now sources the LLMsVerifier bridge — no local llama.cpp/Ollama, no SSH-local worker); Superseding-item=`pkg/bridge` (LLMsVerifier→agent bridge); Operator-approval=2026-06-15 D1/D2 (forbid all local runtimes) + R-1 redirect + the R-4 “keep distributed API, remove SSH-local” decision (2026-06-16); Triple-check evidence=source files removed (FACT: `git ls-files` empty for the removed cmd/* dirs + pkg/sshworker + pkg/modelsbridge + ollama.go/llamacpp*.go) + `CM-NO-LOCAL-RUNTIME` gate PASS. The KEPT distributed-API path is NOT Obsolete.
| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| translate-ssh | SSH worker 4-step FB2→EPUB translator | CLI | **Obsolete (→ Fixed.md)** | Wired | FB2→MD → translate (remote llama.cpp) → MD→EPUB; standalone worker invoked by coordinator | Not-inventoried | **OPERATOR-BLOCKED** — needs a remote SSH worker host running llama.cpp; none available. Binary builds + runs as far as possible (blocked-binaries demo) | OPERATOR-BLOCKED; evidence:/Volumes/T7/Downloads/Recordings/helixtranslate-cmd-blocked-binaries_20260615-172456.mp4 |
| translate-ssh | `-input` | CLI | **Obsolete (→ Fixed.md)** | Wired | Input ebook (required) | Not-inventoried | Source-confirmed | N/A |
| translate-ssh | `-output` | CLI | **Obsolete (→ Fixed.md)** | Wired | Output EPUB (required) | Not-inventoried | Source-confirmed | N/A |
| translate-ssh | `-host` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH host (required) | Not-inventoried | Source-confirmed | N/A |
| translate-ssh | `-user` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH username (required) | Not-inventoried | Source-confirmed | N/A |
| translate-ssh | `-password` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH password (required) | Not-inventoried | Source-confirmed | N/A |
| translate-ssh | `-port` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH port | Not-inventoried | Source-confirmed | N/A |
| translate-ssh | `-remote-dir` | CLI | **Obsolete (→ Fixed.md)** | Wired | Remote working directory | Not-inventoried | Source-confirmed | N/A |
| translate-ssh | `-report-dir` | CLI | **Obsolete (→ Fixed.md)** | Wired | Report output directory | Not-inventoried | Source-confirmed | N/A |

## cmd/translator

> **Obsolete-Details:** Since=bridge phase-2 R-2/R-4 (2026-06-16); Reason=`feature-removed` (local-runtime + SSH-local translation removed; default path now sources the LLMsVerifier bridge — no local llama.cpp/Ollama, no SSH-local worker); Superseding-item=`pkg/bridge` (LLMsVerifier→agent bridge); Operator-approval=2026-06-15 D1/D2 (forbid all local runtimes) + R-1 redirect + the R-4 “keep distributed API, remove SSH-local” decision (2026-06-16); Triple-check evidence=source files removed (FACT: `git ls-files` empty for the removed cmd/* dirs + pkg/sshworker + pkg/modelsbridge + ollama.go/llamacpp*.go) + `CM-NO-LOCAL-RUNTIME` gate PASS. The KEPT distributed-API path is NOT Obsolete.
| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| translator | Documented FB2 local/remote translator | CLI | **Obsolete (→ Fixed.md)** | Wired | Remote SSH/llama.cpp path produces a documentation report; LOCAL path is a STUB — emits "local translation not yet implemented" (captured). Remote path is OPERATOR-BLOCKED (no SSH worker host) | Not-inventoried | **GAP — local STUB captured; remote OPERATOR-BLOCKED** | PENDING/OPERATOR-BLOCKED; evidence:/Volumes/T7/Downloads/Recordings/helixtranslate-cmd-blocked-binaries_20260615-172456.mp4 |
| translator | `-i/-input` | CLI | **Obsolete (→ Fixed.md)** | Wired | Input ebook | Not-inventoried | Source-confirmed | N/A |
| translator | `-o/-output` | CLI | **Obsolete (→ Fixed.md)** | Wired | Output file | Not-inventoried | Source-confirmed | N/A |
| translator | `-ssh-host` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH host (remote mode) | Not-inventoried | Source-confirmed | N/A |
| translator | `-ssh-user` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH username | Not-inventoried | Source-confirmed | N/A |
| translator | `-ssh-password` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH password | Not-inventoried | Source-confirmed | N/A |
| translator | `-ssh-port` | CLI | **Obsolete (→ Fixed.md)** | Wired | SSH port | Not-inventoried | Source-confirmed | N/A |
| translator | `-remote-dir` | CLI | **Obsolete (→ Fixed.md)** | Wired | Remote working directory | Not-inventoried | Source-confirmed | N/A |
| translator | `-workers` | CLI | **Obsolete (→ Fixed.md)** | Wired | Parallel workers | Not-inventoried | Source-confirmed | N/A |
| translator | `-chunk-size` | CLI | **Obsolete (→ Fixed.md)** | Wired | Text chunk size | Not-inventoried | Source-confirmed | N/A |
| translator | `-concurrency` | CLI | **Obsolete (→ Fixed.md)** | Wired | Max concurrent operations | Not-inventoried | Source-confirmed | N/A |
| translator | `-verify` | CLI | **Obsolete (→ Fixed.md)** | Wired | Verify translated output | Not-inventoried | Source-confirmed | N/A |
| translator | `-verbose` | CLI | **Obsolete (→ Fixed.md)** | Wired | Verbose logging | Not-inventoried | Source-confirmed | N/A |
| translator | `-llama-binary` | CLI | **Obsolete (→ Fixed.md)** | Wired | Path to llama.cpp binary | Not-inventoried | Source-confirmed | N/A |
| translator | `-temperature` | CLI | **Obsolete (→ Fixed.md)** | Wired | LLM temperature | Not-inventoried | Source-confirmed | N/A |
| translator | `-context` | CLI | **Obsolete (→ Fixed.md)** | Wired | LLM context size | Not-inventoried | Source-confirmed | N/A |
| translator | `-version` | CLI | **Obsolete (→ Fixed.md)** | Wired | Show version | Not-inventoried | Source-confirmed | N/A |
| translator | `-help` | CLI | **Obsolete (→ Fixed.md)** | Wired | Show help | Not-inventoried | Source-confirmed | N/A |
| translator | `-hash-codebase` | CLI | **Obsolete (→ Fixed.md)** | Wired | Compute codebase hash | Not-inventoried | Source-confirmed | N/A |

## cmd/unified-translator (PRIMARY CLI)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| unified-translator | Primary multi-provider translator | CLI | Implemented | Wired | Parse → MD → translate (API/llamacpp/ssh) → script-normalize → output (epub/fb2/html/txt/md) + session report | Not-inventoried | **PASS — real run captured** (DeepSeek → real Serbian output) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-deepseek-translate_20260615-172211.mp4 |
| unified-translator | DeepSeek translate (provider=deepseek) | CLI | Implemented | Wired | EN→Serbian DeepSeek run, real LLM output written to TXT | Not-inventoried | **PASS — real Serbian translation captured** | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-deepseek-translation-FIXED_20260615_163824.mp4 |
| unified-translator | EPUB→TXT conversion | CLI | Implemented | Wired | EPUB parse → DeepSeek translate → TXT writer, real Serbian text | Not-inventoried | **PASS — captured** | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-format-conversion-epub-to-txt-20260615.mp4 |
| unified-translator | HTML→EPUB conversion | CLI | Implemented | Wired | HTML parse → DeepSeek translate → EPUB writer | Not-inventoried | **PASS — captured** | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-html-to-epub-deepseek_20260615-171805.mp4 |
| unified-translator | HTML→Markdown conversion | CLI | Implemented | Wired | HTML parse → translate (openai/gpt-4 via verified bridge) → Markdown (`.md`) writer; real Spanish md output | Not-inventoried | **PASS — captured** (Wave 2; real-use video, validator ffprobe+frame-verified — unified-translator HTML→Markdown, openai/gpt-4, real Spanish markdown produced) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-html-to-md-20260616-114228.mp4 |
| unified-translator | FB2→EPUB conversion | CLI | Implemented | Wired | FB2 parse → DeepSeek translate → EPUB writer | Not-inventoried | **PASS — captured** | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-fb2-to-epub-deepseek_20260615-171725.mp4 |
| unified-translator | FB2→FB2 conversion | CLI | Implemented | Wired | FB2 parse → DeepSeek translate → FB2 writer (namespaces preserved) | Not-inventoried | **PASS — captured** (Wave 3f re-confirm: FB2 namespace `http://www.gribuser.ru/xml/fictionbook/2.0` PRESERVED + real Spanish "Este es un párrafo de prueba…", frame-verified §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-fb2-to-fb2-deepseek_20260615-172006.mp4 (Wave-3f re-confirm: /Volumes/T7/Downloads/Recordings/helix_translate-fb2-convert-wave3e-20260616.mp4) |
| unified-translator | Serbian Cyrillic output | CLI | Implemented | Wired | Translate with Serbian-Cyrillic target output | Not-inventoried | **PASS — captured** | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-serbian-cyrillic_20260615-164729.mp4 |
| unified-translator | EPUB→EPUB conversion | CLI | Implemented | Wired | EPUB parse → DeepSeek translate → EPUB writer (round-trip, real translated text in produced .epub) | Not-inventoried | **PASS — captured** (Wave 3f net-new: EN .epub input → real deepseek run → OUTPUT EPUB carries real Spanish "La Antigua Farola la vieja farola se alzaba sola en la rocosa costa…", frame-verified §11.4.107 — start frame English-only ≠ output frame Spanish) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-epub-output-wave3e-20260616.mp4 |
| unified-translator | TXT→TXT translation | CLI | Implemented | Wired | Plain-text parse → DeepSeek translate → TXT writer (real translated text, no format conversion) | Not-inventoried | **PASS — captured** (Wave 3f net-new: EN .txt "The river flowed quietly through the green valley…" → real deepseek run → ES .txt "El río fluía tranquilamente a través del valle verde. Los pájaros cantaban bajo la luz de la mañana.", frame-verified §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-txt-translation-wave3f-20260616.mp4 |
| unified-translator | HTML→HTML round-trip | CLI | Implemented | Wired | HTML parse → DeepSeek translate → HTML writer (round-trip, real translated text in produced .html) | Not-inventoried | **PASS — captured** (Wave 6 net-new: EN .html → real deepseek run → valid Spanish `<html>` "la farola guiaba a los barcos a través de la noche neblinosa…", frame-verified §11.4.107 — input-frame English ≠ output-frame Spanish). NOTE Wave-6 MINOR finding: chapter-title duplication on single-section chapters (cosmetic, PENDING_FORENSICS — core translation correct). | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-html_html_roundtrip-wave6-20260616-175612.mp4 |
| unified-translator | PDF input accepted | CLI | Implemented | Wired | PDF parse → DeepSeek translate (real PDF 1.7 input) | Not-inventoried | **PASS — captured** (Wave 6 net-new: from "PDF document, version 1.7" → real Spanish "El bosque se hizo silencioso a la caída del sol. Los pinos altos susurraban…", frame-verified §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-pdf_input-wave6-20260616-175612.mp4 |
| unified-translator | DOCX input accepted | CLI | Implemented | Wired | DOCX parse → DeepSeek translate (real Word 2007+ input) | Not-inventoried | **PASS — captured** (Wave 6 net-new: from "Microsoft Word 2007+" → real Spanish "Los barcos de pesca regresaron al puerto al atardecer. Gaviotas volaban sobre las redes…", frame-verified §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-docx_input-wave6-20260616-175612.mp4 |
| unified-translator | Format detection | CLI | Implemented | Wired | Input-format auto-detection on real input | Not-inventoried | **PASS — captured** (Wave 3f re-confirm: extensionless `mystery_input.dat` with FB2 XML inside → detected-by-content → real Spanish output, frame-verified §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-format-detection_20260615-164729.mp4 (Wave-3f re-confirm: /Volumes/T7/Downloads/Recordings/helix_translate-format-detect-wave3e-20260616.mp4) |
| unified-translator | help/version output | CLI | Implemented | Wired | `-help`/`-version` usage banner | Not-inventoried | **PASS — captured** | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-help_20260615-164729.mp4 |
| unified-translator | `-i/-input` | CLI | Implemented | Wired | Input ebook | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-o/-output` | CLI | Implemented | Wired | Output file (auto if empty) | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-source-lang` / `-target-lang` (multi-language-pair capability) | CLI | Implemented | Wired | Source/target language selection driving real per-language LLM translation | Not-inventoried | **PASS — video-confirmed** (Waves 4–6 drove the SAME EN source into 7 distinct target languages producing 7 genuinely different, language-correct outputs — proving real per-language LLM calls, not echo/cache: FR "Le soleil matinal se leva…", DE "Der morgendliche Sonnenaufgang…", IT "Il sole mattutino sorse…", PT "O antigo farol erguia-se…", JA "古い灯台は…", ZH "一列列车在黎明时分…", RU "Библиотека стояла тихо под дождем…", each ffprobe+frame-verified §11.4.107) | Confirmed: Wave-4 /Volumes/T7/Downloads/Recordings/helix_translate-multilang-en-{fr,de,it}-wave4-20260616-17331{6,9}.mp4 ; Wave-5 helix_translate-en-{pt,ja}-translate-wave5-20260616-173814.mp4 ; Wave-6 helix_translate-en_{zh,ru}_translate-wave6-20260616-175612.mp4 |
| unified-translator | `-script` (Serbian Cyrillic↔Latin) | CLI | Implemented | Wired | Target script (cyrillic/latin) script normalization on real run | Not-inventoried | **PASS — captured** (Wave 3f re-confirm: RUN A `-script latin` "Ovo je testni paragraf…" ≠ RUN B `-script cyrillic` "Систем треба да преводи…" — same text, genuinely different script, frame-verified §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-serbian-script-cyrillic-latin_20260615-171843.mp4 (Wave-3f re-confirm: /Volumes/T7/Downloads/Recordings/helix_translate-script-conversion-wave3e-20260616.mp4) |
| unified-translator | `-provider` | CLI | Implemented | Wired | Provider (openai/anthropic/zhipu/deepseek/qwen/gemini/ollama/llamacpp/ssh) | Not-inventoried | Source-confirmed; deepseek path captured | N/A |
| unified-translator | `-model` | CLI | Implemented | Wired | Model name | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-api-key` | CLI | Implemented | Wired | Provider API key | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-base-url` | CLI | Implemented | Wired | Provider base URL | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-temperature` | CLI | Implemented | Wired | LLM temperature | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-max-tokens` | CLI | Implemented | Wired | Max tokens | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-timeout` | CLI | Implemented | Wired | Request timeout | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-ssh-host` | CLI | Implemented | Wired | SSH host (provider=ssh) | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-ssh-user` | CLI | Implemented | Wired | SSH username | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-ssh-password` | CLI | Implemented | Wired | SSH password | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-ssh-port` | CLI | Implemented | Wired | SSH port | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-remote-dir` | CLI | Implemented | Wired | Remote directory | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-llama-binary` | CLI | Implemented | Wired | Path to llama.cpp binary | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-llama-model` | CLI | Implemented | Wired | Path to llama.cpp model | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-context-size` | CLI | Implemented | Wired | LLM context size | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-workers` | CLI | Implemented | Wired | Parallel workers | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-chunk-size` | CLI | Implemented | Wired | Text chunk size | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-concurrency` | CLI | Implemented | Wired | Concurrent operations | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-verify` (output-quality check) | CLI | Implemented | Wired | Built-in per-step verification: input/markdown/translation-quality/valid-EPUB verdicts in session report. NOTE: this is the unified-translator's own verify pass, NOT the `pkg/verification` multipass engine (that engine is not wired into this CLI — see gap row below) | Not-inventoried | **PASS — captured** (session report verdicts) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-verify-flag-quality-check_20260615-171920.mp4 |
| unified-translator | `-verbose` | CLI | Implemented | Wired | Verbose logging | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-monitoring` | CLI/Web | Implemented | Wired | Enable web monitoring | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-monitoring-port` | CLI/Web | Implemented | Wired | Monitoring server port | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-use-verifier` | CLI | Implemented | Wired | Use LLMsVerifier for model selection | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-verifier-url` | CLI | Implemented | Wired | LLMsVerifier API URL | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-verifier-api-key` | CLI | Implemented | Wired | LLMsVerifier API key | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-version` | CLI | Implemented | Wired | Show version | Not-inventoried | Source-confirmed | N/A |
| unified-translator | `-help` | CLI | Implemented | Wired | Show help | Not-inventoried | Source-confirmed | N/A |

## cmd/verify-models

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| verify-models | In-process LLMsVerifier runner | CLI | Implemented | Wired | Runs verification pipeline against env-keyed providers; persists verified models to SQLite | Not-inventoried | **PASS — real LLMsVerifier discovery/scoring/selection captured** (`./build/verify-models -db /tmp/vm_record.db -max 1` ran the in-process pipeline against 14 env-keyed providers and printed real per-provider reachability/auth/HTTP-status/candidate/verified counts — e.g. `nvidia reachable=true auth=true status=200 candidates=1`, `openrouter status=200`, `upstage auth=false status=403`, `siliconflow verified=1` — with summary `total verified: 4 · persisted rows: 4`, then `model-bridge list` showed the selection ranking `#1 novita/Sao10K/L3-8B-Stheno-v3.2 score=0.91875 … #4 siliconflow/…Seed-OSS-36B score=0.88125`; Wave-3d, asciinema→agg→ffmpeg, ffprobe 8.0s/790×560 yuv420p + 16 frames at fps=2, ≥6 frames content-verified per §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-verify-models-wave3d-20260616.mp4 |
| verify-models | `-db` | CLI | Implemented | Wired | SQLite store path | Not-inventoried | Source-confirmed | N/A |
| verify-models | `-max` | CLI | Implemented | Wired | Max models/provider through pipeline | Not-inventoried | Source-confirmed | N/A |
| verify-models | `-min-score` | CLI | Implemented | Wired | Min overall score to persist | Not-inventoried | Source-confirmed | N/A |
| verify-models | `-timeout` | CLI | Implemented | Wired | Overall verification timeout | Not-inventoried | Source-confirmed | N/A |

## cmd/model-bridge

LLMsVerifier→component+agent bridge (no local models): selects the best LLMsVerifier-scored model, exposes it to agents/components via a CLI and an MCP stdio server wired in `.mcp.json`. Source: `cmd/model-bridge/{main.go,mcp.go}`, `pkg/bridge/`, commit ab1bed3 (+ routing seam a5860b2).

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| model-bridge | LLMsVerifier→agent bridge CLI | CLI | Implemented | Wired | Selects best LLMsVerifier-scored model (top-1 + fallback chain) and translates/invokes through it; no local models | Covered (pkg/bridge unit+integration, `go test -race` green this session) | **PASS — real run captured** (best-model selection `novita/Sao10K/L3-8B-Stheno-v3.2` score 0.919; `Invoke` EN→ES "Good morning, friend." → "Buenos días, amigo.") | **CONFIRMED** — real-use video `/Volumes/T7/Downloads/Recordings/helixtranslate-bridge-bestmodel-translate-20260615.mp4` (asciinema→agg→ffmpeg pipeline; ffprobe 8.0s / 80 frames / 790×560; ≥6 frames readable — selected model `novita/Sao10K/L3-8B-Stheno-v3.2` (score 0.919, base_url https://api.novita.ai/v3/openai) + live OUTPUT "Buenos días, amigo." visible, not frozen/spinner per §11.4.107) |
| model-bridge | MCP stdio server (`.mcp.json`) | CLI/Infra | Implemented | Wired | Exposes the selected model to MCP clients over stdio; wired as `model-bridge` in `.mcp.json` (`./build/model-bridge`) | Covered (`cmd/model-bridge/mcp_test.go`) | **PASS — real MCP JSON-RPC round-trip captured** (initialize→tools/list→`bridge_best_model`→`bridge_invoke` over stdio; tool `bridge_invoke` returned live verified-model translation "Le pont relie deux rives.", isError:None) | **CONFIRMED** — real-use video `/Volumes/T7/Downloads/Recordings/helixtranslate-bridge-mcp-stdio-20260615.mp4` (asciinema→agg→ffmpeg; ffprobe 11.88s / 10 frames / 790×560; ≥6 frames readable showing the progressive REAL MCP-stdio session — `tools/list`=[bridge_invoke, bridge_best_model, bridge_list], `bridge_best_model` strongest `novita/Sao10K/L3-8B-Stheno-v3.2` score 0.919 + fallback chain, and tool `bridge_invoke` real verified-model OUTPUT "Le pont relie deux rives." with isError:None — not frozen/spinner per §11.4.107) |

## cmd/workable-items

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| workable-items | §11.4.93 workable-items SQLite tool | CLI | Implemented | Wired | Bidirectional sync between Issues/Fixed markdown and SQLite single-source-of-truth | Not-inventoried | **PASS — real DB CLI output captured** (`./build/workable-items validate -db docs/workable_items.db` → `OK: DB docs/workable_items.db matches markdown (81 items)`, then `list` printed real ATM-NNN rows with status + descriptions — ATM-001 `splitText dropped paragraph separators…`, ATM-005 `Distributed ResultCache spurious eviction…`, ATM-014 `Preparation failed-chapter analysis misattribution…`; Wave-3d, asciinema→agg→ffmpeg, ffprobe 5.0s/790×560 yuv420p + 10 frames at fps=2, early validate-OK frame ≠ late list frame proving live/advancing per §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-workable-items-wave3d-20260616.mp4 |
| workable-items | `sync md-to-db` | CLI | Implemented | Wired | Parse ATM-NNN headings from markdown into DB | Not-inventoried | Source-confirmed | N/A |
| workable-items | `sync db-to-md` | CLI | Implemented | Wired | Regenerate Issues_Summary/Fixed_Summary from DB | Not-inventoried | Source-confirmed | N/A |
| workable-items | `validate` | CLI | Implemented | Wired | Verify DB matches markdown; exit non-zero on drift | Not-inventoried | Source-confirmed | N/A |
| workable-items | `diff` | CLI | Implemented | Wired | Print md-vs-db drift human-readably | Not-inventoried | Source-confirmed | N/A |
| workable-items | `list` | CLI | Implemented | Wired | Print items in DB | Not-inventoried | Source-confirmed | N/A |
| workable-items | `record-event` | CLI | Implemented | Wired | Append event to item_history audit trail | Not-inventoried | Source-confirmed | N/A |
| workable-items | path/value flags | CLI | Implemented | Wired | `-issues`/`-fixed`/`-db`/`-issues-summary`/`-fixed-summary`/`-atm`/`-event`/`-on`/`-note` | Not-inventoried | Source-confirmed | N/A |

---

# 2. Library Packages — pkg/

## pkg/translator/llm — LLM provider clients & factory

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| LLM Core | LLMClient interface & factory | Library | Implemented | Wired | `LLMClient` interface + `NewLLMTranslator` factory dispatching to ~32 providers | Not-inventoried | Source-confirmed | N/A |
| LLM Core | Type-coercion utilities | Library | Implemented | Wired | JSON config conversion helpers | Not-inventoried | Source-confirmed | N/A |
| LLM Core | Verified factory | Library | Implemented | Wired | LLMsVerifier-gated translator construction w/ fallback chain (CONST-034) | Not-inventoried | Source-confirmed | N/A |
| LLM Core | Mock client | Library | Implemented | Wired | Test mock LLMClient with fault injection (test-only) | Not-inventoried | Source-confirmed | N/A |
| OpenAI | OpenAI client | Library | Implemented | Wired | GPT chat-completions translation | Not-inventoried | Source-confirmed | PENDING |
| Anthropic | Anthropic Claude client | Library | Implemented | Wired | Claude /v1/messages translation | Not-inventoried | Source-confirmed | PENDING |
| Google | Gemini client | Library | Implemented | Wired | Google Gemini generativelanguage API | Not-inventoried | **PASS — real Gemini translation captured** (Wave-3b real-use video: `unified-translator -provider gemini -model gemini-2.0-flash`, real key, EN→SR-latin "Staro svjetiljko stajalo je samotočno na kamenitoj obali. Noću svakog dana njeno svjetlo vodilo brodove bezbedno kući kroz oluju."; ffprobe 7.16s / 790×560 / non-zero frames; ≥6 frames extracted, f_05 content-verified showing provider+model+distinct SR output per §11.4.107; output distinct from deepseek/zhipu/cohere proving genuine per-provider model) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-provider-gemini-translate-wave3b-20260616.mp4 |
| DeepSeek | DeepSeek client | Library | Implemented | Wired | DeepSeek OpenAI-compatible chat-completions | Not-inventoried | **Exercised in unified-translator real run** (re-confirmed Wave-3b: `-provider deepseek -model deepseek-chat` EN→SR-latin "Stari svjetionik stajao je sam na kamenitoj obali…"; ffprobe 4.7s/790×560, f_03 content-verified — additional clip, no double-count) | Confirmed (via unified-translator):/Volumes/T7/Downloads/Recordings/helixtranslate-cli-deepseek-translation-FIXED_20260615_163824.mp4 (additional Wave-3b: /Volumes/T7/Downloads/Recordings/helix_translate-provider-deepseek-translate-wave3b-20260616.mp4) |
| Zhipu | Zhipu (GLM) client | Library | Implemented | Wired | Zhipu GLM /api/paas/v4 | Not-inventoried | **PASS — real Zhipu translation captured** (Wave-3b real-use video: `unified-translator -provider zhipu -model glm-4-flash`, real key, EN→SR-latin "Stara svjetiljka stojeća samostalno na kamenitoj obali. Noću svaki put svjetlo vodi brodove bezbedno kući kroz oluju."; ffprobe 5.05s / 790×560 / non-zero frames; ≥6 frames extracted, f_04 content-verified per §11.4.107; output distinct from deepseek/gemini/cohere) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-provider-zhipu-translate-wave3b-20260616.mp4 |
| Alibaba | Qwen client | Library | Implemented | Wired | Qwen/DashScope (OAuth2-capable) | Not-inventoried | Source-confirmed | PENDING |
| Ollama | Ollama local client | Library | **Obsolete (→ Fixed.md)** | Wired | Local /api/generate inference — REMOVED (bridge phase-2 R-2). See Obsolete-Details below. | Not-inventoried | Source-confirmed | N/A |
| llama.cpp | llama.cpp client | Library | **Obsolete (→ Fixed.md)** | Wired | llama-cli wrapper w/ hardware detection, model download, GPU accel — REMOVED (bridge phase-2 R-3). See Obsolete-Details below. | Not-inventoried | Source-confirmed | N/A |
| llama.cpp | llama.cpp multi-worker coordinator | Library | **Obsolete (→ Fixed.md)** | Wired | Worker pool for parallel local llama.cpp inference — REMOVED (bridge phase-2 R-3). See Obsolete-Details below. | Not-inventoried | Source-confirmed | N/A |

> **Obsolete-Details (Ollama + llama.cpp provider/coordinator rows above):** Since=bridge phase-2 R-2/R-3 (2026-06-16); Reason=`feature-removed` (local-runtime inference removed; the default path now sources the LLMsVerifier bridge — no local llama.cpp/Ollama); Superseding-item=`pkg/bridge`; Operator-approval=2026-06-15 D1/D2 (forbid all local runtimes) + R-1 redirect; Triple-check evidence=`pkg/translator/llm/{ollama.go,llamacpp.go,llamacpp_provider.go}` removed (FACT: `git ls-files` empty) + `CM-NO-LOCAL-RUNTIME` gate PASS. The KEPT API provider clients (OpenAI/Anthropic/DeepSeek/…) in this section are NOT Obsolete.
| Groq | Groq client | Library | Implemented | Wired | OpenAI-compatible wrapper | Not-inventoried | Source-confirmed | PENDING |
| Cohere | Cohere client | Library | Implemented | Wired | command-r via OpenAI-compat endpoint | Not-inventoried | **PASS — real Cohere translation captured** (Wave-3b real-use video: `unified-translator -provider cohere -model command-r-08-2024 -base-url https://api.cohere.ai/compatibility/v1`, real key, EN→SR-latin "Stari svjetionik stojeo je sam na kamenitoj obali. Noću svaki put svijeća njegova vodila brodove sigurno kući kroz oluju."; log shows provider=cohere; ffprobe 6.24s / 790×560 / non-zero frames; ≥6 frames extracted, f_04 content-verified per §11.4.107; output distinct from deepseek/gemini/zhipu) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-provider-cohere-translate-wave3b-20260616.mp4 |
| Mistral | Mistral client | Library | Implemented | Wired | OpenAI-compatible wrapper | Not-inventoried | Source-confirmed | PENDING |
| xAI | xAI (Grok) client | Library | Implemented | Wired | OpenAI-compatible wrapper | Not-inventoried | Source-confirmed | PENDING |
| Replicate | Replicate client | Library | Implemented | Wired | Async prediction-polling API | Not-inventoried | Source-confirmed | PENDING |
| Cerebras | Cerebras client | Library | Implemented | Wired | OpenAI-compatible wrapper (`https://api.cerebras.ai/v1`, model `llama3.1-70b`) | Not-inventoried | **PASS — real EN→ES translation captured** (via unified-translator `-provider cerebras -model llama3.1-70b`: "The morning sun rose over the quiet village." → "El sol de la mañana se alzó sobre el tranquilo pueblo." — distinct from SambaNova's output, proving a genuine per-provider model; Wave-3a, asciinema→agg→ffmpeg, ffprobe 4.03s/4 frames + 11 extracted frames, ≥6 frames content-verified, start-frame ≠ output-frame per §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-provider-cerebras-translate-wave3a-20260616.mp4 |
| Cloudflare | Cloudflare Workers AI client | Library | Implemented | Wired | /client/v4 Workers AI | Not-inventoried | Source-confirmed | PENDING |
| SiliconFlow | SiliconFlow client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| Hyperbolic | Hyperbolic client | Library | Implemented | Wired | OpenAI-compatible wrapper (`https://api.hyperbolic.xyz/v1`, model `meta-llama/Meta-Llama-3.1-70B-Instruct`) | Not-inventoried | **PASS — real EN→ES translation captured** (via unified-translator `-provider openai -base-url https://api.hyperbolic.xyz/v1 -model meta-llama/Meta-Llama-3.1-70B-Instruct`: "The morning sun rose over the quiet village. An old fisherman walked along the shore…" → "Al amanecer, el sol salía sobre el tranquilo pueblo. Un anciano pescador caminaba por la orilla…" — distinct verbs (salía/caminaba) from Fireworks's output (ascendió/paseaba), proving a genuine per-provider model; Wave-3c, asciinema→agg→ffmpeg, ffprobe 8.01s/790×560 yuv420p + 16 extracted frames at fps=2, ≥6 frames content-verified, start-frame (input only) ≠ output-frame (Spanish) per §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-provider-hyperbolic-translate-wave3c-20260616.mp4 |
| TogetherAI | TogetherAI client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| SambaNova | SambaNova client | Library | Implemented | Wired | OpenAI-compatible wrapper (`https://api.sambanova.ai/v1`, model `Meta-Llama-3.1-70B-Instruct`) | Not-inventoried | **PASS — real EN→ES translation captured** (via unified-translator `-provider sambanova -model Meta-Llama-3.1-70B-Instruct`: "The morning sun rose over the quiet village." → "El sol matutino se alzó sobre el tranquilo pueblo." — distinct from Cerebras's output, proving a genuine per-provider model; Wave-3a, asciinema→agg→ffmpeg, ffprobe 4.39s/4 frames + 12 extracted frames, ≥6 frames content-verified, start-frame ≠ output-frame per §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-provider-sambanova-translate-wave3a-20260616.mp4 |
| Kimi | Kimi (Moonshot) client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| Novita | Novita client | Library | Implemented | Wired | /v3/openai-compatible wrapper | Not-inventoried | Source-confirmed | PENDING |
| NLPCloud | NLPCloud client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| Upstage | Upstage (Solar) client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| Sarvam | Sarvam client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| Modal | Modal client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| PublicAI | PublicAI client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| NIA | NIA client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| Vulavula | Vulavula client | Library | Implemented | Wired | OpenAI-compatible wrapper (specifics UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |

## pkg/translator — translation core

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| Translator Core | Translator interface + BaseTranslator | Library | Implemented | Wired | `Translate`/`TranslateWithProgress`/`GetStats` + base impl w/ caching & stats | Not-inventoried | Source-confirmed | N/A |
| Translator Core | Translation config | Library | Implemented | Wired | `TranslationConfig` (provider/model/key/temp/tokens/lang-pair/script) | Not-inventoried | Source-confirmed | N/A |
| Translator Core | UniversalTranslator | Library | Implemented | Wired | High-level ebook translator: lang detection, per-chapter translation, progress events | Not-inventoried | Source-confirmed | PENDING |

## pkg/ebook — format parsers & writers

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| ebook | Universal parser dispatcher | Library | Implemented | Wired | Multi-format parser (FB2/EPUB/TXT/HTML/DOCX/PDF) | Not-inventoried | Source-confirmed | N/A |
| ebook | FB2 parse | Library | Implemented | Wired | Parse FictionBook 2.0 → Book, namespace-preserving | Not-inventoried | Source-confirmed | N/A |
| ebook | FB2 write | Library | Implemented | Wired | Write Book → FB2 XML | Not-inventoried | Source-confirmed | N/A |
| ebook | EPUB parse | Library | Implemented | Wired | Parse EPUB (ZIP+OPF/NCX) | Not-inventoried | Source-confirmed | N/A |
| ebook | EPUB write | Library | Implemented | Wired | Write Book → EPUB 2.0 ZIP | Not-inventoried | Source-confirmed | N/A |
| ebook | DOCX parse | Library | Implemented | Wired | Parse .docx (archive/zip + encoding/xml) | Not-inventoried | Source-confirmed | N/A |
| ebook | HTML parse | Library | Implemented | Wired | Parse HTML → Book | Not-inventoried | Source-confirmed | N/A |
| ebook | TXT parse | Library | Implemented | Wired | Parse plain text (configurable 64 MiB line cap) | Not-inventoried | Source-confirmed | N/A |
| ebook | PDF text extraction | Library | Implemented | Wired | Extract text via ledongthuc/pdf | Not-inventoried | Source-confirmed | N/A |
| ebook | DOCX write (output) | Library | Implemented | Wired | Pure-Go OOXML writer `pkg/ebook/docx_writer.go` (stdlib archive/zip + encoding/xml, zero new deps); emits the 4 required OOXML parts; XML specials auto-escaped via xml.Marshal chardata. Wired into unified-translator generateOutput (`case docx`). Real DeepSeek EN→ES → `garden_es.docx` = `Microsoft Word 2007+`, well-formed WordprocessingML with the real translation. Guards: pkg/ebook/docx_writer_test.go + .docx subtest in cmd/unified-translator/output_format_test.go (mutation-proven) | Covered | **PASS — real artifact verified + Wave-2 video** (produced .docx = `Microsoft Word 2007+`, 4 OOXML parts, real translated content + correct XML escaping; commit 87cd2be; Wave-2 real-use video: unified-translator DOCX output, openai/gpt-4, the `.docx` carries real Spanish text — validator ffprobe+frame-verified) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-docx-output-20260616-112943.mp4 |
| ebook | PDF write (output) | Library | **Implemented** | Wired | PDF writer `pkg/ebook/pdf_writer.go` (commit fb265e7): renders the in-memory Book to UTF-8 HTML5 then hands it to `weasyprint` to emit a VALID, Cyrillic-faithful PDF (Standard-14 fonts can't render Cyrillic — weasyprint embeds glyphs, avoiding a silent-glyph-drop bluff). Wired into unified-translator generateOutput (`case .pdf` → `ebook.NewPDFWriter().Write`, main.go:1160); `.pdf` is a supported `-output` extension. Honest typed `ErrWeasyPrintUnavailable` when weasyprint absent (§11.4.3/§11.4.6). Guards: pkg/ebook/pdf_writer_test.go (`TestPDFWriter_ProducesValidPDFCarryingTranslatedText`, `_UnavailableWeasyPrint`) | Covered | **PASS** — `go test ./pkg/ebook` green this session (`TestPDFWriter_ProducesValidPDFCarryingTranslatedText` asserts a valid PDF carrying the translated text; weasyprint present at /opt/homebrew/bin/weasyprint); Wave-2 real-use video: unified-translator PDF output, openai/gpt-4, valid PDF with real Spanish (validator ffprobe+frame-verified) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-pdf-output-20260616-113057.mp4 |

## pkg/fb2 — FB2-specific XML

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| fb2 | FB2 parser (mixed-content) | Library | Implemented | Wired | Parse FB2 preserving paragraphs/verses/inline mixed content | Not-inventoried | Source-confirmed | N/A |
| fb2 | FB2→Markdown converter | Library | Implemented | Wired | Convert FB2 → Markdown (author/annotation/epigraph/poem/cite) | Not-inventoried | Source-confirmed | N/A |

## pkg/format

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| format | Format detector | Library | Implemented | Wired | Detect format by magic bytes/extension/content; ZIP disambiguation (EPUB/DOCX/AZW3) | Not-inventoried | Source-confirmed | N/A |

## pkg/markdown — EPUB↔Markdown workflow

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| markdown | EPUB→Markdown converter | Library | Implemented | Wired | EPUB→MD w/ YAML frontmatter, image extraction, GFM tables | Not-inventoried | Source-confirmed | N/A |
| markdown | Markdown→EPUB converter | Library | Implemented | Wired | MD→EPUB 2.0 w/ OPF/NCX, lists/tables/blockquotes | Not-inventoried | Source-confirmed | N/A |
| markdown | Simple workflow | Library | Implemented | Wired | ebook→markdown→translate→ebook pipeline w/ progress | Not-inventoried | Source-confirmed | PENDING |
| markdown | Markdown translator | Library | Implemented | Wired | Line-by-line MD translation preserving formatting/code blocks | Not-inventoried | Source-confirmed | PENDING |

## pkg/script

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| script | Cyrillic↔Latin conversion | Library | Implemented | Wired | Bidirectional Serbian script conversion, context-aware digraphs | Not-inventoried | Source-confirmed | N/A |

## pkg/storage

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| storage | Storage interface | Library | Implemented | Wired | Abstract session + translation-cache persistence | Not-inventoried | Source-confirmed | N/A |
| storage | PostgreSQL backend | Library | Implemented | Wired | Pooled Postgres w/ UNIQUE-tuple dedup | Not-inventoried | Source-confirmed | N/A |
| storage | Redis backend | Library | Implemented | Wired | Redis cache, sha256-hashed tuple keys (collision-proof) | Not-inventoried | Source-confirmed | N/A |
| storage | SQLite backend | Library | Implemented | Wired | SQLite (optional SQLCipher) w/ UNIQUE-tuple migrations | Not-inventoried | Source-confirmed | N/A |
| storage | Cache-key hashing | Library | Implemented | Wired | Length-prefixed tuple encoding for collision-resistant keys | Not-inventoried | Source-confirmed | N/A |

## pkg/security

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| security | JWT auth service | Library | Implemented | Wired | JWT generate/validate/refresh w/ timing-attack mitigation | Not-inventoried | Source-confirmed | N/A |
| security | API-key store | Library | Implemented | Wired | Thread-safe key validation w/ expiry & revocation | Not-inventoried | Source-confirmed | N/A |
| security | Rate limiter | Library | Implemented | Wired | Token-bucket per-key limiting w/ cleanup & stats | Not-inventoried | Source-confirmed | N/A |
| security | User auth service | Library | Implemented | Wired | Login/user-create w/ timing & enumeration mitigation | Not-inventoried | Source-confirmed | N/A |

## pkg/verification — quality verification & polishing

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| verification | Heuristic verifier | Library | Implemented | Wired | Untranslated-block/HTML-artifact detection, quality scoring | Not-inventoried | Source-confirmed | N/A |
| verification | Multi-pass polisher | Library | Implemented | **Wired into unified-translator (`-multipass`, d53e085)** | Iterative multi-LLM polishing w/ note-taking & consensus. Now reachable via the unified-translator opt-in `-multipass` flag (+ `-multipass-passes`, `-multipass-db`; default OFF, zero overhead/behavior change when absent). NOTE: distinct from `-verify` (the CLI's own per-step output check) — `-multipass` invokes this `pkg/verification` engine. Guard: cmd/unified-translator/multipass_wiring_test.go (httptest + real DeepSeek). §11.4.150 research: docs/research/20260615_video_surfaced_fixes/verification_multipass_wiring.md | Not-inventoried | **PENDING_FORENSICS (§11.4.138 demoted)** — the prior `helixtranslate-cli-multipass-verify-20260615.mp4` PASS is superseded: the default model `gpt-4` fails the `ValidModels` whitelist for deepseek/most providers and the polish pass silently no-ops keeping the base translation (wave-5 `docs/qa/video_wave5_20260616-173814/EVIDENCE.md`), and even with an explicit valid model the path can write an LLM meta-response/UNCHANGED-marker instead of polished prose (wave-3e `docs/qa/wave3e-multipass-regression/FINDING.md`). Tracked source fix + re-record owed before re-confirmation. | PENDING_FORENSICS (was Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-cli-multipass-verify-20260615.mp4 — superseded by §11.4.138) |
| verification | Book polisher | Library | Implemented | Wired | Multi-LLM per-dimension scoring & consensus validation | Not-inventoried | Source-confirmed | PENDING |
| verification | Polishing reporter | Library | Implemented | Wired | Stats aggregation (consensus rates, issue/provider breakdown) | Not-inventoried | Source-confirmed | N/A |
| verification | Polishing database | Library | Implemented | Wired | SQLite persistence for multi-pass sessions/pass records | Not-inventoried | Source-confirmed | N/A |
| verification | Literary notes | Library | Implemented | Wired | Character/tone/theme/terminology/cultural note collection | Not-inventoried | Source-confirmed | N/A |

## internal/verifier — LLMsVerifier consumer integration

Consumer-side LLMsVerifier verification gate (client + scoring + discovery + selection). Source: `internal/verifier/pipeline.go`.

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| internal/verifier | Affirmative-response hard-gate | Library | Implemented | Wired (verification pipeline) | Correctness fix (commit 97a8afd): a model with `affirmative_response=0` is now a HARD disqualifier — rejected by the verification gate regardless of other scores | Covered + **§11.4.115 polarity guard** `pipeline_affirmative_gate_test.go` (GREEN-default: the standing suite asserts the defect ABSENT; RED reproduction is OPT-IN only under `RED_MODE=1`) | **PASS** — observed this session: default suite GREEN (`ok internal/verifier`); the defect reproduction FAILs only under explicit `RED_MODE=1` (by-design §11.4.115) | N/A (§11.4.6 — internal verification gate, no user-visible surface) |

## pkg/preparation — pre-translation analysis

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| preparation | Preparation coordinator | Library | Implemented | Wired | Multi-pass content analysis (character/terminology/cultural) | Not-inventoried | Source-confirmed | PENDING |
| preparation | Preparation-aware translator | Library | Implemented | Wired | Wraps base translator w/ analysis phase + context passing | Not-inventoried | Source-confirmed | PENDING |
| preparation | Prompt builder | Library | Implemented | Wired | Context-aware analysis prompts refined across passes | Not-inventoried | Source-confirmed | N/A |
| preparation | Domain types | Library | Implemented | Wired | Content-classification/character/cultural/terminology structs | Not-inventoried | Source-confirmed | N/A |
| preparation | Utilities | Library | Implemented | Wired | JSON serialization + human-readable summaries | Not-inventoried | Source-confirmed | N/A |

## pkg/coordination — multi-LLM coordination

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| coordination | Multi-LLM coordinator | Library | Implemented | Wired | Multiple LLM instances w/ round-robin, retry, distributed fallback | Not-inventoried | Source-confirmed | PENDING |
| coordination | Multi-LLM translator wrapper | Library | Implemented | Wired | Adapts coordinator to `translator.Translator` interface | Not-inventoried | Source-confirmed | N/A |

## pkg/distributed — SSH distributed workers

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| distributed | Coordinator (distribute + aggregate) | Library | Implemented | Wired | Distributes work across remote LLM instances, round-robin, failover to local | Not-inventoried | Source-confirmed | PENDING |
| distributed | SSH connection pool | Library | Implemented | Wired | Pooled SSH connections w/ idle-timeout cleanup | Not-inventoried | Source-confirmed | N/A |
| distributed | Worker pairing/discovery | Library | Implemented | Wired | Discover/pair/health-check remote translator services | Not-inventoried | Source-confirmed | PENDING |
| distributed | Fallback / graceful degradation | Library | Implemented | Wired | Retry w/ backoff, degraded-mode entry/exit, recovery tracking | Not-inventoried | Source-confirmed | N/A |
| distributed | Version manager | Library | Implemented | Wired | Version check/update/rollback, signed packages, backups, drift alerts (no LLM) | Not-inventoried | Source-confirmed | N/A |
| distributed | Performance instrumentation | Library | Implemented | Wired | Conn pooling, TTL result cache, circuit breaker, batch processing | Not-inventoried | Source-confirmed | N/A |
| distributed | Security hardening | Library | Implemented | Wired | SSH host-key verify, TLS validation, network ACLs, security event log | Not-inventoried | Source-confirmed | N/A |
| distributed | Manager / orchestration | Library | Implemented | Wired | Init/discover/pair/version-manage + translation delegation | Not-inventoried | Source-confirmed | PENDING |

## pkg/sshworker

> **Obsolete-Details:** Since=bridge phase-2 R-2/R-4 (2026-06-16); Reason=`feature-removed` (local-runtime + SSH-local translation removed; default path now sources the LLMsVerifier bridge — no local llama.cpp/Ollama, no SSH-local worker); Superseding-item=`pkg/bridge` (LLMsVerifier→agent bridge); Operator-approval=2026-06-15 D1/D2 (forbid all local runtimes) + R-1 redirect + the R-4 “keep distributed API, remove SSH-local” decision (2026-06-16); Triple-check evidence=source files removed (FACT: `git ls-files` empty for the removed cmd/* dirs + pkg/sshworker + pkg/modelsbridge + ollama.go/llamacpp*.go) + `CM-NO-LOCAL-RUNTIME` gate PASS. The KEPT distributed-API path is NOT Obsolete.
| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| sshworker | SSH worker | Library | **Obsolete (→ Fixed.md)** | Wired | SSH exec, file up/download, codebase sync, key generation | Not-inventoried | Source-confirmed | N/A |
| sshworker | Monitored worker | Library | **Obsolete (→ Fixed.md)** | Wired | Event-monitored progress tracking for long commands | Not-inventoried | Source-confirmed | N/A |

## pkg/deployment

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| deployment | Deployment orchestrator | Library | Implemented | Wired | Automated multi-host instance deployment | Not-inventoried | Source-confirmed | PENDING |
| deployment | Docker orchestrator | Library | Implemented | Wired | Generate/manage docker-compose for multi-service deploy | Not-inventoried | Source-confirmed | N/A |
| deployment | SSH deployer | Library | Implemented | Wired | SSH-based deploy w/ validation & error handling | Not-inventoried | Source-confirmed | N/A |
| deployment | Network discovery | Library | Implemented | Wired | UDP service discovery/broadcast | Not-inventoried | Source-confirmed | N/A |
| deployment | API comm logger | Library | Implemented | Wired | Retrofit-style REST request/response logging between nodes | Not-inventoried | Source-confirmed | N/A |
| deployment | Deployment types | Library | Implemented | Wired | DeploymentPlan/Status/NetworkService/APICommunicationLog | Not-inventoried | Source-confirmed | N/A |

## pkg/grpc

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| grpc | gRPC TranslationService | Library | Implemented | Wired | gRPC server: sessions, event streaming, provider registry, concurrency | Not-inventoried | Source-confirmed | PENDING |
| grpc | Core translator pipeline | Library | Implemented | Wired | Translation job exec w/ step tracking + progress via llm factory | Not-inventoried | Source-confirmed | PENDING |

## pkg/api (library layer)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api | REST API server | Library | Implemented | Wired | HTTP server: auth, CSRF, rate-limit, routing | Not-inventoried | Source-confirmed | PENDING |
| api | Translation endpoints | Library | Implemented | Wired | text/FB2/ebook translate, batch ops, lang/provider listing | Not-inventoried | Source-confirmed | PENDING |
| api | Batch translation handlers | Library | Implemented | Wired | Batch requests w/ queue, progress, concurrency | Not-inventoried | Source-confirmed | PENDING |
| api | Verifier endpoints | Library | Implemented | Wired | LLMsVerifier model verification/scoring/selection; /verified-models | Not-inventoried | Source-confirmed | PENDING |

## pkg/events, websocket, batch, progress, report, logger, hardware, hash

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| events | Event pub/sub bus | Library | Implemented | Wired | Thread-safe pub/sub w/ subscription IDs + type filtering | Not-inventoried | Source-confirmed | N/A |
| websocket | WebSocket hub | Library | Implemented | Wired | Real-time session event broadcast to dashboards | Not-inventoried | Source-confirmed | PENDING |
| batch | Batch processor | Library | Implemented | Wired | File/dir translation w/ concurrency & progress | Not-inventoried | Source-confirmed | PENDING |
| progress | Progress tracker | Library | Implemented | Wired | Chapter/section progress w/ ETA | Not-inventoried | Source-confirmed | N/A |
| report | Report generator | Library | Implemented | Wired | Session reporting (issues/warnings/logs/results) | Not-inventoried | Source-confirmed | N/A |
| logger | Structured logger | Library | Implemented | Wired | Text/JSON logging w/ levels | Not-inventoried | Source-confirmed | N/A |
| hardware | Hardware detector | Library | Implemented | Wired | CPU/RAM/GPU detection + model-size estimation | Not-inventoried | Source-confirmed | N/A |
| hash | Codebase hash generator | Library | Implemented | Wired | SHA256 of source/config/deps/binaries | Not-inventoried | Source-confirmed | N/A |

## pkg/language

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| language | Language registry / heuristic detector | Library | Implemented | Wired | ISO 639-1 registry + heuristic detection (18+ langs) | Not-inventoried | Source-confirmed | N/A |
| language | LLM language detector | Library | Implemented | Wired | LLM-driven language detection | Not-inventoried | Source-confirmed | PENDING |

## pkg/models & modelsbridge

> **Obsolete-Details:** Since=bridge phase-2 R-2/R-4 (2026-06-16); Reason=`feature-removed` (local-runtime + SSH-local translation removed; default path now sources the LLMsVerifier bridge — no local llama.cpp/Ollama, no SSH-local worker); Superseding-item=`pkg/bridge` (LLMsVerifier→agent bridge); Operator-approval=2026-06-15 D1/D2 (forbid all local runtimes) + R-1 redirect + the R-4 “keep distributed API, remove SSH-local” decision (2026-06-16); Triple-check evidence=source files removed (FACT: `git ls-files` empty for the removed cmd/* dirs + pkg/sshworker + pkg/modelsbridge + ollama.go/llamacpp*.go) + `CM-NO-LOCAL-RUNTIME` gate PASS. The KEPT distributed-API path is NOT Obsolete.
| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| models | Model registry | Library | Implemented | Wired | Translation-model metadata registry (50+ models) | Not-inventoried | Source-confirmed | N/A |
| models | Model downloader | Library | Implemented | Wired | HTTP model download w/ caching + sha256 verification | Not-inventoried | Source-confirmed | N/A |
| models | User domain | Library | Implemented | Wired | User struct, bcrypt hashing, in-memory repo | Not-inventoried | Source-confirmed | N/A |
| models | Error definitions | Library | Implemented | Wired | Domain error constants | Not-inventoried | Source-confirmed | N/A |
| models | Verifier registry | Library | Implemented | Wired | LLMsVerifier-backed registry w/ thresholds & filtering | Not-inventoried | Source-confirmed | PENDING |
| modelsbridge | Models bridge | Library | **Obsolete (→ Fixed.md)** | Wired | Adapter mapping models.LLMRequest/Response → llm.LLMClient | Not-inventoried | Source-confirmed | N/A |

## pkg/bridge — LLMsVerifier→component+agent bridge

In-process bridge (no local models) that selects the best LLMsVerifier-scored model and routes component/agent calls to it; backs `cmd/model-bridge` CLI + MCP. Source: `pkg/bridge/bridge.go`, commit ab1bed3 (+ routing mutation seam a5860b2).

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| bridge | Best-model selection | Library | Implemented | Wired (via cmd/model-bridge CLI+MCP) | Picks top-1 LLMsVerifier-scored model with a provider-diverse fallback chain; no local models | Covered (`pkg/bridge/bridge_test.go` unit+integration, -race green) | **PASS — real selection captured** (`novita/Sao10K/L3-8B-Stheno-v3.2` score 0.919; provider-diverse fallback chain novita/mistral/groq/siliconflow) | **CONFIRMED** — real-use video `/Volumes/T7/Downloads/Recordings/helix_translate-bridge-ensemble-translate-20260616-104846.mp4` (Wave 1; asciinema→agg→ffmpeg; ffprobe 7.77s / 33 frames / 8 distinct frame-hashes; ≥6 frames readable — `best-model` shows strongest `novita/Sao10K/L3-8B-Stheno-v3.2` 0.919 + the provider-diverse fallback chain, then real `invoke` OUTPUT "Buenos días, querido amigo." visible, not frozen per §11.4.107) |
| bridge | `Invoke` routes to BestModel | Library | Implemented | Wired | Routes the invocation to the selected best model | Covered + **§1.1 paired mutation seam** `TestBridge_Invoke_RoutesToBestModel` (RED FAILs on mis-route, -race green) | **PASS** (`Invoke`→"Buenos días, amigo." through selected model; routing guard catches mis-route) | N/A (internal routing — surfaced via cmd/model-bridge) |
| bridge | `BestTranslator` (single strongest translator + fallback chain) | Library | Implemented | Wired (R-1 redirect seam) | `bridge.go:242` — returns the single strongest verified `translator.Translator` plus its fallback-chain model IDs for the task; the non-ensemble single-translator source | Covered at unit level (`pkg/bridge/bridge_test.go`, `pkg/bridge/ensemble_test.go`, `go test ./pkg/bridge` green) — but unit tests use whitelisted models, masking the runtime bug below | **PARTIAL — UNIT PASS, RUNTIME BLOCKED (§11.4.108):** `go test ./pkg/bridge` → ok, BUT a real run via unified-translator/gRPC FAILS at translate time — the verified strongest model (`novita/Sao10K/L3-8B-Stheno-v3.2`) is rejected by the stale `ValidModels` whitelist in `VerifiedFactory.CreateTranslatorWithFallback`→`NewLLMTranslatorWithConfig` (`pkg/translator/llm/llm.go:237`), unlike the `BestModel`/invoke path which correctly bypasses it. See **Wave-1 discovered blocker** below | PENDING — NOT recordable until the whitelist blocker is fixed (recording a "working" video here would be a §11.4.107 bluff) |
| bridge | `BestTranslatorFunc` (context-only single-translator closure) | Library | Implemented | Wired (R-1 redirect seam) | `ensemble.go:37` — context-only closure yielding the single strongest verified translator (fallback chain discarded); plain function value passed to single-translator sites WITHOUT importing the bridge (§11.4.28 decoupling); no API key captured/logged (§11.4.10) | Unit-covered (`pkg/bridge/ensemble_test.go`, -race green) — masks the runtime bug | **PARTIAL — UNIT PASS, RUNTIME BLOCKED (§11.4.108):** delegates to `BestTranslator`; same `ValidModels`-whitelist runtime failure (see Wave-1 discovered blocker) | PENDING — NOT recordable until the blocker is fixed |
| bridge | `EnsembleFactory` (provider-diverse ensemble seam) | Library | Implemented | Wired (R-1 ensemble redirect) | `ensemble.go:25` — context-only closure yielding the provider-diverse verified translator set (one per distinct provider, strongest-first); structural shape of the per-component `EnsembleTranslatorFactory` seams, passed to `NewMultiLLMCoordinatorWithFactory` / `NewPreparationCoordinatorWithFactory` / `NewBookPolisherWithFactory` WITHOUT those packages importing the bridge (§11.4.28); no API key captured/logged (§11.4.10) | Unit-covered (`pkg/bridge/ensemble_test.go`, -race green) | **PASS — Wave-1 blocker FIXED, runtime confirmed (§11.4.108):** backed by `ProviderDiverseTranslators`; the `ValidModels`-whitelist blocker (caveat 8) is resolved — verified non-whitelisted models now translate. Wave-2 provider clips show real per-provider translations through this ensemble seam | Confirmed (via ProviderDiverseTranslators provider clips):/Volumes/T7/Downloads/Recordings/helix_translate-provider-novita-translate-20260616-114032.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-mistral-translate-20260616-114039.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-groq-translate-20260616-114045.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-siliconflow-translate-20260616-114051.mp4 |
| bridge | `ProviderDiverseTranslators` | Library | Implemented | Wired (backs `EnsembleFactory`) | `bridge.go:527` — returns provider-diverse verified `translator.Translator` set (one per distinct provider, strongest-first); the ensemble translator source | Covered (`pkg/bridge/bridge_test.go`, `pkg/bridge/ensemble_test.go`, -race green) | **PASS — Wave-1 blocker FIXED, runtime confirmed (§11.4.108):** the `ValidModels`-whitelist blocker (caveat 8) is resolved — `ProviderDiverseTranslators` now yields working verified translators per provider. Wave-2 provider clips show real EN→ES translations through each provider's verified translator (novita "Hola, mundo. Esta es una pequeña prueba." · mistral "Hola, mundo. Este es un pequeño test." · groq "Hola, mundo. Este es un pequeño prueba." · siliconflow "Hola, mundo. Esta es una pequeña prueba.") — validator ffprobe+frame-verified | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-provider-novita-translate-20260616-114032.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-mistral-translate-20260616-114039.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-groq-translate-20260616-114045.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-siliconflow-translate-20260616-114051.mp4 |
| bridge | `ProviderDiverseClients` | Library | Implemented | Wired | `bridge.go:492` — returns provider-diverse verified low-level LLM clients (one per distinct provider, strongest-first) for callers needing the client rather than a translator | Unit-covered (`pkg/bridge/bridge_test.go`, -race green) | **PASS (client path)** — builds via the OpenAI-compatible `clientBuild` (NOT the `ValidModels` whitelist), so it is NOT affected by the translator-path blocker; `go test ./pkg/bridge` → ok | PENDING (recording owed — distinct from the catalogue/best-model videos already CONFIRMED) |
| bridge | `ProviderDiverseModels` | Library | Implemented | Wired (backs the ProviderDiverse* set) | `bridge.go:455` — returns the provider-diverse verified `ModelInfo` list (one per distinct provider, strongest-first) underlying the ProviderDiverse clients/translators | Covered (`pkg/bridge/bridge_test.go`, -race green) | **PASS** (`go test ./pkg/bridge` → ok) | **CONFIRMED** — real-use video `/Volumes/T7/Downloads/Recordings/helix_translate-bridge-verified-catalogue-translate-20260616-105327.mp4` (Wave 1; `model-bridge list` renders the provider-diverse verified ModelInfo catalogue — novita/mistral/groq/siliconflow each with real base_url + score — then a real `invoke` returns "Le pont relie deux rives."; ffprobe 13.71s / 40 frames / 6 distinct frame-hashes; ≥6 frames readable per §11.4.107) |
| bridge | `BestClient` (single strongest low-level client) | Library | Implemented | Wired | `bridge.go:413` — returns the single strongest verified low-level LLM client for the task (client-level analogue of `BestTranslator`) | Covered (`pkg/bridge/bridge_test.go`, -race green) | **PASS** (`go test ./pkg/bridge` → ok) | N/A (internal client helper — surfaced via cmd/model-bridge / ProviderDiverse*) |

## pkg/version

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| version | App version | Library | Implemented | Wired | Singleton version constant (2.3.0) | Not-inventoried | Source-confirmed | N/A |
| version | Codebase hasher | Library | Implemented | Wired | SHA256 codebase versioning over dirs/extensions w/ exclude patterns | Not-inventoried | Source-confirmed | N/A |

## pkg/challenge_runner — anti-bluff test harness (uses `digital.vasic.challenges`)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| challenge_runner | Challenge orchestrator | Library | Implemented | Wired | Coordinates 18+ integration challenges, parallel exec + filtering | Not-inventoried | Source-confirmed | PENDING |
| challenge_runner | Build & format-detection challenges | Library | Implemented | Wired | Build-verification + FB2/EPUB/TXT/HTML/PDF/DOCX format-detection challenges | Not-inventoried | Source-confirmed | PENDING |
| challenge_runner | Infrastructure challenges | Library | Implemented | Wired | Event-bus + other infra challenges (run `go test`) | Not-inventoried | Source-confirmed | PENDING |
| challenge_runner | Provider challenges | Library | Implemented | Wired | Verify LLM provider factory/interface compile+load (build, not live calls) | Not-inventoried | Source-confirmed | PENDING |

---

# 3. Web Dashboards & Endpoints

## Web Dashboards

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| monitor.html | WebSocket connection status | Web | Implemented | Wired | Green/red pulse dot + Connected/Disconnected indicator | Not-inventoried | Source-confirmed | PENDING |
| monitor.html | Session/Client ID display | Web | Implemented | Wired | Static fields (demo-session-123, web-demo) | Not-inventoried | Source-confirmed; static demo values | PENDING |
| monitor.html | Overall progress bar + percent | Web | Implemented | Wired | Driven by WS translation_progress/step_completed events | Not-inventoried | Source-confirmed | PENDING |
| monitor.html | Current step / step message panel | Web | Implemented | Wired | Shows current step + message from WS | Not-inventoried | Source-confirmed | PENDING |
| monitor.html | Duration timer | Web | Implemented | Wired | Client-side elapsed clock | Not-inventoried | Source-confirmed | PENDING |
| monitor.html | Status badge | Web | Implemented | Wired | Pending/Running/Completed/Failed | Not-inventoried | Source-confirmed | PENDING |
| monitor.html | Event log | Web | Implemented | Wired | Scrolling list of received WS events | Not-inventoried | Source-confirmed | PENDING |
| monitor.html | Control Panel: Simulate Progress | Web | Implemented | Wired | Locally fakes & echoes progress events over WS (demo only) | Not-inventoried | Source-confirmed; demo-only | PENDING |
| monitor.html | Clear Log / Disconnect / Reconnect WS | Web | Implemented | Wired | Local log clear + WS connect/disconnect to ws://localhost:8090/ws | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Stats header (Active/Completed/Errors) | Web | Implemented | Wired | Aggregated counters from WS-tracked sessions | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Control panel: Connect/Disconnect/Clear | Web | Implemented | Wired | WS to ws://localhost:8090/ws?client_id=dashboard | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Monitor specific session | Web | Implemented | Wired | Input + button to filter UI to a given session_id | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Worker Information panel | Web | Implemented | Wired | Host:port, model, capacity from WS worker_info | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Current Session Progress panel | Web | Implemented | Wired | Progress bar, step, current/total items, duration, status | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Live progress chart (Chart.js) | Web | Implemented | Wired | Progress % over time, last 20 points | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Event log | Web | Implemented | Wired | Color-coded WS event stream | Not-inventoried | Source-confirmed | PENDING |
| enhanced-monitor.html | Session History table | Web | Implemented | Wired | Last 10 sessions: id, worker, status, duration, progress, start | Not-inventoried | Source-confirmed | PENDING |
| dashboard.html | Stats cards (Total/Completed/Running/Failed) | Web | Implemented | Wired | Polls GET /api/v1/translations every 5s | Not-inventoried | Source-confirmed | PENDING |
| dashboard.html | Active Sessions table | Web | Implemented | Wired | Lists sessions w/ provider, status, progress, duration | Not-inventoried | Source-confirmed | PENDING |
| dashboard.html | Session Details view | Web | Implemented | Wired | Progress bar, steps, generated files via GET /api/v1/translations/:id | Not-inventoried | Source-confirmed | PENDING |
| dashboard.html | Live progress via WebSocket | Web | Implemented | Wired | Connects ws://localhost:8080/ws?session_id=...; progress_update/step_* | Not-inventoried | Source-confirmed | PENDING |
| dashboard.html | New Translation modal | Web | Implemented | Wired | File input, provider selector, source/target lang, API-key/SSH/llama fields (UI only) | Not-inventoried | Source-confirmed | PENDING |
| dashboard.html | Start Translation (submit) | Web | Implemented | Wired | POST /api/v1/translations with provider config — triggers backend translation | Not-inventoried | Source-confirmed | PENDING |
| dashboard.html | Cancel session | Web | Implemented | Wired | DELETE /api/v1/translations/:id | Not-inventoried | Source-confirmed | PENDING |

## Web Dashboard backend handler — pkg/api/dashboard.go

Backend that serves the dashboard page (`/`, `/dashboard`, `/monitor`) and the translations endpoints. Was previously UNROUTED (returned 404); now wired so the dashboard genuinely translates for end users. Source: `pkg/api/dashboard.go`, commit f6ba5cc.

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api/dashboard.go | Dashboard page + translations endpoints | Web | Implemented | Wired (was Unwired/404 pre-fix) | Serves the dashboard HTML at `/`, `/dashboard`, `/monitor` and the `GET/POST /api/v1/translations` path; Start-Translation now runs a real translation through the wired UI path (no longer a 404) | Covered + **§11.4.115 RED→GREEN guard** `dashboard_test.go` (default GREEN asserts pages 200 + real translation start; `RED_MODE=1` reproduces the pre-fix 404) — `go test -race ./pkg/api` GREEN this session | **PASS (unit/integration)** — `dashboard_test.go` observed green this session; live run produced real translation "Dobro jutro, prijatelju." through the dashboard handler | **SKIP (§11.4.3/§11.4.52)** — autonomous browser+video capture needs the HelixQA web/video backend, off-limits to this session; tracked migration item owed (a `helixtranslate-web-dashboard-*` recording requires that backend). NOT a faked confirmation |

## REST API — pkg/api/server.go (standalone `Server`, gin)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api/server.go | GET /health | API | Implemented | Wired | Status + translator name | Not-inventoried | Source-confirmed | PENDING |
| api/server.go | POST /api/translate | API | Implemented | Wired | `translator.Translate(...)` | Not-inventoried | Source-confirmed | PENDING |
| api/server.go | GET /api/languages | API | Implemented | Wired | Static language list | Not-inventoried | Source-confirmed; static return | PENDING |
| api/server.go | GET /api/stats | API | Implemented | Wired | Returns hardcoded zeros (not live metrics) | Not-inventoried | Source-confirmed; hardcoded | PENDING |
| api/server.go | POST /api/upload | API | Implemented | Wired | Save uploaded file, detect format | Not-inventoried | Source-confirmed | PENDING |
| api/server.go | POST /api/batch | API | **Partial** | Wired | Returns batch_id `queued` only; performs NO translation | Not-inventoried | Source-confirmed; no real work | PENDING |

## REST API — pkg/api/handler.go (primary `Handler`, gin)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api/handler.go | GET /health, /, /api/v1/version, /api/v1/providers, /api/v1/stats | API | Implemented | Wired | Static/info/version/provider list/cache stats | Not-inventoried | **PASS — captured** (/health, /api/v1/version, /api/v1/providers real provider list returned). Re-confirmed Wave-3c against a live `cmd/server` TLS instance (port 9443): `/health`→`{"status":"healthy",…}`, `/api/v1/version`→codebase 2.3.0 + git_commit 29aa0e7, `/api/v1/providers`→real list (openai/anthropic/zhipu/deepseek/ollama/llamacpp); ffprobe 10.63s/790×560 yuv420p, ≥6 frames content-verified, health-frame ≠ providers-frame per §11.4.107 (re-confirmation, no double-count) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-api-endpoints_20260615-170755.mp4 ; Re-confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-rest-endpoints-wave3c-20260616.mp4 |
| api/handler.go | GET /ws | API | Implemented | Wired | WebSocket upgrade, register client to wsHub | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/translate | API | Implemented | Wired | Distributed or `TranslateWithProgress`; honors `target_lang` (FIXED a5e8866 — was hardcoded) | Not-inventoried | **PASS — real DeepSeek translation captured** (real Serbian output via API; target_lang=spanish FIXED run returns real Spanish). Re-confirmed Wave-3c against a live `cmd/server` TLS instance (port 9443): `curl -k -X POST .../api/v1/translate` `{"text":"The morning sun rose over the quiet village.","source_lang":"en","target_lang":"es","provider":"deepseek","model":"deepseek-chat"}` → real JSON `{"provider":"llm-deepseek","translated":"El sol de la mañana se alzó sobre el tranquilo pueblo.","stats":{"Translated":1,…},"session_id":…}`; ffprobe 4.24s/790×560 yuv420p, ≥6 frames at fps=2 content-verified, request-frame ≠ response-frame per §11.4.107 (re-confirmation, no double-count) | Confirmed:/Volumes/T7/Downloads/Recordings/helixtranslate-api-translate-deepseek_20260615-170716.mp4 ; Re-confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-rest-translate-wave3c-20260616.mp4 |
| api/handler.go | POST /api/v1/translate/fb2 | API | Implemented | Wired | Parse ebook + translate sections | Not-inventoried | **PASS — video-confirmed** (Wave 5: FB2 English input → POST → valid EPUB 2016 bytes, mimetype `application/epub+zip`, extracted chapter text real Serbian "Стари свјетионич стоје сам на каменитој обали…"; ffprobe 16 frames/8.0s/790×560 yuv420p, input-frame English ≠ output-frame Serbian per §11.4.107). NOTE: handler hardcoded ru→sr (`pkg/api/handler.go:84-85`), ignores requested langs — tracked fix. | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-rest-translate-fb2-wave5-20260616-173814.mp4 |
| api/handler.go | POST /api/v1/translate/batch | API | Implemented | Wired | Loops texts, translates each | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/convert/script | API | Implemented | Wired | Cyrillic↔Latin (deterministic) | Not-inventoried | **PASS — video-confirmed** (Wave 5: real JSON both directions — Cyrillic→Latin `{"converted":"Sistem treba prevesti","original":"Систем треба превести","target":"latin"}`, Latin→Cyrillic `{"converted":"Ово је тестни текст","original":"Ovo je testni tekst","target":"cyrillic"}`; ffprobe 6 frames/3.0s, original ≠ converted per §11.4.107) | Confirmed:/Volumes/T7/Downloads/Recordings/helix_translate-rest-convert-script-wave5-20260616-173814.mp4 |
| api/handler.go | GET /api/v1/status/:session_id | API | Implemented | Wired | Returns hardcoded `completed` (not a live session lookup) | Not-inventoried | Source-confirmed; hardcoded | PENDING |
| api/handler.go | GET /api/v1/languages | API | Implemented | Wired | Static list from language.GetSupportedLanguages() | Not-inventoried | Source-confirmed; static | PENDING |
| api/handler.go | POST /api/v1/translate/validate | API | Implemented | Wired | Validates request (lang/provider/length); no execution | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/preparation/analyze | API | **Stub** | Wired | File stat/count + emits events; stub analysis, no LLM | Not-inventoried | Source-confirmed; STUB | PENDING |
| api/handler.go | GET /api/v1/preparation/result/:session_id | API | **Stub** | Wired | Returns mock result | Not-inventoried | Source-confirmed; STUB | PENDING |
| api/handler.go | POST /api/v1/translate/ebook | API | **Stub** | Wired | Emits start event, returns mock; does NOT translate | Not-inventoried | Source-confirmed; STUB | PENDING |
| api/handler.go | POST /api/v1/translate/cancel/:session_id | API | Implemented | Wired | Emits cancel event, returns `cancelled` | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | GET /api/v1/distributed/status | API | Implemented | Wired | `dm.GetStatus()` | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/distributed/workers/discover | API | Implemented | Wired | SSH worker discovery | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/distributed/workers/:worker_id/pair | API | Implemented | Wired | Pair worker | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | DELETE /api/v1/distributed/workers/:worker_id/pair | API | Implemented | Wired | Unpair worker | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/distributed/translate | API | Implemented | Wired | `dm.TranslateDistributed(...)` | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/update/{upload,apply,rollback} | API | Implemented | Wired | Worker self-update (tar.gz upload/install/rollback) | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | GET /api/v1/monitoring/version/{metrics,alerts,health,dashboard} | API | Implemented | Wired | Version-sync monitoring reads | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/monitoring/version/drift-check | API | Implemented | Wired | Trigger drift check | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | GET /api/v1/monitoring/version/alerts/history | API | Implemented | Wired | Alert history | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/monitoring/version/alerts/:alert_id/acknowledge | API | Implemented | Wired | Ack alert | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/monitoring/version/alerts/channels/{email,webhook,slack} | API | Implemented | Wired | Add alert channels | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | GET /api/v1/monitoring/version/dashboard.html | Web | Implemented | Wired | Serve embedded dashboard HTML | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | POST /api/v1/auth/login, /auth/token | API | Implemented | Wired | JWT auth (if Security.EnableAuth) | Not-inventoried | Source-confirmed | PENDING |
| api/handler.go | GET /api/v1/profile | API | Implemented | Wired | Return claims (behind auth middleware) | Not-inventoried | Source-confirmed | PENDING |

## REST API — pkg/api/batch_handlers.go (under /api/v1)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api/batch_handlers.go | POST /api/v1/translate/string | API | Implemented | Wired | `NewLLMTranslator` + `Translate` | Not-inventoried | Source-confirmed | PENDING |
| api/batch_handlers.go | POST /api/v1/translate/directory | API | Implemented | Wired | Batch processor translates each file | Not-inventoried | Source-confirmed | PENDING |

## REST API — pkg/api/verifier_handlers.go (LLMsVerifier, under /api/v1)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api/verifier_handlers.go | GET /api/v1/verified-models | API | Implemented | Wired (only when LLMsVerifier enabled) | List verified models (LLMsVerifier API call). GAP (§11.4.6): LLMsVerifier is DISABLED in the current config, so the route is not mounted and returns HTTP 404. Not a defect — feature-disabled-by-config; needs a verifier-enabled config to confirm | Not-inventoried | **GAP — 404 captured** (verifier disabled in config; route not mounted) | PENDING (verifier-enabled config needed); evidence:/Volumes/T7/Downloads/Recordings/helixtranslate-api-endpoints_20260615-170755.mp4 |
| api/verifier_handlers.go | GET /api/v1/verified-models/:id | API | Implemented | Wired | Get one verified model | Not-inventoried | Source-confirmed | PENDING |
| api/verifier_handlers.go | GET /api/v1/verification-status | API | Implemented | Wired | Ping LLMsVerifier | Not-inventoried | Source-confirmed | PENDING |
| api/verifier_handlers.go | POST /api/v1/verification/refresh | API | Implemented | Wired | Refresh scores | Not-inventoried | Source-confirmed | PENDING |
| api/verifier_handlers.go | GET /api/v1/providers/verified | API | Implemented | Wired | Aggregated verified providers | Not-inventoried | Source-confirmed | PENDING |
| api/verifier_handlers.go | POST /api/v1/translate-with-verification | API | **Stub** | Wired | Selects/verifies a model, returns metadata + text preview; does NOT translate | Not-inventoried | Source-confirmed; STUB | PENDING |

## REST API — cmd/api-server/main.go (gRPC-proxying REST, :8080)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| api-server | POST /api/v1/translations | API | Implemented | Wired | Proxies grpcClient.StartTranslation | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET /api/v1/translations/:session_id | API | Implemented | Wired | Get status (proxy) | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET /api/v1/translations | API | Implemented | Wired | List (proxy) | Not-inventoried | Source-confirmed | PENDING |
| api-server | DELETE /api/v1/translations/:session_id | API | Implemented | Wired | Cancel (proxy) | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET /api/v1/translations/:session_id/stream | API | Implemented | Wired | WS proxy of gRPC StreamTranslationProgress | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET /api/v1/providers | API | Implemented | Wired | Get providers (proxy) | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET /api/v1/health | API | Implemented | Wired | Derived from gRPC conn state | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET /api/v1/metrics | API | Implemented | Wired | Returns static/zero metrics (not live) | Not-inventoried | Source-confirmed; static | PENDING |
| api-server | GET /ws | Web | Implemented | Wired | WS upgrade, register to wsHub (requires session_id) | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET /static/* | Web | Implemented | Wired | Static file server (./web/static) | Not-inventoried | Source-confirmed | PENDING |
| api-server | GET / | Web | Implemented | Wired | Render dashboard.html template | Not-inventoried | Source-confirmed | PENDING |

## WebSocket — hub & fan-out

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| websocket/hub.go | /ws endpoint (package mux) | Infra | Implemented | Wired | WS upgrade; reads client_id + session_id | Not-inventoried | Source-confirmed | PENDING |
| websocket/hub.go | Event fan-out | Infra | Implemented | Wired | Subscribes to ALL bus events; marshals events.Event JSON; per-session filter; Broadcast() | Not-inventoried | Source-confirmed | PENDING |

## gRPC — pkg/grpc/server.go (service `TranslationService`)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| grpc | StartTranslation | gRPC | Implemented | Wired | Reserves session, spawns runTranslation → translator.Translate(...) | Not-inventoried | Source-confirmed | PENDING |
| grpc | GetTranslationStatus | gRPC | Implemented | Wired | Snapshot session + translator.GetStatus | Not-inventoried | Source-confirmed | PENDING |
| grpc | ListTranslations | gRPC | Implemented | Wired | Iterate sessions | Not-inventoried | Source-confirmed | PENDING |
| grpc | CancelTranslation | gRPC | Implemented | Wired | Cancel ctx + translator.Cancel | Not-inventoried | Source-confirmed | PENDING |
| grpc | StreamTranslationProgress | gRPC | Implemented | Wired | Stream session progress events | Not-inventoried | Source-confirmed | PENDING |
| grpc | GetProviders | gRPC | Implemented | Wired | Static ProviderRegistry (openai, anthropic, ssh) | Not-inventoried | Source-confirmed; static | PENDING |
| grpc | SubscribeEvents | gRPC | Implemented | Wired | Stream filtered SystemEvents from bus | Not-inventoried | Source-confirmed | PENDING |

## CORS / Middleware / Auth (Infra)

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| cmd/server | corsMiddleware | Infra | Implemented | Wired | Driven by cfg.Security.CORSOrigins; reflects allowlisted Origin + Allow-Credentials; wildcard `*` no-credentials; OPTIONS→204 | Not-inventoried | Source-confirmed | N/A |
| cmd/server | rateLimitMiddleware | Infra | Implemented | Wired | Per-ClientIP via security.RateLimiter; 429 on exceed | Not-inventoried | Source-confirmed | N/A |
| pkg/api/handler.go | authMiddleware | Infra | Implemented | Wired | Bearer JWT validation; applied to protected subgroup, gated by Security.EnableAuth | Not-inventoried | Source-confirmed | N/A |
| pkg/api/server.go | authMiddleware (standalone) | Infra | Implemented | Wired | X-API-Key header == config.Security.APIKey, if Security.RequireAuth | Not-inventoried | Source-confirmed | N/A |
| pkg/grpc/server.go | message-size limits | Infra | Implemented | Wired | 4 MB recv/send msg-size limits; no interceptors/auth/rate-limit registered | Not-inventoried | Source-confirmed | N/A |

---

# 4. Owned Submodules

## challenges/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| challenges | Challenge core engine | Submodule | Implemented | Wired | Challenge interface, Config, Result, BaseChallenge, ProgressReporter (liveness) | Not-inventoried | Source-confirmed | PENDING |
| challenges | Registry + dependency ordering | Submodule | Implemented | Wired | Challenge registration, Kahn topological sort by deps | Not-inventoried | Source-confirmed | N/A |
| challenges | Runner / execution engine | Submodule | Implemented | Wired | Sequential/parallel/pipeline execution, liveness/stall monitoring | Not-inventoried | Source-confirmed | PENDING |
| challenges | Assertion engine | Submodule | Implemented | Wired | 16 built-in evaluators (not_empty, contains, min_count, max_latency…) + custom | Not-inventoried | Source-confirmed | N/A |
| challenges | Report generation | Submodule | Implemented | Wired | Markdown / JSON / HTML reporters | Not-inventoried | Source-confirmed | N/A |
| challenges | Live monitoring | Submodule | Implemented | Wired | WebSocket dashboard, progress liveness | Not-inventoried | Source-confirmed | PENDING |
| challenges | Metrics | Submodule | Implemented | Wired | Prometheus-compatible challenge metrics | Not-inventoried | Source-confirmed | N/A |
| challenges | HTTP client | Submodule | Implemented | Wired | Generic REST client with JWT auth, functional options | Not-inventoried | Source-confirmed | N/A |
| challenges | Challenge bank loader | Submodule | Implemented | Wired | Load challenge definitions from JSON/YAML | Not-inventoried | Source-confirmed | N/A |
| challenges | Plugin system | Submodule | Implemented | Wired | Custom challenge types + assertion plugins | Not-inventoried | Source-confirmed | N/A |
| challenges | Infra bridge | Submodule | Implemented | Wired | Bridge to digital.vasic.containers for test infra | Not-inventoried | Source-confirmed | PENDING |
| challenges | User-flow automation | Submodule | Implemented | Wired | 8 adapter interfaces / 21 impls (Playwright, ADB, Tauri, HTTP, gRPC, WS, Gradle, Cargo, npm), 19 templates, 12 evaluators | Not-inventoried | Source-confirmed; LLM-in-evaluators UNCONFIRMED | PENDING |
| challenges | userflow-runner CLI | Submodule | Implemented | Wired | CLI runner for user-flow challenges | Not-inventoried | Source-confirmed | PENDING |
| challenges | panoptic / i18n / env / logging | Submodule | Implemented | Wired | Observation, i18n, env-var redaction, structured logging | Not-inventoried | Source-confirmed | N/A |
| challenges | anti_bluff lib | Submodule | Implemented | Wired | Shell anti-bluff helper library | Not-inventoried | Source-confirmed | N/A |

## containers/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| containers | boot manager | Submodule | Implemented | Wired | On-demand infra boot/start orchestration | Not-inventoried | Source-confirmed | PENDING |
| containers | compose | Submodule | Implemented | Wired | docker/podman compose file orchestration | Not-inventoried | Source-confirmed | N/A |
| containers | health checking | Submodule | Implemented | Wired | TCP/HTTP health-check primitives | Not-inventoried | Source-confirmed | N/A |
| containers | runtime auto-detect | Submodule | Implemented | Wired | Docker/Podman/Kubernetes runtime detection | Not-inventoried | Source-confirmed | N/A |
| containers | endpoint | Submodule | Implemented | Wired | ServiceEndpoint builder | Not-inventoried | Source-confirmed | N/A |
| containers | lifecycle | Submodule | Implemented | Wired | Container lifecycle management | Not-inventoried | Source-confirmed | PENDING |
| containers | service discovery / registry | Submodule | Implemented | Wired | Service discovery + registry | Not-inventoried | Source-confirmed | N/A |
| containers | orchestrator / scheduler | Submodule | Implemented | Wired | Orchestration + scheduling | Not-inventoried | Source-confirmed | PENDING |
| containers | emulator suite | Submodule | Implemented | Wired | Android/device emulator boot/matrix/cleanup/canary | Not-inventoried | Source-confirmed | PENDING |
| containers | genymotion | Submodule | Implemented | Wired | Genymotion device support | Not-inventoried | Source-confirmed | PENDING |
| containers | vm-matrix | Submodule | Implemented | Wired | VM matrix orchestration | Not-inventoried | Source-confirmed | PENDING |
| containers | distributed build/test | Submodule | Implemented | Wired | Distributed build + test commands | Not-inventoried | Source-confirmed | PENDING |
| containers | crossbuild | Submodule | Implemented | Wired | Cross-platform build | Not-inventoried | Source-confirmed | PENDING |
| containers | ctop | Submodule | Implemented | Wired | Container top / monitoring TUI | Not-inventoried | Source-confirmed | PENDING |
| containers | network / volume | Submodule | Implemented | Wired | Network + volume primitives | Not-inventoried | Source-confirmed | N/A |
| containers | supporting primitives | Submodule | Implemented | Wired | cache/metrics/monitor/event/policy/remote/lazyservice/brokertest/envconfig | Not-inventoried | Source-confirmed | N/A |

## doc_processor/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| doc_processor | Document loader | Submodule | Implemented | Wired | Loads Markdown / YAML / HTML docs (10 MB cap) | Not-inventoried | Source-confirmed | N/A |
| doc_processor | Feature-map builder | Submodule | Implemented | Wired | Builds structured FeatureMap; heuristic extraction + LLM Enrich | Not-inventoried | Source-confirmed | PENDING |
| doc_processor | LLMAgent interface | Submodule | Implemented | Wired | Summarize/ExtractFeatures/ClassifyFeature/InferScreens/GenerateTestSteps | Not-inventoried | Source-confirmed | PENDING |
| doc_processor | Coverage tracker | Submodule | Implemented | Wired | Verification coverage tracking by platform/category | Not-inventoried | Source-confirmed | N/A |
| doc_processor | DocGraph | Submodule | Implemented | Wired | Doc node/edge graph | Not-inventoried | Source-confirmed | N/A |
| doc_processor | Config / i18n | Submodule | Implemented | Wired | Configuration + i18n | Not-inventoried | Source-confirmed | N/A |
| doc_processor | docprocessor CLI | Submodule | Implemented | Wired | CLI running feature extraction against a docs dir | Not-inventoried | Source-confirmed | PENDING |
| doc_processor | archdoc (internal) | Submodule | Implemented | Wired | Architecture-doc helper (LLM-use UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |

## docs_chain/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| docs_chain | `sync` command | Submodule | Implemented | Wired | Propagate changes through chain (regenerate + re-export atomically) | Not-inventoried | Source-confirmed; used to produce this doc's exports | PENDING |
| docs_chain | `verify` command | Submodule | Implemented | Wired | Deterministic CI/pre-build gate (chain in sync) | Not-inventoried | Source-confirmed | PENDING |
| docs_chain | `doctor` command | Submodule | Implemented | Wired | Diagnostics | Not-inventoried | Source-confirmed | PENDING |
| docs_chain | `graph` command | Submodule | Implemented | Wired | Show DAG / topo order | Not-inventoried | Source-confirmed | PENDING |
| docs_chain | `watch` daemon | Submodule | Implemented | Wired | fsnotify-driven continuous sync | Not-inventoried | Source-confirmed | PENDING |
| docs_chain | `exec:` transforms | Submodule | Implemented | Wired | Shell-out transforms (pandoc md→html/docx, weasyprint html→pdf) | Not-inventoried | Source-confirmed | PENDING |
| docs_chain | content-hash engine | Submodule | Implemented | Wired | Content-hash incremental recompute; roster/corpus fingerprint (§11.4.86) | Not-inventoried | Source-confirmed | N/A |
| docs_chain | DAG + recompute | Submodule | Implemented | Wired | Node/Edge/Graph, cycle detection, Kahn topo, early-cutoff, sync-edge conflict | Not-inventoried | Source-confirmed | N/A |
| docs_chain | adapters | Submodule | Implemented | Wired | FileAdapter, SQLiteAdapter (pure-Go modernc), DERIVED pandoc/weasyprint (ToolAbsentError) | Not-inventoried | Source-confirmed | N/A |
| docs_chain | orchestrator | Submodule | Implemented | Wired | Atomic run: staged writes, SQLite-txn commit + rollback | Not-inventoried | Source-confirmed | N/A |
| docs_chain | DB builtins | Submodule | Implemented | Wired | Bidirectional md-to-sqlite / sqlite-to-md (row-level drift); colorize-html | Not-inventoried | Source-confirmed | N/A |
| docs_chain | config + state | Submodule | Implemented | Wired | YAML multi-context loader; state.json | Not-inventoried | Source-confirmed | N/A |

## llm_orchestrator/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| llm_orchestrator | Agent interface + AgentPool | Submodule | Implemented | Wired | Thread-safe pool, capability matching, acquire/release, health | Not-inventoried | Source-confirmed | PENDING |
| llm_orchestrator | CLI agent adapters | Submodule | Implemented | Wired | OpenCode, ClaudeCode, Gemini, Junie, QwenCode adapters (BaseAdapter) | Not-inventoried | Source-confirmed | PENDING |
| llm_orchestrator | Transport / protocol | Submodule | Implemented | Wired | PipeTransport (JSON-line stdin/stdout) + FileTransport, protocol types | Not-inventoried | Source-confirmed | N/A |
| llm_orchestrator | Response parser | Submodule | Implemented | Wired | Parses agent JSON-line responses | Not-inventoried | Source-confirmed | N/A |
| llm_orchestrator | HealthMonitor + CircuitBreaker | Submodule | Implemented | Wired | Health tracking + fault tolerance over agents | Not-inventoried | Source-confirmed | N/A |
| llm_orchestrator | Config / i18n | Submodule | Implemented | Wired | Pool/agent config; i18n | Not-inventoried | Source-confirmed | N/A |
| llm_orchestrator | orchestrator CLI | Submodule | Implemented | Wired | CLI entry for orchestration | Not-inventoried | Source-confirmed | PENDING |

## llm_provider/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| llm_provider | LLMProvider interface | Submodule | Implemented | Wired | Complete/CompleteStream/HealthCheck/GetCapabilities/ValidateConfig | Not-inventoried | Source-confirmed | N/A |
| llm_provider | Circuit breaker | Submodule | Implemented | Wired | closed/open/half-open fault tolerance | Not-inventoried | Source-confirmed | N/A |
| llm_provider | Health monitor | Submodule | Implemented | Wired | healthy/degraded/unhealthy/unknown | Not-inventoried | Source-confirmed | N/A |
| llm_provider | Retry logic | Submodule | Implemented | Wired | Exponential backoff + jitter, HTTP-status detection | Not-inventoried | Source-confirmed | N/A |
| llm_provider | Model discovery | Submodule | Implemented | Wired | Provider/model discovery (HTTP/endpoint-based) | Not-inventoried | Source-confirmed | PENDING |
| llm_provider | API-key management | Submodule | Implemented | Wired | API key handling | Not-inventoried | Source-confirmed | N/A |
| llm_provider | HTTP client + models + i18n | Submodule | Implemented | Wired | Shared HTTP client, request/response models, i18n | Not-inventoried | Source-confirmed | N/A |
| llm_provider | Provider implementations (43) | Submodule | Implemented | Wired | ai21, anthropic, cerebras, chutes, claude, cloudflare, codestral, cohere, deepseek, fireworks, gemini, generic, githubmodels, groq, huggingface, hyperbolic, junie, kilo, kimi, mistral, modal, nia, nlpcloud, novita, nvidia, ollama, openai, openrouter, perplexity, publicai, qwen, replicate, sambanova, sarvam, siliconflow, together, upstage, venice, vulavula, xai, zai, zen, zhipu (source `llm_provider/pkg/providers/<name>/`) | Not-inventoried | **PASS — `fireworks` provider real EN→ES translation captured** (the `llm_provider/pkg/providers/fireworks/fireworks.go` client, reached via unified-translator `-provider openai -base-url https://api.fireworks.ai/inference/v1 -model accounts/fireworks/models/llama-v3p1-70b-instruct`: "The morning sun rose over the quiet village. An old fisherman walked along the shore…" → "Al amanecer, el sol ascendió sobre el tranquilo pueblo. Un anciano pescador paseaba por la orilla…" — distinct verbs (ascendió/paseaba) from Hyperbolic's output (salía/caminaba), proving a genuine per-provider model; Wave-3c, asciinema→agg→ffmpeg, ffprobe 4.54s/790×560 yuv420p + 9 extracted frames at fps=2, ≥6 frames content-verified, start-frame (input only) ≠ output-frame (Spanish) per §11.4.107). **Wave-3d added 2 net-new: `nvidia` + `openrouter`** — `nvidia` (reached via unified-translator `-provider openai -base-url https://integrate.api.nvidia.com/v1 -model meta/llama-3.1-8b-instruct`): "At dawn, the sun rose over the quiet village. An old fisherman walked along the shore." → "Al amanecer, el sol se alzó sobre el tranquilo pueblo. Un anciano pescador **caminaba** a lo largo de la orilla." (`helix_translate-provider-nvidia-translate-wave3d-20260616.mp4`, ffprobe 6.5s/790×560 yuv420p + 13 frames at fps=2); `openrouter` (`-base-url https://openrouter.ai/api/v1 -model meta-llama/llama-3.1-8b-instruct`): same EN source → "Al amanecer, el sol se alzó sobre el tranquilo pueblo. Un anciano pescador **paseaba** a lo largo de la orilla." (`helix_translate-provider-openrouter-translate-wave3d-20260616.mp4`, ffprobe 5.5s/790×560 yuv420p + 11 frames at fps=2) — distinct verbs (caminaba/paseaba) prove genuine per-provider models, each start-frame (English only) ≠ output-frame (Spanish) per §11.4.107, ≥6 frames content-verified. The remaining 40 provider dirs remain Source-confirmed, PENDING a per-provider recording. | Confirmed (fireworks/nvidia/openrouter):/Volumes/T7/Downloads/Recordings/helix_translate-provider-fireworks-translate-wave3c-20260616.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-nvidia-translate-wave3d-20260616.mp4 · /Volumes/T7/Downloads/Recordings/helix_translate-provider-openrouter-translate-wave3d-20260616.mp4 |

## llms_verifier/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| llms_verifier | Model verification | Submodule | Implemented | Wired | "Do you see my code?" + existence/responsiveness/latency/streaming/function-calling/vision/embeddings tests | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | Provider adapters | Submodule | Implemented | Wired | OpenAI, Anthropic, Cohere, Groq, Together, Mistral, xAI, Replicate, DeepSeek, Cerebras, Cloudflare, SiliconFlow (12+) | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | Scoring engine | Submodule | Implemented | Wired | Quality scoring system (LLM-use UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | Capabilities detection | Submodule | Implemented | Wired | Per-model capability discovery | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | Failover / health monitoring | Submodule | Implemented | Wired | Real-time monitoring + intelligent failover (probes call LLM) | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | AI analytics / recommendations | Submodule | Implemented | Wired | AI-powered insights, trend analysis, model recommendations | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | RAG / context mgmt | Submodule | Implemented | Wired | Vector DB semantic search, LLM-powered summarization | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | LLMOps / self-improve / benchmark | Submodule | Implemented | Wired | LLM operations, self-improvement loop, benchmarking | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | Messaging | Submodule | Implemented | Wired | Notification/messaging integration (Splunk/DataDog/etc.) | Not-inventoried | Source-confirmed | N/A |
| llms_verifier | suffix / verified export | Submodule | Implemented | Wired | Mandatory branding suffix + verified-only config export | Not-inventoried | Source-confirmed | N/A |
| llms_verifier | Auth / security / API / API-keys | Submodule | Implemented | Wired | LDAP/SSO/SAML/OIDC, SQLCipher, REST API, key mgmt | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | Multi-platform clients | Submodule | Implemented | Wired | CLI, TUI, Web, Desktop, Mobile (UI; LLM-backed) | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | helixqa / cliagents / crush / opencode | Submodule | Implemented | Wired | Integration packages (LLM-use UNCONFIRMED) | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | scheduler / partners / bigdata / multimodal / performance | Submodule | Implemented | Wired | Scheduling, partners, big-data, multimodal, perf (multimodal likely LLM) | Not-inventoried | Source-confirmed | PENDING |
| llms_verifier | main CLI + ACP | Submodule | Implemented | Wired | Main CLI; ACP (Agent Client Protocol) | Not-inventoried | Source-confirmed | PENDING |

## security/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| security | Content guardrails | Submodule | Implemented | Wired | Rule pipeline (MaxLength, ForbiddenPatterns, RequireFormat) w/ severities | Not-inventoried | Source-confirmed | N/A |
| security | PII detection + redaction | Submodule | Implemented | Wired | Email/Phone/SSN/CreditCard(Luhn)/IPv4 detection; Mask/Hash/Remove | Not-inventoried | Source-confirmed | N/A |
| security | Content filtering | Submodule | Implemented | Wired | Composable chain: Length/Pattern/Keyword filters | Not-inventoried | Source-confirmed | N/A |
| security | Policy enforcement | Submodule | Implemented | Wired | Named policies, 8 condition operators, Allow/Deny/Audit + audit trail | Not-inventoried | Source-confirmed | N/A |
| security | Vulnerability scanner | Submodule | Implemented | Wired | Findings/reports, severity filtering + merging | Not-inventoried | Source-confirmed | N/A |
| security | HTTP security headers | Submodule | Implemented | Wired | Middleware: CSP/HSTS/X-Frame-Options + 7 standard headers | Not-inventoried | Source-confirmed | N/A |
| security | Secure storage | Submodule | Implemented | Wired | AES-256-GCM encrypted file KV store, Argon2id KDF | Not-inventoried | Source-confirmed | N/A |
| security | Privilege-escalation scanner | Submodule | Implemented | Wired | Probes /proc/self/status, capability bitmasks, namespace cgroups | Not-inventoried | Source-confirmed | N/A |
| security | SSRF denial | Submodule | Implemented | Wired | Denies SSRF on caller's outbound HTTP client | Not-inventoried | Source-confirmed | N/A |
| security | E2EE | Submodule | Implemented | Wired | End-to-end encryption | Not-inventoried | Source-confirmed | N/A |
| security | Attestation / GPU attestation | Submodule | Implemented | Wired | Attestation primitives | Not-inventoried | Source-confirmed | N/A |
| security | i18n | Submodule | Implemented | Wired | i18n | Not-inventoried | Source-confirmed | N/A |

## vision_engine/

| Component | Feature | Category | Implementation | Wiring | Real-use | Tests | Validation | Video-confirmation |
|---|---|---|---|---|---|---|---|---|
| vision_engine | Analyzer interface | Submodule | Implemented | Wired | UI-element/text-region/visual-issue detection, screen analysis/diff, key frames | Not-inventoried | Source-confirmed | PENDING |
| vision_engine | LLM vision adapters | Submodule | Implemented | Wired | OpenAI GPT-4o, Anthropic Claude, Gemini 2.0 Flash, Qwen VL, Kimi, StepFun, Ollama + Fallback chain | Not-inventoried | Source-confirmed | PENDING |
| vision_engine | NavigationGraph | Submodule | Implemented | Wired | Directed graph of screens + transitions | Not-inventoried | Source-confirmed | N/A |
| vision_engine | Remote Ollama deployment | Submodule | Implemented | Wired | SSH auto-install, model pull, lifecycle; GPU/CPU/RAM detection; distributed llama.cpp RPC | Not-inventoried | Source-confirmed | PENDING |
| vision_engine | OpenCV (stub + build-tag real) | Submodule | Partial | Wired | OpenCV-based CV — stub by default, real impl behind build tag | Not-inventoried | Source-confirmed; real impl gated behind build tag | PENDING |
| vision_engine | Config / i18n | Submodule | Implemented | Wired | Configuration (HELIX_VISION_PROVIDER etc.), i18n | Not-inventoried | Source-confirmed | N/A |

---

## Honest gaps & caveats (§11.4.6)

1. **Video coverage is 30 feature rows.** 30 features are confirmed by real, ffprobe- and content-verified recordings (Wave 2 of 2026-06-16 added 5 net-new — DOCX output, PDF output, HTML→Markdown conversion, `ProviderDiverseTranslators`, `EnsembleFactory` — plus an additional re-confirming clip on the already-confirmed gRPC TranslationService row, no double-count; Wave 1 added 2 — bridge-ensemble best-model selection + bridge provider-diverse catalogue; see the Anti-bluff note + the Coverage-summary `Video-confirmed` row for the authoritative count and full list — reconciled with both, §11.4.60). DOCX output and PDF output, formerly real-artifact-only, are now video-confirmed by Wave-2 recordings. Every other runtime/user-visible feature is `PENDING` a recording or `N/A`. This is stated, not hidden.
1a. **DOCX output is Implemented (Rev 3, commit 87cd2be)** — pure-Go OOXML writer; a real run produced a `Microsoft Word 2007+` `.docx`; **now video-confirmed (Wave 2)** by `helix_translate-docx-output-20260616-112943.mp4` (openai/gpt-4, real Spanish in the `.docx`). **PDF output is Implemented (Rev 4, commit fb265e7)** — `pkg/ebook/pdf_writer.go` (Book→HTML5→weasyprint→Cyrillic-faithful PDF), wired into unified-translator (`.pdf` output, main.go:1160); `pkg/ebook` PDF tests green; **now video-confirmed (Wave 2)** by `helix_translate-pdf-output-20260616-113057.mp4` (openai/gpt-4, valid PDF with real Spanish). Output formats are EPUB/FB2/TXT/HTML/MD/DOCX/PDF.
1b. **`-multipass` ≠ `-verify`.** The unified-translator `-verify` flag runs the CLI's own per-step output check; the separate `-multipass` flag (d53e085) invokes the `pkg/verification` multi-pass polisher engine — now wired and video-confirmed.
1c. **`cmd/translator`, `cmd/translate-ssh`, `cmd/ebook-translator` are now Obsolete (→ Fixed.md), Rev 7** — RECONCILED with the per-binary table rows (lines ~136/218/233) which mark all three `Obsolete (→ Fixed.md)` with Reason=`feature-removed`. The local-runtime + SSH-local translation paths were REMOVED in bridge phase-2 R-2..R-4 (operator-approved 2026-06-15 D1/D2 + R-4 "keep distributed API, remove SSH-local"); the default path now sources the LLMsVerifier bridge (`pkg/bridge`). These are therefore NO LONGER open STUB / OPERATOR-BLOCKED items — the earlier "STUB / OPERATOR-BLOCKED" wording was stale and is corrected here. The KEPT distributed-API translation path is NOT Obsolete.
1e. **`GET /api/v1/verified-models` returns 404** — LLMsVerifier is disabled in the current config so the route is not mounted (feature-disabled-by-config, not a defect).
1f. **`cmd/cli` title path-leak FIXED** (d53e085) + **write-safety FIXED** (87cd2be, atomic write/no-clobber, guard cmd/cli/no_partial_output_test.go).
2. **Tests = `Not-inventoried`** for all rows: the raw inventory did not perform per-feature test attribution. The repo carries extensive `_test.go` + `test/` suites, but mapping each to a feature is future work — recorded as honest unknown, never guessed Covered/None.
3. **4 STUB endpoints** return mock data and do NOT do the named work: `POST /api/v1/translate/ebook`, `POST /api/v1/preparation/analyze`, `GET /api/v1/preparation/result/:id`, `POST /api/v1/translate-with-verification`.
4. **1 Partial endpoint:** standalone `pkg/api/server.go POST /api/batch` returns a `queued` batch_id only and performs no translation.
5. **Static/hardcoded returns** (not stubs, but not live): `GET /api/stats` (zeros), `GET /api/v1/status/:id` (hardcoded `completed`), `GET /api/v1/metrics` (static), monitor-server `GET /api/v1/status` (static `monitoring_active`), several static language/provider lists. Marked Implemented with the static behaviour stated in Real-use.
6. **vision_engine OpenCV** is `Partial` — stub by default, real CV only behind a build tag.
7. **UNCONFIRMED LLM determinations** carried verbatim from the inventory for thin OpenAI-compatible providers, several `llms_verifier` packages, `challenges/userflow` evaluators, and `doc_processor/archdoc` — noted inline rather than asserted.
8. **✅ Wave-1 DISCOVERED BLOCKER — RESOLVED (Wave 2, §11.4.4 / §11.4.107 / §11.4.108 / §11.4.138).** The default bridge TRANSLATOR-path runtime failure described below is FIXED: the Wave-2 recordings (DOCX output, PDF output, gRPC translate, and the 4 per-provider unified clips novita/mistral/groq/siliconflow) are real-use videos validator-verified (ffprobe+frame) showing the verified-bridge translator path producing real target-language output — the exact path that failed in Wave 1 now works, which is why those rows flip to CONFIRMED this wave. The historical forensic record of the blocker is preserved below for §11.4.6 / §11.4.138 traceability. **[HISTORICAL — Wave-1 finding, now fixed]** While recording Wave 1 (DOCX/PDF/gRPC), a real run of `unified-translator` (and, by source, the gRPC backend `pkg/grpc/core_translator.go` which uses `BestTranslatorFunc`) FAILS at translate time with: `bridge: no verified translator available: model 'Sao10K/L3-8B-Stheno-v3.2' is not valid for provider 'novita'. Valid models: [meta-llama/llama-3.1-70b-instruct mistralai/mixtral-8x7b-instruct]`. **Root cause (FACT, §11.4.6):** `VerifiedFactory.CreateTranslatorWithFallback` / `CreateTranslator` build the verified translator via `NewLLMTranslatorWithConfig`, which re-validates the model against the STALE hardcoded `ValidModels` whitelist (`pkg/translator/llm/llm.go:227-241`); a genuinely LLMsVerifier-verified model whose ID is not in that whitelist is rejected. The `BestModel`/`invoke` path (used by `model-bridge invoke` + Wave-1 videos) correctly BYPASSES this whitelist (its own code comment, `bridge.go:250-258`, documents exactly why a verified model must not be gated on the whitelist) — and so does NOT exhibit the bug. **Impact:** unified-translator (all outputs: TXT/HTML/EPUB/FB2/MD/**DOCX**/**PDF**) + gRPC translate are non-functional via the default bridge path until fixed. This BLOCKS Wave-1 videos #2 (DOCX), #3 (PDF), #4 (gRPC), and the per-provider unified-path; recording any of them as "working" would be a §11.4.107 bluff, so they are honest SKIPs this wave. **Fix direction (not yet applied — needs its own §11.4.115 RED→GREEN + §11.4.142 independent review + §11.4.145 impact research + §11.4.40 full-suite retest):** route `VerifiedFactory`'s verified-translator construction through the `ResolvedProvider` resolver (FactoryProvider="openai" + BaseURL) the way `bridge.realClientBuild` already does, bypassing the `ValidModels` whitelist for already-verified models. Tracked as a Wave-2 / dedicated fix item (severity HIGH — release-critical default path).
