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

	// Check build directory structure
	fmt.Println("Checking llama.cpp build directory structure...")
	treeCmd := "find /home/milosvasic/llama.cpp/build -type f -name '*main*' -o -name '*llama*' | head -20"
	result, err := worker.ExecuteCommand(ctx, treeCmd)
	if err != nil {
		fmt.Printf("Failed to find files: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Build files - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Check for any executable file in build directory
	fmt.Println("Checking for any executables in build directory...")
	findExecCmd := "find /home/milosvasic/llama.cpp/build -type f -executable | head -10"
	result, err = worker.ExecuteCommand(ctx, findExecCmd)
	if err != nil {
		fmt.Printf("Failed to find executables: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Executables in build - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Check if there's a bin directory
	fmt.Println("Checking for bin directories...")
	binCmd := "find /home/milosvasic/llama.cpp -type d -name 'bin'"
	result, err = worker.ExecuteCommand(ctx, binCmd)
	if err != nil {
		fmt.Printf("Failed to find bin dirs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Bin dirs - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}
}