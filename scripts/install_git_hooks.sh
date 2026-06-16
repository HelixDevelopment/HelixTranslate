#!/usr/bin/env bash
# install_git_hooks.sh — idempotent installer for the project's local git hooks.
#
# Purpose:    Symlink (or copy) the tracked hooks in scripts/git_hooks/ into
#             .git/hooks/ so the §11.4.75 Layer-1/3 credential guard runs locally.
# Usage:      bash scripts/install_git_hooks.sh
# Inputs:     none (operates on the repo it is run from).
# Outputs:    installs pre-commit + pre-push hooks; prints what it did.
# Side-effects: writes .git/hooks/{pre-commit,pre-push}; backs up any pre-existing
#               non-managed hook to <hook>.local-backup once.
# Dependencies: bash, git.
# Cross-references: scripts/git_hooks/*; docs/scripts/credential_scan.md;
#                   constitution §11.4.75 (mechanical enforcement), §11.4.10.A clause 5.
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
SRC="$ROOT/scripts/git_hooks"
DST="$ROOT/.git/hooks"
mkdir -p "$DST"

install_one() {
  local name="$1" src="$SRC/$1" dst="$DST/$1"
  [ -f "$src" ] || { echo "skip: $src missing"; return 0; }
  chmod +x "$src"
  # Preserve a pre-existing, non-managed hook exactly once.
  if [ -e "$dst" ] && [ ! -L "$dst" ] && ! grep -q '§11.4.75' "$dst" 2>/dev/null; then
    [ -e "$dst.local-backup" ] || cp "$dst" "$dst.local-backup"
  fi
  # Use a copy (symlinks are not portable across some git GUIs); idempotent.
  cp "$src" "$dst"
  chmod +x "$dst"
  echo "installed: .git/hooks/$name"
}

install_one pre-commit
install_one pre-push
chmod +x "$SRC/credential_scan.sh" 2>/dev/null || true
echo "git hooks installed (§11.4.75 credential guard active)."
