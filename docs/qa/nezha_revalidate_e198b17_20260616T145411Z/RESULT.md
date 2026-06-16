# Re-validation of e198b17 (commentary-aside strip fix) on nezha — §11.4.130 / §11.4.108

**Run UTC:** 20260616T145411Z
**Commit:** e198b17 (on 6a1aa8c)
**Live image BEFORE this redeploy:** server 2d0c925, grpc 2d0c925
**Live image AFTER:** server/grpc/api all **ebb82df46d9c** (contamination fix)
**Surface:** server-TLS `https://nezha.local:18443/api/v1/translate`

## Targeted-fix validation FIRST (§11.4.130)
| Check | Result |
|---|---|
| unsupported provider `nope-xyz` | HTTP **400** `{"error":"unsupported provider: nope-xyz"}` — task-1 fix still holds (no regression) |
| valid `deepseek` ru→sr | HTTP **200** `translated:"Добро јутро."` (real Serbian, no regression) |
| 18× ru→sr clean batch | **18/18 HTTP 200**, contaminated=**0/18** (parenthetical/bracket meta-aside + newline-commentary scan) |

The exact source that produced the pre-fix contamination — `Я пью кофе утром.` →
pre-fix `'Ја пијем кафу јутро.\n\n(Using Ekavica dialect ... as per guidelines)'` —
now returns **clean** `'Ја пијем кафу јутро.'` on the live binary.

## Defect → fix (§11.4.4 / §11.4.102 / §11.4.115 / §11.4.135)
- **FACT root cause:** `isCommentaryBlock` only stripped a trailing fully-enclosed `()`/`[]` block when its inner text started with `note` or contained `translat`. A live aside `(Using Ekavica dialect and pure Serbian vocabulary as per guidelines)` matched neither → leaked into the translation.
- **Fix:** widened parenthetical-inner detection with `commentaryParenSignalWords` (using/dialect/vocabulary/guideline/register/as per/i used/…). Genuine in-content trailing parentheticals preserved (guard test).
- **Anti-bluff:** strip_commentary_test.go RED fixtures (sr_paren_dialect_aside, en_bracket_style_aside) FAIL pre-fix / PASS post-fix; KeepsBenignTrailingParenthetical over-strip guard; §1.1 mutation (remove signal-word loop → FAIL → restore → PASS); full `pkg/translator/llm` + `pkg/translator` GREEN; vet clean.

## Honest boundary (§11.4.6)
- Contamination is intermittent (1/20 pre-fix). The DETERMINISTIC unit test + §1.1 mutation are the authoritative proof the stripping logic catches the class; the live 18/18-clean batch is corroborating runtime evidence (§11.4.108 runtime signature = the live binary contains + runs the widened stripper).
- Translation *quality* on the novita-routed model is imperfect (e.g. cat→"Коња"/horse). That is a model-selection quality concern, OUT of scope for the contamination defect — queued separately if pursued.

## Verdict: PASS — commentary-aside contamination class eliminated on the live stack; task-1 unsupported-provider fix unregressed.
