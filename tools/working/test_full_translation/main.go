package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
)

func main() {
	// Initialize SSH worker
	config := sshworker.SSHWorkerConfig{
		Host:       "thinker.local",
		Port:       22,
		Username:   "milosvasic",
		Password:   "WhiteSnake8587",
		RemoteDir:  "/tmp/translate-ssh",
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout: 120 * time.Second, // Longer timeout for translation
	}

	loggerConfig := logger.LoggerConfig{
		Level: logger.INFO,
		Format: logger.FORMAT_TEXT,
	}
	log := logger.NewLogger(loggerConfig)
	
	worker, err := sshworker.NewSSHWorker(config, log)
	if err != nil {
		fmt.Printf("Failed to create SSH worker: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	
	// Connect to worker
	if err := worker.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer worker.Disconnect()

	// Fix permissions first
	fmt.Println("Fixing llama.cpp permissions...")
	chmodCmd := "chmod +x /home/milosvasic/llama.cpp/build/tools/main"
	result, err := worker.ExecuteCommand(ctx, chmodCmd)
	if err != nil {
		fmt.Printf("Failed to chmod: %v\n", err)
		os.Exit(1)
	}

	// Upload updated script
	fmt.Println("Uploading updated translation script...")
	scriptPath := "/Users/milosvasic/Projects/Translate/internal/scripts/translate_llm_only.py"
	if err := worker.UploadFile(ctx, scriptPath, "/tmp/translate-ssh/translate_llm_only.py"); err != nil {
		fmt.Printf("Failed to upload script: %v\n", err)
		os.Exit(1)
	}

	// Create a simple test input file
	testInput := `# Test Chapter

This is a test paragraph in Russian. Переведите меня на сербский пожалуйста.

## Section 2

Here is another paragraph with Russian text: Я хочу перевести этот текст на сербский язык.`
	
	fmt.Println("Uploading test input file...")
	if err := worker.UploadData(ctx, []byte(testInput), "/tmp/translate-ssh/test_input.md"); err != nil {
		fmt.Printf("Failed to upload test input: %v\n", err)
		os.Exit(1)
	}

	// Run translation
	fmt.Println("Running translation...")
	translateCmd := "cd /tmp/translate-ssh && python3 translate_llm_only.py test_input.md test_output.md"
	result, err = worker.ExecuteCommand(ctx, translateCmd)
	if err != nil {
		fmt.Printf("Failed to run translation: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Translation Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Translation STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("Translation STDERR:\n%s\n", result.Stderr)
	}

	// Download and show output
	fmt.Println("Downloading translated file...")
	if err := worker.DownloadFile(ctx, "/tmp/translate-ssh/test_output.md", "/Users/milosvasic/Projects/Translate/internal/working/test_output.md"); err != nil {
		fmt.Printf("Failed to download output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Translated file content:")
	outputContent, err := os.ReadFile("/Users/milosvasic/Projects/Translate/internal/working/test_output.md")
	if err != nil {
		fmt.Printf("Failed to read output file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s\n", outputContent)
}