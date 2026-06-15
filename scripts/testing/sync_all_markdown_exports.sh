#!/usr/bin/env bash
# scripts/testing/sync_all_markdown_exports.sh
#
# Purpose:   Generate/refresh synchronized .html + .pdf siblings for every
#            in-scope Markdown document in the project (§11.4.65 universal
#            Markdown export mandate). HTML via pandoc, PDF via weasyprint from
#            the generated HTML. Idempotent: a sibling is (re)generated only when
#            it is missing or older than its source .md. Honest about missing
#            tools (§11.4.106) — never writes a fake/empty/placeholder sibling.
# Usage:     scripts/testing/sync_all_markdown_exports.sh [--check] [--force]
#              (no args) generate missing/stale siblings, print a summary.
#              --check   do NOT generate; exit 1 if any in-scope .md is missing a
#                        sibling or has a stale (older) sibling. For gates/CI.
#              --force   regenerate every in-scope sibling regardless of mtime.
# Inputs:    In-scope .md per §11.4.65 — project-root *.md, docs/**/*.md,
#            scripts/**/*.md companions.
# Outputs:   <name>.html + <name>.pdf next to each in-scope <name>.md.
# Side-effects: writes only .html/.pdf siblings; never edits any .md.
# Dependencies: pandoc (HTML), weasyprint (PDF), coreutils. No network.
#            Parses clean under `sh -n` AND `bash -n` (§11.4.67).
# Cross-references: doc docs/scripts/sync_all_markdown_exports.md;
#            §11.4.65 (export mandate), §11.4.18 (script docs), §11.4.106
#            (no fake transform), §11.4.67 (target-shell-parseable).

set -eu

SELF_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/../.." && pwd)"
cd "$ROOT"

MODE="generate"
FORCE=0
for arg in "$@"; do
    case "$arg" in
        --check) MODE="check" ;;
        --force) FORCE=1 ;;
        -h|--help) sed -n '2,30p' "$0"; exit 0 ;;
        *) echo "ERROR: unknown argument: $arg" >&2; exit 2 ;;
    esac
done

# §11.4.106: honest tool availability — never fabricate a sibling.
HAVE_PANDOC=0
HAVE_WEASYPRINT=0
command -v pandoc >/dev/null 2>&1 && HAVE_PANDOC=1
command -v weasyprint >/dev/null 2>&1 && HAVE_WEASYPRINT=1

# Third-party / build / vendored / gitignored trees excluded per §11.4.65.
# These are anchored at the project root.
EXCLUDE_PATHS="
./.git
./external
./prebuilts
./build
./out
./dist
./node_modules
./vendor
./qa-results
"

# Owned submodules own their own exports — exclude them by BASENAME at ANY depth,
# so a vendored/nested copy (e.g. a docs/research/helix_qa/ workspace dump) is
# pruned too, not only the top-level submodule (§11.4.65 + project scope).
EXCLUDE_NAMES="
.git
node_modules
vendor
qa-results
challenges
containers
helix_qa
doc_processor
llm_orchestrator
llm_provider
vision_engine
llms_verifier
constitution
docs_chain
"

# Build the find -prune expression from both exclusion sets.
prune_args=""
for d in $EXCLUDE_PATHS; do
    prune_args="$prune_args -path $d -prune -o"
done
for n in $EXCLUDE_NAMES; do
    prune_args="$prune_args -name $n -prune -o"
done

# §11.4.65 in-scope roots: project-root *.md, docs/**, scripts/**.
# Hard cap to keep the run bounded (§11.4.65 cap-at-500 spirit).
MAX_FILES=500

# Collect in-scope .md files into a deterministic, newline-safe list.
TMP_LIST="$(mktemp "${TMPDIR:-/tmp}/md_export_list.XXXXXX")"
trap 'rm -f "$TMP_LIST"' EXIT INT TERM

# Root-level *.md (maxdepth 1).
# shellcheck disable=SC2086
find . -maxdepth 1 -type f -name '*.md' -print >>"$TMP_LIST" 2>/dev/null || true
# docs/** and scripts/** recursively, honoring prunes.
for sub in ./docs ./scripts; do
    [ -d "$sub" ] || continue
    # shellcheck disable=SC2086
    find "$sub" $prune_args -type f -name '*.md' -print >>"$TMP_LIST" 2>/dev/null || true
done

# Per-file exclusion: raw DATA sources that are NOT published documents and so
# do NOT get .html/.pdf exports (§11.4.65 governs documents, not data sources).
# docs/features/.feature_inventory_raw.md is a dotfile-prefixed raw inventory
# DATA source (self-described "DATA-GATHERING raw material to seed Status.md",
# per §11.4.153) — Status.md / Status_Summary.md ARE the published feature docs
# and DO get exports; the raw dotfile feeding them does not. MUST stay in sync
# with the identical exclusion in scripts/pre_build_verification.sh
# gate_doc_sibling_sync (§11.4.65 — gate and generator agree on scope).
grep -vxF './docs/features/.feature_inventory_raw.md' "$TMP_LIST" >"$TMP_LIST.f" \
    && mv "$TMP_LIST.f" "$TMP_LIST"

# Sort + de-dup.
sort -u "$TMP_LIST" -o "$TMP_LIST"

TOTAL="$(wc -l <"$TMP_LIST" | tr -d ' ')"
if [ "$TOTAL" -gt "$MAX_FILES" ]; then
    echo "ERROR: $TOTAL in-scope .md exceeds cap of $MAX_FILES — refusing to run" >&2
    echo "       Narrow scope or raise MAX_FILES deliberately." >&2
    exit 2
fi

# is_stale <md> <sibling> -> 0 (true) if sibling missing or older than md.
is_stale() {
    md="$1"; sib="$2"
    [ -f "$sib" ] || return 0
    [ "$md" -nt "$sib" ] && return 0
    return 1
}

gen_html=0
gen_pdf=0
need_html=0
need_pdf=0
fail_html=0
fail_pdf=0
skipped_pandoc=0
skipped_weasy=0
checked_missing=0

while IFS= read -r md; do
    [ -n "$md" ] || continue
    base="${md%.md}"
    html="$base.html"
    pdf="$base.pdf"

    html_stale=1
    pdf_stale=1
    if [ "$FORCE" -eq 1 ]; then
        html_stale=0; pdf_stale=0
    else
        is_stale "$md" "$html" && html_stale=0 || html_stale=1
        is_stale "$md" "$pdf" && pdf_stale=0 || pdf_stale=1
    fi

    if [ "$MODE" = "check" ]; then
        if [ "$html_stale" -eq 0 ] || [ "$pdf_stale" -eq 0 ]; then
            echo "STALE-OR-MISSING: $md"
            checked_missing=$((checked_missing + 1))
        fi
        continue
    fi

    # HTML (pandoc).
    if [ "$html_stale" -eq 0 ]; then
        need_html=$((need_html + 1))
        if [ "$HAVE_PANDOC" -eq 1 ]; then
            if timeout 60 pandoc --standalone --from gfm --to html5 \
                --metadata title="$(basename "$base")" \
                -o "$html" "$md" 2>/dev/null && [ -s "$html" ]; then
                gen_html=$((gen_html + 1))
            else
                echo "FAIL-HTML: pandoc could not render $md" >&2
                rm -f "$html"  # never leave a partial/empty sibling
                fail_html=$((fail_html + 1))
            fi
        else
            skipped_pandoc=$((skipped_pandoc + 1))
        fi
    fi

    # PDF (weasyprint from the freshly-built HTML).
    if [ "$pdf_stale" -eq 0 ]; then
        need_pdf=$((need_pdf + 1))
        if [ "$HAVE_WEASYPRINT" -eq 1 ] && [ -s "$html" ]; then
            if timeout 60 weasyprint "$html" "$pdf" >/dev/null 2>&1 \
                && [ -s "$pdf" ] && head -c 4 "$pdf" | grep -q '%PDF'; then
                gen_pdf=$((gen_pdf + 1))
            else
                echo "FAIL-PDF: weasyprint could not render $html -> $pdf" >&2
                rm -f "$pdf"  # never leave a partial/non-%PDF sibling
                fail_pdf=$((fail_pdf + 1))
            fi
        elif [ "$HAVE_WEASYPRINT" -eq 0 ]; then
            skipped_weasy=$((skipped_weasy + 1))
        else
            # HTML absent (pandoc missing or failed) — cannot honestly build PDF.
            echo "SKIP-PDF: no HTML source for $pdf (pandoc missing/failed)" >&2
            skipped_weasy=$((skipped_weasy + 1))
        fi
    fi
done <"$TMP_LIST"

if [ "$MODE" = "check" ]; then
    echo "----------------------------------------"
    echo "CHECK: in-scope .md = $TOTAL ; stale-or-missing = $checked_missing"
    [ "$checked_missing" -eq 0 ] && { echo "OK: all in-scope .md have fresh siblings"; exit 0; }
    echo "FAIL: $checked_missing in-scope .md lack a fresh sibling (§11.4.65)"
    exit 1
fi

echo "----------------------------------------"
echo "in-scope .md       : $TOTAL"
echo "HTML needed/gen    : $need_html / $gen_html (fail=$fail_html)"
echo "PDF  needed/gen    : $need_pdf / $gen_pdf (fail=$fail_pdf)"
[ "$HAVE_PANDOC" -eq 0 ]     && echo "WARN: pandoc not found — $skipped_pandoc HTML skipped (§11.4.106 honest skip)"
[ "$HAVE_WEASYPRINT" -eq 0 ] && echo "WARN: weasyprint not found — $skipped_weasy PDF skipped (§11.4.106 honest skip)"

# Non-zero exit if any render genuinely failed (anti-bluff: failures are loud).
[ "$fail_html" -eq 0 ] && [ "$fail_pdf" -eq 0 ] || exit 1
exit 0
