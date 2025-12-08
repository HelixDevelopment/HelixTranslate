package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/fb2"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/translator/llm"
	"digital.vasic.translator/pkg/version"
	"digital.vasic.translator/pkg/websocket"
)

// SSHTranslationSystem manages the complete SSH-based translation workflow
type SSHTranslationSystem struct {
	logger           logger.Logger
	sshWorker        *sshworker.SSHWorker
	versionManager   *version.CodebaseHasher
	eventManager     *events.Manager
	wsHub            *websocket.Hub
	config           SystemConfig
	workflowID       string
}

// SystemConfig holds system configuration
type SystemConfig struct {
	SSH              sshworker.SSHWorkerConfig
	Translation      TranslationConfig
	Workspace        string
	EnableMonitoring bool
}

// TranslationConfig holds translation configuration
type TranslationConfig struct {
	FromLang     string
	ToLang       string
	ChunkSize    int
	MaxRetries   int
	LLMProviders []LLMProviderConfig
}

// LLMProviderConfig holds LLM provider configuration
type LLMProviderConfig struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	BinaryPath     string            `json:"binary_path"`
	ModelPath      string            `json:"model_path"`
	ModelName      string            `json:"model_name"`
	MaxTokens      int               `json:"max_tokens"`
	Temperature    float64           `json:"temperature"`
	AdditionalArgs map[string]string `json:"additional_args"`
}

// TranslationProgress tracks translation progress
type TranslationProgress struct {
	Phase           string    `json:"phase"`
	CurrentStep     int       `json:"current_step"`
	TotalSteps      int       `json:"total_steps"`
	Message         string    `json:"message"`
	CurrentFile     string    `json:"current_file"`
	Timestamp       time.Time `json:"timestamp"`
	BytesProcessed  int64     `json:"bytes_processed"`
	TotalBytes      int64     `json:"total_bytes"`
}

// WorkflowStep represents a step in the translation workflow
type WorkflowStep struct {
	Name        string
	Handler     func(context.Context, *WorkflowContext) error
	Description string
}

// WorkflowContext contains context for workflow execution
type WorkflowContext struct {
	InputFile      string
	OutputFile     string
	Workspace      string
	Progress       *TranslationProgress
	TempFiles      []string
	Metadata       map[string]interface{}
}

func main() {
	// Initialize logger
	logFile := filepath.Join("logs", "ssh_translation.log")
	os.MkdirAll("logs", 0755)
	
	logger := logger.NewLogger(logger.LoggerConfig{
		Level:      "info",
		Output:     "file",
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     30, // days
	})

	// Generate unique workflow ID
	workflowID := generateWorkflowID()

	// Initialize event manager and WebSocket hub if monitoring is enabled
	var eventManager *events.Manager
	var wsHub *websocket.Hub
	
	eventManager = events.NewEventManager(logger)
	wsHub = websocket.NewHub(logger)
	go wsHub.Start()

	// System configuration
	config := SystemConfig{
		SSH: sshworker.SSHWorkerConfig{
			Host:              "thinker.local",
			Username:          "milosvasic",
			Password:          "WhiteSnake8587",
			Port:              22,
			RemoteDir:         "/tmp/translation-workspace",
			ConnectionTimeout: 30 * time.Second,
			CommandTimeout:    300 * time.Second,
		},
		Translation: TranslationConfig{
			FromLang:   "russian",
			ToLang:     "serbian-cyrillic",
			ChunkSize:  1000,
			MaxRetries: 3,
			LLMProviders: []LLMProviderConfig{
				{
					Name:       "llama.cpp-main",
					Type:       "llamacpp",
					BinaryPath: "/usr/local/bin/llama.cpp",
					ModelPath:  "/models/translation-model.gguf",
					ModelName:  "translation-model",
					MaxTokens:  2048,
					Temperature: 0.3,
				},
				{
					Name:       "llama.cpp-secondary",
					Type:       "llamacpp",
					BinaryPath: "/usr/local/bin/llama.cpp",
					ModelPath:  "/models/translation-model-v2.gguf",
					ModelName:  "translation-model-v2",
					MaxTokens:  2048,
					Temperature: 0.4,
				},
			},
		},
		Workspace:        "/tmp/translation-workspace",
		EnableMonitoring: true,
	}

	// Create translation system
	system, err := NewSSHTranslationSystem(config, logger, eventManager, wsHub, workflowID)
	if err != nil {
		logger.Error("Failed to create translation system", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Start WebSocket monitoring
	if config.EnableMonitoring {
		go func() {
			if err := system.StartMonitoringServer(":8090"); err != nil {
				logger.Error("Failed to start monitoring server", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}()
		logger.Info("WebSocket monitoring server started on :8090", nil)
	}

	// Define input file
	inputFile := "internal/materials/books/book1.fb2"

	// Execute translation workflow
	logger.Info("Starting SSH-based translation workflow", map[string]interface{}{
		"workflow_id": workflowID,
		"input_file":  inputFile,
		"target_lang": config.Translation.ToLang,
	})

	err = system.ExecuteTranslationWorkflow(context.Background(), inputFile)
	if err != nil {
		logger.Error("Translation workflow failed", map[string]interface{}{
			"workflow_id": workflowID,
			"error":       err.Error(),
		})
		
		// Send failure event
		eventManager.SendEvent(events.Event{
			Type:      "workflow_failed",
			Data:      map[string]interface{}{"error": err.Error()},
			Timestamp: time.Now(),
		})
		
		os.Exit(1)
	}

	logger.Info("Translation workflow completed successfully", map[string]interface{}{
		"workflow_id": workflowID,
		"input_file":  inputFile,
	})

	// Send completion event
	eventManager.SendEvent(events.Event{
		Type:      "workflow_completed",
		Data:      map[string]interface{}{"workflow_id": workflowID},
		Timestamp: time.Now(),
	})

	// Keep the monitoring server running for a while
	logger.Info("Translation completed. Monitoring server will run for 2 minutes...", nil)
	time.Sleep(2 * time.Minute)
}

// NewSSHTranslationSystem creates a new SSH translation system
func NewSSHTranslationSystem(config SystemConfig, logger logger.Logger, eventManager *events.Manager, wsHub *websocket.Hub, workflowID string) (*SSHTranslationSystem, error) {
	// Create SSH worker
	sshWorker, err := sshworker.NewSSHWorker(config.SSH, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH worker: %w", err)
	}

	// Create version manager
	versionManager := version.NewCodebaseHasher()

	return &SSHTranslationSystem{
		logger:         logger,
		sshWorker:      sshWorker,
		versionManager: versionManager,
		eventManager:   eventManager,
		wsHub:          wsHub,
		config:         config,
		workflowID:     workflowID,
	}, nil
}

// ExecuteTranslationWorkflow executes the complete translation workflow
func (s *SSHTranslationSystem) ExecuteTranslationWorkflow(ctx context.Context, inputFile string) error {
	// Validate input file
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", inputFile)
	}

	// Create workspace
	s.createWorkspace()

	// Initialize workflow context
	workflowCtx := &WorkflowContext{
		InputFile:  inputFile,
		Workspace:  s.config.Workspace,
		Progress:   &TranslationProgress{},
		Metadata:   make(map[string]interface{}),
		TempFiles:  []string{},
	}

	// Define workflow steps
	steps := []WorkflowStep{
		{
			Name:        "connect_and_sync",
			Handler:     s.connectAndSyncCodebase,
			Description: "Connect to SSH worker and synchronize codebase",
		},
		{
			Name:        "convert_fb2_to_markdown",
			Handler:     s.convertFB2ToMarkdown,
			Description: "Convert FB2 file to Markdown",
		},
		{
			Name:        "transfer_files",
			Handler:     s.transferFilesToRemote,
			Description: "Transfer files to remote worker",
		},
		{
			Name:        "translate_markdown",
			Handler:     s.translateMarkdownOnRemote,
			Description: "Translate Markdown using remote LLM",
		},
		{
			Name:        "convert_translated_to_epub",
			Handler:     s.convertTranslatedToEPUB,
			Description: "Convert translated Markdown to EPUB",
		},
		{
			Name:        "verify_output",
			Handler:     s.verifyOutput,
			Description: "Verify output files",
		},
		{
			Name:        "cleanup",
			Handler:     s.cleanup,
			Description: "Clean up temporary files",
		},
	}

	// Execute workflow steps
	for i, step := range steps {
		s.updateProgress(step.Name, i, len(steps), step.Description, "")
		
		s.logger.Info("Executing workflow step", map[string]interface{}{
			"step":        step.Name,
			"description": step.Description,
			"workflow_id": s.workflowID,
		})

		// Send progress event
		s.eventManager.SendEvent(events.Event{
			Type: "step_started",
			Data: map[string]interface{}{
				"step":        step.Name,
				"description": step.Description,
				"progress":    s.Progress(),
			},
			Timestamp: time.Now(),
		})

		// Execute step with timeout
		stepCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()

		if err := step.Handler(stepCtx, workflowCtx); err != nil {
			return fmt.Errorf("step '%s' failed: %w", step.Name, err)
		}

		// Send completion event
		s.eventManager.SendEvent(events.Event{
			Type: "step_completed",
			Data: map[string]interface{}{
				"step":        step.Name,
				"description": step.Description,
				"progress":    s.Progress(),
			},
			Timestamp: time.Now(),
		})
	}

	return nil
}

// connectAndSyncCodebase connects to SSH worker and synchronizes codebase
func (s *SSHTranslationSystem) connectAndSyncCodebase(ctx context.Context, workflowCtx *WorkflowContext) error {
	// Connect to SSH worker
	s.logger.Info("Connecting to SSH worker", map[string]interface{}{
		"host": s.config.SSH.Host,
		"port": s.config.SSH.Port,
	})

	if err := s.sshWorker.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to SSH worker: %w", err)
	}

	// Calculate local codebase hash
	s.logger.Info("Calculating local codebase hash", nil)
	localHash, err := s.versionManager.CalculateHash()
	if err != nil {
		return fmt.Errorf("failed to calculate local codebase hash: %w", err)
	}

	s.logger.Info("Local codebase hash calculated", map[string]interface{}{
		"hash": localHash,
	})

	// Get remote codebase hash
	s.logger.Info("Getting remote codebase hash", nil)
	remoteHash, err := s.getRemoteCodebaseHash(ctx)
	if err != nil {
		return fmt.Errorf("failed to get remote codebase hash: %w", err)
	}

	s.logger.Info("Remote codebase hash calculated", map[string]interface{}{
		"hash": remoteHash,
	})

	// Compare hashes and sync if needed
	if localHash != remoteHash {
		s.logger.Info("Codebase hashes differ, synchronizing...", map[string]interface{}{
			"local_hash":  localHash,
			"remote_hash": remoteHash,
		})

		if err := s.syncCodebase(ctx); err != nil {
			return fmt.Errorf("failed to sync codebase: %w", err)
		}

		// Verify synchronization
		remoteHash, err = s.getRemoteCodebaseHash(ctx)
		if err != nil {
			return fmt.Errorf("failed to verify codebase synchronization: %w", err)
		}

		if localHash != remoteHash {
			return fmt.Errorf("codebase synchronization failed: hashes still differ")
		}

		s.logger.Info("Codebase synchronized successfully", nil)
	} else {
		s.logger.Info("Codebase hashes match, no synchronization needed", nil)
	}

	// Setup LLM workers on remote
	if err := s.setupRemoteLLMWorkers(ctx); err != nil {
		return fmt.Errorf("failed to setup remote LLM workers: %w", err)
	}

	return nil
}

// getRemoteCodebaseHash gets the codebase hash from remote worker
func (s *SSHTranslationSystem) getRemoteCodebaseHash(ctx context.Context) (string, error) {
	// Create remote hash calculation script
	script := fmt.Sprintf(`
#!/bin/bash
cd %s
# Create a temporary hash calculation script
cat > calculate_hash.go << 'EOF'
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type CodebaseHasher struct {
	RelevantDirectories []string
	RelevantExtensions  []string
	ExcludePatterns     []string
}

func NewCodebaseHasher() *CodebaseHasher {
	return &CodebaseHasher{
		RelevantDirectories: []string{
			"cmd",
			"pkg", 
			"internal",
			"scripts",
			"docs",
		},
		RelevantExtensions: []string{
			".go",
			".json",
			".yaml",
			".yml",
			".md",
			".sh",
			".txt",
			"Dockerfile",
			"Makefile",
		},
		ExcludePatterns: []string{
			".git",
			"node_modules",
			"__pycache__",
			".DS_Store",
			"*.log",
			"*.tmp",
			"*.pid",
			"coverage*.out",
			"*.test",
			"vendor",
			".env",
			"._*",
		},
	}
}

func (h *CodebaseHasher) CalculateHash() (string, error) {
	hasher := sha256.New()
	
	for _, dir := range h.RelevantDirectories {
		if err := h.processDirectory(hasher, dir); err != nil {
			return "", err
		}
	}
	
	if err := h.processRootFiles(hasher); err != nil {
		return "", err
	}
	
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (h *CodebaseHasher) processDirectory(hasher io.Writer, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			for _, pattern := range h.ExcludePatterns {
				if strings.Contains(path, pattern) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		
		if h.shouldIncludeFile(path) {
			return h.addFileToHash(hasher, path, info)
		}
		
		return nil
	})
}

func (h *CodebaseHasher) processRootFiles(hasher io.Writer) error {
	entries, err := os.ReadDir(".")
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		path := entry.Name()
		if h.shouldIncludeFile(path) && !h.isDirectoryInRelevantList(path) {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := h.addFileToHash(hasher, path, info); err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (h *CodebaseHasher) shouldIncludeFile(path string) bool {
	for _, pattern := range h.ExcludePatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}
	
	for _, ext := range h.RelevantExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	
	return false
}

func (h *CodebaseHasher) isDirectoryInRelevantList(path string) bool {
	for _, dir := range h.RelevantDirectories {
		if path == dir {
			return true
		}
	}
	return false
}

func (h *CodebaseHasher) addFileToHash(hasher io.Writer, path string, info os.FileInfo) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fmt.Fprintf(hasher, "file:%%s\\n", path)
	fmt.Fprintf(hasher, "size:%%d\\n", info.Size())
	
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	
	fmt.Fprintf(hasher, "---FILE_SEPARATOR---\\n")
	return nil
}

func main() {
	hasher := NewCodebaseHasher()
	hash, err := hasher.CalculateHash()
	if err != nil {
		fmt.Printf("Error: %%v\\n", err)
		os.Exit(1)
	}
	fmt.Printf("%%s\\n", hash)
}
EOF

# Run the hash calculation
go run calculate_hash.go
rm calculate_hash.go
`, s.config.SSH.RemoteDir)

	// Execute script on remote
	output, err := s.sshWorker.ExecuteCommandWithOutput(ctx, script)
	if err != nil {
		return "", fmt.Errorf("failed to execute remote hash calculation: %w", err)
	}

	// Extract hash from output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "Error:") {
			return strings.TrimSpace(line), nil
		}
	}

	return "", fmt.Errorf("no hash found in remote output")
}

// syncCodebase synchronizes the codebase to remote worker
func (s *SSHTranslationSystem) syncCodebase(ctx context.Context) error {
	s.logger.Info("Starting codebase synchronization", nil)

	// Create archive of relevant files
	archivePath := filepath.Join(s.config.Workspace, "codebase.tar.gz")
	if err := s.createCodebaseArchive(archivePath); err != nil {
		return fmt.Errorf("failed to create codebase archive: %w", err)
	}

	// Transfer archive to remote
	if err := s.sshWorker.TransferFile(ctx, archivePath, filepath.Join(s.config.SSH.RemoteDir, "codebase.tar.gz")); err != nil {
		return fmt.Errorf("failed to transfer codebase archive: %w", err)
	}

	// Extract archive on remote
	extractScript := fmt.Sprintf(`
cd %s
tar -xzf codebase.tar.gz
rm codebase.tar.gz
chmod +x scripts/*.sh
go mod tidy
`, s.config.SSH.RemoteDir)

	_, err := s.sshWorker.ExecuteCommand(ctx, extractScript)
	if err != nil {
		return fmt.Errorf("failed to extract codebase archive: %w", err)
	}

	// Build remote binaries
	buildScript := fmt.Sprintf(`
cd %s
go build -o translator ./cmd/translator
go build -o translator-server ./cmd/server
go build -o markdown-translator ./cmd/markdown-translator
`, s.config.SSH.RemoteDir)

	_, err = s.sshWorker.ExecuteCommand(ctx, buildScript)
	if err != nil {
		return fmt.Errorf("failed to build remote binaries: %w", err)
	}

	// Clean up local archive
	os.Remove(archivePath)

	s.logger.Info("Codebase synchronization completed", nil)
	return nil
}

// createCodebaseArchive creates an archive of relevant codebase files
func (s *SSHTranslationSystem) createCodebaseArchive(outputPath string) error {
	// Create tar command
	cmd := fmt.Sprintf("tar -czf %s cmd pkg internal scripts docs go.mod go.sum Makefile VERSION README.md", outputPath)
	
	// Execute tar command
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	// Use shell to create archive
	if err := exec.Command("bash", "-c", cmd).Run(); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}

	return nil
}

// setupRemoteLLMWorkers sets up LLM workers on remote host
func (s *SSHTranslationSystem) setupRemoteLLMWorkers(ctx context.Context) error {
	s.logger.Info("Setting up remote LLM workers", nil)

	// Create LLM setup script
	setupScript := fmt.Sprintf(`
cd %s

# Check if llama.cpp binaries exist
if [ ! -f "/usr/local/bin/llama.cpp" ]; then
    echo "ERROR: llama.cpp binary not found at /usr/local/bin/llama.cpp"
    exit 1
fi

# Check if models exist
for model in %s; do
    if [ ! -f "$model" ]; then
        echo "WARNING: Model file not found: $model"
    else
        echo "Model found: $model"
    fi
done

# Create model directory if it doesn't exist
mkdir -p models

# Test llama.cpp functionality
echo "Testing llama.cpp functionality..."
/usr/local/bin/llama.cpp --help > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "llama.cpp is accessible"
else
    echo "ERROR: llama.cpp is not accessible"
    exit 1
fi

echo "LLM worker setup completed"
`, s.config.SSH.RemoteDir, s.getRemoteModelPaths())

	// Execute setup script
	output, err := s.sshWorker.ExecuteCommand(ctx, setupScript)
	if err != nil {
		return fmt.Errorf("failed to setup remote LLM workers: %w", err)
	}

	s.logger.Info("Remote LLM workers setup completed", map[string]interface{}{
		"output": output,
	})

	return nil
}

// getRemoteModelPaths returns the remote model paths
func (s *SSHTranslationSystem) getRemoteModelPaths() string {
	var paths []string
	for _, provider := range s.config.Translation.LLMProviders {
		paths = append(paths, provider.ModelPath)
	}
	return strings.Join(paths, " ")
}

// convertFB2ToMarkdown converts FB2 file to Markdown
func (s *SSHTranslationSystem) convertFB2ToMarkdown(ctx context.Context, workflowCtx *WorkflowContext) error {
	s.logger.Info("Converting FB2 to Markdown", map[string]interface{}{
		"input_file": workflowCtx.InputFile,
	})

	// Create FB2 converter
	converter := fb2.NewMarkdownConverter(s.logger)

	// Generate output path
	inputFile := workflowCtx.InputFile
	markdownFile := filepath.Join(
		filepath.Dir(inputFile),
		strings.TrimSuffix(filepath.Base(inputFile), filepath.Ext(inputFile))+".md",
	)

	// Convert FB2 to Markdown
	if err := converter.ConvertToMarkdown(inputFile, markdownFile); err != nil {
		return fmt.Errorf("failed to convert FB2 to Markdown: %w", err)
	}

	// Add to temp files for cleanup
	workflowCtx.TempFiles = append(workflowCtx.TempFiles, markdownFile)

	// Store markdown file for next step
	workflowCtx.Metadata["markdown_file"] = markdownFile

	// Verify the conversion
	if err := s.verifyMarkdownFile(markdownFile); err != nil {
		return fmt.Errorf("markdown conversion verification failed: %w", err)
	}

	s.logger.Info("FB2 converted to Markdown successfully", map[string]interface{}{
		"output_file": markdownFile,
		"file_size":   s.getFileSize(markdownFile),
	})

	return nil
}

// transferFilesToRemote transfers necessary files to remote worker
func (s *SSHTranslationSystem) transferFilesToRemote(ctx context.Context, workflowCtx *WorkflowContext) error {
	s.logger.Info("Transferring files to remote worker", nil)

	// Get markdown file from metadata
	markdownFile, ok := workflowCtx.Metadata["markdown_file"].(string)
	if !ok {
		return fmt.Errorf("markdown file not found in workflow metadata")
	}

	// Transfer markdown file to remote
	remoteMarkdownFile := filepath.Join(s.config.SSH.RemoteDir, "input.md")
	if err := s.sshWorker.TransferFile(ctx, markdownFile, remoteMarkdownFile); err != nil {
		return fmt.Errorf("failed to transfer markdown file: %w", err)
	}

	// Store remote file path
	workflowCtx.Metadata["remote_markdown_file"] = remoteMarkdownFile

	s.logger.Info("Files transferred to remote worker successfully", map[string]interface{}{
		"local_file":  markdownFile,
		"remote_file": remoteMarkdownFile,
	})

	return nil
}

// translateMarkdownOnRemote translates markdown using remote LLM
func (s *SSHTranslationSystem) translateMarkdownOnRemote(ctx context.Context, workflowCtx *WorkflowContext) error {
	s.logger.Info("Translating markdown using remote LLM", nil)

	// Get remote markdown file
	remoteMarkdownFile, ok := workflowCtx.Metadata["remote_markdown_file"].(string)
	if !ok {
		return fmt.Errorf("remote markdown file not found in workflow metadata")
	}

	// Create translation script
	translationScript := s.createTranslationScript(remoteMarkdownFile)

	// Execute translation on remote
	output, err := s.sshWorker.ExecuteCommand(ctx, translationScript)
	if err != nil {
		return fmt.Errorf("failed to execute remote translation: %w", err)
	}

	// Define remote translated file path
	remoteTranslatedFile := filepath.Join(s.config.SSH.RemoteDir, "translated.md")
	workflowCtx.Metadata["remote_translated_file"] = remoteTranslatedFile

	s.logger.Info("Remote translation completed", map[string]interface{}{
		"output": output,
		"remote_file": remoteTranslatedFile,
	})

	return nil
}

// createTranslationScript creates the translation script for remote execution
func (s *SSHTranslationSystem) createTranslationScript(inputFile string) string {
	var modelConfigs []string
	for i, provider := range s.config.Translation.LLMProviders {
		config := fmt.Sprintf(`
{
    "id": "worker_%d",
    "binary_path": "%s",
    "model_path": "%s",
    "model_name": "%s",
    "max_tokens": %d,
    "temperature": %.2f,
    "additional_args": %s
}`, i, provider.BinaryPath, provider.ModelPath, provider.ModelName, provider.MaxTokens, provider.Temperature, "{}")
		modelConfigs = append(modelConfigs, config)
	}

	script := fmt.Sprintf(`
cd %s

# Create translation configuration
cat > translation_config.json << 'EOF'
{
    "input_file": "%s",
    "output_file": "translated.md",
    "from_lang": "%s",
    "to_lang": "%s",
    "chunk_size": %d,
    "llm_providers": [%s]
}
EOF

# Create translation script
cat > translate.py << 'EOF'
import json
import subprocess
import sys
import time

def translate_with_llamacpp(text, binary_path, model_path, model_name, max_tokens, temperature):
    """Translate text using llama.cpp"""
    try:
        # Create prompt
        prompt = f"""Translate the following text from Russian to Serbian Cyrillic. 
Provide ONLY the translation without any explanations, notes, or additional text.
Maintain the original formatting, line breaks, and structure.

Source text:
{text}

Translation:"""
        
        # Run llama.cpp
        cmd = [
            binary_path,
            "-m", model_path,
            "-p", prompt,
            "-n", str(max_tokens),
            "--temp", str(temperature),
            "-c", "2048",
            "--ctx-size", "2048"
        ]
        
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        
        if result.returncode != 0:
            print(f"llama.cpp error: {result.stderr}", file=sys.stderr)
            return None
            
        # Extract translation from output
        output = result.stdout.strip()
        
        # Find the translation part (after "Translation:")
        if "Translation:" in output:
            translation = output.split("Translation:", 1)[1].strip()
            return translation
        else:
            return output
            
    except Exception as e:
        print(f"Translation error: {e}", file=sys.stderr)
        return None

def main():
    # Load configuration
    with open('translation_config.json', 'r') as f:
        config = json.load(f)
    
    # Read input file
    with open(config['input_file'], 'r', encoding='utf-8') as f:
        content = f.read()
    
    print(f"Starting translation of {len(content)} characters...")
    
    # Split content into chunks
    chunk_size = config['chunk_size']
    chunks = [content[i:i+chunk_size] for i in range(0, len(content), chunk_size)]
    
    translated_chunks = []
    
    # Process each chunk
    for i, chunk in enumerate(chunks):
        print(f"Translating chunk {i+1}/{len(chunks)} ({len(chunk)} chars)...")
        
        # Try each provider until success
        translated = None
        for provider in config['llm_providers']:
            try:
                translated = translate_with_llamacpp(
                    chunk,
                    provider['binary_path'],
                    provider['model_path'],
                    provider['model_name'],
                    provider['max_tokens'],
                    provider['temperature']
                )
                
                if translated:
                    print(f"Successfully translated chunk {i+1} using {provider['model_name']}")
                    break
                else:
                    print(f"Failed to translate chunk {i+1} using {provider['model_name']}")
                    
            except Exception as e:
                print(f"Error with provider {provider['model_name']}: {e}")
                continue
        
        if not translated:
            print(f"Failed to translate chunk {i+1} with all providers", file=sys.stderr)
            translated = chunk  # Use original text as fallback
        
        translated_chunks.append(translated)
        
        # Small delay to prevent overwhelming the system
        time.sleep(0.1)
    
    # Write translated content
    translated_content = ''.join(translated_chunks)
    
    with open(config['output_file'], 'w', encoding='utf-8') as f:
        f.write(translated_content)
    
    print(f"Translation completed. Output written to {config['output_file']}")
    print(f"Original length: {len(content)} chars")
    print(f"Translated length: {len(translated_content)} chars")

if __name__ == "__main__":
    main()
EOF

# Run translation
python3 translate.py

# Cleanup
rm translation_config.json translate.py
`, s.config.SSH.RemoteDir, inputFile, s.config.Translation.FromLang, s.config.Translation.ToLang, s.config.Translation.ChunkSize, strings.Join(modelConfigs, ","))

	return script
}

// convertTranslatedToEPUB converts translated markdown to EPUB
func (s *SSHTranslationSystem) convertTranslatedToEPUB(ctx context.Context, workflowCtx *WorkflowContext) error {
	s.logger.Info("Converting translated markdown to EPUB", nil)

	// Get remote translated file
	remoteTranslatedFile, ok := workflowCtx.Metadata["remote_translated_file"].(string)
	if !ok {
		return fmt.Errorf("remote translated file not found in workflow metadata")
	}

	// Transfer translated file back from remote
	localTranslatedFile := filepath.Join(
		filepath.Dir(workflowCtx.InputFile),
		strings.TrimSuffix(filepath.Base(workflowCtx.InputFile), filepath.Ext(workflowCtx.InputFile))+"_translated.md",
	)

	if err := s.sshWorker.TransferFileFromRemote(ctx, remoteTranslatedFile, localTranslatedFile); err != nil {
		return fmt.Errorf("failed to transfer translated file from remote: %w", err)
	}

	// Add to temp files for cleanup
	workflowCtx.TempFiles = append(workflowCtx.TempFiles, localTranslatedFile)

	// Create EPUB output path
	epubFile := filepath.Join(
		filepath.Dir(workflowCtx.InputFile),
		strings.TrimSuffix(filepath.Base(workflowCtx.InputFile), filepath.Ext(workflowCtx.InputFile))+"_sr.epub",
	)

	// Create markdown workflow for EPUB conversion
	translateFunc := func(text string) (string, error) {
		return text, nil // No translation needed, just conversion
	}

	translator := markdown.NewMarkdownTranslator(translateFunc)

	// Convert markdown to EPUB
	if err := translator.ConvertMarkdownToEPUB(localTranslatedFile, epubFile); err != nil {
		return fmt.Errorf("failed to convert markdown to EPUB: %w", err)
	}

	// Store final output file
	workflowCtx.OutputFile = epubFile
	workflowCtx.Metadata["translated_epub"] = epubFile

	s.logger.Info("Translated markdown converted to EPUB successfully", map[string]interface{}{
		"input_file":  localTranslatedFile,
		"output_file": epubFile,
		"file_size":   s.getFileSize(epubFile),
	})

	return nil
}

// verifyOutput verifies the output files
func (s *SSHTranslationSystem) verifyOutput(ctx context.Context, workflowCtx *WorkflowContext) error {
	s.logger.Info("Verifying output files", nil)

	// Get files from metadata
	translatedEPUB, ok := workflowCtx.Metadata["translated_epub"].(string)
	if !ok {
		return fmt.Errorf("translated EPUB file not found in workflow metadata")
	}

	translatedMarkdown, ok := workflowCtx.Metadata["remote_translated_file"].(string)
	if !ok {
		return fmt.Errorf("translated markdown file not found in workflow metadata")
	}

	// Transfer translated markdown for verification
	localTranslatedMarkdown := filepath.Join(
		filepath.Dir(workflowCtx.InputFile),
		strings.TrimSuffix(filepath.Base(workflowCtx.InputFile), filepath.Ext(workflowCtx.InputFile))+"_translated.md",
	)

	if err := s.sshWorker.TransferFileFromRemote(ctx, translatedMarkdown, localTranslatedMarkdown); err != nil {
		return fmt.Errorf("failed to transfer translated markdown for verification: %w", err)
	}

	// Verify EPUB file
	if err := s.verifyEPUBFile(translatedEPUB, s.config.Translation.ToLang); err != nil {
		return fmt.Errorf("EPUB verification failed: %w", err)
	}

	// Verify translated markdown
	if err := s.verifyTranslatedMarkdown(localTranslatedMarkdown); err != nil {
		return fmt.Errorf("translated markdown verification failed: %w", err)
	}

	s.logger.Info("Output files verified successfully", map[string]interface{}{
		"epub_file":    translatedEPUB,
		"markdown_file": localTranslatedMarkdown,
	})

	return nil
}

// verifyMarkdownFile verifies a markdown file is valid
func (s *SSHTranslationSystem) verifyMarkdownFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot access markdown file: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("markdown file is empty")
	}

	// Read and verify content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read markdown file: %w", err)
	}

	contentStr := string(content)
	if len(strings.TrimSpace(contentStr)) < 100 {
		return fmt.Errorf("markdown file content too short")
	}

	return nil
}

// verifyEPUBFile verifies an EPUB file is valid and in correct language
func (s *SSHTranslationSystem) verifyEPUBFile(filePath, expectedLang string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot access EPUB file: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("EPUB file is empty")
	}

	// Basic EPUB structure validation
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("cannot open EPUB file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to check EPUB magic
	buffer := make([]byte, 4)
	_, err = file.Read(buffer)
	if err != nil {
		return fmt.Errorf("cannot read EPUB file: %w", err)
	}

	// EPUB files should start with PK (ZIP format)
	if string(buffer) != "PK" {
		return fmt.Errorf("file does not appear to be a valid EPUB (missing ZIP header)")
	}

	s.logger.Info("EPUB file validation passed", map[string]interface{}{
		"file_path": filePath,
		"file_size": info.Size(),
	})

	return nil
}

// verifyTranslatedMarkdown verifies translated markdown content
func (s *SSHTranslationSystem) verifyTranslatedMarkdown(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read translated markdown: %w", err)
	}

	contentStr := string(content)
	if len(strings.TrimSpace(contentStr)) < 100 {
		return fmt.Errorf("translated markdown content too short")
	}

	// Check for Cyrillic characters (Serbian Cyrillic)
	cyrillicCount := 0
	totalChars := 0
	
	for _, r := range contentStr {
		if r >= 0x0400 && r <= 0x04FF { // Cyrillic range
			cyrillicCount++
		}
		if r >= 32 && r <= 126 { // ASCII printable
			totalChars++
		}
	}

	// At least 30% of non-ASCII characters should be Cyrillic for Serbian translation
	if cyrillicCount > 0 && totalChars > 0 {
		cyrillicRatio := float64(cyrillicCount) / float64(cyrillicCount + totalChars)
		if cyrillicRatio < 0.3 {
			s.logger.Warning("Low Cyrillic character ratio detected", map[string]interface{}{
				"cyrillic_count": cyrillicCount,
				"total_chars":    totalChars,
				"ratio":          cyrillicRatio,
			})
		}
	}

	s.logger.Info("Translated markdown validation passed", map[string]interface{}{
		"file_path":      filePath,
		"cyrillic_chars": cyrillicCount,
		"content_length": len(contentStr),
	})

	return nil
}

// cleanup cleans up temporary files
func (s *SSHTranslationSystem) cleanup(ctx context.Context, workflowCtx *WorkflowContext) error {
	s.logger.Info("Cleaning up temporary files", map[string]interface{}{
		"temp_files_count": len(workflowCtx.TempFiles),
	})

	// Clean up local temp files
	for _, tempFile := range workflowCtx.TempFiles {
		if err := os.Remove(tempFile); err != nil && !os.IsNotExist(err) {
			s.logger.Warning("Failed to remove temp file", map[string]interface{}{
				"file":  tempFile,
				"error": err.Error(),
			})
		}
	}

	// Clean up remote temp files
	cleanupScript := fmt.Sprintf(`
cd %s
rm -f input.md translated.md
rm -f *.log *.tmp
`, s.config.SSH.RemoteDir)

	_, err := s.sshWorker.ExecuteCommand(ctx, cleanupScript)
	if err != nil {
		s.logger.Warning("Failed to cleanup remote temp files", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Close SSH connection
	s.sshWorker.Disconnect()

	s.logger.Info("Cleanup completed", nil)
	return nil
}

// updateProgress updates the progress tracking
func (s *SSHTranslationSystem) updateProgress(phase string, currentStep, totalSteps int, message, currentFile string) {
	s.Progress().Phase = phase
	s.Progress().CurrentStep = currentStep
	s.Progress().TotalSteps = totalSteps
	s.Progress().Message = message
	s.Progress().CurrentFile = currentFile
	s.Progress().Timestamp = time.Now()

	// Send progress event via WebSocket
	if s.wsHub != nil {
		progressData := map[string]interface{}{
			"workflow_id":     s.workflowID,
			"phase":           s.Progress().Phase,
			"current_step":    s.Progress().CurrentStep,
			"total_steps":     s.Progress().TotalSteps,
			"message":         s.Progress().Message,
			"current_file":    s.Progress().CurrentFile,
			"timestamp":       s.Progress().Timestamp,
			"bytes_processed":  s.Progress().BytesProcessed,
			"total_bytes":     s.Progress().TotalBytes,
		}

		s.wsHub.BroadcastMessage("progress", progressData)
	}

	s.logger.Info("Progress updated", map[string]interface{}{
		"phase":        s.Progress().Phase,
		"current_step": s.Progress().CurrentStep,
		"total_steps":  s.Progress().TotalSteps,
		"message":      s.Progress().Message,
	})
}

// Progress returns the current progress
func (s *SSHTranslationSystem) Progress() *TranslationProgress {
	// Initialize progress if nil
	if s.workflowID == "" {
		return &TranslationProgress{}
	}
	// For now, create a simple progress object
	return &TranslationProgress{
		Phase:           "initialization",
		CurrentStep:     0,
		TotalSteps:      7,
		Message:         "Initializing...",
		Timestamp:       time.Now(),
		BytesProcessed:  0,
		TotalBytes:      0,
	}
}

// StartMonitoringServer starts the WebSocket monitoring server
func (s *SSHTranslationSystem) StartMonitoringServer(addr string) error {
	return s.wsHub.StartServer(addr)
}

// createWorkspace creates the workspace directory
func (s *SSHTranslationSystem) createWorkspace() {
	if err := os.MkdirAll(s.config.Workspace, 0755); err != nil {
		s.logger.Error("Failed to create workspace", map[string]interface{}{
			"workspace": s.config.Workspace,
			"error":     err.Error(),
		})
	}
}

// getFileSize gets the file size
func (s *SSHTranslationSystem) getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// generateWorkflowID generates a unique workflow ID
func generateWorkflowID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:12]
}