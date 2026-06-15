# update_readme_doc_links.sh — companion guide

**Revision:** 1
**Last modified:** 2026-06-15T16:30:00Z
**Authority:** §11.4.18 (script documentation) · §11.4.57 (README doc-link section) · §11.4.44 (revision header) · §11.4.67 (target-shell parseability)

## Overview

`scripts/testing/update_readme_doc_links.sh` renders the §11.4.57
"Tracked-Items + Status Documents" section into `README.md`. It discovers the
canonical tracked documents, reads each one's §11.4.44 header fields
(`**Revision:**` and `**Last modified:**`), and rewrites the markdown table
between the section markers. Only the marked region changes; the rest of
`README.md` is preserved verbatim.

## Prerequisites

- POSIX `sh`, `awk`, `sed`, `grep`, `find`, `git`, coreutils. No network.
- `README.md` MUST already contain both section markers:
  - `<!-- doc-link-section:begin -->`
  - `<!-- doc-link-section:end -->`
  The script aborts (exit 3) if either is absent — it never invents the section.

## Usage

```bash
scripts/testing/update_readme_doc_links.sh            # rewrites <repo-root>/README.md
scripts/testing/update_readme_doc_links.sh path/to/README.md
```

Exit codes: `0` success · `2` README not found · `3` markers missing.

## What it links

Linked ONLY when the file actually exists (§11.4.6 — no fabricated rows):

- `docs/Issues.md`, `docs/Issues_Summary.md` (§11.4.12 / §11.4.15 / §11.4.16)
- `docs/Fixed.md`, `docs/Fixed_Summary.md` (§11.4.19 / §11.4.53)
- `docs/CONTINUATION.md` (§12.10)
- every `docs/**/Status.md` (auto-discovered via `find docs -name Status.md`)
  and its sibling `Status_Summary.md` (§11.4.45 / §11.4.56) — this includes
  `docs/features/Status.md` + `docs/features/Status_Summary.md`.

Columns: `Document | Last modified | Revision | Markdown | HTML | PDF`. A `DOCX`
column is added automatically when at least one linked doc has a tracked `.docx`
sibling (the §11.4.153 four-format Feature-Status class). Per-row format cells
that have no on-disk sibling render `—`, never a broken link.

## Edge cases

- A doc with no §11.4.44 header renders `—` for Revision / Last modified rather
  than failing (honest absence, §11.4.6).
- Status docs are sorted by path (`LC_ALL=C sort`) for deterministic, idempotent
  output (§11.4.50 / §11.4.86) — re-running with no doc changes yields no diff.
- The DOCX column appears for the whole table (not per-row) so alignment stays
  uniform; rows without a `.docx` show `—`.

## Internal behaviour

1. Resolve repo root via `git rev-parse --show-toplevel` (fallback: two levels up).
2. Build an ordered candidate list (root canonical set, then discovered Status
   pairs) into a temp file; skip non-existent files.
3. Decide whether to emit the DOCX column.
4. Extract header fields with `awk` (scoped to the first 40 lines).
5. Render the section body to a temp file, then splice it between the markers
   with `awk` (no in-place `sed` dialect risk), and write back to `README.md`.

## Related scripts

- `scripts/testing/generate_issues_summary.sh` — Issues_Summary generator.
- `scripts/testing/generate_fixed_summary.sh` — Fixed_Summary generator.
- The §11.4.59 README export refresh (pandoc HTML + weasyprint PDF) is run by the
  caller after this script so `README.html` / `README.pdf` stay in sync.

## Last verified

- 2026-06-15 — `sh -n` + `bash -n` clean; generator run populated the README
  section with 7 doc rows (Issues, Issues Summary, Fixed, Fixed Summary,
  CONTINUATION, features Status, features Status Summary) and re-run was a no-op
  diff (idempotent).
