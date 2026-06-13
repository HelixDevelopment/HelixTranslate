package preparation

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
)

// consolidationMock distinguishes the prompt kinds PrepareBook issues:
//   - chapter-analysis prompts → per-chapter JSON (with chapter_num + summary)
//   - consolidation prompts     → a realistic consolidated analysis that, like a
//     real LLM, OMITS chapter_analyses (the consolidation prompt never asks for them)
//   - book-level analysis prompts → a generic content analysis
type consolidationMock struct {
	re *regexp.Regexp
}

func newConsolidationMock() *consolidationMock {
	return &consolidationMock{re: regexp.MustCompile(`Chapter (\d+)`)}
}

func (m *consolidationMock) Translate(ctx context.Context, text, _ string) (string, error) {
	switch {
	case strings.Contains(text, "You are analyzing Chapter"):
		num := 0
		if mt := m.re.FindStringSubmatch(text); len(mt) > 1 {
			num, _ = strconv.Atoi(mt[1])
		}
		return `{"chapter_num": ` + strconv.Itoa(num) +
			`, "summary": "SUMMARY-OF-CHAPTER-` + strconv.Itoa(num) + `"}`, nil
	case strings.Contains(text, "FINAL CONSOLIDATED ANALYSIS"):
		// Realistic consolidated output: prose fields only, NO chapter_analyses.
		return `{"content_type": "fiction", "genre": "drama", "tone": "neutral"}`, nil
	default:
		// Book-level pass analysis.
		return `{"content_type": "fiction", "genre": "drama", "tone": "neutral"}`, nil
	}
}

func (m *consolidationMock) TranslateWithProgress(ctx context.Context, text, c string, _ *events.EventBus, _ string) (string, error) {
	return m.Translate(ctx, text, c)
}
func (m *consolidationMock) GetStats() translator.TranslationStats {
	return translator.TranslationStats{}
}
func (m *consolidationMock) GetName() string { return "consolidation-mock" }

// TestPrepareBook_ConsolidationPreservesChapterAnalyses asserts that when
// multi-pass consolidation runs AND chapter analysis is enabled, the chapter
// analyses survive into FinalAnalysis. A real LLM's consolidated JSON omits
// chapter_analyses, so naively replacing FinalAnalysis with the consolidated
// parse silently discards all chapter analysis work — leaving the translator
// with no per-chapter context.
func TestPrepareBook_ConsolidationPreservesChapterAnalyses(t *testing.T) {
	const numChapters = 3
	chapters := make([]ebook.Chapter, numChapters)
	for i := range chapters {
		chapters[i] = ebook.Chapter{
			Title:    "Chapter " + strconv.Itoa(i+1),
			Sections: []ebook.Section{{Content: "Content " + strconv.Itoa(i+1)}},
		}
	}
	book := &ebook.Book{Chapters: chapters}
	book.Metadata.Title = "Test Book"

	mock := newConsolidationMock()
	coord := &PreparationCoordinator{
		config: PreparationConfig{
			SourceLanguage:  "en",
			TargetLanguage:  "sr",
			PassCount:       2, // >1 → consolidation path
			AnalyzeChapters: true,
		},
		providers: []translator.Translator{mock},
	}

	result, err := coord.PrepareBook(context.Background(), book)
	if err != nil {
		t.Fatalf("PrepareBook error: %v", err)
	}

	if got := len(result.FinalAnalysis.ChapterAnalyses); got != numChapters {
		t.Fatalf("FinalAnalysis.ChapterAnalyses lost after consolidation: got %d, want %d. "+
			"Chapter analysis work was performed but discarded by the consolidation success path.",
			got, numChapters)
	}
}
