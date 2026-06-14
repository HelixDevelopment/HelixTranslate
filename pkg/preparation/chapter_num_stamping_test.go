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

// chapterNoNumMock simulates a REALISTIC LLM that returns a valid per-chapter
// analysis but OMITS the "chapter_num" field from its JSON (a common real-world
// behaviour — LLMs frequently drop fields even when the prompt asks for them).
// It fails on one configured chapter to trigger the slice-compaction path.
//
// Because the JSON carries no chapter_num, the parsed ChapterAnalysis has
// ChapterNum == 0 unless the coordinator stamps the authoritative number it
// already knows (chapterIdx+1). Without that stamping, lookupChapterAnalysis
// sees no numbered entries and falls back to positional indexing — re-opening
// the exact mis-attribution that §11.4.135-class guard was meant to close.
type chapterNoNumMock struct {
	re     *regexp.Regexp
	failOn int
}

func newChapterNoNumMock(failOn int) *chapterNoNumMock {
	return &chapterNoNumMock{re: regexp.MustCompile(`Chapter (\d+)`), failOn: failOn}
}

func (m *chapterNoNumMock) Translate(ctx context.Context, text, _ string) (string, error) {
	num := 0
	if strings.Contains(text, "You are analyzing Chapter") {
		if mt := m.re.FindStringSubmatch(text); len(mt) > 1 {
			num, _ = strconv.Atoi(mt[1])
		}
	}
	if num == m.failOn {
		return "", fmt.Errorf("simulated chapter %d analysis failure", num)
	}
	// NOTE: deliberately NO "chapter_num" key — only summary content.
	return fmt.Sprintf(`{"summary": "SUMMARY-OF-CHAPTER-%d"}`, num), nil
}

func (m *chapterNoNumMock) TranslateWithProgress(ctx context.Context, text, c string, _ *events.EventBus, _ string) (string, error) {
	return m.Translate(ctx, text, c)
}
func (m *chapterNoNumMock) GetStats() translator.TranslationStats { return translator.TranslationStats{} }
func (m *chapterNoNumMock) GetName() string                       { return "chapter-no-num-mock" }

// TestAnalyzeChapters_StampsAuthoritativeChapterNum proves the coordinator must
// stamp the chapter number it already knows onto each parsed analysis, rather
// than trusting the LLM's (possibly missing/wrong) value.
//
// With 5 chapters and chapter 2 failing, the surviving slice positions are
// [c1, c3, c4, c5]. If ChapterNum is left at 0 (LLM omitted it), the downstream
// by-number lookup degrades to positional indexing and chapter 3 receives c4's
// summary, etc. — the user-visible mis-attribution.
func TestAnalyzeChapters_StampsAuthoritativeChapterNum(t *testing.T) {
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
		providers: []translator.Translator{newChapterNoNumMock(failChapter)},
	}

	analyses, err := coord.analyzeChapters(context.Background(), book)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Direct invariant: each surviving analysis must carry its authoritative
	// chapter number so downstream by-number attribution works regardless of
	// what the LLM returned.
	for i := range analyses {
		summary := analyses[i].Summary
		// Derive the true chapter from the summary the mock encoded.
		var trueNum int
		_, _ = fmt.Sscanf(summary, "SUMMARY-OF-CHAPTER-%d", &trueNum)
		if trueNum == 0 {
			t.Fatalf("could not parse true chapter from summary %q", summary)
		}
		if analyses[i].ChapterNum != trueNum {
			t.Fatalf("analysis with summary %q has ChapterNum=%d, want %d "+
				"(coordinator did not stamp the authoritative chapter number; the LLM omitted chapter_num)",
				summary, analyses[i].ChapterNum, trueNum)
		}
	}

	// End-user-visible invariant: no surviving chapter receives another
	// chapter's context.
	analysis := &ContentAnalysis{ChapterAnalyses: analyses}
	for chapterNum := 1; chapterNum <= numChapters; chapterNum++ {
		got := GetTranslationContext(analysis, chapterNum)
		if chapterNum == failChapter {
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
			t.Fatalf("GetTranslationContext(chapter=%d) missing its own summary %q "+
				"(LLM-omitted chapter_num degraded by-number lookup to positional indexing). Context:\n%s",
				chapterNum, want, got)
		}
		for other := 1; other <= numChapters; other++ {
			if other == chapterNum {
				continue
			}
			wrong := fmt.Sprintf("SUMMARY-OF-CHAPTER-%d", other)
			if strings.Contains(got, wrong) {
				t.Fatalf("GetTranslationContext(chapter=%d) wrongly contains %q — mis-attribution.",
					chapterNum, wrong)
			}
		}
	}
}
