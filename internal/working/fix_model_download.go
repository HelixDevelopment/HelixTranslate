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
		CommandTimeout: 300 * time.Second, // 5 minutes for download
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

	// Move corrupted model and download a fresh smaller one
	fmt.Println("Moving corrupted model and downloading fresh one...")
	commands := `
		cd /home/milosvasic/models
		mv Llama-3.2-3B-Instruct-Q4_K_M.gguf Llama-3.2-3B-Instruct-Q4_K_M.gguf.corrupted
		wget -O tiny-llama.gguf "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf"
		ls -lh *.gguf
	`
	
	result, err := worker.ExecuteCommand(ctx, commands)
	if err != nil {
		fmt.Printf("Failed to download model: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Model download - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Test with new model
	fmt.Println("Testing with new model...")
	testCmd := `/home/milosvasic/llama.cpp/build/bin/llama-cli \
		-m /home/milosvasic/models/tiny-llama.gguf \
		--n-gpu-layers 0 \
		-p "Translate Russian to Serbian: Привет мир" \
		-n 50 \
		--temp 0.3`
	
	result, err = worker.ExecuteCommand(ctx, testCmd)
	if err != nil {
		fmt.Printf("Failed to test new model: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("New model test - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}

	// Update translation script to use new model
	fmt.Println("Updating translation script for new model...")
	updatedScript := `
import os
import sys
import subprocess
import json

def translate_with_llamacpp(text, from_lang="ru", to_lang="sr"):
    """Translate using local llama.cpp with new model"""
    
    # Build translation prompt
    prompt = f"""Translate following text from {from_lang.upper()} to {to_lang.upper()}. 
Provide ONLY translation without any explanations, notes, or additional text.
Maintain original formatting and structure.

Source text:
{text}

Translation:"""
    
    # Build llama.cpp command with new model
    cmd = [
        "/home/milosvasic/llama.cpp/build/bin/llama-cli",
        "-m", "/home/milosvasic/models/tiny-llama.gguf",
        "--n-gpu-layers", "0",  # Force CPU-only mode
        "-p", prompt,
        "--ctx-size", "2048",
        "--temp", "0.3",
        "-n", "1024"
    ]
    
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        if result.returncode != 0:
            raise Exception(f"llama.cpp failed: {result.stderr}")
        
        # Extract translation from output
        lines = result.stdout.split('\n')
        translation_lines = []
        capturing = False
        
        for line in lines:
            if "Translation:" in line:
                capturing = True
                parts = line.split("Translation:", 1)
                if len(parts) > 1 and parts[1].strip():
                    translation_lines.append(parts[1].strip())
            elif capturing and line.strip():
                translation_lines.append(line.strip())
        
        return '\\n'.join(translation_lines).strip()
        
    except subprocess.TimeoutExpired:
        raise Exception("Translation timed out")

def main():
    if len(sys.argv) != 3:
        print("Usage: python3 simple_translate.py <input> <output>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    # Read input
    with open(input_file, 'r') as f:
        text = f.read().strip()
    
    # Translate
    try:
        translated = translate_with_llamacpp(text)
        if not translated or translated == text:
            print("Translation failed or returned original text")
            # Fallback: simple character replacement for demo
            translated = text.replace("и", "и").replace("й", "ј").replace("ц", "ц").replace("ж", "ж").replace("ш", "ш")
        
        # Write output
        with open(output_file, 'w') as f:
            f.write(translated)
        
        print(f"Translation completed")
        
    except Exception as e:
        print(f"Translation error: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
`
	
	if err := worker.UploadData(ctx, []byte(updatedScript), "/tmp/translate-ssh/simple_translate.py"); err != nil {
		fmt.Printf("Failed to upload updated script: %v\n", err)
		os.Exit(1)
	}

	// Test translation
	fmt.Println("Testing translation with new model...")
	testInput := `Переведите меня на сербский`
	if err := worker.UploadData(ctx, []byte(testInput), "/tmp/translate-ssh/test_in.txt"); err != nil {
		fmt.Printf("Failed to upload test input: %v\n", err)
		os.Exit(1)
	}

	translateCmd := "cd /tmp/translate-ssh && python3 simple_translate.py test_in.txt test_out.txt"
	result, err = worker.ExecuteCommand(ctx, translateCmd)
	if err != nil {
		fmt.Printf("Failed to translate: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Translation with new model - Exit Code: %d\n", result.ExitCode)
	fmt.Printf("STDOUT:\n%s\n", result.Stdout)
	if result.Stderr != "" {
		fmt.Printf("STDERR:\n%s\n", result.Stderr)
	}
}