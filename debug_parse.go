package main

import (
	"archive/zip"
	"fmt"
	"log"
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/markdown"
	"os"
)

func main() {
	// Test creating an EPUB using the MarkdownToEPUBConverter
	converter := markdown.NewMarkdownToEPUBConverter()
	
	// Create a simple markdown file
	mdContent := `# Test Book

This is a test book with some content.

## Chapter 1

This is first chapter.

## Chapter 2

This is second chapter.
`
	
	// Write markdown to temp file
	tmpDir := "/tmp/test_epub"
	os.MkdirAll(tmpDir, 0755)
	mdPath := tmpDir + "/test.md"
	os.WriteFile(mdPath, []byte(mdContent), 0644)
	
	// Convert to EPUB
	epubPath := tmpDir + "/test.epub"
	err := converter.ConvertMarkdownToEPUB(mdPath, epubPath)
	if err != nil {
		log.Fatalf("Failed to convert to EPUB: %v", err)
	}
	
	// Parse with EPUB parser to see what we get
	parser := ebook.NewEPUBParser()
	book, err := parser.Parse(epubPath)
	if err != nil {
		log.Fatalf("Failed to parse EPUB: %v", err)
	}
	
	fmt.Printf("Title: %s\n", book.Metadata.Title)
	fmt.Printf("Authors: %v\n", book.Metadata.Authors)
	fmt.Printf("Chapters: %d\n", len(book.Chapters))
	for i, chapter := range book.Chapters {
		fmt.Printf("  Chapter %d: %s (%d sections)\n", i+1, chapter.Title, len(chapter.Sections))
		for j, section := range chapter.Sections {
			fmt.Printf("    Section %d: %s\n", j+1, section.Content[:50])
		}
	}
	
	// Check chapter files
	r, err := zip.OpenReader(epubPath)
	if err != nil {
		log.Fatalf("Failed to open EPUB: %v", err)
	}
	defer r.Close()
	
	fmt.Println("\nChapter files in EPUB:")
	for _, f := range r.File {
		if f.Name == "OEBPS/chapter1.xhtml" || f.Name == "OEBPS/chapter2.xhtml" {
			fmt.Printf("  %s\n", f.Name)
		}
	}
	
	// Clean up
	os.RemoveAll(tmpDir)
}