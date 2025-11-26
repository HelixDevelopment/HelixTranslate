package main

import (
	"context"
	"fmt"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
)

func main() {
	config := sshworker.SSHWorkerConfig{
		Host:              "thinker.local",
		Username:          "milosvasic",
		Password:          "WhiteSnake8587",
		Port:              22,
		RemoteDir:         "/tmp/translate-ssh",
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    180 * time.Second,
	}
	
	log := logger.NewNoOpLogger()
	worker, err := sshworker.NewSSHWorker(config, log)
	if err != nil {
		fmt.Printf("Failed to create SSH worker: %v\n", err)
		return
	}
	defer worker.Close()

	fmt.Println("Connecting to remote...")
	ctx := context.Background()
	if err := worker.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}

	// Upload new pure LLM script
	fmt.Println("Uploading pure LLM translation script...")
	cmd := `echo 'Я русский человек.' > test_ru.txt && \
/home/milosvasic/llama.cpp/build/bin/llama-cli \
-m /home/milosvasic/models/tiny-llama-working.gguf \
--n-gpu-layers 0 \
-p "You are a professional translator from Russian to Serbian. Translate this text: Я русский человек. Serbian translation:" \
--ctx-size 2048 \
--temp 0.1 \
-n 100`

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to test pure LLM: %v\n", err)
		fmt.Printf("Stderr: %s\n", result.Stderr)
		return
	}

	fmt.Printf("Pure LLM test output:\n%s\n", result.Stdout)
	
	if result.Stderr != "" {
		fmt.Printf("Stderr:\n%s\n", result.Stderr)
	}
}