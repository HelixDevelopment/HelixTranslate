# Catalogizer Repository — HelixQA Integration Analysis

**Date**: 2025-07-10  
**Repository**: [vasic-digital/Catalogizer](https://github.com/vasic-digital/Catalogizer)  
**HelixQA**: [HelixDevelopment/HelixQA](https://github.com/HelixDevelopment/HelixQA)  
**Purpose**: Reference implementation analysis for HelixQA integration into HelixTranslate

---

## 1. Project Overview

**Catalogizer** is a comprehensive multi-platform media collection management system that:
- Automatically detects, categorizes, and organizes media files across SMB, FTP, NFS, WebDAV, and local filesystems
- Provides real-time monitoring, advanced analytics, and a modern web interface
- Supports 11 media types: movies, TV shows, seasons, episodes, music artists, albums, songs, games, software, books, comics

### Components (7 applications):
| Component | Technology | Description |
|-----------|-----------|-------------|
| `catalog-api` | Go 1.25 / Gin | REST API backend with WebSocket support |
| `catalog-web` | React 18 / TypeScript / Vite | Modern responsive web frontend |
| `catalogizer-desktop` | Tauri (Rust + React) | Cross-platform desktop application |
| `installer-wizard` | Tauri (Rust + React) | SMB configuration wizard |
| `catalogizer-android` | Kotlin / Jetpack Compose | Android mobile app |
| `catalogizer-androidtv` | Kotlin / Jetpack Compose / Leanback | Android TV app |
| `catalogizer-api-client` | TypeScript | Shared API client library |

### Version System
Versions tracked in `versions.json` (currently global v2.3.0, build #25). Every component tracks `last_build_number`, `last_build_date`, `last_source_hash`, and `last_git_commit`.

---

## 2. HelixQA Integration Points

HelixQA is integrated at **every level** of the Catalogizer project. It is not a peripheral tool — it is the **sole authorized testing framework** for all UI/UX automated testing.

### 2.1 Governance Layer
- **CONSTITUTION.md**: Contains Article V (100% test coverage), Article VII (Full-QA Master Cycle), Article IX (HelixQA tool hygiene), Article XI (Anti-Bluff Testing)
- **CLAUDE.md**: Dedicated `## HelixQA: Autonomous LLM-Driven Testing` section with invariants, vision architecture, bank coverage, and platform-specific notes
- **AGENTS.md**: Lists `HELIXQA ONLY` as a critical non-negotiable constraint; references HelixQA in the full-QA cycle

### 2.2 Submodule Integration
HelixQA is added as a **git submodule** at the project root:

```
[submodule "HelixQA"]
    path = HelixQA
    url = git@github.com:HelixDevelopment/HelixQA.git
```

It lives alongside 8 other AI/QA submodules:
- `DocProcessor/` — Documentation loading, feature map building
- `LLMOrchestrator/` — CLI agent management (opencode, claude-code, gemini, etc.)
- `LLMProvider/` — LLM API abstraction
- `VisionEngine/` — GoCV mechanical vision + LLM Vision API
- `ScreenDiff/` — Screen comparison
- `ReplayBuffer/` — Replay recording
- `VisualRegression/` — Visual regression testing
- `TrainingCollector/` — Training data collection

### 2.3 Orchestrator Script
`./scripts/helixqa-orchestrator.sh` is the entry point:

```bash
./scripts/helixqa-orchestrator.sh [platforms]   # all platforms by default
./scripts/helixqa-orchestrator.sh androidtv     # one platform
```

**Pipeline phases:**
1. Environment validation
2. Device connect (`.devconnect`)
3. APK install (builds if needed)
4. Background health monitoring
5. Autonomous testing
6. Report generation

### 2.4 Platform-Specific Scripts
- `./scripts/run-helixqa-api.sh` — API bank tests
- `./scripts/run-helixqa-web.sh` — Web bank tests
- `./scripts/run-helixqa-desktop.sh` — Desktop bank tests

### 2.5 Key Command References (from CLAUDE.md)
```bash
# HelixQA binary
helixqa run --banks tests/banks/ --platform all
helixqa list --banks tests/banks/ --platform android
helixqa report --input qa-results --format html
helixqa autonomous --project /path/to/Catalogizer --platforms android,desktop,web

# Bank format conversion (YAML → JSON)
python3 -c "import yaml,json; json.dump(yaml.safe_load(open('bank.yaml')), open('bank.json','w'))"
```

---

## 3. Submodule Configuration

### 3.1 `.gitmodules` Structure
Total: **43 git submodules** organized as:

**Go Modules (23)** — `digital.vasic.*` namespace:
| Submodule | Purpose |
|-----------|---------|
| `Auth/` | JWT authentication, bcrypt helpers |
| `Cache/` | Redis-backed caching |
| `Challenges/` | Structured test scenario framework |
| `Concurrency/` | Retry, backoff, offline cache |
| `Config/` | Configuration management |
| `Containers/` | Container orchestration |
| `Database/` | Dual SQLite/PostgreSQL |
| `Discovery/` | Network/service discovery |
| `Entities/` | Media entity system |
| `EventBus/` | Typed event channels |
| `Filesystem/` | Multi-protocol client |
| `Lazy/` | Lazy loading utilities |
| `Media/` | Media detection/analysis |
| `Memory/` | Memory management |
| `Middleware/` | HTTP middleware |
| `Observability/` | Prometheus/OpenTelemetry |
| `RateLimiter/` | Pluggable rate limiting |
| `Recovery/` | Recovery patterns |
| `Security/` | CORS, CSP, SSRF guards |
| `Storage/` | S3-compatible object storage |
| `Streaming/` | WebSocket hub |
| `Watcher/` | Filesystem watcher |

**TypeScript/React Modules (6)** — `@vasic-digital/*` namespace:
| Submodule | Purpose |
|-----------|---------|
| `WebSocket-Client-TS/` | Generic WebSocket client with React hooks |
| `UI-Components-React/` | Reusable React UI component library |
| `Media-Types-TS/` | Shared TypeScript types |
| `Catalogizer-API-Client-TS/` | API client library |
| `Auth-Context-React/` | React auth context |
| `Media-Browser-React/` | Media browser components |
| `Dashboard-Analytics-React/` | Analytics dashboard |
| `Media-Player-React/` | Media player components |
| `Collection-Manager-React/` | Collection management |

**AI/QA Modules (9)** — HelixQA ecosystem:
| Submodule | Source Org |
|-----------|-----------|
| `HelixQA/` | `HelixDevelopment/HelixQA` |
| `DocProcessor/` | `HelixDevelopment/DocProcessor` |
| `LLMOrchestrator/` | `HelixDevelopment/LLMOrchestrator` |
| `LLMProvider/` | `HelixDevelopment/LLMProvider` |
| `VisionEngine/` | `HelixDevelopment/VisionEngine` |
| `ScreenDiff/` | `vasic-digital/ScreenDiff` |
| `ReplayBuffer/` | `vasic-digital/ReplayBuffer` |
| `VisualRegression/` | `vasic-digital/VisualRegression` |
| `TrainingCollector/` | `vasic-digital/TrainingCollector` |

### 3.2 Wiring Conventions
- **Go modules**: `replace` directives in `catalog-api/go.mod`
- **TS/React modules**: `file:../` links in `catalog-web/package.json`
- **All URLs**: SSH only (`git@github.com:...`)

### 3.3 Submodule Lifecycle Commands
```bash
git submodule update --init --recursive
git submodule update --remote --recursive
./scripts/setup-submodule.sh ModuleName [--create-repos] [--go|--ts|--kotlin]
cd SubmoduleName && commit "message"    # pushes to all upstreams
```

---

## 4. Directory Structure for QA

### 4.1 QA-Related Top-Level Directories
```
Catalogizer/
├── HelixQA/                    # Git submodule — full QA framework
├── Challenges/                 # Git submodule — challenge framework
├── challenges/                 # Catalogizer-specific challenge configs
│   ├── config/                 # Bank definitions
│   │   ├── helixqa-validation.yaml
│   │   └── ...
│   └── scripts/                # Challenge scripts
│       ├── no_suspend_calls_challenge.sh
│       ├── host_no_auto_suspend_challenge.sh
│       └── ...
├── qa-results/                 # (gitignored) HelixQA output
│   └── session-<timestamp>/
├── scripts/                    # Automation scripts
│   ├── helixqa-orchestrator.sh
│   ├── run-helixqa-api.sh
│   ├── run-helixqa-web.sh
│   ├── run-helixqa-desktop.sh
│   ├── run-all-tests.sh
│   ├── devconnect.sh
│   └── ...
├── docs/
│   ├── reports/
│   │   └── qa-sessions/
│   │       └── <YYYY-MM-DD-THH-MM>/
│   │           ├── FINAL-REPORT.md
│   │           ├── logs/
│   │           ├── challenges/
│   │           ├── helixqa/
│   │           ├── videos/
│   │           ├── screenshots/
│   │           ├── tickets/
│   │           └── analysis/
│   ├── plans/
│   │   └── 2026-04-18-full-qa-cycle-master-plan.md
│   └── ...
├── tests/
│   └── k6/                     # Load testing scripts
│       ├── load_test.js
│       ├── stress_test.js
│       └── soak_test.js
├── .devignore                  # Devices to exclude from QA
├── .devconnect                 # (gitignored) Device IPs for auto-connect
├── CLAUDE.md                   # Agent guidance
├── AGENTS.md                   # Autonomous agent constraints
├── CONSTITUTION.md             # Non-negotiable rules
├── versions.json               # Component version tracking
└── .gitmodules                 # 43 submodule definitions
```

### 4.2 HelixQA Internal Structure
```
HelixQA/
├── cmd/
│   ├── helixqa/                # Main CLI (run, list, report, autonomous, version)
│   ├── helixqa-x11grab/        # X11 capture sidecar
│   ├── helixqa-capture-linux/  # Linux capture sidecar (C)
│   ├── helixqa-kmsgrab/        # KMS grab sidecar (C)
│   └── ocu-probe/              # OCU diagnostic tool
├── pkg/
│   ├── autonomous/             # SessionCoordinator, PlatformWorker, PhaseManager
│   ├── navigator/              # NavigationEngine, ActionExecutor (ADB/Playwright/X11)
│   ├── issuedetector/          # LLM-powered bug detection
│   ├── planning/               # Test plan generation
│   │   └── androidtv_channels_framework.go
│   ├── session/                # SessionRecorder, Timeline, VideoManager
│   ├── config/                 # Configuration types
│   ├── testbank/               # YAML bank management
│   ├── detector/               # Crash/ANR detection
│   ├── validator/              # Step-by-step validation
│   ├── evidence/               # Evidence collection
│   ├── ticket/                 # Ticket generation
│   ├── reporter/               # Report generation
│   ├── orchestrator/           # Pipeline coordinator
│   ├── bridge/                 # Sidecar bridging
│   ├── capture/                # Frame capture
│   ├── vision/                 # Vision pipeline
│   ├── nexus/                  # OCU automation surface
│   └── ...
├── banks/
│   ├── full-qa-api.yaml        # API test bank
│   ├── full-qa-web.yaml        # Web test bank
│   ├── full-qa-androidtv.yaml  # Android TV test bank
│   ├── full-qa-android.yaml    # Android test bank
│   ├── full-qa-cross-platform.yaml
│   ├── fixes-validation.yaml   # Regression tests
│   ├── docs-audit.yaml         # Documentation checks
│   ├── phase1-gocore.yaml      # Phase-specific bank
│   └── ocu-*.json              # OCU-specific banks
├── challenges/config/          # HelixQA's own challenges
├── docs/
│   ├── nexus/ocu-roadmap.md
│   ├── releases/
│   ├── security/
│   ├── hooks/README.md
│   ├── ocu-replay-format.md
│   ├── website/
│   │   ├── challenges-dashboard/
│   │   └── ticket-viewer/
│   └── ...
├── scripts/
│   ├── openclaw-full-campaign.sh
│   ├── hooks/no-sudo.sh
│   └── ...
├── CONSTITUTION.md
├── CLAUDE.md
├── README.md
├── ARCHITECTURE.md
├── API_REFERENCE.md
├── USER_GUIDE_AUTONOMOUS.md
├── VIDEO_COURSE_AUTONOMOUS.md
├── .env.example
└── Makefile
```

---

## 5. Challenge System

### 5.1 Framework
The `Challenges/` submodule (`digital.vasic.challenges`) provides:
- Go struct-based challenge definitions embedding `challenge.BaseChallenge`
- Custom `Execute()` methods
- Registration via `catalog-api/challenges/register.go` → `RegisterAll()`
- REST API exposure at `/api/v1/challenges`

### 5.2 User Flow Challenges (174 total)
| File | Platform | Count |
|------|----------|-------|
| `userflow_api.go` | Go API (HTTP) | 49 |
| `userflow_web.go` | React web (Playwright) | 59 |
| `userflow_desktop.go` | Tauri desktop + wizard | 28 |
| `userflow_mobile.go` | Android + Android TV | 38 |

### 5.3 Constraints
- `RunAll` is synchronous/blocking
- 5-minute stale-progress threshold kills stuck challenges
- `challenge.NewConfig()` defaults `Timeout=5min`
- `config.json` `write_timeout` must be 900 (not 30)
- **Must execute via running catalog-api binary** — no curl/scripts

### 5.4 Runner
CLI: `Challenges/cmd/userflow-runner`
Flags: `--platform`, `--report`, `--compose`, `--root`, `--timeout`, `--output`

---

## 6. HelixQA Test Bank Files

### 6.1 Bank File Format (YAML)
```yaml
version: "1.0"
name: "Catalogizer Full QA - Web"
description: "Comprehensive QA test bank..."
metadata:
  author: "vasic-digital"
  app: "Catalogizer"
  version: "2.3.0"

test_cases:
  - id: FQA-WEB-001
    name: "Login page loads correctly"
    category: functional
    priority: critical          # critical | high | medium | low
    platforms: [web]           # web | android | androidtv | desktop | api
    steps:
      - name: "Navigate to login page"
        action: "Open http://localhost:3000/login in the browser"
        expected: "Login page renders with username field, password field, and Sign In button"
    tags: [auth, login, page-load]
    estimated_duration: "10s"
    expected_result: "Login page loads with all form elements and branding visible"
```

### 6.2 Bank Coverage
Banks cover:
- **11 media types**: all CRUD operations per type
- **All screens**: login, dashboard, media browser, entity browser, collections, playlists, favorites, settings, admin
- **All auth flows**: login, logout, register, session persistence, protected routes
- **Adversarial inputs**: Cyrillic text, SQL injection, XSS payloads
- **Boundary values**: empty fields, max-length inputs, special characters
- **Responsive**: mobile (375px), tablet (768px), desktop viewports
- **Accessibility**: keyboard navigation, screen readers
- **Performance**: loading states, error boundaries, data refresh

### 6.3 Runtime Format
Banks are **JSON at runtime** — conversion from YAML:
```bash
python3 -c "import yaml,json; json.dump(yaml.safe_load(open('bank.yaml')), open('bank.json','w'))"
```

### 6.4 Bank Files in HelixQA
| Bank File | Platform | Purpose |
|-----------|----------|---------|
| `full-qa-api.yaml` | API | Backend endpoint testing |
| `full-qa-web.yaml` | Web | Frontend UI testing |
| `full-qa-androidtv.yaml` | Android TV | TV interface testing |
| `full-qa-android.yaml` | Android | Mobile app testing |
| `full-qa-cross-platform.yaml` | All | Cross-platform flows |
| `fixes-validation.yaml` | All | Regression tests for every fix |
| `docs-audit.yaml` | Internal | Documentation checks |

---

## 7. CI/CD Integration

### 7.1 No CI/CD Pipelines
**Critical**: Catalogizer explicitly **forbids** CI/CD pipelines:
- No `.github/workflows/`
- No `.gitlab-ci.yml`
- No `Jenkinsfile`
- No Git hooks
- All builds and tests run **manually or via Makefile/script targets**

### 7.2 Manual QA Cycle
The Full-QA Master Cycle is run by operators:

```
Clean rebuild → All tests → All Challenges → All HelixQA banks
→ Autonomous QA per platform → Video+screenshot review
→ Ticket every defect → Root-cause fix (4 artefacts)
→ Rebuild → Repeat until clean pass
→ Version-bump + archive to releases/
```

### 7.3 Security Scanning (manual)
```bash
./scripts/security-scan.sh    # govulncheck + npm audit + Semgrep + Snyk + Trivy + Gosec + Hadolint
./scripts/run-sonarqube-scan.sh
./scripts/security-scan.sh
```

### 7.4 Build Pipeline
```bash
scripts/release-build.sh --container --force --skip-tests    # All 7 components
scripts/run-all-tests.sh                                     # All tests + security
```

---

## 8. Configuration Files

### 8.1 HelixQA `.env.example`
Key configuration groups:
- **Master switch**: Enable/disable, platform selection, timeout, coverage target
- **LLMsVerifier**: Strategy, score thresholds, caching
- **API keys**: OpenAI, Anthropic, Google, Groq, Mistral, DeepSeek, xAI, etc. (~35 providers)
- **CLI agents**: Enabled agents, binary paths, pool size, retry config
- **Vision**: Provider selection, OpenCV toggle, SSIM threshold
- **Recording**: Video/screenshot capture, ffmpeg path, quality
- **Platforms**: Android device, web URL/browser, desktop process/display

### 8.2 Catalogizer `.env`
```env
PORT=8080
GIN_MODE=debug
DB_TYPE=sqlite
JWT_SECRET=your-dev-secret-key
ADMIN_PASSWORD=admin123
TMDB_API_KEY=your_tmdb_key
```

### 8.3 Device Configuration
- `.devignore` — Devices excluded from QA (e.g., ATMOSphere)
- `.devconnect` — (gitignored) One IP per line for Android auto-connect

---

## 9. Documentation Architecture

### 9.1 Governance Hierarchy
```
CONSTITUTION.md (non-negotiable rules)
├── Article V: 100% Test Coverage (10 categories)
├── Article VI: Open-Points Closure Brief
├── Article VII: Full-QA Master Cycle (§7.1-§7.11)
├── Article VIII: Device State Preservation
├── Article IX: HelixQA Tool Hygiene
└── Article XI: Anti-Bluff Testing (§11.1-§11.9)

CLAUDE.md (agent-ingestible rules + architecture)
├── Overview, Commands, Architecture
├── HelixQA section (invariants, vision architecture, banks)
├── Challenge System
└── Universal Mandatory Constraints

AGENTS.md (autonomous-agent constraints)
├── Critical Constraints
├── Essential Commands
├── Testing Quirks
└── Setup Requirements
```

### 9.2 Key Documentation Files
| Document | Location | Purpose |
|----------|----------|---------|
| CONSTITUTION.md | Root | Non-negotiable program rules |
| CLAUDE.md | Root | Claude Code agent guidance |
| AGENTS.md | Root | Autonomous agent constraints |
| MEMORY.md | Root | Auto-memory index for persistent state |
| ENV_VARIABLES.md | docs/ | Complete .env reference |
| OPEN_POINTS_CLOSURE.md | docs/ | Operator-action item checklist |
| SESSION_HANDOFF_*.md | docs/ | Latest session's completed work |
| Full-QA Master Plan | docs/plans/ | QA cycle execution plan |
| OCU Roadmap | HelixQA/docs/nexus/ | OpenClaw Ultimate phases P0-P7 |
| HelixQA Release Notes | HelixQA/docs/releases/ | Per-release ship notes |
| QA Sessions Archive | docs/reports/qa-sessions/ | Per-session reports |
| Build Container Dispatch | docs/ | Container build routing |
| Security Audits | HelixQA/docs/security/ | Per-phase posture notes |
| LD_PRELOAD Hook Guide | HelixQA/docs/hooks/ | Per-target shim compilation |
| OCU Replay DSL | HelixQA/docs/ | Ticket replay spec |
| Challenges Dashboard | HelixQA/docs/website/ | Static HTML dashboard |
| Ticket Viewer | HelixQA/docs/website/ | Ticket HTML renderer |

---

## 10. Screenshot/Capture/Visual Testing Integration

### 10.1 Mandatory Video Recording
**Every device/emulator session MUST record video:**
- Minimum 16 Mbps bitrate
- 1920×1080 resolution
- Frames extracted for post-analysis

### 10.2 Platform-Specific Capture

| Platform | Method |
|----------|--------|
| **Android 9 and below** | `adb shell screenrecord --bit-rate 4000000 /sdcard/qa_session.mp4` |
| **Android 10+** | Rapid `screencap` frames → assembled into video via ffmpeg |
| **Android 15 (SDK 35)** | `screenrecord` fails → screenshot-to-video fallback |
| **Web** | Playwright `--video on` or `ffmpeg x11grab` |
| **Desktop (Tauri)** | `ffmpeg x11grab` or `Xvfb` |

### 10.3 Vision Architecture
HelixQA uses **vision-driven testing exclusively**:

**Phase-specific model selection via LLMsVerifier strategies:**
- `NavigationStrategy` (Execute/Curiosity phases): JSON-action models
- `AnalysisStrategy` (Analyze phase): Rich-description models
- `PlanningStrategy` (Learn/Plan phases): Reasoning models

**Vision backends:**
1. **llama.cpp RPC** — Primary local backend
2. **Astica.AI** — Complementary cloud
3. **Gemini** — Complementary cloud
4. **OpenAI** — Complementary cloud
5. **GoCV** — Mechanical vision (contour detection, edge detection)
6. **ScreenDiff** — Screen comparison

### 10.4 Evidence Types
- Screenshots (per step)
- Video recordings (full session)
- Logcat / browser console / app logs
- Stack traces
- ANR/crash dumps
- Diff overlays
- OCR dumps
- Accessibility tree dumps
- HAR files

### 10.5 Screen-State Tracking
- Compare frame N to N+1
- Stagnation (>10s identical after action) = critical failure
- "100% pass" when app never progressed past login = fraudulent detection

---

## 11. Constitution/CLAUDE.md/AGENTS.md — Anti-Bluff Testing Mandate

### 11.1 The Problem (Historical)
> "We had been in position that all tests do execute with success and all Challenges as well, but in reality the most of the features does not work and can't be used!"

This explicit user mandate led to **Article XI — Anti-Bluff Testing** being codified in the Constitution.

### 11.2 The Contract (6 Rules)
Every test, Challenge, and HelixQA bank entry **must**:

1. **Assert on a concrete end-user-visible outcome** — rendered DOM text, DB row, playable media file, visible search results. NOT just "no error" or "200 OK"
2. **Run against the real system** — mocks only in unit tests (`go test -short`). All other tests use real containers, databases, services
3. **Include a matching negative** — assertion that fails when feature is broken
4. **Emit copy-pasteable evidence** — body, screenshot, video frame, DB row dump, log excerpt
5. **Verify "fails when feature is removed"** — comment out implementation, re-run, test MUST fail
6. **No blind shells** — no `&& echo PASS`, no `|| true`, no `tee` exit-code laundering

### 11.3 HelixQA-Specific Anti-Bluff Rules
- Bank entries declare **executable actions** (never prose): `adb_shell: input text admin`, `playwright: page.click('text=Sign In')`
- Each entry declares **concrete success predicates**: `assertVisible: 'Movies'`, `assertNotVisible: 'Sign In'`
- **Stagnation guard** — frame N+1 identical to N for >10s = FAIL
- Vision-model `verified=true` with empty/tautological reasoning = `INCONCLUSIVE` (not PASS)

### 11.4 Challenge-Specific Rules
- Must replay user journey end-to-end through actual deliverables
- Sub-1-second challenges are almost always bluff
- No curl or third-party scripts — use the actual binaries

### 11.5 PR Requirement
Every PR adding/modifying a test must include `## Anti-Bluff Verification` block:
- Command run
- Pasted output
- Proof test fails when feature is broken (second run with feature commented out)

---

## 12. Full-QA Master Cycle (Article VII)

### 12.1 The Cycle
```
Phase 1:  Governance + plan + session directory
Phase 2:  Clean rebuild (all 7 components)
Phase 3:  Unit + integration tests (every submodule)
Phase 4:  Challenges bank run (via /api/v1/challenges)
Phase 5:  HelixQA bank tests (per platform)
Phase 6:  HelixQA autonomous QA (per platform)
Phase 7:  Post-session review (video + screenshot analysis)
Phase 8:  Fix loop (4-artefact regression per ticket)
Phase 9:  Version bump + release artefacts
Phase 10: Final session report
```

### 12.2 Fix Loop (4 Artefacts)
Every ticket fix **must** include:
1. **Unit test** — regression guard
2. **Integration test** — end-to-end verification
3. **`fixes-validation.yaml` entry** — permanent bank entry
4. **Challenge registration** — platform-specific verification

### 12.3 Stop Conditions
- **Clean pass**: All tests, all challenges, all banks pass → version bump + release
- **FATAL BLOCKER**: Unresolvable infrastructure issue → pause
- **NOTHING LEFT**: No more fixes possible → stop

### 12.4 Session Archive
Mandated layout per session:
```
docs/reports/qa-sessions/<YYYY-MM-DD-THH-MM>/
├── FINAL-REPORT.md         # Aggregated results, deep analysis
├── logs/                   # Build + test logs
├── challenges/             # Challenge results
├── helixqa/                # HelixQA results
├── videos/                 # Session recordings
├── screenshots/            # Captured screenshots
├── tickets/                # Generated tickets with evidence
└── analysis/               # Post-analysis scripts + data
```

---

## 13. Lessons Learned — Patterns to Follow

### 13.1 What Works Well

1. **Constitutional Governance**: Having non-negotiable rules in CONSTITUTION.md that cascade to every submodule prevents quality erosion. The "Universal Mandatory Constraints" pattern ensures consistency.

2. **Three-Layer Documentation** (CONSTITUTION.md → CLAUDE.md → AGENTS.md): Each file serves a distinct audience (project rules → AI agent guidance → autonomous agent constraints).

3. **Vision-Driven Testing Only**: The ban on hardcoded coordinates, sleep timers, and keystroke sequences forces real testing. The stagnation guard is brilliant — it detects when tests "pass" but the app never actually progressed.

4. **4-Artefact Fix Loop**: Every bug fix must produce 4 regression artefacts (unit test + integration test + bank entry + challenge). This prevents regressions permanently.

5. **Anti-Bluff Testing (Article XI)**: The most important lesson. Testing that passes when features are broken is worse than no testing. The mandatory "verify fails when feature is removed" check is the key insight.

6. **Device State Preservation**: QA sessions must not leave devices in a modified state. `device_preserve.go` snapshots and restores all settings.

7. **Session Archiving**: Every QA session gets a permanent, structured archive with videos, screenshots, tickets, and analysis. This provides an audit trail.

8. **Open Points Closure Brief**: A single document tracking all operator-action items with checkboxes. Prevents work from falling through cracks.

9. **Submodule Architecture**: 43 independent modules with their own tests, docs, and repos. Clean separation of concerns.

10. **No CI/CD**: Counter-intuitive but effective. Forces manual, thoughtful testing rather than automated green-washing.

### 13.2 Key Patterns for HelixTranslate Integration

| Pattern | Implementation | HelixTranslate Equivalent |
|---------|---------------|--------------------------|
| Git submodule | `HelixQA/` at root | Same — add as git submodule |
| Orchestrator script | `scripts/helixqa-orchestrator.sh` | `scripts/run-helixqa.sh` |
| Bank files | `banks/full-qa-*.yaml` per platform | `banks/full-qa-*.yaml` for translation flows |
| Governance docs | CONSTITUTION.md + CLAUDE.md + AGENTS.md | Same 3-file pattern |
| Test categories | 10 mandatory categories | Adapt for translation context |
| Fix validation | `banks/fixes-validation.yaml` | Same pattern |
| Session archives | `docs/reports/qa-sessions/` | Same structure |
| Vision architecture | LLMsVerifier strategies | Same — reuse HelixQA's vision system |
| Challenge system | Go structs + API endpoints | Adapt for translation challenges |
| Device config | `.devignore` + `.devconnect` | Web-only → simpler |

### 13.3 Simplifications for HelixTranslate

HelixTranslate is web-only (no Android/desktop/mobile), so:
- **No ADB** — remove all Android-specific code
- **No device management** — no `.devignore`/`.devconnect`
- **No container builds** — simpler if backend is TypeScript/Next.js
- **No video recording** — unless browser-based recording is needed
- **Banks focus**: Translation UI, API endpoints, i18n completeness, language switching
- **Challenge scope**: Translation quality, language detection, batch operations, file upload/download

---

## 14. Summary — Integration Checklist for HelixTranslate

Based on the Catalogizer reference implementation, integrating HelixQA into HelixTranslate requires:

### Must-Have
- [ ] Add `HelixQA` as git submodule in `.gitmodules`
- [ ] Create `CONSTITUTION.md` with Article V (test coverage), Article VII (QA cycle), Article XI (anti-bluff)
- [ ] Create `CLAUDE.md` with HelixQA section (invariants, commands, bank format)
- [ ] Create `AGENTS.md` with HelixQA constraint
- [ ] Create `banks/full-qa-web.yaml` covering all translation UI flows
- [ ] Create `banks/full-qa-api.yaml` covering all API endpoints
- [ ] Create `banks/fixes-validation.yaml` for regression tests
- [ ] Create `scripts/run-helixqa.sh` orchestrator
- [ ] Create session archive structure under `docs/reports/qa-sessions/`
- [ ] Ensure bank entries use executable actions (not prose)
- [ ] Ensure all test assertions check concrete end-user-visible outcomes

### Nice-to-Have
- [ ] Challenges framework integration (Go → adapt for TS)
- [ ] Vision-driven testing for translation UI
- [ ] Screenshot capture for visual regression
- [ ] `OPEN_POINTS_CLOSURE.md` for operator items
- [ ] `versions.json` for component tracking
- [ ] Anti-bluff verification in PR template
