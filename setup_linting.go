package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("Installing golangci-lint...")
	
	// Install golangci-lint
	cmd := exec.Command("go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to install golangci-lint: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("golangci-lint installed successfully!")
	
	// Initialize config if not exists
	if _, err := os.Stat(".golangci.yml"); os.IsNotExist(err) {
		fmt.Println("Initializing golangci-lint config...")
		initCmd := exec.Command("golangci-lint", "run", "--init")
		initCmd.Stdout = os.Stdout
		initCmd.Stderr = os.Stderr
		if err := initCmd.Run(); err != nil {
			fmt.Printf("Failed to initialize config: %v\n", err)
		}
	}
	
	// Run linting
	fmt.Println("Running linter...")
	lintCmd := exec.Command("golangci-lint", "run", "./...")
	lintCmd.Stdout = os.Stdout
	lintCmd.Stderr = os.Stderr
	if err := lintCmd.Run(); err != nil {
		fmt.Printf("Linting found issues: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Linting completed successfully!")
}