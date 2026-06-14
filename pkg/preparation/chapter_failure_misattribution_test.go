package preparation

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// chapterFailureMock returns a valid per-chapter analysis for every chapter
// EXCEPT a configured one, for which it returns an error (simulating an LLM
// failure / timeout on that single chapter). The returned JSON encodes the true
// chapter_num so downstream attribution can be checked.
type chapterFailureMock struct {
	re       *regexp.Regexp
	failOn   int // chapter number whose analysis fails
}

func newChapterFailureMock(failOn int) *chapterFailureMock {
	return &chapterFailureMock{re: regexp.MustCompile(`Chapter (\d+)`), failOn: failOn}
}

func (m *chapterFailureMock) Translate(ctx context.Context, text, _ string) (string, error) {
	num := 0
	if strings.Contains(text, "You are analyzing Chapter") {
		if mt := m.re.FindStringSubmatch(text); len(mt) > 1 {
			num, _ = strconv.Atoi(mt[1])
		}
	}
	if num == m.failOn {
		return "", fmt.Errorf("simulated chapter %d analysis failure", num)
	}
	return fmt.Sprintf(`{"chapter_num": %d, "summary": "SUMMARY-OF-CHAPTER-%d"}`, num, num), nil
}

func (m *chapterFailureMock) TranslateWithProgress(ctx context.Context, text, c string, _ *events.EventBus, _ string) (string, error) {
	return m.Translate(ctx, text, c)
}
func (m *chapterFailureMock) GetStats() translator.TranslationStats { return translator.TranslationStats{} }
func (m *chapterFailureMock) GetName() string                       { return "chapter-failure-mock" }

// TestGetTranslationContext_ChapterFailure_NoMisattribution proves the
// end-user-visible defect: when one chapter's analysis fails, analyzeChapters
// COMPACTS the slice (drops the failed chapter), so positional indexing in
// GetTranslationContext (ChapterAnalyses[chapterNum-1]) mis-attributes every
// surviving chapter AFTER the failed one — chapter N+1 receives chapter N+2's
// summary, and the last chapter falls out of range and gets NO context.
//
// With 5 chapters and chapter 2 failing, the surviving slice is
// [c1, c3, c4, c5]. Positional lookup then yields:
//   chapter 3 -> ChapterAnalyses[2] = c4  (WRONG, should be c3)
//   chapter 4 -> ChapterAnalyses[3] = c5  (WRONG, should be c4)
//   chapter 5 -> ChapterAnalyses[4] = out-of-range (no context)
func TestGetTranslationContext_ChapterFailure_NoMisattribution(t *testing.T) {
	const numChapters = 5
	const failChapter = 2

	chapters := make([]ebook.Chapter, numChapters)
	for i := range chapters {
		chapters[i] = ebook.Chapter{
			Title:    fmt.Sprintf("Chapter %d", i+1),
			Sections: []ebook.Section{{Content: fmt.Sprintf("Content of chapter %d", i+1)}},
		}
	}
	book := &ebook.Book{Chapters: chapters}

	coord := &PreparationCoordinator{
		config:    PreparationConfig{SourceLanguage: "en", TargetLanguage: "sr"},
		providers: []translator.Translator{newChapterFailureMock(failChapter)},
	}

	analyses, err := coord.analyzeChapters(context.Background(), book)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	analysis := &ContentAnalysis{ChapterAnalyses: analyses}

	// Every chapter that SUCCEEDED must receive ITS OWN summary (or, for the
	// failed chapter, simply no wrong-chapter summary). Crucially, no surviving
	// chapter may receive a DIFFERENT chapter's summary.
	for chapterNum := 1; chapterNum <= numChapters; chapterNum++ {
		got := GetTranslationContext(analysis, chapterNum)

		if chapterNum == failChapter {
			// Failed chapter: must NOT receive any other chapter's summary.
			for other := 1; other <= numChapters; other++ {
				if other == failChapter {
					continue
				}
				wrong := fmt.Sprintf("SUMMARY-OF-CHAPTER-%d", other)
				if strings.Contains(got, wrong) {
					t.Fatalf("chapter %d (failed) wrongly received %q", chapterNum, wrong)
				}
			}
			continue
		}

		want := fmt.Sprintf("SUMMARY-OF-CHAPTER-%d", chapterNum)
		if !strings.Contains(got, want) {
			t.Fatalf("GetTranslationContext(chapter=%d) missing its own summary %q (mis-attributed "+
				"after a failed chapter dropped from the slice). Context was:\n%s", chapterNum, want, got)
		}
		// And it must not contain any OTHER chapter's summary.
		for other := 1; other <= numChapters; other++ {
			if other == chapterNum {
				continue
			}
			wrong := fmt.Sprintf("SUMMARY-OF-CHAPTER-%d", other)
			if strings.Contains(got, wrong) {
				t.Fatalf("GetTranslationContext(chapter=%d) wrongly contains %q — positional indexing "+
					"mis-attributed a dropped-chapter neighbour.", chapterNum, wrong)
			}
		}
	}
}
