#!/usr/bin/env bash
# scripts/testing/update_readme_doc_links.sh
#
# Purpose:   Render the §11.4.57 "Tracked-Items + Status Documents" section into
#            README.md. Discovers the canonical tracked docs (Issues / Fixed /
#            their summaries / CONTINUATION + every docs/**/Status.md and its
#            Status_Summary.md pair), extracts each doc's §11.4.44 `**Revision:**`
#            + `**Last modified:**` header fields, and renders a markdown table
#            (Document | Last modified | Revision | Markdown | HTML | PDF [| DOCX])
#            between the `<!-- doc-link-section:begin -->` /
#            `<!-- doc-link-section:end -->` markers. Idempotent — only the marked
#            region changes; re-running with no doc changes is a no-op diff.
# Usage:     scripts/testing/update_readme_doc_links.sh [README_MD]
#            (default: <repo-root>/README.md)
# Inputs:    docs/Issues.md, docs/Issues_Summary.md, docs/Fixed.md,
#            docs/Fixed_Summary.md, docs/CONTINUATION.md (root canonical set,
#            §11.4.48 carve-out), plus every docs/**/Status.md (auto-discovered
#            via `find docs -name Status.md`) and its sibling Status_Summary.md.
#            ONLY docs that actually exist are linked (§11.4.6 — no fabricated
#            rows). A DOCX column appears ONLY when at least one linked doc has a
#            tracked .docx sibling (§11.4.153 four-format class).
# Outputs:   README.md — the region between the begin/end markers is replaced.
# Side-effects: rewrites README.md only (markers required; aborts if absent).
# Dependencies: POSIX sh/awk/sed, coreutils, git. No network. Parses clean under
#            sh -n + bash -n (§11.4.67).
# Cross-references: §11.4.57 (README doc-link section), §11.4.44 (revision
#            header source), §11.4.59 (README always-sync — exports refreshed by
#            the caller), §11.4.45 / §11.4.56 (Status pair enumeration),
#            §12.10 (CONTINUATION). Companion doc: docs/scripts/update_readme_doc_links.md.

set -eu

SELF_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(git -C "$SELF_DIR" rev-parse --show-toplevel 2>/dev/null || true)"
[ -z "$ROOT" ] && ROOT="$(cd "$SELF_DIR/../.." && pwd)"

README="${1:-$ROOT/README.md}"
BEGIN_MARKER='<!-- doc-link-section:begin -->'
END_MARKER='<!-- doc-link-section:end -->'

[ -f "$README" ] || { echo "error: README not found: $README" >&2; exit 2; }
grep -qF "$BEGIN_MARKER" "$README" || { echo "error: begin marker not found in $README" >&2; exit 3; }
grep -qF "$END_MARKER" "$README" || { echo "error: end marker not found in $README" >&2; exit 3; }

# extract_field <file> <FieldLabel>  → value or empty
extract_field() {
  _f="$1"; _label="$2"
  [ -f "$_f" ] || { printf '%s' ''; return 0; }
  # Match "**Revision:** N" or "**Last modified:** ..." within the header region
  # (first 20 non-blank lines is more than enough for the §11.4.44 block).
  awk -v label="$_label" '
    NR <= 40 {
      pat = "^\\*\\*" label ":\\*\\*[[:space:]]*"
      if ($0 ~ pat) {
        sub(pat, "")
        sub(/[[:space:]]+$/, "")
        print
        exit
      }
    }
  ' "$_f"
}

# Build the ordered list of candidate docs (root canonical set first, then
# discovered Status pairs sorted by path). Only existing files are emitted.
CAND_FILE="$(mktemp "${TMPDIR:-/tmp}/readme_doclinks.XXXXXX")"
trap 'rm -f "$CAND_FILE"' EXIT INT TERM

emit_if_exists() {
  _label="$1"; _rel="$2"
  if [ -f "$ROOT/$_rel" ]; then
    printf '%s\t%s\n' "$_label" "$_rel" >> "$CAND_FILE"
  fi
}

emit_if_exists "Issues"            "docs/Issues.md"
emit_if_exists "Issues Summary"    "docs/Issues_Summary.md"
emit_if_exists "Fixed"             "docs/Fixed.md"
emit_if_exists "Fixed Summary"     "docs/Fixed_Summary.md"
emit_if_exists "CONTINUATION"      "docs/CONTINUATION.md"

# Auto-discover every docs/**/Status.md (and its Status_Summary.md sibling),
# sorted for deterministic output (§11.4.50 / §11.4.86 stable ordering).
if [ -d "$ROOT/docs" ]; then
  find "$ROOT/docs" -name 'Status.md' -type f 2>/dev/null | LC_ALL=C sort | while IFS= read -r status_abs; do
    rel="${status_abs#"$ROOT"/}"
    dir_rel="$(dirname "$rel")"
    # Human label from the parent directory, e.g. docs/features/Status.md -> "features Status"
    base_label="$(basename "$dir_rel")"
    printf '%s\t%s\n' "$base_label Status" "$rel" >> "$CAND_FILE"
    sib="$dir_rel/Status_Summary.md"
    if [ -f "$ROOT/$sib" ]; then
      printf '%s\t%s\n' "$base_label Status Summary" "$sib" >> "$CAND_FILE"
    fi
  done
fi

# Determine whether ANY candidate has a tracked .docx sibling → add DOCX column.
WANT_DOCX=0
while IFS="$(printf '\t')" read -r _label rel; do
  docx="${rel%.md}.docx"
  if [ -f "$ROOT/$docx" ]; then WANT_DOCX=1; break; fi
done < "$CAND_FILE"

# Render the section body into a temp file.
SECTION_FILE="$(mktemp "${TMPDIR:-/tmp}/readme_section.XXXXXX")"
trap 'rm -f "$CAND_FILE" "$SECTION_FILE"' EXIT INT TERM

{
  printf '%s\n' "$BEGIN_MARKER"
  printf '## Tracked-Items + Status Documents\n\n'
  printf 'Auto-generated by `scripts/testing/update_readme_doc_links.sh` (§11.4.57). '
  printf 'Each row shows the document'"'"'s `Revision` + `Last modified` (§11.4.44) and links every synced format (§11.4.60 / §11.4.65). Do not hand-edit between the markers.\n\n'
  if [ "$WANT_DOCX" -eq 1 ]; then
    printf '| Document | Last modified | Revision | Markdown | HTML | PDF | DOCX |\n'
    printf '| --- | --- | --- | --- | --- | --- | --- |\n'
  else
    printf '| Document | Last modified | Revision | Markdown | HTML | PDF |\n'
    printf '| --- | --- | --- | --- | --- | --- |\n'
  fi

  while IFS="$(printf '\t')" read -r label rel; do
    rev="$(extract_field "$ROOT/$rel" "Revision")"
    mod="$(extract_field "$ROOT/$rel" "Last modified")"
    [ -n "$rev" ] || rev="—"
    [ -n "$mod" ] || mod="—"
    md_link="[md]($rel)"
    html_rel="${rel%.md}.html"
    pdf_rel="${rel%.md}.pdf"
    if [ -f "$ROOT/$html_rel" ]; then html_link="[html]($html_rel)"; else html_link="—"; fi
    if [ -f "$ROOT/$pdf_rel" ]; then pdf_link="[pdf]($pdf_rel)"; else pdf_link="—"; fi
    if [ "$WANT_DOCX" -eq 1 ]; then
      docx_rel="${rel%.md}.docx"
      if [ -f "$ROOT/$docx_rel" ]; then docx_link="[docx]($docx_rel)"; else docx_link="—"; fi
      printf '| %s | %s | %s | %s | %s | %s | %s |\n' "$label" "$mod" "$rev" "$md_link" "$html_link" "$pdf_link" "$docx_link"
    else
      printf '| %s | %s | %s | %s | %s | %s |\n' "$label" "$mod" "$rev" "$md_link" "$html_link" "$pdf_link"
    fi
  done < "$CAND_FILE"

  printf '\n'
  printf '%s\n' "$END_MARKER"
} > "$SECTION_FILE"

# Replace the marked region in README.md using awk (no in-place sed dialect issues).
NEW_README="$(mktemp "${TMPDIR:-/tmp}/readme_new.XXXXXX")"
trap 'rm -f "$CAND_FILE" "$SECTION_FILE" "$NEW_README"' EXIT INT TERM

awk -v begin="$BEGIN_MARKER" -v end="$END_MARKER" -v sectionfile="$SECTION_FILE" '
  $0 == begin { inside = 1; while ((getline line < sectionfile) > 0) print line; close(sectionfile); next }
  $0 == end   { inside = 0; next }
  !inside     { print }
' "$README" > "$NEW_README"

cat "$NEW_README" > "$README"
rm -f "$CAND_FILE" "$SECTION_FILE" "$NEW_README"
trap - EXIT INT TERM

echo "update_readme_doc_links: README section refreshed ($README)"
