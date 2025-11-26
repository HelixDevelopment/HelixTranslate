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
		CommandTimeout:    300 * time.Second,
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

	// Download a working small model from HuggingFace
	fmt.Println("Downloading working model...")
	cmd := `cd /home/milosvasic/models && \
rm -f tiny-llama-corrupted.gguf && \
wget -O tiny-llama-working.gguf "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q2_K.gguf"`

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to download model: %v\n", err)
		fmt.Printf("Stderr: %s\n", result.Stderr)
		return
	}

	fmt.Printf("Model download output:\n%s\n", result.Stdout)
	
	// Check if model was downloaded successfully
	cmd = `ls -la /home/milosvasic/models/tiny-llama-working.gguf`
	result, err = worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to check model: %v\n", err)
		return
	}
	
	fmt.Printf("Model file info:\n%s\n", result.Stdout)
}