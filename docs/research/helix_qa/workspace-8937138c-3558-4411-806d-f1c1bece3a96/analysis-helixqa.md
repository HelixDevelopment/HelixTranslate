# HelixQA — Comprehensive Repository Analysis

**Repository**: `https://github.com/HelixDevelopment/HelixQA`  
**Module**: `digital.vasic.helixqa`  
**Language**: Go 1.26 (module says go 1.26; CLAUDE.md says 1.24+)  
**License**: Apache-2.0  
**Version**: `0.2.0` (hardcoded in `cmd/helixqa/main.go`)  
**Author**: Milos Vasic (SPDX headers)  
**Date of Analysis**: 2025

---

## 1. PROJECT OVERVIEW

HelixQA is an **AI-driven QA orchestration framework** for cross-platform testing. It provides:

- **Real-time crash detection** (Android ADB, Web browser, Desktop process monitoring)
- **Step-by-step validation** with evidence collection to prevent false positives
- **LLM-powered autonomous QA sessions** that use computer vision to navigate applications, verify features, discover bugs, and generate comprehensive reports
- **Evidence-based reporting** in Markdown, HTML, and JSON formats
- **Markdown ticket generation** for AI fix pipelines
- **YAML test bank** support with platform targeting, priority, and documentation references

HelixQA is built on two foundational sibling modules:
- `digital.vasic.challenges` — Test execution engine (bank loading, challenge running, assertion evaluation, reporting)
- `digital.vasic.containers` — Container lifecycle, health checking, service discovery

**Key Philosophy**: 100% project-agnostic / fully decoupled. HelixQA MUST work with ANY project, never hardcoded to one specific consumer.

---

## 2. TECHNOLOGY STACK

### Core Language
- **Go 1.26** (module path: `digital.vasic.helixqa`)

### Go Dependencies (Direct)
| Dependency | Purpose |
|---|---|
| `digital.vasic.challenges` | Test execution engine, challenge definitions, report generation |
| `digital.vasic.containers` | Container lifecycle management |
| `digital.vasic.docprocessor` | Documentation loading, feature map building |
| `digital.vasic.llmorchestrator` | Headless CLI agent management |
| `digital.vasic.llmsverifier` | Strategy-based LLM selection and scoring |
| `digital.vasic.security` | Security utilities |
| `digital.vasic.visionengine` | GoCV mechanical vision + LLM Vision API analysis |
| `github.com/mattn/go-sqlite3` | SQLite for memory store |
| `github.com/stretchr/testify` | Test assertions (testify) |
| `gopkg.in/yaml.v3` | YAML parsing for test banks |

### Key Indirect Dependencies
| Dependency | Purpose |
|---|---|
| `chromedp/chromedp` | Browser automation (headless Chrome DevTools Protocol) |
| `go-rod/rod` | Browser automation alternative |
| `testcontainers-go` | Test containers for integration tests |
| `gocv.io/x/gocv` | OpenCV bindings for Go (computer vision) |
| `otiai10/gosseract/v2` | OCR (Tesseract) |
| `prometheus/client_golang` | Prometheus metrics |
| `opentelemetry/otel` | OpenTelemetry tracing |
| `nats-io/nats.go` | NATS messaging |
| `gorilla/websocket` | WebSocket support |
| `pion/webrtc/v4` | WebRTC for streaming |
| `philippgille/chromem-go` | In-memory vector store |
| `minio/minio-go/v7` | S3-compatible object storage |
| `failsafe-go/failsafe-go` | Resilience patterns (retry, circuit breaker) |

### Sibling Modules (Go `replace` directives)
```
digital.vasic.challenges  => ../Challenges
digital.vasic.containers  => ../Containers
digital.vasic.docprocessor => ../DocProcessor
digital.vasic.llmorchestrator => ../LLMOrchestrator
digital.vasic.llmprovider  => ../LLMProvider
digital.vasic.security     => ../Security
digital.vasic.visionengine => ../VisionEngine
digital.vasic.llmsverifier => ../LLMsVerifier
```

### Testing Framework
- `testify` for assertions
- Table-driven tests where appropriate
- Test naming convention: `Test<Struct>_<Method>_<Scenario>`
- `go test ./... -count=1 -race` (235 tests total)

---

## 3. DIRECTORY STRUCTURE

```
HelixQA/
├── cmd/
│   └── helixqa/
│       └── main.go                  # CLI entry point (subcommands: run, list, report, autonomous, replay, version)
├── pkg/
│   ├── config/
│   │   └── config.go                # Configuration types (Platform, SpeedMode, ReportFormat, Config, AutonomousConfig)
│   ├── testbank/
│   │   └── bank.go                  # YAML test bank management with platform/priority filtering
│   ├── detector/
│   │   ├── detector.go              # Platform-agnostic crash/ANR detection interface
│   │   ├── android.go               # ADB-based detection (pidof, logcat, screencap)
│   │   ├── web.go                   # Browser process monitoring (pgrep)
│   │   └── desktop.go               # JVM/process monitoring (pgrep, kill)
│   ├── validator/
│   │   └── validator.go             # Step-by-step validation with evidence collection
│   ├── evidence/
│   │   └── collector.go             # Centralized evidence collection (screenshots, video, logs, logcat)
│   ├── ticket/
│   │   └── ticket.go                # Markdown ticket generation for AI fix pipelines
│   ├── reporter/
│   │   └── reporter.go              # QA report generation (Markdown, HTML, JSON)
│   ├── orchestrator/
│   │   └── orchestrator.go          # Main QA brain — ties detector, validator, reporter together
│   ├── autonomous/
│   │   ├── pipeline.go              # SessionPipeline — 4-phase autonomous QA coordinator
│   │   ├── phase.go                 # PhaseManager — state machine for phase transitions
│   │   ├── session.go               # SessionCoordinator — manages workers and phases
│   │   ├── worker.go                # PlatformWorker — per-platform testing worker
│   │   ├── fallback.go              # FallbackChain — refuses scripted fallbacks when LLM fails
│   │   ├── real_executor.go         # ActionExecutor factory (ADB, Playwright, X11)
│   │   ├── screenshot.go            # IsBlankScreenshot() — validates screenshot quality
│   │   ├── geo_probe.go             # ProbeGeoRestriction() — geo-restriction detection
│   │   ├── device_preserve.go       # Device state preservation (font_scale, brightness, etc.)
│   │   └── findings_bridge.go       # FindingsBridge — deduplication and ticket persistence
│   ├── navigator/
│   │   ├── engine.go                # NavigationEngine — BFS-based UI navigation
│   │   ├── adb_executor.go          # ADBExecutor — Android via adb shell input
│   │   ├── playwright_executor.go   # PlaywrightExecutor — Web browser via CDP
│   │   ├── x11_executor.go          # X11Executor — Desktop Linux via xdotool
│   │   └── state.go                 # StateTracker — navigation history and back stack
│   ├── issuedetector/
│   │   ├── detector.go              # IssueDetector — LLM-powered bug detection
│   │   └── types.go                 # Issue types (visual, UX, accessibility, functional, performance, crash)
│   ├── llm/
│   │   ├── adaptive.go              # AdaptiveProvider — auto-selects best LLM, rate limiting, prompt optimization
│   │   ├── cost_tracker.go          # CostTracker — per-call cost accumulation
│   │   ├── vision_ranking.go        # rankVisionProviders() — dynamic provider scoring
│   │   ├── phase_selector.go        # PhaseModelSelector — phase-specific model selection
│   │   ├── bridge.go                # BridgedCLIProvider — wraps CLI tools (claude, qwen-coder, opencode)
│   │   └── providers.go             # ProviderEnvKeys — 40+ provider registry
│   ├── planning/
│   │   └── androidtv_channels_framework.go  # Android TV Channels testing framework
│   ├── session/
│   │   ├── recorder.go              # SessionRecorder — video + screenshot management
│   │   ├── timeline.go              # Timeline — event tracking and queries
│   │   └── video.go                 # VideoManager — recording lifecycle (ffmpeg, adb screenrecord)
│   ├── controller/
│   │   └── controller.go            # QA Process Controller — watchdog for stuck steps
│   ├── memory/
│   │   └── store.go                 # SQLite-based memory store for pass tracking
│   ├── infra/
│   │   └── infra.go                 # QA Infrastructure boot — backend service health checks
│   └── analysis/
│       └── types.go                 # AnalysisFinding, Evidence fields
├── tools/
│   ├── opensource/
│   │   ├── scrcpy/                  # Android screen mirroring (Genymobile/scrcpy)
│   │   ├── allure2/                 # Test reporting framework (allure-framework)
│   │   ├── leakcanary/              # Memory leak detection (square)
│   │   ├── docker-android/          # Android emulator in Docker (budtmo)
│   │   ├── appium/                  # Mobile test automation
│   │   ├── midscene/                # AI-powered UI testing (web-infra-dev)
│   │   ├── mem0/                    # AI memory layer (mem0ai)
│   │   ├── moondream/               # Vision language model (vikhyat)
│   │   ├── ui-tars/                 # UI agent (bytedance)
│   │   ├── perfetto/                # System profiling (google)
│   │   ├── chroma/                  # Vector database
│   │   ├── shortest/                # AI browser automation (antiwork)
│   │   ├── marker/                  # Document conversion (VikParuchuri)
│   │   ├── kiwi-tcms/               # Test case management
│   │   ├── testdriverai/            # AI test driver
│   │   ├── stagehand/               # Browser automation (browserbase)
│   │   ├── unstructured/            # Document processing
│   │   ├── redroid/                 # Android in container (remote-android)
│   │   ├── signoz/                  # Observability platform
│   │   ├── docling/                 # Document understanding (DS4SD)
│   │   ├── llama-index/             # LLM data framework
│   │   ├── appcrawler/              # Android app crawler (nicetester)
│   │   ├── browser-use/             # AI browser agent
│   │   ├── skyvern/                 # AI workflow automation
│   │   ├── anthropic-quickstarts/   # Anthropic examples
│   │   └── ui-tars-desktop/         # Desktop UI agent (bytedance)
│   └── test-apps/
│       └── rest-demo/               # REST demo test app
├── .env.example                     # Configuration template
├── .gitmodules                      # Git submodule definitions
├── Makefile                         # Build targets
├── go.mod                           # Go module definition
├── README.md                        # Main documentation
├── CLAUDE.md                        # Agent rules + architecture pointers
├── AGENTS.md                        # Development guide + constraints
├── CONSTITUTION.md                  # Non-negotiable project rules
├── ARCHITECTURE.md                  # Full architecture documentation with diagrams
├── API_REFERENCE.md                 # Complete API documentation for all packages
├── USER_GUIDE_AUTONOMOUS.md         # Autonomous QA session guide
└── VIDEO_COURSE_AUTONOMOUS.md       # Video tutorial for autonomous mode
```

---

## 4. SUBMODULES LIST

### 4a. Sibling Go Modules (via `go.mod` `replace` directives)

These are sibling directories in the parent repository — NOT git submodules of HelixQA, but rather co-located Go modules:

| Module | Path | Purpose |
|---|---|---|
| `digital.vasic.challenges` | `../Challenges` | Test execution engine with bank loading, challenge running, assertion evaluation, and report generation |
| `digital.vasic.containers` | `../Containers` | Container lifecycle, health checking, and service discovery |
| `digital.vasic.docprocessor` | `../DocProcessor` | Documentation loading, feature map building, coverage tracking |
| `digital.vasic.llmorchestrator` | `../LLMOrchestrator` | Headless CLI agent management (opencode, claude-code, gemini, junie, qwen-code) |
| `digital.vasic.llmprovider` | `../LLMProvider` | LLM provider abstraction layer |
| `digital.vasic.security` | `../Security` | Security utilities (CORS, CSP, sanitization) |
| `digital.vasic.visionengine` | `../VisionEngine` | GoCV mechanical vision + LLM Vision API analysis |
| `digital.vasic.llmsverifier` | `../LLMsVerifier` | Strategy-based LLM selection and scoring |

### 4b. Git Submodules (via `.gitmodules`)

**Open-source tools** (26 submodules in `tools/opensource/`):

| Submodule | URL | Purpose |
|---|---|---|
| `scrcpy` | `git@github.com:Genymobile/scrcpy.git` | Android screen mirroring/capture |
| `allure2` | `git@github.com:allure-framework/allure2.git` | Test reporting framework |
| `leakcanary` | `git@github.com:square/leakcanary.git` | Memory leak detection |
| `docker-android` | `git@github.com:budtmo/docker-android.git` | Android emulator in Docker |
| `appium` | `git@github.com:appium/appium.git` | Mobile test automation |
| `midscene` | `git@github.com:web-infra-dev/midscene.git` | AI-powered UI testing |
| `mem0` | `git@github.com:mem0ai/mem0.git` | AI memory layer |
| `moondream` | `git@github.com:vikhyat/moondream.git` | Vision language model |
| `ui-tars` | `git@github.com:bytedance/UI-TARS.git` | UI agent (ByteDance) |
| `perfetto` | `git@github.com:google/perfetto.git` | System profiling |
| `chroma` | `git@github.com:chroma-core/chroma.git` | Vector database |
| `shortest` | `git@github.com:antiwork/shortest.git` | AI browser automation |
| `marker` | `git@github.com:VikParuchuri/marker.git` | Document conversion |
| `kiwi-tcms` | `git@github.com:kiwitcms/Kiwi.git` | Test case management |
| `testdriverai` | `git@github.com:testdriverai/testdriverai.git` | AI test driver |
| `stagehand` | `git@github.com:browserbase/stagehand.git` | Browser automation |
| `unstructured` | `git@github.com:Unstructured-IO/unstructured.git` | Document processing |
| `redroid` | `git@github.com:remote-android/redroid-doc.git` | Android in container |
| `signoz` | `git@github.com:SigNoz/signoz.git` | Observability platform |
| `docling` | `git@github.com:DS4SD/docling.git` | Document understanding |
| `llama-index` | `git@github.com:run-llama/llama_index.git` | LLM data framework |
| `appcrawler` | `git@github.com:nicetester/AppCrawler.git` | Android app crawler |
| `browser-use` | `git@github.com:browser-use/browser-use.git` | AI browser agent |
| `skyvern` | `git@github.com:Skyvern-AI/skyvern.git` | AI workflow automation |
| `anthropic-quickstarts` | `git@github.com:anthropics/anthropic-quickstarts.git` | Anthropic examples |
| `ui-tars-desktop` | `git@github.com:bytedance/UI-TARS-desktop.git` | Desktop UI agent |

**Test apps** (1 submodule):
| Submodule | URL | Purpose |
|---|---|---|
| `rest-demo` | `git@github.com:nicehash/rest-clients-demo.git` | REST API test application |

---

## 5. TESTING FRAMEWORK — Complete Testing Mechanism

### 5.1 Standard QA Pipeline (`helixqa run`)

The standard pipeline orchestrates testing as follows:

1. **Load test banks** from YAML files or directories
2. **For each platform** (Android, AndroidTV, Web, Desktop):
   - Create platform-specific crash detector
   - Execute each challenge definition via `digital.vasic.challenges` runner
   - Run step-by-step validation (pre/post screenshots, crash detection)
   - Apply speed-mode delays (slow: 1s, normal: 100ms, fast: 0ms)
3. **Generate combined QA report** (Markdown/HTML/JSON)

### 5.2 Autonomous QA Pipeline (`helixqa autonomous`)

The autonomous pipeline is a 5-phase system:

| Phase | Description | Model Type |
|---|---|---|
| **0. Deploy** | Auto-deploy Ollama + vision model on remote GPU host via SSH | — |
| **1. Setup/Learn** | Scan project docs, code, git. Select LLMs via LLMsVerifier. Build feature map. Spawn CLI agents. Init VisionEngine. | Chat (PlanningStrategy) |
| **2. Plan** | LLM generates test cases from knowledge base | Chat (PlanningStrategy) |
| **3. Execute** | Doc-driven verification — parallel PlatformWorkers verify every documented feature, capturing screenshots/video at each step | Vision (NavigationStrategy) |
| **3.5 Curiosity** | LLM vision drives exploration of undiscovered areas (login, browse, favorites, play, edge cases) | Vision (NavigationStrategy) |
| **4. Analyze** | LLM vision analyzes screenshots, creates deduplicated issue tickets | Vision (AnalysisStrategy) / Chat |

### 5.3 Test Count
- **235 tests** total, all passing with `-race` flag
- Run via: `make test` or `go test ./... -count=1`

### 5.4 Anti-Bluff Testing System

HelixQA has an **extensive anti-bluff validation system** to ensure tests validate REAL functionality:

1. **Tests MUST validate user-visible behaviour**, not just metadata
2. **PASS/FAIL/SKIP must be mechanically distinguishable** — SKIP always carries explicit reason
3. **Every gate MUST have a paired mutation test** — deliberately break feature, gate MUST FAIL
4. **Challenges must cross-reference real device telemetry** (logcat, captured frames, network probes, kernel state)
5. **The bar for shipping is "users can use the feature"**, not "tests pass"
6. **No false-success results are tolerable** — green test + broken feature > honest red test

**Bluff taxonomy** (forbidden patterns):
- **Wrapper bluff**: assertions PASS but wrapper exit-code logic is buggy
- **Contract bluff**: system advertises capability but rejects it in dispatch
- **Structural bluff**: checks file exists but doesn't verify content
- **Comment bluff**: code comment promises behavior code doesn't have
- **Skip bluff**: `t.Skip("not running yet")` without `SKIP-OK: #<ticket>` marker

### 5.5 Mutation Testing
- Every gate must have a paired mutation test in `scripts/testing/meta_test_false_positive_proof.sh` (parent repo)
- Mutation deliberately breaks the feature → the gate MUST then FAIL
- A gate without a paired mutation = BLUFF gate = Constitution violation

---

## 6. CHALLENGE SYSTEM

### 6.1 Test Bank Format (YAML)

```yaml
version: "1.0"
name: "Yole Core Tests"
test_cases:
  - id: TC-001
    name: "Create new document"
    category: functional
    priority: critical               # critical|high|medium|low
    platforms: [android, web, desktop]
    steps:
      - name: "Open app"
        action: "Launch application"
        expected: "Main editor screen visible"
    tags: [core, smoke]
    documentation_refs:
      - type: user_guide
        section: "3.1"
        path: "docs/USER_MANUAL.md"
    estimated_duration: "30s"
    expected_result: "Document created successfully"
```

### 6.2 Challenge Categories

| Category | Description |
|---|---|
| `functional` | Core functionality tests |
| `smoke` | Quick sanity checks |
| `regression` | Regression guards |
| `performance` | Performance benchmarks |
| `security` | Security/penetration tests |
| `accessibility` | Accessibility validation |
| `navigation` | UI navigation tests |

### 6.3 Priority Levels

| Priority | Testing Order | Description |
|---|---|---|
| `happy` | 1st (FIRST) | Primary user flows |
| `standard` | 2nd | Reasonable variations |
| `edge` | 3rd | Edge cases and error scenarios |
| `adversarial` | 4th (LAST) | Invalid/stress inputs |

### 6.4 Challenge Execution

Challenges are loaded from YAML banks, converted to `challenge.Definition`, and executed via the `digital.vasic.challenges` runner. Each challenge:
1. Gets a `challenge.Config` with ID, verbose, timeout, results dir
2. Runs via `runner.Run(ctx, def.ID, cfg)` 
3. Produces a `challenge.Result` with status (passed/failed/error)
4. Step validation runs between challenges (pre/post screenshots, crash detection)

### 6.5 Platform Targeting
- Each test case can specify `platforms: [android, web, desktop]`
- `helixqa list --platform android` filters to Android-only tests
- Categories: `byCategory()`, `byPriority()`, `byTag()`

---

## 7. SCREENSHOT/CAPTURE SYSTEM

### 7.1 Evidence Collection (`pkg/evidence/collector.go`)

```go
type Collector struct { /* ... */ }
type Item struct {
    Type      Type      // screenshot|video|logcat|stacktrace|console_log
    Path      string
    Platform  config.Platform
    Step      string
    Timestamp time.Time
    Size      int64
}

func (c *Collector) CaptureScreenshot(ctx, name) (*Item, error)
func (c *Collector) CaptureLogcat(ctx, name, lines) (*Item, error)
func (c *Collector) StartRecording(ctx, name) error
func (c *Collector) StopRecording(ctx) (*Item, error)
```

### 7.2 Screenshot Validation

- `IsBlankScreenshot()` — validates screenshots before vision analysis
- Samples 81 pixels in a 9×9 grid; fails if max channel range < 20
- Prevents blank/black screenshots from being sent to LLM vision

### 7.3 Video Recording

- **Android**: `adb shell screenrecord` with 180-second cap → looped segments → concatenated via `ffmpeg -f concat -c copy`
- **Web**: Playwright video capture API
- **Desktop**: X11 recording
- A 2-hour session produces a continuous 2-hour MP4

### 7.4 Session Recording (`pkg/session/`)

```go
type SessionRecorder struct { /* ... */ }
func (sr *SessionRecorder) StartRecording(ctx, platform) error
func (sr *SessionRecorder) StopRecording(ctx, platform) (string, error)
func (sr *SessionRecorder) CaptureScreenshot(ctx, platform, name) (Screenshot, error)
func (sr *SessionRecorder) RecordEvent(event TimelineEvent)
func (sr *SessionRecorder) VideoTimestamp(platform) time.Duration
func (sr *SessionRecorder) ExportTimeline() []TimelineEvent
```

### 7.5 Audio Recording
- Optional audio capture during test execution
- Detects audio reproduction problems (glitches, dropouts, distortion)
- Configurable: quality (standard/high/ultra), format (wav/flac), device (default/adb)

---

## 8. ANTI-BLUFF TESTING (Detailed)

Constitution §8.1 + §11 mandate the most rigorous anti-bluff system:

### 8.1 Five-Constraint Rule
1. Tests MUST validate user-visible behaviour, not just metadata
2. PASS/FAIL/SKIP must be mechanically distinguishable
3. Every gate MUST have a paired mutation test
4. Challenges and tests are in the same boat — cross-reference real device telemetry
5. The bar for shipping is "users can use the feature"

### 8.2 Evidence Requirements
- Every PASS must carry positive evidence captured during execution
- No metadata-only PASS, no configuration-only PASS, no "absence-of-error" PASS
- Evidence types: kernel `/proc/*` state, captured audio/video, dumpsys output, real input-event delivery, real surface composition

### 8.3 Functional Probe Floor
- TCP-open is the FLOOR, not the ceiling
- Postgres → `SELECT 1`; Redis → `PING` returns `PONG`; HTTP → real request, real response, non-empty body
- Container `Up` ≠ application healthy

### 8.4 Enforceable Violation Detection
- `jq` queries on `pipeline-report.json` to detect violations
- Example: grep for raw coordinates or sleep commands in action logs
- Example: check every playback test has prior geo probe
- Example: verify phases run in correct order

---

## 9. INTEGRATION METHOD

### 9.1 As a Go Module
```bash
# Install CLI
go install digital.vasic.helixqa/cmd/helixqa@latest

# Or use as a library
go get digital.vasic.helixqa
```

### 9.2 Sibling Directory Structure
HelixQA expects sibling modules in the parent directory:
```
parent-dir/
├── HelixQA/           # This repo
├── Challenges/        # digital.vasic.challenges
├── Containers/        # digital.vasic.containers
├── DocProcessor/      # digital.vasic.docprocessor
├── LLMOrchestrator/   # digital.vasic.llmorchestrator
├── LLMProvider/       # digital.vasic.llmprovider
├── Security/          # digital.vasic.security
├── VisionEngine/      # digital.vasic.visionengine
└── LLMsVerifier/      # digital.vasic.llmsverifier
```

### 9.3 As a CLI Tool
Build binary at `bin/helixqa`:
```bash
make build    # -> bin/helixqa
make install  # -> $GOPATH/bin/helixqa
```

### 9.4 Project Integration (Consumer Side)
1. Clone HelixQA into your project structure as sibling directory
2. Create YAML test banks (`banks/*.yaml`) with project-specific test cases
3. Configure `.env` with platform settings and API keys
4. Run: `helixqa run --banks tests/banks/ --platform all`

---

## 10. CLI COMMANDS

| Command | Description | Key Flags |
|---|---|---|
| `helixqa run` | Execute QA pipeline across platforms | `--banks`, `--platform`, `--device`, `--package`, `--speed`, `--report`, `--timeout`, `--browser-url`, `--desktop-process`, `--validate`, `--record`, `--tickets` |
| `helixqa list` | List test cases from banks | `--banks`, `--platform`, `--category`, `--priority`, `--tag`, `--json` |
| `helixqa report` | Generate report from existing results | `--input`, `--format`, `--output` |
| `helixqa autonomous` | Run autonomous LLM-driven QA session | `--project`, `--platforms`, `--env`, `--timeout`, `--coverage-target`, `--output`, `--report`, `--curiosity`, `--curiosity-timeout`, `--verbose` |
| `helixqa replay` | Replay a ticket's OCU action chain (dry-run) | Ticket-specific arguments |
| `helixqa version` | Print version (v0.2.0) | — |
| `helixqa help` | Show usage | — |

---

## 11. CONFIGURATION

### 11.1 `.env` Configuration Groups

| Group | Key Variables |
|---|---|
| **Master Switch** | `HELIX_AUTONOMOUS_ENABLED`, `HELIX_AUTONOMOUS_PLATFORMS`, `HELIX_AUTONOMOUS_TIMEOUT`, `HELIX_AUTONOMOUS_COVERAGE_TARGET`, `HELIX_AUTONOMOUS_CURIOSITY_ENABLED` |
| **LLMsVerifier** | `LLMSVERIFIER_CONFIG`, `LLMSVERIFIER_STRATEGY`, `LLMSVERIFIER_MIN_SCORE`, `LLMSVERIFIER_MAX_MODELS`, `LLMSVERIFIER_CACHE_RESULTS` |
| **Distributed Vision** | `HELIX_VISION_HOSTS`, `HELIX_VISION_MULTI_USER`, `HELIX_LLAMACPP_RPC_MODEL` |
| **Vision Providers** | `ASTICA_API_KEY`, `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `KIMI_API_KEY`, `STEPFUN_API_KEY`, `NVIDIA_API_KEY`, `GROQ_API_KEY`, `MISTRAL_API_KEY`, `DEEPSEEK_API_KEY`, `XAI_API_KEY`, `TOGETHER_API_KEY`, `QWEN_API_KEY`, `JUNIE_API_KEY` |
| **Local Vision** | `HELIX_OLLAMA_URL`, `HELIX_OLLAMA_MODEL` |
| **CLI Agents** | `HELIX_AGENTS_ENABLED`, `HELIX_AGENT_OPENCODE_PATH`, `HELIX_AGENT_CLAUDE_PATH`, `HELIX_AGENT_GEMINI_PATH`, `HELIX_AGENT_JUNIE_PATH`, `HELIX_AGENT_QWEN_PATH` |
| **Vision Engine** | `HELIX_VISION_PROVIDER`, `HELIX_VISION_OPENCV_ENABLED`, `HELIX_VISION_SSIM_THRESHOLD` |
| **Doc Processor** | `HELIX_DOCS_ROOT`, `HELIX_DOCS_AUTO_DISCOVER`, `HELIX_DOCS_FORMATS` |
| **Recording** | `HELIX_RECORDING_VIDEO`, `HELIX_RECORDING_SCREENSHOTS`, `HELIX_RECORDING_VIDEO_QUALITY`, `HELIX_FFMPEG_PATH` |
| **Audio Recording** | `HELIXQA_RECORDING_AUDIO`, `HELIXQA_RECORDING_AUDIO_QUALITY`, `HELIXQA_RECORDING_AUDIO_FORMAT`, `HELIXQA_RECORDING_AUDIO_DEVICE` |
| **Platform-specific** | `HELIX_ANDROID_DEVICE`, `HELIX_ANDROID_PACKAGE`, `HELIX_WEB_URL`, `HELIX_WEB_BROWSER`, `HELIX_DESKTOP_PROCESS`, `HELIX_DESKTOP_DISPLAY` |
| **Output** | `HELIX_OUTPUT_DIR`, `HELIX_REPORT_FORMATS`, `HELIX_TICKETS_ENABLED`, `HELIX_TICKETS_MIN_SEVERITY` |
| **Remote Vision Auto-Deploy** | `HELIX_VISION_HOST`, `HELIX_VISION_USER`, `HELIX_VISION_MODEL` |
| **llama.cpp RPC** | `HELIX_LLAMACPP_RPC_ENABLED`, `HELIX_LLAMACPP_RPC_WORKERS`, `HELIX_LLAMACPP_FREE_GPU` |
| **Device Exclusion** | `HELIX_ADB_EXCLUDE` (substring match on `adb devices -l`) |
| **QA Infrastructure** | `HELIX_INFRA_HOST`, `HELIX_INFRA_API_SERVICE`, `HELIX_INFRA_API_PORT`, `HELIX_INFRA_API_HEALTH_PATH` |

### 11.2 Speed Modes
| Mode | Step Delay | Use Case |
|---|---|---|
| `slow` | 1 second | Debugging |
| `normal` | 100ms | Default |
| `fast` | 0ms | CI pipelines |

### 11.3 Report Formats
- `markdown` — Default
- `html` — Interactive HTML report
- `json` — Machine-readable JSON

---

## 12. TEST TYPES SUPPORTED

| Test Type | Support | Implementation |
|---|---|---|
| **Unit** | ✅ Full | `go test ./...`, testify assertions, table-driven |
| **Integration** | ✅ Full | testcontainers-go, real services |
| **E2E** | ✅ Full | Autonomous pipeline, real devices |
| **Functional** | ✅ Full | YAML test banks, challenge execution |
| **Regression** | ✅ Full | Mutation testing pairs, CONST-032 |
| **Performance** | ✅ Benchmark | `go test -bench` |
| **Security** | ✅ Full | SonarQube, Snyk, Trivy |
| **Stress** | ✅ Full | Resource-limited concurrent tests |
| **Crash Detection** | ✅ Real-time | ADB logcat, pgrep, process monitoring |
| **ANR Detection** | ✅ Real-time | Android ANR parsing |
| **Visual/UI** | ✅ LLM-powered | Vision model screenshot analysis |
| **Accessibility** | ✅ LLM-powered | Accessibility analysis via IssueDetector |
| **UX** | ✅ LLM-powered | UX issue detection via navigation graph analysis |
| **Geo-restriction** | ✅ Automatic | ProbeGeoRestriction() before playback tests |
| **Challenge** | ✅ Full | Challenge scripts validating real-life use cases |

---

## 13. CLIENT TESTING CAPABILITIES

### 13.1 Android
- **ADBExecutor**: `adb shell input tap/text/keyevent/swipe`
- **Crash detection**: `adb logcat` parsing, `pidof` process checks
- **ANR detection**: Android ANR log parsing
- **Screenshot**: `adb shell screencap`
- **Video**: `adb shell screenrecord` (180s segments, auto-concatenated)
- **Device preservation**: Captures and restores system settings (font_scale, brightness, rotation, etc.)
- **Device exclusion**: `.devignore` file + `HELIX_ADB_EXCLUDE` env var
- **Multi-device**: Auto-detects all ADB devices, tests each

### 13.2 Android TV
- Full Android testing + **Channels testing framework** (`pkg/planning/androidtv_channels_framework.go`)
- Automatic detection and testing of:
  - Default channel
  - Category channels
  - Watch Next row
  - Deep links
- DPAD navigation support

### 13.3 Web
- **PlaywrightExecutor**: Full browser automation via Chrome DevTools Protocol
- **chromedp**: Alternative headless Chrome automation
- **go-rod**: Additional browser automation library
- **Crash detection**: Browser process monitoring (pgrep)
- **Console error collection**: Captures browser console errors
- Supported browsers: Chromium, Chrome, Firefox

### 13.4 Desktop (Linux)
- **X11Executor**: Desktop automation via `xdotool`
- **Process monitoring**: pgrep + kill for JVM processes
- **stderr monitoring**: Desktop app error capture
- **Display targeting**: Configurable X11 display (`:0`)

### 13.5 CLI
- Platform constant defined (`PlatformCLI`)
- Process-level testing

### 13.6 API
- Platform constant defined (`PlatformAPI`)
- HTTP endpoint testing

---

## 14. API TESTING

HelixQA supports API testing through:
- `PlatformAPI` constant and filtering
- QA Infrastructure boot (`pkg/infra/`): health checks on backend services (database, cache, API)
- Env-configurable: `HELIX_INFRA_HOST`, `HELIX_INFRA_API_SERVICE`, `HELIX_INFRA_API_PORT`, `HELIX_INFRA_API_HEALTH_PATH`
- Challenge-based API testing via YAML test banks

---

## 15. REPORTING

### 15.1 Report Formats
| Format | Description |
|---|---|
| **Markdown** | Default, human-readable, includes platform breakdown, crash/ANR counts, step validation tables |
| **HTML** | Interactive web report |
| **JSON** | Machine-readable (`qa-report.json`) |

### 15.2 Report Structure
```go
type QAReport struct {
    Title            string
    GeneratedAt      time.Time
    PlatformResults  []*PlatformResult
    TotalChallenges  int
    PassedChallenges int
    FailedChallenges int
    TotalCrashes     int
    TotalANRs        int
    TotalDuration    time.Duration
    OutputDir        string
}
```

### 15.3 Output Structure
```
qa-results/
├── latest -> session-NNNN           # Symlink to most recent session
├── session-1774785711/
│   ├── screenshots/                 # PNG screenshots (execute + curiosity phases)
│   ├── videos/                      # MP4 recordings (pulled from Android device)
│   ├── evidence/                    # Logcat dumps, crash traces
│   ├── frames/                      # Video frame extracts
│   ├── tickets/                     # Generated markdown tickets
│   ├── pipeline-report.json         # Session results (tests, coverage, issues)
│   ├── qa-report.md                 # Markdown report
│   ├── qa-report.html               # HTML report
│   └── qa-report.json               # JSON report
```

### 15.4 Ticket Generation
- Markdown tickets with: severity, platform, reproduction steps, expected/actual behavior, stack traces, logs, screenshot evidence
- Video reference: exact filename + MM:SS timestamp
- Session reference: HelixQA session ID + step number
- Deduplication: same-title findings skipped, related findings grouped

### 15.5 LLM Cost Tracking
- Per-call cost records (thread-safe via `sync.RWMutex`)
- Auto-records after every successful `Chat()` and `Vision()` call
- Cost summary attached to `PipelineResult`
- Provider cost rates tracked in `visionModelRegistry`

---

## 16. DOCUMENTATION

| Document | Purpose |
|---|---|
| `README.md` | Main documentation — features, installation, usage, test bank format, architecture overview |
| `CLAUDE.md` | Agent rules, architecture pointers, constitutions, vision provider architecture, build & test |
| `AGENTS.md` | Development guide, constraints, anti-bluff covenants, mandatory rules |
| `CONSTITUTION.md` | Non-negotiable rules inherited from HelixAgent root project |
| `ARCHITECTURE.md` | Full architecture with Mermaid diagrams (component, sequence, class, state, flowchart) |
| `API_REFERENCE.md` | Complete API documentation for ALL packages with types and functions |
| `USER_GUIDE_AUTONOMOUS.md` | Autonomous QA session tutorial |
| `VIDEO_COURSE_AUTONOMOUS.md` | Video tutorial for autonomous mode |
| `.env.example` | Configuration template with all env vars documented |

---

## 17. DEPENDENCIES

### 17.1 Sibling Module Dependencies (Critical)

| Module | Version | Required | Purpose |
|---|---|---|---|
| `digital.vasic.challenges` | local (replace) | YES | Test execution engine |
| `digital.vasic.containers` | local (replace) | YES | Container management |
| `digital.vasic.docprocessor` | local (replace) | YES (autonomous) | Document processing |
| `digital.vasic.llmorchestrator` | local (replace) | YES (autonomous) | CLI agent pool |
| `digital.vasic.llmsverifier` | local (replace) | YES (autonomous) | LLM selection/scoring |
| `digital.vasic.llmprovider` | local (replace) | YES | LLM abstraction |
| `digital.vasic.security` | local (replace) | YES | Security utilities |
| `digital.vasic.visionengine` | local (replace) | YES (autonomous) | Computer vision |

### 17.2 External Go Dependencies (Key Direct)

| Package | Purpose |
|---|---|
| `chromedp/chromedp` + `go-rod/rod` | Browser automation |
| `testcontainers-go` | Integration test containers |
| `gocv.io/x/gocv` | OpenCV bindings |
| `otiai10/gosseract` | OCR (Tesseract) |
| `prometheus/client_golang` | Prometheus metrics |
| `opentelemetry/otel` | OpenTelemetry tracing |
| `nats-io/nats.go` | NATS messaging |
| `gorilla/websocket` | WebSocket |
| `pion/webrtc` | WebRTC streaming |
| `chromem-go` | In-memory vector store |
| `minio/minio-go` | S3 object storage |
| `failsafe-go` | Resilience (retry, circuit breaker) |
| `google/uuid` | UUID generation |
| `mattn/go-sqlite3` | SQLite (memory store) |
| `stretchr/testify` | Test assertions |

### 17.3 System Dependencies
- **ffmpeg** — Video concatenation and processing
- **adb** (Android Debug Bridge) — Android device control
- **xdotool** — Desktop Linux UI automation
- **scrcpy** — Android screen mirroring
- **Ollama** — Local LLM inference
- **llama.cpp** (with `-DGGML_RPC=ON`) — Distributed model inference

---

## 18. BUILD/RUN INSTRUCTIONS

### 18.1 Prerequisites
- Go 1.24+ (module says 1.26)
- Sibling directories: `../Challenges`, `../Containers` (and others for autonomous mode)
- ffmpeg, adb (for Android testing)
- xdotool (for desktop testing)
- At least one LLM API key or local Ollama for autonomous mode

### 18.2 Build
```bash
# Clone (SSH only — NO HTTPS)
git clone git@github.com:HelixDevelopment/HelixQA.git

# Initialize submodules
git submodule init && git submodule update --recursive

# Build binary
make build                    # -> bin/helixqa
# OR
make install                  # -> $GOPATH/bin/helixqa
```

### 18.3 Test
```bash
make test          # 235 tests
make test-race     # with race detection
make test-cover    # with coverage report
make vet           # static analysis
make lint          # golangci-lint
```

### 18.4 Run Standard QA
```bash
helixqa run --banks tests/banks/ --platform all
helixqa run --banks tests/ --platform android --device emulator-5554 --package com.example.app
helixqa list --banks tests/banks/ --platform android
helixqa report --input qa-results --format html
```

### 18.5 Run Autonomous QA
```bash
# 1. Configure
cp .env.example .env
# Edit .env — set at least one API key and platform settings

# 2. Run
helixqa autonomous --project /path/to/project \
  --platforms android,desktop,web \
  --env .env \
  --timeout 2h \
  --output qa-results/

# 3. View results
cat qa-results/qa-report.md
ls qa-results/tickets/
ls qa-results/videos/
```

### 18.6 Makefile Targets
| Target | Command |
|---|---|
| `all` | vet + test + build (default) |
| `build` | `go build -o bin/helixqa ./cmd/helixqa` |
| `install` | `go install ./cmd/helixqa` |
| `test` | `go test ./... -count=1` |
| `test-race` | `go test ./... -race -count=1` |
| `test-cover` | `go test ./... -coverprofile=coverage.out` + HTML report |
| `vet` | `go vet ./...` |
| `lint` | `golangci-lint run ./...` |
| `fmt` | `gofmt -w .` |
| `tidy` | `go mod tidy` |
| `clean` | Remove bin/, coverage, qa-results/ |

---

## 19. KEY CONSTITUTIONS & CONSTRAINTS

### 19.1 Non-Negotiable Rules
1. **NO CI/CD pipelines** — no GitHub Actions, GitLab CI, Jenkins, etc. All builds/tests via Makefile
2. **NO HTTPS for Git** — SSH URLs only
3. **NO manual container commands** — orchestrator handles everything
4. **100% project-agnostic** — never hardcoded to one consumer
5. **Fully autonomous LLM-driven QA** — ALL navigation by real LLM vision models, no hardcoded coordinates
6. **Geo-restriction detection** — probe before playback, never report as failure
7. **Testing priority order** — happy paths → standard → edge → adversarial
8. **Device state preservation** — restore settings after session
9. **Evidence-backed tickets** — video + screenshot + timestamp required
10. **No sudo/root** — all operations at user level
11. **Zero unfinished work** — no TODOs, no empty implementations
12. **Host session safety** — no suspend/hibernate/logout, memory budget ceiling at 60%
13. **No false-success** — green test + broken feature = worst outcome

### 19.2 Resource Limits
- Tests limited to 30-40% of host resources
- `GOMAXPROCS=2`, `nice -n 19`, `ionice -c 3`
- Memory budget: 60% max of system RAM
- Container memory limits required

---

## 20. VISION PROVIDER ARCHITECTURE

### 20.1 Phase-Specific Model Selection

| Phase | Strategy | Model Type | Optimized For |
|---|---|---|---|
| Learn/Plan | PlanningStrategy | Chat | Reasoning 35%, context 25%, structured output 20% |
| Execute/Curiosity | NavigationStrategy | Vision | JSON compliance 40%, GUI understanding 25%, speed 20% |
| Analyze | AnalysisStrategy | Vision | Description quality 35%, OCR 20%, object detection 20% |

### 20.2 Provider Scoring
- Score formula: `(0.6 * quality + 0.4 * reliability) * availability * costBonus`
- Free providers get 1.10x; cheap (<$0.002/1k) get 1.05x
- Configured API keys get 2x availability multiplier
- No hardcoded model preferences — all selection score-based

### 20.3 Supported Vision Providers
- **llama.cpp RPC** (distributed, MANDATORY when multiple hosts available)
- **Astica.AI** (cloud, Analyze only — specialized computer vision)
- **Gemini/OpenAI/Kimi/StepFun/NVIDIA/xAI/GitHub Models** (cloud)
- **Ollama** (local fallback, inferior to llama.cpp)

### 20.4 Bridged CLI Models
- Claude Code, Qwen Coder, OpenCode — discovered automatically
- Zero token cost (CLI handles billing)
- Only Claude Code supports vision input
- Compete alongside cloud and local models in scoring pool

---

## 21. RESILIENCE ARCHITECTURE

### 5 Degradation Levels
1. **Full capability** — All LLM + Vision working
2. **Degraded vision** — LLM Vision fails: GoCV-only mechanical analysis
3. **Degraded navigation** — Agent failures: partial evidence, partial report
4. **Session abort** — Unrecoverable errors: clean shutdown with error report
5. **Per-agent circuit breaker** — 3 consecutive failures → mark unhealthy, get replacement

### Resilience Patterns
- **Exponential backoff**: 1s/2s/4s for LLM calls
- **Malformed JSON fallback**: Re-prompt on bad LLM output
- **Prompt injection sanitization**: Sanitize paths, commands, content
- **FallbackChain**: Tries providers in score-ranked order

---

## SUMMARY

HelixQA is a sophisticated, production-grade QA orchestration framework that combines:
- **Traditional test execution** (YAML banks, challenge running, crash detection)
- **Autonomous LLM-driven testing** (computer vision navigation, doc-driven verification, curiosity exploration)
- **Rigorous anti-bluff validation** (mutation testing, positive evidence only, no false positives)
- **Cross-platform support** (Android, Android TV, Web, Desktop, CLI, API)
- **Rich evidence collection** (screenshots, video, logcat, audio, timeline)
- **Flexible reporting** (Markdown, HTML, JSON tickets with evidence)
- **Distributed vision inference** (llama.cpp RPC across multiple hosts)
- **40+ LLM provider support** (OpenAI, Anthropic, Gemini, Groq, Mistral, DeepSeek, xAI, Ollama, etc.)

The framework is designed as a thin orchestration layer over `digital.vasic.challenges` and `digital.vasic.containers`, following strict composition-over-reimplementation principles.
