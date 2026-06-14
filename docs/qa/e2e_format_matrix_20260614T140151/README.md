# E2E proof — all 5 input formats → real DeepSeek → Serbian-Cyrillic EPUB (§11.4.83/§11.4.107/§11.4.25)

**Revision:** 1
**Last modified:** 2026-06-14T14:02:00Z

Real-system, anti-bluff proof that EVERY supported text input format translates
end-to-end after this session's fixes. Each run: real `.<fmt>` asset →
unified-translator → real DeepSeek (deepseek-chat) → valid EPUB; chapter text
extracted + checked for Serbian Cyrillic, no source phrase, no placeholder.

This round also FIXED a real bug: FB2 and EPUB INPUT were broken end-to-end — the
CLI's convertToMarkdown re-wrote already-extracted text to a temp .fb2/.epub and
re-parsed it as that format ("failed to parse FB2: EOF" / "zip: not a valid zip").
Now all formats use the already-extracted content uniformly.

| Input format | Provider | Exit | Cyrillic chars | Verdict |
|---|---|---|---|---|
| FB2  | deepseek | 0 | 101 | PASS |
| EPUB | deepseek | 0 | 123 | PASS |
| TXT  | deepseek | 0 | 204 | PASS |
| DOCX | deepseek | 0 | 113 | PASS |
| HTML | deepseek | 0 | 128 | PASS |

Example (FB2): `Гавран и врч … Жедан гавран пронађе врч с мало воде на дну. Убацивао је каменчиће у врч, један по један…`

Per-format extracted Serbian text: `*_translated_sr.txt` in this directory.
Raw EPUB outputs: qa-results/e2e_matrix2_20260614T140151/ (git-ignored). No key leak (§11.4.10).
