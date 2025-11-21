package llm

import (
	"bytes"
	"context"
	"digital.vasic.translator/pkg/hardware"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/translator"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LlamaCppClient implements llama.cpp integration for local LLM inference
type LlamaCppClient struct {
	config       translator.TranslationConfig
	modelPath    string
	modelInfo    *models.ModelInfo
	hardwareCaps *hardware.Capabilities
	threads      int
	contextSize  int
	executable   string
}

// NewLlamaCppClient creates a new llama.cpp client with automatic hardware detection and model selection
func NewLlamaCppClient(config translator.TranslationConfig) (*LlamaCppClient, error) {
	// Detect hardware capabilities
	detector := hardware.NewDetector()
	caps, err := detector.Detect()
	if err != nil {
		return nil, fmt.Errorf("hardware detection failed: %w", err)
	}

	// Find llama-cli executable
	executable, err := findLlamaCppExecutable()
	if err != nil {
		return nil, fmt.Errorf("llama.cpp not found: %w (install with: brew install llama.cpp)", err)
	}

	// Initialize model registry
	registry := models.NewRegistry()

	// Determine which model to use
	var modelInfo *models.ModelInfo
	var modelPath string

	if config.Model != "" {
		// User specified a model
		modelInfo, exists := registry.Get(config.Model)
		if !exists {
			return nil, fmt.Errorf("model not found: %s (use --list-models to see available models)", config.Model)
		}

		// Check if system can run this model
		if !caps.CanRunModel(modelInfo.Parameters) {
			return nil, fmt.Errorf(
				"insufficient resources for model %s (%dB parameters). Your system supports up to %dB parameters",
				modelInfo.Name,
				modelInfo.Parameters/1_000_000_000,
				caps.MaxModelSize/1_000_000_000,
			)
		}
	} else {
		// Auto-select best model for hardware and languages
		modelInfo, err = registry.FindBestModel(
			caps.AvailableRAM,
			[]string{"ru", "sr"}, // Russian to Serbian translation
			caps.HasGPU,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find suitable model: %w", err)
		}

		fmt.Fprintf(os.Stderr, "[LLAMACPP] Auto-selected model: %s\n", modelInfo.Name)
	}

	// Check if model is already downloaded
	downloader := models.NewDownloader()
	modelPath, err = downloader.GetModelPath(modelInfo)
	if err != nil {
		// Model not downloaded, download it now
		fmt.Fprintf(os.Stderr, "[LLAMACPP] Downloading model: %s\n", modelInfo.Name)
		fmt.Fprintf(os.Stderr, "[LLAMACPP] This may take several minutes depending on your connection...\n")

		modelPath, err = downloader.DownloadModel(modelInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to download model: %w", err)
		}

		fmt.Fprintf(os.Stderr, "[LLAMACPP] Download complete: %s\n", modelPath)
	} else {
		fmt.Fprintf(os.Stderr, "[LLAMACPP] Using cached model: %s\n", modelPath)
	}

	// Configure threads (use 75% of physical cores for optimal performance)
	threads := int(float64(caps.CPUCores) * 0.75)
	if threads < 1 {
		threads = 1
	}

	// Configure context size (from model info or default)
	contextSize := modelInfo.ContextLength
	if contextSize == 0 {
		contextSize = 8192 // Default
	}

	fmt.Fprintf(os.Stderr, "[LLAMACPP] Configuration: %d threads, %d context size\n", threads, contextSize)
	if caps.HasGPU {
		fmt.Fprintf(os.Stderr, "[LLAMACPP] GPU acceleration: %s\n", caps.GPUType)
	}

	return &LlamaCppClient{
		config:       config,
		modelPath:    modelPath,
		modelInfo:    modelInfo,
		hardwareCaps: caps,
		threads:      threads,
		contextSize:  contextSize,
		executable:   executable,
	}, nil
}

// findLlamaCppExecutable locates the llama-cli executable
func findLlamaCppExecutable() (string, error) {
	// Try common locations
	candidates := []string{
		"llama-cli",                           // In PATH
		"/opt/homebrew/bin/llama-cli",        // Homebrew on Apple Silicon
		"/usr/local/bin/llama-cli",           // Homebrew on Intel
		"/usr/bin/llama-cli",                 // System install
		filepath.Join(os.Getenv("HOME"), ".local/bin/llama-cli"), // Local install
	}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("llama-cli not found in standard locations")
}

// GetProviderName returns the provider name
func (c *LlamaCppClient) GetProviderName() string {
	return "llamacpp"
}

// Translate translates text using llama.cpp local inference
func (c *LlamaCppClient) Translate(ctx context.Context, text string, prompt string) (string, error) {
	if text == "" || strings.TrimSpace(text) == "" {
		return text, nil
	}

	// Build command with optimized parameters for translation
	args := []string{
		"-m", c.modelPath,
		"-p", prompt,
		"-n", "4096",                    // max tokens to generate (increased for book translation)
		"-t", fmt.Sprintf("%d", c.threads),
		"-c", fmt.Sprintf("%d", c.contextSize),
		"--temp", "0.3",                 // low temperature for consistent, accurate translation
		"--top-p", "0.9",                // nucleus sampling
		"--top-k", "40",                 // top-k sampling
		"--repeat-penalty", "1.1",       // prevent repetition
		"--no-display-prompt",           // don't echo the prompt in output
	}

	// Enable GPU acceleration if available
	if c.hardwareCaps.HasGPU {
		switch c.hardwareCaps.GPUType {
		case "metal":
			args = append(args, "-ngl", "99") // offload all layers to Metal GPU
		case "cuda":
			args = append(args, "-ngl", "99") // offload all layers to CUDA
		case "rocm":
			args = append(args, "-ngl", "99") // offload all layers to ROCm
		}
	}

	// Create command with context for cancellation
	cmd := exec.CommandContext(ctx, c.executable, args...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Log the inference start
	startTime := time.Now()
	fmt.Fprintf(os.Stderr, "[LLAMACPP] Starting inference (text length: %d bytes)\n", len(text))

	// Execute
	err := cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		// Include stderr in error message for debugging
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("llama.cpp execution failed: %w\nStderr: %s", err, stderrStr)
		}
		return "", fmt.Errorf("llama.cpp execution failed: %w", err)
	}

	result := stdout.String()

	// Log performance metrics
	tokensPerSecond := float64(len(result)) / duration.Seconds()
	fmt.Fprintf(os.Stderr, "[LLAMACPP] Inference complete: %.1f tokens/sec, duration: %v\n",
		tokensPerSecond, duration.Round(time.Millisecond))

	// Post-process: remove any prompt echo that might have slipped through
	result = strings.TrimSpace(result)

	// Remove prompt if it appears at the start
	if strings.HasPrefix(result, prompt) {
		result = strings.TrimPrefix(result, prompt)
		result = strings.TrimSpace(result)
	}

	return result, nil
}

// GetModelInfo returns information about the currently loaded model
func (c *LlamaCppClient) GetModelInfo() *models.ModelInfo {
	return c.modelInfo
}

// GetHardwareInfo returns detected hardware capabilities
func (c *LlamaCppClient) GetHardwareInfo() *hardware.Capabilities {
	return c.hardwareCaps
}

// Validate checks if the client is properly configured
func (c *LlamaCppClient) Validate() error {
	// Check if model file exists
	if _, err := os.Stat(c.modelPath); err != nil {
		return fmt.Errorf("model file not found: %s", c.modelPath)
	}

	// Check if executable exists
	if _, err := os.Stat(c.executable); err != nil {
		return fmt.Errorf("llama-cli not found: %s", c.executable)
	}

	// Check if we have enough RAM
	requiredRAM := c.modelInfo.MinRAM
	if c.hardwareCaps.AvailableRAM < requiredRAM {
		return fmt.Errorf(
			"insufficient RAM: model requires %.1f GB, but only %.1f GB available",
			float64(requiredRAM)/(1024*1024*1024),
			float64(c.hardwareCaps.AvailableRAM)/(1024*1024*1024),
		)
	}

	return nil
}
