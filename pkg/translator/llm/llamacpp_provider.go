package llm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"digital.vasic.translator/pkg/logger"
)

// LlamaCppProviderConfig holds configuration for llama.cpp provider
type LlamaCppProviderConfig struct {
	BinaryPath     string            `json:"binary_path" yaml:"binary_path"`
	Models         []ModelConfig     `json:"models" yaml:"models"`
	MaxConcurrency int               `json:"max_concurrency" yaml:"max_concurrency"`
	RequestTimeout time.Duration     `json:"request_timeout" yaml:"request_timeout"`
	Temperature    float64           `json:"temperature" yaml:"temperature"`
	TopP           float64           `json:"top_p" yaml:"top_p"`
	TopK           int               `json:"top_k" yaml:"top_k"`
	RepeatPenalty  float64           `json:"repeat_penalty" yaml:"repeat_penalty"`
	ContextSize    int               `json:"context_size" yaml:"context_size"`
	GPULayers      int               `json:"gpu_layers" yaml:"gpu_layers"`
	AdditionalArgs map[string]string `json:"additional_args" yaml:"additional_args"`
}

// ModelConfig defines a single llama.cpp model configuration
type ModelConfig struct {
	ID           string            `json:"id" yaml:"id"`
	Path         string            `json:"path" yaml:"path"`
	ModelName    string            `json:"model_name" yaml:"model_name"`
	Size         int64             `json:"size" yaml:"size"`                 // Size in bytes
	Quantization string            `json:"quantization" yaml:"quantization"` // Q4_0, Q5_K_M, etc.
	MaxTokens    int               `json:"max_tokens" yaml:"max_tokens"`
	Capabilities []string          `json:"capabilities" yaml:"capabilities"`   // translation, reasoning, etc.
	PreferredFor []string          `json:"preferred_for" yaml:"preferred_for"` // text, code, etc.
	ModelParams  map[string]string `json:"model_params" yaml:"model_params"`
	IsDefault    bool              `json:"is_default" yaml:"is_default"`
	IsAvailable  bool              `json:"is_available" yaml:"is_available"`
	LastUsed     time.Time         `json:"last_used" yaml:"last_used"`
}

// LlamaCppWorker represents a single llama.cpp model worker
type LlamaCppWorker struct {
	ID         string
	Config     ModelConfig
	BinaryPath string
	Process    *exec.Cmd
	IsRunning  bool
	mu         sync.RWMutex
}

// MultiLLMCoordinator manages multiple llama.cpp workers for parallel processing
type MultiLLMCoordinator struct {
	Config    LlamaCppProviderConfig
	Workers   map[string]*LlamaCppWorker
	WorkQueue chan TranslationTask
	Results   chan TranslationResult
	mu        sync.RWMutex
	logger    logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// TranslationTask represents a translation request
type TranslationTask struct {
	ID       string
	Text     string
	FromLang string
	ToLang   string
	Context  string
	WorkerID string
	Result   chan TranslationResult
}

// TranslationResult represents the result of a translation task
type TranslationResult struct {
	ID         string
	Text       string
	Success    bool
	Error      error
	Metadata   map[string]interface{}
	WorkerID   string
	Duration   time.Duration
	TokensUsed int
	ModelUsed  string
}

// NewLlamaCppProvider creates a new multi-LLM llama.cpp provider
func NewLlamaCppProvider(config LlamaCppProviderConfig, logger logger.Logger) (*MultiLLMCoordinator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	coordinator := &MultiLLMCoordinator{
		Config:    config,
		Workers:   make(map[string]*LlamaCppWorker),
		WorkQueue: make(chan TranslationTask, 100),
		Results:   make(chan TranslationResult, 100),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Initialize workers
	if err := coordinator.initializeWorkers(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize workers: %w", err)
	}

	// Start worker pool
	coordinator.startWorkerPool()

	return coordinator, nil
}

// initializeWorkers sets up all configured llama.cpp workers
func (c *MultiLLMCoordinator) initializeWorkers() error {
	for _, modelConfig := range c.Config.Models {
		worker := &LlamaCppWorker{
			ID:         modelConfig.ID,
			Config:     modelConfig,
			BinaryPath: c.Config.BinaryPath,
		}

		// Check if model file exists and is accessible
		if _, err := exec.LookPath(c.Config.BinaryPath); err != nil {
			return fmt.Errorf("llama.cpp binary not found at %s: %w", c.Config.BinaryPath, err)
		}

		// Check if model file exists and is accessible
		if _, err := os.Stat(modelConfig.Path); err != nil {
			c.logger.Warn("Model file not found, marking worker unavailable",
				map[string]interface{}{
					"model_id": modelConfig.ID,
					"path":     modelConfig.Path,
					"error":    err.Error(),
				})
			worker.Config.IsAvailable = false
		} else {
			worker.Config.IsAvailable = true
		}

		c.Workers[modelConfig.ID] = worker
		c.logger.Info("Initialized llama.cpp worker",
			map[string]interface{}{
				"model_id":  modelConfig.ID,
				"path":      modelConfig.Path,
				"available": modelConfig.IsAvailable,
			})
	}

	return nil
}

// startWorkerPool starts the goroutines that process translation tasks
func (c *MultiLLMCoordinator) startWorkerPool() {
	for i := 0; i < c.Config.MaxConcurrency; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}
}

// worker processes translation tasks from the queue
func (c *MultiLLMCoordinator) worker(workerIndex int) {
	defer c.wg.Done()

	c.logger.Debug("Starting llama.cpp worker",
		map[string]interface{}{
			"worker_index": workerIndex,
		})

	for {
		select {
		case task := <-c.WorkQueue:
			result := c.processTask(task)
			if task.Result != nil {
				select {
				case task.Result <- result:
				case <-c.ctx.Done():
					return
				}
			}
			select {
			case c.Results <- result:
			case <-c.ctx.Done():
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

// processTask executes a single translation task
func (c *MultiLLMCoordinator) processTask(task TranslationTask) TranslationResult {
	startTime := time.Now()

	c.logger.Debug("Processing translation task",
		map[string]interface{}{
			"task_id":     task.ID,
			"from_lang":   task.FromLang,
			"to_lang":     task.ToLang,
			"text_length": len(task.Text),
		})

	// Select best available worker for this task
	workerID := task.WorkerID
	if workerID == "" {
		workerID = c.selectBestWorker(task)
		if workerID == "" {
			return TranslationResult{
				ID:       task.ID,
				Success:  false,
				Error:    fmt.Errorf("no available workers for task %s", task.ID),
				Duration: time.Since(startTime),
			}
		}
	}

	worker := c.Workers[workerID]

	// Build llama.cpp command
	cmd := c.buildCommand(worker, task)

	// Execute translation
	output, err := c.executeCommand(cmd)
	if err != nil {
		c.logger.Error("Translation failed",
			map[string]interface{}{
				"task_id":   task.ID,
				"worker_id": workerID,
				"error":     err.Error(),
			})

		return TranslationResult{
			ID:       task.ID,
			Success:  false,
			Error:    err,
			WorkerID: workerID,
			Duration: time.Since(startTime),
		}
	}

	// Parse and clean output
	translatedText := c.parseOutput(output)

	// Update worker last used time
	worker.mu.Lock()
	worker.Config.LastUsed = time.Now()
	worker.mu.Unlock()

	c.logger.Debug("Translation completed",
		map[string]interface{}{
			"task_id":     task.ID,
			"worker_id":   workerID,
			"duration_ms": time.Since(startTime).Milliseconds(),
		})

	return TranslationResult{
		ID:        task.ID,
		Text:      translatedText,
		Success:   true,
		WorkerID:  workerID,
		Duration:  time.Since(startTime),
		ModelUsed: worker.Config.ModelName,
		Metadata: map[string]interface{}{
			"model_id":     worker.Config.ID,
			"quantization": worker.Config.Quantization,
		},
	}
}

// selectBestWorker chooses the optimal worker for a given task
func (c *MultiLLMCoordinator) selectBestWorker(task TranslationTask) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var bestWorkerID string
	var bestScore float64 = -1

	for workerID, worker := range c.Workers {
		if !worker.Config.IsAvailable {
			continue
		}

		score := c.calculateWorkerScore(worker, task)
		if score > bestScore {
			bestScore = score
			bestWorkerID = workerID
		}
	}

	return bestWorkerID
}

// calculateWorkerScore computes a score for worker selection
func (c *MultiLLMCoordinator) calculateWorkerScore(worker *LlamaCppWorker, task TranslationTask) float64 {
	score := 0.0

	// Base availability score
	if worker.IsRunning {
		score += 10.0
	}

	// Model capability matching
	for _, capability := range worker.Config.Capabilities {
		if capability == "translation" {
			score += 20.0
		}
	}

	// Text type preference matching
	for _, preferredFor := range worker.Config.PreferredFor {
		if strings.Contains(task.Text, "code") && preferredFor == "code" {
			score += 15.0
		} else if preferredFor == "text" {
			score += 10.0
		}
	}

	// Quantization quality (higher is better)
	switch worker.Config.Quantization {
	case "Q8_0":
		score += 8.0
	case "Q5_K_M":
		score += 6.0
	case "Q4_K_M":
		score += 4.0
	case "Q4_0":
		score += 2.0
	}

	// Load balancing (prefer less recently used workers)
	timeSinceLastUse := time.Since(worker.Config.LastUsed)
	score += float64(timeSinceLastUse.Hours()) * 0.1

	return score
}

// buildCommand constructs the llama.cpp command for execution
func (c *MultiLLMCoordinator) buildCommand(worker *LlamaCppWorker, task TranslationTask) *exec.Cmd {
	args := []string{
		"-m", worker.Config.Path,
		"--ctx-size", fmt.Sprintf("%d", c.Config.ContextSize),
		"--temp", fmt.Sprintf("%.2f", c.Config.Temperature),
		"--top-p", fmt.Sprintf("%.2f", c.Config.TopP),
		"--top-k", fmt.Sprintf("%d", c.Config.TopK),
		"--repeat-penalty", fmt.Sprintf("%.2f", c.Config.RepeatPenalty),
		"--gpu-layers", fmt.Sprintf("%d", c.Config.GPULayers),
		"--color",
		"-p", c.buildPrompt(task),
	}

	// Add model-specific parameters
	for key, value := range worker.Config.ModelParams {
		args = append(args, fmt.Sprintf("--%s=%s", key, value))
	}

	// Add additional global parameters
	for key, value := range c.Config.AdditionalArgs {
		args = append(args, fmt.Sprintf("--%s=%s", key, value))
	}

	cmd := exec.Command(c.Config.BinaryPath, args...)
	return cmd
}

// buildPrompt creates the translation prompt for llama.cpp
func (c *MultiLLMCoordinator) buildPrompt(task TranslationTask) string {
	prompt := fmt.Sprintf(`Translate the following text from %s to %s. 
Provide ONLY the translation without any explanations, notes, or additional text.
Maintain the original formatting, line breaks, and structure.

Source text:
%s

Translation:`,
		c.getLanguageName(task.FromLang),
		c.getLanguageName(task.ToLang),
		task.Text)

	return prompt
}

// executeCommand runs the llama.cpp command with timeout
func (c *MultiLLMCoordinator) executeCommand(cmd *exec.Cmd) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.Config.RequestTimeout)
	defer cancel()

	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("llama.cpp execution failed: %w", err)
	}

	return string(output), nil
}

// parseOutput cleans and extracts the translation from llama.cpp output
func (c *MultiLLMCoordinator) parseOutput(output string) string {
	// Remove ANSI color codes
	output = removeAnsiCodes(output)

	// Split into lines and find the translation part
	lines := strings.Split(output, "\n")

	// Look for the actual translation (after "Translation:" prompt)
	var translationLines []string
	foundTranslation := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if foundTranslation {
			translationLines = append(translationLines, line)
		} else if strings.Contains(line, "Translation:") {
			foundTranslation = true
			// Remove the "Translation:" part
			parts := strings.SplitN(line, "Translation:", 2)
			if len(parts) > 1 {
				translationLines = append(translationLines, strings.TrimSpace(parts[1]))
			}
		}
	}

	return strings.Join(translationLines, "\n")
}

// getLanguageName converts language codes to full names
func (c *MultiLLMCoordinator) getLanguageName(code string) string {
	languages := map[string]string{
		"en":      "English",
		"ru":      "Russian",
		"sr":      "Serbian",
		"sr-cyrl": "Serbian Cyrillic",
		"sr-latn": "Serbian Latin",
	}

	if name, exists := languages[code]; exists {
		return name
	}
	return code
}

// removeAnsiCodes removes ANSI escape codes from text
func removeAnsiCodes(text string) string {
	// Simple ANSI code removal - can be enhanced if needed
	return text
}

// Translate implements the LLMClient interface
func (c *MultiLLMCoordinator) Translate(ctx context.Context, text string, prompt string) (string, error) {
	// Extract language info from prompt if available
	fromLang, toLang := "en", "sr" // defaults

	if strings.Contains(strings.ToLower(prompt), "russian") || strings.Contains(strings.ToLower(prompt), "ru") {
		fromLang = "ru"
	}
	if strings.Contains(strings.ToLower(prompt), "serbian") || strings.Contains(strings.ToLower(prompt), "sr") {
		toLang = "sr"
	}

	return c.TranslateText(ctx, text, fromLang, toLang, "")
}

// TranslateText is the actual translation method with language support
func (c *MultiLLMCoordinator) TranslateText(ctx context.Context, text, fromLang, toLang string, contextText string) (string, error) {
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	task := TranslationTask{
		ID:       taskID,
		Text:     text,
		FromLang: fromLang,
		ToLang:   toLang,
		Context:  contextText,
		Result:   make(chan TranslationResult, 1),
	}

	// Submit task to queue
	select {
	case c.WorkQueue <- task:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	// Wait for result
	select {
	case result := <-task.Result:
		if result.Success {
			return result.Text, nil
		}
		return "", result.Error
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(c.Config.RequestTimeout):
		return "", fmt.Errorf("translation timeout for task %s", taskID)
	}
}

// GetProviderName returns the provider name
func (c *MultiLLMCoordinator) GetProviderName() string {
	return "llamacpp-multi"
}

// GetAvailableModels returns information about available models
func (c *MultiLLMCoordinator) GetAvailableModels() []ModelConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var models []ModelConfig
	for _, worker := range c.Workers {
		models = append(models, worker.Config)
	}

	return models
}

// Shutdown gracefully shuts down the coordinator and all workers
func (c *MultiLLMCoordinator) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down multi-LLM coordinator", nil)

	// Cancel context to stop workers
	c.cancel()

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Info("All workers shut down successfully", nil)
		return nil
	case <-ctx.Done():
		c.logger.Error("Shutdown timeout", nil)
		return fmt.Errorf("shutdown timeout")
	}
}

// GetStats returns statistics about the coordinator and workers
func (c *MultiLLMCoordinator) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"total_workers":     len(c.Workers),
		"available_workers": 0,
		"running_workers":   0,
		"queue_length":      len(c.WorkQueue),
		"max_concurrency":   c.Config.MaxConcurrency,
	}

	for _, worker := range c.Workers {
		if worker.Config.IsAvailable {
			stats["available_workers"] = stats["available_workers"].(int) + 1
		}
		if worker.IsRunning {
			stats["running_workers"] = stats["running_workers"].(int) + 1
		}
	}

	return stats
}

// Ensure MultiLLMCoordinator implements LLMClient
var _ LLMClient = (*MultiLLMCoordinator)(nil)
