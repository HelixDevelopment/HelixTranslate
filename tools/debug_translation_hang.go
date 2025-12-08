package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	// First test the translate_llm_only.py script locally to identify the issue
	fmt.Println("=== Local Debug Test ===")

	// Check if script exists
	if _, err := os.Stat("../internal/scripts/translate_llm_only.py"); os.IsNotExist(err) {
		log.Fatalf("Script not found: ../internal/scripts/translate_llm_only.py")
	}

	// Create test input
	testInput := "Это тестовый текст для перевода."
	if err := os.WriteFile("/tmp/test_input.md", []byte(testInput), 0644); err != nil {
		log.Fatalf("Failed to create test input: %v", err)
	}

	// Test the script locally with timeout
	fmt.Println("Testing translation script locally with 30 second timeout...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "../internal/scripts/translate_llm_only.py", "/tmp/test_input.md", "/tmp/test_output.md")

	start := time.Now()
	output, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	if ctx.Err() == context.DeadlineExceeded {
		fmt.Printf("❌ Script timed out after %v (this is the hanging issue!)\n", elapsed)
		fmt.Printf("Partial output: %s\n", string(output))
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("❌ Script failed after %v: %v\n", elapsed, err)
		fmt.Printf("Output: %s\n", string(output))
		os.Exit(1)
	}

	fmt.Printf("✅ Script completed in %v\n", elapsed)
	fmt.Printf("Output: %s\n", string(output))

	// Check output file
	if outputContent, err := os.ReadFile("/tmp/test_output.md"); err == nil {
		fmt.Printf("Translation result: %s\n", string(outputContent))
	}

	fmt.Println("\n=== Testing llama.cpp directly ===")

	sshOutput, err := exec.CommandContext(context.Background(), "ssh", "thinker.local", "echo 'SSH connection test'").CombinedOutput()
	if err != nil {
		fmt.Printf("❌ SSH connection failed: %v\n", err)
	} else {
		fmt.Printf("✅ SSH connection works: %s\n", string(sshOutput))
	}

	fmt.Println("=== Debug Summary ===")
	fmt.Println("1. If local script hangs: Issue is in translate_llm_only.py")
	fmt.Println("2. If local script works but remote hangs: Issue is SSH/environment")
	fmt.Println("3. Check llama.cpp model availability and paths")
}
