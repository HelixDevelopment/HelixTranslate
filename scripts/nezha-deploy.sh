#!/usr/bin/env bash
# ============================================================================
# scripts/nezha-deploy.sh — durable nezha deployment wrapper (§11.4.77, #3 fix)
# ============================================================================
# Purpose:
#   Deploy / rebuild / reboot the full HelixTranslate stack on the nezha host
#   from compose.nezha.yml, ALWAYS using .env.nezha as the compose dotenv source
#   so ${VAR} interpolation (ports, JWT_SECRET) resolves correctly out of the
#   box — without the manual `set -a; . .env.nezha` shell-export workaround.
#
#   Root cause this fixes (heavy-testing iter2 finding #3): podman-compose reads
#   ${VAR:-default} substitutions from its DEFAULT dotenv (`.env`) + the shell
#   environment, NOT from a service's `env_file:` directive (which only injects
#   vars INTO the container). The nezha `.env` carries only HELIX_RELEASE_PREFIX,
#   so *_HOST_PORT and JWT_SECRET fell back to their :-defaults (50051/8080/8443/
#   8090, change-me-in-env-nezha) unless the operator manually exported
#   .env.nezha. Passing `--env-file .env.nezha` makes compose use it for ${}
#   substitution — a durable, no-workaround deploy.
#
# Usage:
#   bash scripts/nezha-deploy.sh [up|build|reboot|config|down|ps] [service...]
#     up      (default) podman-compose up -d --build [service...]
#     build              podman-compose build [service...]
#     reboot             up -d --build for the named service(s) (rebuild+restart)
#     config             podman-compose config  (validate interpolation)
#     down               podman-compose down
#     ps                 podman ps for the stack
#
# Inputs:
#   compose.nezha.yml (required), .env.nezha (required, gitignored secrets)
# Outputs:
#   Running/updated stack; exit 0 on success.
# Side-effects:
#   Starts/stops/rebuilds containers via podman-compose.
# Dependencies:
#   podman-compose >= 1.5.0 (supports --env-file), podman.
# Cross-references:
#   compose.nezha.yml, .env.nezha.example, docs/scripts/nezha-deploy.md,
#   Constitution §11.4.77 (regeneration mechanism), §11.4.6 (no-guessing).
# ============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${PROJECT_DIR}"

COMPOSE_FILE="compose.nezha.yml"
ENV_FILE=".env.nezha"

if [ ! -f "${COMPOSE_FILE}" ]; then
	echo "ERROR: ${COMPOSE_FILE} not found in ${PROJECT_DIR}" >&2
	exit 1
fi
if [ ! -f "${ENV_FILE}" ]; then
	echo "ERROR: ${ENV_FILE} not found. Provision it from .env.nezha.example (§11.4.10 secrets)." >&2
	exit 1
fi

# The single durable fix: --env-file ${ENV_FILE} makes podman-compose use
# .env.nezha for ${VAR} interpolation, so ports + JWT never silently fall back.
COMPOSE=(podman-compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}")

action="${1:-up}"
shift || true

case "${action}" in
	up)
		"${COMPOSE[@]}" up -d --build "$@"
		;;
	build)
		"${COMPOSE[@]}" build "$@"
		;;
	reboot)
		# Rebuild, then EXPLICITLY stop+rm+up the named service(s) so the new image
		# actually replaces the running container. podman-compose 1.5.0 does NOT
		# recreate a running container when the rebuilt image keeps the same tag —
		# even `--force-recreate` no-ops in 1.5.0 — so the live container keeps
		# running the OLD binary (§11.4.108 SOURCE→ARTIFACT→RUNTIME gap: the rebuilt
		# image exists but never reaches the user). stop+rm forces podman-compose's
		# subsequent `up` to start a fresh container from the new image.
		# NOTE: podman-compose 1.5.0 has NO `rm` subcommand and its `up --build`
		# / `--force-recreate` do NOT replace a same-tag container. The reliable
		# sequence is: build → compose stop → native `podman rm` of the named
		# containers → compose up (which then creates fresh containers from the
		# new image). Native podman rm is used because compose `rm` does not exist.
		"${COMPOSE[@]}" build "$@"
		if [ "$#" -gt 0 ]; then
			"${COMPOSE[@]}" stop "$@" || true
			for svc in "$@"; do
				# Map compose service name -> container_name (compose.nezha.yml).
				case "${svc}" in
					grpc-server) cname=helixtranslate-grpc ;;
					api-server) cname=helixtranslate-api ;;
					server) cname=helixtranslate-server ;;
					monitor-server) cname=helixtranslate-monitor ;;
					postgres) cname=helixtranslate-postgres ;;
					redis) cname=helixtranslate-redis ;;
					*) cname="${svc}" ;;
				esac
				podman rm -f "${cname}" 2>/dev/null || true
			done
		fi
		"${COMPOSE[@]}" up -d "$@"
		;;
	config)
		"${COMPOSE[@]}" config
		;;
	down)
		"${COMPOSE[@]}" down
		;;
	ps)
		podman ps --filter "name=helixtranslate-" --format "{{.Names}} {{.Status}} {{.Ports}}"
		;;
	*)
		echo "Unknown action: ${action}" >&2
		echo "Usage: $0 [up|build|reboot|config|down|ps] [service...]" >&2
		exit 2
		;;
esac
