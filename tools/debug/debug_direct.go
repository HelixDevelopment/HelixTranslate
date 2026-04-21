package main

import (
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/markdown"
	"fmt"
	"os"
)

func main() {
	// Create a simple markdown file directly
	mdContent := `---
title: Test Book
---

## Chapter 1

This is some content.
`

	// Write to temp file
	tmpMd := "/tmp/test_direct.md"
	os.WriteFile(tmpMd, []byte(mdContent), 0644)
	defer os.Remove(tmpMd)

	// Convert to EPUB using MarkdownToEPUBConverter
	epubPath := "/tmp/test_direct.epub"
	converter := markdown.NewMarkdownToEPUBConverter()
	err := converter.ConvertMarkdownToEPUB(tmpMd, epubPath)
	if err != nil {
		fmt.Printf("Failed to convert: %v\n", err)
		return
	}
	defer os.Remove(epubPath)

	// Check format
	detector := format.NewDetector()
	detectedFormat, err := detector.DetectFile(epubPath)
	if err != nil {
		fmt.Printf("Detection error: %v\n", err)
		return
	}

	fmt.Printf("Detected format: %s\n", detectedFormat)
}
