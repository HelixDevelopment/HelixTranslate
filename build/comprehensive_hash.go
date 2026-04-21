package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: comprehensive_hash <project_root> <output_file>")
		os.Exit(1)
	}

	projectRoot := os.Args[1]
	outputFile := os.Args[2]

	// Get comprehensive hash of all relevant code
	hash, err := calculateComprehensiveHash(projectRoot)
	if err != nil {
		fmt.Printf("Error calculating hash: %v\n", err)
		os.Exit(1)
	}

	// Write hash to output file
	if err := os.WriteFile(outputFile, []byte(hash), 0644); err != nil {
		fmt.Printf("Error writing hash file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Comprehensive codebase hash: %s\n", hash)
	fmt.Printf("Written to: %s\n", outputFile)
}

// calculateComprehensiveHash generates a comprehensive hash of all source code
func calculateComprehensiveHash(projectRoot string) (string, error) {
	hash := sha256.New()

	var files []string
	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".go" || ext == ".mod" || ext == ".sum" || ext == ".json" ||
			ext == ".yaml" || ext == ".yml" || ext == ".proto" || ext == ".sh" {
			rel, err := filepath.Rel(projectRoot, path)
			if err == nil {
				files = append(files, rel)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk project root: %w", err)
	}

	sort.Strings(files)

	for _, f := range files {
		if _, err := hash.Write([]byte(f)); err != nil {
			return "", err
		}
		if _, err := hash.Write([]byte{0}); err != nil {
			return "", err
		}

		path := filepath.Join(projectRoot, f)
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			continue
		}
		file.Close()

		if _, err := hash.Write([]byte{0}); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
