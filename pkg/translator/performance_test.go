package translator_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/pkg/hardware"
	"digital.vasic.translator/pkg/models"
	"digital.vasic.translator/pkg/translator"
	"digital.vasic.translator/pkg/translator/llm"
)

// BenchmarkHardwareDetection benchmarks hardware detection performance
func BenchmarkHardwareDetection(b *testing.B) {
	detector := hardware.NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.Detect()
		if err != nil {
			b.Fatalf("Hardware detection failed: %v", err)
		}
	}
}

// BenchmarkModelSelection benchmarks model selection performance
func BenchmarkModelSelection(b *testing.B) {
	registry := models.NewRegistry()
	maxRAM := uint64(16 * 1024 * 1024 * 1024) // 16GB
	languages := []string{"ru", "sr"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.FindBestModel(maxRAM, languages, true)
		if err != nil {
			b.Fatalf("Model selection failed: %v", err)
		}
	}
}

// BenchmarkModelFiltering benchmarks model filtering operations
func BenchmarkModelFiltering(b *testing.B) {
	registry := models.NewRegistry()

	b.Run("FilterByRAM", func(b *testing.B) {
		maxRAM := uint64(8 * 1024 * 1024 * 1024) // 8GB
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.FilterByRAM(maxRAM)
		}
	})

	b.Run("FilterByLanguages", func(b *testing.B) {
		languages := []string{"ru", "sr"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = registry.FilterByLanguages(languages)
		}
	})
}

// BenchmarkTranslation benchmarks actual translation performance
func BenchmarkTranslation(b *testing.B) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		b.Skip("DEEPSEEK_API_KEY not set")
	}

	config := translator.TranslationConfig{
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
	}

	client, err := llm.NewDeepSeekClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	testText := "Привет, мир!"
	prompt := "Translate: " + testText

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, err := client.Translate(ctx, testText, prompt)
		cancel()
		if err != nil {
			b.Fatalf("Translation failed: %v", err)
		}
	}
}

// TestStressTranslation tests translation under stress
func TestStressTranslation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	config := translator.TranslationConfig{
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
	}

	client, err := llm.NewDeepSeekClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test concurrent requests
	const numConcurrent = 10
	var wg sync.WaitGroup
	errors := make(chan error, numConcurrent)

	startTime := time.Now()

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			testText := fmt.Sprintf("Тест номер %d", id)
			prompt := "Translate: " + testText

			_, err := client.Translate(ctx, testText, prompt)
			if err != nil {
				errors <- fmt.Errorf("translation %d failed: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(startTime)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Error: %v", err)
		errorCount++
	}

	t.Logf("Stress test completed:")
	t.Logf("  Concurrent requests: %d", numConcurrent)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Average per request: %v", duration/numConcurrent)
	t.Logf("  Errors: %d", errorCount)

	if errorCount > numConcurrent/2 {
		t.Errorf("Too many errors: %d/%d", errorCount, numConcurrent)
	}
}

// TestLargeTextTranslation tests translation of large texts
func TestLargeTextTranslation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large text test in short mode")
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	config := translator.TranslationConfig{
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
	}

	client, err := llm.NewDeepSeekClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Generate large text (approximately 1000 words)
	baseText := "Это длинный текст для тестирования производительности перевода. "
	largeText := ""
	for i := 0; i < 100; i++ {
		largeText += baseText
	}

	t.Logf("Testing translation of %d characters", len(largeText))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	prompt := "Translate the following Russian text to Serbian:\n\n" + largeText

	startTime := time.Now()
	result, err := client.Translate(ctx, largeText, prompt)
	duration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	t.Logf("Large text translation completed:")
	t.Logf("  Input length: %d characters", len(largeText))
	t.Logf("  Output length: %d characters", len(result))
	t.Logf("  Duration: %v", duration)
	t.Logf("  Speed: %.2f chars/second", float64(len(largeText))/duration.Seconds())

	if len(result) == 0 {
		t.Error("Translation result is empty")
	}
}

// TestMemoryUsage tests memory usage during translation
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Test hardware detection memory usage
	t.Run("Hardware Detection", func(t *testing.T) {
		detector := hardware.NewDetector()

		// Run detection many times to check for memory leaks
		for i := 0; i < 1000; i++ {
			_, err := detector.Detect()
			if err != nil {
				t.Fatalf("Hardware detection failed: %v", err)
			}
		}
	})

	// Test model registry memory usage
	t.Run("Model Registry", func(t *testing.T) {
		// Create and destroy many registries
		for i := 0; i < 1000; i++ {
			registry := models.NewRegistry()
			_ = registry.List()
		}
	})

	t.Log("Memory usage test completed - check with profiling tools for detailed analysis")
}

// TestResponseTime tests response time consistency
func TestResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping response time test in short mode")
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Skip("DEEPSEEK_API_KEY not set")
	}

	config := translator.TranslationConfig{
		Provider:       "deepseek",
		Model:          "deepseek-chat",
		SourceLanguage: "ru",
		TargetLanguage: "sr",
	}

	client, err := llm.NewDeepSeekClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Measure response times for multiple requests
	const numRequests = 5
	durations := make([]time.Duration, numRequests)

	for i := 0; i < numRequests; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		testText := "Привет"
		prompt := "Translate: " + testText

		startTime := time.Now()
		_, err := client.Translate(ctx, testText, prompt)
		durations[i] = time.Since(startTime)

		cancel()

		if err != nil {
			t.Fatalf("Translation %d failed: %v", i, err)
		}

		// Small delay between requests
		time.Sleep(1 * time.Second)
	}

	// Calculate statistics
	var total time.Duration
	var min, max time.Duration
	min = durations[0]
	max = durations[0]

	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := total / time.Duration(numRequests)

	t.Logf("Response time statistics:")
	t.Logf("  Requests: %d", numRequests)
	t.Logf("  Average: %v", avg)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Variance: %v", max-min)

	// Check if variance is too high (more than 10x)
	if max > min*10 {
		t.Logf("Warning: High response time variance detected")
	}
}

// BenchmarkConcurrentOperations benchmarks concurrent operations
func BenchmarkConcurrentOperations(b *testing.B) {
	detector := hardware.NewDetector()
	registry := models.NewRegistry()

	b.Run("Parallel Hardware Detection", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := detector.Detect()
				if err != nil {
					b.Fatalf("Hardware detection failed: %v", err)
				}
			}
		})
	})

	b.Run("Parallel Model Selection", func(b *testing.B) {
		maxRAM := uint64(16 * 1024 * 1024 * 1024)
		languages := []string{"ru", "sr"}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := registry.FindBestModel(maxRAM, languages, true)
				if err != nil {
					b.Fatalf("Model selection failed: %v", err)
				}
			}
		})
	})
}

// TestCachingPerformance tests caching performance improvements
func TestCachingPerformance(t *testing.T) {
	registry := models.NewRegistry()
	maxRAM := uint64(16 * 1024 * 1024 * 1024)
	languages := []string{"ru", "sr"}

	// First call (no cache)
	start1 := time.Now()
	model1, err := registry.FindBestModel(maxRAM, languages, true)
	duration1 := time.Since(start1)
	if err != nil {
		t.Fatalf("Model selection failed: %v", err)
	}

	// Second call (should use internal caching if implemented)
	start2 := time.Now()
	model2, err := registry.FindBestModel(maxRAM, languages, true)
	duration2 := time.Since(start2)
	if err != nil {
		t.Fatalf("Model selection failed: %v", err)
	}

	t.Logf("Model selection performance:")
	t.Logf("  First call: %v", duration1)
	t.Logf("  Second call: %v", duration2)
	t.Logf("  Speedup: %.2fx", float64(duration1)/float64(duration2))

	if model1.ID != model2.ID {
		t.Error("Different models selected for same criteria")
	}
}
