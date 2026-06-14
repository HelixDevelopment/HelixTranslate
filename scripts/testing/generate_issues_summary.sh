#!/usr/bin/env bash
# scripts/testing/generate_issues_summary.sh
#
# Purpose:   Regenerate docs/Issues_Summary.md from docs/Issues.md (§11.4.12
#            Issues_Summary sync + §11.4.54 ATM-NNN leftmost column +
#            §11.4.19 column-alignment + §11.4.91 clear one-line descriptions).
# Usage:     scripts/testing/generate_issues_summary.sh [ISSUES_MD] [SUMMARY_MD]
#            (defaults: docs/Issues.md → docs/Issues_Summary.md)
# Inputs:    docs/Issues.md — headings of form `### §N. [ATM-NNN] <title>` each
#            followed (within 8 lines) by `**Status:**` and `**Type:**` lines.
# Outputs:   docs/Issues_Summary.md — a markdown table
#            `ATM ID | Level | Status | Type | One-line description`.
# Side-effects: overwrites the summary file only (never the source).
# Dependencies: POSIX awk, coreutils. No network. Parses clean under sh -n + bash -n.
# Cross-references: companion generate_fixed_summary.sh; doc docs/scripts/generate_issues_summary.md.

set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/../.." && pwd)"

SRC="${1:-$ROOT/docs/Issues.md}"
OUT="${2:-$ROOT/docs/Issues_Summary.md}"

[ -f "$SRC" ] || { echo "error: source not found: $SRC" >&2; exit 2; }

NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

{
  printf '# HelixTranslate — Issues Summary\n\n'
  printf '**Revision:** 1\n'
  printf '**Last modified:** %s\n' "$NOW"
  printf '**Authority:** §11.4.12 (Issues_Summary sync) · §11.4.54 (ATM ID column) · §11.4.19 (column-alignment) · §11.4.91 (clear descriptions)\n'
  printf '**Generated:** auto-generated from `docs/Issues.md` by `scripts/testing/generate_issues_summary.sh` — do not hand-edit.\n\n'
  printf '| ATM ID | Level | Status | Type | One-line description |\n'
  printf '| --- | --- | --- | --- | --- |\n'

  awk '
    function flush() {
      if (atm != "") {
        # Level derives from Status urgency: operator-blocked/blocked = High, design = Medium, queued = Normal.
        ls = tolower(status)
        if (ls ~ /operator-blocked/ || ls ~ /blocked/) level = "High"
        else if (ls ~ /design/) level = "Medium"
        else level = "Normal"
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

  printf '\nTotal open items: '
  grep -c '^### .*\[ATM-' "$SRC" || true
} > "$OUT"

echo "wrote $OUT"
