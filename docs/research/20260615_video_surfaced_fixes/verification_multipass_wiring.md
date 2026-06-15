# Deep-research — Wiring the dormant `pkg/verification` multipass polisher into `unified-translator`

**Revision:** 1
**Last modified:** 2026-06-15T00:00:00Z
**Author:** background feature-completion stream (§11.4.124 / §11.4.150)
**Scope:** `cmd/unified-translator/main.go` + `pkg/verification`

---

## 1. The gap (FACT, this session)

`pkg/verification` contains a substantial, actively-maintained multi-pass
LLM verification/polishing engine (`MultiPassPolisher`, `BookPolisher`,
`Verifier`, note-taking, SQLite persistence, reporter). It is **never wired
into any `cmd/` binary**. The CLI `-verify` flag only runs lightweight
heuristic output-quality checks (`verifyTranslation` = target-script
character presence; `verifyEPUB` = structural check). The real multipass
*polishing* capability ships dormant.

### Git-history investigation (§11.4.124 — investigate-before-wire)

- `git log --all -S "pkg/verification" -- cmd/` → **empty**. No `cmd/`
  binary has *ever* imported the package across the entire history.
- First add: commit `717bf18` ("Auto-commit", 2025-11-20) created the
  package already unwired, alongside `Documentation/VERIFICATION_SYSTEM.md`.
- Current importers (whole tree): only `test/performance/…`,
  `test/unit/verification_test.go`, `test/e2e/…` plus the package's own
  `_test.go` files. Zero production importers.
- `verifier.go` has 5 commits; `multipass.go` carries recent
  mutation-proven defensive fixes (chapter-count cap to avoid panic;
  high-water-mark dedup of DB writes; drop-empty-provider consensus to
  avoid score dilution / content-wipe). This is **production code that
  was finished but never connected**, not scaffolding.

**Determination:** genuinely-built capability, never wired (born unwired in
an auto-commit). Per §11.4.124 the correct action is to **wire it in
properly**, not remove it.

### Engine soundness check (is it correct, or scaffolding?)

`BookPolisher.polishSection` → `verifyWithLLM` → `translator.Translate`
(real LLM call) → `parseVerificationResponse` (parses SPIRIT/LANGUAGE/
CONTEXT/VOCABULARY scores, ISSUES, POLISHED_TEXT) → `buildConsensus`
(averages scores, picks polished candidate, **defaults to the existing
translation and honours a `UNCHANGED` sentinel**). The engine genuinely
calls the LLM and applies a parsed, consensus-selected polished version —
it is not a stub.

---

## 2. Deep multi-angle research (§11.4.150 — ≥2 angles, latest sources)

### Angle 1 — Technique: iterative LLM MT post-editing is established practice

The canonical translation-specific method is **TEaR (Translate, Estimate,
Refine)** — exactly the "translate → verify/critique → polish" shape we are
wiring. Key findings:

- A **single** refinement round is best by design; COMET peaks at pass 1
  then plateaus/declines (78.67 → 79.50 → 79.29 → 79.28 on WMT23 Zh-En),
  later passes regress due to hallucinated estimation feedback.
  <https://arxiv.org/html/2402.16379v3> (accessed 2026-06-15);
  <https://arxiv.org/abs/2402.16379> (accessed 2026-06-15).
- Stopping = a quality-estimation **gate**: if no errors detected, return
  the original **unchanged** — the built-in regression guard.
- Multi-LLM consensus per pass (M-MAD / multi-agent debate, best-of-N QE
  reranking) is a credible *verification*-stage pattern.
  <https://www.emergentmind.com/topics/multi-agent-debate-mad-paradigm>
  (accessed 2026-06-15);
  <https://arxiv.org/html/2510.08870v1> (accessed 2026-06-15).
- Standard gating metrics: COMET / GEMBA / LLM-as-judge.
  <https://kvashee.medium.com/mt-quality-evaluation-in-the-age-of-llm-mt-ca8a9540a2ab>
  (accessed 2026-06-15).

### Angle 2 — Risk / failure-modes: refinement can HARM without guards

- Intrinsic self-correction without an external signal often fails to
  improve or **degrades** output (inherits the same hallucinations that
  caused the original error). TACL critical survey "When Can LLMs Actually
  Correct Their Own Mistakes":
  <https://direct.mit.edu/tacl/article/doi/10.1162/tacl_a_00713/125177/>
  (accessed 2026-06-15; full PDF returned HTTP 403, finding quoted from the
  publisher abstract/excerpt — honest gap).
- Vanilla multi-agent debate **frequently underperforms a single agent**.
  <https://d2jud02ci9yv69.cloudfront.net/2025-04-28-mad-159/blog/mad/>
  (accessed 2026-06-15); <https://arxiv.org/pdf/2502.08788> (accessed
  2026-06-15).
- Cost/latency multiply per pass and per voter (a real concern for a CLI).

---

## 3. Verdict

- **(A) Wiring is sound, current best practice — YES.** TEaR is precisely
  this pattern and improves quality on the first refinement pass when gated
  by a genuine verify/critique step. Our engine already gates (parses
  scores + issues, honours `UNCHANGED`, defaults to the existing
  translation), so wiring it does NOT mask a deeper problem.
- **(B) Guardrails are mandatory.** They map onto what the engine already
  provides + the CLI defaults we choose.

### Guardrails honoured by this wiring

- **Cap passes low.** New `-multipass-passes` flag defaults to **1**
  (TEaR's optimum); operator may raise it but the default never regresses.
- **Gate on a real verify signal.** The engine's per-provider LLM critique
  + consensus IS the gate (not blind "improve again").
- **Preserve original on no improvement.** Engine defaults `PolishedText`
  to the existing translation and skips the `UNCHANGED` sentinel — kept
  intact; if multipass yields empty/identical text the original markdown is
  retained.
- **Opt-in, back-compat preserved.** Multipass is a SEPARATE `-multipass`
  flag. The existing `-verify` heuristic path is untouched (§11.4.1 — no
  solve-A-create-B). Multipass OFF by default (it costs real LLM calls).
- **Don't mask upstream defects.** Multipass runs *after* the base
  translation and only polishes; a base-translation failure still surfaces
  as a translation error before multipass runs.

---

## Sources verified 2026-06-15

- TEaR: <https://arxiv.org/html/2402.16379v3>, <https://arxiv.org/abs/2402.16379>
- QE reranking: <https://arxiv.org/html/2510.08870v1>
- MAD paradigm: <https://www.emergentmind.com/topics/multi-agent-debate-mad-paradigm>
- MAD underperformance: <https://d2jud02ci9yv69.cloudfront.net/2025-04-28-mad-159/blog/mad/>, <https://arxiv.org/pdf/2502.08788>
- MT QE metrics: <https://kvashee.medium.com/mt-quality-evaluation-in-the-age-of-llm-mt-ca8a9540a2ab>
- Self-correction limits: <https://direct.mit.edu/tacl/article/doi/10.1162/tacl_a_00713/125177/> (abstract only; PDF 403 — honest gap)

Deep-research 2026-06-15: NO external solution found that contradicts wiring; TEaR confirms the pattern, guardrails enumerated above.
