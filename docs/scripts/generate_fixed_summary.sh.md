# generate_fixed_summary.sh

**Revision:** 1
**Last modified:** 2026-06-14T15:37:53Z

## Overview
Regenerates `docs/Fixed_Summary.md` from `docs/Fixed.md`, satisfying §11.4.53 (Fixed_Summary parity), §11.4.54 (ATM-NNN leftmost column), §11.4.19 (column-alignment with Status + Type), and §11.4.91 (clear one-line descriptions). The summary is a derived artifact — never hand-edit it; edit `docs/Fixed.md` and re-run.

## Prerequisites
- POSIX `awk`, coreutils (`date`, `grep`). No network. No credentials.

## Usage
```bash
scripts/testing/generate_fixed_summary.sh                 # docs/Fixed.md → docs/Fixed_Summary.md
scripts/testing/generate_fixed_summary.sh SRC.md OUT.md   # explicit paths
```

## Internal behaviour
Each `### §N. [ATM-NNN] <title>` heading in the source is parsed for its ATM id and one-line description (the heading text after the id); the following `**Status:**` and `**Type:**` lines (within the entry) supply the table cells. Emits `ATM ID | Level | Status | Type | One-line description` plus a total count.

## Edge cases
- A heading with no `[ATM-NNN]` token is skipped.
- Missing Status/Type render as empty cells (surfaces an under-specified entry).
- Overwrites only the summary file; the source is never modified.

## Related scripts
- `scripts/testing/generate_issues_summary.sh` — the open-tracker companion.

Last verified: 2026-06-14 (sh -n + bash -n clean; round-trip produced 64-row table).
