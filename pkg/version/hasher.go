package version

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CodebaseHasher manages codebase versioning through file hashing
type CodebaseHasher struct {
	// RelevantDirectories specifies directories to include in hash calculation
	RelevantDirectories []string
	// RelevantExtensions specifies file extensions to include
	RelevantExtensions []string
	// ExcludePatterns specifies patterns to exclude
	ExcludePatterns []string
}

// NewCodebaseHasher creates a new codebase hasher with sensible defaults
func NewCodebaseHasher() *CodebaseHasher {
	return &CodebaseHasher{
		RelevantDirectories: []string{
			"cmd",
			"pkg",
			"internal",
			"scripts",
			"docs",
		},
		RelevantExtensions: []string{
			".go",
			".json",
			".yaml",
			".yml",
			".md",
			".sh",
			".txt",
			"Dockerfile",
			"Makefile",
		},
		ExcludePatterns: []string{
			".git",
			"node_modules",
			"__pycache__",
			".DS_Store",
			"*.log",
			"*.tmp",
			"*.pid",
			"coverage*.out",
			"*.test",
			"vendor",
			".env",
			"._*", // Exclude AppleDouble/metadata files
		},
	}
}

// CalculateHash computes the comprehensive hash of the codebase
func (h *CodebaseHasher) CalculateHash() (string, error) {
	hasher := sha256.New()
	
	// Note: Removed timestamp to ensure consistent hashes across systems

	// Process each relevant directory
	for _, dir := range h.RelevantDirectories {
		if err := h.processDirectory(hasher, dir); err != nil {
			return "", fmt.Errorf("failed to process directory %s: %w", dir, err)
		}
	}

	// Process root-level files
	if err := h.processRootFiles(hasher); err != nil {
		return "", fmt.Errorf("failed to process root files: %w", err)
	}

	// Binary hashes temporarily disabled for SSH deployment consistency
	// if err := h.addBinaryHashes(hasher); err != nil {
	// 	return "", fmt.Errorf("failed to add binary hashes: %w", err)
	// }

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// processDirectory recursively processes a directory
func (h *CodebaseHasher) processDirectory(hasher io.Writer, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory should be excluded
			for _, pattern := range h.ExcludePatterns {
				if strings.Contains(path, pattern) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file should be included
		if h.shouldIncludeFile(path) {
			return h.addFileToHash(hasher, path, info)
		}

		return nil
	})
}

// processRootFiles processes files in the root directory
func (h *CodebaseHasher) processRootFiles(hasher io.Writer) error {
	entries, err := os.ReadDir(".")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := entry.Name()
		if h.shouldIncludeFile(path) && !h.isDirectoryInRelevantList(path) {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := h.addFileToHash(hasher, path, info); err != nil {
				return err
			}
		}
	}

	return nil
}

// shouldIncludeFile determines if a file should be included in the hash
func (h *CodebaseHasher) shouldIncludeFile(path string) bool {
	// Check exclude patterns first
	for _, pattern := range h.ExcludePatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}

	// Check if file has relevant extension
	for _, ext := range h.RelevantExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	return false
}

// isDirectoryInRelevantList checks if path is a directory we've already processed
func (h *CodebaseHasher) isDirectoryInRelevantList(path string) bool {
	for _, dir := range h.RelevantDirectories {
		if path == dir {
			return true
		}
	}
	return false
}

// addFileToHash adds a single file to the hash calculation
func (h *CodebaseHasher) addFileToHash(hasher io.Writer, path string, info os.FileInfo) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write file path and size to hash (removed modtime for consistency)
	fmt.Fprintf(hasher, "file:%s\n", path)
	fmt.Fprintf(hasher, "size:%d\n", info.Size())

	// Hash file content
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	// Add separator
	fmt.Fprintf(hasher, "---FILE_SEPARATOR---\n")
	return nil
}

// addBinaryHashes adds hashes of generated binaries to the calculation
func (h *CodebaseHasher) addBinaryHashes(hasher io.Writer) error {
	// Check for common binary names
	binaries := []string{
		"translator",
		"translator-server",
		"cli",
		"server",
		"markdown-translator",
		"preparation-translator",
	}

	for _, binary := range binaries {
		if info, err := os.Stat(binary); err == nil && !info.IsDir() {
			fmt.Fprintf(hasher, "binary:%s\n", binary)
			fmt.Fprintf(hasher, "size:%d\n", info.Size())
			fmt.Fprintf(hasher, "modtime:%d\n", info.ModTime().Unix())

			// Hash the binary
			file, err := os.Open(binary)
			if err != nil {
				continue
			}
			io.Copy(hasher, file)
			file.Close()

			fmt.Fprintf(hasher, "---BINARY_SEPARATOR---\n")
		}
	}

	return nil
}

// CompareVersions compares local and remote codebase versions
func (h *CodebaseHasher) CompareVersions(localHash, remoteHash string) bool {
	return localHash == remoteHash
}

// GenerateVersionInfo generates detailed version information
func (h *CodebaseHasher) GenerateVersionInfo() (*VersionInfo, error) {
	hash, err := h.CalculateHash()
	if err != nil {
		return nil, err
	}

	return &VersionInfo{
		Hash:      hash,
		Timestamp: time.Now(),
		Directories: h.RelevantDirectories,
		Extensions: h.RelevantExtensions,
	}, nil
}

// VersionInfo contains detailed version information
type VersionInfo struct {
	Hash       string    `json:"hash"`
	Timestamp  time.Time `json:"timestamp"`
	Directories []string  `json:"directories"`
	Extensions []string  `json:"extensions"`
}

// SaveVersionInfo saves version info to a file
func (vi *VersionInfo) SaveVersionInfo(filename string) error {
	// This would save to JSON - implement if needed
	return nil
}