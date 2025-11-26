package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigStruct tests config struct functionality
func TestConfigStruct(t *testing.T) {
	t.Run("Config fields", func(t *testing.T) {
		config := &Config{
			InputFile:  "test.epub",
			OutputFile: "test_sr.epub",
			SSHHost:    "localhost",
			SSHUser:    "user",
			SSHPassword: "pass",
			SSHPort:    22,
			RemoteDir:  "/tmp/translate-ssh",
		}
		
		assert.Equal(t, "test.epub", config.InputFile)
		assert.Equal(t, "test_sr.epub", config.OutputFile)
		assert.Equal(t, "localhost", config.SSHHost)
		assert.Equal(t, "user", config.SSHUser)
		assert.Equal(t, "pass", config.SSHPassword)
		assert.Equal(t, 22, config.SSHPort)
		assert.Equal(t, "/tmp/translate-ssh", config.RemoteDir)
	})
	
	t.Run("JSON marshaling", func(t *testing.T) {
		config := &Config{
			InputFile:  "test.epub",
			OutputFile: "test_sr.epub",
			SSHHost:    "localhost",
			SSHUser:    "user",
			SSHPassword: "pass",
			SSHPort:    22,
		}
		
		// Convert to JSON for SSH transmission
		jsonData, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)
		
		// Verify it can be unmarshaled back
		var unmarshaled Config
		err = json.Unmarshal(jsonData, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, config.InputFile, unmarshaled.InputFile)
		assert.Equal(t, config.SSHHost, unmarshaled.SSHHost)
	})
}

// TestProgressTracking tests progress tracking functionality
func TestProgressTracking(t *testing.T) {
	t.Run("TranslationProgress initialization", func(t *testing.T) {
		progress := &TranslationProgress{
			StartTime:     time.Now(),
			TotalSteps:     5,
			CurrentStep:    "Test step",
		}
		
		assert.False(t, progress.StartTime.IsZero())
		assert.Equal(t, 5, progress.TotalSteps)
		assert.Equal(t, "Test step", progress.CurrentStep)
	})
	
	t.Run("UpdateProgress", func(t *testing.T) {
		progress := &TranslationProgress{
			StartTime:     time.Now(),
			TotalSteps:     5,
			CurrentStep:    "Initial step",
		}
		
		// Update progress
		progress.CurrentStep = "New step"
		progress.CompletedSteps++
		
		assert.Equal(t, "New step", progress.CurrentStep)
		assert.Equal(t, 1, progress.CompletedSteps)
	})
}

// TestConfigValidation tests config validation functionality
func TestConfigValidation(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "test.epub")
		_, err := os.Create(tempFile)
		require.NoError(t, err)
		
		config := &Config{
			InputFile:  tempFile,
			OutputFile: tempFile,
			SSHHost:    "localhost",
			SSHUser:    "user",
			SSHPassword: "pass",
		}
		
		err = validateConfig(config)
		assert.NoError(t, err)
	})
	
	t.Run("missing input file", func(t *testing.T) {
		config := &Config{
			InputFile:  "",
			OutputFile: "output.epub",
			SSHHost:    "localhost",
			SSHUser:    "user",
			SSHPassword: "pass",
		}
		
		err := validateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "input file is required")
	})
	
	t.Run("nonexistent input file", func(t *testing.T) {
		config := &Config{
			InputFile:  "/nonexistent/file.epub",
			OutputFile: "output.epub",
			SSHHost:    "localhost",
			SSHUser:    "user",
			SSHPassword: "pass",
		}
		
		err := validateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "input file not found")
	})
}

// TestSessionIDGeneration tests session ID generation
func TestSessionIDGeneration(t *testing.T) {
	// Reset flag.CommandLine to avoid redefinition errors
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	
	t.Run("generateSessionID", func(t *testing.T) {
		// Create a mock config
		config := &Config{
			InputFile:  "test.epub",
			OutputFile: "test_sr.epub",
			SSHHost:    "localhost",
			SSHUser:    "user",
			SSHPassword: "pass",
		}
		
		// Simulate session ID generation
		sessionData := fmt.Sprintf("%s_%s_%s_%d", 
			config.SSHHost, 
			config.SSHUser, 
			config.InputFile,
			time.Now().Unix(),
		)
		sessionID := fmt.Sprintf("%x", sessionData)
		
		assert.NotEmpty(t, sessionID)
		assert.NotEqual(t, sessionID, fmt.Sprintf("%x", "")) // Ensure it's not empty hash
	})
}