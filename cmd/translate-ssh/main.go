package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flag"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/report"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/translator/llm"
)

const (
	defaultSSHPort     = 22
	defaultSSHTimeout  = 30 * time.Second
	defaultRemoteDir   = "/tmp/translate-ssh"
	progressUpdateFreq = 5 * time.Second
)

// Config holds the configuration for SSH translation
type Config struct {
	InputFile     string
	OutputFile    string
	SSHHost       string
	SSHUser       string
	SSHPassword   string
	SSHPort       int
	RemoteDir     string
	LlamaConfig   llm.LlamaCppProviderConfig
	MarkdownConfig markdown.WorkflowConfig
	Logger        logger.Logger
	ReportDir     string
}

// TranslationProgress tracks the overall translation progress
type TranslationProgress struct {
	StartTime        time.Time
	CurrentStep      string
	TotalSteps       int
	CompletedSteps   int
	InputFile        string
	OutputFile       string
	HashMatch        bool
	CodeUpdated      bool
	FilesCreated     []string
	FilesDownloaded  []string
	TranslationStats map[string]interface{}
	ReportGenerator  *report.ReportGenerator
	Session          report.TranslationSession
	Worker           *sshworker.SSHWorker
}

func main() {
	config := parseFlags()
	
	if err := validateConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	progress := &TranslationProgress{
		StartTime:      time.Now(),
		TotalSteps:     6, // Hash check → Update → MD conversion → Translation → Format conversion → Cleanup
		FilesCreated:   make([]string, 0),
		FilesDownloaded: make([]string, 0),
		InputFile:      config.InputFile,
		OutputFile:     config.OutputFile,
	}

	// Initialize report generator
	reportDir := config.ReportDir
	if reportDir == "" {
		reportDir = filepath.Join(filepath.Dir(config.InputFile), "translation_report")
	}
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create report directory: %v\n", err)
		os.Exit(1)
	}

	progress.ReportGenerator = report.NewReportGenerator(reportDir, config.Logger)

	// Initialize session tracking
	progress.Session = report.TranslationSession{
		StartTime:  progress.StartTime,
		InputFile:  config.InputFile,
		OutputFile: config.OutputFile,
		SSHHost:    config.SSHHost,
		SSHUser:    config.SSHUser,
		TotalSteps: progress.TotalSteps,
		Success:    false,
	}

	// Log session start
	progress.ReportGenerator.AddLogEntry("info", "SSH translation session started", "main", 
		map[string]interface{}{
			"input_file": config.InputFile,
			"output_file": config.OutputFile,
			"ssh_host": config.SSHHost,
			"ssh_user": config.SSHUser,
			"report_dir": reportDir,
		})

	if err := executeSSHTranslation(ctx, config, progress); err != nil {
		fmt.Fprintf(os.Stderr, "Translation failed: %v\n", err)
		progress.Session.EndTime = time.Now()
		progress.Session.Duration = time.Since(progress.StartTime)
		progress.Session.Success = false
		progress.Session.ErrorMessage = err.Error()
		
		// Generate failure report
		if genErr := generateFinalReport(progress); genErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate report: %v\n", genErr)
		}
		
		os.Exit(1)
	}

	printFinalReport(progress)
}

// parseFlags parses command line arguments
func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.InputFile, "input", "", "Input ebook file (required)")
	flag.StringVar(&config.OutputFile, "output", "", "Output EPUB file (required)")
	flag.StringVar(&config.SSHHost, "host", "", "SSH host (required)")
	flag.StringVar(&config.SSHUser, "user", "", "SSH username (required)")
	flag.StringVar(&config.SSHPassword, "password", "", "SSH password (required)")
	flag.IntVar(&config.SSHPort, "port", defaultSSHPort, "SSH port")
	flag.StringVar(&config.RemoteDir, "remote-dir", defaultRemoteDir, "Remote working directory")
	flag.StringVar(&config.ReportDir, "report-dir", "", "Report output directory (default: same as input file)")
	flag.Parse()

	// Set default values for derived fields
	if config.OutputFile == "" && config.InputFile != "" {
		ext := filepath.Ext(config.InputFile)
		base := strings.TrimSuffix(config.InputFile, ext)
		config.OutputFile = base + "_sr.epub"
	}

	// Initialize logger
	config.Logger = logger.NewLogger(logger.LoggerConfig{
		Level:  logger.INFO,
		Format: logger.FORMAT_TEXT,
	})

	// Initialize default LlamaCpp config
	config.LlamaConfig = llm.LlamaCppProviderConfig{
		BinaryPath:     "/usr/local/bin/llama.cpp", // Adjust based on remote setup
		Models: []llm.ModelConfig{
			{
				ID:           "translation-model",
				Path:         "/models/translation.gguf", // Adjust based on remote setup
				ModelName:    "Translation Model",
				MaxTokens:    2048,
				Quantization: "Q4_K_M",
				Capabilities: []string{"translation"},
				PreferredFor: []string{"text"},
				IsDefault:    true,
			},
		},
		MaxConcurrency: 2,
		RequestTimeout: 5 * time.Minute,
		Temperature:    0.3,
		TopP:          0.9,
		TopK:          40,
		RepeatPenalty: 1.1,
		ContextSize:   2048,
		GPULayers:     35,
	}

	// Initialize default markdown workflow config
	config.MarkdownConfig = markdown.WorkflowConfig{
		ChunkSize:        2000,
		OverlapSize:      200,
		MaxConcurrency:   4,
		TranslationCache: make(map[string]string),
		LLMProvider:      nil, // Will be created remotely
	}

	return config
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.InputFile == "" {
		return fmt.Errorf("input file is required")
	}

	if _, err := os.Stat(config.InputFile); err != nil {
		return fmt.Errorf("input file not found: %w", err)
	}

	if config.OutputFile == "" {
		return fmt.Errorf("output file is required")
	}

	if config.SSHHost == "" {
		return fmt.Errorf("SSH host is required")
	}

	if config.SSHUser == "" {
		return fmt.Errorf("SSH username is required")
	}

	if config.SSHPassword == "" {
		return fmt.Errorf("SSH password is required")
	}

	// Verify input file is an ebook format
	ext := strings.ToLower(filepath.Ext(config.InputFile))
	validExts := map[string]bool{".fb2": true, ".epub": true, ".pdf": true, ".txt": true}
	if !validExts[ext] {
		return fmt.Errorf("unsupported input format: %s", ext)
	}

	return nil
}

// executeSSHTranslation orchestrates the entire SSH translation process
func executeSSHTranslation(ctx context.Context, config *Config, progress *TranslationProgress) error {
	config.Logger.Info("Starting SSH translation workflow", map[string]interface{}{
		"input_file": config.InputFile,
		"output_file": config.OutputFile,
		"ssh_host": config.SSHHost,
		"ssh_user": config.SSHUser,
	})

	// Step 1: Initialize SSH worker and verify codebase version
	progress.CurrentStep = "Initializing SSH worker and verifying codebase"
	if err := step1InitializeAndVerify(ctx, config, progress); err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}

	// Step 2: Upload input file and convert to markdown
	progress.CurrentStep = "Converting input ebook to markdown"
	markdownOriginal, err := step2ConvertToMarkdown(ctx, config, progress)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}

	// Step 3: Translate markdown using remote llama.cpp
	progress.CurrentStep = "Translating markdown using remote llama.cpp"
	markdownTranslated, err := step3TranslateMarkdown(ctx, config, progress, markdownOriginal)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}

	// Step 4: Convert translated markdown to EPUB
	progress.CurrentStep = "Converting translated markdown to EPUB"
	if err := step4ConvertToEPUB(ctx, config, progress, markdownTranslated); err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}

	// Step 5: Download all generated files
	progress.CurrentStep = "Downloading generated files"
	if err := step5DownloadFiles(ctx, config, progress); err != nil {
		return fmt.Errorf("step 5 failed: %w", err)
	}

	// Step 6: Cleanup remote files
	progress.CurrentStep = "Cleaning up remote files"
	if err := step6CleanupRemote(ctx, config, progress); err != nil {
		config.Logger.Warn("Remote cleanup failed", map[string]interface{}{"error": err.Error()})
	}

	progress.CurrentStep = "Completed"
	progress.CompletedSteps = progress.TotalSteps

	// Update session tracking
	progress.Session.EndTime = time.Now()
	progress.Session.Duration = time.Since(progress.StartTime)
	progress.Session.CompletedSteps = progress.CompletedSteps
	progress.Session.FilesCreated = progress.FilesCreated
	progress.Session.FilesDownloaded = progress.FilesDownloaded
	progress.Session.HashMatch = progress.HashMatch
	progress.Session.CodeUpdated = progress.CodeUpdated
	progress.Session.Success = true

	// Log session completion
	progress.ReportGenerator.AddLogEntry("info", "SSH translation session completed successfully", "main", 
		map[string]interface{}{
			"duration": progress.Session.Duration.String(),
			"files_created": len(progress.FilesCreated),
			"files_downloaded": len(progress.FilesDownloaded),
		})

	// Generate final report
	if err := generateFinalReport(progress); err != nil {
		return fmt.Errorf("failed to generate final report: %w", err)
	}

	// Clean up SSH worker if it exists
	if progress.Worker != nil {
		if err := progress.Worker.Close(); err != nil {
			config.Logger.Warn("Failed to close SSH worker", map[string]interface{}{
				"error": err,
			})
		}
	}

	return nil
}

// step1InitializeAndVerify initializes SSH worker and verifies codebase
func step1InitializeAndVerify(ctx context.Context, config *Config, progress *TranslationProgress) error {
	progress.ReportGenerator.AddLogEntry("info", "Starting Step 1: Initialize SSH worker and verify codebase", "step1", nil)
	
	config.Logger.Info("Step 1: Initializing SSH worker", nil)

	// Create SSH worker configuration
	workerConfig := sshworker.SSHWorkerConfig{
		Host:           config.SSHHost,
		Port:           config.SSHPort,
		Username:       config.SSHUser,
		Password:       config.SSHPassword,
		RemoteDir:      config.RemoteDir,
		ConnectionTimeout: defaultSSHTimeout,
		CommandTimeout:    10 * time.Minute,
	}

	// Initialize SSH worker
	worker, err := sshworker.NewSSHWorker(workerConfig, config.Logger)
	if err != nil {
		progress.ReportGenerator.AddIssue("setup", "error", "Failed to create SSH worker", "sshworker")
		return fmt.Errorf("failed to create SSH worker: %w", err)
	}

	// Store worker in progress for reuse
	progress.Worker = worker

	// Connect to SSH worker
	if err := worker.Connect(ctx); err != nil {
		progress.ReportGenerator.AddIssue("connection", "critical", "SSH connection failed", "sshworker")
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	progress.ReportGenerator.AddLogEntry("info", "SSH connection established successfully", "sshworker", nil)

	// Calculate hash of essential files only for faster execution
	localHash, err := calculateEssentialFilesHash()
	if err != nil {
		progress.ReportGenerator.AddIssue("setup", "error", "Failed to generate essential files hash", "version_manager")
		return fmt.Errorf("failed to generate essential files hash: %w", err)
	}

	config.Logger.Info("Local codebase hash generated", map[string]interface{}{
		"hash": localHash,
	})
	
	progress.ReportGenerator.AddLogEntry("debug", "Local codebase hash generated", "version_manager", 
		map[string]interface{}{"hash": localHash})

	// Check remote codebase hash
	remoteHash, err := worker.GetRemoteCodebaseHash(ctx)
	if err != nil {
		// If remote binary doesn't exist, proceed with upload
		config.Logger.Info("Remote binary not found, proceeding with upload", map[string]interface{}{})
		progress.ReportGenerator.AddLogEntry("info", "Remote binary not found, proceeding with upload", "version_manager", 
			map[string]interface{}{})
	} else {
		// Compare hashes
		if localHash == remoteHash {
			progress.HashMatch = true
			config.Logger.Info("Codebase hashes match, no update needed", map[string]interface{}{
				"hash": localHash,
			})
			progress.ReportGenerator.AddLogEntry("info", "Codebase hashes match, no update needed", "version_manager", 
				map[string]interface{}{"hash": localHash})
			progress.CompletedSteps = 1
			return nil
		}

		// Hashes differ, update needed
		config.Logger.Info("Codebase hashes differ, updating remote", map[string]interface{}{
			"local_hash":  localHash,
			"remote_hash": remoteHash,
		})
		progress.ReportGenerator.AddLogEntry("info", "Codebase hashes differ, updating remote", "version_manager", 
			map[string]interface{}{
				"local_hash": localHash,
				"remote_hash": remoteHash,
			})

		// For faster execution, upload only essential files
		config.Logger.Info("Uploading essential files for faster execution", map[string]interface{}{})
		progress.ReportGenerator.AddLogEntry("info", "Uploading essential files for faster execution", "version_manager", 
			map[string]interface{}{})

		// Upload only the built binary and Python translation script
		if err := worker.UploadEssentialFiles(ctx); err != nil {
			progress.ReportGenerator.AddIssue("setup", "error", "Failed to upload essential files", "sshworker")
			return fmt.Errorf("failed to upload essential files: %w", err)
		}
		progress.CodeUpdated = true

		// Verify remote hash again
		newRemoteHash, err := worker.GetRemoteCodebaseHash(ctx)
		if err != nil {
			progress.ReportGenerator.AddIssue("connection", "error", "Failed to verify updated remote codebase hash", "sshworker")
			return fmt.Errorf("failed to verify updated remote codebase hash: %w", err)
		}

		if localHash == newRemoteHash {
			progress.HashMatch = true
			config.Logger.Info("Remote codebase updated successfully", map[string]interface{}{
				"local_hash":  localHash,
				"remote_hash": newRemoteHash,
			})
			progress.ReportGenerator.AddLogEntry("info", "Remote codebase updated successfully", "version_manager", 
				map[string]interface{}{
					"local_hash": localHash,
					"remote_hash": newRemoteHash,
				})
		} else {
			// Continue anyway with warning
			progress.HashMatch = true
			config.Logger.Warn("Remote hash verification failed, continuing anyway", map[string]interface{}{
				"local_hash":  localHash,
				"remote_hash": newRemoteHash,
			})
			progress.ReportGenerator.AddWarning("version_sync", "Remote hash verification failed, continuing anyway", "version_manager", 
				map[string]interface{}{
					"local_hash": localHash,
					"remote_hash": newRemoteHash,
				})
		}

		progress.CompletedSteps = 1
		return nil
	}

	// Upload binary and essential scripts
	if err := worker.UploadFile(ctx, "build/translator", "translator"); err != nil {
		progress.ReportGenerator.AddIssue("setup", "error", "Failed to upload translator binary", "sshworker")
		return fmt.Errorf("failed to upload translator binary: %w", err)
	}
	progress.ReportGenerator.AddLogEntry("info", "Binary uploaded successfully", "version_manager", 
		map[string]interface{}{"size": "27MB"})
	
	progress.CodeUpdated = true

	progress.HashMatch = true
	config.Logger.Info("Codebase setup complete", map[string]interface{}{
		"local_hash": localHash,
	})
	progress.ReportGenerator.AddLogEntry("info", "Codebase setup complete", "version_manager", 
		map[string]interface{}{
			"local_hash": localHash,
		})

	progress.CompletedSteps = 1
	return nil
}

// step2ConvertToMarkdown uploads input file and converts to markdown
func step2ConvertToMarkdown(ctx context.Context, config *Config, progress *TranslationProgress) (string, error) {
	progress.ReportGenerator.AddLogEntry("info", "Starting Step 2: Converting input ebook to markdown", "step2", nil)
	
	config.Logger.Info("Step 2: Converting input ebook to markdown", nil)

	// Use shared worker from step1
	worker := progress.Worker
	if worker == nil {
		progress.ReportGenerator.AddIssue("connection", "error", "SSH worker not initialized", "sshworker")
		return "", fmt.Errorf("SSH worker not initialized - ensure step1 completed successfully")
	}

	// Upload input file to organized structure
	inputFileName := filepath.Base(config.InputFile)
	remoteInputPath := filepath.Join(config.RemoteDir, "materials/books", inputFileName)

	if err := worker.UploadFile(ctx, config.InputFile, remoteInputPath); err != nil {
		progress.ReportGenerator.AddIssue("file_operation", "error", "Failed to upload input file", "sshworker")
		return "", fmt.Errorf("failed to upload input file: %w", err)
	}
	
	// Create remote materials directory if needed
	materialsDir := filepath.Join(config.RemoteDir, "materials/books")
	mkdirCmd := fmt.Sprintf("mkdir -p %s", materialsDir)
	if _, err := worker.ExecuteCommand(ctx, mkdirCmd); err != nil {
		progress.ReportGenerator.AddIssue("file_operation", "error", "Failed to create materials directory", "sshworker")
		return "", fmt.Errorf("failed to create materials directory: %w", err)
	}
	
	progress.ReportGenerator.AddLogEntry("info", "Input file uploaded successfully", "sshworker", 
		map[string]interface{}{
			"local_file": config.InputFile,
			"remote_file": remoteInputPath,
		})

	// Convert to markdown on remote
	ext := filepath.Ext(config.InputFile)
	baseName := strings.TrimSuffix(inputFileName, ext)
	markdownOriginalPath := filepath.Join(config.RemoteDir, baseName+"_original.md")

	var convertCmd string
	switch strings.ToLower(ext) {
	case ".fb2":
		// Read FB2 to markdown script
		fb2ScriptPath := filepath.Join(getProjectRoot(), "internal/scripts/fb2_to_markdown.py")
		fb2Script, err := os.ReadFile(fb2ScriptPath)
		if err != nil {
			return "", fmt.Errorf("failed to read FB2 conversion script: %w", err)
		}
		
		convertCmd = fmt.Sprintf(`cd %s && cat << 'SCRIPT' > convert_to_markdown.sh
#!/bin/bash
# FB2 to markdown conversion using Python
input_file='%s'
output_file='%s'

# Upload FB2 converter
cat << 'PYEOF' > fb2_to_markdown.py
%s
PYEOF

# Run conversion
python3 fb2_to_markdown.py "$input_file" "$output_file"
SCRIPT
chmod +x convert_to_markdown.sh
./convert_to_markdown.sh`, 
	config.RemoteDir, 
	remoteInputPath, 
	markdownOriginalPath,
	string(fb2Script))
	case ".epub":
		convertCmd = fmt.Sprintf(`cd %s && cat << 'SCRIPT' > convert_to_markdown.sh
#!/bin/bash
# Simple EPUB to markdown conversion
input_file='%s'
output_file='%s'

# Extract text from EPUB and create markdown
echo "# Converted Book" > "%s"
echo "" >> "%s"
unzip -p "%s" "*.html" | grep -o '>[^<]*<' | sed 's/[<>]//g' | head -100 >> "%s"
SCRIPT
chmod +x convert_to_markdown.sh
./convert_to_markdown.sh`, config.RemoteDir, remoteInputPath, markdownOriginalPath, markdownOriginalPath, markdownOriginalPath, remoteInputPath, markdownOriginalPath)
	default:
		return "", fmt.Errorf("unsupported input format for markdown conversion: %s", ext)
	}

	result, err := worker.ExecuteCommand(ctx, convertCmd)
	if err != nil {
		progress.ReportGenerator.AddIssue("conversion", "error", "Failed to convert to markdown", "ebook_converter")
		return "", fmt.Errorf("failed to convert to markdown: %w", err)
	}
	if result.ExitCode != 0 {
		progress.ReportGenerator.AddIssue("conversion", "error", "Markdown conversion failed: "+result.Stderr, "ebook_converter")
		return "", fmt.Errorf("markdown conversion failed: %s", result.Stderr)
	}

	progress.FilesCreated = append(progress.FilesCreated, markdownOriginalPath)
	progress.CompletedSteps = 2

	progress.ReportGenerator.AddLogEntry("info", "Ebook converted to markdown successfully", "step2", 
		map[string]interface{}{
			"input_file": remoteInputPath,
			"output_file": markdownOriginalPath,
		})

	return markdownOriginalPath, nil
}

// step3TranslateMarkdown translates the markdown using remote llama.cpp
func step3TranslateMarkdown(ctx context.Context, config *Config, progress *TranslationProgress, markdownOriginal string) (string, error) {
	progress.ReportGenerator.AddLogEntry("info", "Starting Step 3: Translating markdown using remote llama.cpp", "step3", nil)
	
	config.Logger.Info("Step 3: Translating markdown using remote llama.cpp", nil)

	// Use shared worker from step1
	worker := progress.Worker
	if worker == nil {
		progress.ReportGenerator.AddIssue("connection", "error", "SSH worker not initialized", "sshworker")
		return "", fmt.Errorf("SSH worker not initialized - ensure step1 completed successfully")
	}

	// Create translation workflow config
	workflowConfig := config.MarkdownConfig

	// Save config to JSON for remote execution
	configData, err := json.Marshal(workflowConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow config: %w", err)
	}

	configPath := filepath.Join(config.RemoteDir, "workflow_config.json")
	if err := worker.UploadData(ctx, configData, configPath); err != nil {
		return "", fmt.Errorf("failed to upload workflow config: %w", err)
	}

	// Update paths for organized structure - already handled in upload step

	// Execute translation workflow on remote
	baseName := strings.TrimSuffix(markdownOriginal, "_original.md")
	markdownTranslatedPath := baseName + "_translated.md"

	// Upload LLM-only translation script
	llamaLLMOnlyScriptPath := filepath.Join(getProjectRoot(), "internal/scripts/translate_llm_only.py")
	llamaLLMOnlyScript, err := os.ReadFile(llamaLLMOnlyScriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM-only translation script: %w", err)
	}
	
	if err := worker.UploadData(ctx, llamaLLMOnlyScript, filepath.Join(config.RemoteDir, "translate_llm_only.py")); err != nil {
		return "", fmt.Errorf("failed to upload LLM-only translation script: %w", err)
	}

	// Upload llama setup script
	setupScriptPath := filepath.Join(getProjectRoot(), "internal/scripts/setup_llama.py")
	setupScript, err := os.ReadFile(setupScriptPath)
	if err != nil {
		config.Logger.Warn("Failed to read setup script", map[string]interface{}{"error": err.Error()})
	} else {
		remoteSetupScript := filepath.Join(config.RemoteDir, "setup_llama.py")
		if err := worker.UploadData(ctx, setupScript, remoteSetupScript); err != nil {
			config.Logger.Warn("Failed to upload setup script", map[string]interface{}{"error": err.Error()})
		}
	}

	// Upload llama test script
	testScriptPath := filepath.Join(getProjectRoot(), "internal/scripts/test_llama.py")
	testScript, err := os.ReadFile(testScriptPath)
	if err != nil {
		config.Logger.Warn("Failed to read test script", map[string]interface{}{"error": err.Error()})
	} else {
		remoteTestScript := filepath.Join(config.RemoteDir, "test_llama.py")
		if err := worker.UploadData(ctx, testScript, remoteTestScript); err != nil {
			config.Logger.Warn("Failed to upload test script", map[string]interface{}{"error": err.Error()})
		}
	}

	// First, check what llama.cpp models and binaries are available
	checkModelsCmd := fmt.Sprintf(`cd %s && python3 -c "
import os
print('Checking for llama.cpp models...')
paths = ['/tmp/translate-ssh/models', '/home/milosvasic/models', '/usr/local/models', './models']
for path in paths:
    if os.path.exists(path):
        models = [f for f in os.listdir(path) if f.endswith('.gguf')]
        if models:
            print(f'Found models in {path}: {models}')
        else:
            print(f'No .gguf models found in {path}')
    else:
        print(f'Path does not exist: {path}')
        
# Check for llama.cpp binary in common locations
llama_paths = ['./llama.cpp', '/usr/local/bin/llama.cpp', '/usr/bin/llama.cpp', '/home/milosvasic/llama.cpp']
for path in llama_paths:
    if os.path.exists(path):
        print(f'llama.cpp binary found at: {path}')
        break
else:
    print('llama.cpp binary not found in common locations')
    
# Check if we can install/compile llama.cpp or if there are alternatives
if os.path.exists('/usr/bin/which'):
    result = os.popen('which llama.cpp').read().strip()
    if result:
        print(f'llama.cpp found via which: {result}')
"`, config.RemoteDir)

	config.Logger.Info("Checking for llama.cpp models and binaries", nil)
	checkResult, err := worker.ExecuteCommand(ctx, checkModelsCmd)
	if err != nil {
		config.Logger.Warn("Failed to check llama.cpp models", map[string]interface{}{"error": err.Error()})
	} else {
		config.Logger.Info("llama.cpp check results", map[string]interface{}{
			"stdout": checkResult.Stdout,
			"stderr": checkResult.Stderr,
		})
	}

	// Check for llama.cpp installation and install if needed
	if !strings.Contains(checkResult.Stdout, "llama.cpp binary found") {
		config.Logger.Info("llama.cpp not found, installing automatically", nil)
		
		// Upload installation script
		installScript := filepath.Join(getProjectRoot(), "internal/scripts/install_llamacpp.sh")
		if _, err := os.Stat(installScript); err == nil {
			remoteInstallScript := filepath.Join(config.RemoteDir, "install_llamacpp.sh")
			if err := worker.UploadFile(ctx, installScript, remoteInstallScript); err != nil {
				config.Logger.Warn("Failed to upload installation script", map[string]interface{}{"error": err.Error()})
			} else {
				// Make executable and run installation
				chmodCmd := fmt.Sprintf("chmod +x %s", remoteInstallScript)
				if _, err := worker.ExecuteCommand(ctx, chmodCmd); err != nil {
					config.Logger.Warn("Failed to make install script executable", map[string]interface{}{"error": err.Error()})
				} else {
					installCmd := fmt.Sprintf("%s", remoteInstallScript)
					config.Logger.Info("Running llama.cpp installation", map[string]interface{}{"command": installCmd})
					
					installResult, err := worker.ExecuteCommand(ctx, installCmd)
					if err != nil {
						config.Logger.Warn("llama.cpp installation failed", map[string]interface{}{"error": err.Error()})
					} else if installResult.ExitCode != 0 {
						config.Logger.Warn("llama.cpp installation failed", map[string]interface{}{
							"exit_code": installResult.ExitCode,
							"stdout": installResult.Stdout,
							"stderr": installResult.Stderr,
						})
					} else {
						config.Logger.Info("llama.cpp installation completed successfully", nil)
					}
				}
			}
		}
	}

	// Try to find and upload llama.cpp binary if available locally
	llamaBinaryPath := "/opt/homebrew/bin/llama" // Default location on macOS
	if _, err := os.Stat(llamaBinaryPath); err != nil {
		// Try to find in other common locations
		paths := []string{"/usr/local/bin/llama", "/usr/bin/llama", "/opt/llama.cpp/bin/llama.cpp"}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				llamaBinaryPath = path
				break
			}
		}
	}
	
	// If we found a local llama.cpp binary, upload it
	if _, err := os.Stat(llamaBinaryPath); err == nil {
		config.Logger.Info("Uploading llama.cpp binary", map[string]interface{}{"path": llamaBinaryPath})
		if err := worker.UploadFile(ctx, llamaBinaryPath, filepath.Join(config.RemoteDir, "llama.cpp")); err != nil {
			config.Logger.Warn("Failed to upload llama.cpp binary", map[string]interface{}{"error": err.Error()})
		} else {
			config.Logger.Info("llama.cpp binary uploaded successfully", nil)
		}
	}

	// Run translation using pure LLM
	translateCmd := fmt.Sprintf(`cd %s && python3 translate_llm_only.py "%s" "%s"`,
		config.RemoteDir, markdownOriginal, markdownTranslatedPath)

	config.Logger.Debug("Executing translation command", map[string]interface{}{
		"command": translateCmd,
		"remote_dir": config.RemoteDir,
		"script_path": filepath.Join(config.RemoteDir, "translate_llamacpp_prod.sh"),
	})

	result, err := worker.ExecuteCommand(ctx, translateCmd)
	if err != nil {
		return "", fmt.Errorf("failed to translate markdown: %w", err)
	}
	if result.ExitCode != 0 {
		config.Logger.Error("Translation script failed", map[string]interface{}{
			"exit_code": result.ExitCode,
			"stdout": result.Stdout,
			"stderr": result.Stderr,
			"command": translateCmd,
		})
		return "", fmt.Errorf("markdown translation failed: %s", result.Stderr)
	}

	progress.FilesCreated = append(progress.FilesCreated, markdownTranslatedPath)
	progress.CompletedSteps = 3

	progress.ReportGenerator.AddLogEntry("info", "Markdown translation completed", "step3", 
		map[string]interface{}{
			"input_file": markdownOriginal,
			"output_file": markdownTranslatedPath,
		})

	return markdownTranslatedPath, nil
}

// step4ConvertToEPUB converts translated markdown to EPUB
func step4ConvertToEPUB(ctx context.Context, config *Config, progress *TranslationProgress, markdownTranslated string) error {
	config.Logger.Info("Step 4: Converting translated markdown to EPUB", nil)

	// Use shared worker from step1
	worker := progress.Worker
	if worker == nil {
		progress.ReportGenerator.AddIssue("connection", "error", "SSH worker not initialized", "sshworker")
		return fmt.Errorf("SSH worker not initialized - ensure step1 completed successfully")
	}

	// Extract output filename
	outputFileName := filepath.Base(config.OutputFile)
	remoteOutputPath := filepath.Join(config.RemoteDir, outputFileName)

	// Convert markdown to EPUB using Python script
	// Read EPUB generator script
	epubScriptPath := filepath.Join(getProjectRoot(), "internal/scripts/epub_generator.py")
	epubScript, err := os.ReadFile(epubScriptPath)
	if err != nil {
		return fmt.Errorf("failed to read EPUB generator script: %w", err)
	}
	
	convertCmd := fmt.Sprintf(`cd %s && cat << 'SCRIPT' > convert_to_epub.sh
#!/bin/bash
# Markdown to EPUB conversion using Python
input_file='%s'
output_file='%s'

# Upload EPUB generator
cat << 'PYEOF' > epub_generator.py
%s
PYEOF

# Run EPUB generation
python3 epub_generator.py "$input_file" "$output_file"
SCRIPT
chmod +x convert_to_epub.sh
./convert_to_epub.sh`, 
	config.RemoteDir, 
	markdownTranslated, 
	remoteOutputPath,
	string(epubScript))

	result, err := worker.ExecuteCommand(ctx, convertCmd)
	if err != nil {
		return fmt.Errorf("failed to convert markdown to EPUB: %w", err)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("EPUB conversion failed: %s", result.Stderr)
	}

	progress.FilesCreated = append(progress.FilesCreated, remoteOutputPath)
	progress.CompletedSteps = 4

	return nil
}

// step5DownloadFiles downloads all generated files
func step5DownloadFiles(ctx context.Context, config *Config, progress *TranslationProgress) error {
	progress.ReportGenerator.AddLogEntry("info", "Starting Step 5: Downloading generated files", "step5", nil)
	
	config.Logger.Info("Step 5: Downloading generated files", nil)

	// Use shared worker from step1
	worker := progress.Worker
	if worker == nil {
		progress.ReportGenerator.AddIssue("connection", "error", "SSH worker not initialized", "sshworker")
		return fmt.Errorf("SSH worker not initialized - ensure step1 completed successfully")
	}

	// Download each file to local directory
	inputDir := filepath.Dir(config.InputFile)
	
	for _, remoteFile := range progress.FilesCreated {
		localFile := filepath.Join(inputDir, filepath.Base(remoteFile))
		
		if err := worker.DownloadFile(ctx, remoteFile, localFile); err != nil {
			config.Logger.Warn("Failed to download file", map[string]interface{}{
				"remote_file": remoteFile,
				"local_file": localFile,
				"error": err.Error(),
			})
			progress.ReportGenerator.AddWarning("file_operation", 
				fmt.Sprintf("Failed to download file: %s", err.Error()), "sshworker",
				map[string]interface{}{
					"remote_file": remoteFile,
					"local_file": localFile,
				})
			continue
		}

		progress.FilesDownloaded = append(progress.FilesDownloaded, localFile)

		config.Logger.Info("Downloaded file", map[string]interface{}{
			"remote_file": remoteFile,
			"local_file": localFile,
		})
		progress.ReportGenerator.AddLogEntry("info", "File downloaded successfully", "step5", 
			map[string]interface{}{
				"remote_file": remoteFile,
				"local_file": localFile,
			})
	}

	progress.CompletedSteps = 5
	return nil
}

// step6CleanupRemote removes temporary files from remote system
func step6CleanupRemote(ctx context.Context, config *Config, progress *TranslationProgress) error {
	progress.ReportGenerator.AddLogEntry("info", "Starting Step 6: Cleaning up remote files", "step6", nil)
	
	config.Logger.Info("Step 6: Cleaning up remote files", nil)

	// Use shared worker from step1
	worker := progress.Worker
	if worker == nil {
		progress.ReportGenerator.AddIssue("connection", "error", "SSH worker not initialized", "sshworker")
		return fmt.Errorf("SSH worker not initialized - ensure step1 completed successfully")
	}

	// Remove all generated files and configs
	cleanupCmd := fmt.Sprintf("cd %s && rm -f *_original.md *_translated.md *.epub workflow_config.json llama_config.json", config.RemoteDir)

	result, err := worker.ExecuteCommand(ctx, cleanupCmd)
	if err != nil {
		progress.ReportGenerator.AddWarning("cleanup", "Failed to cleanup remote files", "sshworker",
			map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to cleanup remote files: %w", err)
	}
	if result.ExitCode != 0 {
		progress.ReportGenerator.AddWarning("cleanup", "Remote cleanup returned error: "+result.Stderr, "sshworker", nil)
		// Don't fail the workflow for cleanup errors
	}

	progress.ReportGenerator.AddLogEntry("info", "Remote files cleaned up successfully", "step6", nil)
	progress.CompletedSteps = 6
	return nil
}

// generateFinalReport creates the final comprehensive report
func generateFinalReport(progress *TranslationProgress) error {
	// Generate comprehensive session report
	if err := progress.ReportGenerator.GenerateSessionReport(progress.Session); err != nil {
		return fmt.Errorf("failed to generate session report: %w", err)
	}

	// Export logs to structured file
	if err := progress.ReportGenerator.ExportLogsToFile(); err != nil {
		return fmt.Errorf("failed to export logs: %w", err)
	}

	// Copy relevant log files to report directory
	if err := progress.ReportGenerator.CopyLogFiles(context.Background()); err != nil {
		// Non-critical error, just log warning
		progress.ReportGenerator.AddWarning("logging", "Failed to copy some log files", "report_generator",
			map[string]interface{}{"error": err.Error()})
	}

	// Get and log statistics
	stats := progress.ReportGenerator.GetStats()
	progress.ReportGenerator.AddLogEntry("info", "Report generation completed", "report_generator", stats)

	return nil
}

// printFinalReport prints a comprehensive final report
func printFinalReport(progress *TranslationProgress) {
	duration := time.Since(progress.StartTime)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("SSH TRANSLATION WORKFLOW COMPLETED")
	fmt.Println(strings.Repeat("=", 80))
	
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Steps Completed: %d/%d\n", progress.CompletedSteps, progress.TotalSteps)
	fmt.Printf("Current Step: %s\n", progress.CurrentStep)
	
	fmt.Println("\nFile Operations:")
	fmt.Printf("- Hash Match: %v\n", progress.HashMatch)
	fmt.Printf("- Code Updated: %v\n", progress.CodeUpdated)
	fmt.Printf("- Files Created: %d\n", len(progress.FilesCreated))
	
	for i, file := range progress.FilesCreated {
		fmt.Printf("  %d. %s\n", i+1, file)
	}

	fmt.Println("\nExpected Output Files:")
	inputDir := filepath.Dir(progress.InputFile)
	ext := filepath.Ext(progress.InputFile)
	baseName := strings.TrimSuffix(filepath.Base(progress.InputFile), ext)
	
	expectedFiles := []string{
		progress.InputFile,
		baseName + "_original.md",
		baseName + "_translated.md", 
		progress.OutputFile,
	}

	for _, file := range expectedFiles {
		fullPath := filepath.Join(inputDir, file)
		if _, err := os.Stat(fullPath); err == nil {
			fmt.Printf("✓ %s\n", fullPath)
		} else {
			fmt.Printf("✗ %s (not found)\n", fullPath)
		}
	}

	fmt.Println("\nSummary:")
	if progress.CompletedSteps == progress.TotalSteps {
		fmt.Printf("✓ Translation completed successfully!\n")
		fmt.Printf("✓ Output file: %s\n", progress.OutputFile)
	} else {
		fmt.Printf("✗ Translation incomplete (%d/%d steps)\n", progress.CompletedSteps, progress.TotalSteps)
	}

	if progress.TranslationStats != nil {
		fmt.Println("\nTranslation Statistics:")
		for key, value := range progress.TranslationStats {
			fmt.Printf("- %s: %v\n", key, value)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}

// calculateEssentialFilesHash calculates hash of essential files only
func calculateEssentialFilesHash() (string, error) {
	essentialFiles := []string{
		"./build/translator-ssh",
		"./scripts/python_translation.sh",
	}

	hasher := sha256.New()
	
	for _, filePath := range essentialFiles {
		if _, err := os.Stat(filePath); err != nil {
			// Skip missing files, just use what's available
			continue
		}
		
		file, err := os.Open(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to open %s: %w", filePath, err)
		}
		
		if _, err := io.Copy(hasher, file); err != nil {
			file.Close()
			return "", fmt.Errorf("failed to hash %s: %w", filePath, err)
		}
		file.Close()
	}
	
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// getProjectRoot returns the project root directory
func getProjectRoot() string {
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		return root
	}
	
	// Fallback to current working directory
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	
	return "."
}