package main

import (
	"context"
	"fmt"
	"os"

	"digital.vasic.translator/cmd/translate-ssh"
	"digital.vasic.translator/pkg/sshworker"
	"digital.vasic.translator/pkg/logger"
)

func test_llama_connection() {
	// Setup config
	config := translate_ssh.Config{
		InputFile:     "materials/books/book1.fb2",
		OutputFile:    "test_output.epub",
		SSHHost:       "thinker.local",
		SSHUser:       "milosvasic",
		SSHPassword:   "WhiteSnake8587",
		SSHPort:       22,
		RemoteDir:     "/tmp/translate-ssh",
	}
	
	// Setup logger
	logConfig := logger.LoggerConfig{
		Level:  "debug",
		Format: "text",
	}
	logger := logger.NewLogger(logConfig)
	config.Logger = logger
	
	// Initialize SSH worker
	ctx := context.Background()
	worker, err := sshworker.NewSSHWorker(config.SSHHost, config.SSHUser, config.SSHPassword, config.SSHPort, config.Logger)
	if err != nil {
		fmt.Printf("Failed to create SSH worker: %v\n", err)
		return
	}
	defer worker.Close()
	
	// Test basic connectivity
	result, err := worker.ExecuteCommand(ctx, "echo 'SSH connection test successful'")
	if err != nil {
		fmt.Printf("SSH test failed: %v\n", err)
		return
	}
	
	fmt.Printf("SSH test result: %s\n", result.Stdout)
	
	// Test llama.cpp availability
	llamaTestCmd := "cd /tmp/translate-ssh && python3 test_llama.py"
	result, err = worker.ExecuteCommand(ctx, llamaTestCmd)
	if err != nil {
		fmt.Printf("llama.cpp test failed: %v\n", err)
		return
	}
	
	fmt.Printf("llama.cpp test result: %s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("llama.cpp test errors: %s\n", result.Stderr)
	}
}

func main() {
	test_llama_connection()
}