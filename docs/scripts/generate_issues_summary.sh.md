# generate_issues_summary.sh

**Revision:** 1
**Last modified:** 2026-06-14T15:37:53Z

## Overview
Regenerates `docs/Issues_Summary.md` from `docs/Issues.md`, satisfying §11.4.12 (Issues_Summary sync), §11.4.54 (ATM-NNN leftmost column), §11.4.19 (column-alignment with Status + Type), and §11.4.91 (clear one-line descriptions). The summary is a derived artifact — never hand-edit it; edit `docs/Issues.md` and re-run.

## Prerequisites
- POSIX `awk`, coreutils (`date`, `grep`). No network. No credentials.

## Usage
```bash
scripts/testing/generate_issues_summary.sh                 # docs/Issues.md → docs/Issues_Summary.md
scripts/testing/generate_issues_summary.sh SRC.md OUT.md   # explicit paths
```

## Internal behaviour
Each `### §N. [ATM-NNN] <title>` heading is parsed for its ATM id and one-line description; the following `**Status:**` and `**Type:**` lines supply the cells. The `Level` column derives from Status urgency: Operator-blocked / Blocked → `High`, Design → `Medium`, Queued → `Normal`. Emits `ATM ID | Level | Status | Type | One-line description` plus a total open count.

## Edge cases
- A heading with no `[ATM-NNN]` token is skipped.
- Missing Status/Type render as empty cells.
- Overwrites only the summary file; the source is never modified.

## Related scripts
- `scripts/testing/generate_fixed_summary.sh` — the closed-archive companion.

Last verified: 2026-06-14 (sh -n + bash -n clean; round-trip produced 17-row table).
