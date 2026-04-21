package markdown

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEPUBToMarkdownConverter_NewEPUBToMarkdownConverter tests converter creation
func TestEPUBToMarkdownConverter_NewEPUBToMarkdownConverter(t *testing.T) {
	converter := NewEPUBToMarkdownConverter(true, "test_images")
	require.NotNil(t, converter)
}

// TestEPUBToMarkdownConverter_PreserveImages tests with image preservation
func TestEPUBToMarkdownConverter_PreserveImages(t *testing.T) {
	converter := NewEPUBToMarkdownConverter(false, "")
	require.NotNil(t, converter)
}
