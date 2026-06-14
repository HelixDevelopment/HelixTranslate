# E2E proof — DOCX input → real DeepSeek translation → EPUB (§11.4.83/§11.4.107/§11.4.108)

**Revision:** 1
**Last modified:** 2026-06-14T13:19:00Z

Proves DOCX input now works end-to-end through the real CLI after the stdlib DOCX
parser rewrite (d433210) AND the detector-support wiring fix in this commit.

Before this commit the CLI rejected DOCX with `format docx is not yet supported`
(detector.IsSupported excluded FormatDOCX — a §11.4.108 source→artifact gap: the
parser was fixed/registered but the pipeline gate still refused DOCX). Caught by a
broader multi-format E2E proof.

## Command (real system, real network)
```
./build/unified-translator -i test/assets/crow.docx -o out.epub \
  -source-lang en -target-lang sr -provider deepseek -model deepseek-chat
# => Translation completed successfully | duration=2.05s (exit 0)
```

## Result (real Serbian Cyrillic from a real .docx)
`Гавран и крчаг … Жедан гавран пронашао је крчаг с мало воде на дну. Убацивао је каменчиће један по један док није могао да пије.`

DOCX (zip OOXML) → parsed by the stdlib parser → translated by real DeepSeek →
valid EPUB output. No placeholders, no key leak (§11.4.10).
