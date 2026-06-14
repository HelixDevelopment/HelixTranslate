package llm

import (
	"strings"
	"testing"
)

// TestSplitText_LosslessRoundTrip is a reproduce-first guard (§11.4.115) for a
// real structural data-loss bug: splitText dropped the "\n\n" paragraph
// separator at chunk boundaries (and inconsistently around oversized
// paragraphs), so the per-chunk-translation reassembly `strings.Join(chunks,
// "")` glued the last paragraph of one chunk to the first of the next —
// paragraph breaks vanished from translated large chapters.
//
// The invariant the fix establishes: splitText tiles the input losslessly, i.e.
// strings.Join(splitText(text), "") == text. That makes the existing
// Join("")-based reassembly correct by construction without risking spurious
// mid-paragraph breaks.
func TestSplitText_LosslessRoundTrip(t *testing.T) {
	lt := &LLMTranslator{}

	cases := map[string]string{
		// Two large paragraphs that force a chunk boundary exactly at the "\n\n".
		"two_large_paragraphs": strings.Repeat("A", 15000) + "\n\n" + strings.Repeat("B", 15000),
		// Many small paragraphs spanning multiple chunks.
		"many_small_paragraphs": func() string {
			var b strings.Builder
			for i := 0; i < 60; i++ {
				if i > 0 {
					b.WriteString("\n\n")
				}
				b.WriteString(strings.Repeat("para ", 200)) // ~1000 bytes each
			}
			return b.String()
		}(),
		// A single oversized paragraph split by sentences (no "\n\n").
		"one_oversized_paragraph": strings.Repeat("This is a sentence. ", 1500), // ~30000 bytes
		// Mixed: small para, huge para, small para.
		"mixed_sizes": "Intro paragraph." + "\n\n" +
			strings.Repeat("Big sentence here. ", 1500) + "\n\n" +
			"Closing paragraph.",
		// Consecutive blank paragraphs must be preserved.
		"blank_paragraphs": strings.Repeat("X", 12000) + "\n\n\n\n" + strings.Repeat("Y", 12000),
	}

	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			chunks := lt.splitText(text)
			rejoined := strings.Join(chunks, "")
			if rejoined != text {
				t.Errorf("splitText round-trip is LOSSY for %q: len(in)=%d len(out)=%d; "+
					"reassembling translated chunks with Join(\"\") would lose/alter content",
					name, len(text), len(rejoined))
				// Surface the first divergence for forensic clarity.
				min := len(text)
				if len(rejoined) < min {
					min = len(rejoined)
				}
				for i := 0; i < min; i++ {
					if text[i] != rejoined[i] {
						lo := i - 20
						if lo < 0 {
							lo = 0
						}
						t.Errorf("first divergence at byte %d: in=%q out=%q", i, text[lo:i+5], rejoined[lo:i+5])
						break
					}
				}
			}
		})
	}
}
