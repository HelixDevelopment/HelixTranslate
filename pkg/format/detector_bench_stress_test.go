package format

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// representative content for each detection path. These exercise the real
// DetectFile pipeline end-to-end: magic-byte detection (PDF), content-based FB2
// detection (no distinctive magic prefix), HTML content detection, and the
// plain-text fallback.
var (
	pdfContent = append([]byte("%PDF-1.7\n"), make([]byte, 480)...)

	fb2Content = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description><title-info><book-title>Бенчмарк</book-title></title-info></description>
  <body><section><p>Текст параграфа за детекцију формата.</p></section></body>
</FictionBook>`)

	htmlContent = []byte(`<!DOCTYPE html>
<html lang="sr"><head><title>Doc</title></head>
<body><h1>Naslov</h1><p>Pasus teksta za detekciju.</p></body></html>`)

	txtContent = []byte("Ово је обичан текстуални документ. " +
		"Садржи више реченица да би детектор препознао plain text. " +
		"Plain text fallback path. Line two.\nLine three.\n")
)

// writeTempWithContent writes content to a temp file with the given extension
// and returns its path; the file is registered for cleanup via tb.Cleanup.
func writeTempWithContent(tb testing.TB, ext string, content []byte) string {
	tb.Helper()
	dir := tb.TempDir()
	path := filepath.Join(dir, "doc"+ext)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		tb.Fatalf("write temp: %v", err)
	}
	return path
}

func BenchmarkDetectFilePDF(b *testing.B) {
	path := writeTempWithContent(b, ".pdf", pdfContent)
	d := NewDetector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.DetectFile(path); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDetectFileFB2ByContent benchmarks the content-detection path: an FB2
// document carrying a generic .xml extension so detection MUST inspect content.
func BenchmarkDetectFileFB2ByContent(b *testing.B) {
	path := writeTempWithContent(b, ".xml", fb2Content)
	d := NewDetector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.DetectFile(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDetectFileHTMLByContent(b *testing.B) {
	// Use a generic extension so the content path (detectByContent) runs.
	path := writeTempWithContent(b, ".dat", htmlContent)
	d := NewDetector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.DetectFile(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDetectFileTXTByContent(b *testing.B) {
	path := writeTempWithContent(b, ".dat", txtContent)
	d := NewDetector()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.DetectFile(path); err != nil {
			b.Fatal(err)
		}
	}
}

// TestDetectorConcurrent_StressCorrectness is the §11.4.85 concurrent-contention
// stress test for the format detector. One shared *Detector (stateless) is hit
// by many goroutines, each detecting a known input and asserting the EXACT
// expected Format. It would FAIL if the detector returned a wrong/empty format
// under load or if a data race corrupted results. Run with -race for the
// race-clean evidence.
func TestDetectorConcurrent_StressCorrectness(t *testing.T) {
	d := NewDetector()

	cases := []struct {
		name string
		ext  string
		body []byte
		want Format
	}{
		{"pdf-magic", ".pdf", pdfContent, FormatPDF},
		{"fb2-content", ".xml", fb2Content, FormatFB2},
		{"html-content", ".dat", htmlContent, FormatHTML},
		{"txt-content", ".dat", txtContent, FormatTXT},
	}

	// Pre-create the files (single owner of each temp file: read-only detection).
	paths := make([]string, len(cases))
	for i, tc := range cases {
		paths[i] = writeTempWithContent(t, tc.ext, tc.body)
	}

	// Sanity: prove each golden actually holds single-threaded before stressing.
	for i, tc := range cases {
		got, err := d.DetectFile(paths[i])
		if err != nil {
			t.Fatalf("%s: golden detect error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: golden detect = %s, want %s", tc.name, got, tc.want)
		}
	}

	const goroutines = 24
	const iterations = 200

	var wg sync.WaitGroup
	errCh := make(chan string, goroutines)
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				tc := cases[(g+i)%len(cases)]
				p := paths[(g+i)%len(cases)]
				got, err := d.DetectFile(p)
				if err != nil {
					errCh <- tc.name + ": error " + err.Error()
					return
				}
				if got != tc.want {
					errCh <- tc.name + ": got " + string(got) + " want " + string(tc.want)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errCh)

	if msg, ok := <-errCh; ok {
		t.Fatalf("concurrent DetectFile produced incorrect result under load: %s", msg)
	}
}

// TestDetectorChaos_MalformedInput is the §11.4.85 chaos test: malformed /
// truncated / empty / binary-garbage inputs MUST be handled gracefully — a
// classified Format and no panic. DetectFile recovering from any of these via a
// panic would crash this test.
func TestDetectorChaos_MalformedInput(t *testing.T) {
	d := NewDetector()

	chaosInputs := map[string][]byte{
		"empty":               {},
		"single-byte":         {0x00},
		"truncated-pdf":       []byte("%PD"),
		"truncated-fb2-xml":   []byte("<?xml version=\"1.0\"?><Fiction"),
		"binary-garbage":      {0x00, 0xFF, 0x01, 0xFE, 0x80, 0x7F, 0x00, 0x00},
		"zip-magic-truncated": []byte("PK\x03\x04"),
		"lone-xml-decl":       []byte("<?xml version=\"1.0\"?>"),
		"nul-bytes":           make([]byte, 256), // all zero
	}

	for name, body := range chaosInputs {
		t.Run(name, func(t *testing.T) {
			path := writeTempWithContent(t, ".bin", body)
			// Must not panic; must return a Format (any) and an error value we can read.
			got, err := d.DetectFile(path)
			// We assert it produced a usable verdict: either a known Format value
			// or FormatUnknown, and did not crash. err may be non-nil for ZIP-magic
			// inputs that fail to open as an archive — that is graceful handling.
			_ = err
			if got == "" {
				t.Fatalf("%s: DetectFile returned empty Format (expected a classified value)", name)
			}
		})
	}
}
