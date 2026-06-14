package fb2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/logger"
)

// TestConvertToMarkdown_PoemStanzaTitleSubtitle_NotLost is a reproduce-first RED
// test for FB2->Markdown data loss inside poems. The FB2 schema allows a <stanza>
// to carry its own <title> and <subtitle> (both mixed content). processPoem only
// emits the verse <v> lines and silently drops every stanza <title> and
// <subtitle> — translated heading text inside a poem never reaches the reader.
//
// Cyrillic content so the loss is also exercised on multi-byte text.
func TestConvertToMarkdown_PoemStanzaTitleSubtitle_NotLost(t *testing.T) {
	testLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.ERROR,
		Format: logger.FORMAT_TEXT,
	})
	converter := NewMarkdownConverter(testLogger)

	fb2 := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
	<description>
		<title-info>
			<book-title>Поэма</book-title>
			<lang>ru</lang>
		</title-info>
	</description>
	<body>
		<section>
			<title><p>Глава</p></title>
			<poem>
				<title><p>Название поэмы</p></title>
				<stanza>
					<title><p>ЗаголовокСтрофы</p></title>
					<subtitle>ПодзаголовокСтрофы</subtitle>
					<v>Первая строка стиха</v>
					<v>Вторая строка стиха</v>
				</stanza>
			</poem>
		</section>
	</body>
</FictionBook>`

	dir := t.TempDir()
	in := filepath.Join(dir, "poem.fb2")
	out := filepath.Join(dir, "poem.md")
	if err := os.WriteFile(in, []byte(fb2), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := converter.ConvertToMarkdown(in, out); err != nil {
		t.Fatalf("ConvertToMarkdown: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	md := string(data)

	// Sanity: verses survive (proves the poem itself was processed).
	if !strings.Contains(md, "Первая строка стиха") {
		t.Fatalf("verse text missing — poem not processed at all:\n%s", md)
	}
	// The actual defect: stanza title + subtitle dropped.
	if !strings.Contains(md, "ЗаголовокСтрофы") {
		t.Errorf("stanza <title> text dropped from markdown (data loss):\n%s", md)
	}
	if !strings.Contains(md, "ПодзаголовокСтрофы") {
		t.Errorf("stanza <subtitle> text dropped from markdown (data loss):\n%s", md)
	}
}
