package main

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
)

func main() {
	// Initialize SSH worker
	config := sshworker.SSHWorkerConfig{
		Host:              "thinker.local",
		Username:          "milosvasic",
		Password:          "WhiteSnake8587",
		Port:              22,
		RemoteDir:         "/tmp/translate-ssh",
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    120 * time.Second,
	}
	
	log := logger.NewNoOpLogger()
	worker, err := sshworker.NewSSHWorker(config, log)
	if err != nil {
		fmt.Printf("Failed to create SSH worker: %v\n", err)
		return
	}
	defer worker.Close()

	// Connect to remote
	fmt.Println("Connecting to remote...")
	ctx := context.Background()
	if err := worker.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	// Test the working model with a simple translation
	fmt.Println("Testing working model...")
	cmd := `/home/milosvasic/llama.cpp/build/bin/llama-cli \
-m /home/milosvasic/models/tiny-llama-working.gguf \
--n-gpu-layers 0 \
-p "Translate from Russian to Serbian: Привет мир" \
--ctx-size 2048 \
--temp 0.3 \
-n 100`

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to test model: %v\n", err)
		fmt.Printf("Stderr: %s\n", result.Stderr)
		return
	}

	fmt.Printf("Model test output:\n%s\n", result.Stdout)
	
	if result.Stderr != "" {
		fmt.Printf("Stderr:\n%s\n", result.Stderr)
	}
}