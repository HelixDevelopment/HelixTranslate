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
		CommandTimeout:    60 * time.Second,
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

	// Test with small sample only
	fmt.Println("Testing enhanced conversion with small sample...")
	cmd := `cd /tmp/translate-ssh && \
echo "Я русский человек. Я люблю читать книги. Спасибо." > test_sample.txt && \
python3 translate_llm_only.py test_sample.txt test_output.txt`

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to test conversion: %v\n", err)
		fmt.Printf("Stderr: %s\n", result.Stderr)
		return
	}

	fmt.Printf("Conversion test output:\n%s\n", result.Stdout)
	
	if result.Stderr != "" {
		fmt.Printf("Stderr:\n%s\n", result.Stderr)
	}

	// Get the result
	cmd = "cat /tmp/translate-ssh/test_output.txt"
	result, err = worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to get result: %v\n", err)
		return
	}

	fmt.Printf("\nTranslated result:\n%s\n", result.Stdout)
}