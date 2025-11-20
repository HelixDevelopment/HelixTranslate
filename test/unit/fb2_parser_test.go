package unit

import (
	"digital.vasic.translator/pkg/fb2"
	"strings"
	"testing"
)

func TestFB2Parser(t *testing.T) {
	parser := fb2.NewParser()

	t.Run("ParseSimpleFB2", func(t *testing.T) {
		fb2Content := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <genre>fiction</genre>
      <author>
        <first-name>Иван</first-name>
        <last-name>Петров</last-name>
      </author>
      <book-title>Тестовая книга</book-title>
      <lang>ru</lang>
    </title-info>
  </description>
  <body>
    <section>
      <p>Тестовый параграф.</p>
    </section>
  </body>
</FictionBook>`

		reader := strings.NewReader(fb2Content)
		book, err := parser.ParseReader(reader)

		if err != nil {
			t.Fatalf("Failed to parse FB2: %v", err)
		}

		if book == nil {
			t.Fatal("Book is nil")
		}

		if book.GetTitle() != "Тестовая книга" {
			t.Errorf("Expected title 'Тестовая книга', got '%s'", book.GetTitle())
		}

		if book.GetLanguage() != "ru" {
			t.Errorf("Expected language 'ru', got '%s'", book.GetLanguage())
		}

		if len(book.Body) == 0 {
			t.Fatal("Expected at least one body section")
		}

		if len(book.Body[0].Section) == 0 {
			t.Fatal("Expected at least one section")
		}
	})

	t.Run("SetAndGetTitle", func(t *testing.T) {
		book := &fb2.FictionBook{}
		book.SetTitle("Test Title")

		if book.GetTitle() != "Test Title" {
			t.Errorf("Expected 'Test Title', got '%s'", book.GetTitle())
		}
	})

	t.Run("SetAndGetLanguage", func(t *testing.T) {
		book := &fb2.FictionBook{}
		book.SetLanguage("sr")

		if book.GetLanguage() != "sr" {
			t.Errorf("Expected 'sr', got '%s'", book.GetLanguage())
		}
	})
}
