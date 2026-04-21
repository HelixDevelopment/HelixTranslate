package utils

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestConfig provides configuration for tests
type TestConfig struct {
	DatabaseURL string
	APIToken    string
	TestTimeout time.Duration
	TempDir     string
}

// SetupTestEnvironment creates test environment
func SetupTestEnvironment(t *testing.T) *TestConfig {
	t.Helper()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	return &TestConfig{
		DatabaseURL: "sqlite://" + dbPath,
		APIToken:    "test-token-" + t.Name(),
		TestTimeout: 30 * time.Second,
		TempDir:     tempDir,
	}
}

// CleanupTestEnvironment cleans up test resources
func CleanupTestEnvironment(t *testing.T, config *TestConfig) {
	t.Helper()

	if config != nil && config.DatabaseURL != "" {
		os.Remove(config.DatabaseURL)
	}
}

// CreateTestEPUB creates a test EPUB file
func CreateTestEPUB(t *testing.T, title, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	epubPath := filepath.Join(tempDir, "test.epub")

	// Create minimal EPUB content for testing
	epubContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf">
  <metadata>
    <dc:title xmlns:dc="http://purl.org/dc/elements/1.1/">%s</dc:title>
  </metadata>
  <manifest>
    <item id="chapter1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter1"/>
  </spine>
</package>`, title)

	require.NoError(t, os.WriteFile(epubPath, []byte(epubContent), 0644))

	return epubPath
}

// CreateTestContext creates a test context with timeout
func CreateTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// CreateTestFB2 creates a test FB2 file
func CreateTestFB2(t *testing.T, title, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	fb2Path := filepath.Join(tempDir, "test.fb2")

	fb2Content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
<description>
<title-info>
<book-title>%s</book-title>
<lang>en</lang>
</title-info>
</description>
<body>
<section>
<title><p>%s</p></title>
<p>%s</p>
</section>
</body>
</FictionBook>`, title, title, content)

	require.NoError(t, os.WriteFile(fb2Path, []byte(fb2Content), 0644))

	return fb2Path
}

// CreateTestTXT creates a test text file
func CreateTestTXT(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	txtPath := filepath.Join(tempDir, "test.txt")

	require.NoError(t, os.WriteFile(txtPath, []byte(content), 0644))

	return txtPath
}

// CreateTestHTML creates a test HTML file
func CreateTestHTML(t *testing.T, title, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	htmlPath := filepath.Join(tempDir, "test.html")

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s</title></head>
<body>
<h1>%s</h1>
<p>%s</p>
</body>
</html>`, title, title, content)

	require.NoError(t, os.WriteFile(htmlPath, []byte(htmlContent), 0644))

	return htmlPath
}
