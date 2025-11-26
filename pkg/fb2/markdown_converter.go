package fb2

import (
	"fmt"
	"os"
	"strings"

	"digital.vasic.translator/pkg/logger"
)

// MarkdownConverter handles FB2 to Markdown conversion
type MarkdownConverter struct {
	parser *Parser
	logger logger.Logger
}

// NewMarkdownConverter creates a new FB2 to Markdown converter
func NewMarkdownConverter(logger logger.Logger) *MarkdownConverter {
	return &MarkdownConverter{
		parser: NewParser(),
		logger: logger,
	}
}

// ConvertToMarkdown converts an FB2 file to Markdown format
func (c *MarkdownConverter) ConvertToMarkdown(inputPath, outputPath string) error {
	c.logger.Info("Converting FB2 to Markdown", map[string]interface{}{
		"input_file":  inputPath,
		"output_file": outputPath,
	})

	// Parse the FB2 file
	fb, err := c.parser.Parse(inputPath)
	if err != nil {
		return fmt.Errorf("failed to parse FB2 file: %w", err)
	}

	// Build markdown content
	var markdown strings.Builder

	// Add title
	if fb.Description.TitleInfo.BookTitle != "" {
		markdown.WriteString("# ")
		markdown.WriteString(fb.Description.TitleInfo.BookTitle)
		markdown.WriteString("\n\n")
	}

	// Add authors
	if len(fb.Description.TitleInfo.Author) > 0 {
		markdown.WriteString("## Authors\n\n")
		for _, author := range fb.Description.TitleInfo.Author {
			authorName := formatAuthorName(author)
			if authorName != "" {
				markdown.WriteString("- ")
				markdown.WriteString(authorName)
				markdown.WriteString("\n")
			}
		}
		markdown.WriteString("\n")
	}

	// Add annotation if available
	if fb.Description.TitleInfo.Annotation.Paragraphs != nil {
		markdown.WriteString("## Annotation\n\n")
		for _, para := range fb.Description.TitleInfo.Annotation.Paragraphs {
			text := extractTextFromParagraph(para)
			if text != "" {
				markdown.WriteString(text)
				markdown.WriteString("\n\n")
			}
		}
	}

	// Process the body content
	if len(fb.Body) > 0 {
		for _, body := range fb.Body {
			if len(body.Section) > 0 {
				c.processSections(&markdown, body.Section, 2)
			}
		}
	}

	// Write markdown to file
	if err := os.WriteFile(outputPath, []byte(markdown.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	c.logger.Info("FB2 converted to Markdown successfully", map[string]interface{}{
		"input_file":   inputPath,
		"output_file":  outputPath,
		"chars_count": markdown.Len(),
	})

	return nil
}

// processSections recursively processes sections and subsections
func (c *MarkdownConverter) processSections(markdown *strings.Builder, sections []Section, level int) {
	for _, section := range sections {
		// Process section title if available
		if section.Title.Paragraphs != nil && len(section.Title.Paragraphs) > 0 {
			title := extractTextFromParagraph(section.Title.Paragraphs[0])
			if title != "" {
				markdown.WriteString(strings.Repeat("#", level))
				markdown.WriteString(" ")
				markdown.WriteString(title)
				markdown.WriteString("\n\n")
			}
		}

		// Process paragraphs
		for _, para := range section.Paragraph {
			text := extractTextFromParagraph(para)
			if text != "" {
				markdown.WriteString(text)
				markdown.WriteString("\n\n")
			}
		}

		// Process epigraphs
		for _, epigraph := range section.Epigraph {
			c.processEpigraph(markdown, epigraph)
		}

		// Process subtitles
		for _, subtitle := range section.Subtitle {
			if subtitle != "" {
				markdown.WriteString("### ")
				markdown.WriteString(subtitle)
				markdown.WriteString("\n\n")
			}
		}

		// Process poems
		for _, poem := range section.Poem {
			c.processPoem(markdown, poem)
		}

		// Process citations
		for _, cite := range section.Cite {
			c.processCite(markdown, cite)
		}

		// Process empty lines
		for i := 0; i < len(section.EmptyLine); i++ {
			markdown.WriteString("\n")
		}

		// Process nested sections recursively
		if len(section.Section) > 0 {
			c.processSections(markdown, section.Section, level+1)
		}
	}
}

// processEpigraph processes an epigraph
func (c *MarkdownConverter) processEpigraph(markdown *strings.Builder, epigraph Epigraph) {
	markdown.WriteString("> ")
	
	// Process epigraph paragraphs
	for i, para := range epigraph.Paragraph {
		text := extractTextFromParagraph(para)
		if text != "" {
			if i > 0 {
				markdown.WriteString("> ")
			}
			markdown.WriteString(text)
			markdown.WriteString("\n")
		}
	}
	
	// Process text author if available
	for _, textAuthor := range epigraph.TextAuthor {
		if textAuthor != "" {
			markdown.WriteString("> \u2014 ")
			markdown.WriteString(textAuthor)
			markdown.WriteString("\n")
		}
	}
	
	markdown.WriteString("\n")
}

// processPoem processes a poem
func (c *MarkdownConverter) processPoem(markdown *strings.Builder, poem Poem) {
	// Process poem title if available
	if poem.Title.Paragraphs != nil && len(poem.Title.Paragraphs) > 0 {
		title := extractTextFromParagraph(poem.Title.Paragraphs[0])
		if title != "" {
			markdown.WriteString("### ")
			markdown.WriteString(title)
			markdown.WriteString("\n\n")
		}
	}
	
	markdown.WriteString("\n")
	
	// Process stanzas
	for _, stanza := range poem.Stanza {
		for _, verse := range stanza.V {
			if verse.Text != "" {
				markdown.WriteString("    ")
				markdown.WriteString(verse.Text)
				markdown.WriteString("\n")
			}
		}
		markdown.WriteString("\n")
	}
}

// processCite processes a citation
func (c *MarkdownConverter) processCite(markdown *strings.Builder, cite Cite) {
	markdown.WriteString("> ")
	
	// Process cite paragraphs
	for i, para := range cite.Paragraph {
		text := extractTextFromParagraph(para)
		if text != "" {
			if i > 0 {
				markdown.WriteString("> ")
			}
			markdown.WriteString(text)
			markdown.WriteString("\n")
		}
	}
	
	// Process subtitles if available
	for _, subtitle := range cite.Subtitle {
		if subtitle != "" {
			markdown.WriteString("> \u2014 ")
			markdown.WriteString(subtitle)
			markdown.WriteString("\n")
		}
	}
	
	markdown.WriteString("\n")
}

// formatAuthorName formats an author's name
func formatAuthorName(author Author) string {
	var parts []string
	if author.FirstName != "" {
		parts = append(parts, author.FirstName)
	}
	if author.LastName != "" {
		parts = append(parts, author.LastName)
	}
	if len(parts) == 0 && author.Nickname != "" {
		return author.Nickname
	}
	return strings.Join(parts, " ")
}

// extractTextFromParagraph extracts clean text from a paragraph
func extractTextFromParagraph(para Paragraph) string {
	if para.Text != "" {
		return para.Text
	}
	
	// Extract text from mixed content
	var textParts []string
	for _, content := range para.Content {
		if str, ok := content.(string); ok {
			textParts = append(textParts, str)
		}
	}
	
	return strings.Join(textParts, "")
}