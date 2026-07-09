package markdown

import (
	"strings"
	"testing"
)

// TestMarkdownStructurePreservation verifies that TranslateMarkdown preserves
// markdown structure (headings, lists, code blocks, bold, italic, links).
// ATM-072.
func TestMarkdownStructurePreservation(t *testing.T) {
	// Mock translate function that wraps text markers
	translateFunc := func(text string) (string, error) {
		return "[TR]" + text, nil
	}
	translator := NewMarkdownTranslator(translateFunc)

	input := `# Main Title

## Subtitle

Some paragraph text here.

- List item 1
- List item 2
- List item 3

1. Numbered item 1
2. Numbered item 2

**Bold text** and *italic text*.

` + "```" + `
code block
` + "```" + `

> Blockquote text

[Link text](http://example.com)

---

Final paragraph.`

	result, err := translator.TranslateMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify structure is preserved
	structureChecks := []struct {
		name string
		want string
	}{
		{"heading1", "# [TR]Main Title"},
		{"heading2", "## [TR]Subtitle"},
		{"list_item", "- [TR]List item 1"},
		{"numbered_item", "1. [TR]Numbered item 1"},
		{"bold", "**[TR]Bold text**"},
		{"code_block_open", "```"},
		{"code_block_content", "code block"},
		{"horizontal_rule", "---"},
	}

	for _, check := range structureChecks {
		if !strings.Contains(result, check.want) {
			t.Errorf("%s: expected %q in output\nactual: %q", check.name, check.want, result)
		}
	}

	// Italic check — format markers preserved
	if !strings.Contains(result, "*[TR]italic text*") {
		t.Errorf("italic formatting should be preserved in output\nactual: %q", result)
	}

	// Verify code blocks are NOT translated
	if strings.Contains(result, "[TR]code block") {
		t.Error("code block content should not be translated")
	}
}

// TestMarkdownFrontmatterPreserved verifies frontmatter is not translated.
func TestMarkdownFrontmatterPreserved(t *testing.T) {
	translateFunc := func(text string) (string, error) {
		return "[TR]" + text, nil
	}
	translator := NewMarkdownTranslator(translateFunc)

	input := `---
title: My Document
author: Test
---

# Content

Body text.`

	result, err := translator.TranslateMarkdown(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Frontmatter should NOT be translated
	if strings.Contains(result, "[TR]title: My Document") {
		t.Error("frontmatter should not be translated")
	}

	// Content SHOULD be translated
	if !strings.Contains(result, "# [TR]Content") {
		t.Error("heading should be translated")
	}
	if !strings.Contains(result, "[TR]Body text.") {
		t.Error("body text should be translated")
	}
}
