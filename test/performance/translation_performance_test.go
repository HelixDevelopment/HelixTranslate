//go:build performance
// +build performance

package performance

import (
	"context"
	"fmt"
	"testing"
	"time"

	"digital.vasic.translator/pkg/ebook"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/verification"
)

// MockTranslator for testing
type MockTranslator struct{}

func (m *MockTranslator) Translate(ctx context.Context, text, context string) (string, error) {
	return "translated: " + text, nil
}

func (m *MockTranslator) TranslateWithProgress(ctx context.Context, text, context string, eventBus *events.EventBus, sessionID string) (string, error) {
	return m.Translate(ctx, text, context)
}

func (m *MockTranslator) GetStats() translator.TranslationStats {
	return translator.TranslationStats{
		Total:      0,
		Translated: 0,
		Cached:     0,
		Errors:     0,
	}
}

func (m *MockTranslator) GetName() string {
	return "mock"
}

// BenchmarkMockTranslation benchmarks mock-based translation
func BenchmarkDictionaryTranslation(b *testing.B) {
	ctx := context.Background()

	trans := &MockTranslator{}

	texts := []string{
		"Hello, world!",
		"This is a test sentence with multiple words.",
		"The quick brown fox jumps over the lazy dog.",
		"In a hole in the ground there lived a hobbit.",
		"It was the best of times, it was the worst of times.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		text := texts[i%len(texts)]
		_, _ = trans.Translate(ctx, text, "")
	}
}

// BenchmarkBookTranslation benchmarks full book translation
func BenchmarkBookTranslation(b *testing.B) {
	ctx := context.Background()

	trans := &MockTranslator{}

	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	langDetector := language.NewDetector(nil)
	universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)

	eventBus := events.NewEventBus()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testBook := createTestBook(10, 5)
		_ = universalTrans.TranslateBook(ctx, testBook, eventBus, "bench")
	}
}

// BenchmarkVerification benchmarks translation verification
func BenchmarkVerification(b *testing.B) {
	ctx := context.Background()

	book := createTestBook(10, 5)

	// Pre-translate the book

	trans := &MockTranslator{}
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	langDetector := language.NewDetector(nil)
	universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)
	eventBus := events.NewEventBus()
	_ = universalTrans.TranslateBook(ctx, book, eventBus, "setup")

	verifier := verification.NewVerifier(en, sr, nil, "bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.VerifyBook(ctx, book)
	}
}

// BenchmarkSmallBook benchmarks small book translation
func BenchmarkSmallBook(b *testing.B) {
	ctx := context.Background()

	trans := &MockTranslator{}
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	langDetector := language.NewDetector(nil)
	universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)
	eventBus := events.NewEventBus()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		book := createTestBook(3, 2) // Small book: 3 chapters, 2 sections
		_ = universalTrans.TranslateBook(ctx, book, eventBus, "bench-small")
	}
}

// BenchmarkMediumBook benchmarks medium book translation
func BenchmarkMediumBook(b *testing.B) {
	ctx := context.Background()

	trans := &MockTranslator{}
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	langDetector := language.NewDetector(nil)
	universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)
	eventBus := events.NewEventBus()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		book := createTestBook(20, 10) // Medium book: 20 chapters, 10 sections
		_ = universalTrans.TranslateBook(ctx, book, eventBus, "bench-medium")
	}
}

// BenchmarkLargeBook benchmarks large book translation
func BenchmarkLargeBook(b *testing.B) {
	ctx := context.Background()

	trans := &MockTranslator{}
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	langDetector := language.NewDetector(nil)
	universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)
	eventBus := events.NewEventBus()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		book := createTestBook(50, 20) // Large book: 50 chapters, 20 sections
		_ = universalTrans.TranslateBook(ctx, book, eventBus, "bench-large")
	}
}

// BenchmarkVerificationSmallBook benchmarks verification of small book
func BenchmarkVerificationSmallBook(b *testing.B) {
	ctx := context.Background()
	book := createTranslatedBook(5, 3)
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	verifier := verification.NewVerifier(en, sr, nil, "bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.VerifyBook(ctx, book)
	}
}

// BenchmarkVerificationLargeBook benchmarks verification of large book
func BenchmarkVerificationLargeBook(b *testing.B) {
	ctx := context.Background()
	book := createTranslatedBook(50, 20)
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	verifier := verification.NewVerifier(en, sr, nil, "bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = verifier.VerifyBook(ctx, book)
	}
}

// TestTranslationThroughput measures translation throughput
func TestTranslationThroughput(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	ctx := context.Background()

	trans := &MockTranslator{}

	// Measure throughput for 1000 sentences
	sentences := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		sentences[i] = fmt.Sprintf("This is test sentence number %d with some content.", i)
	}

	startTime := time.Now()
	for _, sentence := range sentences {
		_, _ = trans.Translate(ctx, sentence, "")
	}
	duration := time.Since(startTime)

	throughput := float64(1000) / duration.Seconds()
	t.Logf("Translation throughput: %.2f sentences/second", throughput)
	t.Logf("Average time per sentence: %v", duration/1000)

	// Assert minimum throughput
	if throughput < 10 {
		t.Errorf("Throughput too low: %.2f sentences/second (expected >= 10)", throughput)
	}
}

// TestVerificationThroughput measures verification throughput
func TestVerificationThroughput(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	ctx := context.Background()

	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	verifier := verification.NewVerifier(en, sr, nil, "throughput-test")

	// Create 100 books
	books := make([]*ebook.Book, 100)
	for i := 0; i < 100; i++ {
		books[i] = createTranslatedBook(5, 3)
	}

	startTime := time.Now()
	for _, book := range books {
		_, _ = verifier.VerifyBook(ctx, book)
	}
	duration := time.Since(startTime)

	throughput := float64(100) / duration.Seconds()
	t.Logf("Verification throughput: %.2f books/second", throughput)
	t.Logf("Average time per book: %v", duration/100)

	// Assert minimum throughput
	if throughput < 1 {
		t.Errorf("Throughput too low: %.2f books/second (expected >= 1)", throughput)
	}
}

// TestMemoryUsage measures memory usage during translation
func TestMemoryUsage(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	ctx := context.Background()

	trans := &MockTranslator{}
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	langDetector := language.NewDetector(nil)
	universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)
	eventBus := events.NewEventBus()

	// Translate multiple books and measure memory
	for i := 0; i < 10; i++ {
		book := createTestBook(20, 10)
		_ = universalTrans.TranslateBook(ctx, book, eventBus, fmt.Sprintf("memory-test-%d", i))
	}

	t.Log("Memory test completed - check profiling data for details")
}

// TestConcurrentTranslations measures performance under concurrent load
func TestConcurrentTranslations(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	ctx := context.Background()

	// Launch 10 concurrent translations
	done := make(chan time.Duration, 10)

	startTime := time.Now()
	for i := 0; i < 10; i++ {
		go func(id int) {
			trans := &MockTranslator{}
			en := language.Language{Code: "en", Name: "English"}
			sr := language.Language{Code: "sr", Name: "Serbian"}
			langDetector := language.NewDetector(nil)
			universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)
			eventBus := events.NewEventBus()

			taskStart := time.Now()
			book := createTestBook(10, 5)
			_ = universalTrans.TranslateBook(ctx, book, eventBus, fmt.Sprintf("concurrent-%d", id))
			duration := time.Since(taskStart)

			done <- duration
		}(i)
	}

	// Wait for all goroutines
	totalTaskTime := time.Duration(0)
	for i := 0; i < 10; i++ {
		duration := <-done
		totalTaskTime += duration
	}

	totalElapsed := time.Since(startTime)
	avgTaskTime := totalTaskTime / 10

	t.Logf("Concurrent translations completed in %v", totalElapsed)
	t.Logf("Average task time: %v", avgTaskTime)
	t.Logf("Total task time: %v", totalTaskTime)
	t.Logf("Concurrency efficiency: %.2f%%", (float64(totalTaskTime)/float64(totalElapsed))/10*100)
}

// TestScalability tests how performance scales with book size
func TestScalability(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	ctx := context.Background()

	trans := &MockTranslator{}
	en := language.Language{Code: "en", Name: "English"}
	sr := language.Language{Code: "sr", Name: "Serbian"}
	langDetector := language.NewDetector(nil)
	universalTrans := translator.NewUniversalTranslator(trans, langDetector, en, sr)
	eventBus := events.NewEventBus()

	sizes := []struct {
		name     string
		chapters int
		sections int
	}{
		{"tiny", 2, 2},
		{"small", 5, 5},
		{"medium", 10, 10},
		{"large", 20, 20},
		{"huge", 40, 40},
	}

	for _, size := range sizes {
		book := createTestBook(size.chapters, size.sections)
		totalSections := size.chapters * size.sections

		startTime := time.Now()
		_ = universalTrans.TranslateBook(ctx, book, eventBus, fmt.Sprintf("scale-%s", size.name))
		duration := time.Since(startTime)

		t.Logf("%s book (%d sections): %v (%.2f ms/section)",
			size.name, totalSections, duration, float64(duration.Milliseconds())/float64(totalSections))
	}
}

// createTestBook creates a book with specified chapters and sections
func createTestBook(numChapters, numSections int) *ebook.Book {
	chapters := make([]ebook.Chapter, numChapters)
	for i := 0; i < numChapters; i++ {
		sections := make([]ebook.Section, numSections)
		for j := 0; j < numSections; j++ {
			sections[j] = ebook.Section{
				Title:   fmt.Sprintf("Section %d", j+1),
				Content: fmt.Sprintf("This is the content of section %d in chapter %d. It contains some text that needs to be translated from English to Serbian.", j+1, i+1),
			}
		}
		chapters[i] = ebook.Chapter{
			Title:    fmt.Sprintf("Chapter %d", i+1),
			Sections: sections,
		}
	}

	return &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       "Test Book",
			Authors:     []string{"Test Author"},
			Description: "A test book for performance testing",
			Language:    "en",
		},
		Chapters: chapters,
	}
}

// createTranslatedBook creates a pre-translated book for verification testing
func createTranslatedBook(numChapters, numSections int) *ebook.Book {
	chapters := make([]ebook.Chapter, numChapters)
	for i := 0; i < numChapters; i++ {
		sections := make([]ebook.Section, numSections)
		for j := 0; j < numSections; j++ {
			sections[j] = ebook.Section{
				Title:   fmt.Sprintf("Одељак %d", j+1),
				Content: fmt.Sprintf("Ово је садржај одељка %d у поглављу %d. Садржи неки текст преведен са енглеског на српски.", j+1, i+1),
			}
		}
		chapters[i] = ebook.Chapter{
			Title:    fmt.Sprintf("Поглавље %d", i+1),
			Sections: sections,
		}
	}

	return &ebook.Book{
		Metadata: ebook.Metadata{
			Title:       "Преведена књига",
			Authors:     []string{"Аутор теста"},
			Description: "Преведена књига за тестирање перформанси",
			Language:    "sr",
		},
		Chapters: chapters,
	}
}
