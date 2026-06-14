# HelixTranslate — Issues Summary

**Revision:** 1
**Last modified:** 2026-06-14T15:41:17Z
**Authority:** §11.4.12 (Issues_Summary sync) · §11.4.54 (ATM ID column) · §11.4.19 (column-alignment) · §11.4.91 (clear descriptions)
**Generated:** auto-generated from `docs/Issues.md` by `scripts/testing/generate_issues_summary.sh` — do not hand-edit.

| ATM ID | Level | Status | Type | One-line description |
| --- | --- | --- | --- | --- |
| ATM-065 | High | Operator-blocked | Task | Decide the single authoritative version number (2.3.x vs 3.0.0) |
| ATM-066 | High | Operator-blocked | Bug | Provider credentials absent/invalid (OpenAI/Anthropic/Gemini/Zhipu) block allowlist refresh |
| ATM-067 | High | Operator-blocked | Task | ~30 other provider allowlists not audited against live current models |
| ATM-068 | Medium | Design | Task | Inert CLI flags in unified-translator (-chunk-size/-workers/-concurrency/-verify, -monitoring stub) |
| ATM-069 | Medium | Design | Task | Inert config fields (DOCXConfig.MinTextLength/IgnoreStyles, PDFConfig.MinTextLength) |
| ATM-070 | Medium | Design | Bug | Verifier MinScoreThreshold scale inconsistency (0-100 raw vs 0-10 normalized) |
| ATM-071 | Medium | Design | Feature | Reasoning-model structured-content support (content as LIST, not STRING) |
| ATM-072 | Medium | Design | Feature | Markdown not a first-class CLI input format |
| ATM-073 | High | Blocked | Bug | cmd/translator intermediate-markdown download-dir inconsistency (needs live SSH) |
| ATM-074 | High | Operator-blocked | Task | pkg/hash is a dead package (zero importers) — investigate per §11.4.124 |
| ATM-075 | Normal | Queued | Task | Pre-build CM-* gate suite not implemented |
| ATM-076 | Normal | Queued | Task | §11.4.65 universal markdown export audit across all tracked docs |
| ATM-077 | High | Blocked | Task | Owned-submodule bug-hunt + brittle-test fixes (§11.4.28 equal-codebase) |
| ATM-078 | Normal | Queued | Task | Per-feature test-type matrix + HelixQA + Challenges coverage (§11.4.25/§11.4.27) |
| ATM-079 | Normal | Queued | Task | docs/qa/<run-id> evidence per shipped feature (§11.4.83) |
| ATM-080 | High | Operator-blocked | Task | Full §11.4.40 7-step release retest not yet run |
| ATM-081 | High | Operator-blocked | Task | No §11.4.151 prefixed release tag yet |

Total open items: 17
