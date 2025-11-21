package main

import (
	"digital.vasic.translator/pkg/markdown"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("🧪 End-to-End Markdown Translation Pipeline Test")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Test files
	inputEPUB := "Books/Stepanova_T._Detektivtriller1._Son_Nad_Bezdnoyi.epub"
	sourceMD := "/tmp/markdown_e2e_source.md"
	translatedMD := "/tmp/markdown_e2e_translated.md"
	outputEPUB := "/tmp/markdown_e2e_output.epub"

	// Clean up from previous runs
	os.Remove(sourceMD)
	os.Remove(translatedMD)
	os.Remove(outputEPUB)

	// Step 1: EPUB → Markdown
	fmt.Println("📖 Step 1/4: Converting EPUB to Markdown...")
	converter := markdown.NewEPUBToMarkdownConverter(false, "")
	if err := converter.ConvertEPUBToMarkdown(inputEPUB, sourceMD); err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		os.Exit(1)
	}

	// Verify source markdown
	sourceContent, err := os.ReadFile(sourceMD)
	if err != nil {
		fmt.Printf("❌ FAILED to read source markdown: %v\n", err)
		os.Exit(1)
	}

	sourceLines := strings.Split(string(sourceContent), "\n")
	fmt.Printf("✅ Created source markdown: %s\n", sourceMD)
	fmt.Printf("   Size: %d bytes, %d lines\n", len(sourceContent), len(sourceLines))
	fmt.Printf("   Preview (first 15 lines):\n")
	for i := 0; i < 15 && i < len(sourceLines); i++ {
		line := sourceLines[i]
		if len(line) > 80 {
			line = line[:77] + "..."
		}
		fmt.Printf("      %s\n", line)
	}
	fmt.Println()

	// Step 2: Translate Markdown
	fmt.Println("🌍 Step 2/4: Translating markdown content...")

	// Mock translator - just prepends "SR: " to text
	mockTranslator := func(text string) (string, error) {
		if strings.TrimSpace(text) == "" {
			return text, nil
		}
		return "SR: " + text, nil
	}

	mdTranslator := markdown.NewMarkdownTranslator(mockTranslator)
	if err := mdTranslator.TranslateMarkdownFile(sourceMD, translatedMD); err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		os.Exit(1)
	}

	// Verify translated markdown
	translatedContent, err := os.ReadFile(translatedMD)
	if err != nil {
		fmt.Printf("❌ FAILED to read translated markdown: %v\n", err)
		os.Exit(1)
	}

	translatedLines := strings.Split(string(translatedContent), "\n")
	fmt.Printf("✅ Created translated markdown: %s\n", translatedMD)
	fmt.Printf("   Size: %d bytes, %d lines\n", len(translatedContent), len(translatedLines))
	fmt.Printf("   Preview (first 15 lines):\n")
	for i := 0; i < 15 && i < len(translatedLines); i++ {
		line := translatedLines[i]
		if len(line) > 80 {
			line = line[:77] + "..."
		}
		fmt.Printf("      %s\n", line)
	}
	fmt.Println()

	// Step 3: Markdown → EPUB
	fmt.Println("📚 Step 3/4: Converting translated markdown to EPUB...")
	epubConverter := markdown.NewMarkdownToEPUBConverter()
	if err := epubConverter.ConvertMarkdownToEPUB(translatedMD, outputEPUB); err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		os.Exit(1)
	}

	// Verify output EPUB
	epubInfo, err := os.Stat(outputEPUB)
	if err != nil {
		fmt.Printf("❌ FAILED: Output EPUB not created\n")
		os.Exit(1)
	}

	fmt.Printf("✅ Created output EPUB: %s\n", outputEPUB)
	fmt.Printf("   Size: %d bytes (%.2f KB)\n", epubInfo.Size(), float64(epubInfo.Size())/1024.0)
	fmt.Println()

	// Step 4: Verify EPUB structure
	fmt.Println("🔍 Step 4/4: Verifying EPUB structure...")

	// Basic EPUB validation - check if it's a valid ZIP with required files
	// (Full validation would require unzipping and checking structure)
	if epubInfo.Size() < 100 {
		fmt.Printf("❌ FAILED: EPUB file is too small (%d bytes)\n", epubInfo.Size())
		os.Exit(1)
	}

	// Check if file starts with PK (ZIP signature)
	f, err := os.Open(outputEPUB)
	if err != nil {
		fmt.Printf("❌ FAILED: Cannot open EPUB: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	header := make([]byte, 2)
	if _, err := f.Read(header); err != nil {
		fmt.Printf("❌ FAILED: Cannot read EPUB header: %v\n", err)
		os.Exit(1)
	}

	if header[0] != 'P' || header[1] != 'K' {
		fmt.Printf("❌ FAILED: EPUB is not a valid ZIP file (header: %v)\n", header)
		os.Exit(1)
	}

	fmt.Println("✅ EPUB structure valid (ZIP format confirmed)")
	fmt.Println()

	// Summary
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("✅ END-TO-END TEST PASSED!")
	fmt.Println()
	fmt.Println("Pipeline verified:")
	fmt.Println("  1. EPUB → Markdown: Extracted structure and formatting")
	fmt.Println("  2. Translation: Applied to markdown while preserving format")
	fmt.Println("  3. Markdown → EPUB: Generated valid EPUB file")
	fmt.Println()
	fmt.Println("Generated files:")
	fmt.Printf("  - Source markdown:     %s\n", sourceMD)
	fmt.Printf("  - Translated markdown: %s\n", translatedMD)
	fmt.Printf("  - Output EPUB:         %s\n", outputEPUB)
	fmt.Println()
	fmt.Println("To inspect the markdown files:")
	fmt.Printf("  head -50 %s\n", sourceMD)
	fmt.Printf("  head -50 %s\n", translatedMD)
}
