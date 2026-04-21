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

	// List all binaries in bin directory
	fmt.Println("Listing all binaries in llama.cpp/bin directory...")
	lsCmd := "ls -la /home/milosvasic/llama.cpp/build/bin/"
	result, err := worker.ExecuteCommand(ctx, lsCmd)
	if err != nil {
		fmt.Printf("Failed to list bin: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Bin listing - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Test one of the binaries
	fmt.Println("Testing llama-cli binary...")
	testCmd := "/home/milosvasic/llama.cpp/build/bin/llama-gemma3-cli --help"
	result, err = worker.ExecuteCommand(ctx, testCmd)
	if err != nil {
		fmt.Printf("Failed to test binary: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Binary test - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT (first 500 chars):\n%s\n", result.Stdout[:500])
	if len(result.Stderr) > 0 {
		fmt.Printf("STDERR (first 500 chars):\n%s\n", result.Stderr[:500])
	}
}