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

	// Upload the updated translate_llm_only.py script
	fmt.Println("Uploading updated translate_llm_only.py...")
	scriptPath := "/Users/milosvasic/Projects/Translate/internal/scripts/translate_llm_only.py"
	if err := worker.UploadFile(ctx, scriptPath, "/tmp/translate-ssh/translate_llm_only.py"); err != nil {
		fmt.Printf("Failed to upload script: %v\n", err)
		os.Exit(1)
	}

	// Create debug script to test find_llama_binary function
	debugScript := `#!/usr/bin/env python3
import sys
sys.path.insert(0, '/tmp/translate-ssh')
from translate_llm_only import find_llama_binary

print("Testing find_llama_binary function...")
result = find_llama_binary()
print(f"Result: {result}")
`
	
	fmt.Println("Uploading debug script...")
	if err := worker.UploadData(ctx, []byte(debugScript), "/tmp/translate-ssh/debug_find_binary.py"); err != nil {
		fmt.Printf("Failed to upload debug script: %v\n", err)
		os.Exit(1)
	}

	// Run debug script
	fmt.Println("Running debug script...")
	result, err := worker.ExecuteCommand(ctx, "cd /tmp/translate-ssh && python3 debug_find_binary.py")
	if err != nil {
		fmt.Printf("Failed to execute debug script: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Debug result - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}
}