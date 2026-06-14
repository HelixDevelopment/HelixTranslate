package ebook

import (
	"os"
	"path/filepath"
	"testing"
)

// benchFB2Document is a small but realistic multi-chapter FB2 document used to
// benchmark the real FB2 parse path (XML decode + chapter/section extraction +
// metadata). It preserves the FB2 namespace and uses Cyrillic prose so the
// parser exercises UTF-8 handling like a production document.
const benchFB2Document = `<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <author><first-name>Петар</first-name><last-name>Петровић</last-name></author>
      <book-title>Бенчмарк књига</book-title>
      <lang>sr</lang>
    </title-info>
  </description>
  <body>
    <section>
      <title><p>Прво поглавље</p></title>
      <p>Ово је први параграф првог поглавља са довољно текста да парсер има шта да обради.</p>
      <p>Други параграф наставља причу о књижевности и преводу између писама.</p>
      <p>Трећи параграф додаје још садржаја ради реалистичног мерења перформанси.</p>
    </section>
    <section>
      <title><p>Друго поглавље</p></title>
      <p>Друго поглавље доноси нове ликове и нове сцене у овој причи.</p>
      <p>Завршни параграф затвара поглавље јасном поентом.</p>
    </section>
    <section>
      <title><p>Треће поглавље</p></title>
      <p>Треће поглавље служи као епилог целе приповести.</p>
    </section>
  </body>
</FictionBook>`

func writeFB2Temp(tb testing.TB) string {
	tb.Helper()
	dir := tb.TempDir()
	path := filepath.Join(dir, "bench.fb2")
	if err := os.WriteFile(path, []byte(benchFB2Document), 0o600); err != nil {
		tb.Fatalf("write fb2: %v", err)
	}
	return path
}

// BenchmarkFB2ParserParse benchmarks the real FB2 parse of a constructed small
// multi-chapter document.
func BenchmarkFB2ParserParse(b *testing.B) {
	path := writeFB2Temp(b)
	parser := NewFB2Parser()
	b.ReportAllocs()
	b.SetBytes(int64(len(benchFB2Document)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		book, err := parser.Parse(path)
		if err != nil {
			b.Fatal(err)
		}
		if book == nil {
			b.Fatal("nil book")
		}
	}
}

// BenchmarkUniversalParserParseFB2 benchmarks the full pipeline (format
// detection + dispatch + parse) through the UniversalParser, the entry point the
// translator actually uses.
func BenchmarkUniversalParserParseFB2(b *testing.B) {
	path := writeFB2Temp(b)
	up := NewUniversalParser()
	b.ReportAllocs()
	b.SetBytes(int64(len(benchFB2Document)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		book, err := up.Parse(path)
		if err != nil {
			b.Fatal(err)
		}
		if book == nil {
			b.Fatal("nil book")
		}
	}
}

// BenchmarkBookExtractText benchmarks text extraction over the parsed book — the
// hot path feeding the translation stage.
func BenchmarkBookExtractText(b *testing.B) {
	path := writeFB2Temp(b)
	parser := NewFB2Parser()
	book, err := parser.Parse(path)
	if err != nil {
		b.Fatalf("setup parse: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = book.ExtractText()
	}
}

// TestFB2ParserChaos_MalformedInput is the §11.4.85 chaos test for the FB2
// parser: malformed / truncated / empty / wrong-content inputs MUST fail
// gracefully with an error and NEVER panic. A parser that panics on any of
// these crashes this test. For inputs that are valid-enough to parse, we assert
// it returns a non-nil book without crashing.
func TestFB2ParserChaos_MalformedInput(t *testing.T) {
	parser := NewFB2Parser()

	chaos := map[string]string{
		"empty":              ``,
		"truncated-mid-tag":  `<?xml version="1.0"?><FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"><body><sec`,
		"not-xml":            `this is just plain prose, not xml at all`,
		"xml-but-not-fb2":    `<?xml version="1.0"?><root><child>data</child></root>`,
		"unclosed-tags":      `<?xml version="1.0"?><FictionBook><body><section><p>text`,
		"binary-garbage":     "\x00\x01\x02\xff\xfe\x80",
		"only-decl":          `<?xml version="1.0" encoding="UTF-8"?>`,
		"deeply-nested-open": `<?xml version="1.0"?><FictionBook><body>` + repeatStr(`<section>`, 200),
	}

	for name, body := range chaos {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "chaos.fb2")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}

			// The contract under chaos: no panic. Either a clean error OR a
			// non-nil book. A nil book with nil error would be a silent failure.
			book, err := parser.Parse(path)
			if err == nil && book == nil {
				t.Fatalf("%s: Parse returned (nil, nil) — silent failure, expected error or book", name)
			}
		})
	}
}

func repeatStr(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
