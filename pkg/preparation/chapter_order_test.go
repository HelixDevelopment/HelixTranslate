package preparation

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// chapterOrderMock returns, per chapter-analysis prompt, JSON whose chapter_num
// AND summary encode the chapter under analysis. A descending per-chapter delay
// (chapter 1 slowest) forces later chapters to complete first, so a
// completion-order append produces a slice that is NOT in chapter order.
type chapterOrderMock struct {
	re *regexp.Regexp
}

func newChapterOrderMock() *chapterOrderMock {
	return &chapterOrderMock{re: regexp.MustCompile(`Chapter (\d+)`)}
}

func (m *chapterOrderMock) Translate(ctx context.Context, text, _ string) (string, error) {
	num := 0
	if strings.Contains(text, "You are analyzing Chapter") {
		if mt := m.re.FindStringSubmatch(text); len(mt) > 1 {
			num, _ = strconv.Atoi(mt[1])
		}
	}
	// Earlier chapters sleep LONGER so they finish LAST under parallelism.
	if num > 0 {
		time.Sleep(time.Duration(40-num) * time.Millisecond)
	}
	return fmt.Sprintf(`{"chapter_num": %d, "summary": "SUMMARY-OF-CHAPTER-%d"}`, num, num), nil
}

func (m *chapterOrderMock) TranslateWithProgress(ctx context.Context, text, c string, _ *events.EventBus, _ string) (string, error) {
	return m.Translate(ctx, text, c)
}
func (m *chapterOrderMock) GetStats() translator.TranslationStats { return translator.TranslationStats{} }
func (m *chapterOrderMock) GetName() string                       { return "chapter-order-mock" }

// TestAnalyzeChapters_PreservesChapterOrder asserts the returned slice is in
// chapter order, because downstream GetTranslationContext indexes
// ChapterAnalyses[chapterNum-1] by position. With completion-order append the
// slice is shuffled and the wrong chapter's analysis is returned.
func TestAnalyzeChapters_PreservesChapterOrder(t *testing.T) {
	const numChapters = 8

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
		providers: []translator.Translator{newChapterOrderMock()},
	}

	analyses, err := coord.analyzeChapters(context.Background(), book)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(analyses) != numChapters {
		t.Fatalf("expected %d analyses, got %d", numChapters, len(analyses))
	}

	// Slice position i MUST correspond to chapter i+1 (GetTranslationContext's assumption).
	for i, a := range analyses {
		wantNum := i + 1
		wantSummary := fmt.Sprintf("SUMMARY-OF-CHAPTER-%d", wantNum)
		if a.ChapterNum != wantNum || a.Summary != wantSummary {
			t.Fatalf("ChapterAnalyses[%d] = {num=%d, summary=%q}; want {num=%d, summary=%q}. "+
				"Slice is in completion order, not chapter order — GetTranslationContext[chapterNum-1] "+
				"mis-attributes analysis to the wrong chapter.",
				i, a.ChapterNum, a.Summary, wantNum, wantSummary)
		}
	}
}

// TestGetTranslationContext_AfterParallelAnalysis_NoMisattribution proves the
// end-user-visible impact: the translation context handed to chapter N must
// describe chapter N, not whichever chapter happened to finish first.
func TestGetTranslationContext_AfterParallelAnalysis_NoMisattribution(t *testing.T) {
	const numChapters = 8

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
		providers: []translator.Translator{newChapterOrderMock()},
	}

	analyses, err := coord.analyzeChapters(context.Background(), book)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	analysis := &ContentAnalysis{ChapterAnalyses: analyses}
	for chapterNum := 1; chapterNum <= numChapters; chapterNum++ {
		got := GetTranslationContext(analysis, chapterNum)
		want := fmt.Sprintf("SUMMARY-OF-CHAPTER-%d", chapterNum)
		if !strings.Contains(got, want) {
			t.Fatalf("GetTranslationContext(chapter=%d) does not contain %q — wrong chapter's "+
				"summary was attributed. Context was:\n%s", chapterNum, want, got)
		}
	}
}
