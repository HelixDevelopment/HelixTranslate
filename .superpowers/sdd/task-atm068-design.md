# ATM-068 Design Proposal: Inert CLI Flags in unified-translator

**Revision:** 1
**Last modified:** 2026-07-08T14:15:00Z
**Scope:** Investigation + design proposal only (no source edits per §11.4.122)

## Summary

4 of 6 investigated flags are **genuinely inert** (parsed but never consumed). 2 flags (`-monitoring`, `-monitoring-port`) are wired to a print-only stub. All flags introduced in commit `de665cb` ("Auto-commit") — never completed.

---

## Per-flag analysis

### 1. `-workers` (default: 1)

**State:** INERT — parsed at line 602, never referenced after `flag.Parse()`.

**Evidence:**
- Definition: `cmd/unified-translator/main.go:602` — `flag.IntVar(&config.Workers, "workers", 1, "Number of parallel workers")`
- Zero post-parse references: `config.Workers` appears only at the definition line.
- Git history: introduced in `de665cb` ("Auto-commit") — never wired.

**OPTION A — Wire:** The translator package (`pkg/translator/llm/llm.go`) processes chunks sequentially in `translateWithRetry`. Wiring `-workers` would require a worker-pool pattern in the translation loop. Non-trivial: needs goroutine pool + error aggregation + progress tracking.

**OPTION B — Remove:** Per §11.4.122, removing a user-facing flag needs operator confirmation. The flag is documented in `--help` output (line 742). Removing it is the cleanest option since chunking is already automatic+correct.

**RECOMMENDATION:** Remove (OPTION B). The flag promises parallelism that doesn't exist. A user setting `-workers 4` gets sequential processing with no warning — a §11.4.108 lie-class defect. Wiring real parallelism is a separate feature (ATM-071 territory) and should be done deliberately, not as a flag fix.

---

### 2. `-chunk-size` (default: 2000)

**State:** INERT — parsed at line 603, never referenced after `flag.Parse()`.

**Evidence:**
- Definition: `cmd/unified-translator/main.go:603` — `flag.IntVar(&config.ChunkSize, "chunk-size", 2000, "Text chunk size")`
- Zero post-parse references: `config.ChunkSize` appears only at the definition line.
- The actual chunking uses a HARDCODED constant: `pkg/translator/llm/llm.go:504` — `const maxChunkSize = 20000` (note: 20000, not 2000 — the flag default is 10× smaller than the real value).
- `splitText` at `llm.go:502` uses this constant, never reads any config field.
- Git history: introduced in `de665cb` — never wired.

**OPTION A — Wire:** Pass `config.ChunkSize` through to `LLMTranslator` and replace the hardcoded `maxChunkSize` constant. Requires: (a) add `ChunkSize` field to `LLMTranslator` struct or pass it as a parameter to `splitText`, (b) update `NewLLMTranslator` constructor, (c) add tests proving different chunk sizes work correctly, (d) validate the value (min/max bounds — 200 is too small, 100000 defeats chunking).

**OPTION B — Remove:** Cleanest. The hardcoded 20000 is battle-tested (the chunk roundtrip test at `llm/chunk_roundtrip_test.go` proves lossless tiling). Exposing it as a flag without careful bounds validation risks breaking translation quality.

**RECOMMENDATION:** Remove (OPTION B). The hardcoded `maxChunkSize = 20000` is correct and tested. Exposing it as a knob invites misconfiguration (user sets `-chunk-size 100` → tiny chunks → worse translation quality from lost context). If chunk-size tuning is ever needed, it should be a config field with documented bounds, not a CLI flag.

---

### 3. `-concurrency` (default: 4)

**State:** INERT — parsed at line 604, never referenced after `flag.Parse()`.

**Evidence:**
- Definition: `cmd/unified-translator/main.go:604` — `flag.IntVar(&config.Concurrency, "concurrency", 4, "Maximum concurrent operations")`
- Zero post-parse references: `config.Concurrency` appears only at the definition line.
- Git history: introduced in `de665cb` — never wired.

**OPTION A — Wire:** Would limit concurrent LLM API calls. Requires: semaphore pattern in the translation loop, integration with the provider's rate limits. Related to `-workers` but controls a different dimension (concurrent API calls vs parallel translation workers).

**OPTION B — Remove:** Clean. The LLM providers already have their own rate limits. Adding an app-level concurrency cap without understanding the provider's limits is premature.

**RECOMMENDATION:** Remove (OPTION B). Same rationale as `-workers`: promises behavior that doesn't exist. If concurrency control is needed, it belongs in the provider layer (`pkg/translator/llm/`), not as a CLI flag.

---

### 4. `-verify` (default: true)

**State:** INERT — parsed at line 605, never referenced after `flag.Parse()`.

**Evidence:**
- Definition: `cmd/unified-translator/main.go:605` — `flag.BoolVar(&config.VerifyOutput, "verify", true, "Verify translated output")`
- `config.VerifyOutput` appears ONLY at the definition line — never checked.
- The verification function `verifyTranslation` (line 1095) ALWAYS runs (called at line 269) — it does NOT check `config.VerifyOutput`. It's a basic heuristic: checks for Cyrillic chars in Serbian mode, or non-empty text otherwise.
- Git history: introduced in `de665cb` — never wired.

**OPTION A — Wire:** Add `if !config.VerifyOutput { return true }` at the top of `verifyTranslation`. Trivial one-line fix. But: the verification is already a lightweight heuristic (not a real quality check), so disabling it has minimal practical value.

**OPTION B — Remove:** Clean. The verification is always-on and harmless (sub-second, no side effects). A flag to disable it serves no user need.

**RECOMMENDATION:** Wire (OPTION A) — it's a one-line fix and the flag is already documented in `--help`. But mark it as LOW priority since the verification is a heuristic, not a quality gate. If the flag is removed instead, update `--help` too.

---

### 5. `-monitoring` / `-monitoring-port` (STUB)

**State:** PARTIALLY WIRED — `EnableMonitoring` gates `startMonitoringServer` (line 151-154), but the function is a print-only stub.

**Evidence:**
- `config.EnableMonitoring` checked at line 151: `if config.EnableMonitoring { go startMonitoringServer(config.MonitoringPort, eventBus) }`
- `startMonitoringServer` body (line 1240-1243): `fmt.Printf("Monitoring server available on port %d\n", port)` — that's it.
- A real monitoring server exists at `cmd/monitor-server/` with WebSocket hub + dashboards.
- Git history: introduced in `de665cb` — stub never completed.

**OPTION A — Wire:** Delegate to the real monitor-server. Either: (a) embed the monitor-server's `main()` logic into `startMonitoringServer` (complex, couples the binaries), or (b) have `-monitoring` launch `cmd/monitor-server` as a subprocess (cleaner, but requires the binary to be built), or (c) just print a helpful message: "Run `cmd/monitor-server` separately for the dashboard at http://localhost:8090/monitor".

**OPTION B — Remove the stub, document the real server:** Remove `-monitoring`/`-monitoring-port` flags. Add to `--help`: "For the monitoring dashboard, run `cmd/monitor-server` separately (port 8090)."

**RECOMMENDATION:** OPTION B (remove flags, document real server). The stub does nothing useful and misleads users into thinking `-monitoring` enables monitoring. The real `cmd/monitor-server` is a separate binary by design (it's a standalone WebSocket hub). Coupling them would be an architectural mistake.

---

## Summary table

| Flag | Status | Wired? | Recommendation | Priority |
|------|--------|--------|---------------|----------|
| `-workers` | INERT | No | Remove (operator confirm §11.4.122) | MED |
| `-chunk-size` | INERT | No | Remove (operator confirm §11.4.122) | MED |
| `-concurrency` | INERT | No | Remove (operator confirm §11.4.122) | MED |
| `-verify` | INERT | No | Wire (1-line fix) or Remove | LOW |
| `-monitoring` | STUB | Print-only | Remove flags, document real server | MED |
| `-monitoring-port` | STUB | Print-only | Remove with `-monitoring` | MED |

## Operator decision required (§11.4.122)

All removals of user-facing flags need operator confirmation. The flags are documented in `--help` (lines 742-753). Proposal: remove all 5 inert/stub flags in one commit, update `--help` to document `cmd/monitor-server` as the monitoring entry point, and add a note that chunking is automatic.

## Git history

All flags introduced in commit `de665cb` ("Auto-commit") — a single scaffold commit that defined the flags but never wired them to the translation pipeline. No subsequent commit attempted to wire any of them.
