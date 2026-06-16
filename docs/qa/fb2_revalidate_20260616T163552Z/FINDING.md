# FB2 BUG-FB2-HARDCODED-LANG — live re-validation on nezha (§11.4.108 runtime signature)

**Run:** 2026-06-16T16:38:12Z · **Image:** `2bb4de5df2c7` (helixtranslate-api, confirmed via `podman inspect`)
**Endpoint:** `POST https://nezha.local:18443/api/v1/translate/fb2` (cmd/server, pkg/api Handler.translateFB2)
**Fix under test:** commit `c2aa7c8` — handler parses+validates `source_lang`/`target_lang`, threads resolved codes (legacy ru→sr kept ONLY when both omitted), unknown → 400.
**Fixture:** `test/fixtures/ebooks/sample.fb2` (English `<lang>en</lang>` source).

## Result: PASS (3 probes, real captured evidence — §11.4.123)

| Probe | Request | Expected | Observed | Verdict |
|---|---|---|---|---|
| A (positive) | `target_lang=es` | Spanish output, NOT Serbian | EPUB w/ "Capítulo de Prueba" · "Este es un documento de prueba FB2 para la validación de la traducción." · "Sección 1.1" | **PASS** |
| C (cross-check) | `target_lang=de` | German, distinct from A | EPUB w/ "Testkapitel" · "Dies ist ein Test-FB2-Dokument zur Prüfung der Übersetzung." · "Abschnitt 1.1" | **PASS** |
| B (negative) | `target_lang=klingon` | HTTP 400, no translation | `{"error":"unknown target_lang: klingon"}` HTTP 400 | **PASS** |

## Why this proves the fix (§11.4.6 no-guessing)
- Source is **English**. Pre-fix, the handler hardcoded ru→sr and ignored `target_lang`, so EVERY FB2 request produced **Serbian**. Probe A produced **Spanish**, Probe C produced **German** (distinct outputs) — only possible if `target_lang` is honored.
- Probe B's 400 with the exact `unknown target_lang: klingon` message proves the validate-BEFORE-translate path is live on the deployed image (pre-fix would silently fall back).
- Output is a real `application/epub+zip` artifact (EPUB document, ffile-verified), with translated chapter prose — not a session_id/status (§11.4 anti-bluff).

## Artifacts
- `probe_a_es_output.epub` + `epub_extracted/OEBPS/chapter1.xhtml` (Spanish)
- `probe_c_de_output.epub` + `epub_de_extracted/OEBPS/chapter1.xhtml` (German)
- `probe_b_klingon_400.txt` (400 response)
- `probe_a_es_meta.txt` / `probe_c_de_meta.txt` (HTTP status + content-type + size)
- `run_meta.txt` (run context)
