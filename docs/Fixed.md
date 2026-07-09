# HelixTranslate — Fixed (Closed Workable Items)

**Revision:** 2
**Last modified:** 2026-07-08T12:00:00Z
**Authority:** §11.4.15 (status) · §11.4.16 (type) · §11.4.19 (column-alignment) · §11.4.33 (type-aware closure vocabulary) · §11.4.53 (Fixed_Summary parity) · §11.4.54 (ATM-NNN ticket IDs)
**Scope:** the closed-archive tracker. Every entry is a real, landed, in-git fix from this session's mutation-proven bug-hunt campaign + the format-matrix / PDF / DeepSeek / WebSocket / SSH work. Each carries a stable `[ATM-NNN]` id, `**Status:**`, `**Type:**`, a 1-2 line root cause, and the commit hash as captured evidence (§11.4.123 — no fix is claimed that is not in `git log`).

Sort order: ATM id ascending (allocation order). The companion `Fixed_Summary.md` sorts most-recent-first.

---

### §1. [ATM-001] splitText dropped paragraph separators at chunk boundaries causing structural data loss
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: large-chapter chunking concatenated chunks without re-inserting the paragraph separator at boundaries, silently flattening structure in the translated output. Mutation-proven.
- Evidence: commit `24b0fd0`

### §2. [ATM-002] In-memory translation cache-key collision served the wrong translation
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the in-memory cache key did not distinguish all inputs, so distinct (text,context) tuples collided onto one entry and a prior translation was returned for new content. Mutation-proven.
- Evidence: commit `bd8a1ef`

### §3. [ATM-003] FB2 writer dropped deeply-nested (>2 level) section content as translated-text data loss
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the FB2 writer recursion stopped serializing section content beyond two nesting levels, dropping translated text for deeply structured books. Mutation-proven.
- Evidence: commit `71babe1`

### §4. [ATM-004] Anthropic client dropped all content blocks after the first, truncating translations
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the Anthropic response parser read only `content[0]`, so multi-block responses were truncated or emptied. Mutation-proven.
- Evidence: commit `87a8048`

### §5. [ATM-005] Distributed ResultCache spurious eviction on update + round-robin index panic after pool shrink
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: cache update path evicted a still-valid entry, and the round-robin worker index was not re-bounded after the pool shrank, causing an out-of-range panic. Mutation-proven.
- Evidence: commit `edadd73`

### §6. [ATM-006] EPUB percent-encoded/relative hrefs dropped chapters and cover; HTML test-hack corrupted content
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the EPUB parser failed to resolve percent-encoded / relative manifest hrefs (losing chapters + cover) and a leftover hardcoded test-hack corrupted HTML content. Mutation-proven.
- Evidence: commit `47ea69d`

### §7. [ATM-007] Verification parseNote dropped EXAMPLES-terminated notes, produced NaN confidence, under-counted multipass sections
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: note parsing terminated early on EXAMPLES, divided by zero into NaN confidence, and the multipass section counter was off. Mutation-proven.
- Evidence: commit `34e09f8`

### §8. [ATM-008] Redis makeCacheKey metadata delimiter-injection collision served the wrong translation
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the Redis cache key concatenated metadata with a delimiter that could appear in the data, letting distinct inputs hash to the same key and return a wrong cached translation. Mutation-proven.
- Evidence: commit `9946d85`

### §9. [ATM-009] EPUB↔Markdown round-trip destroyed code blocks, lost GFM tables, corrupted backslash-escapes
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the markdown round-trip converter mishandled fenced code blocks, GFM tables, and backslash escapes, losing or corrupting content on the EPUB→MD→EPUB path. Mutation-proven.
- Evidence: commit `d055b8c`

### §10. [ATM-010] API translateText silently dropped script=cyrillic and accepted invalid script; verifier handlers returned wrong HTTP status
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the translate API ignored the cyrillic script flag and did not validate the script value; verifier handlers mapped every GetModel error to one wrong status code. Mutation-proven.
- Evidence: commit `b8bc3ec`

### §11. [ATM-011] Verifier fallback chain randomly shuffled (lost score order); OpenRouter pricing decode failed on string values
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the fallback chain was non-deterministically shuffled, discarding score-priority order, and OpenRouter pricing decoding assumed numeric JSON where the API returns strings. Mutation-proven.
- Evidence: commit `4a95b06`

### §12. [ATM-012] gRPC GetTranslationStatus returned the wrong error code; cleanupOldSessions leaked a timeout context
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: status lookup returned an incorrect gRPC code for missing sessions, and the session-cleanup loop never cancelled its per-iteration timeout context. Mutation-proven.
- Evidence: commit `45f6152`

### §13. [ATM-013] CWE-208 username-enumeration timing oracle in AuthenticateUser
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: authentication short-circuited on unknown usernames before the password hash compare, leaking account existence via response timing. Mutation-proven.
- Evidence: commit `5871848`

### §14. [ATM-014] Preparation failed-chapter analysis misattribution + nested-subsection analysis content loss
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: a failed chapter analysis was attributed to the wrong chapter index and nested subsection content was dropped from the analysis pass. Mutation-proven.
- Evidence: commit `367d88b`

### §15. [ATM-015] Directory batch to a non-existent output dir collided all files onto one path (data loss)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: batch directory translation did not create the output directory and computed a single colliding output path, so every file overwrote one another. Mutation-proven.
- Evidence: commit `53b5056`

### §16. [ATM-016] Coordination TranslateWithRetry ignored ctx cancellation; consensus skipped available instances
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the retry loop did not honor context cancellation, and the consensus selector skipped instances that were actually available. Mutation-proven.
- Evidence: commit `df8a8cb`

### §17. [ATM-017] Successful-but-empty translation silently wiped source content (data loss)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: when a provider returned an empty string with a success status, the pipeline wrote the empty result over the source instead of preserving the original. Mutation-proven.
- Evidence: commit `837f528`

### §18. [ATM-018] Dashboard ETA formatDuration trailing space + non-unique progress event IDs
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: `formatDuration` emitted a trailing space and progress event IDs were not unique, breaking dashboard event de-duplication. Mutation-proven.
- Evidence: commit `b1b747f`

### §19. [ATM-019] Qwen client posted to native DashScope path under an OpenAI-compatible base (wrong URL + response-shape mismatch)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the Qwen client mixed the native DashScope endpoint with an OpenAI-compatible base URL, producing a wrong request URL and a response shape the parser could not read. Mutation-proven.
- Evidence: commit `556002d`

### §20. [ATM-020] preparation-translator -providers flag silently ignored; CLI explicit -provider overridden by config
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the `-providers` flag was parsed but never consumed, and an explicit `-provider` was overwritten by the config value. Mutation-proven.
- Evidence: commit `5dfe003`

### §21. [ATM-021] Storage cache dup-tuple served stale translation (added UNIQUE lookup-hash index + idempotent UPSERT)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: duplicate lookup tuples could store divergent rows with no uniqueness constraint, serving a stale translation; fixed with a UNIQUE index, idempotent UPSERT, and migration. Mutation-proven.
- Evidence: commit `367adce`

### §22. [ATM-022] VersionManager.metrics data race / lost updates (+ §11.4.85 stress/chaos suites)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: concurrent metric updates in the distributed VersionManager raced and lost increments; fixed with synchronization plus stress/chaos coverage. Mutation-proven.
- Evidence: commit `bf81517`

### §23. [ATM-023] Qwen compatible-mode alignment to default base + §11.4.98-honest live test
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: Qwen compatible-mode used an inconsistent base URL; aligned to the default compatible base and made the live test honest per §11.4.98.
- Evidence: commit `2f0e359`

### §24. [ATM-024] DOCX input non-functional (license-gated unioffice) — rewritten as a stdlib OOXML parser
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: DOCX parsing depended on the license-gated unioffice library and was inert; rewritten as a stdlib OOXML parser, dropping the dependency. Real-artifact proven.
- Evidence: commit `d433210`

### §25. [ATM-025] API-key gate ignored provider env vars (DEEPSEEK_API_KEY etc.) in unified-translator
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the unified CLI API-key gate did not read provider-specific environment variables, blocking otherwise-valid providers. Real DeepSeek E2E proof committed.
- Evidence: commit `c6c2930`

### §26. [ATM-026] DOCX rejected by the CLI pipeline despite a working parser (detector.IsSupported missing DOCX)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the format detector's supported-set omitted DOCX, so the CLI rejected files the parser could handle. Real E2E proven.
- Evidence: commit `6304f60`

### §27. [ATM-027] FB2 + EPUB input broken end-to-end (convertToMarkdown re-parsed already-extracted text)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the unified pipeline re-ran `convertToMarkdown` over already-extracted FB2/EPUB text, corrupting the content; fixed so all five input formats translate. Real E2E matrix proven.
- Evidence: commit `41063cb`

### §28. [ATM-028] Zhipu missing from the API-key env map in unified-translator (+ stale-allowlist audit)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the Zhipu provider was absent from the env-key map, so its key was never picked up. Multi-provider E2E proof + stale-allowlist audit committed.
- Evidence: commit `9821e0f`

### §29. [ATM-029] Stale provider model-allowlists blocked current gemini/groq/zhipu models
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: hardcoded model allowlists predated current provider catalogs, rejecting valid current models; added current models, live-proven.
- Evidence: commit `b23bcac`

### §30. [ATM-030] Coordination consensus tie-break was non-deterministic (map-iteration order)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: consensus tie-breaking iterated a map, producing non-deterministic winners across runs; made deterministic. Mutation-proven.
- Evidence: commit `77a0c15`

### §31. [ATM-031] Multipass verification chapter loop panicked when translation has fewer chapters than original
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the multipass loop indexed the translation by the original's chapter count, panicking when the translation had fewer chapters.
- Evidence: commit `9640a00`

### §32. [ATM-032] FB2 writer dropped section/subsection titles
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the FB2 writer omitted section and subsection `<title>` elements from the serialized output, losing chapter headings. Mutation-proven.
- Evidence: commit `b83038c`

### §33. [ATM-033] PDF input revived — replaced license-gated unipdf with MIT ledongthuc/pdf
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: PDF input was dead behind the license-gated unipdf dependency; replaced with the MIT ledongthuc/pdf library. Real-translation proven.
- Evidence: commit `18d6f73`

### §34. [ATM-034] CLI ignored -o output format — always emitted EPUB
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the CLI hardcoded EPUB output regardless of the `-o` extension; now honors `.txt/.fb2/.md/.epub` (+ 2 §11.4.120 gate reconciliations).
- Evidence: commit `ca5608f`

### §35. [ATM-035] Add .html/.htm CLI output format (completes the 5-output matrix)
**Status:** Implemented (→ Fixed.md)
**Type:** Feature
- Summary: added HTML-escaped `.html/.htm` output, completing the output-format matrix. Real-translation proven.
- Evidence: commit `fb07a59`

### §36. [ATM-036] DeepSeek model allowlist un-staled (accept deepseek-v4-flash/v4-pro)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the DeepSeek allowlist predated v4 models and rejected them; updated via live `/models` + real-translation verification (the canonical allowlist-refresh template).
- Evidence: commit `0fd1a34`

### §37. [ATM-037] WebSocket monitoring test harness data races + fatal map panic
**Status:** Fixed (→ Fixed.md)
**Type:** Task
- Root cause: the monitoring test harness had concurrent map access causing data races and a fatal map-write panic; serialized access in the harness.
- Evidence: commit `ab7db0e`

### §38. [ATM-038] TestSSHErrorHandling brittle error-string assertion (§11.4.1)
**Status:** Fixed (→ Fixed.md)
**Type:** Task
- Root cause: the SSH error test asserted an exact error substring that varied by environment, producing FAIL-bluffs; assert on a stable condition instead.
- Evidence: commit `817b9dd`

### §39. [ATM-039] Preparation per-chapter analysis stamped no authoritative chapter number
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: per-chapter analysis records lacked the authoritative chapter number, breaking downstream attribution; now stamped.
- Evidence: commit `0df25d9`

### §40. [ATM-040] hardware parseLscpuCores undercounts physical cores on multi-socket hosts
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: `parseLscpuCores` counted cores per socket once instead of multiplying across sockets, undercounting on multi-socket machines and mis-tuning concurrency.
- Evidence: commit `80d627b`

### §41. [ATM-041] Batch same-stem/different-ext inputs collided onto one output file (data loss)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: batch output naming used only the file stem, so `a.txt` and `a.fb2` produced the same output path and overwrote each other.
- Evidence: commit `c3117f8`

### §42. [ATM-042] Security RefreshToken resurrected expired/unbounded sessions into fresh valid tokens
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: token refresh did not re-check session expiry/bounds, so an expired session could be exchanged for a fresh valid token.
- Evidence: commit `d0fe40a`

### §43. [ATM-043] Empty/null LLMsVerifier envelope ({"models":null}) failed to decode, breaking /api/v1/verified-models
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the verified-models decoder did not tolerate a null models array, so the endpoint broke whenever zero models were verified. Mutation-proven.
- Evidence: commit `d8142e5`

### §44. [ATM-044] FB2 <v>/<subtitle>/<text-author> dropped inline-element text
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the FB2 parser did not descend into inline elements inside `<v>`, `<subtitle>`, and `<text-author>`, dropping their text. Mutation-proven.
- Evidence: commit `8f52370`

### §45. [ATM-045] Distributed fallback getFailureRate fabricated a 100% rate on window expiry / zero window
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the failure-rate calculation divided by an empty/expired window and reported a fabricated 100% failure rate, triggering needless fallback.
- Evidence: commit `20beda7`

### §46. [ATM-046] markdown→EPUB chapter <title> shipped unescaped — malformed XHTML
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the markdown→EPUB writer wrote chapter titles into XHTML without escaping `&`/`<`/`>`, producing malformed XHTML for those titles. Mutation-proven.
- Evidence: commit `be81550`

### §47. [ATM-047] Verification polishChapter/polishSectionRecursive index-out-of-range on shorter translations
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the polish pass indexed by the original section/subsection count and went out of range when the translation had fewer. Mutation-proven.
- Evidence: commit `7434ee4`

### §48. [ATM-048] Ollama client dropped temperature/max_tokens — never sent options to /api/generate
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the Ollama client never populated the `options` field, so temperature and max_tokens were silently ignored. Mutation-proven.
- Evidence: commit `a728e57`

### §49. [ATM-049] EPUB chapter text shipped HTML entities literally to readers
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: EPUB chapter text was emitted with HTML entities un-decoded, so readers saw literal entity sequences. Mutation-proven.
- Evidence: commit `107e570`

### §50. [ATM-050] gRPC SubscribeEvents delivered lifecycle events with an empty session_id
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the SubscribeEvents stream forwarded lifecycle events without populating `session_id`, breaking client-side session filtering.
- Evidence: commit `fd07e18`

### §51. [ATM-051] LoadConfig nil-map panic when config omits translation.providers and a provider key env var is set
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: `LoadConfig` wrote into a nil providers map when the config omitted `translation.providers` but a provider API-key env var was present, panicking on startup.
- Evidence: commit `fa35d2e`

### §52. [ATM-052] EPUB cover image mislabeled image/jpeg on round-trip (broke PNG/GIF/WEBP/SVG covers)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the EPUB writer hardcoded `image/jpeg` for the cover media type, breaking non-JPEG covers on round-trip.
- Evidence: commit `c416bd8`

### §53. [ATM-053] BaseTranslator cache/stats data race + concurrent-map-write crash
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: concurrent translators accessed the cache/stats maps without synchronization, causing data races and a concurrent-map-write crash. Race-proven.
- Evidence: commit `8316e3f`

### §54. [ATM-054] markdown-translator -format flag unvalidated — unsupported value produced no output yet reported success
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the markdown-translator did not validate `-format` against `{epub,md}`, so an unsupported value silently produced no output while reporting success.
- Evidence: commit `85b3362`

### §55. [ATM-055] CLI SSH provider ignored -source/target-lang, -script, -llama-binary/-model (hardcoded ru→sr-cyrillic)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the SSH provider path hardcoded ru→sr-cyrillic and ignored the language/script/llama flags, so non-default jobs produced wrong output.
- Evidence: commit `36f740a`

### §56. [ATM-056] cmd/api-server source was gitignored; committed health-check regression guard
**Status:** Fixed (→ Fixed.md)
**Type:** Task
- Root cause: a `.gitignore` rule excluded `cmd/api-server/` source (§11.4.30 violation); un-ignored it and committed a health-check regression guard.
- Evidence: commit `4bbac9a`

### §57. [ATM-057] cmd/cli GEMINI_API_KEY silently ignored — gemini missing from env-key map
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the `cmd/cli` env-key map omitted gemini, so `GEMINI_API_KEY` was never read.
- Evidence: commit `309ca91`

### §58. [ATM-058] translate-ssh delivered the translated EPUB to the wrong path; -output ignored
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the SSH worker wrote the translated EPUB to a derived path and ignored `-output`, delivering the result to the wrong location.
- Evidence: commit `86d10d4`

### §59. [ATM-059] grpc-server ignored documented env-var overrides (GRPC_ADDRESS/PORT, LOG_LEVEL, ENABLE_METRICS/REFLECTION)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the gRPC server documented env-var overrides but never read them, so configuration via environment had no effect.
- Evidence: commit `21c2460`

### §60. [ATM-060] cmd/deployment status action panicked on container IDs shorter than 12 chars
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the deployment status formatter sliced the container ID to 12 chars without a length check, panicking on shorter IDs.
- Evidence: commit `cde2e1a`

### §61. [ATM-061] grpc-server/monitor-server: dead -max-connections flag wired + monitor start error no longer swallowed
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the `-max-connections` flag was parsed but inert, and the monitor server's start error was discarded, hiding startup failures. Mutation-proven.
- Evidence: commit `93e89d7`

### §62. [ATM-062] logger JSON reserved-key collision silently dropped real severity/message/timestamp
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: structured log fields named like reserved keys (severity/message/timestamp) overwrote the real log values in the JSON output. Mutation-proven.
- Evidence: commit `daa2f70`

### §63. [ATM-063] enhanceTranslation capitalization dead for Cyrillic/accented-Latin first letters (byte-indexed guard)
**Status:** Fixed (→ Fixed.md)
**Type:** Bug
- Root cause: the capitalization guard used a byte index assuming ASCII, so multi-byte Cyrillic/accented-Latin first letters were never capitalized. Mutation-proven.
- Evidence: commit `e520c4c`

### §64. [ATM-064] Single-source binary version from authoritative VERSION
**Status:** Fixed (→ Fixed.md)
**Type:** Task
- Root cause: binaries hardcoded divergent `appVersion` literals; reconciled to read the single authoritative `VERSION` (P0.1). Mutation-proven.
- Evidence: commit `a36030e`

### §65. [ATM-069] Inert config fields (DOCXConfig.MinTextLength/IgnoreStyles, PDFConfig.MinTextLength) — wired
**Status:** Implemented (→ Fixed.md)
**Type:** Task
- Root cause: `DOCXConfig.MinTextLength`, `DOCXConfig.IgnoreStyles` were declared/documented but never consumed by the DOCX parser. Wired both with backward-compatible defaults (MinTextLength=0 preserves all paragraphs; IgnoreStyles=false preserves all styles). PDFConfig.MinTextLength left as design-only (PDF parsing uses a different pipeline). RED→GREEN §11.4.43 polarity tests + §11.4.135 regression guards.
- Evidence: commits `b24fced` (MinTextLength), `4c010fc` (IgnoreStyles)

### §66. [ATM-076] §11.4.65 universal markdown export audit across all tracked docs
**Status:** Completed (→ Fixed.md)
**Type:** Task
- Root cause: the docs_chain auto-sync handles most exports, but a full audit was needed to confirm every `docs/*.md` has fresh `.html`+`.pdf` siblings. Audit found 170/170 in-scope `.md` files; 51 HTML + 1 PDF regenerated (stale mtimes); 0 failures.
- Evidence: commit `46202e5`

### §67. [ATM-079] docs/qa/<run-id> evidence per shipped feature (§11.4.83)
**Status:** Completed (→ Fixed.md)
**Type:** Task
- Root cause: §11.4.83 requires a recorded e2e transcript per shipped feature. Created 5 evidence directories for rev92 fixes (sqlite_busy, version_authoritative, redis_observability, request_context, version_guard) with real captured test outputs. Redis test SKIP (no local Redis, honest §11.4.3).
- Evidence: commit `64ed458`

### §68. [ATM-075] Pre-build CM-* gate suite implementation
**Status:** Completed (→ Fixed.md)
**Type:** Task
- Root cause: the constitution references dozens of `CM-*` pre-build gates + paired §1.1 mutations. Implemented 13 gates with paired mutations in `scripts/testing/meta_test_*.sh`: CM-GITIGNORE-PRECOMMIT-AUDIT, CM-NO-FAKES-BEYOND-UNIT, CM-SCRIPT-TARGET-SHELL-PARSEABLE, CM-VERSION-SINGLE-SOURCE, CM-TRACKER-DOCS-PRESENT, CM-ATM-TICKET-IDS, CM-DOC-SIBLING-SYNC, CM-NO-FORCE-PUSH-ABSOLUTE, CM-NO-LOCAL-RUNTIME, CM-ANTI-BLUFF-SMOKE, CM-CONSTITUTION-PROPAGATION, CM-REGRESSION-GUARD-REGISTERED, CM-CONSTITUTION-INHERITANCE.
- Evidence: commits `3e85ffc`, `a96b17b`, `78fa353` + existing gates

### §69. [ATM-078] Per-feature test-type matrix + coverage ledger (§11.4.25/§11.4.27)
**Status:** Completed (→ Fixed.md)
**Type:** Task
- Root cause: §11.4.27 mandates 100% coverage with every test type. Updated `docs/testing/coverage_matrix.md` with current data: ~68% total statement coverage (up from 50.7%), 8/16 mandated test types present, 13 CM-* gates with paired mutations, 22 packages ≥80% coverage. Remaining gaps (chaos, DDoS, scaling, full-automation, UI/UX) documented with prioritized remediation plan.
- Evidence: `docs/testing/coverage_matrix.md` revision 2

### §70. [ATM-065] Single authoritative version number decided
**Status:** Fixed (→ Fixed.md)
**Type:** Task
- Root cause: `VERSION`=2.3.0 but Makefile historically referenced 3.0.0. Operator decided 2.3.1. VERSION file already at 2.3.1, version tests pass, release tag `helix_translate-2.3.1` exists.
- Evidence: operator decision 2026-07-08

### §71. [ATM-068] Inert CLI flags wired to real functionality
**Status:** Implemented (→ Fixed.md)
**Type:** Task
- Root cause: `-verify` was always-on (ignored the flag), `-chunk-size` was hardcoded 20000, `-monitoring` was already wired. Wired `-verify` to control verification, `-chunk-size` plumbed through TranslationConfig to LLMTranslator.splitText(). `-workers`/`-concurrency` reserved for distributed path.
- Evidence: commit `be930f6`

### §72. [ATM-071] Reasoning-model structured-content support
**Status:** Implemented (→ Fixed.md)
**Type:** Feature
- Root cause: OpenAI/Qwen/Zhipu providers assumed `content` is a string. Reasoning models (Mistral magistral, deepseek-reasoner) return content as a structured list. Added `FlexibleContent` type with custom JSON unmarshaler handling string, array-of-objects, and array-of-strings formats. Backward compatible.
- Evidence: commit `a69acf9`

### §73. [ATM-072] Markdown first-class input with structure preservation
**Status:** Implemented (→ Fixed.md)
**Type:** Feature
- Root cause: `.md` files were translated as plain text, losing markdown structure. Now `.md`/`.markdown` files read as raw markdown and translated line-by-line using `pkg/markdown.MarkdownTranslator`. Preserves headings, lists, code blocks, blockquotes, bold/italic, links, frontmatter.
- Evidence: commit `f0b1996`

### §74. [ATM-073] cmd/translator intermediate-markdown download-dir inconsistency
**Status:** Obsolete (→ Fixed.md)
**Type:** Bug
- Root cause: `cmd/translator` no longer exists (removed in bridge phase-2). The SSH-local translation path was removed. The bug is structurally impossible in the current codebase.
- Evidence: `cmd/translator` directory absent, SSH provider returns hard error

### §75. [ATM-074] pkg/hash relocated to cmd/codebase-hash
**Status:** Completed (→ Fixed.md)
**Type:** Task
- Root cause: `pkg/hash` was `package main` under `pkg/` (misplaced, cannot be imported). Relocated to `cmd/codebase-hash/` where standalone commands belong. Functional duplicate of `pkg/version.CodebaseHasher` (already wired).
- Evidence: commit `a0c0f4c`

---

*This archive is regenerated into `Fixed_Summary.md` by `scripts/testing/generate_fixed_summary.sh`. Do not hand-edit the summary.*
