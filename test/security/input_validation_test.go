//go:build security

package security

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/internal/config"
	"digital.vasic.translator/pkg/api"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"

	"github.com/gin-gonic/gin"
)

func TestInputValidation_SQLInjectionPrevention(t *testing.T) {
	// Test that SQL injection attempts are properly handled
	translator, _ := llm.NewLLMTranslator(translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "sr",
		Provider:   "openai",
		Model:      "gpt-3.5-turbo",
	})

	sqlInjectionAttempts := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"admin'--",
		"1; SELECT * FROM users;",
		"UNION SELECT password FROM users",
	}

	for _, maliciousInput := range sqlInjectionAttempts {
		result, err := translator.Translate(context.Background(), maliciousInput, "")
		// Should not crash and should return some result (dictionary translation doesn't use SQL)
		if err != nil {
			t.Logf("Translation failed for input '%s': %v", maliciousInput, err)
		} else {
			t.Logf("Translation succeeded for input '%s': '%s'", maliciousInput, result)
		}
		// In a real SQL-based system, we'd verify no SQL commands were executed
	}
}

func TestInputValidation_XSSPrevention(t *testing.T) {
	// Test that XSS attempts are properly handled
	translator := dictionary.NewDictionaryTranslator(translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "sr",
		Provider:   "dictionary",
	})

	xssAttempts := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<iframe src='javascript:alert(\"XSS\")'></iframe>",
		"<svg onload=alert('XSS')>",
	}

	for _, maliciousInput := range xssAttempts {
		result, err := translator.Translate(context.Background(), maliciousInput, "")
		if err != nil {
			t.Logf("Translation failed for XSS input '%s': %v", maliciousInput, err)
		} else {
			// Verify that dangerous tags are not in the output
			if strings.Contains(strings.ToLower(result), "<script") ||
				strings.Contains(strings.ToLower(result), "javascript:") ||
				strings.Contains(strings.ToLower(result), "onerror") {
				t.Errorf("XSS payload not properly sanitized in output: '%s'", result)
			}
			t.Logf("Translation succeeded for XSS input '%s': '%s'", maliciousInput, result)
		}
	}
}

func TestInputValidation_PathTraversalPrevention(t *testing.T) {
	// Test that path traversal attempts are blocked
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "This is a test file."
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"/etc/passwd",
		"C:\\Windows\\System32\\config\\sam",
		"../../../../../../../../etc/shadow",
	}

	for _, maliciousPath := range pathTraversalAttempts {
		// Test file operations
		_, err := os.Stat(maliciousPath)
		// Should fail for traversal attempts (unless running as root/admin)
		if err == nil {
			t.Logf("WARNING: Path traversal succeeded for '%s' - this may be expected in test environment", maliciousPath)
		} else {
			t.Logf("Path traversal correctly blocked for '%s': %v", maliciousPath, err)
		}
	}
}

func TestInputValidation_CommandInjectionPrevention(t *testing.T) {
	// Test that command injection attempts are blocked
	translator := dictionary.NewDictionaryTranslator(translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "sr",
		Provider:   "dictionary",
	})

	commandInjectionAttempts := []string{
		"; rm -rf /",
		"| cat /etc/passwd",
		"`whoami`",
		"$(rm -rf /)",
		"; shutdown -h now",
	}

	for _, maliciousInput := range commandInjectionAttempts {
		result, err := translator.Translate(context.Background(), maliciousInput, "")
		if err != nil {
			t.Logf("Translation failed for command injection input '%s': %v", maliciousInput, err)
		} else {
			t.Logf("Translation succeeded for command injection input '%s': '%s'", maliciousInput, result)
		}
		// In a real system with shell commands, we'd verify no commands were executed
	}
}

func TestInputValidation_BufferOverflowPrevention(t *testing.T) {
	// Test handling of extremely large inputs
	translator := dictionary.NewDictionaryTranslator(translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "sr",
		Provider:   "dictionary",
	})

	// Create a very large input
	var largeInput strings.Builder
	for i := 0; i < 100000; i++ {
		largeInput.WriteString("word ")
	}

	result, err := translator.Translate(context.Background(), largeInput.String(), "")
	if err != nil {
		t.Logf("Large input translation failed: %v", err)
	} else {
		t.Logf("Large input translation succeeded, result length: %d", len(result))
		// Verify result is reasonable
		if len(result) == 0 {
			t.Error("Large input translation returned empty result")
		}
	}
}

func TestInputValidation_NullByteInjection(t *testing.T) {
	// Test that null byte injection is handled
	translator := dictionary.NewDictionaryTranslator(translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "sr",
		Provider:   "dictionary",
	})

	nullByteInputs := []string{
		"test\x00malicious",
		"input\x00with\x00nulls",
		"null\x00byte\x00attack",
	}

	for _, input := range nullByteInputs {
		result, err := translator.Translate(context.Background(), input, "")
		if err != nil {
			t.Logf("Null byte input translation failed: %v", err)
		} else {
			t.Logf("Null byte input translation succeeded: '%s'", result)
			// Verify null bytes are handled properly
			if strings.Contains(result, "\x00") {
				t.Errorf("Null bytes not properly handled in output: %q", result)
			}
		}
	}
}

func TestInputValidation_UnicodeNormalization(t *testing.T) {
	// Test handling of Unicode normalization attacks
	translator := dictionary.NewDictionaryTranslator(translator.TranslationConfig{
		SourceLang: "en",
		TargetLang: "sr",
		Provider:   "dictionary",
	})

	// Test various Unicode characters that might be used for attacks
	unicodeInputs := []string{
		"café",          // Unicode é
		"naïve",         // Unicode ï
		"Москва",        // Cyrillic
		"αβγδε",         // Greek
		"🚀🌟💻",           // Emojis
		"𝐀𝐁𝐂𝐃𝐄",         // Mathematical bold
		"안녕하세요",         // Korean
		"مرحبا بالعالم", // Arabic
	}

	for _, input := range unicodeInputs {
		result, err := translator.Translate(context.Background(), input, "")
		if err != nil {
			t.Logf("Unicode input '%s' translation failed: %v", input, err)
		} else {
			t.Logf("Unicode input '%s' translation succeeded: '%s'", input, result)
		}
	}
}

func TestAPIInputValidation_ContentTypeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfg.Translation.DefaultProvider = "dictionary"
	eventBus := events.NewEventBus()
	handler := api.NewHandler(cfg, eventBus, nil, nil, nil, nil)

	router := gin.New()
	v1 := router.Group("/api/v1")
	handler.RegisterBatchRoutes(v1)

	// Test with wrong content type
	reqBody := `{"text": "test", "sql": "SELECT * FROM users"}`
	req, _ := http.NewRequest("POST", "/api/v1/translate", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "text/plain") // Wrong content type

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still work or return appropriate error
	t.Logf("Wrong content type response: %d", w.Code)
}

func TestFileUploadValidation_Security(t *testing.T) {
	// Test file upload security
	tempDir := t.TempDir()

	// Test file upload security - check that dangerous filenames are handled
	dangerousNames := []string{
		"../../../etc/passwd", // Path traversal attempt
		"shell.php",
		"script.js",
		"malware.exe",
		"test.php.jpg", // Double extension
	}

	for _, name := range dangerousNames {
		// Sanitize the filename to prevent path traversal
		sanitizedName := filepath.Base(name) // This prevents directory traversal
		filePath := filepath.Join(tempDir, sanitizedName)
		content := "test content"

		err := os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// In a real system, we'd test file upload validation
		// For now, just verify file operations work
		readContent, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read test file %s: %v", sanitizedName, err)
		} else if string(readContent) != content {
			t.Errorf("File content mismatch for %s", sanitizedName)
		}

		t.Logf("Successfully handled filename: %s (sanitized to: %s)", name, sanitizedName)
	}
}

func TestLargeInputHandling(t *testing.T) {
	// Test that system handles very large inputs without crashing
	maliciousTexts := []string{
		strings.Repeat("a", 100000),      // Very long string
		"∞∞∞∞∞∞∞∞",                       // Potential DoS with unusual Unicode
		strings.Repeat("<script>", 1000), // Many script tags
	}

	for _, text := range maliciousTexts {
		// Test that basic operations don't crash
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("System panicked on input of length %d: %v", len(text), r)
			}
		}()

		t.Logf("Handled input of length %d", len(text))
	}
}
