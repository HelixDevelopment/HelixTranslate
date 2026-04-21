package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CodebaseHashGenerator generates comprehensive hash of codebase
type CodebaseHashGenerator struct {
	RootPath      string
	Hashes        map[string]string
	ExcludedDirs  []string
	ExcludedFiles []string
	IncludeBins   bool
}

// NewCodebaseHashGenerator creates a new hash generator
func NewCodebaseHashGenerator(rootPath string) *CodebaseHashGenerator {
	return &CodebaseHashGenerator{
		RootPath: rootPath,
		Hashes:   make(map[string]string),
		ExcludedDirs: []string{
			".git", ".idea", "node_modules", "vendor",
			"build", "dist", "bin", ".vscode",
			"logs", "tmp", ".cache", "*.log",
		},
		ExcludedFiles: []string{
			"*.log", "*.tmp", "*.cache", "*.pid",
			"*.lock", "*.swp", "*.swo", ".DS_Store",
		},
		IncludeBins: true,
	}
}

// GenerateCodebaseHash generates comprehensive hash of entire codebase
func (cgh *CodebaseHashGenerator) GenerateCodebaseHash() (string, error) {
	log.Printf("🔍 Starting codebase hash generation for: %s", cgh.RootPath)

	// 1. Hash Go source files
	if err := cgh.hashGoFiles(); err != nil {
		return "", fmt.Errorf("failed to hash Go files: %w", err)
	}

	// 2. Hash configuration files
	if err := cgh.hashConfigFiles(); err != nil {
		return "", fmt.Errorf("failed to hash config files: %w", err)
	}

	// 3. Hash binaries (if included)
	if cgh.IncludeBins {
		if err := cgh.hashBinaries(); err != nil {
			return "", fmt.Errorf("failed to hash binaries: %w", err)
		}
	}

	// 4. Hash templates and static files
	if err := cgh.hashStaticFiles(); err != nil {
		return "", fmt.Errorf("failed to hash static files: %w", err)
	}

	// 5. Generate final combined hash
	finalHash := cgh.generateFinalHash()

	log.Printf("✅ Codebase hash generated: %s (based on %d files)", finalHash, len(cgh.Hashes))
	return finalHash, nil
}

// hashGoFiles hashes all Go source files
func (cgh *CodebaseHashGenerator) hashGoFiles() error {
	log.Printf("📁 Hashing Go source files...")

	return filepath.WalkDir(cgh.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded directories
		if d.IsDir() {
			for _, excluded := range cgh.ExcludedDirs {
				if strings.Contains(path, excluded) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Only hash .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files for now (they change frequently)
		if strings.Contains(path, "_test.go") {
			return nil
		}

		return cgh.hashFile(path, "go")
	})
}

// hashConfigFiles hashes configuration and important files
func (cgh *CodebaseHashGenerator) hashConfigFiles() error {
	log.Printf("⚙️ Hashing configuration files...")

	configFiles := []string{
		"go.mod", "go.sum",
		"config.json",
		"Makefile",
		"docker-compose.yml", "Dockerfile",
	}

	for _, configFile := range configFiles {
		fullPath := filepath.Join(cgh.RootPath, configFile)
		if _, err := os.Stat(fullPath); err == nil {
			if err := cgh.hashFile(fullPath, "config"); err != nil {
				return fmt.Errorf("failed to hash config file %s: %w", configFile, err)
			}
		}
	}

	// Hash all configs in internal/working/
	configDir := filepath.Join(cgh.RootPath, "internal", "working")
	if _, err := os.Stat(configDir); err == nil {
		return filepath.WalkDir(configDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if !strings.HasSuffix(path, ".json") {
				return nil
			}

			return cgh.hashFile(path, "config")
		})
	}

	return nil
}

// hashBinaries hashes generated binaries
func (cgh *CodebaseHashGenerator) hashBinaries() error {
	log.Printf("🔧 Hashing binaries...")

	// Look for common binary locations
	binDirs := []string{
		"bin", "build", "dist",
	}

	for _, binDir := range binDirs {
		fullPath := filepath.Join(cgh.RootPath, binDir)
		if _, err := os.Stat(fullPath); err == nil {
			return filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}

				// Hash executable files
				if info, err := d.Info(); err == nil {
					// Skip if not executable (Unix)
					if info.Mode().Perm()&0111 == 0 {
						return nil
					}

					return cgh.hashFile(path, "binary")
				}

				return nil
			})
		}
	}

	// Hash main binaries directly
	mainBinaries := []string{
		"translate", "translator", "monitor-server",
	}

	for _, binary := range mainBinaries {
		fullPath := filepath.Join(cgh.RootPath, binary)
		if _, err := os.Stat(fullPath); err == nil {
			cgh.hashFile(fullPath, "binary")
		}
	}

	return nil
}

// hashStaticFiles hashes templates, docs, and other important files
func (cgh *CodebaseHashGenerator) hashStaticFiles() error {
	log.Printf("📄 Hashing static files...")

	staticDirs := []string{
		"docs", "templates", "web", "static", "assets",
		"scripts", "pkg", "cmd",
	}

	for _, dir := range staticDirs {
		fullPath := filepath.Join(cgh.RootPath, dir)
		if _, err := os.Stat(fullPath); err == nil {
			return filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip excluded subdirectories
				if d.IsDir() {
					for _, excluded := range cgh.ExcludedDirs {
						if strings.Contains(path, excluded) {
							return filepath.SkipDir
						}
					}
					return nil
				}

				// Hash relevant static files
				if cgh.shouldHashStaticFile(path) {
					return cgh.hashFile(path, "static")
				}

				return nil
			})
		}
	}

	return nil
}

// shouldHashStaticFile determines if a static file should be hashed
func (cgh *CodebaseHashGenerator) shouldHashStaticFile(path string) bool {
	extensions := []string{
		".html", ".css", ".js", ".json", ".md",
		".yml", ".yaml", ".sh", ".py",
		".proto", ".sql", ".txt",
	}

	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	// Also hash files without extension (like Dockerfile, Makefile)
	if !strings.Contains(filepath.Base(path), ".") {
		return true
	}

	return false
}

// hashFile hashes a single file and stores it in the hashes map
func (cgh *CodebaseHashGenerator) hashFile(filePath, fileType string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Store with relative path and type
	relPath, _ := filepath.Rel(cgh.RootPath, filePath)
	key := fmt.Sprintf("%s:%s", fileType, relPath)
	cgh.Hashes[key] = hashStr

	// Log for debugging
	if strings.Contains(os.Getenv("DEBUG_HASH"), "true") {
		log.Printf("📝 Hashed %s -> %s", key, hashStr[:8]+"...")
	}

	return nil
}

// generateFinalHash creates final combined hash from all individual hashes
func (cgh *CodebaseHashGenerator) generateFinalHash() string {
	// Sort keys for consistent hashing
	keys := make([]string, 0, len(cgh.Hashes))
	for key := range cgh.Hashes {
		keys = append(keys, key)
	}

	// Sort keys for consistency
	sort.Strings(keys)

	// Combine all hashes
	var combined strings.Builder
	for _, key := range keys {
		combined.WriteString(key)
		combined.WriteString(":")
		combined.WriteString(cgh.Hashes[key])
		combined.WriteString(";")
	}

	// Generate final hash
	finalHash := sha256.Sum256([]byte(combined.String()))
	return hex.EncodeToString(finalHash[:])
}

// SaveHashToFile saves the hash to a file
func (cgh *CodebaseHashGenerator) SaveHashToFile(filePath string) error {
	data := map[string]interface{}{
		"hash":         cgh.generateFinalHash(),
		"file_count":   len(cgh.Hashes),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"version":      "1.0",
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, jsonData, 0644)
}

// LoadHashFromFile loads hash from a file
func (cgh *CodebaseHashGenerator) LoadHashFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return "", err
	}

	hash, ok := data["hash"].(string)
	if !ok {
		return "", fmt.Errorf("invalid hash file format")
	}

	return hash, nil
}

// CompareHashes compares local hash with remote hash
func (cgh *CodebaseHashGenerator) CompareHashes(remoteHash string) (bool, error) {
	localHash, err := cgh.GenerateCodebaseHash()
	if err != nil {
		return false, err
	}

	// Log comparison for debugging
	if strings.Contains(os.Getenv("DEBUG_HASH"), "true") {
		log.Printf("🔍 Local hash:  %s", localHash)
		log.Printf("🔍 Remote hash: %s", remoteHash)
	}

	return localHash == remoteHash, nil
}

// PrintHashSummary prints detailed hash summary
func (cgh *CodebaseHashGenerator) PrintHashSummary() {
	fmt.Printf("\n📊 Codebase Hash Summary\n")               //nolint:forbidigo
	fmt.Printf("========================\n")                //nolint:forbidigo
	fmt.Printf("Total files hashed: %d\n", len(cgh.Hashes)) //nolint:forbidigo

	// Group by type
	types := make(map[string]int)
	for key := range cgh.Hashes {
		fileType := strings.Split(key, ":")[0]
		types[fileType]++
	}

	for fileType, count := range types {
		fmt.Printf("%s files: %d\n", cases.Title(language.English).String(fileType), count) //nolint:forbidigo
	}

	fmt.Printf("\nCombined hash: %s\n", cgh.generateFinalHash())        //nolint:forbidigo
	fmt.Printf("Generated at: %s\n\n", time.Now().Format(time.RFC3339)) //nolint:forbidigo
}
func main() {
	generator := NewCodebaseHashGenerator(".")
	if _, err := generator.GenerateCodebaseHash(); err != nil {
		log.Fatalf("Failed to generate hash: %v", err)
	}
	generator.PrintHashSummary()
}
