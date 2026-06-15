package grpc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// CoreTranslatorImpl implements the CoreTranslator interface
type CoreTranslatorImpl struct {
	logger   logger.Logger
	sessions map[string]*TranslationJob
	mutex    sync.RWMutex
}

// TranslationJob represents an active translation job
type TranslationJob struct {
	ID         string
	Status     string
	Progress   float64
	Step       string
	StartTime  time.Time
	UpdateTime time.Time

	Request *proto.TranslationRequest
	Steps   []*proto.TranslationStep
	Files   []*proto.GeneratedFile

	Context    context.Context
	CancelFunc context.CancelFunc
}

// NewCoreTranslator creates a new core translator
func NewCoreTranslator(logger logger.Logger) CoreTranslator {
	return &CoreTranslatorImpl{
		logger:   logger,
		sessions: make(map[string]*TranslationJob),
	}
}

// Translate executes a translation job
func (ct *CoreTranslatorImpl) Translate(ctx context.Context, req *proto.TranslationRequest, eventBus *events.EventBus) (*proto.TranslationStatusResponse, error) {
	ct.logger.Info("Starting core translation", map[string]interface{}{
		"session_id": req.SessionId,
		"input_file": req.InputFile,
		"provider":   req.ProviderConfig.Type,
	})

	// Create job context with cancellation
	jobCtx, cancel := context.WithCancel(ctx)

	// Create translation job
	job := &TranslationJob{
		ID:         req.SessionId,
		Status:     "running",
		Progress:   0,
		Step:       "initializing",
		StartTime:  time.Now(),
		UpdateTime: time.Now(),
		Request:    req,
		Steps:      make([]*proto.TranslationStep, 0),
		Files:      make([]*proto.GeneratedFile, 0),
		Context:    jobCtx,
		CancelFunc: cancel,
	}

	// Store job
	ct.mutex.Lock()
	ct.sessions[req.SessionId] = job
	ct.mutex.Unlock()

	// Execute translation pipeline
	result, err := ct.executeTranslationPipeline(job, eventBus)

	// Update job status
	ct.mutex.Lock()
	if err != nil {
		job.Status = "failed"
		ct.logger.Error("Translation failed", map[string]interface{}{
			"session_id": req.SessionId,
			"error":      err.Error(),
		})
	} else {
		job.Status = "completed"
		job.Progress = 100
		ct.logger.Info("Translation completed", map[string]interface{}{
			"session_id": req.SessionId,
			"files":      len(job.Files),
		})
	}
	job.UpdateTime = time.Now()
	ct.mutex.Unlock()

	return result, err
}

// executeTranslationPipeline runs the full translation pipeline
func (ct *CoreTranslatorImpl) executeTranslationPipeline(job *TranslationJob, eventBus *events.EventBus) (*proto.TranslationStatusResponse, error) {
	req := job.Request

	// Step 1: Parse input ebook
	step := ct.createStep("parsing", "Parsing input ebook")
	ct.updateJobStep(job, step)

	content, format, err := ct.parseInputFile(req.InputFile)
	if err != nil {
		ct.failStep(step, err)
		return ct.createErrorResponse(job, step), err
	}
	ct.completeStep(step)

	// Step 2: Convert to markdown
	step = ct.createStep("markdown_conversion", "Converting to markdown")
	ct.updateJobStep(job, step)

	originalMarkdown, err := ct.convertToMarkdown(content, format)
	if err != nil {
		ct.failStep(step, err)
		return ct.createErrorResponse(job, step), err
	}

	// Save original markdown
	originalMDPath := ct.generatePath(req.InputFile, "_original.md")
	if err := os.WriteFile(originalMDPath, []byte(originalMarkdown), 0644); err != nil {
		ct.failStep(step, fmt.Errorf("failed to save original markdown: %w", err))
		return ct.createErrorResponse(job, step), err
	}

	ct.addGeneratedFile(job, originalMDPath, "original_md", int64(len(originalMarkdown)), true, "Saved successfully")
	ct.completeStep(step)

	// Step 3: Translate content
	step = ct.createStep("translation", fmt.Sprintf("Translating with %s", req.ProviderConfig.Type))
	ct.updateJobStep(job, step)

	translatedMarkdown, err := ct.executeTranslation(job, originalMarkdown, eventBus)
	if err != nil {
		ct.failStep(step, err)
		return ct.createErrorResponse(job, step), err
	}

	// Save translated markdown
	translatedMDPath := ct.generatePath(req.InputFile, "_translated.md")
	if err := os.WriteFile(translatedMDPath, []byte(translatedMarkdown), 0644); err != nil {
		ct.failStep(step, fmt.Errorf("failed to save translated markdown: %w", err))
		return ct.createErrorResponse(job, step), err
	}

	verified := ct.verifyTranslation(translatedMarkdown, req.TargetLang, req.Script)
	ct.addGeneratedFile(job, translatedMDPath, "translated_md", int64(len(translatedMarkdown)), verified,
		map[bool]string{true: "Translation quality verified", false: "Translation needs review"}[verified])
	ct.completeStep(step)

	// Step 4: Generate EPUB
	step = ct.createStep("epub_generation", "Generating EPUB")
	ct.updateJobStep(job, step)

	outputPath := req.OutputFile
	if err := ct.generateEPUB(translatedMarkdown, outputPath, req.InputFile); err != nil {
		ct.failStep(step, err)
		return ct.createErrorResponse(job, step), err
	}

	// Verify EPUB
	epubVerified := ct.verifyEPUB(outputPath)
	epubSize := ct.getFileSize(outputPath)
	ct.addGeneratedFile(job, outputPath, "epub", epubSize, epubVerified,
		map[bool]string{true: "Valid EPUB format", false: "Invalid EPUB format"}[epubVerified])
	ct.completeStep(step)

	// Generate session report. req.Options is a proto3 message field and may be
	// nil on the wire (StartTranslation only validates ProviderConfig + session_id,
	// not Options). A direct req.Options.EnableMonitoring deref panics on a
	// well-formed request whose Options is unset — and by this point the EPUB has
	// already been written, so the panic crashes the translation goroutine after
	// the real work completed, leaving the session stuck "running" and the output
	// silently unreported. GetOptions() is the nil-safe proto accessor (returns a
	// zero-value *TranslationOptions when unset, whose EnableMonitoring is false).
	if req.GetOptions().GetEnableMonitoring() {
		ct.generateSessionReport(job)
	}

	// Return success response
	return &proto.TranslationStatusResponse{
		SessionId:          job.ID,
		Status:             "completed",
		ProgressPercentage: 100,
		CurrentStep:        "completed",
		Message:            "Translation completed successfully",
		StartedAt:          timeToProto(job.StartTime),
		UpdatedAt:          timeToProto(time.Now()),
		Files:              job.Files,
		Steps:              job.Steps,
	}, nil
}

// executeTranslation handles translation based on provider type
func (ct *CoreTranslatorImpl) executeTranslation(job *TranslationJob, text string, eventBus *events.EventBus) (string, error) {
	req := job.Request

	switch req.ProviderConfig.Type {
	case "ssh":
		return ct.executeSSHTranslation(job, text, eventBus)
	case "llamacpp":
		return ct.executeLlamaCppTranslation(job, text, eventBus)
	default:
		return ct.executeAPITranslation(job, text, eventBus)
	}
}

// executeSSHTranslation uses SSH worker
func (ct *CoreTranslatorImpl) executeSSHTranslation(job *TranslationJob, text string, eventBus *events.EventBus) (string, error) {
	req := job.Request
	ctx := job.Context

	ct.logger.Info("Executing SSH translation", map[string]interface{}{
		"session_id": job.ID,
		"host":       req.ProviderConfig.SshHost,
	})

	// Initialize SSH worker
	workerConfig := sshworker.SSHWorkerConfig{
		Host:              req.ProviderConfig.SshHost,
		Port:              int(req.ProviderConfig.SshPort),
		Username:          req.ProviderConfig.SshUser,
		Password:          req.ProviderConfig.SshPassword,
		RemoteDir:         req.ProviderConfig.RemoteDir,
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    time.Duration(req.ProviderConfig.TimeoutSeconds) * time.Second,
	}

	worker, err := sshworker.NewSSHWorker(workerConfig, ct.logger)
	if err != nil {
		return "", fmt.Errorf("failed to create SSH worker: %w", err)
	}
	defer worker.Close()

	if err := worker.Connect(ctx); err != nil {
		return "", fmt.Errorf("failed to connect to SSH worker: %w", err)
	}

	// Upload text to remote
	remoteTextPath := filepath.Join(req.ProviderConfig.RemoteDir, "input.md")
	if err := worker.UploadData(ctx, []byte(text), remoteTextPath); err != nil {
		return "", fmt.Errorf("failed to upload text to remote: %w", err)
	}

	// Emit progress
	ct.emitProgress(eventBus, job.ID, "upload_complete", "translation", 10, "Text uploaded to remote worker")

	// Execute translation using remote llama.cpp
	remoteOutputPath := filepath.Join(req.ProviderConfig.RemoteDir, "output.md")
	cmd := ct.buildSSHCommand(req.ProviderConfig, remoteTextPath, remoteOutputPath)

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute remote translation: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("remote translation failed: %s", result.Stderr)
	}

	// Emit progress
	ct.emitProgress(eventBus, job.ID, "translation_complete", "translation", 80, "Translation completed on remote worker")
	// Download result to temp file
	tempFile := ct.generatePath(req.InputFile, "_translated.md")
	if err := worker.DownloadFile(ctx, remoteOutputPath, tempFile); err != nil {
		return "", fmt.Errorf("failed to download translation result: %w", err)
	}
	translatedData, err := os.ReadFile(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to read downloaded translation: %w", err)
	}
	_ = os.Remove(tempFile)

	return string(translatedData), nil
}

// executeLlamaCppTranslation uses local llama.cpp
func (ct *CoreTranslatorImpl) executeLlamaCppTranslation(job *TranslationJob, text string, eventBus *events.EventBus) (string, error) {
	req := job.Request

	ct.logger.Info("Executing local llama.cpp translation", map[string]interface{}{
		"session_id": job.ID,
		"binary":     req.ProviderConfig.LlamaBinary,
		"model":      req.ProviderConfig.LlamaModel,
	})

	// Create LLM translator
	llmConfig := translator.TranslationConfig{
		SourceLang:  req.SourceLang,
		TargetLang:  req.TargetLang,
		Provider:    "llamacpp",
		Model:       req.ProviderConfig.Model,
		Temperature: req.ProviderConfig.Temperature,
		MaxTokens:   int(req.ProviderConfig.MaxTokens),
		Timeout:     time.Duration(req.ProviderConfig.TimeoutSeconds) * time.Second,
		Options: map[string]interface{}{
			"binary_path":  req.ProviderConfig.LlamaBinary,
			"model_path":   req.ProviderConfig.LlamaModel,
			"context_size": req.ProviderConfig.ContextSize,
		},
	}

	llmTranslator, err := llm.NewLLMTranslator(llmConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create LLM translator: %w", err)
	}

	ct.emitProgress(eventBus, job.ID, "llm_ready", "translation", 10, "LLM translator initialized")

	// Translate
	result, err := llmTranslator.TranslateWithProgress(job.Context, text, "Ebook content", eventBus, job.ID)
	if err != nil {
		return "", fmt.Errorf("LLM translation failed: %w", err)
	}

	ct.emitProgress(eventBus, job.ID, "translation_complete", "translation", 90, "Translation completed")

	return result, nil
}

// executeAPITranslation uses API-based providers
func (ct *CoreTranslatorImpl) executeAPITranslation(job *TranslationJob, text string, eventBus *events.EventBus) (string, error) {
	req := job.Request

	ct.logger.Info("Executing API translation", map[string]interface{}{
		"session_id": job.ID,
		"provider":   req.ProviderConfig.Type,
		"model":      req.ProviderConfig.Model,
	})

	// Create LLM translator
	llmConfig := translator.TranslationConfig{
		SourceLang:  req.SourceLang,
		TargetLang:  req.TargetLang,
		Provider:    req.ProviderConfig.Type,
		Model:       req.ProviderConfig.Model,
		Temperature: req.ProviderConfig.Temperature,
		MaxTokens:   int(req.ProviderConfig.MaxTokens),
		Timeout:     time.Duration(req.ProviderConfig.TimeoutSeconds) * time.Second,
		APIKey:      req.ProviderConfig.ApiKey,
		BaseURL:     req.ProviderConfig.BaseUrl,
	}

	llmTranslator, err := llm.NewLLMTranslator(llmConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create LLM translator: %w", err)
	}

	ct.emitProgress(eventBus, job.ID, "api_ready", "translation", 10, "API client initialized")

	// Translate
	result, err := llmTranslator.TranslateWithProgress(job.Context, text, "Ebook content", eventBus, job.ID)
	if err != nil {
		return "", fmt.Errorf("API translation failed: %w", err)
	}

	ct.emitProgress(eventBus, job.ID, "translation_complete", "translation", 90, "Translation completed")

	return result, nil
}

// Helper methods

func (ct *CoreTranslatorImpl) parseInputFile(filePath string) (string, string, error) {
	detector := format.NewDetector()
	f, err := detector.DetectFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to detect format: %w", err)
	}
	parser := ebook.NewUniversalParser()
	book, err := parser.Parse(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse file: %w", err)
	}
	return book.ExtractText(), f.String(), nil
}

func (ct *CoreTranslatorImpl) convertToMarkdown(content, format string) (string, error) {
	switch format {
	case "epub":
		// Write content to temp file and convert
		tempEpub := filepath.Join(os.TempDir(), "input.epub")
		if err := os.WriteFile(tempEpub, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write temp epub: %w", err)
		}
		defer os.Remove(tempEpub)
		tempMD := filepath.Join(os.TempDir(), "output.md")
		converter := markdown.NewEPUBToMarkdownConverter(true, "")
		if err := converter.ConvertEPUBToMarkdown(tempEpub, tempMD); err != nil {
			return "", fmt.Errorf("failed to convert epub to markdown: %w", err)
		}
		mdBytes, err := os.ReadFile(tempMD)
		if err != nil {
			return "", fmt.Errorf("failed to read markdown: %w", err)
		}
		_ = os.Remove(tempMD)
		return string(mdBytes), nil
	default:
		return content, nil
	}
}

func (ct *CoreTranslatorImpl) generatePath(inputFile, suffix string) string {
	ext := filepath.Ext(inputFile)
	baseName := strings.TrimSuffix(filepath.Base(inputFile), ext)
	return filepath.Join(filepath.Dir(inputFile), baseName+suffix)
}

func (ct *CoreTranslatorImpl) verifyTranslation(text, targetLang, script string) bool {
	if targetLang == "sr" && script == "cyrillic" {
		serbianCyrillic := "љњертзуиопшђжасдфгхјклчћџ"
		for _, char := range text {
			if strings.ContainsRune(serbianCyrillic, char) {
				return true
			}
		}
		return false
	}
	return len(strings.TrimSpace(text)) > 0
}

func (ct *CoreTranslatorImpl) generateEPUB(content, outputPath, inputFile string) error {
	// Write markdown to temp file then convert
	tempMD := filepath.Join(os.TempDir(), "input.md")
	if err := os.WriteFile(tempMD, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp markdown: %w", err)
	}
	defer os.Remove(tempMD)
	generator := markdown.NewMarkdownToEPUBConverter()
	return generator.ConvertMarkdownToEPUB(tempMD, outputPath)
}

func (ct *CoreTranslatorImpl) verifyEPUB(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil {
		return false
	}

	content := string(buffer[:n])
	return strings.Contains(content, "application/epub+zip") && string(buffer[:2]) == "PK"
}

func (ct *CoreTranslatorImpl) getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (ct *CoreTranslatorImpl) buildSSHCommand(config *proto.ProviderConfig, inputPath, outputPath string) string {
	// Build SSH command based on available models and binaries
	cmd := fmt.Sprintf("cd %s && /home/milosvasic/llama.cpp -m /home/milosvasic/models/tiny-llama-working.gguf -p 'Translate from Russian to Serbian Cyrillic: ' -f %s > %s",
		config.RemoteDir, inputPath, outputPath)
	return cmd
}

func (ct *CoreTranslatorImpl) generateSessionReport(job *TranslationJob) {
	reportPath := ct.generatePath(job.Request.OutputFile, "_session_report.md")

	file, err := os.Create(reportPath)
	if err != nil {
		return
	}
	defer file.Close()

	// Basic report generation - this could be enhanced
	file.WriteString("# Translation Session Report\n\n")
	file.WriteString(fmt.Sprintf("**Session ID:** %s\n", job.ID))
	file.WriteString(fmt.Sprintf("**Status:** %s\n", job.Status))
	file.WriteString(fmt.Sprintf("**Provider:** %s\n", job.Request.ProviderConfig.Type))
	file.WriteString(fmt.Sprintf("**Duration:** %s\n", time.Since(job.StartTime).String()))
	file.WriteString(fmt.Sprintf("**Files Generated:** %d\n\n", len(job.Files)))

	for i, step := range job.Steps {
		file.WriteString(fmt.Sprintf("## Step %d: %s\n", i+1, step.Name))
		file.WriteString(fmt.Sprintf("- **Status:** %s\n", step.Status))
		file.WriteString(fmt.Sprintf("- **Duration:** %s\n", step.EndedAt.AsTime().Sub(step.StartedAt.AsTime()).String()))
		if step.ErrorMessage != "" {
			file.WriteString(fmt.Sprintf("- **Error:** %s\n", step.ErrorMessage))
		}
		file.WriteString("\n")
	}
}

// Step management

func (ct *CoreTranslatorImpl) createStep(name, description string) *proto.TranslationStep {
	return &proto.TranslationStep{
		Name:      name,
		Status:    "running",
		StartedAt: timeToProto(time.Now()),
		Message:   description,
	}
}

func (ct *CoreTranslatorImpl) updateJobStep(job *TranslationJob, step *proto.TranslationStep) {
	// These job fields (Step / Steps / Progress / UpdateTime) are read concurrently
	// by GetStatus / Cancel under ct.mutex while a translation is in flight (a
	// status poll of a running job). The pipeline ran these mutations unlocked,
	// racing the locked reader (proven by -race). Guard them under the same lock.
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	job.Step = step.Name
	job.Steps = append(job.Steps, step)

	// Calculate progress based on step index
	stepIndex := len(job.Steps)
	job.Progress = float64(stepIndex-1) / 4.0 * 100 // 4 total steps
	job.UpdateTime = time.Now()
}

func (ct *CoreTranslatorImpl) completeStep(step *proto.TranslationStep) {
	// step is already inside job.Steps, which GetStatus exposes by reference, so its
	// fields are read concurrently by the status RPC. Mutate under ct.mutex.
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	step.Status = "completed"
	step.EndedAt = timeToProto(time.Now())
}

func (ct *CoreTranslatorImpl) failStep(step *proto.TranslationStep, err error) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	step.Status = "failed"
	step.EndedAt = timeToProto(time.Now())
	step.ErrorMessage = err.Error()
}

func (ct *CoreTranslatorImpl) addGeneratedFile(job *TranslationJob, path, fileType string, size int64, verified bool, verification string) {
	file := &proto.GeneratedFile{
		Path:                path,
		Type:                fileType,
		Size:                size,
		ContentType:         ct.getContentType(fileType),
		Verified:            verified,
		VerificationMessage: verification,
		CreatedAt:           timeToProto(time.Now()),
	}
	// job.Files is read concurrently by GetStatus under ct.mutex; appending here
	// without the lock raced the reader (proven by -race). Guard the append.
	ct.mutex.Lock()
	job.Files = append(job.Files, file)
	ct.mutex.Unlock()
}

func (ct *CoreTranslatorImpl) getContentType(fileType string) string {
	switch fileType {
	case "original_md", "translated_md":
		return "text/markdown"
	case "epub":
		return "application/epub+zip"
	case "report":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

func (ct *CoreTranslatorImpl) createErrorResponse(job *TranslationJob, failedStep *proto.TranslationStep) *proto.TranslationStatusResponse {
	return &proto.TranslationStatusResponse{
		SessionId:          job.ID,
		Status:             "failed",
		ProgressPercentage: job.Progress,
		CurrentStep:        failedStep.Name,
		Message:            failedStep.ErrorMessage,
		StartedAt:          timeToProto(job.StartTime),
		UpdatedAt:          timeToProto(time.Now()),
		Files:              job.Files,
		Steps:              job.Steps,
		ErrorMessage:       failedStep.ErrorMessage,
		ErrorCode:          500,
	}
}

func (ct *CoreTranslatorImpl) emitProgress(eventBus *events.EventBus, sessionID, eventType, stepName string, progress float64, message string) {
	if eventBus == nil {
		return
	}

	event := events.NewEvent(events.EventType(eventType), message, map[string]interface{}{
		"session_id": sessionID,
		"step_name":  stepName,
		"progress":   progress,
	})
	event.SessionID = sessionID
	eventBus.Publish(event)
}

// Interface methods

func (ct *CoreTranslatorImpl) Cancel(sessionID string) error {
	ct.mutex.RLock()
	job, exists := ct.sessions[sessionID]
	ct.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("translation session not found: %s", sessionID)
	}

	if job.CancelFunc != nil {
		job.CancelFunc()
	}

	ct.mutex.Lock()
	job.Status = "cancelled"
	job.UpdateTime = time.Now()
	ct.mutex.Unlock()

	return nil
}

func (ct *CoreTranslatorImpl) GetStatus(sessionID string) (*proto.TranslationStatusResponse, error) {
	// Hold the read lock across the FIELD reads, not merely the map lookup. The
	// previous code released the lock right after the lookup, then read the job's
	// mutable fields (Status / Progress / Step / Files / Steps) lock-free — those
	// fields are written concurrently by the running pipeline (updateJobStep /
	// addGeneratedFile / completeStep / failStep) and by Cancel, so a status poll
	// of an in-flight job raced the writers (proven by -race). The reads are pure
	// in-memory field copies (no I/O), so holding RLock across them is safe.
	ct.mutex.RLock()
	defer ct.mutex.RUnlock()

	job, exists := ct.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("translation session not found: %s", sessionID)
	}

	return &proto.TranslationStatusResponse{
		SessionId:          job.ID,
		Status:             job.Status,
		ProgressPercentage: job.Progress,
		CurrentStep:        job.Step,
		StartedAt:          timeToProto(job.StartTime),
		UpdatedAt:          timeToProto(job.UpdateTime),
		Files:              job.Files,
		Steps:              job.Steps,
	}, nil
}
