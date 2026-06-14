#!/usr/bin/env bash
#
# ============================================================================
# pre_build_verification.sh — helix_translate pre-build gate suite (P4.2)
# ============================================================================
#
# Purpose:
#   Run a small set of GENUINE, high-value, mechanically-checkable pre-build
#   gates BEFORE every build (Constitution: MANDATORY TESTING CONSTRAINTS —
#   "Pre-build / pre-merge verification BEFORE every build"). Each gate fails
#   ONLY on a real, user-visible repository-hygiene violation — never on a
#   script-internal bug (§11.4.1) — and each is backed by a PAIRED §1.1
#   mutation test in scripts/testing/ that proves the gate is not a bluff.
#
#   This is the FIRST real CM-* gate runner for this project (prior to P4.2
#   only the constitution-inheritance meta-test existed).
#
# Implemented gates:
#   CM-GITIGNORE-PRECOMMIT-AUDIT  (§11.4.30) — no tracked file matches the
#       forbidden build-artifact / secret-data classes the project's own
#       .gitignore declares; INCLUDES the cmd/api-server regression guard
#       (the prebuilt root 'api-server' binary must stay anchored as
#       '/api-server' so the bare pattern never re-hides the SOURCE dir).
#   CM-NO-FAKES-BEYOND-UNIT      (§11.4.27) — non-unit Go test files
#       (integration/e2e/full-automation, identified by build tag) do not
#       import a mock/stub package path; mocks are permitted in unit tests
#       only. Best-effort, AST-free, documented limitations below.
#   CM-SCRIPT-TARGET-SHELL-PARSEABLE (§11.4.67) — every tracked *.sh under
#       scripts/ + scripts/testing/ parses cleanly under `bash -n`, AND any
#       script declaring an sh shebang (#!/bin/sh, #!/usr/bin/env sh) ALSO
#       parses under `sh -n`. A bash-shebang'd script is only required to be
#       bash-parseable (invoked via its shebang it never runs under sh) — see
#       §11.4.67's "may be invoked under a target shell OTHER than its shebang".
#   CM-VERSION-SINGLE-SOURCE     (P0.1 / a36030e) — no cmd/*/main.go declares
#       a hardcoded semver version literal (e.g. appVersion = "3.0.0"); every
#       binary's version derives from pkg/version.AppVersion (== authoritative
#       VERSION file). Fast grep complement of the Go test
#       TestNoBinaryDeclaresDivergentVersionLiteral.
#
# Usage:
#   scripts/pre_build_verification.sh            # run all gates
#   scripts/pre_build_verification.sh --list     # list gate ids + exit 0
#   scripts/pre_build_verification.sh --gate <CM-ID>   # run one gate
#
# Inputs (env):
#   PBV_REPO_ROOT   Override repo root (mutation tests point this at the tree
#                   they mutate; defaults to the git toplevel of this script).
#
# Outputs:
#   stdout  Per-gate PASS/FAIL lines + a final SUMMARY line.
#   Exit codes:
#     0  every selected gate PASSED
#     1  at least one selected gate FAILED (a real violation)
#     2  usage error / harness error (unknown gate, no repo)
#
# Side-effects: NONE. Read-only over the git index + working tree. No build,
#   no network, no writes. Fast (git ls-files + grep only).
#
# Dependencies: bash or sh, git, grep. (No pandoc/weasyprint/go required.)
#
# Cross-references:
#   docs/scripts/pre_build_verification.sh.md  — companion user guide (§11.4.18)
#   scripts/testing/meta_test_gitignore_precommit_audit.sh — paired mutation (§1.1)
#   scripts/testing/meta_test_no_fakes_beyond_unit.sh      — paired mutation (§1.1)
#   scripts/testing/meta_test_script_target_shell_parseable.sh — paired mutation (§1.1)
#   scripts/testing/meta_test_version_single_source.sh         — paired mutation (§1.1)
#   §11.4.67 target-shell-parseable — passes `bash -n` AND `sh -n`.
#
# Parseability note (§11.4.67): written in POSIX-portable shell. No arrays,
#   no [[ ]], no process substitution. Honest shebang: works under sh and bash.
# ============================================================================

set -u

# ---------------------------------------------------------------------------
# Repo root resolution (read-only).
# ---------------------------------------------------------------------------
_self_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
if [ -n "${PBV_REPO_ROOT:-}" ]; then
  ROOT="$PBV_REPO_ROOT"
else
  ROOT=$(git -C "$_self_dir" rev-parse --show-toplevel 2>/dev/null || true)
  [ -z "$ROOT" ] && ROOT=$(CDPATH= cd -- "$_self_dir/.." && pwd)
fi
if [ ! -d "$ROOT/.git" ] && ! git -C "$ROOT" rev-parse --git-dir >/dev/null 2>&1; then
  echo "pre_build_verification: ERROR — '$ROOT' is not a git repository" >&2
  exit 2
fi
cd "$ROOT" || { echo "pre_build_verification: ERROR — cannot cd to '$ROOT'" >&2; exit 2; }

# ---------------------------------------------------------------------------
# Gate: CM-GITIGNORE-PRECOMMIT-AUDIT (§11.4.30)
#
# Asserts NO tracked file matches a forbidden build-artifact / secret class.
# The forbidden set mirrors the classes this project's .gitignore declares as
# build-derivative or sensitive, PLUS the cmd/api-server anchoring regression:
#   - build/{api-server,grpc-server,unified-translator}   (declared build outputs)
#   - /api-server  (the root prebuilt binary; MUST stay anchored so the bare
#                   token never re-matches cmd/api-server/ SOURCE)
#   - real .env files (NOT .env.example, which is explicitly allow-listed)
#   - config_with_keys.json / api_keys.json / secrets.* / keys.*  (secrets)
#
# A tracked match is a §11.4.30 violation of equal severity to no ignore-line.
# ---------------------------------------------------------------------------
gate_gitignore_precommit_audit() {
  _violations=""

  # 1. Declared build outputs that must never be tracked.
  _build_hits=$(git ls-files -- \
      'build/api-server' 'build/grpc-server' 'build/unified-translator' 2>/dev/null)
  [ -n "$_build_hits" ] && _violations="$_violations
$_build_hits"

  # 2. Root prebuilt binary 'api-server' tracked at repo root (anchoring guard).
  #    A tracked top-level file literally named 'api-server' (no slash) means
  #    the build output leaked into VC AND the anchoring intent was lost.
  _root_bin=$(git ls-files -- 'api-server' 2>/dev/null | grep -E '^api-server$' || true)
  [ -n "$_root_bin" ] && _violations="$_violations
$_root_bin"

  # 3. Real .env files (allow only .env.example).
  _env_hits=$(git ls-files 2>/dev/null \
      | grep -E '(^|/)\.env(\.[^/]+)?$' \
      | grep -vE '(^|/)\.env\.example$' || true)
  [ -n "$_env_hits" ] && _violations="$_violations
$_env_hits"

  # 4. Secret-bearing config files.
  _secret_hits=$(git ls-files 2>/dev/null \
      | grep -E '(^|/)(config_with_keys\.json|api_keys\.json|secrets\.[^/]+|keys\.[^/]+)$' || true)
  [ -n "$_secret_hits" ] && _violations="$_violations
$_secret_hits"

  # Trim leading blank line.
  _violations=$(printf '%s\n' "$_violations" | sed '/^[[:space:]]*$/d')

  if [ -n "$_violations" ]; then
    echo "FAIL CM-GITIGNORE-PRECOMMIT-AUDIT — forbidden tracked file(s) (§11.4.30):"
    printf '%s\n' "$_violations" | sed 's/^/         - /'
    return 1
  fi
  echo "PASS CM-GITIGNORE-PRECOMMIT-AUDIT — no forbidden build-artifact/secret tracked"
  return 0
}

# ---------------------------------------------------------------------------
# Gate: CM-NO-FAKES-BEYOND-UNIT (§11.4.27)
#
# Asserts non-unit Go test files do not import mock/stub package paths.
# Mocks/stubs/fakes are permitted ONLY in unit tests. A "non-unit" test file
# is identified by a Go build tag in its first lines:
#     //go:build integration   (or e2e | stress | performance | security)
# If such a file imports a path ending in /mock, /mocks, /stub, /stubs, /fake
# or /fakes, that is a §11.4.27(A) violation (the test claims to exercise the
# real system but pulls in a fake).
#
# DOCUMENTED LIMITATIONS (honest boundary, §11.4.6):
#   - Detection is line-grep based, not AST-based: it matches import-path
#     string literals, so a mock referenced indirectly (var of a mock type
#     re-exported from a non-mock-named package) is NOT caught.
#   - Only the listed build tags mark a file "non-unit". Untagged _test.go
#     files are treated as unit tests (mocks allowed) — matching the project
#     convention that integration/e2e suites carry build tags.
#   - This gate is a high-value cheap probe, NOT a complete proof; AST-grade
#     enforcement is tracked as future work.
# ---------------------------------------------------------------------------
gate_no_fakes_beyond_unit() {
  _bad=""
  # Enumerate tracked Go test files.
  for _f in $(git ls-files '*_test.go' 2>/dev/null); do
    [ -f "$_f" ] || continue
    # Is this a non-unit test? Look for a build tag in the header (first 15 lines).
    if head -n 15 "$_f" 2>/dev/null \
         | grep -Eq '^//go:build .*(integration|e2e|stress|performance|security)'; then
      # It is non-unit. Does it import a fake/mock/stub path?
      if grep -Eq '"[^"]*/(mocks?|stubs?|fakes?)"' "$_f" 2>/dev/null; then
        _bad="$_bad
$_f"
      fi
    fi
  done
  _bad=$(printf '%s\n' "$_bad" | sed '/^[[:space:]]*$/d')

  if [ -n "$_bad" ]; then
    echo "FAIL CM-NO-FAKES-BEYOND-UNIT — non-unit test imports a mock/stub (§11.4.27):"
    printf '%s\n' "$_bad" | sed 's/^/         - /'
    return 1
  fi
  echo "PASS CM-NO-FAKES-BEYOND-UNIT — no non-unit test imports a fake/mock/stub path"
  return 0
}

# ---------------------------------------------------------------------------
# Gate: CM-SCRIPT-TARGET-SHELL-PARSEABLE (§11.4.67)
#
# Asserts every tracked *.sh under scripts/ + scripts/testing/ parses cleanly.
# Rule (faithful to §11.4.67 "every shell script that may be invoked under a
# target shell OTHER than the one in its shebang MUST parse cleanly under that
# target shell"):
#   - EVERY script MUST pass `bash -n` (bash is the project's superset shell;
#     a script that is not even valid bash is broken outright).
#   - A script whose shebang declares an sh-family interpreter (#!/bin/sh,
#     #!/usr/bin/env sh, plain `sh`) MUST ALSO pass `sh -n` — because POSIX sh
#     parses the WHOLE script before executing (the mksh-on-Android lesson in
#     §11.4.67), so a bash-only construct anywhere is a latent crash.
#   - A script whose shebang declares bash is NOT required to pass `sh -n`:
#     invoked via its shebang it only ever runs under bash, so bash-only
#     constructs (mapfile, < <(...), [[ ]], arrays) are legitimate. Forcing
#     `sh -n` on an honest-bash script would itself be a §11.4.1 FAIL-bluff.
#
# A script that fails its REQUIRED parse is a §11.4.67 violation (a real,
# user-invocable broken script — fix at source per §11.4.1, never the gate).
#
# Requires `bash` AND `sh` on PATH; if either is missing the gate cannot make
# its assertion and returns 2 (harness error), never a false PASS (§11.4.1).
# ---------------------------------------------------------------------------
gate_script_target_shell_parseable() {
  command -v bash >/dev/null 2>&1 || {
    echo "FAIL CM-SCRIPT-TARGET-SHELL-PARSEABLE — 'bash' not on PATH (cannot assert)"; return 2; }
  command -v sh >/dev/null 2>&1 || {
    echo "FAIL CM-SCRIPT-TARGET-SHELL-PARSEABLE — 'sh' not on PATH (cannot assert)"; return 2; }

  _bad=""
  for _f in $(git ls-files 'scripts/*.sh' 'scripts/*/*.sh' 'scripts/*/*/*.sh' 2>/dev/null); do
    [ -f "$_f" ] || continue
    # Every script must be valid bash.
    if ! bash -n "$_f" 2>/dev/null; then
      _bad="$_bad
$_f (bash -n FAILED — not valid bash)"
      continue
    fi
    # Determine declared shell from the shebang's first line.
    _shebang=$(head -n 1 "$_f" 2>/dev/null)
    case "$_shebang" in
      *bash*)
        : # bash-declared: bash -n already passed; sh -n NOT required.
        ;;
      \#!*sh*)
        # sh-family shebang (covers #!/bin/sh, #!/usr/bin/env sh, #!/usr/bin/ksh
        # etc.) — must ALSO parse under sh.
        if ! sh -n "$_f" 2>/dev/null; then
          _bad="$_bad
$_f (sh shebang but sh -n FAILED — bash-only construct in an sh script)"
        fi
        ;;
      *)
        # No recognised shell shebang: treat conservatively (§11.4.6) — require
        # sh -n too, since such a script could be invoked via `sh script.sh`.
        if ! sh -n "$_f" 2>/dev/null; then
          _bad="$_bad
$_f (no shell shebang and sh -n FAILED — could be run under sh)"
        fi
        ;;
    esac
  done
  _bad=$(printf '%s\n' "$_bad" | sed '/^[[:space:]]*$/d')

  if [ -n "$_bad" ]; then
    echo "FAIL CM-SCRIPT-TARGET-SHELL-PARSEABLE — script(s) fail required parse (§11.4.67):"
    printf '%s\n' "$_bad" | sed 's/^/         - /'
    return 1
  fi
  echo "PASS CM-SCRIPT-TARGET-SHELL-PARSEABLE — every tracked scripts/*.sh parses under its required shell"
  return 0
}

# ---------------------------------------------------------------------------
# Gate: CM-VERSION-SINGLE-SOURCE (P0.1 / a36030e)
#
# Asserts NO cmd/*/main.go declares a hardcoded semver version literal. Every
# binary MUST source its version from pkg/version.AppVersion (== authoritative
# VERSION file) so all binaries agree. The forbidden pattern is a Go assignment
# of a 3-part semver string literal to an identifier named (app)version:
#     appVersion = "3.0.0"   |   const version = "2.0.0"
# A reference like `appVersion = version.AppVersion` is COMPLIANT (no string
# literal). XML/EPUB attrs like version="1.0" are NOT matched (no Go ` = `
# assignment spacing + only 2-part). Mirrors the Go test
# TestNoBinaryDeclaresDivergentVersionLiteral as a fast pre-build grep.
#
# Honest boundary (§11.4.6): grep-based, so a version literal assembled at
# runtime (fmt.Sprintf) or held in a non-(app)version identifier is not caught
# — the Go test + this gate together are the high-value cheap probe, not an
# AST-grade proof.
# ---------------------------------------------------------------------------
gate_version_single_source() {
  _offenders=""
  for _f in $(git ls-files 'cmd/*/main.go' 2>/dev/null); do
    [ -f "$_f" ] || continue
    _hits=$(grep -nE '\b(app[Vv]ersion|version)[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"' "$_f" 2>/dev/null || true)
    if [ -n "$_hits" ]; then
      _offenders="$_offenders
$_f:
$(printf '%s\n' "$_hits" | sed 's/^/    /')"
    fi
  done
  _offenders=$(printf '%s\n' "$_offenders" | sed '/^[[:space:]]*$/d')

  if [ -n "$_offenders" ]; then
    echo "FAIL CM-VERSION-SINGLE-SOURCE — cmd binary hardcodes a version literal (P0.1):"
    printf '%s\n' "$_offenders" | sed 's/^/         /'
    return 1
  fi
  echo "PASS CM-VERSION-SINGLE-SOURCE — no cmd/*/main.go hardcodes a version literal (all derive from version.AppVersion)"
  return 0
}

# ---------------------------------------------------------------------------
# Dispatch.
# ---------------------------------------------------------------------------
run_one() {
  case "$1" in
    CM-GITIGNORE-PRECOMMIT-AUDIT)     gate_gitignore_precommit_audit ;;
    CM-NO-FAKES-BEYOND-UNIT)          gate_no_fakes_beyond_unit ;;
    CM-SCRIPT-TARGET-SHELL-PARSEABLE) gate_script_target_shell_parseable ;;
    CM-VERSION-SINGLE-SOURCE)         gate_version_single_source ;;
    *) echo "pre_build_verification: ERROR — unknown gate '$1'" >&2; return 2 ;;
  esac
}

GATES="CM-GITIGNORE-PRECOMMIT-AUDIT CM-NO-FAKES-BEYOND-UNIT CM-SCRIPT-TARGET-SHELL-PARSEABLE CM-VERSION-SINGLE-SOURCE"

if [ "${1:-}" = "--list" ]; then
  for g in $GATES; do echo "$g"; done
  exit 0
fi

if [ "${1:-}" = "--gate" ]; then
  [ -n "${2:-}" ] || { echo "pre_build_verification: --gate needs an id" >&2; exit 2; }
  run_one "$2"
  exit $?
fi

echo "=== pre_build_verification — helix_translate pre-build gate suite ==="
echo "repo: $ROOT"
echo
_rc=0
for g in $GATES; do
  run_one "$g" || _rc=1
done
echo
if [ "$_rc" -eq 0 ]; then
  echo "SUMMARY: PASS — all $(printf '%s\n' $GATES | wc -l | tr -d ' ') gates green"
else
  echo "SUMMARY: FAIL — at least one gate flagged a real violation"
fi
exit "$_rc"
