package format

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRegression_FB2ContentBeatsGenericExtension is the permanent guard for the
// detection defect where an FB2 document carrying a generic/wrong extension was
// classified by extension as plain text (then mangled by the wrong parser).
// FB2 content must win over a generic extension; genuine plain text must NOT be
// reclassified.
//
// Mutation guard: remove the isFB2Content check in DetectFile and the FB2-with-
// .txt / .dat / no-ext cases FAIL (revert to txt/unknown).
func TestRegression_FB2ContentBeatsGenericExtension(t *testing.T) {
	d := NewDetector()
	dir := t.TempDir()
	fb2 := `<?xml version="1.0" encoding="utf-8"?>` + "\n" +
		`<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"><description/></FictionBook>`

	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	cases := []struct {
		name string
		path string
		want Format
	}{
		{"fb2 content, .txt extension", write("a.txt", []byte(fb2)), FormatFB2},
		{"fb2 content, no extension", write("noext", []byte(fb2)), FormatFB2},
		{"fb2 content, .xml extension", write("b.xml", []byte(fb2)), FormatFB2},
		{"fb2 content, UTF-8 BOM + leading ws", write("c.dat", append([]byte{0xEF, 0xBB, 0xBF, '\n', ' '}, []byte(fb2)...)), FormatFB2},
		{"fb2 content, correct .fb2 extension", write("d.fb2", []byte(fb2)), FormatFB2},
		{"genuine plain text .txt stays txt", write("plain.txt", []byte("Ordinary prose, nothing structured.")), FormatTXT},
		{"prose mentioning FictionBook stays txt", write("mention.txt", []byte("I enjoy the FictionBook format.")), FormatTXT},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := d.DetectFile(tc.path)
			if err != nil {
				t.Fatalf("DetectFile: %v", err)
			}
			if got != tc.want {
				t.Errorf("DetectFile = %s, want %s", got, tc.want)
			}
		})
	}
}
