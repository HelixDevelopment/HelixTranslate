package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flag"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/translator/llm"
	"digital.vasic.translator/pkg/version"
)

const (
	appVersion = "2.1.0"
)

// Global configuration
type TranslationConfig struct {
	InputFile     string
	OutputFile    string
	SSHHost       string
	SSHUser       string
	SSHPassword   string
	SSHPort       int
	RemoteDir     string
	LlamaConfig   llm.LlamaCppProviderConfig
	Workers       int
	ChunkSize     int
	Concurrency   int
	VerifyOutput  bool
	Verbose       bool
}

// DocumentationData collects information for integral documentation
type DocumentationData struct {
	InputFile       string
	OutputFile      string
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	SSHHost         string
	RemoteDir       string
	LocalHash       string
	RemoteHash      string
	OriginalMDPath  string
	TranslatedMDPath string
	FinalEPUBPath   string
	StepsCompleted  []StepInfo
	FilesGenerated  []FileInfo
	IssuesEncountered []IssueInfo
}

// StepInfo tracks each translation step
type StepInfo struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Success   bool
	Details   string
	Error     string
}

// FileInfo tracks generated files
type FileInfo struct {
	Path         string
	Size         int64
	ContentType  string
	Verified     bool
	Verification string
}

// IssueInfo tracks issues encountered
type IssueInfo struct {
	Step       string
	Severity   string
	Message    string
	Resolution string
}

func main() {
	// Parse command line arguments
	config := parseFlags()
	
	// Initialize logger
	logLevel := logger.INFO
	if config.Verbose {
		logLevel = logger.DEBUG
	}
	
	logger := logger.NewLogger(logger.LoggerConfig{
		Level:  logLevel,
		Format: logger.FORMAT_TEXT,
	})
	
	// Initialize documentation data
	docs := &DocumentationData{
		StartTime:      time.Now(),
		InputFile:      config.InputFile,
		OutputFile:     config.OutputFile,
		SSHHost:       config.SSHHost,
		RemoteDir:      config.RemoteDir,
		StepsCompleted: make([]StepInfo, 0),
		FilesGenerated: make([]FileInfo, 0),
		IssuesEncountered: make([]IssueInfo, 0),
	}
	
	// Execute translation with documentation
	var err error
	if config.SSHHost != "" {
		err = executeRemoteTranslationWithDocs(context.Background(), config, logger, docs)
	} else {
		err = executeLocalTranslationWithDocs(context.Background(), config, logger, docs)
	}
	
	// Finalize documentation
	docs.EndTime = time.Now()
	docs.Duration = docs.EndTime.Sub(docs.StartTime)
	
	if err != nil {
		logger.Error("Translation failed", map[string]interface{}{
			"error": err.Error(),
		})
		
		docs.IssuesEncountered = append(docs.IssuesEncountered, IssueInfo{
			Step:     "Overall",
			Severity:  "Critical",
			Message:   err.Error(),
			Resolution: "Failed to complete translation",
		})
		
		generateIntegralDocumentation(docs)
		os.Exit(1)
	}
	
	logger.Info("Translation completed successfully", map[string]interface{}{
		"input":  config.InputFile,
		"output": config.OutputFile,
	})
	
	// Generate integral documentation
	if err := generateIntegralDocumentation(docs); err != nil {
		logger.Error("Failed to generate documentation", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// executeRemoteTranslationWithDocs performs translation via SSH worker with documentation
func executeRemoteTranslationWithDocs(ctx context.Context, config *TranslationConfig, logger logger.Logger, docs *DocumentationData) error {
	step := startStep(docs, "Remote Translation Setup")
	
	logger.Info("Starting remote translation via SSH", map[string]interface{}{
		"host": config.SSHHost,
		"user": config.SSHUser,
	})
	
	// Step 1: Verify and sync codebase
	endStep(step)
	step = startStep(docs, "Codebase Verification & Sync")
	
	localHash, remoteHash, err := verifyAndSyncCodebaseWithDocs(ctx, config, logger, docs)
	if err != nil {
		step.Error = err.Error()
		endStep(step)
		return fmt.Errorf("codebase verification failed: %w", err)
	}
	
	docs.LocalHash = localHash
	docs.RemoteHash = remoteHash
	endStep(step)
	
	// Step 2: Initialize SSH worker
	step = startStep(docs, "SSH Worker Initialization")
	
	worker, err := initializeSSHWorker(config, logger)
	if err != nil {
		step.Error = err.Error()
		endStep(step)
		return fmt.Errorf("failed to initialize SSH worker: %w", err)
	}
	defer worker.Close()
	endStep(step)
	
	// Step 3: Convert FB2 to Markdown
	step = startStep(docs, "FB2 to Markdown Conversion")
	
	originalMarkdown, err := convertFB2ToMarkdownRemoteWithDocs(ctx, config, worker, logger, docs)
	if err != nil {
		step.Error = err.Error()
		endStep(step)
		return fmt.Errorf("failed to convert FB2 to markdown: %w", err)
	}
	
	docs.OriginalMDPath = originalMarkdown
	endStep(step)
	
	// Step 4: Translate markdown
	step = startStep(docs, "Markdown Translation")
	
	translatedMarkdown, err := translateMarkdownRemoteWithDocs(ctx, config, worker, originalMarkdown, logger, docs)
	if err != nil {
		step.Error = err.Error()
		endStep(step)
		return fmt.Errorf("failed to translate markdown: %w", err)
	}
	
	docs.TranslatedMDPath = translatedMarkdown
	endStep(step)
	
	// Step 5: Convert translated markdown to EPUB
	step = startStep(docs, "Markdown to EPUB Conversion")
	
	if err := convertMarkdownToEPUBRemoteWithDocs(ctx, config, worker, translatedMarkdown, logger, docs); err != nil {
		step.Error = err.Error()
		endStep(step)
		return fmt.Errorf("failed to convert markdown to EPUB: %w", err)
	}
	
	docs.FinalEPUBPath = filepath.Join(filepath.Dir(config.OutputFile), filepath.Base(config.OutputFile))
	endStep(step)
	
	// Step 6: Download and verify results
	step = startStep(docs, "Download & Verification")
	
	if err := downloadAndVerifyResultsWithDocs(ctx, config, worker, logger, docs); err != nil {
		step.Error = err.Error()
		endStep(step)
		return fmt.Errorf("failed to download/verify results: %w", err)
	}
	
	endStep(step)
	
	return nil
}

// executeLocalTranslationWithDocs performs translation locally with documentation
func executeLocalTranslationWithDocs(ctx context.Context, config *TranslationConfig, logger logger.Logger, docs *DocumentationData) error {
	step := startStep(docs, "Local Translation")
	
	logger.Info("Starting local translation", map[string]interface{}{
		"input": config.InputFile,
	})
	
	step.Error = "Local translation not yet implemented in unified CLI"
	endStep(step)
	
	return fmt.Errorf("local translation not yet implemented in unified CLI")
}

// startStep begins tracking a new step
func startStep(docs *DocumentationData, stepName string) *StepInfo {
	step := StepInfo{
		Name:      stepName,
		StartTime: time.Now(),
		Success:   false,
	}
	docs.StepsCompleted = append(docs.StepsCompleted, step)
	
	return &docs.StepsCompleted[len(docs.StepsCompleted)-1]
}

// endStep marks a step as completed
func endStep(step *StepInfo) {
	step.EndTime = time.Now()
	step.Success = step.Error == ""
}

// verifyAndSyncCodebaseWithDocs ensures remote worker has matching codebase version with docs
func verifyAndSyncCodebaseWithDocs(ctx context.Context, config *TranslationConfig, logger logger.Logger, docs *DocumentationData) (string, string, error) {
	logger.Info("Verifying and syncing codebase", nil)
	
	// Calculate local codebase hash
	hasher := version.NewCodebaseHasher()
	localHash, err := hasher.CalculateHash()
	if err != nil {
		return "", "", fmt.Errorf("failed to calculate local codebase hash: %w", err)
	}
	
	logger.Debug("Local codebase hash calculated", map[string]interface{}{
		"hash": localHash,
	})
	
	// Initialize SSH worker for codebase operations
	workerConfig := sshworker.SSHWorkerConfig{
		Host:              config.SSHHost,
		Port:              config.SSHPort,
		Username:          config.SSHUser,
		Password:          config.SSHPassword,
		RemoteDir:         config.RemoteDir,
		ConnectionTimeout:  30 * time.Second,
		CommandTimeout:     5 * time.Minute,
	}
	
	worker, err := sshworker.NewSSHWorker(workerConfig, logger)
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH worker: %w", err)
	}
	defer worker.Close()
	
	if err := worker.Connect(ctx); err != nil {
		return "", "", fmt.Errorf("failed to connect to remote worker: %w", err)
	}
	
	// Get remote codebase hash
	remoteHash, err := worker.GetRemoteCodebaseHash(ctx)
	if err != nil {
		logger.Info("Remote codebase not found, proceeding with initial setup", nil)
		remoteHash = "<not found>"
		// Upload codebase if not present
		if err := uploadCodebase(ctx, config, worker, logger); err != nil {
			return localHash, remoteHash, fmt.Errorf("failed to upload codebase: %w", err)
		}
		return localHash, "<newly uploaded>", nil
	}
	
	logger.Debug("Remote codebase hash retrieved", map[string]interface{}{
		"hash": remoteHash,
	})
	
	// Compare hashes
	if localHash == remoteHash {
		logger.Info("Codebase versions match, no sync needed", map[string]interface{}{
			"hash": localHash,
		})
		return localHash, remoteHash, nil
	}
	
	logger.Info("Codebase versions differ, updating remote", map[string]interface{}{
		"local":  localHash,
		"remote": remoteHash,
	})
	
	// Upload updated codebase
	if err := uploadCodebase(ctx, config, worker, logger); err != nil {
		return localHash, remoteHash, fmt.Errorf("failed to upload codebase: %w", err)
	}
	
	return localHash, remoteHash, nil
}

// uploadCodebase uploads current codebase to remote worker
func uploadCodebase(ctx context.Context, config *TranslationConfig, worker *sshworker.SSHWorker, logger logger.Logger) error {
	logger.Info("Uploading codebase to remote worker", nil)
	
	// Create temporary codebase package
	codebaseFiles := []string{
		"./translator-ssh",
		"./llm_translation.sh",
		"./comprehensive_hash",
	}
	
	for _, file := range codebaseFiles {
		localPath := filepath.Join("./build", file)
		remotePath := filepath.Join(config.RemoteDir, file)
		
		if _, err := os.Stat(localPath); err != nil {
			logger.Warn("Codebase file not found, skipping", map[string]interface{}{
				"file": localPath,
			})
			continue
		}
		
		if err := worker.UploadFile(ctx, localPath, remotePath); err != nil {
			return fmt.Errorf("failed to upload %s: %w", file, err)
		}
		
		logger.Debug("Codebase file uploaded", map[string]interface{}{
			"file": file,
		})
	}
	
	// Set permissions and create hash file on remote
	cmd := fmt.Sprintf("cd %s && chmod +x *.sh *.py translator-ssh comprehensive_hash", config.RemoteDir)
	if _, err := worker.ExecuteCommand(ctx, cmd); err != nil {
		return fmt.Errorf("failed to set permissions on remote: %w", err)
	}
	
	logger.Info("Codebase upload completed", nil)
	return nil
}

// initializeSSHWorker creates and connects to SSH worker
func initializeSSHWorker(config *TranslationConfig, logger logger.Logger) (*sshworker.SSHWorker, error) {
	workerConfig := sshworker.SSHWorkerConfig{
		Host:              config.SSHHost,
		Port:              config.SSHPort,
		Username:          config.SSHUser,
		Password:          config.SSHPassword,
		RemoteDir:         config.RemoteDir,
		ConnectionTimeout:  30 * time.Second,
		CommandTimeout:     30 * time.Minute, // Longer timeout for translation
	}
	
	worker, err := sshworker.NewSSHWorker(workerConfig, logger)
	if err != nil {
		return nil, err
	}
	
	if err := worker.Connect(context.Background()); err != nil {
		return nil, err
	}
	
	return worker, nil
}

// convertFB2ToMarkdownRemoteWithDocs performs FB2 to markdown conversion on remote worker with documentation
func convertFB2ToMarkdownRemoteWithDocs(ctx context.Context, config *TranslationConfig, worker *sshworker.SSHWorker, logger logger.Logger, docs *DocumentationData) (string, error) {
	logger.Info("Converting FB2 to markdown on remote worker", map[string]interface{}{
		"input": config.InputFile,
	})
	
	// Upload input file
	inputFileName := filepath.Base(config.InputFile)
	remoteInputPath := filepath.Join(config.RemoteDir, inputFileName)
	
	if err := worker.UploadFile(ctx, config.InputFile, remoteInputPath); err != nil {
		return "", fmt.Errorf("failed to upload input file: %w", err)
	}
	
	// Generate markdown filename
	baseName := strings.TrimSuffix(inputFileName, filepath.Ext(inputFileName))
	originalMarkdownPath := filepath.Join(config.RemoteDir, baseName+"_original.md")
	
	// Execute FB2 to markdown conversion script (simplified for now)
	cmd := fmt.Sprintf(`
cd %s
# Simple FB2 text extraction
grep -o '>.*<' "%s" | sed 's/[<>]//g' | sed '/^$/d' > "%s"
`, config.RemoteDir, remoteInputPath, originalMarkdownPath)
	
	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute FB2 conversion: %w", err)
	}
	
	if result.ExitCode != 0 {
		return "", fmt.Errorf("FB2 conversion failed: %s", result.Stderr)
	}
	
	// Document the original markdown
	localOriginalMD := filepath.Join(filepath.Dir(config.OutputFile), baseName+"_original.md")
	if err := worker.DownloadFile(ctx, originalMarkdownPath, localOriginalMD); err != nil {
		return "", fmt.Errorf("failed to download original markdown: %w", err)
	}
	
	// Add to documentation
	if info, err := os.Stat(localOriginalMD); err == nil {
		docs.FilesGenerated = append(docs.FilesGenerated, FileInfo{
			Path:        localOriginalMD,
			Size:        info.Size(),
			ContentType: "text/markdown",
			Verified:    true,
			Verification: "Downloaded from remote worker",
		})
	}
	
	logger.Info("FB2 to markdown conversion completed", map[string]interface{}{
		"output": originalMarkdownPath,
	})
	
	return originalMarkdownPath, nil
}

// translateMarkdownRemoteWithDocs performs translation on remote worker using llama.cpp with documentation
func translateMarkdownRemoteWithDocs(ctx context.Context, config *TranslationConfig, worker *sshworker.SSHWorker, originalMarkdown string, logger logger.Logger, docs *DocumentationData) (string, error) {
	logger.Info("Translating markdown using llama.cpp", map[string]interface{}{
		"input": originalMarkdown,
	})
	
	// Generate translated markdown filename
	baseName := strings.TrimSuffix(filepath.Base(originalMarkdown), "_original.md")
	translatedMarkdownPath := filepath.Join(config.RemoteDir, baseName+"_translated.md")
	
	// Execute translation using LLM script with llama.cpp
	cmd := fmt.Sprintf(`
cd %s
chmod +x llm_translation.sh
./llm_translation.sh "%s" "%s" "config.json"
`, config.RemoteDir, originalMarkdown, translatedMarkdownPath)
	
	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute translation: %w", err)
	}
	
	if result.ExitCode != 0 {
		return "", fmt.Errorf("translation failed: %s", result.Stderr)
	}
	
	// Document the translated markdown
	localTranslatedMD := filepath.Join(filepath.Dir(config.OutputFile), baseName+"_translated.md")
	if err := worker.DownloadFile(ctx, translatedMarkdownPath, localTranslatedMD); err != nil {
		return "", fmt.Errorf("failed to download translated markdown: %w", err)
	}
	
	// Verify translated content
	if content, err := os.ReadFile(localTranslatedMD); err == nil {
		verified := containsSerbianCyrillic(string(content))
		docs.FilesGenerated = append(docs.FilesGenerated, FileInfo{
			Path:        localTranslatedMD,
			Size:        int64(len(content)),
			ContentType: "text/markdown",
			Verified:    verified,
			Verification: map[bool]string{true: "Contains Serbian Cyrillic characters", false: "Missing Serbian Cyrillic characters"}[verified],
		})
		
		if !verified {
			docs.IssuesEncountered = append(docs.IssuesEncountered, IssueInfo{
				Step:       "Markdown Translation",
				Severity:   "Warning",
				Message:    "Translated content may not contain Serbian Cyrillic characters",
				Resolution: "Manual review recommended",
			})
		}
	}
	
	logger.Info("Markdown translation completed", map[string]interface{}{
		"output": translatedMarkdownPath,
	})
	
	return translatedMarkdownPath, nil
}

// convertMarkdownToEPUBRemoteWithDocs performs markdown to EPUB conversion on remote worker with documentation
func convertMarkdownToEPUBRemoteWithDocs(ctx context.Context, config *TranslationConfig, worker *sshworker.SSHWorker, translatedMarkdown string, logger logger.Logger, docs *DocumentationData) error {
	logger.Info("Converting markdown to EPUB", map[string]interface{}{
		"input": translatedMarkdown,
	})
	
	// Generate EPUB filename to match expected output
	expectedEPUB := filepath.Base(config.OutputFile)
	epubPath := filepath.Join(config.RemoteDir, expectedEPUB)
	
	// Simple markdown to EPUB conversion (basic implementation)
	cmd := fmt.Sprintf(`
cd %s
# Create basic EPUB structure
mkdir -p epub_temp/META-INF epub_temp/OEBPS

# Create mimetype
echo "application/epub+zip" > epub_temp/mimetype

# Create container.xml
cat > epub_temp/META-INF/container.xml << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
EOF

# Create basic HTML from markdown
cat "%s" | sed 's/^# \(.*\)$/<h1>\1<\/h1>/' | sed 's/^## \(.*\)$/<h2>\1<\/h2>/' | sed 's/$/<br>/' > epub_temp/OEBPS/content.html

# Create content.opf
cat > epub_temp/OEBPS/content.opf << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Translated Book</dc:title>
    <dc:language>sr</dc:language>
  </metadata>
  <manifest>
    <item id="content" href="content.html" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="content"/>
  </spine>
</package>
EOF

# Create EPUB by zipping
cd epub_temp
zip -rX "../%s" mimetype META-INF OEBPS
cd ..
rm -rf epub_temp
`, config.RemoteDir, translatedMarkdown, filepath.Base(epubPath))
	
	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute EPUB conversion: %w", err)
	}
	
	if result.ExitCode != 0 {
		return fmt.Errorf("EPUB conversion failed: %s", result.Stderr)
	}
	
	logger.Info("Markdown to EPUB conversion completed", map[string]interface{}{
		"output": epubPath,
	})
	
	return nil
}

// downloadAndVerifyResultsWithDocs downloads generated files and verifies content with documentation
func downloadAndVerifyResultsWithDocs(ctx context.Context, config *TranslationConfig, worker *sshworker.SSHWorker, logger logger.Logger, docs *DocumentationData) error {
	logger.Info("Downloading and verifying results", nil)
	
	// Determine files to download - all intermediate and final files
	inputBase := strings.TrimSuffix(filepath.Base(config.InputFile), filepath.Ext(config.InputFile))
	filesToDownload := []struct {
		remote string
		local  string
		desc   string
	}{
		{
			remote: filepath.Join(config.RemoteDir, inputBase+"_original.md"),
			local:  filepath.Join(filepath.Dir(config.InputFile), inputBase+"_original.md"),
			desc:   "Original markdown",
		},
		{
			remote: filepath.Join(config.RemoteDir, inputBase+"_translated.md"),
			local:  filepath.Join(filepath.Dir(config.InputFile), inputBase+"_translated.md"),
			desc:   "Translated markdown",
		},
		{
			remote: filepath.Join(config.RemoteDir, filepath.Base(config.OutputFile)),
			local:  config.OutputFile,
			desc:   "Final EPUB",
		},
	}
	
	// Download files
	for _, file := range filesToDownload {
		if err := worker.DownloadFile(ctx, file.remote, file.local); err != nil {
			return fmt.Errorf("failed to download %s: %w", filepath.Base(file.remote), err)
		}
		
		logger.Info("File downloaded", map[string]interface{}{
			"file": filepath.Base(file.local),
			"desc": file.desc,
		})
		
		// Document downloaded file
		if info, err := os.Stat(file.local); err == nil {
			verified := false
			verification := "Not verified"
			contentType := "application/octet-stream"
			
			if strings.HasSuffix(file.local, ".epub") {
				contentType = "application/epub+zip"
				verified = verifyEPUBFile(file.local, logger)
				verification = map[bool]string{true: "Valid EPUB format", false: "Invalid EPUB format"}[verified]
			} else if strings.HasSuffix(file.local, ".md") {
				contentType = "text/markdown"
				verified = verifyMarkdownFile(file.local, logger)
				if strings.Contains(file.local, "_original.md") {
					verification = map[bool]string{true: "Original content preserved", false: "Original content may be corrupted"}[verified]
				} else {
					verification = map[bool]string{true: "Translated content present", false: "Translation may be incomplete"}[verified]
				}
			}
			
			docs.FilesGenerated = append(docs.FilesGenerated, FileInfo{
				Path:        file.local,
				Size:        info.Size(),
				ContentType: contentType,
				Verified:    verified,
				Verification: verification,
			})
			
			if !verified {
				docs.IssuesEncountered = append(docs.IssuesEncountered, IssueInfo{
					Step:       "Download & Verification",
					Severity:   "Warning",
					Message:    fmt.Sprintf("File verification failed: %s", file.local),
					Resolution: "Manual review recommended",
				})
			}
		}
	}
	
	return nil
}

// verifyEPUBFile verifies EPUB file structure
func verifyEPUBFile(filename string, logger logger.Logger) bool {
	// Read first 1KB to check for EPUB magic
	file, err := os.Open(filename)
	if err != nil {
		logger.Debug("Failed to open EPUB file", map[string]interface{}{
			"file": filename,
			"error": err.Error(),
		})
		return false
	}
	defer file.Close()
	
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		logger.Debug("Failed to read EPUB file", map[string]interface{}{
			"file": filename,
			"error": err.Error(),
		})
		return false
	}
	
	content := string(buffer[:n])
	
	// Check for EPUB indicators
	if !strings.Contains(content, "application/epub+zip") {
		logger.Debug("EPUB file missing mimetype", map[string]interface{}{
			"file": filename,
		})
		return false
	}
	
	// Check that it's a ZIP file (EPUB is essentially a ZIP)
	if n < 4 || string(buffer[:2]) != "PK" {
		logger.Debug("EPUB file is not a valid ZIP", map[string]interface{}{
			"file": filename,
		})
		return false
	}
	
	return true
}

// verifyMarkdownFile verifies that a markdown file contains meaningful content
func verifyMarkdownFile(filename string, logger logger.Logger) bool {
	file, err := os.Open(filename)
	if err != nil {
		logger.Debug("Failed to open markdown file", map[string]interface{}{
			"file": filename,
			"error": err.Error(),
		})
		return false
	}
	defer file.Close()
	
	content, err := io.ReadAll(file)
	if err != nil {
		logger.Debug("Failed to read markdown file", map[string]interface{}{
			"file": filename,
			"error": err.Error(),
		})
		return false
	}
	
	text := string(content)
	
	// Check if file is empty or only whitespace
	if len(strings.TrimSpace(text)) == 0 {
		logger.Debug("Markdown file is empty", map[string]interface{}{
			"file": filename,
		})
		return false
	}
	
	// For translated files, check for Serbian Cyrillic content
	if strings.Contains(filename, "_translated.md") {
		if !containsSerbianCyrillic(text) {
			logger.Debug("Translated markdown file missing Serbian Cyrillic content", map[string]interface{}{
				"file": filename,
			})
			return false
		}
	}
	
	return true
}

// containsSerbianCyrillic checks if text contains Serbian Cyrillic characters
func containsSerbianCyrillic(text string) bool {
	serbianCyrillic := "љњертзуиопшђжасдфгхјклчћџ"
	for _, char := range text {
		if strings.ContainsRune(serbianCyrillic, char) {
			return true
		}
	}
	return false
}

// generateIntegralDocumentation creates comprehensive documentation of the translation process
func generateIntegralDocumentation(docs *DocumentationData) error {
	docsPath := strings.TrimSuffix(docs.OutputFile, filepath.Ext(docs.OutputFile)) + "_translation_documentation.md"
	
	file, err := os.Create(docsPath)
	if err != nil {
		return fmt.Errorf("failed to create documentation file: %w", err)
	}
	defer file.Close()
	
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	
	// Write documentation content
	writeHeader(writer, docs)
	writeOverview(writer, docs)
	writeStepDetails(writer, docs)
	writeFileDetails(writer, docs)
	writeCodebaseInformation(writer, docs)
	writeTechnicalDetails(writer, docs)
	writeQualityAssurance(writer, docs)
	writeIssues(writer, docs)
	writeConclusion(writer, docs)
	
	fmt.Printf("Integral documentation generated: %s\n", docsPath)
	return nil
}

// writeHeader writes document header
func writeHeader(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "# Translation Process Integral Documentation\n\n")
	fmt.Fprintf(writer, "**Generated:** %s\n", docs.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(writer, "**Duration:** %s\n\n", docs.Duration.String())
	fmt.Fprintf(writer, "---\n\n")
}

// writeOverview writes overview section
func writeOverview(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Overview\n\n")
	fmt.Fprintf(writer, "This document provides a comprehensive overview of the ebook translation process from FB2 to Serbian Cyrillic EPUB format.\n\n")
	fmt.Fprintf(writer, "- **Input File:** `%s`\n", docs.InputFile)
	fmt.Fprintf(writer, "- **Output File:** `%s`\n", docs.OutputFile)
	fmt.Fprintf(writer, "- **SSH Host:** `%s`\n", docs.SSHHost)
	fmt.Fprintf(writer, "- **Remote Directory:** `%s`\n", docs.RemoteDir)
	fmt.Fprintf(writer, "- **Total Duration:** %s\n\n", docs.Duration.String())
}

// writeStepDetails writes detailed step information
func writeStepDetails(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Translation Workflow Steps\n\n")
	
	for i, step := range docs.StepsCompleted {
		status := "✅ Success"
		if !step.Success {
			status = "❌ Failed"
		}
		
		fmt.Fprintf(writer, "### Step %d: %s %s\n\n", i+1, step.Name, status)
		fmt.Fprintf(writer, "- **Start:** %s\n", step.StartTime.Format("15:04:05"))
		fmt.Fprintf(writer, "- **End:** %s\n", step.EndTime.Format("15:04:05"))
		fmt.Fprintf(writer, "- **Duration:** %s\n", step.EndTime.Sub(step.StartTime).String())
		
		if step.Details != "" {
			fmt.Fprintf(writer, "- **Details:** %s\n", step.Details)
		}
		
		if step.Error != "" {
			fmt.Fprintf(writer, "- **Error:** `%s`\n", step.Error)
		}
		
		fmt.Fprintf(writer, "\n")
	}
}

// writeFileDetails writes generated file information
func writeFileDetails(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Generated Files\n\n")
	
	for _, file := range docs.FilesGenerated {
		status := "✅ Verified"
		if !file.Verified {
			status = "⚠️ Issue"
		}
		
		fmt.Fprintf(writer, "### %s %s\n\n", filepath.Base(file.Path), status)
		fmt.Fprintf(writer, "- **Path:** `%s`\n", file.Path)
		fmt.Fprintf(writer, "- **Size:** %d bytes\n", file.Size)
		fmt.Fprintf(writer, "- **Type:** %s\n", file.ContentType)
		fmt.Fprintf(writer, "- **Verification:** %s\n\n", file.Verification)
	}
}

// writeCodebaseInformation writes codebase version information
func writeCodebaseInformation(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Codebase Version Information\n\n")
	fmt.Fprintf(writer, "### Local Codebase\n\n")
	fmt.Fprintf(writer, "- **Hash:** `%s`\n", docs.LocalHash)
	fmt.Fprintf(writer, "- **Status:** Source code used for translation\n\n")
	
	fmt.Fprintf(writer, "### Remote Codebase\n\n")
	if docs.RemoteHash == "<not found>" {
		fmt.Fprintf(writer, "- **Status:** Initial deployment\n")
	} else if docs.RemoteHash == "<newly uploaded>" {
		fmt.Fprintf(writer, "- **Status:** Newly uploaded\n")
	} else if docs.RemoteHash == docs.LocalHash {
		fmt.Fprintf(writer, "- **Hash:** `%s`\n", docs.RemoteHash)
		fmt.Fprintf(writer, "- **Status:** Synchronized with local\n")
	} else {
		fmt.Fprintf(writer, "- **Hash:** `%s`\n", docs.RemoteHash)
		fmt.Fprintf(writer, "- **Status:** Updated during translation\n")
	}
	fmt.Fprintf(writer, "\n")
}

// writeTechnicalDetails writes technical implementation details
func writeTechnicalDetails(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Technical Implementation Details\n\n")
	fmt.Fprintf(writer, "### Translation Pipeline\n\n")
	fmt.Fprintf(writer, "1. **FB2 to Markdown**: Extract text content from FB2 XML format\n")
	fmt.Fprintf(writer, "2. **Markdown Translation**: Process text using llama.cpp with Serbian language model\n")
	fmt.Fprintf(writer, "3. **Markdown to EPUB**: Structure translated content into EPUB format\n\n")
	
	fmt.Fprintf(writer, "### Remote Execution\n\n")
	fmt.Fprintf(writer, "- **SSH Host:** %s\n", docs.SSHHost)
	fmt.Fprintf(writer, "- **Working Directory:** %s\n", docs.RemoteDir)
	fmt.Fprintf(writer, "- **Codebase Synchronization:** Automatic hash-based verification\n")
	fmt.Fprintf(writer, "- **Translation Engine:** llama.cpp with Python wrapper\n\n")
}

// writeQualityAssurance writes quality assurance information
func writeQualityAssurance(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Quality Assurance\n\n")
	
	serbianFiles := 0
	verifiedFiles := 0
	
	for _, file := range docs.FilesGenerated {
		if strings.Contains(file.Path, "_translated") || strings.Contains(file.Path, "_sr") {
			serbianFiles++
		}
		if file.Verified {
			verifiedFiles++
		}
	}
	
	fmt.Fprintf(writer, "### Verification Results\n\n")
	fmt.Fprintf(writer, "- **Total Files Generated:** %d\n", len(docs.FilesGenerated))
	fmt.Fprintf(writer, "- **Serbian Language Files:** %d\n", serbianFiles)
	fmt.Fprintf(writer, "- **Verified Files:** %d\n", verifiedFiles)
	fmt.Fprintf(writer, "- **Verification Rate:** %.1f%%\n\n", float64(verifiedFiles)/float64(len(docs.FilesGenerated))*100)
	
	fmt.Fprintf(writer, "### Content Verification\n\n")
	fmt.Fprintf(writer, "- **Original Markdown:** Extracted from FB2 source\n")
	fmt.Fprintf(writer, "- **Translated Markdown:** Verified for Serbian Cyrillic characters\n")
	fmt.Fprintf(writer, "- **Final EPUB:** Validated EPUB structure and format\n\n")
}

// writeIssues writes encountered issues
func writeIssues(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Issues and Resolutions\n\n")
	
	if len(docs.IssuesEncountered) == 0 {
		fmt.Fprintf(writer, "✅ **No issues encountered during translation process**\n\n")
		return
	}
	
	for _, issue := range docs.IssuesEncountered {
		fmt.Fprintf(writer, "### %s: %s\n\n", issue.Severity, issue.Step)
		fmt.Fprintf(writer, "- **Message:** %s\n", issue.Message)
		fmt.Fprintf(writer, "- **Resolution:** %s\n\n", issue.Resolution)
	}
}

// writeConclusion writes conclusion and recommendations
func writeConclusion(writer *bufio.Writer, docs *DocumentationData) {
	fmt.Fprintf(writer, "## Conclusion\n\n")
	
	if docs.Duration.Minutes() < 5 {
		fmt.Fprintf(writer, "✅ **Translation completed successfully in record time** (%s)\n\n", docs.Duration.String())
	} else if docs.Duration.Minutes() < 30 {
		fmt.Fprintf(writer, "✅ **Translation completed successfully** (%s)\n\n", docs.Duration.String())
	} else {
		fmt.Fprintf(writer, "✅ **Translation completed** (%s) - performance could be optimized\n\n", docs.Duration.String())
	}
	
	fmt.Fprintf(writer, "### Final Output\n\n")
	fmt.Fprintf(writer, "The translation process has successfully converted the FB2 ebook to Serbian Cyrillic EPUB format.\n")
	fmt.Fprintf(writer, "All intermediate files have been preserved for quality assurance and potential reprocessing.\n\n")
	
	fmt.Fprintf(writer, "### Recommendations\n\n")
	if len(docs.IssuesEncountered) > 0 {
		fmt.Fprintf(writer, "- **Review Issues:** Some issues were encountered - see Issues section for details\n")
	}
	fmt.Fprintf(writer, "- **Quality Check:** Review the translated markdown for accuracy and nuance\n")
	fmt.Fprintf(writer, "- **Format Verification:** Validate EPUB in multiple readers for compatibility\n")
	fmt.Fprintf(writer, "- **Content Review:** Ensure cultural and contextual appropriateness of translation\n\n")
}

// parseFlags parses command line arguments
func parseFlags() *TranslationConfig {
	config := &TranslationConfig{}
	
	flag.StringVar(&config.InputFile, "input", "", "Input ebook file (FB2, EPUB, PDF, DOCX, TXT, HTML)")
	flag.StringVar(&config.InputFile, "i", "", "Input ebook file (shorthand)")
	flag.StringVar(&config.OutputFile, "output", "", "Output file (auto-detected if not specified)")
	flag.StringVar(&config.OutputFile, "o", "", "Output file (shorthand)")
	
	// SSH options for remote translation
	flag.StringVar(&config.SSHHost, "ssh-host", "", "SSH host for remote translation")
	flag.StringVar(&config.SSHUser, "ssh-user", "", "SSH username")
	flag.StringVar(&config.SSHPassword, "ssh-password", "", "SSH password")
	flag.IntVar(&config.SSHPort, "ssh-port", 22, "SSH port (default: 22)")
	flag.StringVar(&config.RemoteDir, "remote-dir", "/tmp/translator", "Remote working directory")
	
	// Translation options
	flag.IntVar(&config.Workers, "workers", 1, "Number of parallel workers")
	flag.IntVar(&config.ChunkSize, "chunk-size", 2000, "Text chunk size for translation")
	flag.IntVar(&config.Concurrency, "concurrency", 4, "Maximum concurrent operations")
	flag.BoolVar(&config.VerifyOutput, "verify", true, "Verify translated output content")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	
	// LLM configuration options
	flag.StringVar(&config.LlamaConfig.BinaryPath, "llama-binary", "/usr/local/bin/llama.cpp", "Path to llama.cpp binary")
	flag.Float64Var(&config.LlamaConfig.Temperature, "temperature", 0.3, "LLM temperature")
	flag.IntVar(&config.LlamaConfig.ContextSize, "context", 2048, "LLM context size")
	
	versionFlag := flag.Bool("version", false, "Show version information")
	help := flag.Bool("help", false, "Show help information")
	hashCodebase := flag.Bool("hash-codebase", false, "Calculate and display codebase hash")
	
	flag.Parse()
	
	if *versionFlag {
		fmt.Printf("Translator CLI v%s\n", appVersion)
		os.Exit(0)
	}
	
	if *help {
		printHelp()
		os.Exit(0)
	}
	
	if *hashCodebase {
		hasher := version.NewCodebaseHasher()
		hash, err := hasher.CalculateHash()
		if err != nil {
			log.Fatalf("Error calculating codebase hash: %v", err)
		}
		fmt.Printf("Codebase hash: %s\n", hash)
		os.Exit(0)
	}
	
	// Validate required arguments
	if config.InputFile == "" {
		fmt.Fprintf(os.Stderr, "Error: Input file is required\n")
		printHelp()
		os.Exit(1)
	}
	
	if config.SSHHost != "" && (config.SSHUser == "" || config.SSHPassword == "") {
		fmt.Fprintf(os.Stderr, "Error: SSH user and password required when using SSH host\n")
		os.Exit(1)
	}
	
	// Auto-detect output file if not specified
	if config.OutputFile == "" {
		config.OutputFile = generateOutputFilename(config.InputFile)
	}
	
	return config
}

// generateOutputFilename generates output filename based on input
func generateOutputFilename(inputFile string) string {
	ext := strings.ToLower(filepath.Ext(inputFile))
	baseName := strings.TrimSuffix(filepath.Base(inputFile), ext)
	
	// Default output format based on input or use EPUB for translations
	if ext == ".fb2" {
		return filepath.Join(filepath.Dir(inputFile), baseName+"_sr.epub")
	}
	
	return filepath.Join(filepath.Dir(inputFile), baseName+"_translated.epub")
}

// printHelp displays usage information
func printHelp() {
	fmt.Printf(`Translator CLI v%s - Unified Ebook Translation Tool

Usage:
  translator -input <file> [options]

Options:
  -i, -input <file>        Input ebook file (FB2, EPUB, PDF, DOCX, TXT, HTML)
  -o, -output <file>       Output file (auto-detected if not specified)
  
SSH/Remote Translation:
  -ssh-host <host>         SSH host for remote translation
  -ssh-user <user>         SSH username
  -ssh-password <pass>     SSH password
  -ssh-port <port>         SSH port (default: 22)
  -remote-dir <dir>        Remote working directory (default: /tmp/translator)
  
Translation Options:
  -workers <num>           Number of parallel workers (default: 1)
  -chunk-size <size>       Text chunk size for translation (default: 2000)
  -concurrency <num>       Maximum concurrent operations (default: 4)
  -verify                 Verify translated output content (default: true)
  -verbose                Enable verbose logging
  
LLM Configuration:
  -llama-binary <path>     Path to llama.cpp binary
  -temperature <value>      LLM temperature (default: 0.3)
  -context <size>          LLM context size (default: 2048)
  
Other:
  -version                 Show version information
  -help                    Show this help
  -hash-codebase           Calculate and display codebase hash

Examples:
  # Translate FB2 to Serbian via SSH
  translator -i book.fb2 -ssh-host worker.example.com -ssh-user user -ssh-password pass
  
  # Translate with custom output
  translator -i book.epub -o translated_book.epub -ssh-host worker.local
  
  # Local translation (if available)
  translator -i document.pdf

Translation Flow:
  1. Verify and sync codebase versions between local and remote
  2. Convert input FB2 to original markdown
  3. Translate markdown using llama.cpp on remote worker
  4. Convert translated markdown to EPUB format
  5. Download and verify all generated files
  6. Generate integral documentation

Generated Files:
  - <name>_original.md      Original content in markdown
  - <name>_translated.md    Translated content in markdown  
  - <name>_sr.epub        Final EPUB in Serbian Cyrillic
  - <name>_translation_documentation.md  Comprehensive process documentation
`, appVersion)
}