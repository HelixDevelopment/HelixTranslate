# HelixQA → HelixTranslate Full Integration Plan
## Enterprise-Grade Cutting-Edge Translation System — QA Integration

---

> **Document Version:** 1.0  
> **Date:** 2025  
> **Status:** Master Plan — Ready for Execution  
> **Repositories Analyzed:**
> - `HelixDevelopment/HelixTranslate` (main system)
> - `HelixDevelopment/HelixQA` (QA framework)
> - `vasic-digital/Catalogizer` (reference implementation)
>
> **Analysis Artifacts:**
> - `/home/z/my-project/analysis-helixtranslate.md` — Full HelixTranslate analysis
> - `/home/z/my-project/analysis-helixqa.md` — Full HelixQA analysis
> - `/home/z/my-project/analysis-catalogizer.md` — Full Catalogizer reference analysis

---

## TABLE OF CONTENTS

1. [Executive Summary](#1-executive-summary)
2. [Architecture Overview & Current State](#2-architecture-overview--current-state)
3. [Dependency Chain Analysis](#3-dependency-chain-analysis)
4. [Phase 0: Pre-Integration Readiness Audit](#phase-0-pre-integration-readiness-audit)
5. [Phase 1: Constitutional Foundation](#phase-1-constitutional-foundation)
6. [Phase 2: Submodule Integration](#phase-2-submodule-integration)
7. [Phase 3: Testing Strategy & Implementation](#phase-3-testing-strategy--implementation)
8. [Phase 4: Screenshot & On-Demand Capture System](#phase-4-screenshot--on-demand-capture-system)
9. [Phase 5: UX Enterprise-Grade Considerations](#phase-5-ux-enterprise-grade-considerations)
10. [Anti-Bluff Testing Constitution (Critical)](#anti-bluff-testing-constitution)
11. [Full-QA Master Cycle](#full-qa-master-cycle)
12. [Mapping: Test Types → HelixTranslate Components](#mapping-test-types--helixtranslate-components)
13. [Risk Register & Mitigation](#risk-register--mitigation)

---

## 1. EXECUTIVE SUMMARY

### 1.1 Objective
Integrate **HelixQA** (AI-driven QA orchestration framework, v0.2.0) with all its dependency submodules into **HelixTranslate** (enterprise-grade ebook translation system, v2.3.0) to achieve **100% test coverage** across all supported test types with **anti-bluff guarantees** — meaning passing tests MUST confirm working features.

### 1.2 Scope
The integration covers:
- **All system APIs**: REST API (port 8080/8443), gRPC service (port 50051), WebSocket monitoring (port 8090)
- **All client applications**: Web Dashboard (monitor.html), CLI (11 binaries), API consumers
- **All translation providers**: OpenAI, Anthropic, Zhipu, DeepSeek, Qwen, Gemini, Ollama, LlamaCpp
- **All ebook formats**: FB2, EPUB, TXT, HTML, PDF, DOCX
- **Screenshot capture**: On-demand capture from Web dashboard and all CLI tools during QA sessions

### 1.3 Key Constraint
> *"We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used!"*

This mandates the **Anti-Bluff Testing Constitution** — every test MUST validate real user-visible behavior, and MUST fail when the feature it tests is removed.

### 1.4 Reference Implementation
The `vasic-digital/Catalogizer` repository serves as the proven reference implementation. It demonstrates:
- 43 git submodules including 9 AI/QA modules
- Constitutional governance with Articles I-XI
- Full-QA Master Cycle (10-step mandatory loop)
- 4-artefact fix loop per defect
- Vision-driven testing with stagnation detection
- Session archiving with video + screenshots + tickets

---

## 2. ARCHITECTURE OVERVIEW & CURRENT STATE

### 2.1 HelixTranslate Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    HelixTranslate System                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │ REST API │  │  gRPC    │  │WebSocket │  │ 11 CLI   │   │
│  │ :8080    │  │ :50051   │  │ Monitor  │  │ Binaries │   │
│  │ Gin      │  │ protobuf │  │ :8090    │  │ cobra    │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │              │              │              │         │
│  ┌────▼──────────────▼──────────────▼──────────────▼─────┐  │
│  │              EventBus (pub/sub)                        │  │
│  │         pkg/events/events.go                           │  │
│  └────┬──────────────┬──────────────┬──────────────┬─────┘  │
│       │              │              │              │         │
│  ┌────▼─────┐  ┌────▼─────┐  ┌────▼─────┐  ┌────▼─────┐  │
│  │Translation│  │Distributed│  │ Storage  │  │Security  │  │
│  │ Engine    │  │ SSH Workers│  │ PG/R/SQ │  │ JWT/CORS │  │
│  │8 providers│  │Coord/Fall│  │          │  │          │  │
│  └────┬─────┘  └──────────┘  └──────────┘  └──────────┘  │
│       │                                                      │
│  ┌────▼──────────────────────────────────────────────────┐  │
│  │           LLM Provider Layer                          │  │
│  │  OpenAI │ Anthropic │ Zhipu │ DeepSeek │ Qwen │ ...  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │           Web Dashboard                                │  │
│  │  monitor.html / enhanced-monitor.html                  │  │
│  │  web/templates/dashboard.html                          │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Current Test Coverage (Gap Analysis)

| Category | Current State | Target |
|----------|---------------|--------|
| Unit Tests | ~43.6% coverage | 100% |
| Integration Tests | `test/integration/` exists | Full coverage |
| E2E Tests | `test/e2e/` exists | Full coverage |
| Performance Tests | `test/performance/` exists | Full coverage |
| Stress Tests | `test/stress/` exists | Full coverage |
| Security Tests | `test/security/` exists | Full coverage |
| Distributed Tests | `test/distributed/` exists | Full coverage |
| **Challenges** | 2 challenges (CONST-033 only) | All features |
| **HelixQA Banks** | **NONE** | Full coverage |
| **Autonomous QA** | **NONE** | All platforms |
| **Anti-Bluff Validation** | **NONE** | 100% of tests |
| **Screenshot Capture** | **NONE** | All clients |
| **API Endpoint Tests** | Partial (handler_test.go) | All 30+ endpoints |
| **LLM Provider Tests** | Mock only (mock.go) | Real provider validation |

### 2.3 Current Submodules

```
.gitmodules currently contains:
  [submodule "Containers"]
      path = Containers
      url = git@github.com:vasic-digital/Containers.git
  [submodule "Challenges"]
      path = Challenges
      url = git@github.com:vasic-digital/Challenges.git

go.mod replace directives:
  replace digital.vasic.challenges => ./Challenges
  replace digital.vasic.containers => ./Containers
```

---

## 3. DEPENDENCY CHAIN ANALYSIS

### 3.1 HelixQA Full Dependency Tree

HelixQA requires **8 sibling Go modules** co-located as sibling directories in the parent workspace, plus **26 git submodules** for open-source tools.

#### 3.1.1 Required Sibling Modules (MUST be added as git submodules)

| Module | Repository | Purpose | Required For |
|--------|-----------|---------|-------------|
| `HelixQA` | `git@github.com:HelixDevelopment/HelixQA.git` | QA orchestration framework | **All QA operations** |
| `Challenges` | `git@github.com:vasic-digital/Challenges.git` | Test execution engine | **Already exists** ✓ |
| `Containers` | `git@github.com:vasic-digital/Containers.git` | Container lifecycle | **Already exists** ✓ |
| `DocProcessor` | `git@github.com:HelixDevelopment/DocProcessor.git` | Documentation loading, feature maps | Autonomous QA (Learn phase) |
| `LLMOrchestrator` | `git@github.com:HelixDevelopment/LLMOrchestrator.git` | CLI agent management | Autonomous QA (agent pool) |
| `LLMProvider` | `git@github.com:HelixDevelopment/LLMProvider.git` | LLM API abstraction | HelixQA LLM calls |
| `VisionEngine` | `git@github.com:HelixDevelopment/VisionEngine.git` | GoCV + LLM Vision analysis | Vision-driven testing |
| `Security` | `git@github.com:HelixDevelopment/Security.git` | CORS, CSP, sanitization | Web testing |
| `LLMsVerifier` | `git@github.com:HelixDevelopment/LLMsVerifier.git` | Strategy-based LLM selection/scoring | Model selection in autonomous mode |

#### 3.1.2 Optional Sibling Modules (Nice-to-have)

| Module | Repository | Purpose |
|--------|-----------|---------|
| `ScreenDiff` | `git@github.com:vasic-digital/ScreenDiff.git` | Screen comparison for visual regression |
| `ReplayBuffer` | `git@github.com:vasic-digital/ReplayBuffer.git` | Replay recording |
| `VisualRegression` | `git@github.com:vasic-digital/VisualRegression.git` | Visual regression testing |
| `TrainingCollector` | `git@github.com:vasic-digital/TrainingCollector.git` | Training data collection |

#### 3.1.3 Open-Source Tool Submodules (inside HelixQA)

These are **nested submodules** inside the HelixQA directory and are initialized by `git submodule update --recursive`. For HelixTranslate's Web/API testing focus, the critical ones are:

| Submodule | Purpose | Relevance to HelixTranslate |
|-----------|---------|---------------------------|
| `tools/opensource/chromedp` (via go.mod) | Browser automation | **Critical** — Web dashboard testing |
| `tools/opensource/go-rod` (via go.mod) | Headless Chrome | **Critical** — Alternative browser automation |
| `tools/opensource/allure2` | Test reporting | Useful — Test report generation |
| `tools/opensource/stagehand` | Browser automation | Useful — Additional browser support |
| `tools/opensource/browser-use` | AI browser agent | Useful — AI-driven browser testing |
| `tools/opensource/skyvern` | AI workflow automation | Useful — Complex flow testing |

### 3.2 go.mod Wiring

The HelixTranslate `go.mod` must add `replace` directives for each new sibling module:

```go
// Existing (KEEP):
replace digital.vasic.challenges => ./Challenges
replace digital.vasic.containers => ./Containers

// New — HelixQA ecosystem:
replace digital.vasic.helixqa => ./HelixQA
replace digital.vasic.docprocessor => ./DocProcessor
replace digital.vasic.llmorchestrator => ./LLMOrchestrator
replace digital.vasic.llmprovider => ./LLMProvider
replace digital.vasic.security => ./Security
replace digital.vasic.visionengine => ./VisionEngine
replace digital.vasic.llmsverifier => ./LLMsVerifier
```

### 3.3 System Dependencies

| Dependency | Purpose | Platform |
|-----------|---------|----------|
| `chromedp` | Browser automation (headless Chrome) | Linux/macOS/Windows |
| `go-rod/rod` | Alternative browser automation | Linux/macOS/Windows |
| `gocv.io/x/gocv` | OpenCV bindings | Linux (needs OpenCV) |
| `ffmpeg` | Video recording/processing | All platforms |
| `xdotool` | Desktop UI automation (Linux) | Linux X11 |
| `Ollama` | Local LLM inference | All platforms |
| `llama.cpp` | Distributed model inference | GPU hosts |

---

## Phase 0: Pre-Integration Readiness Audit

### 0.1 Validate Existing Infrastructure

**Task 0.1.1: Confirm Go version compatibility**
```bash
go version  # Must be >= 1.25.2 (HelixTranslate requires 1.25.2, HelixQA requires 1.24+)
# HelixQA go.mod says 1.26 but HelixQA CLAUDE.md says 1.24+
# Test with actual build before committing
```

**Task 0.1.2: Verify existing submodules are clean**
```bash
cd HelixTranslate/
git submodule status  # Must show both Containers and Challenges
ls -la Containers/ Challenges/  # Must not be empty
cd Containers && git log -1 --oneline  # Must have real commits
cd ../Challenges && git log -1 --oneline
```

**Task 0.1.3: Verify existing test infrastructure**
```bash
make test 2>&1 | tail -20       # All existing tests must pass
make test-coverage 2>&1 | tail -5 # Record current coverage baseline
make challenges 2>&1 | tail -10  # All existing challenges must pass
```

**Task 0.1.4: Verify API server starts**
```bash
make dev  # Start gRPC + API servers
curl -sk https://localhost:8443/health  # Must return healthy
curl -sk https://localhost:8443/api/v1/providers  # Must list providers
curl -sk https://localhost:8443/api/v1/languages  # Must list languages
# Press Ctrl+C to stop
```

**Task 0.1.5: Verify Web dashboard loads**
```bash
make dev &
sleep 5
curl -s http://localhost:8090/monitor | head -20  # Dashboard HTML must load
```

**Task 0.1.6: Record baseline metrics**
| Metric | Current Value | Target |
|--------|---------------|--------|
| Go test coverage | ~43.6% | 100% |
| Test count | (record `go test ./... -v 2>&1 \| grep -c "^---"`) | (track) |
| Challenge count | 2 | 50+ |
| HelixQA banks | 0 | 5+ |
| API endpoint coverage | Partial | 100% |

### 0.2 Resolve Potential Conflicts

**Task 0.2.1: Check for `digital.vasic.security` import conflicts**
- HelixTranslate has its own `pkg/security/` package
- HelixQA requires `digital.vasic.security` as a sibling module
- These MUST NOT conflict — verify by checking all import paths
- **Resolution**: `digital.vasic.security` is the external module; `pkg/security/` is internal to HelixTranslate

**Task 0.2.2: Check for `digital.vasic.containers` usage overlap**
- HelixTranslate already uses `digital.vasic.containers` via replace directive
- HelixQA also requires it
- **Resolution**: Same module, no conflict — both reference `./Containers`

**Task 0.2.3: Check for testify version compatibility**
- HelixTranslate: `github.com/stretchr/testify v1.11.1`
- HelixQA: `github.com/stretchr/testify v1.11.1`
- **Status**: ✅ Compatible

---

## Phase 1: Constitutional Foundation

### Objective
Establish the governance framework that mandates quality, anti-bluff testing, and the Full-QA Master Cycle. This phase makes quality non-negotiable at the constitutional level.

### 1.1 Update CONSTITUTION.md

**File**: `HelixTranslate/CONSTITUTION.md`  
**Current**: 98 lines, contains CONST-033 only  
**Action**: Add Articles V, VII, VIII, IX, and XI (modeled after Catalogizer)

**Task 1.1.1: Add Article V — 100% Test Coverage**

Insert after existing "## Mandatory Standards" section:

```markdown
### Article V — 100% Test Coverage Across All Categories (MANDATORY)

**§5.1 Required Test Categories**

Every component, service, CLI binary, and API endpoint MUST maintain **no less than 100% coverage** across **every** one of the following test categories:

1. **Unit tests** — individual function/struct behavior, pure logic, edge cases
2. **Integration tests** — cross-module interactions (EventBus→WebSocket, Translator→Storage, API→gRPC)
3. **End-to-end (E2E) tests** — full translation pipelines through live system (upload→translate→verify→download)
4. **Full automation tests** — unattended, reproducible versions of the E2E suite
5. **Stress tests** — high concurrency, large files, sustained load, long-running sessions
6. **Security tests** — JWT auth, API key validation, input sanitization, SQL injection, rate limiting
7. **DDoS / rate-limit tests** — sustained floods, burst attacks, connection exhaustion
8. **Benchmarking** — latency/throughput/memory baselines with regression detection per LLM provider
9. **Challenges** — `digital.vasic.challenges` framework: green challenge for every feature
10. **HelixQA QA testing** — autonomous LLM-driven sessions covering every API endpoint, every CLI binary, every translation provider, every ebook format

**§5.2 "100%" Definition**

Every public function / API endpoint / CLI command has at least one test in each applicable category. Every branch exercised. Every LLM provider integration tested with real API calls (or documented skip with `SKIP-OK: #<ticket>`). Every ebook format parse+translate+verify cycle tested. Every fix has a regression test.

**§5.3 Application-Specific Coverage Targets**

| Component | Coverage Categories |
|-----------|-------------------|
| REST API (`pkg/api/`) | 1-10 (all categories) |
| gRPC Service (`pkg/grpc/`) | 1-10 (all categories) |
| WebSocket Hub (`pkg/websocket/`) | 1-10 (all categories) |
| Translation Engine (`pkg/translator/`) | 1-10 (all categories) |
| LLM Providers (8 files) | 1-10 (all categories) |
| Ebook Parsers (6+ formats) | 1,2,3,9,10 |
| Storage Layer (`pkg/storage/`) | 1-7,9 |
| Distributed System (`pkg/distributed/`) | 1-7,9,10 |
| EventBus (`pkg/events/`) | 1-9 |
| Security (`pkg/security/`) | 1-7,9 |
| Format Detection (`pkg/format/`) | 1,2,9,10 |
| Verification (`pkg/verification/`) | 1-10 |
| Preparation (`pkg/preparation/`) | 1-10 |
| CLI Binaries (11 cmd/ entries) | 1-4,8-10 |
| Web Dashboard (monitor.html) | 10 (HelixQA visual) |
| Batch Processing (`pkg/batch/`) | 1-10 |

**§5.4 Mandatory Retesting Loop**

After any change: rebuild → execute every category → analyze → open tickets → root-cause fix + regression test → return to step 1. Terminates only when every category passes.

**§5.5 Violation Consequences**

Shipping is **prohibited** while any category is incomplete or any ticket is open.
```

**Task 1.1.2: Add Article VII — Full-QA Master Cycle**

```markdown
### Article VII — Full-QA Master Cycle (MANDATORY)

**Every production QA effort must follow this rigid loop. Partial execution is prohibited.**

**§7.1 Mandatory Preconditions**
- All binaries rebuilt from clean slate (`make clean && make build`)
- `.env` must supply all LLMsVerifier-scored model keys so HelixQA runs with real vision models
- At least one LLM provider configured with valid API key for real translation testing
- Docker Compose stack running (PostgreSQL 16, Redis 7) for integration tests

**§7.2 Execution Order:**
1. Clean rebuild (`make clean && make build-all`)
2. Unit + integration tests (`make test-coverage`)
3. Challenges bank run (`./scripts/run-all-challenges.sh`)
4. HelixQA bank tests (`helixqa run --banks tests/banks/ --platform all`)
5. Full autonomous QA (`helixqa autonomous --project . --platforms web,api,cli`)
6. Video + screenshot post-session review
7. Ticket creation for every defect
8. Root-cause fixes + 4-artefact tail (unit test + fixes-validation entry + HelixQA bank entry + challenge)
9. Full rebuild, re-run from step 1 until clean pass
10. Version-code bump; release artefacts archived

**§7.3 Stop Conditions:**
- **Clean pass**: All tests, all challenges, all banks, all autonomous sessions pass → version bump + release
- **FATAL BLOCKER**: Unresolvable infrastructure issue → pause
- **NOTHING LEFT**: No more fixes possible → stop

**§7.4 HelixQA Coverage Contract**
Plan every API endpoint, every CLI command, every translation provider, every ebook format, every edge case. Use real ebook files. Scripted generic inputs prohibited.

**§7.5 Evidence and Ticketing**
Video reference + MM:SS timestamp, screenshots, session ID, reproduction path, root cause.

**§7.6 Fixes Validation — 4-Artefact Requirement**
Every fix MUST produce these 4 artefacts in the same commit:
1. **Unit test** — regression guard in `pkg/*/<feature>_test.go`
2. **Integration test** — end-to-end verification in `test/integration/`
3. **`fixes-validation.yaml` entry** — permanent bank entry in `tests/banks/fixes-validation.yaml`
4. **Challenge registration** — platform-specific verification in `challenges/scripts/`

**§7.7 Live Monitoring**
Console shows current test, progress, running/final result.

**§7.8 Archiving**
Every session to `docs/reports/qa-sessions/<YYYY-MM-DD-THH-MM>/` with: FINAL-REPORT.md, logs/, challenges/, helixqa/, videos/, screenshots/, tickets/, analysis/

**§7.9 Violation Enforcement**
Shipping prohibited without clean pass.
```

**Task 1.1.3: Add Article VIII — Live Log Monitoring**

```markdown
### Article VIII — Real-Time Log Monitoring (MANDATORY)

**§8.1 All QA Sessions Must Monitor Logs in Real-Time**

During ANY QA session execution, real-time log monitoring is **MANDATORY** for:
- **REST API**: Application logs, access logs, error logs
- **gRPC Service**: Service logs, error logs
- **WebSocket Monitor**: Hub connection logs, event dispatch logs
- **Translation Engine**: Provider API call logs, error logs
- **Distributed Workers**: SSH connection logs, worker health logs
- **All LLM Providers**: API request/response logs, error traces

**§8.2 Implementation Requirements**
1. HelixQA must capture and stream logs for all tested services simultaneously
2. Log analysis must happen in real-time, not post-session
3. All log outputs must be saved to session directory
4. Log monitoring is NOT optional — no QA session is valid without it

**§8.3 Violation Consequences**
Any QA session conducted without real-time log monitoring is **INVALID**.
```

**Task 1.1.4: Add Article IX — HelixQA Tool Hygiene**

```markdown
### Article IX — HelixQA Tool Hygiene (MANDATORY)

**§9.1 HelixQA is the Sole Authorized Tool**
All automated UI/UX testing of the Web Dashboard and API endpoints MUST use HelixQA. No custom Playwright scripts, no curl-based test harnesses outside HelixQA banks.

**§9.2 Vision-Driven Only**
All navigation of the Web Dashboard MUST be vision-driven: screenshot → LLM analysis → action decision. No hardcoded element selectors, no CSS selectors, no sleep timers.

**§9.3 Universal Solution Principle**
When HelixQA cannot interact with a HelixTranslate UI element, the fix MUST be implemented in HelixQA itself (improve navigation engine), never by adding test hooks to HelixTranslate.

**§9.4 Screenshot Quality**
All screenshots captured during HelixQA sessions must pass `IsBlankScreenshot()` validation. Blank or black screenshots are invalid evidence.

**§9.5 No Exit-Code Laundering**
Pipeline scripts MUST use `set -o pipefail` and `PIPESTATUS[0]` to capture real exit codes. No `tee` exit-code laundering. No `&& echo PASS`.
```

**Task 1.1.5: Add Article XI — Anti-Bluff Testing Constitution (CRITICAL)**

```markdown
### Article XI — Anti-Bluff Testing Constitution (MANDATORY)

**§11.1 The Problem (Historical Mandate)**
> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used!"

This is the **worst possible outcome**: green tests + broken features. Every rule below exists to prevent this.

**§11.2 The Six Anti-Bluff Rules**

Every test, Challenge, and HelixQA bank entry MUST:

1. **Assert on a concrete end-user-visible outcome** — translated text content, DB row, downloadable file, visible dashboard element, API response body. NOT just "no error" or "200 OK" or "exit code 0"

2. **Run against the real system** — mocks ONLY in unit tests (`go test -short`). All integration/E2E/HelixQA tests use real services, real databases, real LLM providers (or documented `SKIP-OK: #<ticket>`)

3. **Include a matching negative assertion** — test MUST fail when the feature is broken. E.g., if testing "English→Serbian translation", also assert the output is NOT in English

4. **Emit copy-pasteable evidence** — response body snippet, screenshot filename, DB row dump, log excerpt, video timestamp

5. **Verify "fails when feature is removed"** — deliberately break the feature (comment out implementation, change API key to invalid), re-run test, test MUST FAIL

6. **No blind shells** — no `&& echo PASS`, no `|| true`, no `tee` exit-code laundering, no `test -f file && echo "PASS"` without checking file content

**§11.3 HelixQA-Specific Anti-Bluff Rules**

- Bank entries declare **executable actions** (never prose): `playwright: page.click('text=Translate')`, `api: POST /api/v1/translate`
- Each entry declares **concrete success predicates**: `assertBodyContains: 'translated text'`, `assertVisible: 'Translation Complete'`
- **Stagnation guard** — frame N+1 identical to N for >10 seconds = FAIL
- Vision-model `verified=true` with empty/tautological reasoning = `INCONCLUSIVE` (not PASS)
- `IsBlankScreenshot()` must pass before any vision analysis

**§11.4 Functional Probe Floor**
- TCP-open is the FLOOR, not the ceiling
- PostgreSQL → `SELECT 1` returns `1`
- Redis → `PING` returns `PONG`
- API → `GET /health` returns `{"status": "healthy"}` with non-empty body
- Translation → actual translated text in response, not just session_id

**§11.5 Evidence Requirements**
- Every PASS must carry positive evidence captured during execution
- No metadata-only PASS, no configuration-only PASS, no "absence-of-error" PASS
- Evidence types: API response bodies, downloaded files, database query results, browser console output, screenshots, video frames

**§11.6 Bluff Taxonomy (FORBIDDEN Patterns)**
| Bluff Type | Example | Why It's Wrong |
|-----------|---------|---------------|
| **Wrapper bluff** | Test asserts function returns, but caller ignores return value | Passes but feature unused |
| **Contract bluff** | System advertises capability but rejects it in dispatch | Advertises what it can't do |
| **Structural bluff** | Checks file exists but doesn't verify content | File exists but is empty/corrupt |
| **Comment bluff** | Code comment promises behavior code doesn't have | Tests comment, not code |
| **Skip bluff** | `t.Skip("not running yet")` without `SKIP-OK` marker | Hides broken test |

**§11.7 Mutation Testing (Mandatory)**
- Every challenge MUST have a paired mutation test
- Mutation deliberately breaks the feature → the challenge MUST then FAIL
- A challenge without a paired mutation = BLUFF challenge = Constitution violation

**§11.8 Audit Ritual**
Every Full-QA cycle (§7) MUST:
1. Pick 5 random tests + 5 random challenges
2. Comment out the target implementation
3. Re-run tests/challenges
4. Confirm they FAIL
5. Restore implementation
6. Document results in session report

**§11.9 User Mandate (NON-NEGOTIABLE)**
The bar is NOT "tests pass" but **"users can use the feature."**
A translation that "completes" but produces garbage text is a FAIL.
An API that returns 200 but with wrong content is a FAIL.
A dashboard that "loads" but shows no data is a FAIL.
```

**Task 1.1.6: Update Definition of Done**

Append to existing "## Definition of Done" section:

```markdown
5. All HelixQA banks pass for all configured platforms.
6. Anti-bluff audit (§11.8) passes — random tests confirmed to fail when feature removed.
7. Every modified/added feature has a registered challenge.
8. Evidence from the most recent QA session is archived in `docs/reports/qa-sessions/`.
```

### 1.2 Update CLAUDE.md

**File**: `HelixTranslate/CLAUDE.md`  
**Current**: 182 lines  
**Action**: Add HelixQA section and anti-bluff references

**Task 1.2.1: Add HelixQA Testing Section**

Insert before "## Conventions" section:

```markdown
## HelixQA: Autonomous LLM-Driven Testing

HelixQA is the **sole authorized tool** for all automated UI/UX and API testing. Pipeline: **Learn → Plan → Execute → Curiosity → Analyze**.

### Invariants:
- **HelixQA-only for Web Dashboard and API testing.** No custom Playwright scripts, no curl-based harnesses outside HelixQA banks.
- **Vision-driven only.** screenshot → LLM analysis → action decision. No hardcoded selectors, no sleep timers.
- **Universal Solution Principle.** Fix bugs in HelixQA itself, never in HelixTranslate to make it "testable."
- **Live log monitoring.** Every session streams API logs, gRPC logs, translation logs.
- **Screen-state tracking.** Frame N vs N+1. Stagnation >10s = critical failure.
- **Executable actions in banks**, never prose.
- **Video mandatory for Web Dashboard sessions.** Screenshots at every step.
- **Evidence validation.** Post-translation must contain actual translated text, not placeholder.
- **Validation tests are permanent.** Every fix adds to `tests/banks/fixes-validation.yaml`.

### Vision Architecture
Phase-specific model selection via LLMsVerifier strategies:
- `PlanningStrategy` (Learn/Plan): Reasoning-focused chat models
- `NavigationStrategy` (Execute/Curiosity): JSON-compliant vision models
- `AnalysisStrategy` (Analyze): Rich-description vision models

### Bank Coverage & Execution
Banks: `tests/banks/full-qa-{api,web,cli}.yaml` + `tests/banks/fixes-validation.yaml`

```bash
# Standard QA run
helixqa run --banks tests/banks/ --platform all

# List tests
helixqa list --banks tests/banks/ --platform web

# Autonomous QA
helixqa autonomous --project . --platforms web,api,cli --timeout 2h

# Generate report
helixqa report --input qa-results --format html
```

### Platform Configuration for HelixTranslate
- **Web**: `HELIX_WEB_URL=https://localhost:8443` (dashboard + API)
- **API**: `HELIX_INFRA_API_SERVICE=translator-api`, `HELIX_INFRA_API_PORT=8443`
- **CLI**: Tests invoke `./build/unified-translator` directly
```

**Task 1.2.2: Add Anti-Bluff Section**

```markdown
## Anti-Bluff Testing — Article XI

Tests MUST assert concrete end-user-visible outcomes. No blind shells. Every test MUST fail when the feature it tests is removed.

**Translation-Specific Anti-Bluff Rules:**
- A "successful" translation MUST produce verifiable translated text in the target language
- Translation response MUST contain actual content, not just a session_id or status
- Downloaded translated file MUST be a valid ebook in the target format
- Dashboard MUST show actual progress data, not just a loading spinner
- LLM provider tests MUST verify actual API calls, not just mock responses

**Audit Ritual:** Every QA cycle picks 5 tests + 5 challenges at random, comments out target, re-runs, confirms FAIL.
```

### 1.3 Update AGENTS.md

**File**: `HelixTranslate/AGENTS.md`  
**Current**: 511 lines  
**Action**: Add HelixQA constraints and anti-bluff mandate

**Task 1.3.1: Add to "Critical Non-Negotiable Constraints"**

```markdown
- **HELIXQA ONLY**: All automated UI/UX testing must use HelixQA. No custom browser scripts.
- **ANTI-BLUFF MANDATE**: Every test must verify real user-visible behavior. See Article XI.
- **FULL-QA MASTER CYCLE**: Every change requires the full Article VII cycle.
- **4-ARTEFACT FIX**: Every defect fix must produce: unit test + integration test + bank entry + challenge.
- **EVIDENCE OR BUST**: Passing tests without captured evidence = invalid. Screenshot/video/log required.
```

**Task 1.3.2: Add HelixQA Commands Section**

```markdown
## HelixQA Commands

```bash
# Build HelixQA binary
cd HelixQA && make build

# Standard QA
helixqa run --banks tests/banks/ --platform all --speed normal

# List available tests
helixqa list --banks tests/banks/ --platform web --json

# Autonomous QA session
helixqa autonomous --project . --platforms web,api,cli \
  --timeout 2h --output qa-results/ --verbose

# Generate report
helixqa report --input qa-results/ --format html --output qa-report.html
```

## QA Bank Format (YAML)

```yaml
version: "1.0"
name: "HelixTranslate Full QA - API"
test_cases:
  - id: HTQ-API-001
    name: "Health check returns healthy status"
    category: functional
    priority: critical
    platforms: [api]
    steps:
      - name: "Send GET /health"
        action: "GET https://localhost:8443/health"
        expected: "HTTP 200 with {\"status\":\"healthy\"}"
    tags: [health, smoke]
    expected_result: "Health endpoint returns valid JSON with healthy status"
```
```

### 1.4 Create HelixQA Submodule Constitution Files

**Task 1.4.1: Verify HelixQA has its own CONSTITUTION.md**
- HelixQA already has `CONSTITUTION.md` with anti-bluff rules, no CI/CD, SSH-only git
- Verify it contains equivalent anti-bluff provisions

**Task 1.4.2: Verify all sibling submodules have governance**
For each of `DocProcessor`, `LLMOrchestrator`, `LLMProvider`, `VisionEngine`, `Security`, `LLMsVerifier`:
- Check for `CLAUDE.md`, `AGENTS.md`, `CONSTITUTION.md`
- If missing, create minimal governance files referencing parent project

---

## Phase 2: Submodule Integration

### Objective
Add HelixQA and all required dependency submodules to HelixTranslate, wire them into `go.mod`, and verify compilation.

### 2.1 Add Git Submodules

**Task 2.1.1: Add HelixQA and companion modules to `.gitmodules`**

Edit `HelixTranslate/.gitmodules` — add these entries:

```ini
[submodule "HelixQA"]
	path = HelixQA
	url = git@github.com:HelixDevelopment/HelixQA.git

[submodule "DocProcessor"]
	path = DocProcessor
	url = git@github.com:HelixDevelopment/DocProcessor.git

[submodule "LLMOrchestrator"]
	path = LLMOrchestrator
	url = git@github.com:HelixDevelopment/LLMOrchestrator.git

[submodule "LLMProvider"]
	path = LLMProvider
	url = git@github.com:HelixDevelopment/LLMProvider.git

[submodule "VisionEngine"]
	path = VisionEngine
	url = git@github.com:HelixDevelopment/VisionEngine.git

[submodule "LLMsVerifier"]
	path = LLMsVerifier
	url = git@github.com:HelixDevelopment/LLMsVerifier.git
```

**Note:** `Security` and `Challenges` and `Containers` already exist as submodules.

**Task 2.1.2: Initialize new submodules**
```bash
cd HelixTranslate/
git submodule add git@github.com:HelixDevelopment/HelixQA.git HelixQA
git submodule add git@github.com:HelixDevelopment/DocProcessor.git DocProcessor
git submodule add git@github.com:HelixDevelopment/LLMOrchestrator.git LLMOrchestrator
git submodule add git@github.com:HelixDevelopment/LLMProvider.git LLMProvider
git submodule add git@github.com:HelixDevelopment/VisionEngine.git VisionEngine
git submodule add git@github.com:HelixDevelopment/LLMsVerifier.git LLMsVerifier
git submodule update --init --recursive
```

### 2.2 Wire go.mod Replace Directives

**Task 2.2.1: Add replace directives to `go.mod`**

Edit `HelixTranslate/go.mod` — add to existing `replace` block:

```go
replace (
    // Existing (KEEP):
    digital.vasic.challenges => ./Challenges
    digital.vasic.containers => ./Containers
    
    // New — HelixQA ecosystem:
    digital.vasic.helixqa => ./HelixQA
    digital.vasic.docprocessor => ./DocProcessor
    digital.vasic.llmorchestrator => ./LLMOrchestrator
    digital.vasic.llmprovider => ./LLMProvider
    digital.vasic.security => ./Security
    digital.vasic.visionengine => ./VisionEngine
    digital.vasic.llmsverifier => ./LLMsVerifier
)
```

**Task 2.2.2: Resolve go.mod dependencies**
```bash
cd HelixTranslate/
go mod tidy
# This will pull in all transitive dependencies from HelixQA and its siblings
# If there are version conflicts, resolve by updating the minimum required version
```

**Task 2.2.3: Verify compilation**
```bash
go build ./...  # Must compile without errors
go vet ./...    # Must pass static analysis
```

### 2.3 Build HelixQA Binary

**Task 2.3.1: Build HelixQA CLI**
```bash
cd HelixQA/
make build  # Produces bin/helixqa
./bin/helixqa version  # Must output: HelixQA v0.2.0
```

**Task 2.3.2: Run HelixQA tests**
```bash
cd HelixQA/
make test  # Must pass all 235 tests
make test-race  # Must pass with race detection
```

### 2.4 Create HelixQA Configuration for HelixTranslate

**Task 2.4.1: Create `.env.example` at HelixTranslate root**

Based on HelixQA's `.env.example` pattern, create a HelixTranslate-specific configuration:

```bash
# === HELIXQA MASTER SWITCH ===
HELIX_AUTONOMOUS_ENABLED=true
HELIX_AUTONOMOUS_PLATFORMS=web,api,cli
HELIX_AUTONOMOUS_TIMEOUT=2h
HELIX_AUTONOMOUS_COVERAGE_TARGET=0.95
HELIX_AUTONOMOUS_CURIOSITY_ENABLED=true
HELIX_AUTONOMOUS_CURIOSITY_TIMEOUT=30m

# === LLMsVERIFIER ===
LLMSVERIFIER_STRATEGY=balanced
LLMSVERIFIER_MIN_SCORE=0.6
LLMSVERIFIER_MAX_MODELS=5
LLMSVERIFIER_CACHE_RESULTS=true

# === VISION PROVIDERS (for autonomous QA vision testing) ===
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
GEMINI_API_KEY=
DEEPSEEK_API_KEY=

# === LOCAL VISION ===
HELIX_OLLAMA_URL=http://localhost:11434
HELIX_OLLAMA_MODEL=llava

# === VISION ENGINE ===
HELIX_VISION_PROVIDER=auto
HELIX_VISION_OPENCV_ENABLED=false
HELIX_VISION_SSIM_THRESHOLD=0.95

# === DOC PROCESSOR ===
HELIX_DOCS_ROOT=./docs
HELIX_DOCS_AUTO_DISCOVER=true
HELIX_DOCS_FORMATS=md,yaml,html

# === RECORDING ===
HELIX_RECORDING_VIDEO=true
HELIX_RECORDING_SCREENSHOTS=true
HELIX_RECORDING_VIDEO_QUALITY=high
HELIX_FFMPEG_PATH=ffmpeg

# === PLATFORM: WEB ===
HELIX_WEB_URL=https://localhost:8443
HELIX_WEB_BROWSER=chromium

# === PLATFORM: API ===
HELIX_INFRA_API_SERVICE=translator-api
HELIX_INFRA_API_PORT=8443
HELIX_INFRA_API_HEALTH_PATH=/health

# === PLATFORM: CLI ===
# CLI testing uses ./build/unified-translator directly

# === OUTPUT ===
HELIX_OUTPUT_DIR=./qa-results
HELIX_REPORT_FORMATS=markdown,html,json
HELIX_TICKETS_ENABLED=true
HELIX_TICKETS_MIN_SEVERITY=medium

# === TRANSLATION PROVIDERS (for testing) ===
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
DEEPSEEK_API_KEY=
ZHIPU_API_KEY=
QWEN_API_KEY=
GEMINI_API_KEY=
```

**Task 2.4.2: Add `.env` to `.gitignore`**
```bash
echo ".env" >> .gitignore
echo "qa-results/" >> .gitignore
```

### 2.5 Create Orchestrator Script

**Task 2.5.1: Create `scripts/run-helixqa.sh`**

Modeled after Catalogizer's `helixqa-orchestrator.sh`:

```bash
#!/bin/bash
#
# run-helixqa.sh — Master orchestration script for HelixQA testing
#
# Wires everything together:
#   1. Validates environment (API, gRPC, services)
#   2. Starts backend services if needed
#   3. Runs HelixQA bank tests
#   4. Runs HelixQA autonomous testing
#   5. Monitors progress in real-time
#   6. Generates consolidated reports
#
# Usage: ./scripts/run-helixqa.sh [platforms]
#   platforms: web,api,cli,all (default: all)

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HELIXQA_BIN="${PROJECT_ROOT}/HelixQA/bin/helixqa"
QA_RESULTS_DIR="${PROJECT_ROOT}/qa-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
SESSION_DIR="${QA_RESULTS_DIR}/session-${TIMESTAMP}"
PLATFORMS="${1:-all}"

# Expand "all" to individual platforms
if [[ "$PLATFORMS" == "all" ]]; then
    PLATFORMS="web,api,cli"
fi

echo "=== HelixQA Session ${TIMESTAMP} ==="
echo "Platforms: ${PLATFORMS}"
echo "Output: ${SESSION_DIR}"
echo

# Phase 1: Environment Validation
echo "--- Phase 1: Environment Validation ---"
if [[ ! -f "$HELIXQA_BIN" ]]; then
    echo "Building HelixQA..."
    (cd "${PROJECT_ROOT}/HelixQA" && make build)
fi

echo "Checking API server..."
if ! curl -sk --max-time 5 "https://localhost:8443/health" > /dev/null 2>&1; then
    echo "WARNING: API server not running. Starting..."
    (cd "${PROJECT_ROOT}" && make dev > /dev/null 2>&1 &)
    sleep 10
    if ! curl -sk --max-time 5 "https://localhost:8443/health" > /dev/null 2>&1; then
        echo "FATAL: Cannot start API server. Aborting."
        exit 1
    fi
fi
echo "API server: OK"

echo "Checking gRPC server..."
if ! nc -z localhost 50051 2>/dev/null; then
    echo "WARNING: gRPC server not running."
fi

# Phase 2: Run HelixQA Bank Tests
echo
echo "--- Phase 2: Bank Tests ---"
mkdir -p "${SESSION_DIR}"
if [[ -d "${PROJECT_ROOT}/tests/banks" ]]; then
    "${HELIXQA_BIN}" run \
        --banks "${PROJECT_ROOT}/tests/banks/" \
        --platform "${PLATFORMS}" \
        --output "${SESSION_DIR}/banks" \
        --report markdown \
        --verbose \
        2>&1 | tee "${SESSION_DIR}/bank-tests.log"
    BANK_EXIT="${PIPESTATUS[0]}"
else
    echo "No bank files found in tests/banks/. Skipping."
    BANK_EXIT=0
fi

# Phase 3: Run Autonomous QA
echo
echo "--- Phase 3: Autonomous QA ---"
if [[ -f "${PROJECT_ROOT}/.env" ]]; then
    "${HELIXQA_BIN}" autonomous \
        --project "${PROJECT_ROOT}" \
        --platforms "${PLATFORMS}" \
        --env "${PROJECT_ROOT}/.env" \
        --timeout 2h \
        --output "${SESSION_DIR}/autonomous" \
        --report markdown \
        --verbose \
        2>&1 | tee "${SESSION_DIR}/autonomous.log"
    AUTO_EXIT="${PIPESTATUS[0]}"
else
    echo "No .env file found. Skipping autonomous QA."
    echo "Create .env from .env.example to enable autonomous testing."
    AUTO_EXIT=0
fi

# Phase 4: Generate Report
echo
echo "--- Phase 4: Report Generation ---"
mkdir -p "${SESSION_DIR}/report"
"${HELIXQA_BIN}" report \
    --input "${SESSION_DIR}" \
    --format html \
    --output "${SESSION_DIR}/report/qa-report.html" \
    2>&1 || true

# Summary
echo
echo "=== Session Summary ==="
echo "Timestamp: ${TIMESTAMP}"
echo "Platforms: ${PLATFORMS}"
echo "Bank Tests: $([[ $BANK_EXIT -eq 0 ]] && echo PASS || echo FAIL)"
echo "Autonomous: $([[ $AUTO_EXIT -eq 0 ]] && echo PASS || echo FAIL)"
echo "Results: ${SESSION_DIR}"
echo "Report: ${SESSION_DIR}/report/qa-report.html"

# Anti-buff guard: check for "Session failed" even on exit 0
if [[ -f "${SESSION_DIR}/autonomous/pipeline-report.json" ]]; then
    if grep -q '"status":"failed"' "${SESSION_DIR}/autonomous/pipeline-report.json"; then
        echo "WARNING: Pipeline report indicates session failure despite exit code 0!"
        exit 1
    fi
fi

exit 0
```

**Task 2.5.2: Make script executable**
```bash
chmod +x scripts/run-helixqa.sh
```

### 2.6 Update Makefile

**Task 2.6.1: Add HelixQA targets to Makefile**

Append to `HelixTranslate/Makefile`:

```makefile
# HelixQA targets
.PHONY: helixqa-build helixqa-test helixqa-run helixqa-autonomous helixqa-report

helixqa-build:
	cd HelixQA && make build

helixqa-test:
	cd HelixQA && make test

helixqa-run:
	@bash scripts/run-helixqa.sh all

helixqa-autonomous:
	@bash scripts/run-helixqa.sh autonomous

helixqa-report:
	cd HelixQA/bin && ./helixqa report --input ../qa-results --format html

helixqa-list:
	cd HelixQA/bin && ./helixqa list --banks ../tests/banks/ --platform all
```

### 2.7 Create Session Archive Structure

**Task 2.7.1: Create directory structure**
```bash
mkdir -p docs/reports/qa-sessions/
mkdir -p tests/banks/
mkdir -p qa-results/
```

**Task 2.7.2: Add to `.gitignore`**
```bash
echo "qa-results/" >> .gitignore
echo ".env" >> .gitignore
```

---

## Phase 3: Testing Strategy & Implementation

### Objective
Create comprehensive test banks, challenges, and test implementations covering 100% of HelixTranslate's functionality.

### 3.1 Create HelixQA Test Banks

**Task 3.1.1: Create `tests/banks/full-qa-api.yaml`**

```yaml
version: "1.0"
name: "HelixTranslate Full QA - API"
description: "Comprehensive API endpoint testing for HelixTranslate REST API"
metadata:
  author: "HelixDevelopment"
  app: "HelixTranslate"
  version: "2.3.0"

test_cases:
  # === HEALTH & SYSTEM ===
  - id: HTQ-API-001
    name: "Health check returns healthy status with valid JSON body"
    category: functional
    priority: critical
    platforms: [api]
    steps:
      - name: "Send GET /health"
        action: "GET https://localhost:8443/health"
        expected: "HTTP 200 with JSON body containing status field"
    tags: [health, smoke]
    expected_result: "Returns JSON with status field and non-empty body"

  - id: HTQ-API-002
    name: "API info endpoint returns system information"
    category: functional
    priority: critical
    platforms: [api]
    steps:
      - name: "Send GET /"
        action: "GET https://localhost:8443/"
        expected: "HTTP 200 with API information including version"
    tags: [info, smoke]
    expected_result: "Returns valid API info JSON with name, version fields"

  # === AUTHENTICATION ===
  - id: HTQ-API-003
    name: "JWT authentication returns valid token on successful login"
    category: security
    priority: critical
    platforms: [api]
    steps:
      - name: "Send POST /auth/login with valid credentials"
        action: "POST https://localhost:8443/auth/login {username:admin,password:test}"
        expected: "HTTP 200 with JWT token in response body"
    tags: [auth, login, security]
    expected_result: "Valid JWT token returned that can be used for authenticated requests"

  - id: HTQ-API-004
    name: "Protected endpoint rejects request without JWT token"
    category: security
    priority: critical
    platforms: [api]
    steps:
      - name: "Send GET /profile without Authorization header"
        action: "GET https://localhost:8443/profile"
        expected: "HTTP 401 or 403"
    tags: [auth, security, negative]
    expected_result: "Request rejected with authentication error"

  - id: HTQ-API-005
    name: "Protected endpoint rejects request with invalid JWT token"
    category: security
    priority: high
    platforms: [api]
    steps:
      - name: "Send GET /profile with invalid JWT token"
        action: "GET https://localhost:8443/profile Authorization: Bearer invalid.token.here"
        expected: "HTTP 401"
    tags: [auth, security, negative]
    expected_result: "Request rejected with invalid token error"

  - id: HTQ-API-006
    name: "Rate limiting activates after exceeding burst threshold"
    category: security
    priority: high
    platforms: [api]
    steps:
      - name: "Send 25 rapid requests to health endpoint"
        action: "GET https://localhost:8443/health (25 times rapidly)"
        expected: "First 20 succeed, remaining return 429 Too Many Requests"
    tags: [rate-limit, security]
    expected_result: "Rate limiting correctly blocks excess requests"

  # === TRANSLATION ===
  - id: HTQ-API-007
    name: "Text translation returns actual translated text"
    category: functional
    priority: critical
    platforms: [api]
    steps:
      - name: "Send POST /api/v1/translate with English text and target language Serbian"
        action: "POST https://localhost:8443/api/v1/translate {text:'Hello world',source_lang:'en',target_lang:'sr',provider:'openai'}"
        expected: "HTTP 200 with response containing translated text in Serbian"
    tags: [translation, openai, functional]
    expected_result: "Response contains actual Serbian text (e.g., 'Здраво свете'), not English, not empty, not just session_id"
    documentation_refs:
      - type: api_doc
        path: "api/openapi/openapi.yaml"
        section: "POST /translate"

  - id: HTQ-API-008
    name: "Text translation fails gracefully with invalid API key"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Send POST /api/v1/translate with invalid provider API key"
        action: "POST https://localhost:8443/api/v1/translate {text:'test',source_lang:'en',target_lang:'sr',provider:'openai',api_key:'invalid-key'}"
        expected: "HTTP 400 or 500 with error message about authentication"
    tags: [translation, error-handling, negative]
    expected_result: "Error response with meaningful error message, not crash"

  - id: HTQ-API-009
    name: "FB2 ebook upload and translation produces downloadable result"
    category: functional
    priority: critical
    platforms: [api]
    steps:
      - name: "Upload valid FB2 file via POST /api/v1/translate/ebook"
        action: "POST https://localhost:8443/api/v1/translate/ebook (multipart: file=test.fb2, source_lang='en', target_lang='sr')"
        expected: "HTTP 200 with session_id, progress accessible"
      - name: "Poll translation status until complete"
        action: "GET https://localhost:8443/api/v1/status/{session_id}"
        expected: "Status progresses from 'pending' to 'completed'"
      - name: "Verify completed translation is downloadable"
        action: "GET translation download URL from status response"
        expected: "Downloadable FB2 file with translated content"
    tags: [translation, fb2, ebook, e2e]
    expected_result: "Translated FB2 file downloaded with actual translated content in target language"

  - id: HTQ-API-010
    name: "Translation status returns progress percentage"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Start translation and immediately check status"
        action: "POST /translate then GET /api/v1/status/{session_id}"
        expected: "Status JSON contains progress field (0-100)"
    tags: [translation, status, progress]
    expected_result: "Progress field present with numeric value"

  - id: HTQ-API-011
    name: "Translation cancellation stops in-progress translation"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Start a long translation"
        action: "POST /translate with large text"
      - name: "Cancel the translation"
        action: "POST /api/v1/translate/cancel/{session_id}"
        expected: "HTTP 200, subsequent status shows 'cancelled'"
    tags: [translation, cancel, lifecycle]
    expected_result: "Translation stops and status reflects cancellation"

  # === PROVIDERS ===
  - id: HTQ-API-012
    name: "List all available translation providers"
    category: functional
    priority: critical
    platforms: [api]
    steps:
      - name: "Send GET /api/v1/providers"
        action: "GET https://localhost:8443/api/v1/providers"
        expected: "HTTP 200 with JSON listing all 8 providers: openai, anthropic, zhipu, deepseek, qwen, gemini, ollama, llamacpp"
    tags: [providers, smoke]
    expected_result: "Response lists at least 8 provider names with their capabilities"

  - id: HTQ-API-013
    name: "Translation with DeepSeek provider returns valid result"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Translate text using DeepSeek"
        action: "POST /api/v1/translate {text:'Hello',provider:'deepseek',target_lang:'sr'}"
        expected: "HTTP 200 with translated text"
    tags: [translation, deepseek, provider]
    expected_result: "DeepSeek provider returns actual translated content"

  - id: HTQ-API-014
    name: "Translation with Anthropic provider returns valid result"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Translate text using Anthropic Claude"
        action: "POST /api/v1/translate {text:'Hello',provider:'anthropic',target_lang:'sr'}"
        expected: "HTTP 200 with translated text"
    tags: [translation, anthropic, provider]
    expected_result: "Anthropic provider returns actual translated content"

  # === LANGUAGES ===
  - id: HTQ-API-015
    name: "List supported languages returns comprehensive list"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Send GET /api/v1/languages"
        action: "GET https://localhost:8443/api/v1/languages"
        expected: "HTTP 200 with JSON array of supported language codes"
    tags: [languages, smoke]
    expected_result: "At least 10 language codes including en, sr, ru, de, fr, es, zh, ja"

  # === BATCH ===
  - id: HTQ-API-016
    name: "Batch translation processes multiple texts"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Send POST /api/v1/translate/batch with array of texts"
        action: "POST /api/v1/translate/batch {texts:['Hello','Goodbye'],target_lang:'sr'}"
        expected: "HTTP 200 with array of translations"
    tags: [batch, translation]
    expected_result: "Each input text has corresponding translated output"

  # === PREPARATION ===
  - id: HTQ-API-017
    name: "Preparation analysis returns content analysis"
    category: functional
    priority: medium
    platforms: [api]
    steps:
      - name: "Send POST /api/v1/preparation/analyze with text"
        action: "POST /api/v1/preparation/analyze {text:'Sample text for analysis'}"
        expected: "HTTP 200 with analysis including content_type, terminology"
    tags: [preparation, analysis]
    expected_result: "Analysis response contains content_type field with non-empty value"

  # === SCRIPT CONVERSION ===
  - id: HTQ-API-018
    name: "Serbian Cyrillic to Latin script conversion"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Send POST /api/v1/convert/script with Cyrillic text"
        action: "POST /api/v1/convert/script {text:'Здраво',script:'latin'}"
        expected: "HTTP 200 with Latin script output"
    tags: [script, serbian, conversion]
    expected_result: "Cyrillic 'Здраво' converts to Latin 'Zdravo'"

  # === DISTRIBUTED ===
  - id: HTQ-API-019
    name: "Distributed system status returns worker information"
    category: functional
    priority: medium
    platforms: [api]
    steps:
      - name: "Send GET /distributed/status"
        action: "GET https://localhost:8443/distributed/status"
        expected: "HTTP 200 with worker status information"
    tags: [distributed, workers]
    expected_result: "Status response shows distributed system configuration"

  # === MONITORING ===
  - id: HTQ-API-020
    name: "WebSocket connection receives real-time events"
    category: functional
    priority: critical
    platforms: [api]
    steps:
      - name: "Connect to WebSocket /ws"
        action: "WS connect wss://localhost:8443/ws"
        expected: "Connection established, events received on translation activity"
    tags: [websocket, monitoring, realtime]
    expected_result: "WebSocket receives JSON events with type field when translation starts"

  # === NEGATIVE TESTS ===
  - id: HTQ-API-021
    name: "Empty text body returns validation error"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Send POST /translate with empty text"
        action: "POST /api/v1/translate {text:'',target_lang:'sr'}"
        expected: "HTTP 400 with validation error"
    tags: [validation, negative]
    expected_result: "Error response explaining text is required"

  - id: HTQ-API-022
    name: "Unsupported file format returns error"
    category: functional
    priority: high
    platforms: [api]
    steps:
      - name: "Upload unsupported file type"
        action: "POST /api/v1/translate/ebook (multipart: file=test.xyz)"
        expected: "HTTP 400 with format not supported error"
    tags: [validation, negative, format]
    expected_result: "Error message lists supported formats"

  - id: HTQ-API-023
    name: "Oversized file upload rejected"
    category: security
    priority: high
    platforms: [api]
    steps:
      - name: "Upload file exceeding max_upload_size (100MB)"
        action: "POST /api/v1/translate/ebook (multipart: file=large.bin, size=200MB)"
        expected: "HTTP 413 Payload Too Large"
    tags: [security, upload, negative]
    expected_result: "Request rejected before processing"

  - id: HTQ-API-024
    name: "SQL injection in parameters returns error not crash"
    category: security
    priority: critical
    platforms: [api]
    steps:
      - name: "Send request with SQL injection in text parameter"
        action: "POST /api/v1/translate {text:\"'; DROP TABLE translations;--\",target_lang:'sr'}"
        expected: "HTTP 200 with translation or HTTP 400 with error, NOT HTTP 500"
    tags: [security, sql-injection, negative]
    expected_result: "System handles injection gracefully without data corruption"

  - id: HTQ-API-025
    name: "XSS payload in text is sanitized in response"
    category: security
    priority: critical
    platforms: [api]
    steps:
      - name: "Send text containing XSS payload"
        action: "POST /api/v1/translate {text:'<script>alert(1)</script>Hello',target_lang:'sr'}"
        expected: "HTTP 200 with sanitized text in response"
    tags: [security, xss, negative]
    expected_result: "Response does not contain executable script tags"
```

**Task 3.1.2: Create `tests/banks/full-qa-web.yaml`**

```yaml
version: "1.0"
name: "HelixTranslate Full QA - Web Dashboard"
description: "Comprehensive Web Dashboard testing via HelixQA vision-driven navigation"
metadata:
  author: "HelixDevelopment"
  app: "HelixTranslate"
  version: "2.3.0"

test_cases:
  # === DASHBOARD LOAD ===
  - id: HTQ-WEB-001
    name: "Dashboard loads and renders main interface"
    category: functional
    priority: critical
    platforms: [web]
    steps:
      - name: "Navigate to dashboard"
        action: "playwright: page.goto('https://localhost:8443/monitor')"
        expected: "Dashboard page loads with translation interface visible"
    tags: [dashboard, smoke, load]
    expected_result: "Dashboard renders with visible translation controls, not blank page, not error page"

  - id: HTQ-WEB-002
    name: "Dashboard shows real-time monitoring data"
    category: functional
    priority: critical
    platforms: [web]
    steps:
      - name: "Navigate to enhanced dashboard"
        action: "playwright: page.goto('https://localhost:8090/monitor')"
        expected: "Enhanced monitor shows progress visualization"
      - name: "Verify Chart.js elements are visible"
        action: "playwright: page.waitForSelector('canvas')"
        expected: "At least one canvas element (Chart.js) visible"
    tags: [dashboard, monitoring, charts]
    expected_result: "Dashboard shows actual Chart.js visualizations, not placeholder elements"

  # === TRANSLATION FLOW ===
  - id: HTQ-WEB-003
    name: "Translation form accepts input and submits"
    category: functional
    priority: critical
    platforms: [web]
    steps:
      - name: "Navigate to dashboard"
        action: "playwright: page.goto('https://localhost:8443/monitor')"
      - name: "Enter text in translation input field"
        action: "playwright: page.fill('textarea, input[type=text]', 'Hello world')"
        expected: "Input field populated with 'Hello world'"
      - name: "Select target language"
        action: "playwright: page.selectOption('select', 'sr')"
        expected: "Serbian language selected"
    tags: [translation, form, input]
    expected_result: "Form fields populated, translation can be initiated"

  # === RESPONSIVE ===
  - id: HTQ-WEB-004
    name: "Dashboard is responsive on mobile viewport"
    category: accessibility
    priority: high
    platforms: [web]
    steps:
      - name: "Set viewport to mobile (375px width)"
        action: "playwright: page.setViewportSize({width:375,height:812})"
      - name: "Navigate to dashboard"
        action: "playwright: page.goto('https://localhost:8443/monitor')"
        expected: "Dashboard renders without horizontal overflow"
    tags: [responsive, mobile, accessibility]
    expected_result: "No horizontal scroll, all controls accessible on mobile viewport"

  # === ERROR STATES ===
  - id: HTQ-WEB-005
    name: "Dashboard handles API disconnection gracefully"
    category: functional
    priority: high
    platforms: [web]
    steps:
      - name: "Navigate to dashboard"
        action: "playwright: page.goto('https://localhost:8090/monitor')"
      - name: "Verify error handling for disconnected state"
        action: "playwright: page.waitForTimeout(2000)"
        expected: "Dashboard shows connection status, handles errors without crash"
    tags: [error-handling, resilience]
    expected_result: "Dashboard shows meaningful status, not blank or crashed"

  # === KEYBOARD NAVIGATION ===
  - id: HTQ-WEB-006
    name: "All interactive elements are keyboard accessible"
    category: accessibility
    priority: high
    platforms: [web]
    steps:
      - name: "Navigate to dashboard"
        action: "playwright: page.goto('https://localhost:8443/monitor')"
      - name: "Tab through all interactive elements"
        action: "playwright: page.keyboard.press('Tab') (repeat 10 times)"
        expected: "Focus moves through all buttons, inputs, selects in logical order"
    tags: [accessibility, keyboard, a11y]
    expected_result: "All interactive elements reachable via Tab, visible focus indicator"
```

**Task 3.1.3: Create `tests/banks/full-qa-cli.yaml`**

```yaml
version: "1.0"
name: "HelixTranslate Full QA - CLI"
description: "Comprehensive CLI binary testing for all 11 command-line tools"
metadata:
  author: "HelixDevelopment"
  app: "HelixTranslate"
  version: "2.3.0"

test_cases:
  # === UNIFIED TRANSLATOR (PRIMARY) ===
  - id: HTQ-CLI-001
    name: "Unified translator --help displays all flags"
    category: functional
    priority: critical
    platforms: [cli]
    steps:
      - name: "Run unified-translator --help"
        action: "shell: ./build/unified-translator --help"
        expected: "Help output shows all flags: -input, -output, -source-lang, -target-lang, -provider, etc."
    tags: [cli, help, smoke]
    expected_result: "Help text contains at least: -input, -output, -provider, -source-lang, -target-lang flags"

  - id: HTQ-CLI-002
    name: "Unified translator translates text file end-to-end"
    category: functional
    priority: critical
    platforms: [cli]
    steps:
      - name: "Create test input file"
        action: "shell: echo 'Hello world' > /tmp/test-input.txt"
      - name: "Run translation"
        action: "shell: ./build/unified-translator -i /tmp/test-input.txt -o /tmp/test-output.txt -source-lang en -target-lang sr -provider openai"
        expected: "Exit code 0, output file created"
      - name: "Verify output contains translated text"
        action: "shell: cat /tmp/test-output.txt | grep -v '^$'"
        expected: "Output file contains non-empty text in Serbian (Cyrillic characters)"
    tags: [cli, translation, e2e, openai]
    expected_result: "Output file exists, is non-empty, contains Cyrillic characters (Serbian translation)"

  - id: HTQ-CLI-003
    name: "Unified translator rejects invalid provider"
    category: functional
    priority: high
    platforms: [cli]
    steps:
      - name: "Run with non-existent provider"
        action: "shell: ./build/unified-translator -i /tmp/test.txt -o /tmp/out.txt -provider nonexistent"
        expected: "Exit code non-zero, error message about invalid provider"
    tags: [cli, validation, negative]
    expected_result: "Error message lists valid providers, exit code != 0"

  - id: HTQ-CLI-004
    name: "Unified translator shows progress when translating"
    category: functional
    priority: medium
    platforms: [cli]
    steps:
      - name: "Run translation with monitoring"
        action: "shell: ./build/unified-translator -i /tmp/test.txt -o /tmp/out.txt -provider openai -monitoring"
        expected: "Progress output shows translation progress"
    tags: [cli, progress, monitoring]
    expected_result: "Stdout contains progress indicators (percentage or step info)"

  # === ALL CLI BINARIES SMOKE TESTS ===
  - id: HTQ-CLI-005
    name: "gRPC server binary starts and binds to port 50051"
    category: functional
    priority: critical
    platforms: [cli]
    steps:
      - name: "Start gRPC server in background"
        action: "shell: ./build/grpc-server & sleep 2"
      - name: "Check port 50051 is listening"
        action: "shell: nc -z localhost 50051 && echo OK || echo FAIL"
        expected: "Port 50051 is open"
      - name: "Cleanup"
        action: "shell: kill %1 2>/dev/null; wait 2>/dev/null"
    tags: [cli, grpc, smoke]
    expected_result: "gRPC server binds to port 50051 successfully"

  - id: HTQ-CLI-006
    name: "API server binary starts and binds to port 8080"
    category: functional
    priority: critical
    platforms: [cli]
    steps:
      - name: "Start API server in background"
        action: "shell: ./build/api-server & sleep 2"
      - name: "Check port 8080 is listening"
        action: "shell: nc -z localhost 8080 && echo OK || echo FAIL"
        expected: "Port 8080 is open"
      - name: "Cleanup"
        action: "shell: kill %1 2>/dev/null; wait 2>/dev/null"
    tags: [cli, api, smoke]
    expected_result: "API server binds to port 8080 successfully"

  - id: HTQ-CLI-007
    name: "Monitor server binary starts WebSocket on port 8090"
    category: functional
    priority: critical
    platforms: [cli]
    steps:
      - name: "Start monitor server in background"
        action: "shell: ./build/monitor-server & sleep 2"
      - name: "Check port 8090 is listening"
        action: "shell: nc -z localhost 8090 && echo OK || echo FAIL"
        expected: "Port 8090 is open"
      - name: "Cleanup"
        action: "shell: kill %1 2>/dev/null; wait 2>/dev/null"
    tags: [cli, monitor, websocket, smoke]
    expected_result: "Monitor server binds to port 8090 successfully"

  # === FORMAT-SPECIFIC ===
  - id: HTQ-CLI-008
    name: "Ebook translator handles FB2 format end-to-end"
    category: functional
    priority: high
    platforms: [cli]
    steps:
      - name: "Run ebook translator on FB2 file"
        action: "shell: ./build/ebook-translator -i test/fixtures/sample.fb2 -o /tmp/translated.fb2 -target-lang sr"
        expected: "Exit code 0, output FB2 file created with translated content"
    tags: [cli, fb2, ebook, format]
    expected_result: "Output FB2 file is valid XML with translated text content"

  - id: HTQ-CLI-009
    name: "Markdown translator converts EPUB to Markdown"
    category: functional
    priority: medium
    platforms: [cli]
    steps:
      - name: "Run markdown translator on EPUB file"
        action: "shell: ./build/markdown-translator -i test/fixtures/sample.epub -o /tmp/output.md"
        expected: "Exit code 0, output Markdown file created"
    tags: [cli, markdown, epub, format]
    expected_result: "Output Markdown file is non-empty with chapter structure preserved"
```

**Task 3.1.4: Create `tests/banks/fixes-validation.yaml`**

```yaml
version: "1.0"
name: "HelixTranslate - Fixes Validation (Regression Tests)"
description: "Permanent regression tests for every bug fix. Each entry is added when a defect is fixed."
metadata:
  author: "HelixDevelopment"
  app: "HelixTranslate"

test_cases:
  # Template for new fix entries:
  # - id: FIX-NNN
  #   name: "Description of the original defect"
  #   category: regression
  #   priority: critical
  #   platforms: [api/web/cli]
  #   steps:
  #     - name: "Step that would have failed before fix"
  #       action: "..."
  #       expected: "Expected behavior (what was broken before)"
  #   tags: [regression, fix-NNN]
  #   expected_result: "What the correct behavior is"
  #   documentation_refs:
  #     - type: ticket
  #       path: "docs/reports/qa-sessions/<session>/tickets/TICKET-NNN.md"
```

### 3.2 Create Challenge Scripts

**Task 3.2.1: Create `challenges/scripts/translation_api_health_challenge.sh`**

```bash
#!/bin/bash
# translation_api_health_challenge.sh
# Asserts the translation API is fully operational with real responses.
# Anti-bluff: Checks actual response body content, not just HTTP status code.

set -uo pipefail

PASS_COUNT=0; FAIL_COUNT=0; FAIL_DETAILS=()

assert_pass() { echo "  PASS: $*"; PASS_COUNT=$((PASS_COUNT + 1)); }
assert_fail() { echo "  FAIL: $*"; FAIL_COUNT=$((FAIL_COUNT + 1)); FAIL_DETAILS+=("$*"); }

API_URL="${API_URL:-https://localhost:8443}"
SKIP_TLS="${SKIP_TLS:--k}"

echo "=== translation_api_health_challenge ==="

# 1/6 Health endpoint returns real JSON body
RESPONSE=$(curl -s ${SKIP_TLS} "${API_URL}/health" 2>/dev/null)
if echo "$RESPONSE" | jq -e '.status' > /dev/null 2>&1; then
    STATUS=$(echo "$RESPONSE" | jq -r '.status')
    if [[ -n "$STATUS" && "$STATUS" != "null" ]]; then
        assert_pass "Health endpoint returns status='${STATUS}' with real body"
    else
        assert_fail "Health endpoint status is null or empty"
    fi
else
    assert_fail "Health endpoint does not return valid JSON"
fi

# 2/6 Providers endpoint lists real providers
RESPONSE=$(curl -s ${SKIP_TLS} "${API_URL}/api/v1/providers" 2>/dev/null)
PROVIDER_COUNT=$(echo "$RESPONSE" | jq -r '.providers // .[] | length' 2>/dev/null || echo 0)
if [[ "$PROVIDER_COUNT" -ge 8 ]]; then
    assert_pass "Providers endpoint lists ${PROVIDER_COUNT} providers (>= 8 required)"
else
    assert_fail "Providers endpoint lists only ${PROVIDER_COUNT} providers (need >= 8)"
fi

# 3/6 Languages endpoint returns language list
RESPONSE=$(curl -s ${SKIP_TLS} "${API_URL}/api/v1/languages" 2>/dev/null)
LANG_COUNT=$(echo "$RESPONSE" | jq '. | length' 2>/dev/null || echo 0)
if [[ "$LANG_COUNT" -ge 5 ]]; then
    assert_pass "Languages endpoint returns ${LANG_COUNT} languages (>= 5 required)"
else
    assert_fail "Languages endpoint returns only ${LANG_COUNT} languages (need >= 5)"
fi

# 4/6 Version endpoint returns non-empty version
RESPONSE=$(curl -s ${SKIP_TLS} "${API_URL}/api/v1/version" 2>/dev/null)
VERSION=$(echo "$RESPONSE" | jq -r '.version // .Version // empty' 2>/dev/null)
if [[ -n "$VERSION" ]]; then
    assert_pass "Version endpoint returns '${VERSION}'"
else
    assert_fail "Version endpoint does not return version string"
fi

# 5/6 Empty translation request returns validation error
RESPONSE=$(curl -s ${SKIP_TLS} -X POST "${API_URL}/api/v1/translate" \
    -H "Content-Type: application/json" \
    -d '{"text":"","target_lang":"sr"}' 2>/dev/null)
HTTP_CODE=$(curl -s ${SKIP_TLS} -o /dev/null -w "%{http_code}" -X POST "${API_URL}/api/v1/translate" \
    -H "Content-Type: application/json" \
    -d '{"text":"","target_lang":"sr"}' 2>/dev/null)
if [[ "$HTTP_CODE" -ge 400 ]]; then
    assert_pass "Empty text returns HTTP ${HTTP_CODE} (validation error)"
else
    assert_fail "Empty text returns HTTP ${HTTP_CODE} (expected 4xx)"
fi

# 6/6 Dashboard HTML loads with content
RESPONSE=$(curl -s ${SKIP_TLS} "http://localhost:8090/monitor" 2>/dev/null)
if [[ -n "$RESPONSE" && ${#RESPONSE} -gt 100 ]]; then
    if echo "$RESPONSE" | grep -qi "monitor\|dashboard\|chart\|progress"; then
        assert_pass "Dashboard HTML loads with content (${#RESPONSE} bytes)"
    else
        assert_fail "Dashboard HTML loads but appears to be placeholder (no dashboard keywords)"
    fi
else
    assert_fail "Dashboard HTML is empty or too short (${#RESPONSE} bytes)"
fi

echo
echo "=== summary: ${PASS_COUNT} pass, ${FAIL_COUNT} fail ==="
[[ ${FAIL_COUNT} -eq 0 ]] && exit 0 || exit 1
```

**Task 3.2.2: Create `challenges/scripts/translation_e2e_challenge.sh`**

```bash
#!/bin/bash
# translation_e2e_challenge.sh
# End-to-end translation pipeline test.
# Anti-bluff: Verifies actual translated text content, not just success status.

set -uo pipefail

PASS_COUNT=0; FAIL_COUNT=0; FAIL_DETAILS=()

assert_pass() { echo "  PASS: $*"; PASS_COUNT=$((PASS_COUNT + 1)); }
assert_fail() { echo "  FAIL: $*"; FAIL_COUNT=$((FAIL_COUNT + 1)); FAIL_DETAILS+=("$*"); }

API_URL="${API_URL:-https://localhost:8443}"
SKIP_TLS="${SKIP_TLS:--k}"
API_KEY="${API_KEY:-}"

echo "=== translation_e2e_challenge ==="

AUTH_HEADER=""
if [[ -n "$API_KEY" ]]; then
    AUTH_HEADER="-H X-API-Key:${API_KEY}"
fi

# 1/5 Text translation returns actual content
INPUT_TEXT="Hello, how are you today?"
RESPONSE=$(curl -s ${SKIP_TLS} -X POST "${API_URL}/api/v1/translate" \
    -H "Content-Type: application/json" \
    ${AUTH_HEADER} \
    -d "{\"text\":\"${INPUT_TEXT}\",\"source_lang\":\"en\",\"target_lang\":\"sr\"}" 2>/dev/null)

# Check response has actual translated content (not just session_id)
BODY_LENGTH=${#RESPONSE}
TRANSLATED_TEXT=$(echo "$RESPONSE" | jq -r '.translated_text // .translation // .result // .text // empty' 2>/dev/null)

if [[ -n "$TRANSLATED_TEXT" && ${#TRANSLATED_TEXT} -gt 3 ]]; then
    # Verify it's not the same as input (actual translation happened)
    if [[ "$TRANSLATED_TEXT" != "$INPUT_TEXT" ]]; then
        assert_pass "Translation returned '${TRANSLATED_TEXT}' (${#TRANSLATED_TEXT} chars) — different from input"
    else
        assert_fail "Translation returned same text as input (no translation occurred)"
    fi
else
    assert_fail "Translation response has no translated_text field or is empty (body: ${BODY_LENGTH} bytes)"
fi

# 2/5 Batch translation returns array of results
RESPONSE=$(curl -s ${SKIP_TLS} -X POST "${API_URL}/api/v1/translate/batch" \
    -H "Content-Type: application/json" \
    ${AUTH_HEADER} \
    -d '{"texts":["Hello","Goodbye","Thank you"],"target_lang":"sr"}' 2>/dev/null)
RESULT_COUNT=$(echo "$RESPONSE" | jq '. | length' 2>/dev/null || echo 0)
if [[ "$RESULT_COUNT" -eq 3 ]]; then
    assert_pass "Batch translation returned ${RESULT_COUNT} results (expected 3)"
else
    assert_fail "Batch translation returned ${RESULT_COUNT} results (expected 3)"
fi

# 3/5 Script conversion (Serbian Cyrillic→Latin) works
RESPONSE=$(curl -s ${SKIP_TLS} -X POST "${API_URL}/api/v1/convert/script" \
    -H "Content-Type: application/json" \
    ${AUTH_HEADER} \
    -d '{"text":"Здраво свет","script":"latin"}' 2>/dev/null)
CONVERTED=$(echo "$RESPONSE" | jq -r '.result // .text // .converted // empty' 2>/dev/null)
if [[ -n "$CONVERTED" ]] && echo "$CONVERTED" | grep -qi "zdravo"; then
    assert_pass "Script conversion: 'Здраво' → '${CONVERTED}' (contains Latin)"
else
    assert_fail "Script conversion failed or returned unexpected: '${CONVERTED}'"
fi

# 4/5 Non-existent session returns error
RESPONSE=$(curl -s ${SKIP_TLS} "http://localhost:8080/api/v1/status/nonexistent-session-id" 2>/dev/null)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/api/v1/status/nonexistent-session-id" 2>/dev/null)
if [[ "$HTTP_CODE" -ge 400 ]]; then
    assert_pass "Non-existent session returns HTTP ${HTTP_CODE}"
else
    assert_fail "Non-existent session returns HTTP ${HTTP_CODE} (expected 4xx)"
fi

# 5/5 Stats endpoint returns real statistics
RESPONSE=$(curl -s ${SKIP_TLS} "${API_URL}/api/v1/stats" 2>/dev/null)
if echo "$RESPONSE" | jq -e '.' > /dev/null 2>&1; then
    assert_pass "Stats endpoint returns valid JSON"
else
    assert_fail "Stats endpoint does not return valid JSON"
fi

echo
echo "=== summary: ${PASS_COUNT} pass, ${FAIL_COUNT} fail ==="
[[ ${FAIL_COUNT} -eq 0 ]] && exit 0 || exit 1
```

**Task 3.2.3: Create mutation tests for anti-bluff validation**

For each challenge, create a companion mutation test:

```bash
#!/bin/bash
# mutation_test_translation_api.sh
# DELIBERATELY breaks the translation feature to prove the challenge catches it.
# If this mutation test PASSES, the challenge is a BLUFF.

set -uo pipefail

echo "=== Mutation Test: Translation API ==="
echo "This test deliberately breaks the translation endpoint."

# Save original handler
ORIGINAL_HANDLER="pkg/api/handler.go"
BACKUP_HANDLER="/tmp/handler_backup_$$"

if [[ ! -f "$ORIGINAL_HANDLER" ]]; then
    echo "SKIP-OK: Handler file not found"
    exit 0
fi

cp "$ORIGINAL_HANDLER" "$BACKUP_HANDLER"

# MUTATE: Replace translate handler with a stub that always returns success
# This simulates the "bluff" scenario where the API says "success" but doesn't translate
sed -i 's/func.*TranslateHandler/\/\/ MUTATED: func DisabledTranslateHandler/' "$ORIGINAL_HANDLER"
# Add a mock handler that returns 200 with session_id but no actual translation
cat >> "$ORIGINAL_HANDLER" << 'MUTATION_EOF'
func MutatedTranslateHandler(c *gin.Context) {
    c.JSON(200, gin.H{"session_id": "mutated-fake", "status": "completed", "translated_text": ""})
}
MUTATION_EOF

echo "  Mutation applied: handler replaced with no-op stub"
echo "  Running translation challenge..."

# Rebuild
go build -o /tmp/mutated-api ./cmd/api-server 2>/dev/null
if [[ $? -ne 0 ]]; then
    echo "  SKIP: Cannot rebuild with mutation (expected if structure changed)"
    cp "$BACKUP_HANDLER" "$ORIGINAL_HANDLER"
    exit 0
fi

# Run challenge — MUST FAIL
bash challenges/scripts/translation_e2e_challenge.sh
CHALLENGE_EXIT=$?

# Restore original
cp "$BACKUP_HANDLER" "$ORIGINAL_HANDLER"
rm -f "$BACKUP_HANDLER"

if [[ $CHALLENGE_EXIT -ne 0 ]]; then
    echo "  PASS: Challenge correctly FAILED when feature was mutated (anti-bluff verified)"
    exit 0
else
    echo "  FAIL: Challenge PASSED despite mutated feature — THIS IS A BLUFF!"
    exit 1
fi
```

### 3.3 Create Test Fixtures

**Task 3.3.1: Add sample ebook files to `test/fixtures/`**

```
test/fixtures/
├── sample_en.fb2        # English FB2 ebook
├── sample_en.epub       # English EPUB ebook
├── sample_en.txt        # English plain text
├── sample_en.html       # English HTML document
├── sample_en.pdf        # English PDF document
├── sample_ru.fb2        # Russian FB2 ebook
├── sample_sr.fb2        # Serbian FB2 ebook
├── sample_long.fb2      # Large FB2 for performance tests (>1000 paragraphs)
├── special_chars.txt    # Text with special characters (Cyrillic, CJK, emoji)
├── malformed.fb2        # Malformed FB2 for error handling tests
└── empty.txt            # Empty file for edge case tests
```

### 3.4 Update Run-All-Challenges Script

**Task 3.4.1: Create `scripts/run-all-challenges.sh`**

```bash
#!/bin/bash
# run-all-challenges.sh — Run all challenges and report results
# Anti-buff: Uses PIPESTATUS to capture real exit codes

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHALLENGE_DIR="${SCRIPT_DIR}/../challenges/scripts"

TOTAL=0; PASSED=0; FAILED=0; FAILED_LIST=()

echo "=== Running All Challenges ==="
echo

for script in "${CHALLENGE_DIR}"/*.sh; do
    if [[ ! -x "$script" ]]; then
        chmod +x "$script"
    fi
    
    NAME=$(basename "$script")
    echo "--- Running: ${NAME} ---"
    
    if bash "$script" 2>&1; then
        echo "  RESULT: PASS"
        PASSED=$((PASSED + 1))
    else
        EXIT_CODE=$?
        echo "  RESULT: FAIL (exit ${EXIT_CODE})"
        FAILED=$((FAILED + 1))
        FAILED_LIST+=("$NAME")
    fi
    
    TOTAL=$((TOTAL + 1))
    echo
done

echo "=== Challenge Results ==="
echo "Total: ${TOTAL} | Passed: ${PASSED} | Failed: ${FAILED}"

if [[ ${FAILED} -gt 0 ]]; then
    echo
    echo "Failed challenges:"
    for f in "${FAILED_LIST[@]}"; do
        echo "  ✗ ${f}"
    done
    exit 1
fi

echo "All challenges passed!"
exit 0
```

---

## Phase 4: Screenshot & On-Demand Capture System

### Objective
Enable HelixQA to capture screenshots of all HelixTranslate client applications on-demand during QA sessions, with screenshots stored in the session archive for presentation purposes.

### 4.1 Web Dashboard Screenshots (via HelixQA Evidence Collector)

**How it works:**
HelixQA's `pkg/evidence/collector.go` already supports Web platform screenshots via `captureWebScreenshot()`. When HelixQA runs against the HelixTranslate dashboard, it automatically captures screenshots at each step.

**Task 4.1.1: Configure Web dashboard screenshot capture**

In the HelixQA `.env` for HelixTranslate:
```bash
HELIX_RECORDING_SCREENSHOTS=true
HELIX_WEB_URL=https://localhost:8443/monitor
HELIX_RECORDING_VIDEO=true
```

**Task 4.1.2: Add screenshot steps to Web test bank**

Each test case in `full-qa-web.yaml` should include explicit screenshot capture steps:

```yaml
  - id: HTQ-WEB-SS-001
    name: "Capture screenshot of dashboard initial state"
    category: functional
    priority: critical
    platforms: [web]
    steps:
      - name: "Navigate to dashboard"
        action: "playwright: page.goto('https://localhost:8443/monitor')"
      - name: "Capture screenshot"
        action: "evidence: screenshot('dashboard-initial-state')"
        expected: "Screenshot saved to session screenshots/ directory"
    tags: [screenshot, dashboard, capture]
    expected_result: "PNG screenshot of dashboard initial state saved to qa-results/"
```

**Task 4.1.3: Add screenshot capture steps for all major screens**

Create dedicated screenshot capture test cases for presentation purposes:

```yaml
  # === PRESENTATION SCREENSHOTS ===
  - id: HTQ-WEB-PRES-001
    name: "[PRESENTATION] Capture main dashboard screenshot"
    category: functional
    priority: medium
    platforms: [web]
    steps:
      - name: "Navigate to main dashboard"
        action: "playwright: page.goto('https://localhost:8443/monitor')"
      - name: "Wait for full load"
        action: "playwright: page.waitForLoadState('networkidle')"
      - name: "Capture full-page screenshot"
        action: "evidence: screenshot('presentation-main-dashboard', fullPage=true)"
    tags: [screenshot, presentation]
    expected_result: "High-quality full-page screenshot of main dashboard"

  - id: HTQ-WEB-PRES-002
    name: "[PRESENTATION] Capture enhanced monitor with Chart.js"
    category: functional
    priority: medium
    platforms: [web]
    steps:
      - name: "Navigate to enhanced monitor"
        action: "playwright: page.goto('https://localhost:8090/monitor')"
      - name: "Wait for charts to render"
        action: "playwright: page.waitForSelector('canvas', {timeout:10000})"
      - name: "Capture screenshot"
        action: "evidence: screenshot('presentation-enhanced-monitor', fullPage=true)"
    tags: [screenshot, presentation, charts]
    expected_result: "Screenshot showing Chart.js visualizations"

  - id: HTQ-WEB-PRES-003
    name: "[PRESENTATION] Capture translation in progress"
    category: functional
    priority: medium
    platforms: [web]
    steps:
      - name: "Start a translation via API"
        action: "shell: curl -sk -X POST https://localhost:8443/api/v1/translate -d '{...}'"
      - name: "Navigate to monitor to see progress"
        action: "playwright: page.goto('https://localhost:8090/monitor')"
      - name: "Wait for progress bar to appear"
        action: "playwright: page.waitForSelector('[data-progress]', {timeout:5000})"
      - name: "Capture screenshot of in-progress translation"
        action: "evidence: screenshot('presentation-translation-in-progress')"
    tags: [screenshot, presentation, translation]
    expected_result: "Screenshot showing active translation progress"
```

### 4.2 CLI Application Screenshots

**Challenge:** CLI tools don't have graphical interfaces.

**Solution:** HelixQA's evidence collector supports `TypeConsoleLog` evidence. For CLI tools, capture terminal output as evidence instead of screenshots.

**Task 4.2.1: Add CLI output capture to test bank**

```yaml
  - id: HTQ-CLI-PRES-001
    name: "[PRESENTATION] Capture CLI translation output"
    category: functional
    priority: medium
    platforms: [cli]
    steps:
      - name: "Run translation with verbose output"
        action: "shell: ./build/unified-translator -i test/fixtures/sample_en.txt -o /tmp/presentation-output.txt -source-lang en -target-lang sr -provider openai 2>&1 | tee /tmp/cli-output.txt"
      - name: "Capture terminal output as evidence"
        action: "evidence: consoleLog('presentation-cli-translation', '/tmp/cli-output.txt')"
    tags: [screenshot, presentation, cli]
    expected_result: "Terminal output captured showing translation progress and result"
```

**Task 4.2.2: For terminal recording, use asciinema or script**

```bash
# Install asciinema for terminal recording
apt-get install asciinema  # or brew install asciinema

# Record CLI session
asciinema rec -c "./build/unified-translator -i test.txt -o out.txt -provider openai" qa-results/presentation-cli.cast
```

### 4.3 API Response Screenshots

For API endpoints, capture actual HTTP responses as evidence:

```yaml
  - id: HTQ-API-PRES-001
    name: "[PRESENTATION] Capture API translation response"
    category: functional
    priority: medium
    platforms: [api]
    steps:
      - name: "Send translation request"
        action: "shell: curl -sk -X POST https://localhost:8443/api/v1/translate -H 'Content-Type: application/json' -d '{\"text\":\"Hello world\",\"target_lang\":\"sr\"}' | jq . > /tmp/api-response.json"
      - name: "Capture response as evidence"
        action: "evidence: consoleLog('presentation-api-response', '/tmp/api-response.json')"
    tags: [screenshot, presentation, api]
    expected_result: "JSON response captured showing actual translation result"
```

### 4.4 On-Demand Screenshot API

**Task 4.4.1: Add screenshot capture endpoint to HelixTranslate API**

This allows external tools to request screenshots of the web dashboard on-demand.

Add to `pkg/api/handler.go`:

```go
// ScreenshotHandler captures a screenshot of the web dashboard
// @Summary Capture dashboard screenshot
// @Description Captures a screenshot of the web dashboard for presentation purposes
// @Tags Presentation
// @Produce json
// @Success 200 {object} ScreenshotResponse
// @Router /api/v1/screenshot [get]
func (s *Server) ScreenshotHandler(c *gin.Context) {
    // Use chromedp to capture screenshot
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Capture screenshot of running dashboard
    screenshotPath, err := captureDashboardScreenshot(ctx, s.config.Server.Host, s.config.Server.Port)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to capture screenshot"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "screenshot_path": screenshotPath,
        "captured_at":     time.Now().Format(time.RFC3339),
    })
}
```

---

## Phase 5: UX Enterprise-Grade Considerations

### Objective
Ensure the integrated HelixQA + HelixTranslate system delivers enterprise-grade user experience for heavy translation workloads.

### 5.1 Dashboard UX Enhancements

**Task 5.1.1: Real-time QA status widget on dashboard**

Add a collapsible "QA Status" panel to the dashboard showing:
- Last QA session timestamp and result
- Coverage percentage per test category
- Open tickets count
- Link to latest QA report

**Task 5.1.2: Translation quality indicators**

Show per-translation quality metrics:
- Provider used
- Time taken
- Character count (source/target)
- Verification status (if verification enabled)

**Task 5.1.3: Multi-provider comparison**

Allow users to compare translations from multiple providers side-by-side:
- Select text
- Choose 2+ providers
- Show results in comparison view
- Highlight differences

### 5.2 Error Handling & User Feedback

**Task 5.2.1: Comprehensive error messages**

Every API error must include:
- Error code (machine-readable)
- Error message (human-readable)
- Suggested resolution
- Request ID for debugging

```json
{
  "error": {
    "code": "TRANSLATION_PROVIDER_ERROR",
    "message": "OpenAI API returned an error: rate limit exceeded",
    "resolution": "Wait 60 seconds and retry, or switch to a different provider",
    "request_id": "req_abc123"
  }
}
```

**Task 5.2.2: Progress feedback for long operations**

Large file translations must show:
- Percentage complete
- Current step (parsing/translating/verifying/generating)
- Estimated time remaining
- Cancellation option

### 5.3 Performance for Heavy Translation

**Task 5.3.1: Large file handling**

- Files > 10MB: Show warning about processing time
- Files > 100MB: Require explicit confirmation
- Streaming progress via WebSocket

**Task 5.3.2: Batch translation UX**

- Upload multiple files at once
- Show per-file progress in a table
- Allow individual file cancellation
- Download completed files while others are processing

**Task 5.3.3: Provider fallback**

When primary provider fails:
- Automatically retry with next available provider
- Notify user of provider switch
- Show which provider actually handled the translation

---

## Anti-Bluff Testing Constitution

### The Core Principle

> **"Tests pass" ≠ "Feature works"**

Every test, challenge, and HelixQA bank entry exists for ONE purpose: to guarantee that users can actually use the feature. If a test passes but the feature is broken, that test is worse than useless — it's actively harmful because it creates false confidence.

### Anti-Bluff Checklist (For Every Test)

Before marking any test as "done", verify:

| # | Check | Command |
|---|-------|---------|
| 1 | Test asserts **real user-visible outcome** | grep for actual translated text, not just status code |
| 2 | Test uses **real system** (no mocks outside unit tests) | no testify.mock in integration/E2E tests |
| 3 | Test has **matching negative assertion** | test fails when feature is broken |
| 4 | Test emits **copy-pasteable evidence** | response body, screenshot filename, log excerpt |
| 5 | Test **fails when feature is removed** | comment out implementation, re-run, confirm FAIL |
| 6 | No **blind shells** | no `&& echo PASS`, no `|| true`, no `tee` laundering |

### Mutation Testing Protocol

For every challenge in `challenges/scripts/`:

1. Create companion mutation test in `challenges/scripts/mutation_<name>.sh`
2. Mutation test deliberately breaks the feature
3. Runs the original challenge against the broken feature
4. Challenge MUST return FAIL
5. If challenge returns PASS against broken feature → **BLUFF DETECTED**

### Audit Ritual (Per QA Session)

```bash
# Pick 5 random tests
TESTS=$(go test ./... -list '.*' 2>/dev/null | shuf -n 5)

# Pick 5 random challenges
CHALLENGES=$(ls challenges/scripts/*.sh | shuf -n 5)

# For each, verify it fails when feature is removed
# Document results in session report
```

---

## Full-QA Master Cycle

### The 10-Step Mandatory Loop

```
┌────────────────────────────────────────────────────────────────┐
│                    FULL-QA MASTER CYCLE                         │
│                                                                 │
│  ① Clean rebuild (make clean && make build-all)                 │
│       │                                                         │
│  ② Unit + integration tests (make test-coverage)                │
│       │                                                         │
│  ③ Challenges bank run (./scripts/run-all-challenges.sh)        │
│       │                                                         │
│  ④ HelixQA bank tests (helixqa run --banks tests/banks/)        │
│       │                                                         │
│  ⑤ Autonomous QA (helixqa autonomous --project .)               │
│       │                                                         │
│  ⑥ Video + screenshot post-session review                       │
│       │                                                         │
│  ⑦ Ticket creation for every defect                             │
│       │                                                         │
│  ⑧ Root-cause fix + 4-artefact tail:                           │
│       │  ├─ Unit test                                           │
│       │  ├─ Integration test                                     │
│       │  ├─ fixes-validation.yaml entry                          │
│       │  └─ Challenge registration                               │
│       │                                                         │
│  ⑨ Full rebuild, re-run from ① until CLEAN PASS                │
│       │                                                         │
│  ⑩ Version bump + release artefacts                             │
│                                                                 │
│  STOP CONDITIONS:                                               │
│    ✅ CLEAN PASS → ship                                          │
│    ❌ FATAL BLOCKER → pause                                      │
│    🛑 NOTHING LEFT → stop                                        │
└────────────────────────────────────────────────────────────────┘
```

---

## Mapping: Test Types → HelixTranslate Components

| Component | Unit | Integration | E2E | Stress | Security | Challenge | HelixQA |
|-----------|:----:|:-----------:|:---:|:------:|:--------:|:---------:|:-------:|
| REST API (`pkg/api/`) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| gRPC (`pkg/grpc/`) | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| WebSocket Hub | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Translation Engine | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| LLM: OpenAI | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| LLM: Anthropic | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| LLM: DeepSeek | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| LLM: Zhipu | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| LLM: Qwen | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| LLM: Gemini | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| LLM: Ollama | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| LLM: LlamaCpp | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| Mock Provider | ✓ | — | — | — | — | — | — |
| FB2 Parser/Writer | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| EPUB Parser/Writer | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| PDF Processing | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| DOCX Processing | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| Format Detection | ✓ | ✓ | — | — | ✓ | ✓ | ✓ |
| EventBus | ✓ | ✓ | ✓ | ✓ | — | ✓ | — |
| Storage: PostgreSQL | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Storage: Redis | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Storage: SQLite | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Security: JWT | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| Security: Rate Limit | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Security: CORS | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| Distributed: Coordinator | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Distributed: SSH Pool | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Distributed: Fallback | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| Preparation | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| Verification | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| Progress Tracking | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| Batch Processing | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| Serbian Script Conv. | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| CLI: unified-translator | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| CLI: grpc-server | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| CLI: api-server | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| CLI: monitor-server | ✓ | ✓ | ✓ | — | ✓ | ✓ | ✓ |
| CLI: translate-ssh | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| CLI: preparation-translator | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| CLI: markdown-translator | ✓ | ✓ | ✓ | — | — | ✓ | ✓ |
| CLI: ebook-translator | ✓ | ✓ | ✓ | ✓ | — | ✓ | ✓ |
| Web Dashboard | — | — | — | — | — | — | ✓ |

---

## Risk Register & Mitigation

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|:----------:|:------:|------------|
| R1 | Go version incompatibility (1.25.2 vs 1.26) | Medium | High | Test compilation early; pin compatible versions |
| R2 | Import path conflicts (security, containers) | Low | High | Verify all imports; use full module paths |
| R3 | LLM API costs during autonomous QA | High | Medium | Use rate limiting; prefer local models (Ollama) for routine tests |
| R4 | Screenshot capture fails on headless CI | Medium | Medium | Use chromedp headless mode; ensure Chromium installed |
| R5 | Test flakiness from timing dependencies | High | Medium | Use explicit waits; avoid sleep-based timing |
| R6 | Large test suite execution time > 2h | Medium | Low | Parallel test execution; platform-specific subsets |
| R7 | False-positive anti-bluff failures | Medium | Medium | Clear mutation test templates; documented skip procedure |
| R8 | Submodule pointer drift between repos | Medium | Medium | Pin specific commits in .gitmodules; update deliberately |
| R9 | Docker Compose not available in sandbox | Low | Medium | Use SQLite as fallback; mock Redis in tests |
| R10 | Vision model quality insufficient | Medium | Medium | Configure multiple vision providers; fall back to OpenCV |

---

## Execution Order Summary

| Phase | Description | Estimated Effort | Dependencies |
|-------|-------------|-----------------|-------------|
| **Phase 0** | Pre-Integration Readiness Audit | 2-4 hours | None |
| **Phase 1** | Constitutional Foundation | 4-6 hours | Phase 0 |
| **Phase 2** | Submodule Integration | 4-8 hours | Phase 0 |
| **Phase 3** | Testing Strategy & Implementation | 20-40 hours | Phase 1, Phase 2 |
| **Phase 4** | Screenshot & Capture System | 4-8 hours | Phase 2, Phase 3 |
| **Phase 5** | UX Enterprise-Grade | 8-16 hours | Phase 3 |

**Total Estimated Effort: 42-82 hours**

---

*This plan is a living document. As each phase is executed, discoveries should be fed back to update subsequent phases. The Anti-Bluff Testing Constitution and Full-QA Master Cycle are NON-NEGOTIABLE and must be upheld throughout.*
