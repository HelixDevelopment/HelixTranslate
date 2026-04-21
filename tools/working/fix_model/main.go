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

	// Check all models in directory
	fmt.Println("Checking all GGUF models...")
	lsCmd := "ls -la /home/milosvasic/models/*.gguf"
	result, err := worker.ExecuteCommand(ctx, lsCmd)
	if err != nil {
		fmt.Printf("Failed to list models: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Models listing - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Check model file integrity
	fmt.Println("Checking model file integrity...")
	checkCmd := "file /home/milosvasic/models/Llama-3.2-3B-Instruct-Q4_K_M.gguf && ls -lh /home/milosvasic/models/Llama-3.2-3B-Instruct-Q4_K_M.gguf"
	result, err = worker.ExecuteCommand(ctx, checkCmd)
	if err != nil {
		fmt.Printf("Failed to check model: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Model check - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Try downloading a fresh model
	fmt.Println("Downloading a fresh, working model...")
	downloadCmd := "cd /home/milosvasic/models && wget -O Llama-3.2-3B-Instruct-Q4_K_M.gguf 'https://huggingface.co/bartowski/Llama-3.2-3B-Instruct-GGUF/resolve/main/Llama-3.2-3B-Instruct-Q4_K_M.gguf'"
	result, err = worker.ExecuteCommand(ctx, downloadCmd)
	if err != nil {
		fmt.Printf("Failed to download model: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Download - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}
}