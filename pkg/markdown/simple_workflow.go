package markdown

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.translator/pkg/logger"
	"digital.vasic.translator/pkg/translator/llm"
)

// WorkflowConfig holds configuration for the markdown workflow
type WorkflowConfig struct {
	ChunkSize        int
	OverlapSize      int
	MaxConcurrency   int
	TranslationCache map[string]string
	LLMProvider      llm.LLMClient
}

// ProgressCallback function for progress updates
type ProgressCallback func(current, total int, message string)

// SimpleWorkflow handles ebook → markdown → translation → ebook workflow
type SimpleWorkflow struct {
	config     WorkflowConfig
	logger     logger.Logger
	callback   ProgressCallback
}

// NewSimpleWorkflow creates a new simple markdown workflow
func NewSimpleWorkflow(config WorkflowConfig, logger logger.Logger, callback ProgressCallback) *SimpleWorkflow {
	return &SimpleWorkflow{
		config:   config,
		logger:   logger,
		callback: callback,
	}
}

// ConvertToMarkdown converts an ebook file to markdown format
func (sw *SimpleWorkflow) ConvertToMarkdown(ctx context.Context, inputPath, outputPath string) error {
	sw.logger.Info("Converting ebook to markdown", map[string]interface{}{
		"input_path":  inputPath,
		"output_path": outputPath,
	})

	if sw.callback != nil {
		sw.callback(0, 100, "Converting ebook to markdown...")
	}

	// Determine file type and convert accordingly
	ext := strings.ToLower(filepath.Ext(inputPath))
	
	switch ext {
	case ".epub":
		return sw.convertEPUBToMarkdown(ctx, inputPath, outputPath)
	case ".fb2":
		return sw.convertFB2ToMarkdown(ctx, inputPath, outputPath)
	default:
		return fmt.Errorf("unsupported input format: %s", ext)
	}
}

// TranslateMarkdown translates a markdown file using the configured LLM provider
func (sw *SimpleWorkflow) TranslateMarkdown(ctx context.Context, inputPath, outputPath, fromLang, toLang string) error {
	sw.logger.Info("Translating markdown file", map[string]interface{}{
		"input_path":  inputPath,
		"output_path": outputPath,
		"from_lang":   fromLang,
		"to_lang":     toLang,
	})

	if sw.callback != nil {
		sw.callback(1, 100, "Translating markdown content...")
	}

	// Create translation function using the LLM provider
	translateFunc := func(text string) (string, error) {
		prompt := fmt.Sprintf(`Translate the following text from %s to %s. 
Provide ONLY the translation without any explanations, notes, or additional text.
Maintain the original formatting, line breaks, and structure.

Source text:
%s

Translation:`, fromLang, toLang, text)

		return sw.config.LLMProvider.Translate(ctx, text, prompt)
	}

	// Create markdown translator
	markdownTranslator := NewMarkdownTranslator(translateFunc)

	// Translate the markdown file
	if err := markdownTranslator.TranslateMarkdownFile(inputPath, outputPath); err != nil {
		return fmt.Errorf("failed to translate markdown: %w", err)
	}

	if sw.callback != nil {
		sw.callback(2, 100, "Markdown translation completed")
	}

	return nil
}

// ConvertFromMarkdown converts a markdown file to ebook format
func (sw *SimpleWorkflow) ConvertFromMarkdown(ctx context.Context, inputPath, outputPath string) error {
	sw.logger.Info("Converting markdown to ebook", map[string]interface{}{
		"input_path":  inputPath,
		"output_path": outputPath,
	})

	if sw.callback != nil {
		sw.callback(3, 100, "Converting markdown to ebook...")
	}

	// Determine output format and convert accordingly
	ext := strings.ToLower(filepath.Ext(outputPath))
	
	switch ext {
	case ".epub":
		return sw.convertMarkdownToEPUB(ctx, inputPath, outputPath)
	default:
		return fmt.Errorf("unsupported output format: %s", ext)
	}
}

// ExecuteFullWorkflow executes the complete conversion workflow
func (sw *SimpleWorkflow) ExecuteFullWorkflow(ctx context.Context, inputPath, outputPath, fromLang, toLang string) error {
	startTime := time.Now()
	
	// Generate intermediate file paths
	ext := strings.ToLower(filepath.Ext(inputPath))
	baseName := strings.TrimSuffix(inputPath, ext)
	
	originalMarkdownPath := baseName + "_original.md"
	translatedMarkdownPath := baseName + "_translated.md"

	// Step 1: Convert input ebook to markdown
	if err := sw.ConvertToMarkdown(ctx, inputPath, originalMarkdownPath); err != nil {
		return fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// Step 2: Translate markdown
	if err := sw.TranslateMarkdown(ctx, originalMarkdownPath, translatedMarkdownPath, fromLang, toLang); err != nil {
		return fmt.Errorf("failed to translate markdown: %w", err)
	}

	// Step 3: Convert translated markdown to output format
	if err := sw.ConvertFromMarkdown(ctx, translatedMarkdownPath, outputPath); err != nil {
		return fmt.Errorf("failed to convert from markdown: %w", err)
	}

	duration := time.Since(startTime)
	sw.logger.Info("Full workflow completed successfully", map[string]interface{}{
		"duration": duration.String(),
		"input_path": inputPath,
		"output_path": outputPath,
	})

	if sw.callback != nil {
		sw.callback(100, 100, "Workflow completed successfully")
	}

	return nil
}

// convertEPUBToMarkdown converts EPUB to markdown
func (sw *SimpleWorkflow) convertEPUBToMarkdown(ctx context.Context, inputPath, outputPath string) error {
	converter := NewEPUBToMarkdownConverter(false, "") // Don't preserve images for simplicity
	return converter.ConvertEPUBToMarkdown(inputPath, outputPath)
}

// convertFB2ToMarkdown converts FB2 to markdown (simplified version)
func (sw *SimpleWorkflow) convertFB2ToMarkdown(ctx context.Context, inputPath, outputPath string) error {
	// For now, create a simple text extraction
	// In a real implementation, you'd parse the FB2 XML properly
	
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read FB2 file: %w", err)
	}

	// Simple text extraction (this is very basic)
	text := string(content)
	
	// Create basic markdown
	markdownContent := fmt.Sprintf("# Book Content\n\n%s\n", text)
	
	return os.WriteFile(outputPath, []byte(markdownContent), 0644)
}

// convertMarkdownToEPUB converts markdown to EPUB
func (sw *SimpleWorkflow) convertMarkdownToEPUB(ctx context.Context, inputPath, outputPath string) error {
	// Read markdown content
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	// Create a temporary directory for EPUB structure
	tmpDir, err := os.MkdirTemp("", "epub_conversion_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create EPUB directory structure
	os.MkdirAll(filepath.Join(tmpDir, "META-INF"), 0755)
	os.MkdirAll(filepath.Join(tmpDir, "OEBPS"), 0755)

	// Create mimetype file
	err = os.WriteFile(filepath.Join(tmpDir, "mimetype"), []byte("application/epub+zip"), 0644)
	if err != nil {
		return fmt.Errorf("failed to create mimetype: %w", err)
	}

	// Create container.xml
	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	err = os.WriteFile(filepath.Join(tmpDir, "META-INF", "container.xml"), []byte(containerXML), 0644)
	if err != nil {
		return fmt.Errorf("failed to create container.xml: %w", err)
	}

	// Create content.opf
	contentOPF := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="BookId">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Translated Book</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="BookId">translated-book-id</dc:identifier>
  </metadata>
  <manifest>
    <item id="content" href="content.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="content"/>
  </spine>
</package>`
	err = os.WriteFile(filepath.Join(tmpDir, "OEBPS", "content.opf"), []byte(contentOPF), 0644)
	if err != nil {
		return fmt.Errorf("failed to create content.opf: %w", err)
	}

	// Convert markdown to basic XHTML
	markdownText := string(content)
	xhtmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
  <title>Translated Book</title>
  <style>
    body { font-family: serif; line-height: 1.6; margin: 1em; }
    h1, h2, h3 { color: #333; }
    p { margin-bottom: 1em; }
  </style>
</head>
<body>
  <div>
%s
  </div>
</body>
</html>`, convertMarkdownToXHTML(markdownText))

	err = os.WriteFile(filepath.Join(tmpDir, "OEBPS", "content.xhtml"), []byte(xhtmlContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to create content.xhtml: %w", err)
	}

	// Create the EPUB by zipping the contents
	return createEPUBFromDirectory(tmpDir, outputPath)
}

// Helper function to create a proper EPUB file from a directory
func createEPUBFromDirectory(sourceDir, outputPath string) error {
	// Create the EPUB file
	epubFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create EPUB file: %w", err)
	}
	defer epubFile.Close()

	zipWriter := zip.NewWriter(epubFile)
	defer zipWriter.Close()

	// Add files to the ZIP in the correct order
	// First add mimetype (must be uncompressed and first)
	mimetypePath := filepath.Join(sourceDir, "mimetype")
	mimeTypeWriter, err := zipWriter.CreateHeader(&zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store, // No compression
	})
	if err != nil {
		return fmt.Errorf("failed to create mimetype entry: %w", err)
	}

	mimeTypeContent, err := os.ReadFile(mimetypePath)
	if err != nil {
		return fmt.Errorf("failed to read mimetype: %w", err)
	}

	_, err = mimeTypeWriter.Write(mimeTypeContent)
	if err != nil {
		return fmt.Errorf("failed to write mimetype: %w", err)
	}

	// Walk the directory and add all files
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and the mimetype file (already added)
		if info.IsDir() || filepath.Base(path) == "mimetype" {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", relPath, err)
		}

		// Create zip entry
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create entry for %s: %w", relPath, err)
		}

		// Write content
		_, err = writer.Write(content)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", relPath, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	return nil
}

// Convert basic markdown to simple XHTML
func convertMarkdownToXHTML(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var result strings.Builder
	inParagraph := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Headers
		if strings.HasPrefix(line, "# ") {
			if inParagraph {
				result.WriteString("</p>\n")
				inParagraph = false
			}
			result.WriteString(fmt.Sprintf("<h1>%s</h1>\n", strings.TrimPrefix(line, "# ")))
		} else if strings.HasPrefix(line, "## ") {
			if inParagraph {
				result.WriteString("</p>\n")
				inParagraph = false
			}
			result.WriteString(fmt.Sprintf("<h2>%s</h2>\n", strings.TrimPrefix(line, "## ")))
		} else if strings.HasPrefix(line, "### ") {
			if inParagraph {
				result.WriteString("</p>\n")
				inParagraph = false
			}
			result.WriteString(fmt.Sprintf("<h3>%s</h3>\n", strings.TrimPrefix(line, "### ")))
		} else if line == "" {
			// Empty line - close paragraph if open
			if inParagraph {
				result.WriteString("</p>\n")
				inParagraph = false
			}
		} else {
			// Regular text - start or continue paragraph
			if !inParagraph {
				result.WriteString("<p>")
				inParagraph = true
			} else {
				result.WriteString(" ")
			}
			result.WriteString(line)
		}
	}

	// Close any open paragraph
	if inParagraph {
		result.WriteString("</p>\n")
	}

	return result.String()
}