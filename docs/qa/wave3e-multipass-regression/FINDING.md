# Wave-3e multipass regression finding (§11.4.138 / §11.4.102 / §11.4.108)

**Date:** 2026-06-16  **Build HEAD:** 6d544c5  **Found by:** AI (wave-3e recording unit)

## Symptom
`unified-translator -multipass` (both default 1 pass AND `-multipass-passes 2`) produces a
CORRUPTED output artifact: the LLM's polishing-instruction meta-response is written as the
translation instead of the polished text. Observed outputs:
- 1 pass: `[Ту версион мехорада ен диалекто Екавица, о UNCHANGED си ла традуксион ес перфекта]`
- 2 pass: `[No text provided for translation. Please supply the Spanish translation to be assessed.]`

## §11.4.108 SOURCE/report→ARTIFACT gap (the bluff)
The session report (`mp1_session_report.md`) reports ALL green:
`Step 4: Multi-pass Polishing ✅ Success — Polished over 1 pass(es) with deepseek`
and `mp1.epub ✅ Verified` — while the real EPUB content is the meta-response garbage above.
Green report, broken artifact = §11.4 PASS-bluff.

## Root cause (FACT-grade, Phase 1)
The multipass polishing path writes the LLM polishing response directly as the new translation
with NO guard against the LLM returning an instruction-echo / meta-response / UNCHANGED-marker
instead of polished prose. The prior 20260615 video-confirmation got a clean polish response by
chance; the path is non-deterministic and unsafe.

## Impact on Status.md
Row 452 (`verification | Multi-pass polisher`, cited
`helixtranslate-cli-multipass-verify-20260615.mp4`) was marked PASS — video-confirmed. The
feature is BROKEN at runtime now → demoted to PENDING_FORENSICS, video-confirmed count 41 → 40.

## Required follow-up (separate source-fix work item, NOT this docs wave)
TDD §11.4.43/§11.4.115: RED test feeding a meta-response polish reply → assert output FALLS BACK
to the pre-polish translation (never writes the meta-text). Then re-record + re-confirm.
