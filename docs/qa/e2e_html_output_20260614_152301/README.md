# E2E Proof — .html output format added (completes the output matrix)

**Date:** 2026-06-14
**Constitution:** §11.4 anti-bluff, §11.4.107 captured evidence, §11.4.135 regression guard, §11.4.146 extend-to-all-cases

## Change
The output dispatcher (generateOutput) supported .epub/.fb2/.txt/.md and errored
on .html, while the INPUT side fully supports HTML — an asymmetric gap. Added a
.html/.htm case producing a minimal, valid, well-formed HTML5 document with
HTML-escaped content (no markup injection from translated text).

## Real-system proof (this directory)
English PDF → real DeepSeek deepseek-chat → out.html:
- `file` reports "HTML document text, UTF-8"
- `<!DOCTYPE html>` present, 2 `<p>` paragraphs, 140 Cyrillic chars
- NOT a zip (does not start with PK)
- real Serbian content: "Храбри витез јахао је преко тихог пропланка у зору. ..."
- run.log — translator log (no API-key printing)

## Regression guard (§11.4.135)
cmd/unified-translator/output_format_test.go adds:
- "html is well-formed escaped HTML" — DOCTYPE + <p> + content present, not a zip
- "html escapes content (no markup injection)" — raw <script> NOT present;
  &lt;script&gt; IS present (proves html.EscapeString applied)
Mutation: forcing generateOutput to always-EPUB → html subtest FAILs (proven for
the dispatcher this session).
