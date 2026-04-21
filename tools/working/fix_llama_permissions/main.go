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
		CommandTimeout: 60 * time.Second,
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

	// Fix llama.cpp binary permissions
	fmt.Println("Fixing llama.cpp binary permissions...")
	chmodCmd := "chmod +x /home/milosvasic/llama.cpp/build/tools/main"
	result, err := worker.ExecuteCommand(ctx, chmodCmd)
	if err != nil {
		fmt.Printf("Failed to chmod llama.cpp: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Chmod result: Exit Code: %d, STDERR: %s\n", result.ExitCode, result.Stderr)

	// Also make our uploaded llama.cpp binary executable
	fmt.Println("Making uploaded llama.cpp binary executable...")
	chmodUploadCmd := "chmod +x /tmp/translate-ssh/llama.cpp"
	result, err = worker.ExecuteCommand(ctx, chmodUploadCmd)
	if err != nil {
		fmt.Printf("Failed to chmod uploaded llama.cpp: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Chmod uploaded result: Exit Code: %d, STDERR: %s\n", result.ExitCode, result.Stderr)

	// Now test the translation again
	testScriptPath := "/Users/milosvasic/Projects/Translate/internal/working/test_translation_debug.py"
	if err := worker.UploadFile(ctx, testScriptPath, "/tmp/translate-ssh/test_translation_debug.py"); err != nil {
		fmt.Printf("Failed to upload test script: %v\n", err)
		os.Exit(1)
	}

	// Execute the test command
	fullCommand := fmt.Sprintf("cd /tmp/translate-ssh && python3 test_translation_debug.py")
	
	result, err = worker.ExecuteCommand(ctx, fullCommand)
	if err != nil {
		fmt.Printf("Failed to execute command: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}
}