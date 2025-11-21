# llama.cpp Integration Implementation Plan

## Overview

This document outlines the complete architecture and implementation strategy for integrating llama.cpp with the translation system, including hardware detection, model management, comprehensive testing, and documentation organization.

## System Architecture

### Hardware Detection (✅ COMPLETED)
**Location**: `pkg/hardware/detector.go`

**Features**:
- Detects system architecture (arm64, amd64)
- Measures total and available RAM
- Identifies CPU model and core count
- Detects GPU acceleration (Metal, CUDA, ROCm, Vulkan)
- Calculates maximum supportable model size
- Supports macOS, Linux, and Windows

**Capabilities**:
- M3 Pro with 18GB RAM → Can run up to 13B models comfortably
- Automatic model size recommendation based on available resources

**Usage Example**:
```go
detector := hardware.NewDetector()
caps, err := detector.Detect()
if err != nil {
    log.Fatal(err)
}

fmt.Println(caps.String())
// Hardware Capabilities:
//   Architecture: arm64
//   CPU: Apple M3 Pro (12 cores)
//   Total RAM: 18.0 GB
//   Available RAM: 14.2 GB
//   GPU: metal acceleration
//   Max Model Size: 13B parameters
```

### Model Registry (✅ COMPLETED)
**Location**: `pkg/models/registry.go`

**Translation-Optimized Models**:

#### Priority 1: Translation-Specialized Models
1. **Hunyuan-MT-7B (Q4/Q8)** - Tencent's commercial-grade translation model
   - 33 languages including Russian and Serbian
   - Q4: 6GB RAM minimum, Q8: 9GB RAM minimum
   - Best-in-class 7B translation quality

2. **Aya-23-8B** - Cohere's multilingual powerhouse
   - 23 languages with strong Slavic support
   - Excellent for Russian-Serbian translation

#### Priority 2: General-Purpose with Strong Translation
3. **Qwen2.5-7B/14B/27B** - Alibaba's multilingual models
   - Exceptional multilingual capabilities
   - Long context (32K tokens)
   - Scalable from 7B to 27B based on hardware

4. **Mistral-7B-Instruct** - Fast and capable
   - Good general translation quality
   - Efficient inference

#### Priority 3: Compact Models
5. **Phi-3-Mini-3.8B** - Microsoft's efficient model
   - For resource-constrained systems (4GB RAM)
   - Moderate quality but very fast

6. **Gemma-2-9B** - Google's balanced model
   - Good multilingual support
   - Efficient architecture

**Model Selection Algorithm**:
```go
registry := models.NewRegistry()

// Automatic best model selection
bestModel, err := registry.FindBestModel(
    caps.AvailableRAM,
    []string{"ru", "sr"}, // Languages
    caps.HasGPU,
)

// Model scoring considers:
// - Translation optimization (highest weight)
// - Language support
// - Quality rating
// - RAM efficiency
// - Parameter count
// - Context length
```

## Implementation Roadmap

### Phase 1: Core Infrastructure (IN PROGRESS)

#### 1.1 llama.cpp CLI Wrapper
**Location**: `pkg/translator/llm/llamacpp.go`

**Implementation**:
```go
package llm

import (
    "context"
    "digital.vasic.translator/pkg/translator"
    "digital.vasic.translator/pkg/hardware"
    "digital.vasic.translator/pkg/models"
    "os/exec"
    "fmt"
)

// LlamaCppClient implements llama.cpp integration
type LlamaCppClient struct {
    config       translator.TranslationConfig
    modelPath    string
    hardwareCaps *hardware.Capabilities
    threads      int
    contextSize  int
}

// NewLlamaCppClient creates llama.cpp client with auto-configuration
func NewLlamaCppClient(config translator.TranslationConfig) (*LlamaCppClient, error) {
    // Detect hardware
    detector := hardware.NewDetector()
    caps, err := detector.Detect()
    if err != nil {
        return nil, fmt.Errorf("hardware detection failed: %w", err)
    }

    // Configure threads (use 75% of physical cores)
    threads := int(float64(caps.CPUCores) * 0.75)
    if threads < 1 {
        threads = 1
    }

    return &LlamaCppClient{
        config:       config,
        hardwareCaps: caps,
        threads:      threads,
        contextSize:  8192,
    }, nil
}

// Translate uses llama.cpp for translation
func (c *LlamaCppClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
    // Build llama-cli command
    cmd := exec.CommandContext(ctx, "llama-cli",
        "-m", c.modelPath,
        "-p", prompt,
        "-n", "2048",  // max tokens to generate
        "-t", fmt.Sprintf("%d", c.threads),
        "-c", fmt.Sprintf("%d", c.contextSize),
        "--temp", "0.3",  // low temperature for consistent translation
        "--top-p", "0.9",
        "--repeat-penalty", "1.1",
    )

    // Enable GPU acceleration if available
    if c.hardwareCaps.HasGPU {
        switch c.hardwareCaps.GPUType {
        case "metal":
            cmd.Args = append(cmd.Args, "-ngl", "99")  // offload all layers to GPU
        case "cuda":
            cmd.Args = append(cmd.Args, "-ngl", "99")
        }
    }

    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("llama.cpp execution failed: %w", err)
    }

    return string(output), nil
}

// GetProviderName returns provider name
func (c *LlamaCppClient) GetProviderName() string {
    return "llamacpp"
}
```

#### 1.2 Model Download Manager
**Location**: `pkg/models/downloader.go`

**Features**:
- Download models from HuggingFace
- Verify checksums/signatures
- Cache models locally (~/.cache/translator/models/)
- Resume interrupted downloads
- Show progress bars

**Implementation**:
```go
package models

import (
    "crypto/sha256"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
)

type Downloader struct {
    cacheDir string
}

func NewDownloader() *Downloader {
    homeDir, _ := os.UserHomeDir()
    cacheDir := filepath.Join(homeDir, ".cache", "translator", "models")
    os.MkdirAll(cacheDir, 0755)

    return &Downloader{cacheDir: cacheDir}
}

func (d *Downloader) DownloadModel(model *ModelInfo) (string, error) {
    // Check if already downloaded
    modelPath := filepath.Join(d.cacheDir, model.ID+".gguf")
    if _, err := os.Stat(modelPath); err == nil {
        return modelPath, nil  // Already exists
    }

    // Download with progress
    fmt.Printf("Downloading %s...\n", model.Name)

    resp, err := http.Get(model.SourceURL)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    out, err := os.Create(modelPath + ".tmp")
    if err != nil {
        return "", err
    }
    defer out.Close()

    // Copy with progress
    _, err = io.Copy(out, resp.Body)
    if err != nil {
        os.Remove(modelPath + ".tmp")
        return "", err
    }

    // Verify checksum (if available)
    // ...

    // Rename to final name
    os.Rename(modelPath+".tmp", modelPath)

    return modelPath, nil
}
```

#### 1.3 Integration with Translation Pipeline
**Location**: `pkg/translator/llm/llm.go` (update)

Add llamacpp to provider list:
```go
const (
    ProviderOpenAI    Provider = "openai"
    ProviderAnthropic Provider = "anthropic"
    ProviderZhipu     Provider = "zhipu"
    ProviderDeepSeek  Provider = "deepseek"
    ProviderQwen      Provider = "qwen"
    ProviderOllama    Provider = "ollama"
    ProviderLlamaCpp  Provider = "llamacpp"  // NEW
)

// In NewLLMTranslator:
case ProviderLlamaCpp:
    client, err = NewLlamaCppClient(config)
```

### Phase 2: Testing Infrastructure (PENDING)

#### 2.1 Unit Tests

**Test Coverage Requirements**: 100% of new code

**Files to Create**:
1. `pkg/hardware/detector_test.go` - Hardware detection tests
2. `pkg/models/registry_test.go` - Model registry tests
3. `pkg/models/downloader_test.go` - Download manager tests
4. `pkg/translator/llm/llamacpp_test.go` - llama.cpp wrapper tests

**Example Test Structure**:
```go
package hardware_test

import (
    "testing"
    "digital.vasic.translator/pkg/hardware"
)

func TestDetectorRAMDetection(t *testing.T) {
    detector := hardware.NewDetector()
    caps, err := detector.Detect()

    if err != nil {
        t.Fatalf("Detection failed: %v", err)
    }

    if caps.TotalRAM == 0 {
        t.Error("Total RAM should not be zero")
    }

    if caps.AvailableRAM == 0 {
        t.Error("Available RAM should not be zero")
    }

    if caps.AvailableRAM > caps.TotalRAM {
        t.Error("Available RAM cannot exceed total RAM")
    }
}

func TestDetectorGPUDetection(t *testing.T) {
    detector := hardware.NewDetector()
    caps, err := detector.Detect()

    if err != nil {
        t.Fatalf("Detection failed: %v", err)
    }

    // On M3 Pro, should detect Metal
    if caps.Architecture == "arm64" && !caps.HasGPU {
        t.Error("M3 Pro should have GPU detected")
    }

    if caps.HasGPU && caps.GPUType == "" {
        t.Error("GPU type should be set when GPU is available")
    }
}

func TestModelSizeCalculation(t *testing.T) {
    tests := []struct {
        name         string
        ramGB        float64
        hasGPU       bool
        expectedMinB uint64
    }{
        {"4GB System", 4, false, 1_000_000_000},
        {"8GB System", 8, false, 3_000_000_000},
        {"16GB System", 16, false, 7_000_000_000},
        {"18GB M3 Pro", 18, true, 7_000_000_000},
        {"32GB System", 32, false, 13_000_000_000},
    }

    detector := hardware.NewDetector()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            maxSize := detector.calculateMaxModelSize(
                uint64(tt.ramGB * 1024 * 1024 * 1024),
                tt.hasGPU,
            )

            if maxSize < tt.expectedMinB {
                t.Errorf("Expected at least %dB, got %dB", tt.expectedMinB, maxSize)
            }
        })
    }
}
```

#### 2.2 Integration Tests
**Location**: `test/integration/llamacpp_integration_test.go`

Tests full pipeline:
- Hardware detection → Model selection → Download → Translation

#### 2.3 End-to-End Tests
**Location**: `test/e2e/llamacpp_translation_e2e_test.go`

Tests complete translation workflow:
- Load sample Russian text
- Translate to Serbian using llamacpp
- Verify output quality
- Compare with API-based translation

#### 2.4 Performance Tests
**Location**: `test/performance/llamacpp_benchmark_test.go`

Benchmarks:
- Translation speed (tokens/second)
- Memory usage
- CPU utilization
- GPU utilization (if available)
- Concurrent translation performance

#### 2.5 Stress Tests
**Location**: `test/stress/llamacpp_stress_test.go`

Tests:
- 100 concurrent translations
- Long-running translation sessions
- Memory leak detection
- Error recovery

#### 2.6 Security Tests
**Location**: `test/security/model_download_security_test.go`

Tests:
- Model download verification
- Path traversal prevention
- Malicious model detection
- Secure file permissions

### Phase 3: Documentation Organization (PENDING)

#### 3.1 Create Documentation Directory Structure

```
Documentation/
├── README.md (index)
├── Architecture/
│   ├── Overview.md
│   ├── LLM-Integration.md
│   ├── Hardware-Detection.md
│   └── Model-Management.md
├── API/
│   ├── REST-API.md
│   ├── WebSocket-API.md
│   └── CLI-Usage.md
├── Deployment/
│   ├── Installation.md
│   ├── Configuration.md
│   └── Docker.md
├── Development/
│   ├── Contributing.md
│   ├── Testing.md
│   └── Code-Style.md
└── Tutorials/
    ├── Quick-Start.md
    ├── Local-LLM-Setup.md
    └── Advanced-Configuration.md
```

#### 3.2 Move Existing Documentation

Files to move:
- `CLAUDE.md` → `Documentation/Development/Claude-AI-Guide.md`
- `API_KEY_CLEANUP_REPORT.md` → `Documentation/Security/API-Key-Cleanup.md`
- Create new `Documentation/Architecture/LlamaCpp-Integration.md`

#### 3.3 Create llama.cpp Integration Guide

**File**: `Documentation/Tutorials/Local-LLM-Setup.md`

Content outline:
1. Introduction to local LLMs
2. Hardware requirements
3. Installing llama.cpp
4. Model selection guide
5. Configuration examples
6. Performance tuning
7. Troubleshooting

#### 3.4 Update All Cross-References

Scan all `.md` files and update links:
- `CLAUDE.md` → `Documentation/Development/Claude-AI-Guide.md`
- Add table of contents to main README.md
- Link from README.md to all documentation sections

### Phase 4: Advanced Features (FUTURE)

#### 4.1 Model Fine-Tuning
- Fine-tune models for Russian-Serbian translation
- Custom training pipeline
- Evaluation metrics

#### 4.2 Multi-Model Ensemble
- Run multiple models and combine results
- Voting mechanism for best translation
- Confidence scoring

#### 4.3 Real-Time Translation Streaming
- Stream translation as it's generated
- WebSocket integration
- Progressive rendering

## Usage Examples

### Basic Usage with Auto-Detection
```bash
# System automatically detects hardware and selects best model
./translator -input book.epub -locale sr -provider llamacpp -format epub
```

### Explicit Model Selection
```bash
# Use specific model
./translator -input book.epub -locale sr -provider llamacpp \
    -model hunyuan-mt-7b-q4 -format epub
```

### List Available Models
```bash
./translator --list-models

Available Models for Your Hardware (18GB RAM, Metal GPU):
✓ Hunyuan-MT 7B (Q8) - Excellent translation quality [RECOMMENDED]
✓ Qwen 2.5 14B (Q4) - High-quality multilingual
✓ Aya 23 8B (Q4) - Multilingual specialist
✓ Qwen 2.5 7B (Q4) - Balanced performance
✓ Mistral 7B (Q4) - Fast general-purpose

Use with: -model <model-id>
```

### Hardware Detection
```bash
./translator --hardware-info

Hardware Capabilities:
  Architecture: arm64
  CPU: Apple M3 Pro (12 cores)
  Total RAM: 18.0 GB
  Available RAM: 14.2 GB
  GPU: metal acceleration
  Max Model Size: 13B parameters

Recommended Models:
  1. Hunyuan-MT 7B (Q8) - Best quality
  2. Qwen 2.5 14B (Q4) - Larger model
  3. Aya 23 8B (Q4) - Multilingual
```

## Testing Strategy

### Test Execution Order
1. Unit tests (fast, no external dependencies)
2. Integration tests (model downloads, requires network)
3. E2E tests (full translation pipeline)
4. Performance tests (requires representative data)
5. Stress tests (long-running)
6. Security tests (specific scenarios)

### Continuous Integration
```yaml
# .github/workflows/llamacpp-tests.yml
name: llama.cpp Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: macos-latest  # For Metal support

    steps:
      - uses: actions/checkout@v2

      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21

      - name: Install llama.cpp
        run: brew install llama.cpp

      - name: Run unit tests
        run: go test ./pkg/hardware/... ./pkg/models/... -v

      - name: Run integration tests
        run: go test ./test/integration/... -v

      - name: Run e2e tests
        run: go test ./test/e2e/... -v -timeout=30m
```

### Test Coverage Goals
- Unit tests: 100% coverage
- Integration tests: All critical paths
- E2E tests: All user-facing workflows
- Performance tests: Baseline metrics established
- Stress tests: System limits identified
- Security tests: All attack vectors covered

## Implementation Checklist

### Core Infrastructure
- [x] Hardware detection system
- [x] Model registry with translation-optimized models
- [x] llama.cpp installation
- [ ] llama.cpp CLI wrapper
- [ ] Model download manager
- [ ] Integration with LLM provider system

### Testing
- [ ] Unit tests for hardware detection
- [ ] Unit tests for model registry
- [ ] Unit tests for model downloader
- [ ] Unit tests for llamacpp translator
- [ ] Integration tests
- [ ] E2E translation tests
- [ ] Performance benchmarks
- [ ] Stress tests
- [ ] Security tests

### Documentation
- [ ] Create Documentation/ directory structure
- [ ] Move existing documentation
- [ ] Write llama.cpp integration guide
- [ ] Update README.md with links
- [ ] Fix all cross-references
- [ ] Add code examples
- [ ] Create troubleshooting guide

### Validation
- [ ] Verify Pass 3 translation completion
- [ ] Validate all test suites pass
- [ ] Confirm documentation builds correctly
- [ ] Check all links work
- [ ] Force push cleaned git history

## Next Steps

1. **Immediate** (Current Session):
   - Continue monitoring Pass 3 translation
   - Create `pkg/translator/llm/llamacpp.go` wrapper
   - Create `pkg/models/downloader.go` manager

2. **Short-term** (Next Session):
   - Implement comprehensive test suites
   - Organize documentation structure
   - Update all cross-references

3. **Long-term** (Future Sessions):
   - Fine-tuning capabilities
   - Multi-model ensemble
   - Real-time streaming
   - Production deployment guides

## Resources

- llama.cpp GitHub: https://github.com/ggerganov/llama.cpp
- Hunyuan-MT: https://huggingface.co/Tencent/Hunyuan-MT-7B
- GGUF Format: https://github.com/ggerganov/ggml/blob/master/docs/gguf.md
- Model Quantization Guide: https://github.com/ggerganov/llama.cpp/blob/master/examples/quantize/README.md

## Contact & Support

For questions or issues with llama.cpp integration:
1. Check `Documentation/Tutorials/Local-LLM-Setup.md`
2. Review troubleshooting guide
3. Open GitHub issue with system info from `--hardware-info`
