#!/usr/bin/env bash
# Fail if any test skip directive is present without a SKIP-OK: #<ticket> annotation.
#
# Part of the Definition of Done enforcement arm. A skipped test is invisible debt —
# this script makes that debt loud. If a skip is genuinely needed, annotate it with
# a ticket reference:
#
#   t.Skip("flake under race — SKIP-OK: #1234")
#   @Ignore("waiting on upstream — SKIP-OK: #1234")
#   it.skip('pending feature — SKIP-OK: #1234', ...)
#
# Portable across Go / Kotlin / Java / TS / JS / Python / Swift / Rust.
# Wire into: make no-silent-skips  →  make ci-validate-all
#
# Env:
#   NO_SILENT_SKIPS_WARN_ONLY=1   — log violations but exit 0 (transition mode)
#   NO_SILENT_SKIPS_EXCLUDES=...  — colon-separated extra directory names to skip

set -uo pipefail
cd "$(dirname "$0")/.."  # repo root (script installed under scripts/)

PATTERNS='t\.Skip\(|@Ignore\b|\bxit\(|\.skip\(|@pytest\.mark\.skip|@unittest\.skip|#\[ignore\]|XCTSkipIf'
INCLUDES=(--include='*.go' --include='*.kt' --include='*.kts' --include='*.java'
          --include='*.ts' --include='*.tsx' --include='*.js' --include='*.jsx'
          --include='*.py' --include='*.swift' --include='*.rs')

# Default excludes — third-party/vendored/generated trees.
EXCLUDES=(--exclude-dir=.git --exclude-dir=vendor --exclude-dir=node_modules
          --exclude-dir=external --exclude-dir=target --exclude-dir=build
          --exclude-dir=.gradle --exclude-dir=.idea --exclude-dir=dist
          --exclude-dir=releases --exclude-dir=reports --exclude-dir=test-results
          --exclude-dir=.next --exclude-dir=.nuxt --exclude-dir=coverage
          --exclude-dir=.venv --exclude-dir=__pycache__)

# Caller-provided extras (colon-separated directory names).
if [ -n "${NO_SILENT_SKIPS_EXCLUDES:-}" ]; then
  IFS=':' read -r -a extras <<< "$NO_SILENT_SKIPS_EXCLUDES"
  for d in "${extras[@]}"; do
    [ -n "$d" ] && EXCLUDES+=("--exclude-dir=$d")
  done
fi

violations=$(grep -rnE "$PATTERNS" "${INCLUDES[@]}" "${EXCLUDES[@]}" . 2>/dev/null \
             | grep -v -E 'SKIP-OK: #[0-9]+' || true)

if [ -n "$violations" ]; then
  count=$(printf '%s\n' "$violations" | wc -l | tr -d ' ')
  echo "⚠️  $count silent-skip violation(s) detected." >&2
  echo "" >&2
  printf '%s\n' "$violations" | head -30 >&2
  if [ "$count" -gt 30 ]; then
    echo "... ($((count - 30)) more — re-run '$0' without head)" >&2
  fi
  echo "" >&2
  echo "Annotate each with a trailing '// SKIP-OK: #<ticket>' (or '# SKIP-OK: #<ticket>')" >&2
  echo "comment so the skip is tracked, or remove the skip if it is no longer needed." >&2
  if [ "${NO_SILENT_SKIPS_WARN_ONLY:-0}" = "1" ]; then
    echo "" >&2
    echo "(warn-only mode — set NO_SILENT_SKIPS_WARN_ONLY=0 to fail the build)" >&2
    exit 0
  fi
  exit 1
fi

echo "no-silent-skips: OK (no unannotated skip directives found)"
