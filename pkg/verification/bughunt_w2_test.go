package verification

import (
	"math"
	"strings"
	"testing"
)

// RED: a note with CONTENT then EXAMPLES but NO IMPLICATIONS marker must NOT be
// dropped. parseNote only flushes contentBuilder->note.Content on the
// IMPLICATIONS: marker or when currentField=="content" at end-of-block. If the
// block ends in the "examples" field, note.Content stays "" and the note is
// silently rejected by the required-fields check.
func TestParseNote_ContentThenExamplesNoImplications(t *testing.T) {
	nt := NewNoteTaker(nil, "prov")
	block := strings.Join([]string{
		"NOTE: [TONE]",
		"IMPORTANCE: [high]",
		"TITLE: Somber mood",
		"CONTENT: The scene is heavy with grief.",
		"EXAMPLES:",
		"The rain fell.",
		"Nobody spoke.",
	}, "\n")

	note := nt.parseNote(block, 1, "sec", "Chapter 1")
	if note == nil {
		t.Fatal("parseNote dropped a valid NOTE+TITLE+CONTENT+EXAMPLES note that omits IMPLICATIONS")
	}
	if !strings.Contains(note.Content, "The scene is heavy with grief.") {
		t.Errorf("Content not captured: %q", note.Content)
	}
	if len(note.Examples) != 2 {
		t.Errorf("Examples count = %d, want 2: %#v", len(note.Examples), note.Examples)
	}
}

// RED: buildConsensus divides maxAgreement by count without guarding count==0.
// With zero verifications (empty providers / all-errored), count==0 and
// result.Confidence = 0.0/0.0 = NaN, which then poisons report.AverageConfidence
// and any downstream score math.
func TestBuildConsensus_ZeroVerificationsNoNaN(t *testing.T) {
	bp := &BookPolisher{config: PolishingConfig{MinConsensus: 2}}
	res := bp.buildConsensus("sec", "Chapter 1", "orig", "trans", nil)
	if res == nil {
		t.Fatal("buildConsensus returned nil")
	}
	if math.IsNaN(res.Confidence) {
		t.Errorf("Confidence is NaN for zero verifications: %v", res.Confidence)
	}
	if res.Confidence != 0.0 {
		t.Errorf("Confidence = %v, want 0.0 for zero verifications", res.Confidence)
	}
}

// RED: generateFinalReport uses the LAST pass's Report pointer as finalReport,
// then zeroes finalReport.TotalSections (which aliases the last pass's
// TotalSections) BEFORE summing all passes. The summation loop therefore reads
// the last pass's count as 0 — the combined total under-counts by the last
// pass's section count.
func TestGenerateFinalReport_SumsAllPassSections(t *testing.T) {
	mpp := &MultiPassPolisher{}
	r1 := &PolishingReport{TotalSections: 3}
	r2 := &PolishingReport{TotalSections: 5}
	result := &MultiPassResult{
		PassResults: []*PassResult{
			{Report: r1},
			{Report: r2},
		},
	}
	final := mpp.generateFinalReport(result)
	if final == nil {
		t.Fatal("generateFinalReport returned nil")
	}
	if final.TotalSections != 8 {
		t.Errorf("combined TotalSections = %d, want 8 (3+5)", final.TotalSections)
	}
}
