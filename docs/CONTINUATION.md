# CONTINUATION — HelixTranslate session-resumption file

**Revision:** 33
**Last modified:** 2026-06-14T15:10:00Z

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
