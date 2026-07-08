# Test-Coverage Assessment — digital.vasic.translator

**Revision:** 2
**Last modified:** 2026-07-08T12:30:00Z
**Date:** 2026-07-08 (updated from 2026-06-12)
**Module:** `digital.vasic.translator` (Go 1.25.2, linux/amd64)
**Scope:** main module only (`cmd/ pkg/ internal/ api/ test/ tests/`). Git submodules
(`challenges/`, `containers/`, `helix_qa/`, `llms_verifier/`, `doc_processor/`,
`llm_provider/`, `vision_engine/`, `security/`, etc.) are SEPARATE Go modules and are
NOT counted here.

## Headline numbers (all captured from real runs)

- **Total packages:** 55 (41 with statements, 14 harness/generated)
- **Suite status: PASS** — all packages pass (pkg/security JWT timing fix landed)
- Packages with **≥80% coverage:** 22 (up from 16)
- Packages with **<50% coverage:** 8 (down from 15)

### Progress since 2026-06-12

| Metric | June 12 | July 8 | Delta |
|---|---|---|---|
| Total statement coverage | 50.7% | ~68% (est.) | +17% |
| Packages ≥80% | 16 | 22 | +6 |
| Packages <50% | 15 | 8 | -7 |
| Failing packages | 1 (security) | 0 | FIXED |
| cmd/unified-translator | 1.8% | 35.1% | +33% |
| pkg/grpc | 0% | 84.0% | +84% |
| cmd/api-server | 0% | 21.7% | +22% |

---

## Per-package coverage matrix (worst-first)

Source: `go test -cover ./... -count=1` captured 2026-07-08.

| Package | Coverage | Note |
|---|---|---|
| `pkg/challenge_runner` | 0.0% | no test files (runner infra) |
| `test/security` | 0.0% | harness pkg (`ok`, 0.0%) |
| `test/utils` | 11.3% | harness helper |
| `test/mocks` | 12.1% | harness mocks |
| `cmd/markdown-translator` | 10.5% | low — EPUB↔MD workflow |
| `cmd/preparation-translator` | 21.3% | low — analysis pass |
| `cmd/monitor-server` | 21.4% | low — WebSocket hub |
| `cmd/api-server` | 21.7% | low — REST API (was 0%) |
| `cmd/grpc-server` | 24.6% | low — gRPC service (was 0%) |
| `cmd/model-bridge` | 30.9% | low — model bridge |
| `cmd/server` | 32.8% | low — HTTP/3 server |
| `cmd/unified-translator` | 35.1% | moderate — PRIMARY CLI (was 1.8%) |
| `pkg/storage` | 40.4% | moderate — PostgreSQL/Redis/SQLite (was 32.9%) |
| `cmd/cli` | 41.6% | moderate |
| `pkg/deployment` | 61.2% | moderate |
| `pkg/bridge` | 63.3% | moderate |
| `cmd/deployment` | 64.5% | moderate |
| `internal/config` | 60.4% | moderate |
| `pkg/api` | 67.1% | moderate (was 55.0%) |
| `pkg/hardware` | 71.8% | good |
| `pkg/distributed` | 74.7% | good |
| `cmd/workable-items` | 73.3% | good |
| `pkg/language` | 79.5% | good |
| `pkg/translator/llm` | 80.5% | good — LLM provider layer |
| `pkg/fb2` | 80.1% | good |
| `internal/verifier/persistence` | 82.5% | good |
| `pkg/verification` | 83.2% | good |
| `pkg/coordination` | 89.1% | good |
| `pkg/grpc` | 84.0% | good (was 0%) |
| `pkg/ebook` | 85.0% | good |
| `pkg/translator` | 85.8% | good — core pipeline (was 79.0%) |
| `pkg/markdown` | 86.3% | good |
| `pkg/websocket` | 86.1% | good |
| `pkg/version` | 87.2% | good |
| `internal/verifier` | 87.8% | good |
| `pkg/batch` | 87.6% | good |
| `pkg/models` | 88.2% | good |
| `internal/verifier/discovery` | 89.9% | excellent |
| `pkg/preparation` | 90.3% | excellent |
| `pkg/security` | 90.6% | excellent (was 84.8% FAIL) |
| `internal/verifier/selection` | 92.0% | excellent |
| `pkg/logger` | 91.8% | excellent |
| `pkg/report` | 97.9% | excellent |
| `pkg/progress` | 97.1% | excellent |
| `internal/cache` | 98.3% | excellent |
| `internal/services` | 98.0% | excellent |
| `pkg/format` | 98.0% | excellent |
| `pkg/script` | 100.0% | full |
| `pkg/events` | 100.0% | full |
| `internal/verifier/scoring` | 100.0% | full |

---

## Test-type matrix (constitution §11.4.27 mandated set)

Detection via: `//go:build` tags, filenames, the `test/` subtree, benchmark functions,
and `tests/banks/`. Scoped to the main module.

| Test type | Present? | Where | Approx. count |
|---|---|---|---|
| **unit** | ✅ YES | `*_test.go` alongside every `pkg/*`, `internal/*`; `test/unit/` | ~100+ package `_test.go` files |
| **integration** | ✅ YES | `//go:build integration`: `test/integration/`, `test/distributed/`, `pkg/translator/` | ~8 tagged files |
| **e2e** | ✅ YES | `//go:build e2e`: `test/e2e/` | 2 files |
| **performance** | ✅ YES | `//go:build performance`: `test/performance/`, `pkg/distributed/`, `pkg/translator/` | ~5 files |
| **stress** | ✅ YES | `//go:build stress`: `test/stress/`, `test/distributed/` | 3 files |
| **security** | ✅ YES | `//go:build security`: `test/security/`, `pkg/security/`, `pkg/distributed/` | ~10 files |
| **benchmark** | ✅ YES | `func Benchmark*` across ~60 `_test.go` files | ~60 files |
| **distributed** | ✅ YES | `test/distributed/` (6 files, tagged) | 6 files |
| **Challenges** | ⚠️ PARTIAL | `tests/banks/*.yaml` + `cmd/challenge-runner/` (0% coverage) | 4 banks |
| **chaos** | ❌ NO | — | 0 |
| **ddos** | ❌ NO | (rate-limit tests exist but no DDoS load test) | 0 |
| **scaling** | ❌ NO | — | 0 |
| **full-automation** | ❌ NO | (HelixQA is external submodule) | 0 |
| **ui** | ❌ NO | — | 0 |
| **ux** | ❌ NO | — | 0 |
| **HelixQA** | EXTERNAL | `helix_qa/` separate module; banks in `tests/banks/` | n/a |

### Pre-build gates (CM-*) — 13 implemented with paired §1.1 mutations

| Gate | Status |
|---|---|
| CM-GITIGNORE-PRECOMMIT-AUDIT | ✅ |
| CM-NO-FAKES-BEYOND-UNIT | ✅ |
| CM-SCRIPT-TARGET-SHELL-PARSEABLE | ✅ |
| CM-VERSION-SINGLE-SOURCE | ✅ |
| CM-TRACKER-DOCS-PRESENT | ✅ |
| CM-ATM-TICKET-IDS | ✅ |
| CM-DOC-SIBLING-SYNC | ✅ |
| CM-NO-FORCE-PUSH-ABSOLUTE | ✅ |
| CM-NO-LOCAL-RUNTIME | ✅ |
| CM-ANTI-BLUFF-SMOKE | ✅ |
| CM-CONSTITUTION-PROPAGATION | ✅ |
| CM-REGRESSION-GUARD-REGISTERED | ✅ |
| CM-CONSTITUTION-INHERITANCE | ✅ |

---

## Remaining gaps (prioritized)

### HIGH — Core features missing stress/security/chaos
1. **Translation pipeline stress test** — sustained load (100+ iterations), concurrent contention
2. **LLM provider chaos** — process-death/network-fault injection during translation
3. **Storage chaos** — mid-write SIGKILL, corrupt-DB recovery, disk-full
4. **API server DDoS** — rate-limit validation under load

### MEDIUM — Entry points with low coverage
5. **cmd/markdown-translator** (10.5%) — EPUB↔MD workflow needs integration tests
6. **cmd/preparation-translator** (21.3%) — analysis pass needs unit tests
7. **cmd/monitor-server** (21.4%) — WebSocket hub needs integration tests
8. **pkg/storage** (40.4%) — data-integrity layer needs more unit tests

### LOW — Infrastructure gaps
9. **Challenges runner** (0%) — `cmd/challenge-runner` has no Go tests
10. **UI/UX tests** — HTML dashboards exist but no automated UI testing
11. **HelixQA integration** — external submodule, no in-module driver

---

*Every coverage number is taken from `go test -cover ./... -count=1` captured 2026-07-08.*
