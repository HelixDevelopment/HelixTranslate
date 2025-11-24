package main

import (
	"digital.vasic.translator/pkg/markdown"
	"digital.vasic.translator/pkg/ebook"
	"fmt"
	"os"
)

func main() {
	// Create temp directory
	tmpDir := "/tmp/test_simple_epub"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Create book with cover
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Test Book",
			Authors: []string{"Test Author"},
			Cover:   []byte{0xFF, 0xD8, 0xFF, 0xE0}, // JPEG header
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Test content"},
				},
			},
		},
	}

	// Write EPUB
	epubPath := tmpDir + "/test.epub"
	converter := markdown.NewMarkdownToEPUBConverter()
	
	// Create a markdown representation of the book
	mdContent := `---
title: Test Book
authors: Test Author
cover: cover.jpg
---

# Test Book

**Test Author**

---

## Chapter 1

Test content
`
	
	if err := os.WriteFile(tmpDir+"/test.md", []byte(mdContent), 0644); err != nil {
		panic(err)
	}
	
	// Create a 4-byte cover image
	coverPath := tmpDir + "/cover.jpg"
	if err := os.WriteFile(coverPath, []byte{0xFF, 0xD8, 0xFF, 0xE0}, 0644); err != nil {
		panic(err)
	}
	
	if err := converter.ConvertMarkdownToEPUB(tmpDir+"/test.md", epubPath); err != nil {
		fmt.Printf("Error creating EPUB: %v\n", err)
		return
	}

	// Convert back to markdown
	mdPath := tmpDir + "/output.md"
	epubToMd := markdown.NewEPUBToMarkdownConverter(false, "")
	if err := epubToMd.ConvertEPUBToMarkdown(epubPath, mdPath); err != nil {
		fmt.Printf("Error converting EPUB to Markdown: %v\n", err)
		return
	}

	// Read result
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		fmt.Printf("Error reading output: %v\n", err)
		return
	}

	fmt.Printf("Success! Generated markdown with %d bytes\n", len(mdData))
	fmt.Printf("First 200 chars: %s\n", string(mdData[:200]))
}