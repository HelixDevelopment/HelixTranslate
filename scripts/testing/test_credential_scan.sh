#!/usr/bin/env bash
# test_credential_scan.sh — permanent regression guard for the credential scanner.
#
# Purpose:    Pin the behaviour of scripts/git_hooks/credential_scan.sh so two
#             failure modes can never silently return:
#               (A) FALSE POSITIVE — an HTML-escaped redaction marker
#                   (&lt;redacted...&gt;) in a committed .html export being
#                   flagged as a leak (the 2026-06-16 defect this guards).
#               (B) FALSE NEGATIVE — a real inline secret slipping past the
#                   placeholder exemptions (proves the exemption is not a hole).
# Usage:      bash scripts/testing/test_credential_scan.sh
# Inputs:     none (self-contained temp fixtures).
# Outputs:    PASS/FAIL lines to stdout; exit 0 = all pass, exit 1 = any fail.
# Side-effects: writes + removes temp fixtures only.
# Dependencies: bash, the scanner under test.
# Cross-references: scripts/git_hooks/credential_scan.sh; docs/scripts/credential_scan.md;
#                   constitution §11.4.10 / §11.4.10.A / §11.4.115 / §11.4.135.
#
# §11.4.115 polarity: RED_MODE=1 reproduces defect (A) on a pre-fix scanner that
# lacks the &lt;...&gt; exemption (the case below MUST be flagged → guard asserts
# it is NOT, so RED_MODE=1 against a reverted scanner FAILs). Default RED_MODE=0
# is the standing GREEN guard.
set -euo pipefail

HERE="$(cd "$(dirname "$0")/../.." && pwd)"
SCANNER="$HERE/scripts/git_hooks/credential_scan.sh"
RED_MODE="${RED_MODE:-0}"
fails=0

run_case() { # <description> <expected-exit 0|1> <file-content>
  local desc="$1" want="$2" content="$3" tmp got
  tmp="$(mktemp)"; printf '%s\n' "$content" > "$tmp"
  set +e; bash "$SCANNER" "$tmp" >/dev/null 2>&1; got=$?; set -e
  rm -f "$tmp"
  if [ "$got" = "$want" ]; then
    echo "PASS  $desc (exit $got)"
  else
    echo "FAIL  $desc (got exit $got, want $want)"; fails=$((fails+1))
  fi
}

# (A) HTML-escaped redaction markers in .html exports MUST be exempt (exit 0).
run_case "HTML-escaped redaction marker is exempt" 0 'REMOTE_PASS="&lt;redacted-per-§11.4.10&gt;"'
run_case "HTML-escaped generic angle placeholder is exempt" 0 'REMOTE_PASSWORD="&lt;ssh-pw-redacted&gt;"'
# Pre-existing literal exemptions stay intact.
run_case "literal <redacted> marker is exempt" 0 'REMOTE_PASS="<redacted-per-§11.4.10>"'
run_case "empty value is exempt" 0 'REMOTE_PASS=""'
run_case 'env-var ref is exempt' 0 'REMOTE_PASS="${SSH_WORKER_PASSWORD}"'
# (B) Real inline secrets MUST still be caught (exit 1) — the exemption is not a hole.
run_case "real inline password literal is caught" 1 'REMOTE_PASS="hunter2isReal456"'
run_case "real --password flag literal is caught" 1 'cmd --password supersecret99'

echo "---"
if [ "$fails" -eq 0 ]; then
  echo "RESULT: PASS — credential scanner behaviour pinned (false-positive A guarded, false-negative B guarded)"
  exit 0
else
  echo "RESULT: FAIL — $fails case(s) regressed"
  exit 1
fi
