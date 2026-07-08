# HelixTranslate — Fixed Summary

**Revision:** 1
**Last modified:** 2026-07-08T17:36:50Z
**Authority:** §11.4.53 (Fixed_Summary parity) · §11.4.54 (ATM ID column) · §11.4.19 (column-alignment) · §11.4.91 (clear descriptions)
**Generated:** auto-generated from `docs/Fixed.md` by `scripts/testing/generate_fixed_summary.sh` — do not hand-edit.

| ATM ID | Level | Status | Type | One-line description |
| --- | --- | --- | --- | --- |
| ATM-001 | Bug | Fixed (→ Fixed.md) | Bug | splitText dropped paragraph separators at chunk boundaries causing structural data loss |
| ATM-002 | Bug | Fixed (→ Fixed.md) | Bug | In-memory translation cache-key collision served the wrong translation |
| ATM-003 | Bug | Fixed (→ Fixed.md) | Bug | FB2 writer dropped deeply-nested (>2 level) section content as translated-text data loss |
| ATM-004 | Bug | Fixed (→ Fixed.md) | Bug | Anthropic client dropped all content blocks after the first, truncating translations |
| ATM-005 | Bug | Fixed (→ Fixed.md) | Bug | Distributed ResultCache spurious eviction on update + round-robin index panic after pool shrink |
| ATM-006 | Bug | Fixed (→ Fixed.md) | Bug | EPUB percent-encoded/relative hrefs dropped chapters and cover; HTML test-hack corrupted content |
| ATM-007 | Bug | Fixed (→ Fixed.md) | Bug | Verification parseNote dropped EXAMPLES-terminated notes, produced NaN confidence, under-counted multipass sections |
| ATM-008 | Bug | Fixed (→ Fixed.md) | Bug | Redis makeCacheKey metadata delimiter-injection collision served the wrong translation |
| ATM-009 | Bug | Fixed (→ Fixed.md) | Bug | EPUB↔Markdown round-trip destroyed code blocks, lost GFM tables, corrupted backslash-escapes |
| ATM-010 | Bug | Fixed (→ Fixed.md) | Bug | API translateText silently dropped script=cyrillic and accepted invalid script; verifier handlers returned wrong HTTP status |
| ATM-011 | Bug | Fixed (→ Fixed.md) | Bug | Verifier fallback chain randomly shuffled (lost score order); OpenRouter pricing decode failed on string values |
| ATM-012 | Bug | Fixed (→ Fixed.md) | Bug | gRPC GetTranslationStatus returned the wrong error code; cleanupOldSessions leaked a timeout context |
| ATM-013 | Bug | Fixed (→ Fixed.md) | Bug | CWE-208 username-enumeration timing oracle in AuthenticateUser |
| ATM-014 | Bug | Fixed (→ Fixed.md) | Bug | Preparation failed-chapter analysis misattribution + nested-subsection analysis content loss |
| ATM-015 | Bug | Fixed (→ Fixed.md) | Bug | Directory batch to a non-existent output dir collided all files onto one path (data loss) |
| ATM-016 | Bug | Fixed (→ Fixed.md) | Bug | Coordination TranslateWithRetry ignored ctx cancellation; consensus skipped available instances |
| ATM-017 | Bug | Fixed (→ Fixed.md) | Bug | Successful-but-empty translation silently wiped source content (data loss) |
| ATM-018 | Bug | Fixed (→ Fixed.md) | Bug | Dashboard ETA formatDuration trailing space + non-unique progress event IDs |
| ATM-019 | Bug | Fixed (→ Fixed.md) | Bug | Qwen client posted to native DashScope path under an OpenAI-compatible base (wrong URL + response-shape mismatch) |
| ATM-020 | Bug | Fixed (→ Fixed.md) | Bug | preparation-translator -providers flag silently ignored; CLI explicit -provider overridden by config |
| ATM-021 | Bug | Fixed (→ Fixed.md) | Bug | Storage cache dup-tuple served stale translation (added UNIQUE lookup-hash index + idempotent UPSERT) |
| ATM-022 | Bug | Fixed (→ Fixed.md) | Bug | VersionManager.metrics data race / lost updates (+ §11.4.85 stress/chaos suites) |
| ATM-023 | Bug | Fixed (→ Fixed.md) | Bug | Qwen compatible-mode alignment to default base + §11.4.98-honest live test |
| ATM-024 | Bug | Fixed (→ Fixed.md) | Bug | DOCX input non-functional (license-gated unioffice) — rewritten as a stdlib OOXML parser |
| ATM-025 | Bug | Fixed (→ Fixed.md) | Bug | API-key gate ignored provider env vars (DEEPSEEK_API_KEY etc.) in unified-translator |
| ATM-026 | Bug | Fixed (→ Fixed.md) | Bug | DOCX rejected by the CLI pipeline despite a working parser (detector.IsSupported missing DOCX) |
| ATM-027 | Bug | Fixed (→ Fixed.md) | Bug | FB2 + EPUB input broken end-to-end (convertToMarkdown re-parsed already-extracted text) |
| ATM-028 | Bug | Fixed (→ Fixed.md) | Bug | Zhipu missing from the API-key env map in unified-translator (+ stale-allowlist audit) |
| ATM-029 | Bug | Fixed (→ Fixed.md) | Bug | Stale provider model-allowlists blocked current gemini/groq/zhipu models |
| ATM-030 | Bug | Fixed (→ Fixed.md) | Bug | Coordination consensus tie-break was non-deterministic (map-iteration order) |
| ATM-031 | Bug | Fixed (→ Fixed.md) | Bug | Multipass verification chapter loop panicked when translation has fewer chapters than original |
| ATM-032 | Bug | Fixed (→ Fixed.md) | Bug | FB2 writer dropped section/subsection titles |
| ATM-033 | Bug | Fixed (→ Fixed.md) | Bug | PDF input revived — replaced license-gated unipdf with MIT ledongthuc/pdf |
| ATM-034 | Bug | Fixed (→ Fixed.md) | Bug | CLI ignored -o output format — always emitted EPUB |
| ATM-035 | Feature | Implemented (→ Fixed.md) | Feature | Add .html/.htm CLI output format (completes the 5-output matrix) |
| ATM-036 | Bug | Fixed (→ Fixed.md) | Bug | DeepSeek model allowlist un-staled (accept deepseek-v4-flash/v4-pro) |
| ATM-037 | Task | Fixed (→ Fixed.md) | Task | WebSocket monitoring test harness data races + fatal map panic |
| ATM-038 | Task | Fixed (→ Fixed.md) | Task | TestSSHErrorHandling brittle error-string assertion (§11.4.1) |
| ATM-039 | Bug | Fixed (→ Fixed.md) | Bug | Preparation per-chapter analysis stamped no authoritative chapter number |
| ATM-040 | Bug | Fixed (→ Fixed.md) | Bug | hardware parseLscpuCores undercounts physical cores on multi-socket hosts |
| ATM-041 | Bug | Fixed (→ Fixed.md) | Bug | Batch same-stem/different-ext inputs collided onto one output file (data loss) |
| ATM-042 | Bug | Fixed (→ Fixed.md) | Bug | Security RefreshToken resurrected expired/unbounded sessions into fresh valid tokens |
| ATM-043 | Bug | Fixed (→ Fixed.md) | Bug | Empty/null LLMsVerifier envelope ({"models":null}) failed to decode, breaking /api/v1/verified-models |
| ATM-044 | Bug | Fixed (→ Fixed.md) | Bug | FB2 <v>/<subtitle>/<text-author> dropped inline-element text |
| ATM-045 | Bug | Fixed (→ Fixed.md) | Bug | Distributed fallback getFailureRate fabricated a 100% rate on window expiry / zero window |
| ATM-046 | Bug | Fixed (→ Fixed.md) | Bug | markdown→EPUB chapter <title> shipped unescaped — malformed XHTML |
| ATM-047 | Bug | Fixed (→ Fixed.md) | Bug | Verification polishChapter/polishSectionRecursive index-out-of-range on shorter translations |
| ATM-048 | Bug | Fixed (→ Fixed.md) | Bug | Ollama client dropped temperature/max_tokens — never sent options to /api/generate |
| ATM-049 | Bug | Fixed (→ Fixed.md) | Bug | EPUB chapter text shipped HTML entities literally to readers |
| ATM-050 | Bug | Fixed (→ Fixed.md) | Bug | gRPC SubscribeEvents delivered lifecycle events with an empty session_id |
| ATM-051 | Bug | Fixed (→ Fixed.md) | Bug | LoadConfig nil-map panic when config omits translation.providers and a provider key env var is set |
| ATM-052 | Bug | Fixed (→ Fixed.md) | Bug | EPUB cover image mislabeled image/jpeg on round-trip (broke PNG/GIF/WEBP/SVG covers) |
| ATM-053 | Bug | Fixed (→ Fixed.md) | Bug | BaseTranslator cache/stats data race + concurrent-map-write crash |
| ATM-054 | Bug | Fixed (→ Fixed.md) | Bug | markdown-translator -format flag unvalidated — unsupported value produced no output yet reported success |
| ATM-055 | Bug | Fixed (→ Fixed.md) | Bug | CLI SSH provider ignored -source/target-lang, -script, -llama-binary/-model (hardcoded ru→sr-cyrillic) |
| ATM-056 | Task | Fixed (→ Fixed.md) | Task | cmd/api-server source was gitignored; committed health-check regression guard |
| ATM-057 | Bug | Fixed (→ Fixed.md) | Bug | cmd/cli GEMINI_API_KEY silently ignored — gemini missing from env-key map |
| ATM-058 | Bug | Fixed (→ Fixed.md) | Bug | translate-ssh delivered the translated EPUB to the wrong path; -output ignored |
| ATM-059 | Bug | Fixed (→ Fixed.md) | Bug | grpc-server ignored documented env-var overrides (GRPC_ADDRESS/PORT, LOG_LEVEL, ENABLE_METRICS/REFLECTION) |
| ATM-060 | Bug | Fixed (→ Fixed.md) | Bug | cmd/deployment status action panicked on container IDs shorter than 12 chars |
| ATM-061 | Bug | Fixed (→ Fixed.md) | Bug | grpc-server/monitor-server: dead -max-connections flag wired + monitor start error no longer swallowed |
| ATM-062 | Bug | Fixed (→ Fixed.md) | Bug | logger JSON reserved-key collision silently dropped real severity/message/timestamp |
| ATM-063 | Bug | Fixed (→ Fixed.md) | Bug | enhanceTranslation capitalization dead for Cyrillic/accented-Latin first letters (byte-indexed guard) |
| ATM-064 | Task | Fixed (→ Fixed.md) | Task | Single-source binary version from authoritative VERSION |
| ATM-069 | Task | Implemented (→ Fixed.md) | Task | Inert config fields (DOCXConfig.MinTextLength/IgnoreStyles, PDFConfig.MinTextLength) — wired |
| ATM-076 | Task | Completed (→ Fixed.md) | Task | §11.4.65 universal markdown export audit across all tracked docs |
| ATM-079 | Task | Completed (→ Fixed.md) | Task | docs/qa/<run-id> evidence per shipped feature (§11.4.83) |
| ATM-075 | Task | Completed (→ Fixed.md) | Task | Pre-build CM-* gate suite implementation |
| ATM-078 | Task | Completed (→ Fixed.md) | Task | Per-feature test-type matrix + coverage ledger (§11.4.25/§11.4.27) |

Total closed items: 69
