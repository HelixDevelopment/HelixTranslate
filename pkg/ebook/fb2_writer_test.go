package ebook

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFB2WriterWrite(t *testing.T) {
	book := &Book{
		Metadata: Metadata{
			Title:       "Test Book",
			Authors:     []string{"John Doe"},
			Description: "A test book for FB2 writing",
			Language:    "en",
		},
		Chapters: []Chapter{
			{
				Title: "Chapter 1",
				Sections: []Section{
					{Title: "Section 1.1", Content: "This is the first paragraph."},
					{Title: "Section 1.2", Content: "This is the second paragraph."},
				},
			},
			{
				Title: "Chapter 2",
				Sections: []Section{
					{Title: "Section 2.1", Content: "Another chapter content."},
				},
			},
		},
	}

	writer := NewFB2Writer()
	tmpFile := "test_output.fb2"
	defer os.Remove(tmpFile)

	err := writer.Write(book, tmpFile)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	// Verify XML structure
	assert.True(t, strings.HasPrefix(string(content), `<?xml version="1.0" encoding="UTF-8"?>`))
	assert.Contains(t, string(content), "<FictionBook")
	assert.Contains(t, string(content), "Test Book")
	assert.Contains(t, string(content), "<first-name>John</first-name>")
	assert.Contains(t, string(content), "<last-name>Doe</last-name>")
	assert.Contains(t, string(content), "Chapter 1")
	assert.Contains(t, string(content), "Chapter 2")
	assert.Contains(t, string(content), "This is the first paragraph.")
	assert.Contains(t, string(content), "HelixTranslate")

	// Verify valid XML
	var fb fictionBook
	err = xml.Unmarshal(content, &fb)
	require.NoError(t, err)
	assert.Equal(t, "Test Book", fb.Description.TitleInfo.BookTitle)
	assert.Equal(t, "en", fb.Description.TitleInfo.Lang)
	assert.Len(t, fb.Body.Sections, 2)
	assert.Equal(t, "Chapter 1", fb.Body.Sections[0].Title.P)
	assert.Equal(t, "Chapter 2", fb.Body.Sections[1].Title.P)
}

func TestFB2WriterXMLEscape(t *testing.T) {
	book := &Book{
		Metadata: Metadata{
			Title:    "Book with <special> & \"chars\"",
			Authors:  []string{"Author"},
			Language: "en",
		},
		Chapters: []Chapter{
			{
				Title: "Chapter",
				Sections: []Section{
					{Content: "Text with <tag> & \"quotes\"."},
				},
			},
		},
	}

	writer := NewFB2Writer()
	tmpFile := "test_escape.fb2"
	defer os.Remove(tmpFile)

	err := writer.Write(book, tmpFile)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	// Verify special chars are escaped by XML encoder
	assert.Contains(t, string(content), "&lt;special&gt;")
	assert.Contains(t, string(content), "&amp;")
	assert.Contains(t, string(content), `&#34;chars&#34;`)
	assert.Contains(t, string(content), "&lt;tag&gt;")
}

func TestFB2WriterMultipleAuthors(t *testing.T) {
	book := &Book{
		Metadata: Metadata{
			Title:    "Multi-Author Book",
			Authors:  []string{"Alice Smith", "Bob Jones"},
			Language: "en",
		},
		Chapters: []Chapter{
			{Title: "Ch1", Sections: []Section{{Content: "Text"}}},
		},
	}

	writer := NewFB2Writer()
	tmpFile := "test_authors.fb2"
	defer os.Remove(tmpFile)

	err := writer.Write(book, tmpFile)
	require.NoError(t, err)

	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "Alice")
	assert.Contains(t, string(content), "Smith")
	assert.Contains(t, string(content), "Bob")
	assert.Contains(t, string(content), "Jones")
}

func TestFB2WriterInvalidPath(t *testing.T) {
	book := &Book{
		Metadata: Metadata{Title: "Test", Authors: []string{"A"}, Language: "en"},
		Chapters: []Chapter{{Title: "Ch1", Sections: []Section{{Content: "Text"}}}},
	}

	writer := NewFB2Writer()
	// Try to write to a directory that doesn't exist
	err := writer.Write(book, "/nonexistent/dir/book.fb2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write FB2")
}

func TestEscapeFB2Text(t *testing.T) {
	// Double newlines should be handled
	result := escapeFB2Text("Line 1\n\nLine 2")
	assert.Contains(t, result, "</p>")
	assert.Contains(t, result, "<p>")
}
