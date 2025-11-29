package distributed

import (
	"testing"
	"os"
)

func TestGetCodebaseVersion(t *testing.T) {
	t.Run("getCodebaseVersion_NoVersionFile", func(t *testing.T) {
		// Create a temporary directory without VERSION file
		tmpDir := t.TempDir()
		oldWD, _ := os.Getwd()
		defer os.Chdir(oldWD)
		
		// Change to temp dir
		os.Chdir(tmpDir)
		
		// Get version - should try git and fallback to unknown
		version := getCodebaseVersion()
		
		// Should return unknown since no VERSION file and not in a git repo
		if version != "unknown" {
			t.Errorf("Expected 'unknown' version without VERSION file or git repo, got: %s", version)
		}
	})
	
	t.Run("getCodebaseVersion_WithVersionFile", func(t *testing.T) {
		// Create a temporary directory with VERSION file
		tmpDir := t.TempDir()
		oldWD, _ := os.Getwd()
		defer os.Chdir(oldWD)
		
		// Create VERSION file
		versionFile := tmpDir + "/VERSION"
		err := os.WriteFile(versionFile, []byte("v1.2.3\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create VERSION file: %v", err)
		}
		
		// Change to temp dir
		os.Chdir(tmpDir)
		
		// Get version - should read from VERSION file
		version := getCodebaseVersion()
		
		if version != "v1.2.3" {
			t.Errorf("Expected 'v1.2.3' from VERSION file, got: %s", version)
		}
	})
	
	t.Run("getCodebaseVersion_EmptyVersionFile", func(t *testing.T) {
		// Create a temporary directory with empty VERSION file
		tmpDir := t.TempDir()
		oldWD, _ := os.Getwd()
		defer os.Chdir(oldWD)
		
		// Create empty VERSION file
		versionFile := tmpDir + "/VERSION"
		err := os.WriteFile(versionFile, []byte("  \n  "), 0644)
		if err != nil {
			t.Fatalf("Failed to create VERSION file: %v", err)
		}
		
		// Change to temp dir
		os.Chdir(tmpDir)
		
		// Get version - should read from VERSION file and trim whitespace
		version := getCodebaseVersion()
		
		if version != "" {
			t.Errorf("Expected empty string from empty VERSION file, got: '%s'", version)
		}
	})
}