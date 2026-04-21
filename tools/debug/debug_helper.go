package main

import (
	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/format"
	"digital.vasic.translator/pkg/markdown"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	// Test creating an EPUB using the helper
	book := &ebook.Book{
		Metadata: ebook.Metadata{
			Title:   "Test",
			Authors: []string{"Author"},
		},
		Chapters: []ebook.Chapter{
			{
				Title: "Chapter 1",
				Sections: []ebook.Section{
					{Content: "Content here"},
				},
			},
		},
	}

	outputPath := "/tmp/test_helper.epub"

	// Test our helper
	err := createSimpleEPUBForTest(book, outputPath)
	if err != nil {
		log.Fatalf("Helper failed: %v", err)
	}

	// Try to parse it
	parser := ebook.NewUniversalParser()
	parsedBook, err := parser.Parse(outputPath)
	if err != nil {
		log.Fatalf("Parser failed: %v", err)
	}

	fmt.Printf("Successfully parsed! Title: %s, Chapters: %d\n", parsedBook.Metadata.Title, len(parsedBook.Chapters))

	os.Remove(outputPath)
}

// createSimpleEPUBForTest creates a simple EPUB file from a Book structure for testing
func createSimpleEPUBForTest(book *ebook.Book, outputPath string) error {
	// Use the MarkdownToEPUBConverter which creates valid EPUBs
	converter := markdown.NewMarkdownToEPUBConverter()

	// Create a markdown representation of the book
	var md strings.Builder

	// Add frontmatter
	md.WriteString(fmt.Sprintf("---\ntitle: %s\n", book.Metadata.Title))
	if len(book.Metadata.Authors) > 0 {
		md.WriteString(fmt.Sprintf("authors: %s\n", strings.Join(book.Metadata.Authors, ", ")))
	}
	md.WriteString("---\n\n")

	// Add expected format after frontmatter (title, author, separators)
	md.WriteString(fmt.Sprintf("# %s\n\n", book.Metadata.Title))
	md.WriteString(fmt.Sprintf("**%s**\n\n", strings.Join(book.Metadata.Authors, ", ")))
	md.WriteString("---\n\n")

	// Add chapters
	for _, chapter := range book.Chapters {
		md.WriteString(fmt.Sprintf("# %s\n\n", chapter.Title))
		for _, section := range chapter.Sections {
			md.WriteString(fmt.Sprintf("%s\n\n", section.Content))
		}
	}

	// Write to temporary markdown file
	tmpMd := outputPath + ".md"
	mdContent := md.String()
	fmt.Printf("Generated markdown:\n%s\n", mdContent)
	if err := os.WriteFile(tmpMd, []byte(mdContent), 0644); err != nil {
		return err
	}

	// Convert markdown to EPUB
	if err := converter.ConvertMarkdownToEPUB(tmpMd, outputPath); err != nil {
		return err
	}

	// Check format right away
	detector := format.NewDetector()
	detectedFormat, _ := detector.DetectFile(outputPath)
	fmt.Printf("Detected format after creation: %s\n", detectedFormat)

	// Remove temp markdown AFTER checking format
	os.Remove(tmpMd)

	return nil
}
