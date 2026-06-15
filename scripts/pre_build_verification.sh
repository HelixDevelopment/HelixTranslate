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
#   CM-TRACKER-DOCS-PRESENT      (§11.4.15/.16/.53) — the four canonical
#       workable-item tracker docs (docs/Issues.md, docs/Fixed.md,
#       docs/Issues_Summary.md, docs/Fixed_Summary.md) all exist AND each
#       carries a §11.4.44 revision header (a '**Revision:**' line AND a
#       '**Last modified:**' line). A missing doc or a missing header is a
#       documentation-layer §11.4 violation (the tracker constellation is
#       incomplete / unversioned).
#   CM-ATM-TICKET-IDS            (§11.4.54) — every '###/## §' workable-item
#       heading in docs/Issues.md + docs/Fixed.md carries an '[ATM-NNN]'
#       ticket token; the ATM ids are UNIQUE across both files AND MONOTONIC
#       with no gaps (the contiguous sequence ATM-001..ATM-NNN). A heading
#       without an id, a duplicate id, or a gap in the sequence is a §11.4.54
#       violation.
#   CM-DOC-SIBLING-SYNC          (§11.4.65) — every in-scope tracked *.md
#       (project-root *.md, docs/**, scripts/** companions; EXCLUDING owned
#       submodule trees + build/vendor/qa dirs) has BOTH a tracked .html AND a
#       tracked .pdf sibling, and each sibling's mtime is >= the .md mtime. A
#       missing or stale (older) sibling is a §11.4.65 universal-Markdown-export
#       violation (operators/agents reading the HTML/PDF get a divergent view).
#   CM-NO-FORCE-PUSH-ABSOLUTE    (§11.4.113) — no tracked script under scripts/
#       contains an ACTUAL force-push invocation: a `git push` carrying
#       --force / --force-with-lease / -f, OR a `git push` with a leading-'+'
#       forced refspec. Force-push is STRICTLY FORBIDDEN with no exception.
#       Comment lines, case-pattern arms, and die/echo refusal strings (the
#       commit_all.sh §11.4.113 GUARD) are NOT invocations and do NOT trip it.
#   CM-NO-LOCAL-RUNTIME          (§11.4.69 / bridge phase-2 R-5) — the default
#       translator provisioning path sources ONLY the LLMsVerifier bridge; no
#       local-runtime (llama.cpp / Ollama) client is constructed on it. Three
#       arms over the redirect-DEFAULT construction sites (cmd/unified-translator,
#       cmd/cli, cmd/server, cmd/markdown-translator, cmd/preparation-translator,
#       pkg/api/handler.go, pkg/grpc/core_translator.go):
#         Arm 1 — no default-path file constructs a local-runtime client
#                 (NewLlamaCppClient / NewOllamaClient / NewLlamaCppProvider, or
#                 a ProviderLlamaCpp / ProviderOllama provider-string construction).
#         Arm 2 (primary/durable) — each present default-path file references the
#                 bridge ('bridge.' or 'bridgeTranslator(') so the redirect is real.
#         Arm 3 — pkg/bridge/bridge.go still carries the no-fail-open prohibition
#                 literal 'local llama.cpp fallback is not permitted' (§11.4.69).
#       DOCUMENTED EXCEPTIONS (never flagged): the retained proto wire-fields +
#       cmd/api-server proto use, config.distributed.* / config.worker.json,
#       comments / flag-name / help mentions, pkg/translator/llm/mock.go, *_test.go.
#       No fail-open / SKIP (§11.4.69).
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
#   scripts/testing/meta_test_tracker_docs_present.sh          — paired mutation (§1.1)
#   scripts/testing/meta_test_atm_ticket_ids.sh               — paired mutation (§1.1)
#   scripts/testing/meta_test_doc_sibling_sync.sh             — paired mutation (§1.1)
#   scripts/testing/meta_test_no_force_push_absolute.sh       — paired mutation (§1.1)
#   scripts/testing/meta_test_no_local_runtime.sh             — paired mutation (§1.1)
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
# Gate: CM-TRACKER-DOCS-PRESENT (§11.4.15/.16/.53)
#
# Asserts the four canonical workable-item tracker docs exist AND each carries
# a §11.4.44 revision header. The required docs:
#   docs/Issues.md          (open tracker — §11.4.15/.16)
#   docs/Fixed.md           (closed archive — §11.4.19/.33)
#   docs/Issues_Summary.md  (open summary — §11.4.12)
#   docs/Fixed_Summary.md   (closed summary — §11.4.53)
# A §11.4.44 revision header here means BOTH a line beginning '**Revision:**'
# AND a line beginning '**Last modified:**' appear in the doc's header block
# (checked over the first 12 non-blank lines, where the header lives).
#
# A missing doc OR a missing header line is a real, user-visible violation:
# the tracker constellation is incomplete (an operator/agent cannot resume
# from a half-present tracker) or unversioned (§11.4.44 freshness breaks).
# ---------------------------------------------------------------------------
gate_tracker_docs_present() {
  _bad=""
  for _doc in docs/Issues.md docs/Fixed.md docs/Issues_Summary.md docs/Fixed_Summary.md; do
    if [ ! -f "$_doc" ]; then
      _bad="$_bad
$_doc (MISSING)"
      continue
    fi
    # Read the header block (first 12 lines) and check for both header lines.
    _hdr=$(head -n 12 "$_doc" 2>/dev/null)
    if ! printf '%s\n' "$_hdr" | grep -q '^\*\*Revision:\*\*'; then
      _bad="$_bad
$_doc (no §11.4.44 '**Revision:**' header line)"
    fi
    if ! printf '%s\n' "$_hdr" | grep -q '^\*\*Last modified:\*\*'; then
      _bad="$_bad
$_doc (no §11.4.44 '**Last modified:**' header line)"
    fi
  done
  _bad=$(printf '%s\n' "$_bad" | sed '/^[[:space:]]*$/d')

  if [ -n "$_bad" ]; then
    echo "FAIL CM-TRACKER-DOCS-PRESENT — tracker doc missing or unversioned (§11.4.15/.16/.53/.44):"
    printf '%s\n' "$_bad" | sed 's/^/         - /'
    return 1
  fi
  echo "PASS CM-TRACKER-DOCS-PRESENT — all 4 tracker docs present with §11.4.44 revision headers"
  return 0
}

# ---------------------------------------------------------------------------
# Gate: CM-ATM-TICKET-IDS (§11.4.54)
#
# Asserts every '###/## §' workable-item heading in docs/Issues.md +
# docs/Fixed.md carries an '[ATM-NNN]' token, AND the union of all ids is
# UNIQUE and MONOTONIC with no gaps — the contiguous sequence ATM-001..ATM-N
# where N is the count of ids (so min=1, max=count, every integer present once).
#
# Three failure modes, each a real §11.4.54 violation:
#   (1) a workable-item heading missing its [ATM-NNN] token (unidentifiable item)
#   (2) a duplicate id across the two files (the id is not a stable unique key)
#   (3) a gap in the sequence (renumber/skip — §11.4.54 forbids gaps)
#
# Honest boundary (§11.4.6): "workable-item heading" is matched as a line of
# the form '^#{2,3} §' (the project's §X. heading convention). Headings not
# matching that shape (e.g. the doc H1, '---' rules) are not items and are not
# required to carry an id. This is a fast grep probe, faithful to the tree's
# actual heading convention (verified: Issues §1..17, Fixed §1..64).
# ---------------------------------------------------------------------------
gate_atm_ticket_ids() {
  # 1. Every workable-item heading must carry an [ATM-NNN] token.
  _missing=""
  for _doc in docs/Issues.md docs/Fixed.md; do
    [ -f "$_doc" ] || { _missing="$_missing
$_doc (MISSING file)"; continue; }
    # Headings of the §X. workable-item form that lack an [ATM-NNN] token.
    _hits=$(grep -nE '^#{2,3} §' "$_doc" 2>/dev/null | grep -vE '\[ATM-[0-9]+\]' || true)
    if [ -n "$_hits" ]; then
      _missing="$_missing
$_doc:
$(printf '%s\n' "$_hits" | sed 's/^/    /')"
    fi
  done
  _missing=$(printf '%s\n' "$_missing" | sed '/^[[:space:]]*$/d')
  if [ -n "$_missing" ]; then
    echo "FAIL CM-ATM-TICKET-IDS — workable-item heading without an [ATM-NNN] token (§11.4.54):"
    printf '%s\n' "$_missing" | sed 's/^/         /'
    return 1
  fi

  # 2/3. Collect all ids (as integers) from both files; check unique + monotonic.
  _ids=$(grep -ohE '\[ATM-[0-9]+\]' docs/Issues.md docs/Fixed.md 2>/dev/null \
           | grep -oE '[0-9]+' | sed 's/^0*//; s/^$/0/')
  if [ -z "$_ids" ]; then
    echo "FAIL CM-ATM-TICKET-IDS — no [ATM-NNN] ids found in trackers (§11.4.54)"
    return 1
  fi

  _sorted=$(printf '%s\n' "$_ids" | sort -n)
  _count=$(printf '%s\n' "$_sorted" | wc -l | tr -d ' ')

  # Duplicate detection: distinct count must equal total count.
  _distinct=$(printf '%s\n' "$_sorted" | uniq | wc -l | tr -d ' ')
  if [ "$_distinct" != "$_count" ]; then
    _dups=$(printf '%s\n' "$_sorted" | uniq -d | sed 's/^/ATM-/')
    echo "FAIL CM-ATM-TICKET-IDS — duplicate ATM id(s) across trackers (§11.4.54):"
    printf '%s\n' "$_dups" | sed 's/^/         - /'
    return 1
  fi

  # Monotonic no-gap: min must be 1 and max must equal count (contiguous 1..N).
  _min=$(printf '%s\n' "$_sorted" | head -n 1)
  _max=$(printf '%s\n' "$_sorted" | tail -n 1)
  if [ "$_min" != "1" ] || [ "$_max" != "$_count" ]; then
    echo "FAIL CM-ATM-TICKET-IDS — ATM sequence not contiguous 001..NNN (§11.4.54):"
    echo "         - have $_count distinct id(s), min=ATM-$(printf '%03d' "$_min"), max=ATM-$(printf '%03d' "$_max")"
    echo "         - expected min=ATM-001 and max=ATM-$(printf '%03d' "$_count") (a gap or out-of-range id exists)"
    return 1
  fi

  echo "PASS CM-ATM-TICKET-IDS — every item carries a unique, contiguous [ATM-001..ATM-$(printf '%03d' "$_count")] id"
  return 0
}

# ---------------------------------------------------------------------------
# Gate: CM-DOC-SIBLING-SYNC (§11.4.65)
#
# Asserts every in-scope tracked *.md has BOTH a tracked .html AND a tracked
# .pdf sibling whose mtime is >= the .md's mtime (siblings never older than
# their source). In-scope per §11.4.65: project-root *.md, docs/**, scripts/**
# companions — EXCLUDING owned-submodule trees (challenges, containers,
# helix_qa, doc_processor, llm_orchestrator, llm_provider, vision_engine,
# llms_verifier, constitution, docs_chain — they own their own exports) and
# build/vendor/qa dirs (build, out, dist, external, prebuilts, node_modules,
# vendor, qa-results). The exclusion set mirrors
# scripts/testing/sync_all_markdown_exports.sh (the generator that keeps
# siblings fresh) so the gate and the generator agree on scope.
#
# Three failure modes, each a real §11.4.65 violation:
#   (1) an in-scope .md with NO tracked .html sibling (unexported)
#   (2) an in-scope .md with NO tracked .pdf sibling (unexported)
#   (3) a sibling whose mtime is OLDER than its .md (stale — divergent view)
#
# Operates over TRACKED files only (git ls-files): an untracked stray .md is
# not a versioned doc and an untracked sibling does not satisfy the export
# mandate (the export must be committed). Honest boundary (§11.4.6): mtime is
# a working-tree property; a fresh clone checks out arbitrary mtimes, so the
# mtime arm asserts "in THIS working tree the sibling is not older" — the
# presence arm is the durable cross-checkout invariant, the mtime arm is the
# local freshness guard the §11.4.65 generator enforces on every sync.
# ---------------------------------------------------------------------------
gate_doc_sibling_sync() {
  # Basename/path exclusion regex — mirrors sync_all_markdown_exports.sh.
  _excl='(^|/)(challenges|containers|helix_qa|doc_processor|llm_orchestrator|llm_provider|vision_engine|llms_verifier|constitution|docs_chain|node_modules|vendor|external|prebuilts|build|out|dist|qa-results)/'

  # In-scope tracked .md: root-level *.md OR under docs/ OR under scripts/,
  # minus the excluded trees.
  # Per-file exclusion: raw DATA sources that are NOT published documents and so
  # do NOT get .html/.pdf exports (§11.4.65 governs documents, not data sources).
  # docs/features/.feature_inventory_raw.md is a dotfile-prefixed raw inventory
  # DATA source (self-described "DATA-GATHERING raw material to seed Status.md",
  # per §11.4.153) — Status.md / Status_Summary.md ARE the published feature docs
  # and DO get exports; the raw dotfile feeding them does not. MUST stay in sync
  # with the identical exclusion in
  # scripts/testing/sync_all_markdown_exports.sh (gate and generator agree on
  # scope, §11.4.65).
  _excl_files='^docs/features/\.feature_inventory_raw\.md$'

  _mds=$(git ls-files -- '*.md' 2>/dev/null \
           | grep -Ev "$_excl" \
           | grep -Ev "$_excl_files" \
           | grep -E '^[^/]+\.md$|^docs/|^scripts/' || true)

  _bad=""
  for _md in $_mds; do
    [ -f "$_md" ] || continue
    _base=${_md%.md}
    for _ext in html pdf; do
      _sib="$_base.$_ext"
      # Sibling must be TRACKED (committed export).
      if ! git ls-files --error-unmatch -- "$_sib" >/dev/null 2>&1; then
        _bad="$_bad
$_sib (MISSING tracked sibling for $_md)"
        continue
      fi
      # And present on disk and not older than the .md.
      if [ ! -f "$_sib" ]; then
        _bad="$_bad
$_sib (tracked but absent from working tree)"
      elif [ "$_md" -nt "$_sib" ]; then
        _bad="$_bad
$_sib (STALE — older than $_md)"
      fi
    done
  done
  _bad=$(printf '%s\n' "$_bad" | sed '/^[[:space:]]*$/d')

  if [ -n "$_bad" ]; then
    echo "FAIL CM-DOC-SIBLING-SYNC — in-scope .md missing or stale .html/.pdf sibling (§11.4.65):"
    printf '%s\n' "$_bad" | sed 's/^/         - /'
    return 1
  fi
  echo "PASS CM-DOC-SIBLING-SYNC — every in-scope tracked .md has fresh .html + .pdf siblings"
  return 0
}

# ---------------------------------------------------------------------------
# Gate: CM-NO-FORCE-PUSH-ABSOLUTE (§11.4.113)
#
# Asserts NO tracked script under scripts/ contains an ACTUAL force-push
# invocation. §11.4.113 forbids force-push with NO exception. The forbidden
# invocation forms:
#   - `git push ... --force`            (incl. --force=...)
#   - `git push ... --force-with-lease` (incl. --force-with-lease=...)
#   - `git push ... -f`                 (the short force flag, as a token)
#   - `git push ... +<ref>`             (a leading-'+' forced refspec)
#
# CRITICAL false-positive avoidance (§11.4.1 — a FAIL-bluff is forbidden): the
# project's OWN commit_all.sh GUARD against force-push legitimately names these
# tokens in (a) comment lines, (b) `case` pattern arms ('--force|...)' ), and
# (c) die/echo refusal strings. None of those is an invocation. The gate
# therefore scans ONLY lines that look like a `git push` COMMAND and carry a
# force token on the SAME logical command, while excluding:
#   - comment lines (first non-blank char is '#')
#   - case-pattern arms (a bare token-list ending in ')')
#   - lines whose force token appears only inside a quoted die/echo message
#     (i.e. lines that contain 'die ' or 'echo ' before the token, OR a
#      §11.4.113 reference) — these are the GUARD, not a push.
#
# A genuine `git push --force` (or +refspec) in any tracked scripts/*.sh is a
# §11.4.113 violation; a §11.4.109-class PreToolUse guard blocks the class at
# the tool-call boundary, this gate is the committed-tree complement.
# ---------------------------------------------------------------------------
gate_no_force_push_absolute() {
  _bad=""
  for _f in $(git ls-files 'scripts/*.sh' 'scripts/*/*.sh' 'scripts/*/*/*.sh' 2>/dev/null); do
    [ -f "$_f" ] || continue
    # Candidate lines: contain 'git push' AND a force token — either a force
    # FLAG (--force / --force-with-lease / -f as a word) anywhere on the line,
    # OR a '+'-prefixed forced refspec token (e.g. '+main:main') which may sit
    # after the remote ('git push origin +main:main'), so it is matched as a
    # whitespace-led '+<word>' token, NOT only immediately after 'push'.
    _hits=$(grep -nE 'git[[:space:]]+push' "$_f" 2>/dev/null \
              | grep -E '(--force|--force-with-lease|[[:space:]]-f([[:space:]]|$)|[[:space:]]\+[A-Za-z][A-Za-z0-9_./-]*(:|[[:space:]]|$))' \
              || true)
    [ -n "$_hits" ] || continue
    # Filter out non-invocation lines (comments / case arms / die-echo guards).
    while IFS= read -r _line; do
      [ -n "$_line" ] || continue
      # Strip the leading 'N:' line-number prefix grep -n adds.
      _body=${_line#*:}
      # Trim leading whitespace.
      _trim=$(printf '%s' "$_body" | sed 's/^[[:space:]]*//')
      # (a) comment line.
      case "$_trim" in '#'*) continue ;; esac
      # (b) the GUARD: a die/echo refusal string or a §11.4.113 reference.
      case "$_body" in
        *die\ *|*echo\ *|*11.4.113*) continue ;;
      esac
      # (c) case-pattern arm: a token-list ending in ')' with no command verb.
      #     e.g. '--force|--force-with-lease|-f)'  — these are option matchers.
      case "$_trim" in
        *')') case "$_trim" in *git\ push*) : ;; *) continue ;; esac ;;
      esac
      # Survivor: a real-looking `git push` carrying a force token.
      _bad="$_bad
$_f: $_body"
    done <<EOF2
$_hits
EOF2
  done
  _bad=$(printf '%s\n' "$_bad" | sed '/^[[:space:]]*$/d')

  if [ -n "$_bad" ]; then
    echo "FAIL CM-NO-FORCE-PUSH-ABSOLUTE — force-push invocation in tracked script (§11.4.113):"
    printf '%s\n' "$_bad" | sed 's/^/         - /'
    return 1
  fi
  echo "PASS CM-NO-FORCE-PUSH-ABSOLUTE — no tracked scripts/*.sh invokes a force-push"
  return 0
}

# ---------------------------------------------------------------------------
# Gate: CM-NO-LOCAL-RUNTIME (§11.4.69 / bridge phase-2 R-5)
#
# OPTION A (default-path-only): assert the default translator provisioning path
# sources ONLY the LLMsVerifier bridge, never a local runtime (llama.cpp/Ollama).
#
# The redirect-DEFAULT construction sites (each routes the default arm through
# bridge.BestTranslator / bridge.BestClient / bridgeTranslator):
_NLR_DEFAULT_PATH_FILES="cmd/unified-translator/main.go cmd/cli/main.go cmd/server/main.go cmd/markdown-translator/main.go cmd/preparation-translator/main.go pkg/api/handler.go pkg/grpc/core_translator.go"
_NLR_BRIDGE_FILE="pkg/bridge/bridge.go"
# Local-runtime CONSTRUCTOR tokens forbidden on the default path (Arm 1). These
# are construction sites, not flag-name / help / comment mentions: NewLlamaCppClient,
# NewOllamaClient, NewLlamaCppProvider (factory constructors); ProviderLlamaCpp(/
# ProviderOllama( as a function/conversion-style construction. Bare ProviderOllama
# without a trailing '(' would be a const reference; the removal already deleted
# the consts, so any reappearance constructs/uses a local provider — flagged.
_NLR_CTOR_RE='NewLlamaCppClient|NewOllamaClient|NewLlamaCppProvider|ProviderLlamaCpp|ProviderOllama|OllamaClient\{'
# Arm 2 durable proof: the default arm references the bridge.
_NLR_BRIDGE_RE='bridge\.|bridgeTranslator\('
# Arm 3 no-fail-open prohibition literal that MUST remain in pkg/bridge/bridge.go.
_NLR_PROHIBITION='local llama.cpp fallback is not permitted'

gate_no_local_runtime() {
  _arm1=""   # default-path files that construct a local-runtime client
  _arm2=""   # present default-path files that do NOT reference the bridge
  _present=0

  for _f in $_NLR_DEFAULT_PATH_FILES; do
    # Only check files actually tracked + present (a tmp meta-test repo may
    # contain only a subset; an absent canonical file is caught by the build).
    [ -f "$_f" ] || continue
    git ls-files --error-unmatch "$_f" >/dev/null 2>&1 || continue
    _present=$((_present + 1))

    # Arm 1 — local-runtime constructor on the default path. Strip // comments
    # and // doc lines so a comment naming a removed provider never trips it.
    _ctor=$(grep -nE "$_NLR_CTOR_RE" "$_f" 2>/dev/null \
              | grep -vE '^[0-9]+:[[:space:]]*//' || true)
    if [ -n "$_ctor" ]; then
      _arm1="$_arm1
$_f:
$(printf '%s\n' "$_ctor" | sed 's/^/    /')"
    fi

    # Arm 2 — file must reference the bridge somewhere (durable redirect proof).
    if ! grep -qE "$_NLR_BRIDGE_RE" "$_f" 2>/dev/null; then
      _arm2="$_arm2
$_f (no '$_NLR_BRIDGE_RE' reference — default path does not route through the bridge)"
    fi
  done

  # Arm 3 — the no-fail-open prohibition literal must remain in bridge.go.
  _arm3=""
  if [ -f "$_NLR_BRIDGE_FILE" ] && git ls-files --error-unmatch "$_NLR_BRIDGE_FILE" >/dev/null 2>&1; then
    if ! grep -qF "$_NLR_PROHIBITION" "$_NLR_BRIDGE_FILE" 2>/dev/null; then
      _arm3="$_NLR_BRIDGE_FILE (missing no-fail-open literal: '$_NLR_PROHIBITION')"
    fi
  fi

  _arm1=$(printf '%s\n' "$_arm1" | sed '/^[[:space:]]*$/d')
  _arm2=$(printf '%s\n' "$_arm2" | sed '/^[[:space:]]*$/d')

  if [ -n "$_arm1" ] || [ -n "$_arm2" ] || [ -n "$_arm3" ]; then
    echo "FAIL CM-NO-LOCAL-RUNTIME — default translator path is not bridge-only (§11.4.69):"
    [ -n "$_arm1" ] && { echo "         Arm1 — local-runtime constructor on the default path:"; printf '%s\n' "$_arm1" | sed 's/^/           /'; }
    [ -n "$_arm2" ] && { echo "         Arm2 — default-path file does not reference the bridge:"; printf '%s\n' "$_arm2" | sed 's/^/           - /'; }
    [ -n "$_arm3" ] && { echo "         Arm3 — bridge no-fail-open prohibition removed:"; printf '%s\n' "         - $_arm3"; }
    return 1
  fi
  echo "PASS CM-NO-LOCAL-RUNTIME — default path bridge-only across $_present site(s); no local-runtime constructor; bridge prohibition intact"
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
    CM-TRACKER-DOCS-PRESENT)          gate_tracker_docs_present ;;
    CM-ATM-TICKET-IDS)                gate_atm_ticket_ids ;;
    CM-DOC-SIBLING-SYNC)              gate_doc_sibling_sync ;;
    CM-NO-FORCE-PUSH-ABSOLUTE)        gate_no_force_push_absolute ;;
    CM-NO-LOCAL-RUNTIME)              gate_no_local_runtime ;;
    *) echo "pre_build_verification: ERROR — unknown gate '$1'" >&2; return 2 ;;
  esac
}

GATES="CM-GITIGNORE-PRECOMMIT-AUDIT CM-NO-FAKES-BEYOND-UNIT CM-SCRIPT-TARGET-SHELL-PARSEABLE CM-VERSION-SINGLE-SOURCE CM-TRACKER-DOCS-PRESENT CM-ATM-TICKET-IDS CM-DOC-SIBLING-SYNC CM-NO-FORCE-PUSH-ABSOLUTE CM-NO-LOCAL-RUNTIME"

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
