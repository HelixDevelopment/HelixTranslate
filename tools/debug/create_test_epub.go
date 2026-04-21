package main

import (
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/markdown"
	"fmt"
	"os"
)

func main() {
	// Create a simple test book
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
	epubPath := "/tmp/test_simple.epub"
	if err := createSimpleEPUB(book, epubPath); err != nil {
		fmt.Printf("Failed to create EPUB: %v\n", err)
		return
	}

	fmt.Printf("Created EPUB: %s\n", epubPath)
}
