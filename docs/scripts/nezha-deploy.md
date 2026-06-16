# nezha-deploy.sh

**Revision:** 1
**Last modified:** 2026-06-16T00:00:00Z

## Overview

Durable deployment wrapper for the full HelixTranslate stack on the **nezha**
host. It always invokes `podman-compose` with `--env-file .env.nezha`, fixing
heavy-testing iteration-2 finding #3: a fresh `podman-compose up` previously
interpolated `${VAR}` substitutions (ports, `JWT_SECRET`) from the default
`.env` (which on nezha holds only `HELIX_RELEASE_PREFIX`) instead of
`.env.nezha`, so ports/JWT silently fell back to their `:-default` values
unless the operator manually exported `.env.nezha` into the shell.

## Prerequisites

- `podman` + `podman-compose >= 1.5.0` (the `--env-file` flag).
- `compose.nezha.yml` and a provisioned `.env.nezha` (gitignored; see
  `.env.nezha.example`) present in the project root.

## Usage examples

```bash
# Fresh from-scratch boot of the whole stack (correct ports/JWT, no workaround):
bash scripts/nezha-deploy.sh up

# Rebuild + reboot a single service after a source fix:
bash scripts/nezha-deploy.sh reboot api-server
bash scripts/nezha-deploy.sh reboot server

# Validate that interpolation resolves to the real ports (not defaults):
bash scripts/nezha-deploy.sh config | grep -E '18080|18443|50061|18090'

# Stack status / teardown:
bash scripts/nezha-deploy.sh ps
bash scripts/nezha-deploy.sh down
```

## Edge cases

- Missing `compose.nezha.yml` or `.env.nezha` → script exits non-zero with an
  actionable message (never deploys with defaults silently).
- Unknown action → usage message + exit 2.

## Internal behaviour

The single load-bearing line is the `COMPOSE` array:
`podman-compose --env-file .env.nezha -f compose.nezha.yml`. Every action
funnels through it, so no code path can deploy without `.env.nezha` driving
interpolation.

## Related scripts

- `scripts/nezha-entrypoint.sh` — per-container binary selector (`SERVICE` env).

## Last verified

2026-06-16 — `podman-compose config` on nezha resolves the real
18080/18443/50061/18090 ports + the real JWT_SECRET via this wrapper.
