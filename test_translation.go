package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/sshworker"
)

func main() {
	if len(os.Args) != 6 {
		fmt.Println("Usage: test-translation <input-file> <output-file> <host> <user> <password>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]
	host := os.Args[3]
	user := os.Args[4]
	password := os.Args[5]

	// Initialize SSH worker
	config := sshworker.SSHWorkerConfig{
		Host:              host,
		Username:          user,
		Password:          password,
		Port:              22,
		RemoteDir:         "/tmp/translate-ssh",
		ConnectionTimeout: 30 * time.Second,
		CommandTimeout:    60 * time.Second,
	}
	
	log := logger.NewNoOpLogger()
	worker, err := sshworker.NewSSHWorker(config, log)
	if err != nil {
		fmt.Printf("Failed to create SSH worker: %v\n", err)
		os.Exit(1)
	}
	defer worker.Close()

	// Connect to remote
	fmt.Println("Connecting to remote...")
	ctx := context.Background()
	if err := worker.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}

	// Upload input file
	fmt.Printf("Uploading %s...\n", inputFile)
	if err := worker.UploadFile(ctx, inputFile, "/tmp/translate-ssh/book1_original.md"); err != nil {
		fmt.Printf("Failed to upload input file: %v\n", err)
		os.Exit(1)
	}

	// Upload translation script
	fmt.Println("Uploading translation script...")
	scriptPath := filepath.Join("internal", "scripts", "translate_llm_only.py")
	if err := worker.UploadFile(ctx, scriptPath, "/tmp/translate-ssh/translate_llm_only.py"); err != nil {
		fmt.Printf("Failed to upload translation script: %v\n", err)
		os.Exit(1)
	}

	// Run translation test with just first 10 paragraphs
	fmt.Println("Running translation test...")
	cmd := `cd /tmp/translate-ssh && \
head -100 book1_original.md > test_input.md && \
python3 translate_llm_only.py test_input.md test_output.md`

	result, err := worker.ExecuteCommand(ctx, cmd)
	if err != nil {
		fmt.Printf("Failed to run translation test: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Translation result:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("Stderr:\n%s\n", result.Stderr)
	}

	// Download result
	fmt.Println("Downloading translation result...")
	if err := worker.DownloadFile(ctx, "/tmp/translate-ssh/test_output.md", outputFile); err != nil {
		fmt.Printf("Failed to download result: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Translation test completed. Check %s\n", outputFile)
}