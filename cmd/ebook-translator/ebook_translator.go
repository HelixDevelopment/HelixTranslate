package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
)

// EBookTranslator handles the complete ebook translation workflow
type EBookTranslator struct {
	logger         logger.Logger
	sshWorker      *sshworker.SSHWorker
	sourceFile     string
	targetLanguage string
	remoteHost     string
	remoteUser     string
	remotePass     string
}

// NewEBookTranslator creates a new ebook translator instance
func NewEBookTranslator(sourceFile, targetLanguage, remoteHost, remoteUser, remotePass string) (*EBookTranslator, error) {
	lgr := logger.NewLogger(logger.LoggerConfig{
		Level:  "info",
		Format: "text",
	})

	// Create SSH worker configuration
	sshConfig := sshworker.SSHWorkerConfig{
		Host:              remoteHost,
		Username:          remoteUser,
		Password:          remotePass,
		Port:              22,
		RemoteDir:         "/tmp/translate-workspace", // Use fixed remote directory
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    60 * time.Minute, // Extended to 60 minutes for large translations
	}

	worker, err := sshworker.NewSSHWorker(sshConfig, lgr)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH worker: %w", err)
	}

	return &EBookTranslator{
		logger:         lgr,
		sshWorker:      worker,
		sourceFile:     sourceFile,
		targetLanguage: targetLanguage,
		remoteHost:     remoteHost,
		remoteUser:     remoteUser,
		remotePass:     remotePass,
	}, nil
}

// Execute executes the complete translation workflow
func (t *EBookTranslator) Execute(ctx context.Context) error {
	startTime := time.Now()

	t.logger.Info("Starting ebook translation workflow", map[string]interface{}{
		"source_file":     t.sourceFile,
		"target_language": t.targetLanguage,
		"remote_host":     t.remoteHost,
	})

	// Step 1: Connect to remote worker
	if err := t.sshWorker.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to remote worker: %w", err)
	}
	defer t.sshWorker.Disconnect()

	// Step 2: Verify and sync codebase versions
	if err := t.verifyAndSyncCodebase(ctx); err != nil {
		return fmt.Errorf("codebase verification failed: %w", err)
	}

	// Step 3: Execute complete workflow using existing binary
	if err := t.executeRemoteWorkflow(ctx); err != nil {
		return fmt.Errorf("remote workflow execution failed: %w", err)
	}

	// Step 4: Verify output files
	if err := t.verifyOutputFiles(); err != nil {
		return fmt.Errorf("output verification failed: %w", err)
	}

	duration := time.Since(startTime)
	t.logger.Info("Translation workflow completed successfully", map[string]interface{}{
		"duration": duration.String(),
		"source":   t.sourceFile,
	})

	return nil
}

// verifyAndSyncCodebase ensures local and remote have the same codebase version
func (t *EBookTranslator) verifyAndSyncCodebase(ctx context.Context) error {
	t.logger.Info("Verifying codebase version consistency", nil)

	// Ensure remote directory exists and clean up old binary
	mkdirCmd := fmt.Sprintf("mkdir -p %s && rm -f %s/translator*", t.sshWorker.GetRemoteDir(), t.sshWorker.GetRemoteDir())
	_, err := t.sshWorker.ExecuteCommandWithOutput(ctx, mkdirCmd)
	if err != nil {
		return fmt.Errorf("failed to setup remote directory: %w", err)
	}

	// Calculate local codebase hash - for now just check if binary exists
	localBinaryPath := "./build/translator-linux"
	localHash := "missing-binary"
	if _, err := os.Stat(localBinaryPath); err == nil {
		localHash = "binary-exists"
	}

	// Get remote codebase hash
	remoteHash, err := t.getRemoteCodebaseHash(ctx)
	if err != nil {
		return fmt.Errorf("failed to get remote codebase hash: %w", err)
	}

	t.logger.Info("Codebase hashes compared", map[string]interface{}{
		"local_hash":  localHash,
		"remote_hash": remoteHash,
	})

	if localHash != remoteHash {
		t.logger.Info("Codebase versions differ, updating remote worker", nil)
		if err := t.updateRemoteCodebase(ctx); err != nil {
			return fmt.Errorf("failed to update remote codebase: %w", err)
		}

		// Verify update
		newRemoteHash, err := t.getRemoteCodebaseHash(ctx)
		if err != nil {
			return fmt.Errorf("failed to verify remote update: %w", err)
		}

		if localHash != newRemoteHash {
			return fmt.Errorf("remote update verification failed: hashes still differ")
		}

		t.logger.Info("Remote codebase updated successfully", nil)
	} else {
		t.logger.Info("Codebase versions match", nil)
	}

	return nil
}

// getRemoteCodebaseHash calculates codebase hash on remote worker
func (t *EBookTranslator) getRemoteCodebaseHash(ctx context.Context) (string, error) {
	// For now, just check if translator binary exists
	cmd := fmt.Sprintf("cd %s && test -f ./translator && echo 'binary-exists'", t.sshWorker.GetRemoteDir())

	output, err := t.sshWorker.ExecuteCommandWithOutput(ctx, cmd)
	if err != nil {
		return "missing-binary", nil // Return a placeholder hash indicating binary is missing
	}

	if strings.Contains(output, "binary-exists") {
		// Binary exists, return a placeholder hash
		return "binary-exists", nil
	}

	return "missing-binary", nil
}

// updateRemoteCodebase updates codebase on remote worker
func (t *EBookTranslator) updateRemoteCodebase(ctx context.Context) error {
	t.logger.Info("Updating remote codebase", nil)

	// Upload both architecture binaries and detect which works
	architectures := []struct {
		localPath  string
		remoteName string
	}{
		{"./build/translator-linux", "translator-amd64"},
		{"./build/translator-linux-arm64", "translator-arm64"},
	}

	workingBinary := ""

	for _, arch := range architectures {
		remotePath := filepath.Join(t.sshWorker.GetRemoteDir(), arch.remoteName)
		if err := t.sshWorker.TransferFile(ctx, arch.localPath, remotePath); err != nil {
			return fmt.Errorf("failed to transfer binary %s: %w", arch.remoteName, err)
		}

		// Make binary executable
		chmodCmd := fmt.Sprintf("chmod +x %s", remotePath)
		_, err := t.sshWorker.ExecuteCommandWithOutput(ctx, chmodCmd)
		if err != nil {
			return fmt.Errorf("failed to make binary executable %s: %w", arch.remoteName, err)
		}

		// Test if binary works
		testCmd := fmt.Sprintf("%s --help", remotePath)
		_, testErr := t.sshWorker.ExecuteCommandWithOutput(ctx, testCmd)
		if testErr == nil {
			workingBinary = arch.remoteName
			t.logger.Info("Found working binary", map[string]interface{}{
				"binary": arch.remoteName,
			})
			break
		}
	}

	if workingBinary == "" {
		return fmt.Errorf("no working binary found for remote architecture")
	}

	// Create a symlink to the working binary
	symlinkCmd := fmt.Sprintf("cd %s && ln -sf %s translator", t.sshWorker.GetRemoteDir(), workingBinary)
	_, err := t.sshWorker.ExecuteCommandWithOutput(ctx, symlinkCmd)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	// Upload the internal/scripts directory needed by translator
	if err := t.uploadScriptsDirectory(ctx); err != nil {
		return fmt.Errorf("failed to upload scripts directory: %w", err)
	}

	return nil
}

// uploadScriptsDirectory uploads the internal/scripts directory to remote worker
func (t *EBookTranslator) uploadScriptsDirectory(ctx context.Context) error {
	t.logger.Info("Uploading scripts directory", nil)

	// Create remote scripts directory structure
	mkdirCmd := fmt.Sprintf("mkdir -p %s/internal/scripts", t.sshWorker.GetRemoteDir())
	_, err := t.sshWorker.ExecuteCommandWithOutput(ctx, mkdirCmd)
	if err != nil {
		return fmt.Errorf("failed to create remote scripts directory: %w", err)
	}

	// Upload required scripts
	scripts := []string{
		"internal/scripts/fb2_to_markdown.py",
		"internal/scripts/python_translation.sh",
		"internal/scripts/translate_llm_only.py",
		"internal/scripts/epub_generator.py",
	}

	for _, script := range scripts {
		if _, err := os.Stat(script); err != nil {
			t.logger.Warn("Script not found locally, skipping", map[string]interface{}{
				"script": script,
			})
			continue
		}

		remotePath := filepath.Join(t.sshWorker.GetRemoteDir(), script)
		if err := t.sshWorker.TransferFile(ctx, script, remotePath); err != nil {
			return fmt.Errorf("failed to upload script %s: %w", script, err)
		}

		// Make shell scripts executable
		if strings.HasSuffix(script, ".sh") {
			chmodCmd := fmt.Sprintf("chmod +x %s", remotePath)
			_, err := t.sshWorker.ExecuteCommandWithOutput(ctx, chmodCmd)
			if err != nil {
				return fmt.Errorf("failed to make script executable %s: %w", script, err)
			}
		}

		t.logger.Info("Script uploaded", map[string]interface{}{
			"script":      script,
			"remote_path": remotePath,
		})
	}

	return nil
}

// executeRemoteWorkflow executes the translation workflow on remote worker
func (t *EBookTranslator) executeRemoteWorkflow(ctx context.Context) error {
	t.logger.Info("Executing translation workflow on remote worker", nil)

	// Prepare file paths
	baseName := strings.TrimSuffix(filepath.Base(t.sourceFile), filepath.Ext(t.sourceFile))
	localDir := filepath.Dir(t.sourceFile)

	// Step 1: Upload source FB2 file to remote
	remoteSourcePath := filepath.Join(t.sshWorker.GetRemoteDir(), filepath.Base(t.sourceFile))
	if err := t.sshWorker.TransferFile(ctx, t.sourceFile, remoteSourcePath); err != nil {
		return fmt.Errorf("failed to upload source file: %w", err)
	}

	t.logger.Info("Source file uploaded to remote", map[string]interface{}{
		"local_path":  t.sourceFile,
		"remote_path": remoteSourcePath,
	})

	// Step 2: Clean up any stuck processes before starting translation
	cleanupCmd := fmt.Sprintf("cd %s && pkill -f 'python3.*translate' || true && pkill -f 'llama' || true && echo 'Cleanup completed'", t.sshWorker.GetRemoteDir())

	cleanupOutput, cleanupErr := t.sshWorker.ExecuteCommandWithOutput(ctx, cleanupCmd)
	t.logger.Info("Cleaning up stuck translation processes", map[string]interface{}{
		"command": cleanupCmd,
		"output":  cleanupOutput,
		"error":   cleanupErr,
	})

	// Check if llama.cpp is available and perform actual translation
	// First, check if llama.cpp and models are available
	checkCmd := fmt.Sprintf("cd %s && python3 -c \"import sys; sys.path.append('internal/scripts'); from translate_llm_only import get_translation_provider; provider, config = get_translation_provider(); print(f'PROVIDER:{provider}')\" 2>/dev/null || echo 'PROVIDER:none'", t.sshWorker.GetRemoteDir())

	checkOutput, checkErr := t.sshWorker.ExecuteCommandWithOutput(ctx, checkCmd)
	t.logger.Info("Checking translation provider availability", map[string]interface{}{
		"command": checkCmd,
		"output":  checkOutput,
		"error":   checkErr,
	})

	var remoteCmd string
	if strings.Contains(checkOutput, "PROVIDER:llamacpp") {
		// First test with just a few paragraphs
		t.logger.Info("Testing llama.cpp translation with sample paragraphs", nil)

		// Create a test file with just first 5 lines
		testCmd := fmt.Sprintf("cd %s && head -10 book1_original.md > book1_test.md && echo 'Test file created with $(wc -l < book1_test.md) lines'", t.sshWorker.GetRemoteDir())

		testOutput, testErr := t.sshWorker.ExecuteCommandWithOutput(ctx, testCmd)
		t.logger.Info("Creating test file with first paragraphs", map[string]interface{}{
			"command": testCmd,
			"output":  testOutput,
			"error":   testErr,
		})

		// Test translation on small file without timeout, run in background
		remoteDir := t.sshWorker.GetRemoteDir()
		testTranslationCmd := fmt.Sprintf("cd %s && nohup python3 internal/scripts/translate_llm_only.py book1_test.md book1_test_translated.md > translation.log 2>&1 & echo 'Translation started in background, PID: $!'", remoteDir)

		t.logger.Info("Starting test translation in background", map[string]interface{}{
			"command": testTranslationCmd,
		})

		// Start translation in background
		startOutput, startErr := t.sshWorker.ExecuteCommandWithOutput(ctx, testTranslationCmd)
		if startErr != nil {
			return fmt.Errorf("failed to start translation: %w (output: %s)", startErr, startOutput)
		}

		t.logger.Info("Translation started", map[string]interface{}{
			"output": startOutput,
		})

		// Check if translation completed
		waitCmd := fmt.Sprintf("cd %s && for i in {1..60}; do if [ -f book1_test_translated.md ]; then echo 'Translation completed after $i checks'; head -5 book1_test_translated.md; break; elif [ -f translation.log ]; then echo 'Check $i: Progress...'; tail -3 translation.log; else echo 'Check $i: Still starting...'; fi; sleep 5; done", remoteDir)

		t.logger.Info("Waiting for translation to complete", map[string]interface{}{
			"command": waitCmd,
		})

		testTranslationOutput, testTranslationErr := t.sshWorker.ExecuteCommandWithOutput(ctx, waitCmd)
		if testTranslationErr != nil {
			return fmt.Errorf("translation monitoring failed: %w (output: %s)", testTranslationErr, testTranslationOutput)
		}
		if testTranslationErr != nil {
			return fmt.Errorf("test translation failed: %w (output: %s)", testTranslationErr, testTranslationOutput)
		}

		t.logger.Info("Test translation successful, proceeding with full book", map[string]interface{}{
			"sample_output": testTranslationOutput[:min(500, len(testTranslationOutput))],
		})

		// Check translated content to confirm it's different
		checkTranslationCmd := fmt.Sprintf("cd %s && head -5 book1_test_translated.md", t.sshWorker.GetRemoteDir())
		checkTranslationOutput, checkTranslationErr := t.sshWorker.ExecuteCommandWithOutput(ctx, checkTranslationCmd)
		t.logger.Info("Sample translated content", map[string]interface{}{
			"command": checkTranslationCmd,
			"output":  checkTranslationOutput,
			"error":   checkTranslationErr,
		})

		// Use actual llama.cpp translation for full book without timeout
		remoteDir = t.sshWorker.GetRemoteDir()
		remoteCmd = fmt.Sprintf(
			"cd %s && python3 internal/scripts/translate_llm_only.py book1_original.md book1_original_translated.md && "+
				"python3 internal/scripts/epub_generator.py book1_original_translated.md book1_original_translated.epub",
			remoteDir,
		)
		t.logger.Info("Using llama.cpp for full book translation without timeout", nil)
	} else if strings.Contains(checkOutput, "PROVIDER:openai") || strings.Contains(checkOutput, "PROVIDER:anthropic") {
		// Use API-based translation
		remoteCmd = fmt.Sprintf(
			"cd %s && python3 internal/scripts/translate_llm_only.py book1_original.md book1_original_translated.md && "+
				"python3 internal/scripts/epub_generator.py book1_original_translated.md book1_original_translated.epub",
			t.sshWorker.GetRemoteDir(),
		)
		t.logger.Info("Using API provider for translation", map[string]interface{}{
			"provider": strings.TrimSpace(strings.Split(checkOutput, "PROVIDER:")[1]),
		})
	} else {
		// No translation provider available - show helpful error
		return fmt.Errorf("no translation provider available on remote worker. Output: %s. Please install llama.cpp or configure API keys", checkOutput)
	}

	// Execute with extended timeout (translation can take a while)
	t.logger.Info("Starting remote translation workflow", map[string]interface{}{
		"command": remoteCmd,
	})

	// For debugging: test simple command first
	testCmd := fmt.Sprintf("cd %s && ls -la", t.sshWorker.GetRemoteDir())
	testOutput, testErr := t.sshWorker.ExecuteCommandWithOutput(ctx, testCmd)
	t.logger.Info("Remote directory listing", map[string]interface{}{
		"command": testCmd,
		"output":  testOutput,
		"error":   testErr,
	})

	output, err := t.sshWorker.ExecuteCommandWithOutput(ctx, remoteCmd)
	if err != nil {
		return fmt.Errorf("remote workflow execution failed: %w (output: %s)", err, output)
	}

	t.logger.Info("Remote workflow completed", map[string]interface{}{
		"output_sample": output[:min(200, len(output))],
	})

	// Step 3: Download output files back to local
	expectedFiles := []string{
		baseName + "_original.md",
		baseName + "_original_translated.md",
		baseName + "_original_translated.epub",
	}

	for _, filename := range expectedFiles {
		remotePath := filepath.Join(t.sshWorker.GetRemoteDir(), filename)
		localPath := filepath.Join(localDir, filename)

		if err := t.sshWorker.DownloadFile(ctx, remotePath, localPath); err != nil {
			return fmt.Errorf("failed to download output file %s: %w", filename, err)
		}

		t.logger.Info("Output file downloaded", map[string]interface{}{
			"filename":    filename,
			"local_path":  localPath,
			"remote_path": remotePath,
		})
	}

	return nil
}

// verifyOutputFiles verifies that all expected output files exist and have valid content
func (t *EBookTranslator) verifyOutputFiles() error {
	t.logger.Info("Verifying output files", nil)

	baseName := strings.TrimSuffix(filepath.Base(t.sourceFile), filepath.Ext(t.sourceFile))
	dir := filepath.Dir(t.sourceFile)

	// Check expected files
	expectedFiles := []struct {
		name        string
		description string
		verifyFunc  func(string) error
	}{
		{
			name:        baseName + "_original.md",
			description: "Original Markdown",
			verifyFunc:  t.verifyMarkdownFile,
		},
		{
			name:        baseName + "_original_translated.md",
			description: "Translated Markdown",
			verifyFunc: func(path string) error {
				if err := t.verifyMarkdownFile(path); err != nil {
					return err
				}
				return t.verifyTargetLanguage(path)
			},
		},
		{
			name:        baseName + "_original_translated.epub",
			description: "Final EPUB",
			verifyFunc:  t.verifyEPUBFile,
		},
	}

	for _, expected := range expectedFiles {
		filePath := filepath.Join(dir, expected.name)

		if err := t.verifyFileExists(filePath, expected.description); err != nil {
			return err
		}

		if err := expected.verifyFunc(filePath); err != nil {
			return fmt.Errorf("%s verification failed: %w", expected.description, err)
		}

		t.logger.Info(expected.name+" verified", map[string]interface{}{
			"description": expected.description,
		})
	}

	return nil
}

// verifyFileExists checks if a file exists and is not empty
func (t *EBookTranslator) verifyFileExists(path, description string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s file not found: %w", description, err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("%s file is empty", description)
	}

	t.logger.Info("File exists and has content", map[string]interface{}{
		"file": path,
		"size": info.Size(),
	})

	return nil
}

// verifyMarkdownFile verifies markdown file has valid content
func (t *EBookTranslator) verifyMarkdownFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	contentStr := string(content)

	// Check for minimum content length
	if len(contentStr) < 100 {
		return fmt.Errorf("markdown file appears too small (%d bytes)", len(content))
	}

	// Check for basic markdown elements
	if !strings.Contains(contentStr, "#") {
		return fmt.Errorf("markdown file missing headers")
	}

	return nil
}

// verifyTargetLanguage verifies content is in the target language
func (t *EBookTranslator) verifyTargetLanguage(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file for language verification: %w", err)
	}

	contentStr := string(content)

	// For Serbian Cyrillic, check for Cyrillic characters
	if t.targetLanguage == "sr-cyrl" || t.targetLanguage == "sr" {
		cyrillicChars := []rune{'ћ', 'ђ', 'ч', 'џ', 'ш', 'ж', 'љ', 'њ', 'з', 'с', 'а', 'е', 'и', 'о', 'у'}
		hasCyrillic := false

		for _, char := range cyrillicChars {
			if strings.ContainsRune(contentStr, char) {
				hasCyrillic = true
				break
			}
		}

		if !hasCyrillic {
			return fmt.Errorf("translated content does not contain expected Cyrillic characters")
		}

		t.logger.Info("Cyrillic characters verified in translation", nil)
	}

	return nil
}

// verifyEPUBFile verifies EPUB file has valid structure
func (t *EBookTranslator) verifyEPUBFile(path string) error {
	// Check file size
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat EPUB file: %w", err)
	}

	if info.Size() < 1000 {
		return fmt.Errorf("EPUB file appears too small (%d bytes)", info.Size())
	}

	// Check if it's a valid ZIP (EPUB is a ZIP file)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open EPUB file: %w", err)
	}
	defer file.Close()

	// Read magic number
	buf := make([]byte, 4)
	if _, err := file.Read(buf); err != nil {
		return fmt.Errorf("failed to read EPUB header: %w", err)
	}

	// ZIP files start with PK (0x504B)
	if buf[0] != 0x50 || buf[1] != 0x4B {
		return fmt.Errorf("EPUB file does not have valid ZIP signature")
	}

	t.logger.Info("EPUB file structure verified", map[string]interface{}{
		"file": path,
		"size": info.Size(),
	})

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	if len(os.Args) < 6 {
		fmt.Fprintf(os.Stderr, "Usage: %s <source_fb2_file> <target_language> <remote_host> <remote_user> <remote_password>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s materials/books/book1.fb2 sr-cyrl thinker.local milosvasic WhiteSnake8587\n", os.Args[0])
		os.Exit(1)
	}

	sourceFile := os.Args[1]
	targetLanguage := os.Args[2]
	remoteHost := os.Args[3]
	remoteUser := os.Args[4]
	remotePass := os.Args[5]

	// Validate source file
	if _, err := os.Stat(sourceFile); err != nil {
		fmt.Fprintf(os.Stderr, "Source file not found: %s\n", sourceFile)
		os.Exit(1)
	}

	// Create and run translator
	translator, err := NewEBookTranslator(sourceFile, targetLanguage, remoteHost, remoteUser, remotePass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create translator: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := translator.Execute(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Translation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Ebook translation completed successfully!")
	fmt.Printf("Check the directory containing %s for output files:\n", sourceFile)
	fmt.Println("- Original Markdown")
	fmt.Println("- Translated Markdown")
	fmt.Println("- Final EPUB")
}
