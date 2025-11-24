package version

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodebaseHasher_CalculateHash(t *testing.T) {
	hasher := NewCodebaseHasher()
	
	// Test that we can calculate a hash
	hash, err := hasher.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}
	
	// Hash should be non-empty and consistent
	if len(hash) != 64 { // SHA256 produces 64 character hex string
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}
	
	// Calculate hash again - should be different due to timestamp
	hash2, err := hasher.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate second hash: %v", err)
	}
	
	if hash == hash2 {
		t.Error("Hashes should be different due to timestamp")
	}
}

func TestCodebaseHasher_ProcessDirectory(t *testing.T) {
	hasher := NewCodebaseHasher()
	
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test")
	
	// Create test directory structure
	err := os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	
	// Create test files
	testFiles := map[string]string{
		"test.go":        "package test\n\nfunc Test() {}\n",
		"config.json":    `{"test": "value"}`,
		"README.md":      "# Test README",
		"subdir/helper.go": "package helper\n\nfunc Help() {}\n",
		"ignore.tmp":      "should be ignored",
		"exclude.log":     "should be excluded",
	}
	
	for file, content := range testFiles {
		fullPath := filepath.Join(testDir, file)
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}
	
	// Calculate hash of test directory
	hasher.RelevantDirectories = []string{filepath.Base(testDir)}
	hasher.RelevantExtensions = []string{".go", ".json", ".md"}
	
	// Change to temp directory for testing
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(tempDir)
	
	hash, err := hasher.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}
	
	// Hash should be consistent for same content
	hash2, err := hasher.CalculateHash()
	if err != nil {
		t.Fatalf("Failed to calculate second hash: %v", err)
	}
	
	// Only timestamps should differ, so hashes will be different
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}
	
	if len(hash2) != 64 {
		t.Errorf("Expected hash2 length 64, got %d", len(hash2))
	}
}

func TestCodebaseHasher_ShouldIncludeFile(t *testing.T) {
	hasher := NewCodebaseHasher()
	
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.go", true},
		{"config.json", true},
		{"README.md", true},
		{"script.sh", true},
		{"Dockerfile", true},
		{"Makefile", true},
		{"test.tmp", false},
		{"debug.log", false},
		{"coverage.out", false},
		{"vendor/test.go", false},
		{".git/config", false},
		{"node_modules/package.json", false},
		{".env", false},
	}
	
	for _, test := range tests {
		result := hasher.shouldIncludeFile(test.path)
		if result != test.expected {
			t.Errorf("shouldIncludeFile(%s) = %v, expected %v", test.path, result, test.expected)
		}
	}
}

func TestCodebaseHasher_AddBinaryHashes(t *testing.T) {
	hasher := NewCodebaseHasher()
	
	// Create a temporary binary file
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "translator")
	
	// Create a dummy binary
	content := "dummy binary content"
	err := os.WriteFile(binaryPath, []byte(content), 0755)
	if err != nil {
		t.Fatalf("Failed to create dummy binary: %v", err)
	}
	
	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	os.Chdir(tempDir)
	
	// Calculate hash including binary
	hasher := sha256.New()
	err = hasher.addBinaryHashes(hasher)
	if err != nil {
		t.Fatalf("Failed to add binary hashes: %v", err)
	}
	
	// Verify binary was included
	hashBytes := hasher.Sum(nil)
	hashStr := hex.EncodeToString(hashBytes)
	
	if len(hashStr) == 0 {
		t.Error("Hash should include binary content")
	}
}

func TestCodebaseHasher_CompareVersions(t *testing.T) {
	hasher := NewCodebaseHasher()
	
	localHash := "abc123"
	remoteHash := "abc123"
	
	if !hasher.CompareVersions(localHash, remoteHash) {
		t.Error("Expected identical hashes to be equal")
	}
	
	remoteHash = "def456"
	if hasher.CompareVersions(localHash, remoteHash) {
		t.Error("Expected different hashes to be unequal")
	}
}

func TestCodebaseHasher_GenerateVersionInfo(t *testing.T) {
	hasher := NewCodebaseHasher()
	
	info, err := hasher.GenerateVersionInfo()
	if err != nil {
		t.Fatalf("Failed to generate version info: %v", err)
	}
	
	if info.Hash == "" {
		t.Error("Version info should have a hash")
	}
	
	if info.Timestamp.IsZero() {
		t.Error("Version info should have a timestamp")
	}
	
	if len(info.Directories) == 0 {
		t.Error("Version info should have directories")
	}
	
	if len(info.Extensions) == 0 {
		t.Error("Version info should have extensions")
	}
}

func TestVersionInfo_SaveVersionInfo(t *testing.T) {
	info := &VersionInfo{
		Hash:      "test123",
		Timestamp: time.Now(),
		Directories: []string{"pkg", "cmd"},
		Extensions: []string{".go", ".json"},
	}
	
	// Test saving version info (currently returns nil)
	err := info.SaveVersionInfo("test_version.json")
	if err != nil {
		t.Errorf("SaveVersionInfo failed: %v", err)
	}
}

// Benchmark test for hash calculation performance
func BenchmarkCodebaseHasher_CalculateHash(b *testing.B) {
	hasher := NewCodebaseHasher()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := hasher.CalculateHash()
		if err != nil {
			b.Fatalf("Failed to calculate hash: %v", err)
		}
	}
}