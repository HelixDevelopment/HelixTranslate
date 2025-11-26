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
		CommandTimeout: 120 * time.Second,
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

	// Test llama-cli binary first
	fmt.Println("Testing llama-cli directly...")
	testCmd := "/home/milosvasic/llama.cpp/build/bin/llama-cli --help"
	result, err := worker.ExecuteCommand(ctx, testCmd)
	if err != nil {
		fmt.Printf("Failed to test llama-cli: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("llama-cli test - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("First 300 chars of STDOUT: %s\n", result.Stdout[:300])

	// Upload updated script with correct binary path
	fmt.Println("Uploading updated translation script...")
	scriptPath := "/Users/milosvasic/Projects/Translate/internal/scripts/translate_llm_only.py"
	if err := worker.UploadFile(ctx, scriptPath, "/tmp/translate-ssh/translate_llm_only.py"); err != nil {
		fmt.Printf("Failed to upload script: %v\n", err)
		os.Exit(1)
	}

	// Test translation with simple phrase
	fmt.Println("Testing translation...")
	testInput := `Переведите меня на сербский`
	if err := worker.UploadData(ctx, []byte(testInput), "/tmp/translate-ssh/simple_test.txt"); err != nil {
		fmt.Printf("Failed to upload test: %v\n", err)
		os.Exit(1)
	}

	translateCmd := "cd /tmp/translate-ssh && python3 translate_llm_only.py simple_test.txt simple_test_output.txt"
	result, err = worker.ExecuteCommand(ctx, translateCmd)
	if err != nil {
		fmt.Printf("Failed to translate: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Translation - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Translation STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("Translation STDERR:\n%s\n", result.Stderr)
	}

	// Download and show output
	fmt.Println("Downloading output...")
	if err := worker.DownloadFile(ctx, "/tmp/translate-ssh/simple_test_output.txt", "/Users/milosvasic/Projects/Translate/internal/working/simple_test_output.txt"); err != nil {
		fmt.Printf("Failed to download output: %v\n", err)
		os.Exit(1)
	}

	outputContent, err := os.ReadFile("/Users/milosvasic/Projects/Translate/internal/working/simple_test_output.txt")
	if err != nil {
		fmt.Printf("Failed to read output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Original: %s\n", testInput)
	fmt.Printf("Translated: %s\n", string(outputContent))
	
	if string(outputContent) == testInput {
		fmt.Println("ERROR: Translation returned original text!")
	} else {
		fmt.Println("SUCCESS: Translation worked!")
	}
}