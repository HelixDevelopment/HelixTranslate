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

	// Test translation with improved prompt format
	fmt.Println("Testing improved translation...")
	cmd := `/home/milosvasic/llama.cpp/build/bin/llama-cli \
-m /home/milosvasic/models/tiny-llama-working.gguf \
--n-gpu-layers 0 \
-p "<|im_start|>system
You are a professional translator. Translate the given Russian text to Serbian. Provide ONLY the translation.
<|im_end|>
<|im_start|>user
Translate this to Serbian: Привет мир
<|im_end|>
<|im_start|>assistant
" \
--ctx-size 2048 \
--temp 0.3 \
-n 50`

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to test translation: %v\n", err)
		fmt.Printf("Stderr: %s\n", result.Stderr)
		return
	}

	fmt.Printf("Translation test output:\n%s\n", result.Stdout)
	
	if result.Stderr != "" {
		fmt.Printf("Stderr:\n%s\n", result.Stderr)
	}
}