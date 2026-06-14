package unit

import (
	"digital.vasic.translator/pkg/format"
	"testing"
)

func TestFormatDetector(t *testing.T) {
	detector := format.NewDetector()

	t.Run("DetectByExtension", func(t *testing.T) {
		tests := []struct {
			ext      string
			expected format.Format
		}{
			{".fb2", format.FormatFB2},
			{".epub", format.FormatEPUB},
			{".txt", format.FormatTXT},
			{".html", format.FormatHTML},
			{".pdf", format.FormatPDF},
		}

		for _, tt := range tests {
			// We can't test DetectFile directly without files,
			// but we can test the parsing logic
			result := format.ParseFormat(tt.ext[1:])
			if result != tt.expected {
				t.Errorf("ParseFormat(%s) = %v, want %v", tt.ext, result, tt.expected)
			}
		}
	})

	t.Run("IsSupported", func(t *testing.T) {
		supportedFormats := []format.Format{
			format.FormatFB2,
			format.FormatEPUB,
			format.FormatTXT,
			format.FormatHTML,
			format.FormatDOCX, // backed by the stdlib OOXML parser (real-translation proven)
			format.FormatPDF,  // backed by the MIT ledongthuc/pdf extractor (real-translation proven)
		}

		for _, fmt := range supportedFormats {
			if !detector.IsSupported(fmt) {
				t.Errorf("Format %s should be supported", fmt)
			}
		}

		// MOBI has no parser implementation, so it is genuinely unsupported.
		// (PDF + DOCX moved to supportedFormats above once their real extractors
		// landed — §11.4.120 gate reconciliation, not a fake-pass.)
		unsupportedFormats := []format.Format{
			format.FormatMOBI,
		}

		for _, fmt := range unsupportedFormats {
			if detector.IsSupported(fmt) {
				t.Errorf("Format %s should not be supported yet", fmt)
			}
		}
	})

	t.Run("ParseFormat", func(t *testing.T) {
		tests := []struct {
			input    string
			expected format.Format
		}{
			{"epub", format.FormatEPUB},
			{"EPUB", format.FormatEPUB},
			{"fb2", format.FormatFB2},
			{"txt", format.FormatTXT},
			{"text", format.FormatTXT},
			{"html", format.FormatHTML},
			{"unknown", format.FormatUnknown},
		}

		for _, tt := range tests {
			result := format.ParseFormat(tt.input)
			if result != tt.expected {
				t.Errorf("ParseFormat(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		}
	})
}
