#!/bin/sh
# nezha-entrypoint.sh — per-service dispatch for the multi-binary nezha image.
#
# Purpose:
#   One image (Containerfile.nezha) ships all four runtime services; this script
#   execs exactly one based on the SERVICE environment variable. Used by
#   compose.nezha.yml (each service sets its own SERVICE value).
#
# Inputs (env):
#   SERVICE   grpc | api | server | monitor   (required; defaults to "api")
#   plus each service's own env (GRPC_PORT, HTTP_PORT, GRPC_ADDRESS,
#   MONITOR_SERVER_PORT, DB_*, REDIS_*, provider *_API_KEY, JWT_SECRET, ...).
#
# Outputs / side-effects:
#   exec()s the selected long-running server process (PID 1 signal handling).
#
# Dependencies: the four binaries under /app (built by Containerfile.nezha).
# Cross-references: Containerfile.nezha, compose.nezha.yml, .env.nezha.example.
#
# POSIX-sh only (busybox sh in alpine) — §11.4.67 target-shell parseable.

set -eu

SERVICE="${SERVICE:-api}"

case "$SERVICE" in
  grpc)
    echo "nezha-entrypoint: starting grpc-server (:${GRPC_PORT:-50051})"
    exec /app/grpc-server "$@"
    ;;
  api)
    echo "nezha-entrypoint: starting api-server HTTP (:${HTTP_PORT:-8080})"
    exec /app/api-server "$@"
    ;;
  server)
    echo "nezha-entrypoint: starting server TLS REST/WS (:8443)"
    exec /app/server -config /app/config/config.json "$@"
    ;;
  monitor)
    echo "nezha-entrypoint: starting monitor-server (:${MONITOR_SERVER_PORT:-8090})"
    exec /app/monitor-server "$@"
    ;;
  *)
    echo "nezha-entrypoint: ERROR unknown SERVICE='$SERVICE'" >&2
    echo "  valid values: grpc | api | server | monitor" >&2
    exit 64
    ;;
esac
