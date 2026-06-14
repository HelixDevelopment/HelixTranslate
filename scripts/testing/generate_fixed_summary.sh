#!/usr/bin/env bash
# scripts/testing/generate_fixed_summary.sh
#
# Purpose:   Regenerate docs/Fixed_Summary.md from docs/Fixed.md (§11.4.53
#            Fixed_Summary parity + §11.4.54 ATM-NNN leftmost column +
#            §11.4.19 column-alignment + §11.4.91 clear one-line descriptions).
# Usage:     scripts/testing/generate_fixed_summary.sh [FIXED_MD] [SUMMARY_MD]
#            (defaults: docs/Fixed.md → docs/Fixed_Summary.md)
# Inputs:    docs/Fixed.md — headings of form `### §N. [ATM-NNN] <title>` each
#            followed (within 8 lines) by `**Status:**` and `**Type:**` lines.
# Outputs:   docs/Fixed_Summary.md — a markdown table
#            `ATM ID | Level | Status | Type | One-line description`.
# Side-effects: overwrites the summary file only (never the source).
# Dependencies: POSIX awk, coreutils. No network. Parses clean under sh -n + bash -n.
# Cross-references: companion generate_issues_summary.sh; doc docs/scripts/generate_fixed_summary.md.

set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/../.." && pwd)"

SRC="${1:-$ROOT/docs/Fixed.md}"
OUT="${2:-$ROOT/docs/Fixed_Summary.md}"

[ -f "$SRC" ] || { echo "error: source not found: $SRC" >&2; exit 2; }

NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

{
  printf '# HelixTranslate — Fixed Summary\n\n'
  printf '**Revision:** 1\n'
  printf '**Last modified:** %s\n' "$NOW"
  printf '**Authority:** §11.4.53 (Fixed_Summary parity) · §11.4.54 (ATM ID column) · §11.4.19 (column-alignment) · §11.4.91 (clear descriptions)\n'
  printf '**Generated:** auto-generated from `docs/Fixed.md` by `scripts/testing/generate_fixed_summary.sh` — do not hand-edit.\n\n'
  printf '| ATM ID | Level | Status | Type | One-line description |\n'
  printf '| --- | --- | --- | --- | --- |\n'

  awk '
    function flush() {
      if (atm != "") {
        level = (type == "Bug") ? "Bug" : (type == "Feature" ? "Feature" : "Task")
        printf "| %s | %s | %s | %s | %s |\n", atm, level, status, type, desc
      }
      atm = ""; status = ""; type = ""; desc = ""
    }
    /^### / {
      flush()
      line = $0
      if (match(line, /\[ATM-[0-9]+\]/)) {
        atm = substr(line, RSTART+1, RLENGTH-2)
        rest = substr(line, RSTART+RLENGTH)
        sub(/^[[:space:]]+/, "", rest)
        desc = rest
      }
      next
    }
    /^\*\*Status:\*\*/ { s = $0; sub(/^\*\*Status:\*\*[[:space:]]*/, "", s); status = s; next }
    /^\*\*Type:\*\*/   { t = $0; sub(/^\*\*Type:\*\*[[:space:]]*/, "", t); type = t; next }
    END { flush() }
  ' "$SRC"

  printf '\nTotal closed items: '
  grep -c '^### .*\[ATM-' "$SRC" || true
} > "$OUT"

echo "wrote $OUT"
