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

	// Check what's in the main directory
	fmt.Println("Checking contents of main directory...")
	lsCmd := "ls -la /home/milosvasic/llama.cpp/build/tools/main/"
	result, err := worker.ExecuteCommand(ctx, lsCmd)
	if err != nil {
		fmt.Printf("Failed to list directory: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Directory listing - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Check what's in the parent tools directory for binaries
	fmt.Println("Checking for binaries in tools directory...")
	findCmd := "find /home/milosvasic/llama.cpp/build/tools -name 'main' -type f -executable"
	result, err = worker.ExecuteCommand(ctx, findCmd)
	if err != nil {
		fmt.Printf("Failed to find binaries: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Binary find - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Also check for any executable files
	fmt.Println("Checking for any executable files...")
	findExecCmd := "find /home/milosvasic/llama.cpp/build -name 'main' -type f -executable"
	result, err = worker.ExecuteCommand(ctx, findExecCmd)
	if err != nil {
		fmt.Printf("Failed to find exec files: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Executable find - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}
}