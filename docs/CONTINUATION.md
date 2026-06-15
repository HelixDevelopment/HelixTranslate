# CONTINUATION — HelixTranslate session-resumption file

**Revision:** 55
**Last modified:** 2026-06-15T07:00:00Z

<!-- session 2026-06-14v: cross-submodule bug-hunt campaign — 48 real bugs (21 submodule + 27 main) -->
<!-- OPERATOR AWAY until morning 2026-06-15: autonomous; priority = most-stable build; release tag DEFERRED (zero-risk, needs full §11.4.40). Full ./... sweep GREEN at c56363c. -->

### Session 2026-06-14v — cross-submodule bug-hunt campaign (§11.4.28 equal-codebase)

Parallel subagent-driven (§11.4.70/§11.4.103) hunt across owned submodules — each bug
reproduce-first RED (§11.4.115) → fix → GREEN → mutation-proven (revert→FAIL/race→restore→PASS),
verified by the conductor against repo state before commit (§11.4.142), committed in each
submodule's own repo (raw git, FF-only, no force §11.4.113, multi-mirror §2.1). **18 genuine
mutation-proven bugs fixed + pushed; vision_engine's was independently fixed upstream by a
parallel session (64b7d08) so it was integrated, NOT double-fixed.** Host API rate-limit
(prior session's 6-subagent killer) had recovered — proven by a probe wave that all survived.

| Submodule | bug(s) fixed | commit (after FF) |
|---|---|---|
| llm_provider | health monitor recovered on cumulative not consecutive successes | f58559f |
| llm_provider | claude stream fake-success-on-error; openrouter 4096B SSE truncation; zen dropped system-prompt; zen sessionID data race | 325a469 |
| doc_processor | `truncate` byte-slice corrupts multibyte (Cyrillic) Feature.Description | eec2d3e |
| doc_processor | i18n `interpolate` re-substitutes placeholders, order-dependent Cyrillic corruption | fc4fed8 |
| llm_orchestrator | `SimpleAgentPool.Release` returns untracked agents (double-handout + cross-pool leak) | 1fac8b1 |
| llm_orchestrator | `RoundRobinSelector` never selects a lazy pool with spare build capacity (dead lazy-build path) | 4e08b93 |
| llms_verifier | HTTP/3-disabled nil-panic + ignored maxScore ceiling | 6e5ab147 |
| llms_verifier | nil circuit-breaker deref panic ×3 sites + concurrent map write under RLock (data race) | 6f19503b |
| docs_chain | corrupt SQLite node silently read as empty DB (SSoT-destroying) | 47275bc |
| docs_chain | `verify` masked multi-level staleness — CI gate passed a stale artifact | ad8b9ff |
| challenges | go/cargo test-JSON parsers silently drop failures after >64KB line (PASS-bluff in the anti-bluff harness) | e4ae4ef |
| containers | boot summary counter corruption (Started=-1) + health check ignored ctx cancellation | f3bfbc2 |
| **MAIN** pkg/distributed | slice-header data race (remoteInstances) + BatchProcessor processFn-under-lock deadlock + version_manager backups fatal concurrent-map-write + alerts slice race — all `-race`-proven | 3a5b435 |
| **MAIN** internal/verifier | GetPreferences dropped ALL models without a prior RefreshScores (used uncached score vs threshold); empty verified-models result never cached (TTL defeated) | (wave 7) |
| **MAIN** pkg/ebook + pkg/fb2 | TXT >64KiB single-line total content loss; EPUB multiline `<head>` leaks `<title>` text into chapter; FB2 stanza title/subtitle dropped; FB2 cite/epigraph nested `<poem>` dropped — round-trip data-loss class | (wave 7) |
| containers pkg/crossbuild | copyFile false-FAIL on directory artifacts (jpackage app-images) — successful build reported as failure (§11.4.1), all 3 non-Apple backends | ebf2641 |
| **MAIN** pkg/storage | cache-key NUL-delimiter collision (distinct (lang,provider,model,text) tuples → same sha256 → WRONG cached translation served) across SQLite/Postgres/Redis; length-prefix fix + §11.4.120 reconcile | c8e36d8 |
| **MAIN** pkg/security | JWT with NO `exp` claim accepted → never-expiring session (jwt/v5 treats exp optional); `jwt.WithExpirationRequired()` | (wave 9) |
| **MAIN** pkg/verification | nondeterministic consensus tie-break (map-order-dependent verdict, §11.4.50) + untranslated-char double-count corrupting quality score | 54b50c3 |
| **MAIN** pkg/markdown | inline code-span corrupted by emphasis/link regexes (`file_name_v2`→`file*name*v2`) in EPUB↔MD round-trip; PUA-placeholder protection | (wave 10) |
| **MAIN** pkg/coordination | TranslateWithRetry never retried (gave up after 1 attempt/instance despite maxRetries) — silent loss of recoverable translation | 84d462e |
| **MAIN** pkg/preparation | truncateContent byte-sliced mid-rune → invalid UTF-8 Cyrillic fed to the analysis LLM, degrading character/terminology JSON; rune-boundary backup | ae33585 |
| **MAIN** pkg/grpc | CoreTranslatorImpl GetStatus read job state lock-free while the pipeline mutated it — data race on every in-flight status poll (RPC); RLock + mutator locks | (wave 12) |
| **MAIN** pkg/progress | items-only mode (totalChapters=0) never computed PercentComplete — dashboard bar stuck at 0% to completion; items-driven percent branch | e4c96f6 |
| **MAIN** pkg/language | detectCyrillicLanguage classified `'й'` as Bulgarian → plain Russian (Война и мир, Российская Федерация) detected as Bulgarian → WRONG target language → wrong translation; `'й'` removed + §11.4.120 reconcile of 3 tests | (wave 13) |
| **MAIN** pkg/api | `GET /languages` advertised 29 languages the translate endpoints reject with HTTP 400 (cross-endpoint contract violation); now sourced from language.GetSupportedLanguages() | (wave 13) |
| **MAIN** pkg/sshworker | ProgressTracker.GetProgress leaked the live Details map into the snapshot → concurrent map read+write race (event subscribers); deep-copy under RLock | 3065029 |
| **MAIN** pkg/models | FindBestModel tie-break iterated a map (random order) → different "best" model selected run-to-run on identical inputs (§11.4.50, translation-critical); sort candidates by ID | (wave 14) |
| **MAIN** pkg/deployment | UpdateAllServices double-tagged pinned images (`repo:v1.2.3:latest` invalid) corrupting every pinned service + SSHDeployConfig.Validate mutated a shared config (data race); tag-aware helper + pure-validator/withDefaults-copy + §11.4.120 reconcile | (wave 14) |
| **MAIN** pkg/challenge_runner | 6 mutation challenge scripts had NO trap → an interrupted `go test` left REAL project source mutated with a `.bak` (§11.4.84 residue corruption in the anti-bluff harness itself); restore-trap armed before each sed | 9539233 |
| **security submodule** | SSRF guard octal/hex IP-encoding bypass (`0177.0.0.1`→127.0.0.1, `012.0.0.1`→10.0.0.1 reach INTERNAL via cgo/libc inet_aton) — a real SSRF vuln; ParseInetAtonIP added (verified vs compiled C inet_aton) | 1ef9f4e |
| **MAIN** cmd/unified-translator | auto output filename hardcoded `_sr.epub` ignoring -target-lang → a French/German/… translation silently labelled Serbian (output + session-report filenames); honors targetLang + §11.4.120 reconcile | (wave 15) |
| **MAIN** pkg/version | codebase hasher used strings.Contains for excludes → over-excluded `vendored/`/`prod.env.json` + dead globs (`*.log` never matched) → corrupted the distributed worker version-sync hash (false match/spurious resync); proper component + glob match | (wave 15) |

**Flagged not-hermetically-testable (honest §11.4.6, NOT fabricated):** pkg/sshworker UploadFile/DownloadFile/UploadData build unquoted shell commands (`mkdir -p %s`, `cat %s`, `> %s`) — a remote path with spaces/metacharacters breaks/injects; no executor seam to test hermetically (dials real *ssh.Client). Worth a command-runner-interface refactor → P-list. Also ExecuteCommand hardcodes ExitCode:1 (loses *ssh.ExitError fidelity) — same live-SSH-only constraint.

**More clean non-findings (§11.4.6):** pkg/format + pkg/hardware (detection correct; 2 Windows-only leads UNCONFIRMED-flagged, not fabricated), pkg/report (snapshot+clamp+locks correct). Hunt has reached the well-hardened core — consecutive clean non-findings across format/hardware/report/translator/events/websocket/script signal §11.4.118 completeness for the rule-based + already-hardened packages.

**Clean non-findings (§11.4.6, mutation-verified the existing guards are real):** pkg/translator root (data-loss guard genuine), pkg/events+pkg/websocket (concurrency already hardened), pkg/script (rule-based scope correct). Hunted main packages now broad: distributed, verifier, ebook/fb2, storage, security, verification, markdown, coordination, translator, events, websocket, script + 8 submodules.

**Inline non-finding (§11.4.6):** pkg/script (Serbian Cyrillic↔Latin) audited — conversion correct for its rule-based scope (all 30 letters both directions, uppercase-digraph all-caps logic sound, rune-iterated); the loanword `nj`/`dž` disambiguation (e.g. `injekcija`→`инјекција`) is an inherent rule-based-transliteration limitation needing an exception dictionary, NOT a narrow bug — no fix manufactured.

**Main-repo + submodule-pointer sync (operator directive 2026-06-14):** main repo committed the 4 `pkg/distributed` concurrency fixes (above) + advanced the 8 submodule gitlink pointers (challenges/containers/doc_processor/docs_chain/llm_orchestrator/llm_provider/llms_verifier/vision_engine) to their pushed HEADs so parent ↔ submodules are in sync. NOT staged: `helix_qa` (`m`, another session's in-flight work, §11.4.119); tracked rebuilt binaries `preparation-translator`/`translate-ssh`/`unified-translator` + untracked `hash`/`workable-items` (build artifacts, §11.4.30 — never committed).

**Anti-bluff non-findings (honest, §11.4.6 — did NOT manufacture fixes):** llm_provider
`generic.go` empty-choices→empty-content is by-design tested contract; numerous adapters/packages
audited clean with enumerated coverage; vision_engine gocv-build bug already fixed upstream.

**§11.4.147 NOTE (operator awareness):** a SEPARATE sibling clone exists at
`/Volumes/T7/Projects/llm_orchestrator` (outside the parent tree). A subagent accidentally wrote
the first RoundRobinSelector fix THERE; I left it untouched (unknown provenance) and re-applied the
fix correctly inside the submodule (committed 4e08b93). That sibling now holds a stale, superseded,
uncommitted change — safe to discard, but flagged so it isn't mistaken for live work.

<!-- session 2026-06-14u: wave 14 governance/coverage build-out + e2e fix (HEAD 84f7a46) -->

### Session 2026-06-14u — wave 14: governance depth + coverage + e2e build-fix (HEAD 84f7a46)

- **22a32e3** §11.4.93 SQLite SSoT extended: `sync db-to-md` (byte-stable summary round-trip), `item_history` table + `record-event`, `diff`. (Full per-item-prose body round-trip honestly noted as not-yet-in-schema.)
- **c049907** pre-build gates +2 → **8-gate suite, all green** (CM-DOC-SIBLING-SYNC §11.4.65 + CM-NO-FORCE-PUSH-ABSOLUTE §11.4.113), all 8 paired mutations pass; caught a real gate-regex bug pre-commit.
- **fef82cc** real e2e pipeline coverage: `test/integration/pipeline_roundtrip_test.go` (FB2/TXT × EPUB/FB2 + httptest-provider HTTP path), asserts actual output-artifact content, empty-stub mutation-proven.
- **84f7a46** fixed a pre-existing §11.4.1 build break: `test/e2e` package didn't compile (unused `discovery` import) — removed; `go vet -tags e2e ./test/e2e/` now clean.

**FINAL sweep GREEN** at HEAD 84f7a46 (quiescent): `go test ./... -p 1` = **58 ok / 0 FAIL**; 8-gate pre-build suite PASS; build+vet clean; all FF-pushed both upstreams (no force §11.4.113).

**NEW tracked gap (for follow-on):** `scripts/commit_all.sh` does NOT auto-invoke `sync_all_markdown_exports.sh`, so doc commits leave stale `.html`/`.pdf` siblings (CM-DOC-SIBLING-SYNC catches them post-hoc; the gate agent had to regen CONTINUATION siblings manually). Follow-on: wire the export-sync into commit_all.sh (§11.4.75 layer-2). Until then, regen siblings when committing a doc.

#### SESSION FINAL STATE (unchanged from Rev 43 + wave 14) — autonomous queue DRAINED
Loop correctly rests on operator decisions. Remaining items (all operator-gated or larger multi-session follow-ons) tracked in docs/WORKING_PLAN.md + docs/Issues.md ATM-065..081: version number · pkg/hash remove-vs-relocate · provider keys/balance → allowlist audits · design calls · fuller §11.4.93 body round-trip · commit_all export-sync wiring · owned-submodule waves (another session active on helix_qa) · full §11.4.27 test-type matrix · full §11.4.40 retest → §11.4.151 release tag. Resume: point a fresh session at this file + docs/WORKING_PLAN.md.

<!-- session 2026-06-14t: plan-execution wave 13 — autonomous queue DRAINED (HEAD 6444e38) -->

### Session 2026-06-14t — plan-execution wave 13; autonomous queue DRAINED (HEAD 6444e38)

Wave 13 closed the last clean autonomous WORKING_PLAN items (3 background subagents, disjoint, all real + proven):
- **e623ffb** §11.4.93 workable-items SQLite SSoT: `cmd/workable-items` Go tool (`sync md-to-db`/`validate`/`list`) + tracked `docs/workable_items.db` (§11.4.95). Real run: 81 items (64 fixed + 17 open), validate exit 0, verified via `sqlite3 SELECT`; mutation-proven. (Fuller schema — item_history/obsolete_details/db-to-md/fingerprint — is a tracked follow-on.)
- **78023a7** P4.2 +2 gates: CM-TRACKER-DOCS-PRESENT + CM-ATM-TICKET-IDS → **pre-build suite now 6 gates, all green**, each paired-mutation-proven.
- **6444e38** P4.3/ATM-076 §11.4.65 doc-export: `scripts/testing/sync_all_markdown_exports.sh` (pandoc html + weasyprint pdf, idempotent, `--check` gate mode, §11.4.106 honest tool-absence) + **225 siblings generated (0 failures, 0 stale)** for 120 in-scope docs. Caught + excluded a stray docs/research/helix_qa/ workspace dump (§11.4.30 guard).

**FINAL integration sweep GREEN** at HEAD 6444e38 (quiescent): `go test ./... -p 1` = **58 ok / 0 FAIL**; 6-gate pre-build suite PASS; build+vet clean. All FF-pushed both upstreams (no force §11.4.113).

#### SESSION FINAL STATE — autonomous queue DRAINED
**Delivered:** ~30 genuine mutation-proven bug fixes (13 waves; main-module bug-hunt SATURATED per §11.4.118 completeness audit); format matrix complete (6 in × 5 out); PDF+DOCX input revived; deepseek-v4 allowlist current; version single-source (5 binaries); workable-items tracker constellation (81 ATM tickets) + SQLite SSoT; 6-gate mutation-proven pre-build suite; §11.4.65 doc-export (225 siblings); perf/stress/chaos coverage (5 hot-path packages); docs/WORKING_PLAN.md (canonical unfinished+known-issues inventory).

**REMAINING — all operator-gated or larger tracked follow-ons (NOT autonomously closable; see docs/WORKING_PLAN.md + docs/Issues.md ATM-065..081):**
1. **OPERATOR DECISIONS:** P1.0 version number (2.3.0 vs 3.0.0; wiring done) · P3.1 pkg/hash remove-vs-relocate · P1.1 provider keys/balance (OPENAI/ANTHROPIC absent, GEMINI invalid, ZHIPU no-balance) → unblocks stale-allowlist audits (deepseek pattern proven) · P2.x design calls (inert flags, MinScoreThreshold scale, reasoning-model structured-content, markdown first-class input).
2. **LARGER TRACKED FOLLOW-ONS:** fuller §11.4.93 schema (item_history/db-to-md/fingerprint) · owned-submodule bug-hunt waves (§11.4.28 — ANOTHER SESSION ACTIVE on helix_qa, coordinate per §11.4.119) · full §11.4.27 test-type matrix + HelixQA/Challenges banks · P2.6 cmd/translator SSH download-dir (needs live SSH via §11.4.76 Containers).
3. **RELEASE:** P0.2 full §11.4.40 7-step retest → P0.3 §11.4.151 `helix_translate-<version>` tag (after the operator decisions + the test-type matrix).

**Loop status (§11.4.94(A)/§11.4.101/§11.4.126):** correctly rests on operator decisions — every remaining item is externally/operator-gated and explicitly tracked; nothing silently dropped. Resume by pointing a fresh session at this file + docs/WORKING_PLAN.md.

<!-- session 2026-06-14s: WORKING_PLAN + plan-execution waves 10-12 (HEAD 06a8a16) -->

### Session 2026-06-14s — WORKING_PLAN created + plan-execution begins (HEAD 06a8a16)

Operator mandate: comprehensive subagent-driven plan of all unfinished items + known issues, then execute. Bug-hunt declared SATURATED by the §11.4.118 completeness-critic (wave 10).

- **docs/WORKING_PLAN.md** (78ce2d0) — the canonical no-bluff inventory: P0 release blockers, P1 operator/credential-gated, P2 design decisions, P3 dead code, P4 governance gaps, P5 owned submodules, P6 full test-type coverage. Each item: WHAT/WHY-OPEN/EVIDENCE/SUBAGENT-TASK/ACCEPTANCE.
- **Wave 10 (2 fixes + saturation verdict):** e520c4c `enhanceTranslation` byte→rune capitalization (silently no-op for Cyrillic/accented first letters) + isRussianToSerbian empty-source (told LLM wrong source lang); 5b0dd17 the orphaned guard. docx_parser: no bug (exhaustive). Completeness-critic: main-module product bug-hunt SATURATED.
- **Wave 11 (3 plan items):** a36030e P0.1 version single-source (`pkg/version/app.go AppVersion=2.3.0`; 5 divergent binaries reconciled, runtime-verified, mutation-proven); 981ced9 P3.1 §11.4.124 pkg/hash investigation (dead `package main` duplicate of pkg/version.CodebaseHasher — finding doc written, NOT removed per §11.4.122, **operator decision queued: remove-as-duplicate vs relocate**); 279ac7c P4.2 first real pre-build gate suite (CM-GITIGNORE-PRECOMMIT-AUDIT + CM-NO-FAKES-BEYOND-UNIT + paired mutation meta-tests). 59819fa→06a8a16 §11.4.120 reconcile of stale version assertions in cmd/cli (`v2.0.0`→dynamic) + cmd/server (fixed a tautological `Contains("v1.0.0","1.0.0")` §11.4.1 bluff). [Note: 59819fa committed a test before `go test` verified it — `go build ./...` skips `_test.go`; repaired fix-forward in 06a8a16. Lesson: use `go vet`/`go test` as the pre-commit compile check for test changes.]
- **§11.4.65 finding (P4.3):** only 3 of 87 tracked `docs/*.md` have `.html`/`.pdf` siblings — large export gap (needs pandoc/weasyprint or honest §11.4.106 ToolAbsent skip).
- **Wave 12 in flight:** P4.1 tracker constellation scaffold (Issues/Fixed/summaries + ATM-NNN) · P6 perf/benchmark + stress/chaos coverage · P4.2 gate-suite expansion (CM-SCRIPT-TARGET-SHELL-PARSEABLE + CM-VERSION-SINGLE-SOURCE).

**Integration sweep GREEN** at HEAD 06a8a16 (quiescent): `go test ./... -p 1` = **57 ok / 0 FAIL**. build+vet clean. CAMPAIGN TOTAL: ~30 genuine mutation-proven fixes; main-module bug-hunt saturated; plan execution underway.

**OPERATOR DECISIONS QUEUED:** (1) P1.0 authoritative version number (2.3.0 vs 3.0.0); (2) P3.1 pkg/hash remove-vs-relocate; (3) P1.1 provider keys/balance (OPENAI/ANTHROPIC absent, GEMINI invalid, ZHIPU no balance) to unblock allowlist audits; (4) P2.x design decisions (inert flags, MinScoreThreshold scale, reasoning-content support). See docs/WORKING_PLAN.md.

<!-- session 2026-06-14r: bug-hunt waves 8-9 — 7 more genuine fixes (HEAD daa2f70) -->

### Session 2026-06-14r — bug-hunt waves 8-9: 7 more genuine mutation-proven fixes (HEAD daa2f70)

Continued the §11.4.70/§11.4.103 parallel campaign (background subagents on disjoint cmd/ + pkg/ scopes; reproduce-first + mutation-proven + single-package `-p 1`/`-race`; FF-pushed, no force).

- **86d10d4** cmd/translate-ssh: `-output` ignored — translated EPUB delivered to the INPUT dir instead of the requested path, then `printFinalReport` claimed success pointing at a nonexistent file (wrong-delivery + false-success).
- **309ca91** cmd/cli: `gemini` missing from `getAPIKeyFromEnv` env map → `GEMINI_API_KEY` silently ignored (translation failed with a key present).
- **21c2460** cmd/grpc-server: documented env-var overrides (`GRPC_ADDRESS/PORT`, `LOG_LEVEL`, `ENABLE_METRICS/REFLECTION`) never implemented → silently ignored.
- **cde2e1a** cmd/deployment: `handleStatus` sliced `ContainerID[:12]` unconditionally → panic crashing the whole status command on any <12-char/empty ID.
- **93e89d7** cmd/grpc-server + cmd/monitor-server: wired the dead `-max-connections` flag (was hardcoded 50) + stopped swallowing monitor-server's `router.Run()` bind error (silent failed start).
- **daa2f70** logger: JSON reserved-key collision — a user field named `level`/`message`/`timestamp` overwrote the authoritative metadata → real severity/message/timestamp silently dropped.

**Integration sweep GREEN** at HEAD daa2f70 (quiescent): `go test ./... -p 1` = **57 ok / 0 FAIL** (up from 55 — the wave fixes added committed tests to 5 cmd/ packages that previously had none). build+vet exit 0.

**CAMPAIGN TOTAL (waves 1-9): 25 genuine mutation-proven fixes (27 subagents).** SESSION GRAND TOTAL ~32 real fixes. Wave 10 (completeness-critic audit + docx_parser + llm prompt/cache helpers) in flight to confirm saturation.

REMAINING (operator/design-gated, unchanged): OPENAI/ANTHROPIC keys absent; GEMINI_API_KEY invalid; ZHIPU out of balance (allowlist stale, can't verify); ~30 other provider allowlists need funded keys; verifier `MinScoreThreshold` 0-100-vs-0-10 scale (no caller, operator decision); `-chunk-size` inert flag (design); markdown first-class input (design); reasoning-model structured-`content` client support (design); cmd/translator intermediate-md download-dir inconsistency (needs live SSH); submodules (challenges/containers/helix_qa) being worked by another session — do not collide (§11.4.119). NO release tag yet (needs §11.4.40 full retest + operator confirm + §11.4.151 prefix).

<!-- session 2026-06-14q: parallel subagent bug-hunt campaign — 18 genuine fixes across 7 waves (HEAD 4bbac9a) -->

### Session 2026-06-14q — parallel subagent-driven bug-hunt campaign (§11.4.70/§11.4.103): 18 genuine mutation-proven fixes across 7 waves (HEAD 4bbac9a)

Operator directive: continuous endless loop with 3–4 parallel subagents on all parallelizable workable items, rock-solid evidence, zero bluff. Ran 7 waves (21 subagents on DISJOINT package scopes per §11.4.58/§11.4.119); every fix reproduce-first (§11.4.115) + mutation-proven (§1.1) + single-package `-p 1`(+`-race`) validated; all FF-pushed to both upstreams (no force §11.4.113). 4 subagents honestly reported NO bug (events/websocket, storage, api/services) — no manufactured changes (§11.4.6).

**The 18 fixes:**
- **80d627b** hardware: `parseLscpuCores` undercounts cores on multi-socket hosts (ignored `Socket(s)`; dual-socket 16-core → 8).
- **0df25d9** preparation: coordinator never stamped `ChapterNum` → 0 → positional-fallback chapter-context mis-attribution after a failed chapter compacts.
- **c3117f8** batch: same-stem/different-ext inputs (`book.fb2`+`book.epub`) collide onto one output → data loss; reconciled the 3 stem-only tests (§11.4.120).
- **d0fe40a** security: `RefreshToken` minted fresh tokens from EXPIRED/no-expiry claims (auth-bypass, session resurrection); zero residue confirmed.
- **d8142e5** verifier: `/api/v1/verified-models` hard-errored on zero verified models (`{"models":null}` fell through to array-decode).
- **20beda7** distributed: `getFailureRate` fabricated 100% failure rate on zero/expired window → falsely traps coordinator in degraded mode.
- **8f52370** fb2: `<v>`/`<subtitle>`/`<text-author>` dropped inline-element text (verse/quote/attribution data loss).
- **a728e57** llm/ollama: dropped `temperature`/`max_tokens` (must be in `options` per official API §11.4.99) → non-deterministic output.
- **be81550** markdown: chapter XHTML `<title>` unescaped → malformed EPUB for titles with `&`/`<`/`>`.
- **7434ee4** verification: `polishChapter`/`polishSectionRecursive` index-out-of-range panic when translated has fewer sections.
- **fd07e18** grpc: `SubscribeEvents` delivered lifecycle events with empty `session_id`.
- **107e570** ebook/epub_parser: chapter text shipped HTML entities literally (`Tom &amp; Jerry`, `caf&#233;`).
- **fa35d2e** config: nil-map panic in `LoadConfig` (omitted `providers` + any API-key env var set).
- **8316e3f** translator core: data race / concurrent-map-write crash in `BaseTranslator` cache+stats (race-proven).
- **c416bd8** ebook/epub_writer: cover image mislabeled `image/jpeg` for PNG/GIF/WEBP/SVG covers → broken cover.
- **36f740a** cmd/unified-translator: SSH provider used a hardcoded "ru→sr-cyrillic" prompt + hardcoded binary/model paths, ignoring all user flags (also carried the cmd/api-server health fix below, swept in by the wrapper).
- **(36f740a)** cmd/api-server: `healthCheck` returned 200 "healthy" regardless of gRPC backend state → derives from `conn.GetState()` → 503 when not Ready.
- **85b3362** cmd/markdown-translator: `-format` flag never validated → unsupported value prints success but writes no file (false-success bluff).
- **4bbac9a** .gitignore (§11.4.30): bare `api-server` pattern ignored the whole `cmd/api-server/` SOURCE dir (no repo-root binary exists; build output is `build/api-server`) → hid the package + blocked its tests (api-server had ZERO committed tests). Anchored to `/api-server`; committed the health regression guard.

**Operator-flag (no production caller, design decision):** verifier `MinScoreThreshold` scale inconsistency — handler compares raw 0-100, adapter compares normalized 0-10; `GetPreferences`/`GetProviderScore` have no production caller; needs an operator decision on the canonical scale before either is wired (not guessed, §11.4.6/§11.4.120).

**Process lesson (§11.4.119):** the full `go test ./... -p 1` integration sweep includes pkg/version's CodebaseHasher which hashes the whole tree (.go + .md + docs/) — it MUST run on a QUIESCENT tree; running it concurrently with file-writing subagents produces a spurious pkg/version hash-mismatch FAIL (confirmed green isolated). Sweeps + CONTINUATION syncs are therefore serialized BETWEEN parallel waves.

**Integration sweeps GREEN** at each quiescent checkpoint: waves 1-5 → 54 ok/0; waves 6+7 → **55 ok/0 FAIL** (HEAD 4bbac9a; cmd/api-server now has committed tests). build+vet exit 0. **SESSION GRAND TOTAL: ~25 real fixes** (this campaign's 18 + earlier: PDF revival, format matrix + .html, deepseek-v4 allowlist, websocket race, ssh-test, 2 doc-syncs).

<!-- session 2026-06-14p: zhipu allowlist finding (operator-blocked) + queue-drained status (HEAD a33f20b) -->

### Session 2026-06-14p — Zhipu allowlist §11.4.150 finding (operator-blocked); autonomously-provable queue drained

- **Zhipu allowlist is stale (Operator-blocked, §11.4.21):** live Zhipu /models (account-authoritative, verified 2026-06-14) returns **glm-4.5, glm-4.5-air, glm-4.6, glm-4.7, glm-5, glm-5-turbo, glm-5.1** — NONE of our allowlisted glm-4 family (glm-4/glm-3-turbo/glm-4-plus/glm-4-flash/glm-4-air/glm-4-airx/glm-4-long/glm-4-flashx) are listed. The generic factory (llm.go:218) HARD-REJECTS unlisted models, so a funded account requesting a current model is rejected (this is what the earlier "zhipu rejects documented models" flag actually was). **Why not fixed autonomously:** the account is OUT OF BALANCE (chat API error 1113 "余额不足/insufficient balance") → I cannot obtain §11.4.123 translation proof NOR verify response-shape compatibility (the magistral lesson: glm-5 reasoning models may return structured-list `content` our string-content client cannot parse — unverifiable while balance-blocked). Blindly adding glm-5 could enable a client-incompatible model. **Operator-Block-Details:** WHAT — recharge the Zhipu account OR provide a funded ZHIPU_API_KEY; WHY — every self-resolution exhausted (key present + API reachable, but balance blocks all verification; cannot manufacture credit); UNBLOCK CONDITION — a funded zhipu key so the deepseek-pattern fix (verify-translate + string-content shape, then additive allowlist update + RED-proven gate guard) can be applied; WHO — operator.
- Contrast: **deepseek** had a funded key → fully provable → FIXED this session (0fd1a34). The verify-then-fix pattern is established; it just needs a funded provider account.

**AUTONOMOUSLY-PROVABLE QUEUE DRAINED.** All clearly-actionable, evidence-provable main-module work this session is done + green. Remaining known issues are ALL operator/research/design-gated (cannot be closed with rock-solid proof autonomously):
- **Operator (credentials/balance):** add OPENAI_API_KEY + ANTHROPIC_API_KEY (absent); refresh GEMINI_API_KEY (live /models = "API Key not found"); recharge/replace ZHIPU_API_KEY (balance 1113). Each unblocks a verify-then-fix allowlist/E2E pass.
- **Research (needs funded keys):** ~30 other provider allowlists not audited (audit pattern proven: live /models → verify-translate + string-content shape → additive update).
- **Design:** `-chunk-size` CLI flag is inert (real chunking is automatic+correct via translateWithRetry/splitText; wiring the flag has ambiguous semantics OR removing it needs §11.4.122 operator confirm); markdown not a first-class CLI input (works as TXT today); reasoning-model structured-list `content` support in the OpenAI-compatible clients (would enable magistral/glm-5-class models).
- **Release:** NO tag yet — needs §11.4.40 full retest + operator confirm + §11.4.151 `<prefix>-<version>` naming.

Loop status (§11.4.94(A)/§11.4.101/§11.4.126): idle-on-blocked — every remaining item is externally/operator-gated; no item is autonomously closable with rock-solid proof. Build is the most stable this session: full sweep 54 ok/0 FAIL, websocket race-clean, format matrix complete (6 in × 5 out), deepseek current.

<!-- session 2026-06-14o: latent brittle-test fixes surfaced by the sweeps (HEAD 817b9dd) -->

### Session 2026-06-14o — latent brittle test-harness defects fixed (HEAD 817b9dd); full sweep GREEN 54/0

Two latent test defects (NOT caused by this session's source changes — different packages) surfaced one-per-sweep by the §11.4.118 discovery-pressure of repeated full sweeps, root-caused (§11.4.102) + fixed + proven:

- **ab7db0e** tests: DETERMINISTIC goroutine panic in tests/websocket_monitoring_test.go — concurrent map WRITE at the handleClientMessages defer delete. `go test -race` confirmed 22 races (suite.server.Clients + suite.testSessions + *TestSession fields accessed from the main goroutine, /status handler, and per-client goroutines without consistent locking; concurrent map read+write is a Go FATAL). Fixed: all shared-map access under suite.server.mu; thread-safe read helpers (clientCount/sessionCount/sessionSnapshot/totalSessionEvents, lock+copy, no I/O under lock) for /status + every assertion; TestMultipleClients writers joined via sync.WaitGroup (sleep = no happens-before edge → leaked writer raced another test's write to the same gorilla conn). PROVEN: `-race -count=3` → 0 races; plain `-count=5` → ok.
- **817b9dd** tests: DETERMINISTIC fail in cmd/translate-ssh TestSSHErrorHandling/connection_timeout — the 'connection timeout' case asserted the error contains 'connection refused', but this host's slow localhost resolver returns 'lookup localhost: i/o timeout' (§11.4.1 FAIL-bluff: failed on environment, not product defect). Broadened the connection-class assertion to accept any connection-failure indicator (refused/timeout/no route/unreachable/no such host/lookup); auth case keeps its specific check; still requires a real error. PROVEN: `-count=5` → ok.

**Scope note (§11.4.28):** the same brittle 'connection refused' assertion class exists in the challenges/containers/helix_qa SUBMODULES (separate go.mod → NOT in the main `go test ./...` sweep; their own repos/sweeps — separate work, noted for follow-up).

**FULL SWEEP GREEN** at HEAD 817b9dd: `go test ./... -p 1` = **54 ok / 0 FAIL** (qa-results/full_sweep_20260614_155722.log). build+vet exit 0. All 7 main-stream commits this session (PDF, output-format, .html, doc-sync, deepseek-v4, websocket-race, ssh-test) pushed FF to both upstreams (no force §11.4.113).

<!-- session 2026-06-14n: provider allowlist §11.4.150 audit — deepseek v4 fix + findings (HEAD 0fd1a34) -->

### Session 2026-06-14n — provider model-allowlist §11.4.150 audit (live /models, not memory)

- **0fd1a34** deepseek: ValidModels[ProviderDeepSeek] was stale — only legacy {deepseek-chat,deepseek-coder}; deepseek.go HARD-REJECTS unlisted models, so the CURRENT flagship was rejected. Live DeepSeek /models (verified 2026-06-14) returns **deepseek-v4-flash + deepseek-v4-pro**; v4-flash proven to translate via full CLI (docs/qa/e2e_deepseek_v4_*, 136 Cyrillic chars). Added both (NOT deepseek-reasoner — not in /models, §11.4.6); kept legacy aliases (§11.4.122). RED-proven guard deepseek_v4_models_test.go. **2d98509** doc-sync: generateOutput comment now lists .html/.htm.
- **AUDIT FINDINGS (captured, for operator/future work):**
  - **Mistral**: allowlist {large,medium,small}-latest all present in live /models → FUNCTIONAL (aliases track current). NOT widened: the `magistral-*` reasoning family returns `content` as a STRUCTURED LIST (thinking blocks), which our OpenAI-compatible string-content clients CANNOT parse — adding magistral-medium would break for users. **Latent gap (operator/design):** supporting reasoning-model structured `content` is a non-trivial client change, not a blind allowlist add. (deepseek-v4-flash is safe because it returns `content` as a plain STRING with reasoning in a separate field — verified.)
  - **Gemini**: live /models re-confirms `API Key not found` → §11.4.6 the gemini failures are an OPERATOR CREDENTIAL issue, NOT a code bug. Operator action: refresh GEMINI_API_KEY.
  - Other ~30 providers' allowlists: not audited (no keys / would need per-provider live verification). Adding current models is safe-but-must-be-verified-per-model (string-content shape + real translation), never from memory (§11.4.99/§11.4.150).

<!-- session 2026-06-14m: .html output added — format matrix complete (HEAD fb07a59) -->

### Session 2026-06-14m — .html output added; input×output format matrix complete (HEAD fb07a59)

- **fb07a59** cli: added `.html`/`.htm` output to `generateOutput` (was: errored, while INPUT fully supports HTML — asymmetric gap). Minimal valid HTML5, HTML-escaped content (no markup injection). REAL E2E PROVEN (docs/qa/e2e_html_output_20260614_152301/): English PDF → real DeepSeek → out.html (valid HTML document, DOCTYPE + 2 `<p>`, 140 Cyrillic chars, not a zip). Guard extended (output_format_test.go): well-formed-HTML + escaping (no `<script>` injection) subtests.

**FORMAT MATRIX NOW COMPLETE:** input FB2/EPUB/TXT/HTML/DOCX/PDF (6) × output EPUB/FB2/TXT/MD/HTML (5) — all real-translation proven this session set (PDF/DOCX revived from license-gated-dead; output formats honored; HTML output added). **SESSION GRAND TOTAL: ~84 real bugs/features** + 5 reconciliations. build+vet exit 0; full `go test ./... -p 1` sweep GREEN at HEAD fb07a59: **54 ok / 0 FAIL** (qa-results/full_sweep_20260614_152400.log).

<!-- session 2026-06-14l: CLI output-format honored + 2 gate reconciliations (HEAD ca5608f) -->

### Session 2026-06-14l — CLI honors -o output format + 2 §11.4.120 reconciliations (HEAD ca5608f)

- **ca5608f** cli: `unified-translator` ALWAYS emitted EPUB regardless of the `-o` extension → `-o book.txt`/`-o book.fb2` wrote EPUB (PK-zip) bytes into a misnamed file (silent wrong-output, §11.4). New `generateOutput()` dispatches on extension: `.epub` (default)→EPUB; `.txt`/`.md`→translated text direct; `.fb2`→FB2Writer; unsupported→explicit error (§11.4.6). REAL E2E PROVEN (docs/qa/e2e_output_formats_20260614_151507/): same PDF → real DeepSeek → out.txt (UTF-8 plain, not zip, 135 Cyrillic) + out.fb2 (valid FictionBook XML, 138 Cyrillic). Permanent guard cmd/unified-translator/output_format_test.go, MUTATION-PROVEN (force always-EPUB → .txt/.fb2 FAIL).
- **§11.4.120 reconciliations surfaced by the post-PDF full sweep:**
  - test/unit/format_detector_test.go: PDF + DOCX moved to supportedFormats (real extractors landed); MOBI stays unsupported.
  - cmd/cli TestTranslateEbookFunction/with_app_config: was a §11.4.98 BLUFF — skip-guard fired only with NO key, then forced openai + fake "config-key", dialled REAL OpenAI, 401'd, failed NoError whenever ANY other provider key was in env (DEEPSEEK/ZHIPU/GEMINI/MISTRAL present this session). Rewritten to a self-driving httptest OpenAI mock (config BaseURL→mock); asserts success + mock-hit + non-empty output. No real network, no env dependency.

**SESSION GRAND TOTAL: ~83 real bugs** + 5 reconciliations + §11.4.98 conversions + real E2E proofs (PDF, output-formats, 5-format matrix via DeepSeek + Mistral). All 6 input formats + 4 output formats (.epub/.fb2/.txt/.md) proven end-to-end. Post-fix full `go test ./... -p 1` sweep GREEN at HEAD ca5608f: **54 ok / 0 FAIL** (was 52 ok/2 stale-gate FAIL pre-reconcile; qa-results/full_sweep_20260614_151656.log). build+vet exit 0.

<!-- session 2026-06-14k: PDF input revived (HEAD 18d6f73) -->

### Session 2026-06-14k — PDF input revived: license-gated unipdf → MIT ledongthuc/pdf (HEAD 18d6f73)

- **18d6f73** ebook: PDF INPUT was non-functional — `github.com/unidoc/unipdf`'s `ExtractText` is LICENSE-GATED ("unipdf license code required" for every real PDF), so PDF input shipped dead (same §11.4 "ships but cannot be used" class DOCX was, now both fixed). Swapped to MIT `github.com/ledongthuc/pdf`; dropped `unidoc/unipdf` via `go mod tidy`. Wired `FormatPDF` into `pkg/format/detector.go` IsSupported + GetSupportedFormats (§11.4.108 source→artifact gate; §11.4.120 reconcile of detector_test + pdf_parser_test's unipdf-coupled error-string assertion). REAL E2E PROVEN (docs/qa/e2e_pdf_input_20260614_150421/): minimal English PDF → real DeepSeek deepseek-chat (1.9s) → valid Serbian Cyrillic EPUB ("Храбри витез јахао је преко тихог сванућа у зору"), 138 Cyrillic chars, differs from source, no placeholders, no API-key leak (§11.4.10). Permanent regression guard (§11.4.135): `pkg/ebook/pdf_extraction_regression_test.go` + `testdata/sample_text.pdf` assert real extraction + reject any license-error string; mutation reverting to unipdf → FAIL.

**SESSION GRAND TOTAL: ~82 real bugs** + 3 reconciliations + §11.4.98 conversion + real E2E proofs (PDF + 5-format matrix via DeepSeek + Mistral). PDF + DOCX (the two dead input formats) now both work end-to-end with rock-solid real-translation evidence. ALL 6 input formats (FB2/EPUB/TXT/HTML/DOCX/PDF) proven. Post-fix targeted tests green (pkg/ebook + pkg/format); full -p 1 sweep running in background (FD ~95%, host-safe). build exit 0.

REMAINING KNOWN ISSUES (operator/research/design-gated): OpenAI/Anthropic keys absent; GEMINI_API_KEY invalid; zhipu account/model access; markdown not a first-class CLI input; other ~30 providers' model-allowlists not audited; verifier 3-tier merge precedence + run.go registry flags (design); RefreshToken no expiry re-validate (no caller); CLI always emits EPUB (no TXT writer); SQLite encryption DSN needs SQLCipher; G1/G2 carry-overs; W16 Models→submodule. NO release tag yet (needs §11.4.40 full retest + operator confirm).

<!-- session 2026-06-14j: known-issue cleanup wave (HEAD b83038c) -->

### Session 2026-06-14j — design/known-issue cleanup (HEAD b83038c) — 3 more real fixes + gemini/zhipu clients cleared

Subagent-driven wave (1 completed via subagent, 3 rate-limited → finished inline per §11.4.101/§11.4.147):
- **77a0c15** coordination: TranslateWithConsensus tie-break was non-deterministic (map-iteration order on vote ties) → sort candidates, strict '>', lexicographic tie-break. Mutation-proven (1000 runs identical).
- **9640a00** verification: multipass chapter loop panicked (index out of range) when the translation has FEWER chapters than the original → bound the loop by min(original,current,polished) + log/skip the tail. FACT code-analysis root cause; build+vet+tests green.
- **b83038c** ebook: FB2 writer dropped section/subsection TITLES (content was already lossless) → prepend each non-empty title as a paragraph. RED→GREEN→mutation-proven (real write→read).
- gemini/zhipu CLIENT investigation (issue #5) RESOLVED as **no client bug** (§11.4.6): gemini.go correctly uses https://generativelanguage.googleapis.com/v1beta + ?key= + :generateContent; zhipu.go correctly uses https://open.bigmodel.cn/api/paas/v4 + /chat/completions + Bearer. Their live failures are OPERATOR credential/account issues (invalid GEMINI_API_KEY; zhipu account lacks the model), NOT code.

**SESSION GRAND TOTAL: ~81 real bugs** + 3 reconciliations + §11.4.98 conversion + real E2E proofs (5-format matrix via DeepSeek + Mistral). Post-wave sweep at HEAD b83038c: 53 ok; the single pkg/distributed FAIL was ENFILE host-exhaustion ("too many open files", FD ~95%), NOT a regression — TestSSHConnection_ExecuteCommand passes deterministically in isolation (-count=3 green); nothing in pkg/distributed changed this round. build+vet exit 0.

REMAINING KNOWN ISSUES (operator/research/design-gated — see the operator-facing status report): OpenAI/Anthropic keys absent; GEMINI_API_KEY invalid; zhipu account/model access; PDF input non-functional (license-gated unipdf, same class DOCX was — needs a free PDF lib / go.mod decision); markdown not a first-class CLI input; other ~30 providers' model-allowlists not audited; verifier 3-tier merge precedence + run.go registry flags (design); RefreshToken no expiry re-validate (no caller); CLI always emits EPUB (no TXT writer); SQLite encryption DSN needs SQLCipher; G1/G2 carry-overs; W16 Models→submodule. NO release tag yet (needs §11.4.40 full retest + operator confirm).

<!-- session 2026-06-14i: real-translation E2E matrix caught + fixed more bugs (HEAD b23bcac) -->

### Session 2026-06-14i — real provider×format E2E matrix found + fixed more bugs (HEAD b23bcac)

Driving REAL translations across formats/providers (rate-limited subagents → done inline) surfaced + fixed:
- **41063cb** FB2 + EPUB INPUT broken end-to-end: convertToMarkdown re-parsed already-extracted text as the original format ('failed to parse FB2: EOF' / 'not a valid zip'). Fixed → all 5 input formats translate. PROVEN: real DeepSeek E2E matrix FB2/EPUB/TXT/DOCX/HTML all → valid Serbian-Cyrillic EPUB (docs/qa/e2e_format_matrix_*).
- **9821e0f** zhipu missing from cmd/unified-translator resolveProviderAPIKey env map → ZHIPU_API_KEY never satisfied the gate. Fixed (proven: now reaches the live Zhipu API). + Multi-provider proven: Mistral E2E PASS (docs/qa/e2e_multiprovider_*).
- **b23bcac** stale provider model-allowlists (gemini/groq/zhipu accepted only DEPRECATED models, rejected current → providers unusable). Added current models; live-proven the gate blocker is gone (groq llama-3.3-70b → 429 'recognized'; gemini-1.5-flash → reaches API (auth err); zhipu glm-4-flash → reaches API (account err)). docs/qa/allowlist_fix_*.

**SESSION GRAND TOTAL: ~78 real bugs** + 3 reconciliations + §11.4.98 conversion + real E2E proofs (5-format matrix via DeepSeek + Mistral 2nd-provider). OPERATOR/research-gated remainders: add OPENAI/ANTHROPIC keys (absent); GEMINI_API_KEY appears invalid/expired (gemini reaches API but 'API Key not found'); zhipu key's account/endpoint rejects documented models (possible gemini/zhipu CLIENT endpoint bug like Qwen's — flagged for investigation, not guessed §11.4.6). Post-wave full sweep GREEN at HEAD b23bcac: 54 ok / 0 FAIL. Build+vet exit 0.

<!-- session 2026-06-14h: all 4 operator-selected directives executed (HEAD c6c2930) -->

### Session 2026-06-14h — operator-selected directions ALL executed (HEAD c6c2930); real E2E translation PROVEN

Operator selected all four next-directions; executed with conductor verification + push FF:
- **Qwen endpoint** (556002d + 2f0e359): client posted native DashScope path under a compatible-mode base AND the CODE DEFAULT base was native /api/v1 while the struct+config are compatible-mode → wrong URL + unparseable shape (Qwen was broken for default+production). Fixed: default base → /compatible-mode/v1 + path /chat/completions + choices[] (all consistent, §11.4.150 docs-verified). Live test made §11.4.98-honest (SKIP on auth/endpoint failure).
- **Storage dedup** (367adce): cache served STALE translation on tuple collision (no UNIQUE index; dedup only on id PK) → lookup_hash UNIQUE index + idempotent UPSERT + safe dedup migration, both backends. Mutation-proven.
- **DOCX** (d433210): input was NON-FUNCTIONAL (license-gated unioffice returned 'license required' for every real .docx). Rewrote as a stdlib parser (archive/zip + encoding/xml over word/document.xml + docProps/core.xml); dropped unioffice (+ transitive msoleps) via go mod tidy. Proven on a real in-test .docx (text+metadata+UTF-8). DOCX input now actually works.
- **Stress/chaos + metrics race** (bf81517): added §11.4.85 stress/chaos suites (events/coordination/distributed) which UNCOVERED a real VersionManager.metrics data race + 40-60% lost counter updates → fixed with metricsMu + snapshot-on-read; KNOWN_BUG guard flipped to active GREEN (§11.4.115).
- **CLI env-key gate** (c6c2930): unified-translator's API-key gate ignored provider *_API_KEY env vars → fixed (falls back to resolveProviderAPIKey).
- **REAL E2E TRANSLATION PROOF** (c6c2930, docs/qa/e2e_deepseek_20260614T130829/): unified-translator translated test/assets/crow_and_pitcher_en.txt EN→Serbian via REAL DeepSeek (deepseek-chat, 2.89s) into a VALID EPUB — 205 Cyrillic chars ('Гавран и крчаг…'), differs from source, no placeholders, no key leak (§11.4.83/§11.4.107/§11.4.10). Full detect→parse→translate→EPUB-write pipeline confirmed working after the session's fixes.

- **DOCX CLI wiring** (6304f60): a broader multi-format E2E proof caught that the CLI still rejected .docx with 'format docx is not yet supported' — format.Detector.IsSupported/GetSupportedFormats excluded FormatDOCX (§11.4.108 source→artifact gap: parser fixed+registered but the pipeline gate refused it). Added FormatDOCX to both; reconciled detector_test (§11.4.120). REAL E2E proven (docs/qa/e2e_docx_pipeline_20260614T131850/): real .docx → real DeepSeek → Serbian Cyrillic EPUB. DOCX input now works through the actual CLI.

**SESSION GRAND TOTAL: ~75 real bugs** + 3 reconciliations + §11.4.98 conversion + 2 real E2E proofs (DeepSeek on TXT + on DOCX, both → valid Serbian-Cyrillic EPUB). OPERATOR ACTIONS to widen coverage: add OPENAI_API_KEY / ANTHROPIC_API_KEY to ~/api_keys.sh (not present; ~34 other providers available). FLAGGED (not fixed, needs current-model research §11.4.150): the gemini provider's valid-model allowlist is STALE — accepts only gemini-pro/gemini-pro-vision (both deprecated by Google), rejects gemini-2.0-flash/1.5. Post-wave full sweep GREEN at HEAD 6304f60: 54 ok / 0 FAIL. Build+vet exit 0.

<!-- session 2026-06-14g: 7th (final-coverage) SECOND-PASS wave (HEAD 5dfe003) — 4 more real bugs; comprehensive second pass COMPLETE -->

### Session 2026-06-14g — 7th/final §11.4.118 second-pass wave (HEAD 5dfe003) — 4 more real bugs; whole-codebase second pass essentially COMPLETE

- **b1b747f** progress: formatDuration trailing space in dashboard ETA ('5 hours '); events: generateEventID non-unique within a microsecond (8172 dups/10000 in a burst → broke ID-keyed consumers). Both fixed.
- **5dfe003** cmd: preparation-translator -providers flag silently ignored (hardcoded list); cli explicit -provider silently overridden by config default (precedence inversion). Both fixed.
- CLEAN second-pass (honest §11.4.6, no fabricated bugs): logger, version, hardware, report, format, websocket, script.

**SESSION GRAND TOTAL: 68 real bugs** (5 parallel waves + 3 inline + 7 second-pass waves) + 2 reconciliations + §11.4.98 conversion. ALL reproduce-first + mutation-proven + conductor-reverified (-race/-p 1) + pushed FF (no force §11.4.113). 3 distinct security issues (CWE-204 enumeration, CWE-22 path traversal, CWE-208 timing oracle); extensive data-loss/correctness across ebook pipeline (EPUB/FB2/HTML/markdown round-trips), providers (Anthropic/OpenRouter/Qwen), distributed, storage/cache, verification, preparation, translator-core, batch, coordination, grpc, api. The codebase has had a comprehensive 2-pass hunt; remaining items are operator/review-gated (see flags below). Post-wave full sweep GREEN at HEAD 5dfe003: 54 ok / 0 FAIL (fullsweep_20260614T120039.log). Build+vet exit 0.

OPERATOR/REVIEW-GATED carry-overs (NOT autonomously fixable — need a decision): Qwen native-vs-compatible endpoint mode; storage cache dup-tuple UNIQUE index (schema/dedup-semantics); DOCX unioffice license (go.mod dependency); detected source-lang not propagated to llm translator (cross-package wiring); FB2 writer section-title flatten; verifier 3-tier merge precedence + run.go registry flags; coordination consensus tie-break determinism; G1 verify→server-DB bridge; G2 batched ~30-provider sweep; G3 add OPENAI/ANTHROPIC keys.

<!-- session 2026-06-14f: 6th SECOND-PASS wave (HEAD 837f528) — 4 more real bugs (batch/coordination/translator-core) -->

### Session 2026-06-14f — 6th §11.4.118 second-pass wave (HEAD 837f528) — 4 more real bugs

- **53b5056** batch: directory batch to a non-existent output dir collided ALL files onto one path (only last survived) — data loss; fix: extensionless non-existent OutputPath treated as a directory.
- **df8a8cb** coordination: TranslateWithRetry ignored ctx cancellation (full sweep+sleep on cancelled ctx); TranslateWithConsensus only scanned the first requiredAgreement instances → skipped available-and-agreeing instances behind unavailable ones. Both fixed.
- **837f528** translator: successful-but-empty translation ('', nil) silently OVERWROTE source content across metadata/chapter/section/subsection (7 fields) — 'tests pass but book destroyed' data loss; fix: keepIfTranslationEmpty preserves source when translation empty + source non-empty. Latent flag: detected source-lang not propagated to llm translator.

**SESSION GRAND TOTAL: 64 real bugs** (5 parallel waves + 3 inline + 6 second-pass waves) + 2 reconciliations + §11.4.98 conversion. Post-wave full sweep GREEN at HEAD 837f528: 54 ok / 0 FAIL (fullsweep_20260614T114733.log).

<!-- session 2026-06-14e: 5th SECOND-PASS wave (HEAD 367d88b) — 5 more real bugs incl CWE-208 timing oracle -->

### Session 2026-06-14e — 5th §11.4.118 second-pass wave (HEAD 367d88b) — 5 more real bugs (grpc/security/preparation)

- **45f6152** grpc: GetTranslationStatus returned codes.Unknown for unknown session (→ NotFound); cleanupOldSessions leaked session timeout context (didn't call CancelFunc → 24h timer/goroutine leak). 
- **5871848** security: CWE-208 username-enumeration TIMING oracle — no bcrypt compare on user-not-found path → ~1.7M× faster than wrong-password → reveals valid usernames. Fix: dummy-bcrypt compare at init (OWASP constant-time). 
- **367d88b** preparation: failed-chapter analysis MISATTRIBUTION (positional index after results compaction → chapter got wrong chapter's analysis; fix: lookup by ChapterNum); nested-subsection content DROPPED from analysis input (extractors ignored Section.Subsections; FB2 populates them; fix: recursive writeSectionContent).

**SESSION GRAND TOTAL: 60 real bugs** (5 parallel waves + 3 inline + 5 second-pass waves) + 2 reconciliations + §11.4.98 conversion. Post-wave full sweep GREEN at HEAD 367d88b: 54 ok / 0 FAIL (fullsweep_20260614T113420.log).

<!-- session 2026-06-14c+d: two more parallel SECOND-PASS waves (HEAD 4a95b06) — 11 more real bugs, FD-safe -->

### Session 2026-06-14c/d — two more §11.4.118 second-pass waves (HEAD 4a95b06) — 11 more real bugs; 3-stream low-FD method proven repeatable at ~95% FD, NO ENFILE

Wave c (verification/storage/markdown) = 7 bugs; wave d (api/verifier/script) = 4 bugs + script came back genuinely CLEAN (honest §11.4.6, no fabricated bug). All conductor-reverified (build+vet+ -p 1 per pkg + full sweep -p 1) and pushed FF.

- **34e09f8** verification: parseNote dropped a note ending in EXAMPLES (no IMPLICATIONS); NaN Confidence (count==0 divide); multipass section under-count (pointer aliasing). 
- **9946d85** storage: Redis makeCacheKey metadata delimiter-injection collision (Ollama 'llama3:8b' has ':') → wrong translation; fixed NUL-joined hash.
- **d055b8c** markdown: EPUB↔MD code blocks destroyed, GFM tables lost both ways, backslash-escapes corrupted — all fixed bidirectional.
- **b8bc3ec** api: translateText silently dropped script=cyrillic + accepted invalid script (→ validate+switch); verifier handlers mapped all GetModel errors to wrong status (→ errors.As 404-vs-503).
- **4a95b06** verifier: buildFallbackChain rand.Shuffle destroyed score order + nondeterministic (→ keep FilterVerified order); OpenRouter pricing typed float64 but live API emits quoted strings → whole Tier-2 decode failed → 0 OpenRouter models (→ json.Number). 

Honest flags (NOT fixed — ambiguous/spec-gated): verification polishWithNotes equal-chapter-count assumption; verifier run.go registry CanSeeCode/AffirmativeResponse + 3-tier merge precedence; api getStats nil-cache (not prod-reachable) + distributedManager-gated SSRF/null-JSON (integration-gated); storage dup-tuple unique-index (review-gated); script Convert short-circuit leaves stray off-script chars (deliberate pinned optimization). pkg/script second-pass = CLEAN (mature, all 30 letters map, digraph casing correct).

**SESSION GRAND TOTAL: 55 real bugs** (5 parallel waves + 3 inline + 4 second-pass waves) + 2 reconciliations + §11.4.98 conversion. Post-wave full sweep GREEN at HEAD 4a95b06: go test ./... -p 1 = 54 ok / 0 FAIL (fullsweep_20260614T112302.log). build+vet exit 0.

<!-- session 2026-06-14b: parallel SECOND-PASS discovery wave (HEAD 47ea69d) — 5 more real bugs, FD-safe -->

### Session 2026-06-14b — parallel §11.4.118 second-pass discovery wave (HEAD 47ea69d) — 5 more real bugs; ran 3 subagents at ~95% FD with low-FD discipline, NO ENFILE

Operator repeatedly requested parallel subagents; FDs never freed (~95%, 15 claude procs). Honored the informed repeated instruction with mitigations: 3 streams (not 4), investigation-first + targeted -p 1 tests only, no ./... , no -race during hunt. Result: NO ENFILE, 5 real bugs, conductor-reverified (build+vet+ -p 1 each pkg green), 3 commits pushed FF.

- **87a8048** — Anthropic client returned only response.Content[0].Text but the Messages API returns content as an ARRAY: long output split across text blocks was TRUNCATED, and a leading thinking/redacted_thinking block made Content[0].Text=="" → EMPTY translation. Fix: concatenate all text blocks (skip thinking/tool). httptest RED→GREEN→mutation.
- **edadd73** — distributed performance.go ResultCache.Set evicted on every update-while-full (didn't check key-exists) → shrank cache below maxSize, dropped valid entry. + coordinator.go round-robin currentIndex never reset on DiscoverRemoteInstances rebuild → index-out-of-range PANIC after worker pool shrinks (translation hot path). Both RED→GREEN→mutation (-p 1).
- **47ea69d** — ebook epub_parser.go built opfDir+href verbatim; EPUB OCF hrefs are percent-encoded/relative (RFC 3986) but zip entries are literal → 'chapter%20one.xhtml' never matched 'OEBPS/chapter one.xhtml' → chapter+cover SILENTLY DROPPED (data loss). Fix: resolveEPUBHref (PathUnescape + ./ ../ normalize). + html_parser.go removed a hardcoded 'Nestedtexthere'->'Nested text here' test-hack that corrupted any real doc containing that substring. RED→GREEN→mutation.

Honest §11.4.6 flags (NOT auto-fixed — ambiguous intent/operator-gated): Qwen endpoint↔response-shape mismatch (native DashScope path vs OpenAI-compatible config base-URL — needs operator decision on intended Qwen mode); coordinator reduced_quality fallback lowercases passthrough text; FB2 writer section-title flatten. Markdown→EPUB path clean-verified (lists/blockquotes/fences/headers all handled). EPUB writer clean-verified.

**SESSION GRAND TOTAL: 44 real bugs** (5 parallel waves + 3 inline + this 5-bug second-pass wave) + 2 reconciliations + §11.4.98 conversion. Post-wave full sweep GREEN at HEAD 47ea69d: `go test ./... -p 1` = **54 ok / 0 FAIL** (host-safe -p 1 avoided ENFILE at ~95% FD); evidence qa-results/full_sweep/fullsweep_20260614T104630.log. Build+vet exit 0.

**Purpose:** Single canonical out-of-the-box entry point for any fresh session (§11.4.131 / §12.10 / §11.4.127). To resume: point a new session at THIS file, run `git fetch --all`, and say **continue**.

---

<!-- session 2026-06-14a: host-safe single-stream inline (HEAD bd8a1ef) — 2 more real bugs while FD-pressured -->

### Session 2026-06-14a — host-safe single-stream inline (HEAD bd8a1ef) — 2 more real data/correctness bugs in pkg/translator/llm (the chunking path subagents skipped)

Context: operator returned, asked for 3-4 parallel subagents; chose "free FDs first" but host `kern.num_files` stayed ~94-95% of maxfiles (13+ concurrent claude processes, NOT freed). Per §12 host-safety I did NOT launch parallel subagents (ENFILE risk); instead ran host-safe single-stream INLINE work (targeted -p 1 tests never hit ENFILE all night) and armed a background watcher (b3qgjr30s) to auto-launch the parallel wave the instant FD<80%. 2 commits, pushed FF.

- **24b0fd0** — splitText DROPPED PARAGRAPH SEPARATORS at chunk boundaries (the >20KB size-error retry path subagents had skipped as out-of-scope). It stripped "\n\n" via Split + re-added inconsistently (never around an oversized para, never across a chunk seam); reassembly strings.Join(chunks,"") then glued the last paragraph of one chunk to the first of the next; "\n\n\n\n" collapsed to "\n\n". Structural data loss in translated large chapters. Fix: splitText now tiles losslessly — re-attach each para's "\n\n" so Join(splitText(text),"")==text (makes reassembly correct by construction). RED (5 cases dropped \n\n) -> GREEN -> mutation polarity proven.
- **bd8a1ef** — in-memory cache-key COLLISION served wrong translation: key was fmt.Sprintf("%s:%s",text,contextStr); ":" inside text means ("a:b","c") and ("a","b:c") both -> "a:b:c" -> 2nd request served 1st's translation (same class as the storage Redis fix). Fix: makeLLMCacheKey injective via length prefix "%d:%s:%s". RED (Translate served xlate(a:b) for input "a") -> GREEN -> mutation-proven (helper body).
- **71babe1** — FB2 writer DROPPED deeply-nested section content (>2 levels): chapter loop only descended sec.Content + direct sub.Content, never sub.Subsections (depth 3+) -> deep translated text vanished from the .fb2 (EPUB writer's recursive formatSection preserves it). Fix: collectSectionParagraphs recurses the full Subsections tree. RED (real write->read: LEVEL_3/_4 missing) -> GREEN -> mutation-proven. Honest note (§11.4.6, NOT fixed — ambiguous intent): FB2 writer still flattens section/subsection TITLES to chapter level (sec.Title not written); content now lossless.
- EPUB writer (pkg/ebook/epub_writer.go) §11.4.6 clean-verified: escapeXML correct+complete, OPF/NCX/chapter filenames consistent, shared identifier, Chapter has no orphan Content field — no data-loss; no bug invented.

**SESSION GRAND TOTAL now: 39 real bugs** (waves 1-5 = 32 + wave-5 4... see below) + 2 reconciliations + §11.4.98 conversion. Build STABLE GREEN, `go build ./...` exit 0; pkg/translator/llm passes offline. Watcher b3qgjr30s armed (40min cap) → auto-launch 3-4 parallel subagents for a §11.4.118 second-pass discovery wave the moment FD<80%. While FD-pressured, conductor continues host-safe single-stream inline bug-hunting.

<!-- session 2026-06-13g: bug-hunt wave 5 + STABLE GREEN BUILD + §12 host-safety pace (HEAD ceab1db) -->

### Session 2026-06-13g — bug-hunt wave 5 (HEAD ceab1db) — 4 real bugs incl. CWE-22 path traversal + PairingManager race; build STABLE GREEN; paced for §12 host-safety

Mode: §11.4.126 autonomous loop + §11.4.103 3 parallel subagents (PairingManager mutex · pkg/language · pkg/api) + conductor verify-then-commit. 3 commits (38300e4..ceab1db) pushed FF.

- **38300e4** — pkg/distributed PairingManager.services map DATA RACE (no manager mutex; map + *RemoteService.Status/LastSeen/PairedAt written by health goroutines, read by GetPairedServices) → one RWMutex, deadlock-averse (unlocked-internal addService helper, no re-entrant lock, lock never held across HTTP probe / event Publish; performHealthChecks snapshots under RLock then releases). RED(-race)→GREEN(-race -count=5 ~4s no hang)→mutation; full pkg -race ok ~52s. Resolves the wave-4 flagged latent.
- **5c1a2ae** — pkg/language 2 UTF-8 bugs: detectHeuristic sampled text[:1000] by BYTE → under-counts trailing multi-byte script → majority-Cyrillic book mis-detected as English (→ 1000 RUNES); FormatLanguageCode code[:2] by BYTE → invalid UTF-8 on multi-byte char (→ 2 RUNES); + DetectLanguage prompt sample rune-fix (source-only, network-gated). Mutation-proven, -race ×3.
- **ceab1db** — pkg/api PATH TRAVERSAL (CWE-22): uploadUpdate/applyUpdate interpolated the X-Update-Version header into a filepath.Join, '..' escaped the update dir → arbitrary file write. → validateUpdateVersion allowlist (reject ''/'..'/separators/NUL/>128, charset [A-Za-z0-9.+_-]). RED(200+escaped path)→GREEN(400)→mutation; existing 11 subtests pass, -race ×3.

**BUILD STATE — STABLE GREEN.** Every package passes: the post-wave-5 full sweep showed 52 ok + 2 transient FAILs (pkg/translator TestMemoryUsage + test/integration build) that were proven ENVIRONMENTAL (§11.4.7), NOT code: error was literally "too many open files in system"; `go test -c ./test/integration` compiles exit 0; both re-ran GREEN at `-p 1`. Root cause: host `kern.num_files` at ~95.8% of `kern.maxfiles` (122880) from 5 waves of parallel subagents + repeated full `go test ./...` + a concurrent claude security-review process. Last fully-clean authoritative sweep: HEAD de25a98 = 54 ok / 0 FAIL (qa-results/full_sweep/fullsweep_20260613T230845.log). `go build ./...` exit 0 throughout.

**§12 HOST-SAFETY DECISION (§11.4.101 safest/most-stable):** host file table near saturation → STOP spawning more parallel subagents / full-repo builds until it recovers (adding load risks tipping past maxfiles = other workloads fail to open files = the destabilization §12 forbids). The autonomous-safe disjoint package queue is also essentially EXHAUSTED — the whole codebase surface has been hunted across 5 waves (llm, websocket, coordination, fb2, ebook[epub/html/docx], batch, cmd, verifier, preparation, storage, report, markdown, format, grpc, security, hardware, version, progress, services, config, cache, translator-core, distributed, language, api). Per §11.4.94(A) the remaining items are genuinely blocked → paced re-verification, not idle-by-laziness.

### SESSION GRAND TOTAL (2026-06-13 → 06-14): 36 real bugs fixed across 5 waves (7+10+7+8+4) + 2 reconciliations + §11.4.98 llm offline conversion
All reproduce-first + mutation-proven + conductor-independently-reverified (-race) + 24 descriptive commits pushed FF to milos85vasic/Translator + HelixDevelopment/HelixTranslate (no force §11.4.113). Notable: CWE-204 auth enumeration oracle, CWE-22 path traversal, gRPC remote-DoS nil panic, websocket cross-session event leak, FB2/markdown/ebook round-trip data loss, cache+pairing+versionCache+ssh data races, UTF-8 mid-rune corruption (translator+language).

Flagged latent (review/operator-gated, NOT autonomously safe overnight): storage cache dup-tuple unique-index (schema/dedup-semantics change §11.4.124); DOCX unioffice license-gated (go.mod dependency decision); security RefreshToken re-validate (no caller, defense-in-depth); G1 verify→server-DB bridge; G2 batched ~30-provider sweep; G3 add OPENAI/ANTHROPIC keys to api_keys.sh. These need operator decisions (user away) — correctly deferred per §11.4.101/§11.4.122.

<!-- session 2026-06-13f: parallel bug-hunt wave 4 + FULL-REPO SWEEP GREEN (HEAD de25a98) -->

### Session 2026-06-13f — bug-hunt wave 4 + full-repo sweep GREEN (HEAD de25a98) — 8 real bugs; `go test ./...` = 54 ok / 0 FAIL offline

Mode: §11.4.126 autonomous loop + §11.4.103 4 parallel subagents (internal/services+config · internal/cache · pkg/translator core · pkg/distributed deeper) + conductor verify-then-commit + post-wave full sweep. 4 commits (eeda1ee..de25a98) pushed FF.

- **eeda1ee** — services/config 3 bugs: GetPreferences claimed sorted-by-score but did NO sort (passed only on pre-sorted fixture) → sort desc; FallbackOrder used unfiltered index → first accepted pref got 2 not 1 → contiguous post-filter rank; config all-zero scoring weights fail-open ('total>0 &&' skipped the sum check) → reject. Mutation-proven.
- **9d4f4a4** — internal/cache cleanup goroutine LEAK: NewCache(_,true) launched 'for range ticker.C' with no Close/stop → one goroutine + pinned Cache leaked per instance forever (unbounded on a long-running server). Added stop chan + Once-guarded Close() + cleanupWG; backward-compatible. Mutation-proven (200 caches→200 leaked).
- **c1bd61f** — pkg/translator universal.go language-detection sample sliced sample[:2000] by BYTE → mid-rune split → invalid UTF-8 to Detect (common case for Cyrillic/CJK ebooks) → truncateOnRuneBoundary. Mutation-proven, drives real detector.
- **de25a98** — pkg/distributed 3 bugs: checkServiceHealth set Status='online' unconditionally → every paired worker DEMOTED+dropped after first 30s tick → all work distribution lost (HIGH); versionCache map had NO mutex (existing one is AlertManager's) → BatchUpdateWorkers data race → cacheMu; ssh LastUsed two-locks-one-field race → conn.mu touch()/idleSince(). Mutation-proven, full pkg -race ok.

**STABILITY MILESTONE (re-confirmed):** `go test ./... -count=1` all provider env unset = **54 ok / 0 FAIL**, `go build ./...` exit 0. Evidence: qa-results/full_sweep/fullsweep_20260613T230845.log. SESSION GRAND TOTAL: 32 real bugs (waves 1-4: 7+10+7+8) + 2 reconciliations + §11.4.98 llm offline conversion, all reproduce-first + mutation-proven + conductor-reverified (-race) + pushed FF (no force §11.4.113).

Flagged latent (honest §11.4.6, NOT autonomously fixed — review/larger-blast-radius): PairingManager.services map needs a manager-level mutex (~10 methods); storage cache dup-tuple unique-index (schema/dedup-semantics change); DOCX unioffice license-gated (go.mod); security RefreshToken re-validate (no caller). Operator-gated: G1/G2/G3.

Next-wave queue (disjoint, autonomous-safe, thinning): pkg/language, pkg/api (deeper non-integration logic), the PairingManager mutex (careful single-mutex + -race). After that the autonomous-safe disjoint queue is largely exhausted → legitimate idle per §11.4.94.

<!-- session 2026-06-13e: parallel bug-hunt wave 3 + FULL-REPO SWEEP GREEN (HEAD 8d0b1b6) -->

### Session 2026-06-13e — bug-hunt wave 3 + full-repo sweep GREEN (HEAD 8d0b1b6) — 7 real bugs + 2 interaction-defect reconciliations; `go test ./...` = 54 ok / 0 FAIL offline

Mode: §11.4.126 autonomous loop + §11.4.103 4 parallel subagents (grpc · security · ebook html/docx · hardware/version/progress) + conductor verify-then-commit. Then a full-repo `go test ./...` sweep (§11.4.40-style interaction check) caught 2 latent defects, both reconciled. 6 commits (6e1b1f2..8d0b1b6) pushed FF.

- **6e1b1f2** — grpc 2 bugs: StartTranslation deref'd req.ProviderConfig.Type unconditionally (proto3 optional → nil panic, no grpc recovery → remote-DoS CRASH the serving goroutine) → validate up front, InvalidArgument; session-limit TOCTOU (RLock-check then Lock-insert) + duplicate SessionId silently overwrote prior session WITHOUT CancelFunc (timeout-context leak + orphaned translation) → atomic gate+reserve, AlreadyExists. Mutation-proven over bufconn, -race ×2.
- **96999b3** — security: AuthenticateUser checked IsActive BEFORE password → username/account-status enumeration oracle (CWE-204; API maps to 403 vs 401) on the live login path → validate password FIRST, disclose inactive only with correct password. Also fixed a §11.4.1 JWT-tamper test FAIL-bluff (mutated last sig char, ~1/64 flaky → first char, 200/200). RED(exploit)→GREEN→mutation, -race ×3, downstream pkg/api green.
- **95323d8** — ebook HTML: td/th/dt/dd cells concatenated with no separator ('cell Acell B') + <br> dropped ('line oneline two') → isCellElement newline + <br> newline. Mutation-proven. HONEST PENDING_FORENSICS: DOCX parser depends on license-gated unioffice → non-functional for real DOCX (go.mod-scope, not fixed).
- **044bacb** — hardware: vm_stat malformed page-size discarded parse error → 0 available RAM → guard (err==nil && >0); meminfo kB*1024 uint64 overflow wrap → maxKB guard. progress: GetProgress clobbered Complete()'s 'Completed' ETA → gate override on <100% && status!='completed'. Mutation-proven.
- **8d0b1b6** — full-sweep interaction reconciliations (§11.4.120/§11.4.98): test/unit scoring expectations updated to the FIXED weight formula (0.78→0.675, 0.625→0.5, math-verified, not a revert); cmd/cli TestTranslateEbookFunction/with_app_config dialed real Qwen (env-dependent) → honest §11.4.3 SKIP when no provider key.

**STABILITY MILESTONE:** `go test ./... -count=1` with ALL provider env unset = **54 ok / 0 FAIL / 24 no-test-files**, deterministic offline. `go build ./...` exit 0. Evidence: qa-results/full_sweep/fullsweep_20260613T225207.log. Session total this run: 26 real bugs (waves 1+2+3) + 2 reconciliations, all reproduce-first + mutation-proven + conductor-reverified + pushed FF (no force §11.4.113).

Next-wave queue (disjoint, autonomous-safe): internal/services, internal/config, internal/cache, pkg/translator (deeper, non-llm), pkg/distributed (deeper pure-logic), cmd more. Carry-over latent (review-gated, NOT autonomously fixed): storage dup-tuple unique-index; DOCX unioffice license (go.mod); security RefreshToken re-validate. Operator-gated: G1/G2/G3.

<!-- session 2026-06-13d: parallel subagent bug-hunt wave 2 (HEAD 39ee07a) -->

### Session 2026-06-13d — parallel subagent bug-hunt wave 2 (HEAD 39ee07a) — 10 real bugs, all reproduce-first + mutation-proven + conductor-reverified (-race) + pushed FF (no force §11.4.113)

Mode: §11.4.126 autonomous loop + §11.4.103 4 parallel subagents (disjoint scopes: internal/verifier · pkg/preparation · pkg/storage+report · pkg/markdown+format) + conductor verify-then-commit. 4 descriptive commits (539659b..39ee07a) pushed FF to milos85vasic/Translator + HelixDevelopment/HelixTranslate.

- **539659b** — internal/verifier 3 bugs: scoring/engine.go applied weights.Capability TWICE + DROPPED Recency (sum 1.15, broke 0..1 contract); discovery HF models keyed on legacy `modelId` only → valid `id`-only payload collided every model on ID="" → ZERO usable models; registry FilterVerified/ListModels map-iteration nondeterministic order (latent flake). All mutation-proven, -race.
- **1bee190** — pkg/preparation 3 bugs: analyzeChapters appended goroutine results in COMPLETION order → GetTranslationContext[n-1] mis-attributed each chapter's analysis to the wrong chapter (→ slot-indexed, no mutex); multi-pass consolidation set FinalAnalysis from fresh parse with no chapter_analyses → all per-chapter analysis silently lost (→ re-attach); isUntranslatable strings.Contains(text,"")==true → blank LLM term marked EVERY string untranslatable incl. title (→ skip empty). Mutation-proven, -race ×3.
- **0a415b1** — pkg/storage negative CacheHitRate in GetStatistics (all 3 backends sqlite/postgres/redis): (totalAccess-totalTranslations)/totalAccess went negative when entries inserted-but-rarely-reread (3 entries,1 hit=-200%) → hits/(hits+misses), always [0,100). Mutation-proven via real SQLite round-trip.
- **39ee07a** — pkg/markdown 3 reader-visible EPUB-output bugs in the PRODUCTION chapter path: h4-h6 headers shipped as literal `<p>#### x</p>` (only 1-3 handled; §11.4.108 source→artifact — h4-6 lived only in a dead sibling fn); fenced code blocks flattened to one `<p>` with literal fences+collapsed newlines (→ `<pre><code>`); SimpleWorkflow wrote body/header text RAW → malformed XHTML on `&`/`<`/`>` (→ escapeXHTMLText). Mutation-proven.

Honest clean-verifications (§11.4.6): coordination/scoring/selection concurrency sound; openai/qwen/zhipu/llamacpp clients parse correctly; format detector edge cases correct; Redis cache-key+avg-duration sound (prior fixes intact).

**FLAGGED latent (NOT fixed — needs conductor/operator review per §11.4.124, schema change):** pkg/storage SQLite/Postgres cache has NO UNIQUE index on the lookup tuple (source_text,source_language,target_language,provider,model); storing the same tuple under 2 different ids leaves 2 rows and GetCachedTranslation can return the older STALE row. No production caller constructs cache IDs today (all in tests). Correct fix = UNIQUE index migration changing dedup semantics → review-gated.

Next-wave queue (disjoint, autonomous-safe): pkg/distributed deeper (conductor-owned §11.4.119), pkg/grpc, pkg/security deeper, pkg/ebook html/docx parsers, pkg/hardware, pkg/version/progress. Carry-over: storage dup-tuple unique-index (review-gated); G1/G2/G3 operator-gated.

<!-- session 2026-06-13c: parallel subagent bug-hunt wave (HEAD f451468) -->

### Session 2026-06-13c — parallel subagent bug-hunt wave (HEAD f451468) — 7 real bugs, all reproduce-first + mutation-proven + conductor-reverified (-race) + pushed FF (no force §11.4.113)

Mode: §11.4.126 autonomous loop + §11.4.103 4 parallel bug-hunt subagents (disjoint package scopes) + conductor verify-then-commit (§11.4.147/§11.4.142). Conductor independently re-ran build+vet+`-race` on every affected package and reviewed every production diff BEFORE committing. 6 separate descriptive commits (caa39c1..f451468), pushed FF to milos85vasic/Translator + HelixDevelopment/HelixTranslate.

- **caa39c1** — websocket `Hub.StartServer` SESSION FAN-OUT HOLE: /ws handler left `Client.SessionID` empty → hub's per-session filter skipped → every dashboard got EVERY session's events (real caller cmd/ssh-translation). Fixed (read session_id from query) + moved off global DefaultServeMux→private mux (dup-/ws panic). Mutation-proven (alpha client saw beta's BETA-LEAK frame), -race ×3.
- **91e71c4** — fb2 `Paragraph` had UnmarshalXML but NO MarshalXML (Text/FullText are `xml:"-"`) → parse→write round-trip emitted empty `<p></p>` = TOTAL paragraph data loss. Added MarshalXML (FullParagraphText, id/style attrs, escaped once). Mutation-proven.
- **ef8b3c7** — ebook EPUB fallback cleaner 2 bugs: CleanXMLData blind 2-char prefix rewrite corrupted valid entities (&amp;→&amp;mp;) + text (Q&A); removeHTMLTags regex EXCLUDED self-closing/void tags (<br/> <img/> <hr/>) → leaked literal markup into chapter text. Fixed (escapeBareAmpersands + unified htmlTagRe, RE2-safe). Mutation-proven; stale test reconciled §11.4.120.
- **7e05c5e** — batch `Process(ctx)` NEVER honored cancellation (no ctx.Err/Done in either loop) → ran every remaining file, error stayed nil. Added seq guard + parallel in-worker short-circuit + post-Wait surfacing. Mutation-proven ×2, -race ×3.
- **3225c8d** — cmd UPPERCASE/mixed-case ext not stripped: `TrimSuffix(base, ToLower(ext))` left "Story.EPUB"→"Story.EPUB_sr.epub" (case-insensitive FS). Fixed in unified-translator (3 helpers) + translator. Mutation-proven; stale test reconciled §11.4.120; cmd/translator gained first tests.
- **f451468** — §11.4.98 offline-ize: openai/qwen/zhipu error-paths → httptest; llamacpp auto-download → temp-HOME stub .gguf (no HF dial); HF_TOKEN skip-guards; TestValidate AvailableRAM pinned. Whole pkg/translator/llm now passes OFFLINE ~8s, all provider env unset, mutation-checked each converted test still has teeth. No prod client bug (honest §11.4.6).

Honest clean-verifications (§11.4.6 — investigated, NO bug): pkg/coordination (lock ordering leaf-only, consensus chan cap≥goroutines, immutable instances post-init); openai/qwen/zhipu/llamacpp clients parse correctly (defect was purely tests dialing real APIs).

Next-session queue (disjoint, autonomous-safe): bug-hunt internal/verifier (scoring/discovery/selection/client), pkg/preparation, pkg/storage (deeper), pkg/distributed (deeper pure-logic), pkg/markdown/format/html/docx parsers deeper. Carry-over G1 (verify→server-DB bridge), G2 (batched ~30-provider sweep), G3 (operator: add OPENAI/ANTHROPIC keys to api_keys.sh).

<!-- session 2026-06-13b: parallel subagent bug-hunt wave -->

### Session 2026-06-13b — parallel subagent bug-hunt wave (HEAD 86175d0) — ~15 real bugs, all reproduce-first + mutation-proven + -race + pushed FF (no force §11.4.113)

Mode: §11.4.126 autonomous loop + §11.4.103 ≥3 parallel streams (conductor + 4–5 background bug-hunt subagents/wave). §11.4.147 exercised heavily — multiple subagents were rate-limit-crashed ("Server is temporarily limiting requests") and respawned until complete; conductor independently verified every subagent claim (-race + scope census + §11.4.142 diff review) BEFORE committing. Commits pushed to milos85vasic/Translator + HelixDevelopment/HelixTranslate, fast-forward, no force.

- **a40a9b9** — logger `shouldLog` fail-open (typo level "warning" → logged everything incl DEBUG in prod) + fail-drop (unknown msg level dropped) → `levelRank` fallback (cov 84.3→91.1%); progress tracker 5 bugs (GetProgress data race, Complete() 0% clobber, negative %, ETA float truncation, divide-by-zero PANIC); ollama_test 4 subtests dialed real httpbin.org → httptest (6.6s→0.3s offline) §11.4.98.
- **3a7ae78** — gRPC EventBus handler LEAK (SubscribeEvents never unsubscribed → unbounded `allEvents` growth) → added `SubscriptionID`+`Unsubscribe`+`HandlerCount`, defer Unsubscribe, drop close; + pre-existing session-field DATA RACE (sessionsMutex guarded only the map) → locked Status writes, snapshot-under-RLock GetTranslationStatus, fixed latent recursive-RLock deadlock in ListTranslations. Mutation-proven (events + real bufconn leak test), -race ×3.
- **a2aa3da** — report ReportGenerator data race (unsynced appends) → sync.Mutex + snapshot-under-lock; deepseek/anthropic/gemini tests dialed real api.* → httptest (package 30.6s→19.2s offline) §11.4.98.
- **ef0a60f** — verification multipass O(n²) DUPLICATE DB WRITES (polishWithNotes re-saved the whole accumulated report.SectionResults each chapter → polishing_changes got 1+2+…+N dup rows; section_results PK-conflict inserts failed silently) → `saveNewResults` cursor. Mutation: 15 change rows for 5 chapters vs 5.
- **916dddd** — storage Redis cache-key HASH COLLISION (32-bit poly-31 → distinct texts shared a key → WRONG cached translation served) → sha256; + Redis avg-duration dilution → `accumulateAvgDuration`. Mutation-proven.
- **86175d0** — markdown→EPUB ROUND-TRIP DATA LOSS: links/images/blockquotes left literal in HTML AND the production `convertMarkdownToXHTML` chapter path (readers got `[text](url)` / `&gt;`) → blockquote accumulation + image/link inline conv (image-before-link). `<ol>` start renumber pinned KNOWN-LOSSY. 4 bugs mutation-proven.

Honest clean-verifications (§11.4.6 — investigated, NO bug): gRPC StreamTranslationProgress (streams-map delete-under-lock + non-blocking send ⇒ no send-on-closed); preparation extractJSON (string-aware matchBalanced + json.Valid guard + fenced handling robust; crashed preparation subagent left only scratch, removed §11.4.84).

REMAINING NET §11.4.98 offenders (still dial real api.* — confirmed via keys-stripped per-test timing): **openai** (TestOpenAITranslateErrorPaths), **qwen**, **llamacpp** (HF download paths), **zhipu** invalid_model. Same httptest pattern → strong next-wave item.

Next-session queue (disjoint, autonomous-safe): finish NET httptest-ization (openai/qwen/zhipu/llamacpp); bug-hunt cmd/* entrypoints, pkg/websocket, pkg/coordination, pkg/batch, pkg/fb2, pkg/ebook parsers/writers, pkg/distributed deeper, internal/verifier. Carry-over G1 (verify→server-DB bridge), G2 (batched ~30-provider sweep), G3 (operator: add OPENAI/ANTHROPIC keys to api_keys.sh).

<!-- W13/W15/W16 wrap-up -->

### Session wrap-up — W13, W15 (api/grpc), W16 all resolved (HEAD 1cc66ac)
- **W15 api/grpc DONE (real, evidence-backed)** — subagent-driven. Built build-tagged integration suites: `pkg/api/server_realhttp_integration_test.go` (real HTTP via httptest + real JWT + real Postgres via brokertest: health, JWT 401/200, login→token round-trip, /api/v1/verified-models, session persist round-trip) and `pkg/grpc/server_storage_integration_test.go` (real gRPC over bufconn + real Postgres: submit→persist, status, cancel, list-count, 8× concurrent, error-code via status.FromError; §1.1 mutation-proven). LLM-translate leg = honest §11.4.3 SKIP (no provider key). Normal `go test` boots NOTHING; SKIP-clean without podman.
- **REAL AUTH BUG fixed (W15 api leg, commit 0e2a999)** — `InMemoryUserRepository.Create` stored the caller's pointer; `CreateUser` clears `Password=""` post-Create → blanked the stored bcrypt hash → EVERY user registered via CreateUser could never log in. Fix: repo stores a COPY. Conductor-reproduced RED(401)→GREEN(200) on `TestAPIRealHTTP_LoginTokenRoundTrip` (§11.4.115). Remaining W15: only the pure LLM-translation E2E (operator-gated on a real provider key).
- **W13 DONE (commit 1cc66ac)** — 8 PascalCase `.gitmodules` section labels → lowercase (paths were already compliant). Done §9.2 backup-first; all 11 submodules verified resolving to identical baseline SHAs after `git submodule sync`, dirty challenges/helix_qa preserved, build exit 0, backup removed post-gate.
- **W16 RESOLVED = KEEP Models/ as-is** — the `pkg/modelsbridge` bridge already consumes `digital.vasic.models` (functional need met). Converting `Models/` → a git submodule requires creating a NEW upstream repo under an owned org — an operator-gated repo-creation action (§11.4.101 block-only-when-irreversible+undeterminable; §11.4.122 keeping is the safe no-change path). Convert path remains available on operator request.

**Autonomous-safe queue now EMPTY.** Genuinely-remaining = operator-gated only: W15 LLM-translation E2E (real provider key) + W16 convert (new upstream repo decision). Per §11.4.94 this is legitimate idle.


<!-- LLMsVerifier real-key integration -->

### LLMsVerifier real-key integration — DONE (HEAD e7d6d67; submodule 95fb1cce)
Operator directive: use `~/api_keys.sh` so the LLMsVerifier submodule performs its mandatory steps and provides valid providers + models to the System. **Done, evidence-backed (§11.4.83 docs/qa/llmsverifier_20260613_182600/, key-free):**
- Ran the submodule's REAL discovery/verification with real keys → **89 verified models** (DeepSeek 2, Cerebras 2, Groq 16, Mistral 69; OpenAI/Anthropic honestly fail — no key in api_keys.sh).
- **Two real bugs found + fixed, both §11.4.135 mutation-proven guards (exact production errors):**
  - **Bug B (main, internal/verifier/client.go) load-bearing**: the System's SSOT client couldn't decode the server's `{"count":N,"models":[...]}` envelope → consumed ZERO models while server healthy. Fixed (envelope-aware decode). Guard: client_envelope_test.go. System now fetches all 89 e2e.
  - **Bug A (submodule llmverifier/models.go, 95fb1cce, pushed GitHub+GitLab)**: numeric `context_window` (Groq) crashed the whole provider decode → 0 models. Fixed (tolerant UnmarshalJSON). Guard: context_window_unmarshal_test.go.
- Secrets: keys sourced inline only, never printed/logged/committed (§11.4.10); evidence dir + edits secret-scanned clean; throwaway run artifacts removed (§11.4.14).

**Honest follow-up gaps (tracked, not bluffed — §11.4.6):**
- (G1) Submodule `verify` (report) pipeline does NOT persist into the `server` SQLite DB — the served DB was seeded via live discovery (verifier's own client+database pkgs). Architectural bridge worth a workable item.
- (G2) Full ~30-provider exhaustive per-model coding-capability scoring timed out at 600s in one pass; reliability-scored the seeded set. A bounded/batched sweep is the follow-up.
- (G3) Providers without keys (OpenAI/Anthropic/Perplexity/Together) fail auth honestly; add their keys to api_keys.sh to include them.


<!-- bug-hunt waves 2026-06-13 -->

### Real-bug-hunt waves (HEAD eb7576e) — 17 real bugs fixed this session, all reproduce-first + mutation-proven + pushed
LLMsVerifier real-key integration: 89 verified models flow to the System end-to-end; System's own in-process verifier env-bridge (31 models, cmd/verify-models); real DeepSeek translation proof ("The sky is blue today."->"El cielo esta azul hoy.").

Bugs fixed (each RED->GREEN, §11.4.135 guard, §1.1 mutation-proven, real evidence; commits e7d6d67/dfb250c/f80153b/eb7576e + submodule 95fb1cce):
- Auth lockout (CreateUser blanked stored bcrypt hash); Verifier consume (client.go envelope); Verifier serve (listVerifiedModels dropped all); Groq context_window (submodule); Hardcoded RU->SR prompt (RU->SR Ekavica preserved); Uppercase Cyrillic digraph (LJUBAV); Ordered-list numbering lost; UTF-8 mid-rune truncation (HIGH, Cyrillic); Import data race; FB2 inline+tail text dropped (CRITICAL); FB2 writer escaped markup; FB2 format misdetection; EPUB OPF/NCX id mismatch; Markdown inline whitespace collapse; silent chapter drop; list reverse round-trip; extractJSON wrong span.

Remaining tracked follow-ups (honest, §11.4.6/§11.4.118):
- G1: submodule verify-pipeline->server-DB bridge (serial submodule wave, §11.4.119 contends with main builds; conductor-owned).
- G2: batched full ~30-provider verification sweep (one-pass timed out 600s).
- G3: add OPENAI/ANTHROPIC/etc keys to api_keys.sh (operator) to include those providers.
- Deferred-broader: markdownToHTML round-trip edges; preparation extractJSON LLM-output spec; verification multipass duplicate DB writes; ambiguous nj/lj/dz Latin->Cyrillic (structural §11.4.112 won't-fix).


<!-- wave3 providers/events/coord/security -->

### Bug-hunt wave 3 (HEAD 6c76c3f) — 6 more real bugs (session total ~23), all mutation-proven + pushed
- CRITICAL OpenAIClient panic on non-float64 temperature (blast radius = ALL OpenAI-compatible providers) — toFloat64/toInt coercion.
- CLI -temperature/-max-tokens ignored by openai/anthropic/qwen/zhipu — Options>typed-field>default precedence.
- Gemini discarded valid MAX_TOKENS partial text (data-loss) — accept STOP/MAX_TOKENS/"" (gemini_test.go reconciled §11.4.120).
- EventBus.Publish unbounded goroutine-per-handler + RLock-held-across-spawn → explosion + writer-starvation livelock — synchronous dispatch, lock released first (-race clean).
- LLMInstance.Available/LastUsed data race (mutex in 1/5 sites) — accessors (-race clean).
- APIKeyStore unsynchronized map → fatal concurrent-map panic (DoS) — RWMutex (-race clean). JWT/rate-limit confirmed already-hardened.

### NEW tracked finding (this wave)
- **pkg/translator/llm full `go test` HANGS when api_keys.sh is in-env** — a NON-tagged test reaches the network (§11.4.98 violation: unit tests must be deterministic/offline). The wave's new defect tests are httptest/deterministic; the hang is in a pre-existing test. FIX: build-tag or httptest-ize the offending test. (Bounded `go test -run <name>` works fine.)

### Remaining queue (for fresh-session continuation per §11.4.141/§11.4.127)
- G1: submodule verify-pipeline->server-DB bridge (serial submodule wave; §11.4.119 contends with main builds — conductor-owned, NOT parallel with main-repo subagents).
- G2: batched full ~30-provider verification sweep (one-pass timed out 600s).
- NET: fix the non-tagged network-reaching test in pkg/translator/llm (§11.4.98).
- G3: add OPENAI/ANTHROPIC/etc keys to api_keys.sh (operator) to widen provider coverage.
- More untouched packages to bug-hunt: pkg/storage (deeper), pkg/progress, pkg/report, pkg/logger, cmd/* entrypoints, pkg/grpc (the flagged send-on-closed-channel at server.go:403 — events pkg has no Unsubscribe; real follow-up).
- Deferred-broader: markdownToHTML round-trip edges; gemini GenerationConfig hardcoded (same class as fix #2); verification multipass duplicate DB writes.


## SHORT resumption sentence

> Read `docs/CONTINUATION.md` then `.remember/remember.md`, run `git fetch --all`, and **continue** the HelixTranslate "raise-to-enterprise" mandate at **Phase 4 (tests/Challenges/HelixQA to ~100% per type + expand real ebook-translation evidence)** while finishing the Phase-3 tail (Models decision, governance gaps). Binding: anti-bluff §11.4 (real captured evidence, no false PASS), no-force-push §11.4.113, flat snake_case submodules §11.4.28/§11.4.29, every change independently reviewed §11.4.142, containers-first infra §11.4.76, release tags prefixed `helix_translate-` (§11.4.151). Work subagent-driven (§11.4.70) with ≥3 parallel streams (§11.4.103).

## Live state anchors (moment-valid)

- **Parent HEAD:** see git log (W14b adoption commit via commit_all.sh) on `main`; pushed to BOTH `milos85vasic/Translator` + `HelixDevelopment/HelixTranslate` (verified, fast-forward, no force). Session commit chain: b3dc7f9(llm_provider) → e775088 → 8a6af67 → 7331c54 → 76677aa → 18c5137 → de256dd → 9004477 → 13d5e30 → af1ef47 → a9b38fa → 218bfa0 → 1f06bf5 → a0930fc → 3027da2 → b2377af → 6d4ae47.
- **llm_provider HEAD:** `b3dc7f9` (W7 CONST-036 propagation block + regenerated html/pdf), pushed to `HelixDevelopment/LLMProvider` master.
- **Constitution HEAD:** `5e671fe` (§11.4.151 added), pushed to all 6 upstreams; parent pointer bumped.
- **Build:** `go build ./...` = EXIT 0. **Total test coverage = 50.7%** (`go tool cover -func` total; see `docs/testing/coverage_matrix.md`).
- **Submodules (flat, snake_case):** `challenges containers constitution doc_processor docs_chain helix_qa llm_orchestrator llm_provider llms_verifier security vision_engine`.
- **Release prefix:** `HELIX_RELEASE_PREFIX=helix_translate` (`.env` git-ignored, `.env.example` tracked). First RC: `helix_translate-1.0.0-dev-0.0.1` — create ONLY when genuinely green (§11.4.40), operator-gated.
- **Safety backup:** `../helix_translate_gitbackup_20260611T181545Z/repo.git.mirror` (pre-rename `311c585`).

## DONE

**Phase 1 — flat snake_case submodules** (reviewed GO, pushed): 8 submodules renamed; Go pkg `challenges/`→`pkg/challenge_runner/`; `docs_chain` added; `.env` release-prefix. `go build ./...` EXIT 0.
**Phase 2 — constitution §11.4.151** release-prefix rule → pushed to all 6 constitution upstreams; parent pointer bumped; inheritance meta-test PASS.
**Phase 3 (partial):**
- `Security/` plain dir → flat `security` submodule (proven byte-identical to upstream, zero loss; §11.4.124).
- All submodules + main synced to every upstream (one stale `containers` gitlab mirror FF'd).
- **docs_chain OPERATIONAL**: `.docs_chain/contexts/tracked_docs.yaml` registered; real pandoc+weasyprint sync produced `docs/CONTINUATION.{html,pdf}` + `README.{html,pdf}`; `verify` = in-sync. See `docs/DOCS_CHAIN.md`.
- **Real translation PROVEN (anti-bluff)**: EN→Serbian via DeepSeek `deepseek-chat`, 208 Cyrillic chars, differs from source, no placeholders. Evidence: `qa-results/translation/<ts>/` (raw, git-ignored). Asset: `test/assets/crow_and_pitcher_en.txt`.
- **Fixed a regression I introduced**: challenge scripts' `PROJECT_ROOT="${SCRIPT_DIR}/../.."` broke when moved 1 level deeper → corrected to `../../..`; 3 challenges re-verified PASS (cache_invalidation, model_verification_gate, anti_bluff step1-3).

## CLOSED this session (2026-06-13, batch e775088 / llm_provider b3dc7f9 — all pushed, captured evidence)

- **W4** ✅ `pkg/security` flaky Short-TTL — widened TTL→1min; 50× + `-race` + CPU-load all PASS.
- **W5** ✅ `pkg/distributed` 120s timeout — bounded SMTP send (real defect) + in-process test servers; 3× ~48.5s, vet 0.
- **W6** ✅ `internal/verifier/discovery` 600s timeout — injectable URLs (prod unchanged) + sentinel stub + honest SKIP; 3× ~1s, paired §1.1 mutation proven.
- **W7** ✅ `llm_provider` CONST-036 propagation block — challenge GREEN; html/pdf regenerated.
- **W8** ✅ `anti_bluff_execution_challenge.sh` BSD-sed — portable tmpfile sed (§11.4.67); isolated rc=0 proof.
- **W9** ✅ tracked `.bak`/`.backup` residue — git-history-investigated (§11.4.124), removed + .gitignore; build 0.
- **W10** ✅ `-script latin`→Cyrillic — `pkg/script` was never imported into unified-translator; `normalizeScript` wired; RED→GREEN 7 tests + 3×.
- **W12** ✅ 8 challenge scripts brittle root — `git -C rev-parse --show-toplevel` w/ fallback (§11.4.111); all parse:OK root:OK.
- **D2** ✅ (wave2, 7331c54) `test/unit` intermittent FAIL — `TestVerifier/EventEmission` timing race (50ms Sleep vs goroutine-dispatched handler) + data race (proven `-race`). Mutex eventCollector + bounded poll. BEFORE 1/10 FAIL+race → AFTER `-race -count=10` 10/10 PASS.
- **W2-A** ✅ (wave2) `cmd/unified-translator` coverage 3.2%→18.0%, mutation-verified; + latent dangling-pointer fix (`Steps []TranslationStep`→`[]*TranslationStep`).
- **W2-B** ✅ (wave2) `pkg/storage` coverage 32.9%→35.3% (Redis cache-key helpers 0%→100%, real SQLite round-trips), mutation-verified, daemon paths honest-SKIP.
- **wave3** (18c5137): **W3** pkg/translator/llm stress/chaos + **FIXED real prod data race** in BaseTranslator cache+stats (RWMutex, -race ×3); **W2** sshworker 19.6%→39.7%; **W2** batch 77.2%→90.6%.
- **wave4** (de256dd): **sshworker progress-map leak** on ctx.Done() fixed (defer cleanup, RED 300/300→GREEN -race); **D3** pkg/translator{,/llm} test-only EventBus races fixed (asyncFlag); **D7** pkg/events TestEventBus_Subscribe race fixed; **W2** verification 64.0%→73.5%; **W2** script (already 100% stmt) +11 behavioral/digraph tests.
- **D5** (9004477) **FIXED real bug** pkg/verification parseNote: CONTENT-only note (no IMPLICATIONS) silently dropped → finalize Content; RED→GREEN, mutation-verified.
- **D6** (9004477) **FIXED** verified_factory map-order flake → order-independent ElementsMatch; 10/10.
- **wave5** (af1ef47): **W14** `scripts/commit_all.sh` wrapper (no-force/multi-upstream/FF-only/quiescence/explicit-pathspec/bg-push) + hermetic selftest 13/0 (conductor-run verified) — NOT yet wired to real remotes (line-by-line review pending); **W2** format 0%→97.6%, markdown 73.9%→80.1%, coordination 0%→92.9% + **GENUINE nil-guard fix** in TranslateWithRetry (mutation-verified: revert→nil-panic). §11.4.147: format+coordination were rate-limit-crashed at report stage; conductor verified+adopted their work (not lost, not blindly trusted).

**Session total: ~39 items resolved** (+ D10 pkg/distributed FULLY -race-clean 9→0 races [4 sources incl. FallbackManager.Stop() lifecycle fix]; + D9 hardware parser testability 44.6→68.4%)

**Prior: ~37 items** (+ models bridge wiring digital.vasic.models into the system; + D12 REAL CORS auth-bypass vuln fixed; + D11 doc)

**Prior: ~34 items** (+ W15: real Postgres+Redis on-demand integration proven; PG connection-pool DoS bug found+fixed; containers StartRedis helper added)

**Prior: 31 items** (+ wave7: grpc 0→53.3% bufconn, api 55→62.2% httptest, security +14 adversarial tests — all real-protocol/anti-bluff, auth confirmed hardened)

**Prior: 29 items** (+ wave6: websocket race-fix, hardware/distributed chaos tests, D10#2 api_logger PRODUCT data-race fix)

**Earlier session total: 26 items** (W4-W10,W12 + D2 + W2×6[unified-translator/storage/batch/sshworker/verification/script] + W3 translator + sshworker-leak + D3 + D7 + D5 + D6). Real prod bugs fixed: translator cache race, SMTP no-deadline, dangling step-ptr, sshworker map-leak, event races (D2/D3/D7), note-content loss (D5).

## OPEN workable items (queue — many parallelizable, disjoint scope)

| # | Item | Type | Notes / evidence |
|---|---|---|---|
| W1 | `Models/` (`digital.vasic.models`) orphan module | RESOLVED-via-bridge | Operator directed "create bridge": pkg/modelsbridge now CONSUMES digital.vasic.models (go.mod require+replace=>./Models), contract-tested 100%, mutation-verified (146b2b5). No longer orphan. W16 (convert to git submodule) still optional/operator-gated. |
| W2 | Coverage → ~100% (§11.4.27); ongoing | Task | Done: unified-translator 18%, storage 35.3%, batch 90.6%, sshworker 39.7%, verification 73.5%, script(100%+depth), translator/llm(+stress/chaos). Also grpc 53.3% (bufconn), api 62.2% (httptest), security 84.8%+adversarial, hardware/distributed chaos. NOTE: gRPC/api-server/server/redis/postgres at 0% are integration-infra-gated (need real daemons) — W15 containers-first unlocks these; honest deferral per package. |
| W3 | Missing test types: chaos, ddos, scaling, ui, ux, full-automation (in-module) | Task | §11.4.85 chaos/stress mandatory. |
| W11 | Wire more docs_chain contexts (Issues/Fixed/Status once they exist) | Task | docs_chain proven operational. |
| W13 | `.gitmodules` section labels still PascalCase | Task (cosmetic) | Needs per-submodule `.git/modules/*` gitdir move. |
| W14 | No project commit wrapper `scripts/commit_all.sh` (§11.4.22) | Task | IN-FLIGHT (wave5): multi-upstream push + no-force §11.4.113 + explicit-pathspec + quiescence + background-push §11.4.88 + hermetic self-test. Review before wiring to real remotes. |
| W15 | Containers-first infra (§11.4.76) | Feature | **storage DONE**: real Postgres+Redis on-demand integration via containers/pkg/brokertest (StartPostgres + new StartRedis), build-tagged; pkg/storage CRUD+chaos exercised real DBs; **found+fixed PG unbounded-pool DoS** (default 25/5). REMAINING: api/grpc DB+LLM *pipeline* paths need operator-gated LLM keys/SSH (can't autonomously). Pattern + helpers in place for any future slice. |
| W16 | Convert `Models/`→`models` submodule after W1 resolved | Task | Blocked on W1. |
| D1 | `docs_chain` + `security` CONSTITUTION.md lack literal `CONST-036` | Task | §11.4.118 discovery. They use §11.4.X cascade scheme (not CONST-NNN); challenge doesn't check them. Governance-design decision needed — do NOT blindly inject CONST-NNN (§11.4.6). |
| D4 | `pkg/script` Latin→Cyrillic morpheme-boundary mis-transliteration | Bug (open) | konjugacija→коњугација (should be конјугација); injekcija, nadживети similar. Greedy digraph matching; real fix needs morpheme/dictionary model (high blast radius) — operator-aware decision. Pinned with KNOWN-LOSSY tests. |
| D8 | ✅ FIXED (218bfa0) `pkg/markdown` bare-leading `---` data-loss | Bug (fixed) | Frontmatter now must begin at first non-blank line; later `---` = HR/chapter separator. RED(1ch)→GREEN(2ch), mutation-verified. Minor `<ol>`→dash numbering remains pinned/unfixed (D-list, low priority). |
| D9 | ✅ FIXED (fc66c62) hardware parser testability | Task | Extracted 12 pure parsers from exec callers (behavior-preserving); 44.6→68.4%, mutation-verified. |
| D10 | 3 baseline data races (FACT, captured -race) | Bug | #2 api_logger PRODUCT race ✅ FIXED (3027da2, snapshot, mutation-verified). #1 ssh_pool + #3 performance ConnectionPool. ✅ ALL FIXED (fc66c62): ssh_pool options API, perf locked-map + count-snapshot, FallbackManager.Stop() lifecycle. pkg/distributed now 0 races. |
| D11 | ✅ FIXED (b8a24f5) CLAUDE.md mislocated CORS under pkg/security | Task | Corrected: CORS is server-layer (cmd/server corsMiddleware + internal/config). |
| D12 | ✅ FIXED (b8a24f5) **CORS auth-bypass vuln** | Bug(security) | cmd/server corsMiddleware reflected ARBITRARY origin + Allow-Credentials:true under default `["*"]` → any site could make credentialed cross-origin requests. Fixed: wildcard→literal `*` no-creds; creds only for specific allowlist. RED→GREEN, mutation-verified. |
| W14b | ✅ DONE — reviewed + adopted | Task | Independent review: guards sound (force-reject full-argv, FF-only via merge-base --is-ancestor, multi-upstream URL-dedup, explicit-pathspec, quiescence). FOUND+FIXED cross-platform hardening: mutation grep lacked `-I` → GNU grep (Linux) false-aborts on binary `.pdf`/`.html` pathspecs (BSD grep doesn't; §11.4.81). Self-test 13/0. ADOPTED for normal source/doc commits. Dogfood surfaced finding #2: the §11.4.84 residue scan false-positives on files that legitimately CONTAIN marker strings (the script's own MUTATION_MARKERS def, governance docs, marker-referencing tests) — inherent to grep-based detection; §11.4.84 has no escape hatch (correct), so commit such files via manual explicit-path git. commit_all.sh is for the common case. |

## NEXT phases (priority order)
- **Phase 4** — drive W2–W8 + W10: per-type test suites to ~100% (§11.4.27), real Challenges + HelixQA bank execution with captured evidence, expand real ebook translations (all formats, whole chapters, anti-bluff §11.4/§11.4.69). Subagent-driven, ≥3 parallel streams.
- **Phase 5** — containers-first infra (W15); docs_chain context expansion (W11); full docs (guides/diagrams/SQL/templates) + enterprise Website.
- **Phase 6** — RC tag `helix_translate-1.0.0-dev-0.0.1` across all submodules + main ONLY when genuinely green (§11.4.40), operator-confirmed.

## Binding constraints
Anti-bluff §11.4 (real captured evidence, no false PASS) · no-force-push §11.4.113 (merge-onto-latest-main) · multi-upstream push §2.1 · flat snake_case §11.4.28/§11.4.29 · every change independently reviewed §11.4.142 · host-safety §12 (no suspend/poweroff) · no silent component removal §11.4.122 · containers-first §11.4.76 · release prefix §11.4.151 · subagent-driven default §11.4.70 · ≥3 parallel streams §11.4.103.

## Gotchas
- `/Volumes/T7` is **case-sensitive HFS+** (NOT the internal disk — verify on the actual volume, §11.4.6).
- No `scripts/commit_all.sh`; stage explicit paths, never `git add -A`; never `git add helix_qa` (nested third-party pointer drift).
- `PIPESTATUS` empty in this shell (zsh); use `$?` directly.
- Mutation challenges can leave `.go.backup`/`.bak` residue — scan + clean before commit (§11.4.84).
