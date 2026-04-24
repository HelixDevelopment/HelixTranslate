#!/usr/bin/env bash
# Run the acceptance demo from every CLAUDE.md with one.
#
# Auto-discovers CLAUDE.md files under the repo, extracts the first ```bash```
# block from the "### Acceptance demo for this module" section, runs it with
# a timeout, and reports per-module PASS / FAIL / TODO / NO-DEMO.
#
# A module that hasn't yet written an acceptance demo reports NO-DEMO — that's
# the signal to go write one. The existence of the demo IS the gate.
#
# Env:
#   DEMO_TIMEOUT        per-demo timeout in seconds (default: 180)
#   DEMO_LOG_DIR        where per-module logs land (default: reports/demos)
#   DEMO_MODULES        space-separated subset to run (default: all auto-discovered)
#   DEMO_ALLOW_TODO     pass "1" to treat TODO placeholders as warnings
#   DEMO_ALL_WARN_ONLY  pass "1" to log failures but exit 0 (transition mode)
#   DEMO_EXCLUDE_DIRS   colon-separated directory names to skip during discovery
#                       (defaults: vendor node_modules target build .gradle)

set -uo pipefail
cd "$(dirname "$0")/.."  # repo root (script installed under scripts/)
ROOT="$PWD"

TIMEOUT="${DEMO_TIMEOUT:-180}"
LOG_DIR="${DEMO_LOG_DIR:-reports/demos}"
ALLOW_TODO="${DEMO_ALLOW_TODO:-0}"
mkdir -p "$LOG_DIR"

DEFAULT_EXCLUDES="vendor:node_modules:target:build:.gradle:.git:releases:reports:dist:.next:.venv"
EXCLUDES="${DEMO_EXCLUDE_DIRS:-$DEFAULT_EXCLUDES}"
prune_expr=()
IFS=':' read -r -a exarr <<< "$EXCLUDES"
for d in "${exarr[@]}"; do
  [ -n "$d" ] && prune_expr+=(-name "$d" -o)
done
# Remove trailing -o
if [ ${#prune_expr[@]} -gt 0 ]; then
  unset 'prune_expr[${#prune_expr[@]}-1]'
fi

discover_modules() {
  # Print every directory (relative to root) that contains a CLAUDE.md.
  # Root-level CLAUDE.md is reported as ".".
  local raw
  if [ ${#prune_expr[@]} -gt 0 ]; then
    raw=$(find . \( "${prune_expr[@]}" \) -prune -o -name CLAUDE.md -print 2>/dev/null)
  else
    raw=$(find . -name CLAUDE.md 2>/dev/null)
  fi
  printf '%s\n' "$raw" \
    | awk 'NF > 0 {
        # strip leading ./
        sub(/^\.\//, "")
        # if only "CLAUDE.md" remains, it is the repo root
        if ($0 == "CLAUDE.md") { print "."; next }
        # otherwise strip /CLAUDE.md suffix
        sub(/\/CLAUDE\.md$/, "")
        print
      }' \
    | sort -u
}

if [ -n "${DEMO_MODULES:-}" ]; then
  # shellcheck disable=SC2206
  MODULES=($DEMO_MODULES)
else
  mapfile -t MODULES < <(discover_modules)
fi

pass=0; fail=0; todo=0; missing=0
fail_list=(); todo_list=(); missing_list=()

extract_demo() {
  awk '
    /^### Acceptance demo for this module/ { state = 1; next }
    state == 1 && /^```bash/              { state = 2; next }
    state == 2 && /^```/                  { exit }
    state == 2                            { print }
  ' "$1"
}

is_todo() {
  grep -qE '^[[:space:]]*#[[:space:]]*TODO[[:space:]]*$' <<< "$1" \
    && [ "$(printf '%s\n' "$1" | grep -cvE '^\s*$')" -le 1 ]
}

for mod in "${MODULES[@]}"; do
  # Root CLAUDE.md uses "." as its module id
  if [ "$mod" = "." ]; then
    md="CLAUDE.md"
    mod_label="(root)"
  else
    md="$mod/CLAUDE.md"
    mod_label="$mod"
  fi
  if [ ! -f "$md" ]; then
    echo "[NO-DEMO]  $mod_label (no CLAUDE.md)"
    missing=$((missing + 1)); missing_list+=("$mod_label")
    continue
  fi
  demo=$(extract_demo "$md")
  if [ -z "$demo" ]; then
    echo "[NO-DEMO]  $mod_label (no bash block in acceptance-demo section)"
    missing=$((missing + 1)); missing_list+=("$mod_label")
    continue
  fi
  if is_todo "$demo"; then
    echo "[TODO]     $mod_label (demo still a placeholder)"
    todo=$((todo + 1)); todo_list+=("$mod_label")
    continue
  fi
  log="$LOG_DIR/${mod_label//\//_}.log"
  echo "[RUN]      $mod_label"
  if timeout "$TIMEOUT" bash -c "$demo" > "$log" 2>&1; then
    echo "[PASS]     $mod_label"
    pass=$((pass + 1))
  else
    rc=$?
    if [ "$rc" -eq 124 ]; then
      echo "[FAIL]     $mod_label (timeout after ${TIMEOUT}s — log: $log)"
    else
      echo "[FAIL]     $mod_label (exit $rc — log: $log)"
    fi
    fail=$((fail + 1)); fail_list+=("$mod_label")
  fi
done

echo
echo "================================================================"
echo "demo-all totals: PASS=$pass FAIL=$fail TODO=$todo NO-DEMO=$missing"
echo "================================================================"

[ "$fail" -gt 0 ]    && { echo "FAIL modules:"; printf '  - %s\n' "${fail_list[@]}"; }
[ "$todo" -gt 0 ]    && { echo "TODO modules:"; printf '  - %s\n' "${todo_list[@]}"; }
[ "$missing" -gt 0 ] && { echo "NO-DEMO modules:"; printf '  - %s\n' "${missing_list[@]}"; }

failed=0
if [ "$fail" -gt 0 ] || [ "$missing" -gt 0 ]; then
  failed=1
fi
if [ "$todo" -gt 0 ] && [ "$ALLOW_TODO" != "1" ]; then
  echo "TODO demos failing the run. Set DEMO_ALLOW_TODO=1 to treat as warnings during transition." >&2
  failed=1
fi
if [ "$failed" -eq 1 ]; then
  if [ "${DEMO_ALL_WARN_ONLY:-0}" = "1" ]; then
    echo "(warn-only mode — set DEMO_ALL_WARN_ONLY=0 to fail the build)" >&2
    exit 0
  fi
  exit 1
fi
