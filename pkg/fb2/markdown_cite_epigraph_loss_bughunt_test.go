package fb2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/logger"
)

func convertFB2(t *testing.T, fb2 string) string {
	t.Helper()
	testLogger := logger.NewLogger(logger.LoggerConfig{
		Level:  logger.ERROR,
		Format: logger.FORMAT_TEXT,
	})
	converter := NewMarkdownConverter(testLogger)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.fb2")
	out := filepath.Join(dir, "out.md")
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
	return string(data)
}

// TestConvertToMarkdown_CitePoem_NotLost: a <poem> nested inside a <cite> is part
// of the FB2 schema (cite content: p | subtitle | empty-line | poem | text-author).
// processCite emits paragraphs + subtitles but never descends into cite.Poem, so a
// quoted poem's verses are silently dropped from the markdown — data loss.
func TestConvertToMarkdown_CitePoem_NotLost(t *testing.T) {
	fb2 := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
	<description><title-info><book-title>Кн</book-title><lang>ru</lang></title-info></description>
	<body>
		<section>
			<title><p>Глава</p></title>
			<cite>
				<p>Введение к цитате.</p>
				<poem><stanza><v>СтрокаЦитируемогоСтиха</v></stanza></poem>
			</cite>
		</section>
	</body>
</FictionBook>`
	md := convertFB2(t, fb2)
	if !strings.Contains(md, "Введение к цитате.") {
		t.Fatalf("cite paragraph missing — cite not processed:\n%s", md)
	}
	if !strings.Contains(md, "СтрокаЦитируемогоСтиха") {
		t.Errorf("poem nested in <cite> dropped from markdown (data loss):\n%s", md)
	}
}

// TestConvertToMarkdown_EpigraphPoem_NotLost: a <poem> nested inside an <epigraph>
// is part of the FB2 schema (epigraph content: p | poem | cite | text-author).
// processEpigraph emits paragraphs + text-author but never descends into
// epigraph.Poem (nor epigraph.Cite), so a poetic epigraph's verses are silently
// dropped from the markdown — data loss.
func TestConvertToMarkdown_EpigraphPoem_NotLost(t *testing.T) {
	fb2 := `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
	<description><title-info><book-title>Кн</book-title><lang>ru</lang></title-info></description>
	<body>
		<section>
			<title><p>Глава</p></title>
			<epigraph>
				<poem><stanza><v>СтрокаЭпиграфа</v></stanza></poem>
				<text-author>Автор</text-author>
			</epigraph>
			<p>Текст главы.</p>
		</section>
	</body>
</FictionBook>`
	md := convertFB2(t, fb2)
	if !strings.Contains(md, "Текст главы.") {
		t.Fatalf("body text missing:\n%s", md)
	}
	if !strings.Contains(md, "Автор") {
		t.Fatalf("epigraph text-author missing — epigraph not processed:\n%s", md)
	}
	if !strings.Contains(md, "СтрокаЭпиграфа") {
		t.Errorf("poem nested in <epigraph> dropped from markdown (data loss):\n%s", md)
	}
}
