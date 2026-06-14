# sync_all_markdown_exports.sh

**Revision:** 1
**Last modified:** 2026-06-14T15:58:01Z

## Overview
Generates and refreshes synchronized `.html` + `.pdf` siblings for every in-scope Markdown document in the project, satisfying §11.4.65 (universal Markdown export mandate). HTML is rendered with `pandoc`; the PDF is rendered with `weasyprint` **from the generated HTML** so both formats stay visually consistent. The siblings are derived artifacts — never hand-edit them; edit the source `.md` and re-run.

## Prerequisites
- `pandoc` (HTML rendering) and `weasyprint` (PDF rendering) on `PATH`.
- coreutils (`find`, `sort`, `wc`, `head`, `timeout`). No network. No credentials.
- If a tool is absent the script SKIPs that format with an honest WARN and never writes a fake/empty/partial sibling (§11.4.106).

## Usage
```bash
scripts/testing/sync_all_markdown_exports.sh            # generate missing/stale siblings
scripts/testing/sync_all_markdown_exports.sh --check    # gate mode: exit 1 if any in-scope .md lacks a fresh sibling
scripts/testing/sync_all_markdown_exports.sh --force    # regenerate every sibling regardless of mtime
```

## Internal behaviour
In-scope per §11.4.65: project-root `*.md` (maxdepth 1), `docs/**/*.md`, and `scripts/**/*.md` companion docs. Excluded: `.git`, `external/`, `prebuilts/`, `build/`, `out/`, `dist/`, `node_modules/`, `vendor/`, the gitignored `qa-results/`, and the owned submodule directories (`challenges/`, `containers/`, `helix_qa/`, `doc_processor/`, `llm_orchestrator/`, `llm_provider/`, `vision_engine/`, `llms_verifier/`, `constitution/`, `docs_chain/`) — each submodule owns its own exports. The in-scope set is hard-capped at 500 files.

A sibling is regenerated only when it is **missing or older** than its source `.md` (idempotent), unless `--force` is given. Each generated `.html` is validated non-empty; each `.pdf` is validated non-empty **and** to start with the `%PDF` magic before being accepted — a render that fails any check is deleted (never left partial) and reported loudly with a non-zero exit (§11.4.1 anti-bluff: failures are loud, not silent).

## Edge cases
- Missing `pandoc` ⇒ all HTML skipped (WARN); missing `weasyprint` ⇒ all PDF skipped (WARN); neither fabricates output.
- A `.md` touched concurrently with the run will read stale on the next `--check`; re-run to converge (the run is idempotent).
- Each render is wrapped in `timeout 60` so one pathological file cannot stall the whole pass.

## Related scripts
- `scripts/testing/generate_issues_summary.sh`, `scripts/testing/generate_fixed_summary.sh` — derived-doc generators whose outputs this script then exports.
- `scripts/commit_all.sh` — project commit wrapper (stage the script + generated siblings via explicit paths).

## Constitution references
§11.4.65 (universal Markdown export), §11.4.18 (script documentation), §11.4.106 (no fake transform / honest tool-absence skip), §11.4.67 (target-shell-parseable — parses clean under `bash -n` and `sh -n`), §11.4.1 (FAIL-bluffs forbidden).

_Last verified: 2026-06-14 — pandoc 3.9.0.2 + WeasyPrint 66.0, 129 in-scope `.md`, 0 stale after run._
