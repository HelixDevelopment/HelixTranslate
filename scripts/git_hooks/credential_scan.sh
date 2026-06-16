#!/usr/bin/env bash
# credential_scan.sh — shared credential-pattern scanner for git hooks.
#
# Purpose:    Reject commits/pushes that introduce inline secret-shaped literals
#             (the class that §11.4.10 / §11.4.10.A clause 5 require to be caught
#             mechanically, NOT just secret FILES caught by .gitignore).
# Usage:      credential_scan.sh <file>...        (scans the given working-tree files)
#             credential_scan.sh --staged          (scans the staged diff)
# Inputs:     file paths OR --staged
# Outputs:    prints offending file:line (value MASKED, never the secret) to stderr;
#             exit 0 = clean, exit 1 = leak found.
# Side-effects: none (read-only).
# Dependencies: bash, git, grep, perl.
# Cross-references: scripts/install_git_hooks.sh, scripts/git_hooks/{pre-commit,pre-push};
#                   docs/scripts/credential_scan.md; constitution §11.4.10 / §11.4.10.A / §11.4.75.
#
# Pattern set (closed, extend deliberately):
#   - known leaked-class token shape: WhiteSnake<digits> (the historical leak class)
#   - generic inline password value:  -password[= ]"?<literal>  (NOT ${VAR}, NOT "", NOT prompt)
#   - REMOTE_PASS / *_PASSWORD = "<literal>"  with a non-env, non-empty value
#   - common API-key literal shapes: sk-..., AKIA..., ghp_..., xoxb-...
set -euo pipefail

# Files this hook should never flag (they legitimately document the pattern class /
# the historical token NAME by design). Matched against the path TAIL so it works
# whether the path is repo-relative (pre-commit --staged) or absolute (pre-push).
IGNORE_RE='(^|/)(constitution/|scripts/git_hooks/|scripts/testing/test_credential_scan\.sh|docs/scripts/credential_scan\.md|docs/qa/secret_scrub_)'

# A value counts as a "literal secret" only if it is NOT a shell/expect var ref,
# NOT an empty string, NOT a redaction marker, NOT an obvious placeholder.
is_env_or_placeholder() {
  case "$1" in
    '${'*|'$'*|'$env('*) return 0 ;;          # ${VAR} / $VAR / $env(VAR)
    ''|'""'|"''") return 0 ;;                  # empty
    '<redacted'*|'<'*'>'|'YOUR_'*|'CHANGEME'*|'xxx'*|'***'*) return 0 ;;
    '&lt;redacted'*|'&lt;'*'&gt;') return 0 ;;  # HTML-escaped redaction/placeholder markers (.html exports); mirrors '<redacted…>'/'<…>'
    *) return 1 ;;
  esac
}

scan_content() {
  # stdin: lines of form "<path>:<lineno>:<content>"
  local found=0 line path lno body
  while IFS= read -r line; do
    path="${line%%:*}"; rest="${line#*:}"; lno="${rest%%:*}"; body="${rest#*:}"
    [[ "$path" =~ $IGNORE_RE ]] && continue

    # 1) historical leaked-class token shape
    if printf '%s' "$body" | grep -qE 'WhiteSnake[0-9]{2,}'; then
      echo "CREDENTIAL LEAK (leaked-class token) -> $path:$lno  [value masked]" >&2; found=1; continue
    fi
    # 2) inline -password / --password FLAG value.
    #    Anchored on a word boundary (start or whitespace) so "SSH-password" prose
    #    in a comment does NOT match — only a real CLI flag token does. The value
    #    must be credential-shaped (contains a digit OR length >= 8) to avoid
    #    flagging a following dictionary word.
    if printf '%s' "$body" | grep -qE '(^|[[:space:]])--?password[= ]+"?[^"$ ]'; then
      val=$(printf '%s' "$body" | perl -ne 'print $1 if /(?:^|\s)--?password[= ]+"?([^"\s\\]+)/')
      if [ -n "$val" ] && ! is_env_or_placeholder "$val" \
         && printf '%s' "$val" | grep -qE '[0-9]|^.{8,}$'; then
        echo "CREDENTIAL LEAK (inline -password literal) -> $path:$lno  [value masked]" >&2; found=1; continue
      fi
    fi
    # 3) *_PASS/*_PASSWORD = "literal" / Password: "literal"
    if printf '%s' "$body" | grep -qE '([A-Za-z_]*PASS(WORD)?|Password)[: ]*=?[ ]*"[^"$]'; then
      val=$(printf '%s' "$body" | perl -ne 'print $1 if /(?:PASS(?:WORD)?|Password)[: ]*=?\s*"([^"]+)"/')
      if [ -n "$val" ] && ! is_env_or_placeholder "$val"; then
        echo "CREDENTIAL LEAK (inline password assignment) -> $path:$lno  [value masked]" >&2; found=1; continue
      fi
    fi
    # 4) common provider API-key literal shapes
    if printf '%s' "$body" | grep -qE '(sk-[A-Za-z0-9]{16,}|AKIA[0-9A-Z]{12,}|ghp_[A-Za-z0-9]{20,}|xox[baprs]-[A-Za-z0-9-]{10,})'; then
      echo "CREDENTIAL LEAK (API-key literal shape) -> $path:$lno  [value masked]" >&2; found=1; continue
    fi
  done
  return $found
}

if [ "${1:-}" = "--staged" ]; then
  # Scan only ADDED lines in the staged diff, mapped to file:line.
  git diff --cached --unified=0 --no-color | \
    perl -ne '
      if (/^\+\+\+ b\/(.+)$/) { $f=$1; $n=0; next }
      if (/^@@ -\d+(?:,\d+)? \+(\d+)/) { $n=$1; next }
      if (/^\+(.*)$/ && $f) { print "$f:$n:$1\n"; $n++ }
    ' | scan_content
else
  for f in "$@"; do
    [ -f "$f" ] || continue
    grep -nH '' "$f" | sed 's/^\([^:]*\):\([0-9]*\):/\1:\2:/' | scan_content || exit 1
  done
fi
