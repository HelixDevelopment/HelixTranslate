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

	// Check file permissions and try to run llama.cpp directly
	fmt.Println("Checking llama.cpp binary permissions...")
	lsCmd := "ls -la /home/milosvasic/llama.cpp/build/tools/main"
	result, err := worker.ExecuteCommand(ctx, lsCmd)
	if err != nil {
		fmt.Printf("Failed to check permissions: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Permissions check - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Try running llama.cpp with a simple command
	fmt.Println("Testing direct llama.cpp execution...")
	testCmd := "/home/milosvasic/llama.cpp/build/tools/main --help"
	result, err = worker.ExecuteCommand(ctx, testCmd)
	if err != nil {
		fmt.Printf("Failed to run llama.cpp: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Direct execution - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Check current user
	fmt.Println("Checking current user...")
	userCmd := "whoami && id"
	result, err = worker.ExecuteCommand(ctx, userCmd)
	if err != nil {
		fmt.Printf("Failed to check user: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("User check - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}
}