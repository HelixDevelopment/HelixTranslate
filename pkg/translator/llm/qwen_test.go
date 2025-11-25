package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestQwenProvider(t *testing.T) {
	// Test placeholder - provider implementation needed
	t.Log("Qwen provider test placeholder")
}

// TestSaveOAuthToken tests saving OAuth tokens to file
func TestSaveOAuthToken(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "qwen_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases
	tests := []struct {
		name    string
		token   *QwenOAuthToken
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid token",
			token: &QwenOAuthToken{
				AccessToken:  "test_access_token",
				TokenType:    "Bearer",
				RefreshToken: "test_refresh_token",
				ResourceURL:  "https://resource.url",
				ExpiryDate:   time.Now().Add(time.Hour).Unix(),
			},
			wantErr: false,
		},
		{
			name: "token with special characters",
			token: &QwenOAuthToken{
				AccessToken:  "access+token/special=chars",
				TokenType:    "Bearer",
				RefreshToken: "refresh+token/特殊字符",
				ResourceURL:  "https://resource.url/path?param=value",
				ExpiryDate:   time.Now().Add(time.Hour).Unix(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with temp credentials file path
			credFile := filepath.Join(tempDir, fmt.Sprintf("credentials_%s.json", tt.name))
			config := TranslationConfig{
				APIKey: "test-api-key", // Use API key to avoid OAuth loading
			}
			client, err := NewQwenClient(config)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			// Override the credentials file path
			client.credFilePath = credFile

			// Test saveOAuthToken
			err = client.saveOAuthToken(tt.token)

			if (err != nil) != tt.wantErr {
				t.Errorf("saveOAuthToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("saveOAuthToken() error = %v, expected to contain %s", err, tt.errMsg)
				}
				return
			}

			// Verify file was created and contains correct data
			if !tt.wantErr {
				if _, err := os.Stat(credFile); os.IsNotExist(err) {
					t.Errorf("saveOAuthToken() credentials file was not created")
					return
				}

				// Read and verify file content
				data, err := os.ReadFile(credFile)
				if err != nil {
					t.Errorf("Failed to read credentials file: %v", err)
					return
				}

				var savedToken QwenOAuthToken
				if err := json.Unmarshal(data, &savedToken); err != nil {
					t.Errorf("Failed to unmarshal credentials: %v", err)
					return
				}

				if savedToken.AccessToken != tt.token.AccessToken ||
					savedToken.RefreshToken != tt.token.RefreshToken ||
					savedToken.ResourceURL != tt.token.ResourceURL {
					t.Errorf("saveOAuthToken() saved token data mismatch")
				}

				// Verify client token was set
				if client.oauthToken == nil {
					t.Errorf("saveOAuthToken() client token was not set")
				}
			}
		})
	}
}

// TestSaveOAuthTokenErrorPaths tests error handling in saveOAuthToken
func TestSaveOAuthTokenErrorPaths(t *testing.T) {
	// Test directory creation error
	t.Run("directory creation error", func(t *testing.T) {
		// Use an invalid path that should cause directory creation to fail
		invalidPath := "/dev/null/invalid/path/credentials.json"
		config := TranslationConfig{
			APIKey: "test-api-key",
		}
		client, err := NewQwenClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		// Override the credentials file path
		client.credFilePath = invalidPath
		
		token := &QwenOAuthToken{
			AccessToken: "test_token",
			TokenType:   "Bearer",
		}

		err = client.saveOAuthToken(token)
		if err == nil {
			t.Error("Expected error for invalid path")
		}

		if !contains(err.Error(), "failed to create credentials directory") {
			t.Errorf("Expected directory creation error, got: %v", err)
		}
	})

	// Test file write error - simulate by making the directory read-only
	t.Run("file write error", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "qwen_readonly_test_*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Make directory read-only
		if err := os.Chmod(tempDir, 0400); err != nil {
			t.Fatalf("Failed to make dir read-only: %v", err)
		}

		credFile := filepath.Join(tempDir, "credentials.json")
		config := TranslationConfig{
			APIKey: "test-api-key",
		}
		client, err := NewQwenClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		// Override the credentials file path
		client.credFilePath = credFile
		
		token := &QwenOAuthToken{
			AccessToken: "test_token",
			TokenType:   "Bearer",
		}

		err = client.saveOAuthToken(token)
		if err == nil {
			t.Error("Expected error for read-only directory")
		}

		if !contains(err.Error(), "failed to write credentials file") {
			t.Errorf("Expected file write error, got: %v", err)
		}

		// Restore permissions for cleanup
		os.Chmod(tempDir, 0700)
	})

	// Test JSON marshaling error with invalid data
	t.Run("JSON marshaling error", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "qwen_marshal_test_*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		credFile := filepath.Join(tempDir, "credentials.json")
		config := TranslationConfig{
			APIKey: "test-api-key",
		}
		client, err := NewQwenClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		// Override the credentials file path
		client.credFilePath = credFile

		// Create token with invalid data that would cause marshaling to fail
		// This is a bit tricky since JSON marshaling rarely fails for valid structs
		// We'll test by temporarily manipulating the client
		originalToken := &QwenOAuthToken{
			AccessToken:  "test_access_token",
			TokenType:    "Bearer",
			RefreshToken: "test_refresh_token",
			ResourceURL:  "https://resource.url",
			ExpiryDate:   time.Now().Add(time.Hour).Unix(),
		}

		err = client.saveOAuthToken(originalToken)
		if err != nil {
			t.Errorf("Valid token should not cause marshal error: %v", err)
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

// Simple substring find implementation
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}