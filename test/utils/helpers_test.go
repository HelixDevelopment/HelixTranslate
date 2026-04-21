package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetupTestEnvironment(t *testing.T) {
	config := SetupTestEnvironment(t)

	assert.NotEmpty(t, config.DatabaseURL)
	assert.NotEmpty(t, config.APIToken)
	assert.True(t, config.TestTimeout > 0)
	assert.NotEmpty(t, config.TempDir)
}

func TestCreateTestContext(t *testing.T) {
	ctx := CreateTestContext(t)
	assert.NotNil(t, ctx)
}

func TestCreateTestEPUB(t *testing.T) {
	epubPath := CreateTestEPUB(t, "Test Book", "Test content")
	assert.FileExists(t, epubPath)
}

func TestCreateTestFB2(t *testing.T) {
	fb2Path := CreateTestFB2(t, "Test Book", "Test content")
	assert.FileExists(t, fb2Path)
}

func TestCreateTestTXT(t *testing.T) {
	txtPath := CreateTestTXT(t, "Test content")
	assert.FileExists(t, txtPath)
}

func TestCreateTestHTML(t *testing.T) {
	htmlPath := CreateTestHTML(t, "Test Title", "Test content")
	assert.FileExists(t, htmlPath)
}
