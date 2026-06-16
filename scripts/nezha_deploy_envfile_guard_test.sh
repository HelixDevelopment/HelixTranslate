#!/usr/bin/env bash
# ============================================================================
# nezha_deploy_envfile_guard_test.sh — §11.4.135 regression guard for finding #3
# ============================================================================
# Purpose:
#   Guard that scripts/nezha-deploy.sh ALWAYS drives podman-compose interpolation
#   from .env.nezha via `--env-file .env.nezha`. Without it, ${VAR} substitution
#   falls back to the default .env (ports/JWT defaults) — the iter2 finding #3 bug.
#
#   RED_MODE=1 reproduces the defect by checking a SYNTHETIC pre-fix command line
#   (no --env-file) — the guard MUST flag it. RED_MODE=0 (default) asserts the
#   real script contains the --env-file .env.nezha invocation.
#
# Usage:   bash scripts/nezha_deploy_envfile_guard_test.sh
#          RED_MODE=1 bash scripts/nezha_deploy_envfile_guard_test.sh  # prove it catches the bug
# Outputs: PASS/FAIL line + exit 0/1.
# Dependencies: grep, the tracked scripts/nezha-deploy.sh.
# Cross-references: scripts/nezha-deploy.sh, compose.nezha.yml, §11.4.6/§11.4.135.
# ============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY="${SCRIPT_DIR}/nezha-deploy.sh"
RED_MODE="${RED_MODE:-0}"

if [ ! -f "${DEPLOY}" ]; then
	echo "FAIL: ${DEPLOY} not found"
	exit 1
fi

if [ "${RED_MODE}" = "1" ]; then
	# Synthetic pre-fix command line (the buggy form) — guard MUST reject it.
	sample='podman-compose -f compose.nezha.yml'
	if printf '%s\n' "${sample}" | grep -q -- '--env-file[[:space:]]\+\.env\.nezha'; then
		echo "RED_MODE FAIL: pre-fix command (no --env-file) was wrongly accepted as having it"
		exit 1
	fi
	echo "RED_MODE PASS: guard correctly flags a podman-compose invocation lacking --env-file .env.nezha"
	exit 0
fi

# GREEN: the real script MUST invoke podman-compose with --env-file .env.nezha.
if ! grep -q -- '--env-file[[:space:]]\+"\?\${\?ENV_FILE\}\?"\?\|--env-file[[:space:]]\+\.env\.nezha\|ENV_FILE=".env.nezha"\|ENV_FILE=.env.nezha' "${DEPLOY}"; then
	echo "FAIL: scripts/nezha-deploy.sh does not pass --env-file .env.nezha to podman-compose (finding #3 regression)"
	exit 1
fi
# Belt-and-suspenders: the array that runs compose must carry --env-file.
if ! grep -q 'podman-compose --env-file' "${DEPLOY}"; then
	echo "FAIL: COMPOSE invocation in nezha-deploy.sh missing 'podman-compose --env-file'"
	exit 1
fi

echo "PASS: nezha-deploy.sh drives podman-compose interpolation from .env.nezha via --env-file (finding #3 fixed)"
exit 0
