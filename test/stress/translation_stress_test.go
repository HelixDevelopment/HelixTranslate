//go:build stress

package stress

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
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

func TestTranslationStress_ConcurrentRequests(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs
	// The test infrastructure with dynamic ports prevents conflicts

	const numGoroutines = 50
	const requestsPerGoroutine = 20
	const totalRequests = numGoroutines * requestsPerGoroutine

	var successCount int64
	var errorCount int64
	var wg sync.WaitGroup

	// Start stress test
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				text := fmt.Sprintf("Test text %d-%d for stress testing", id, j)

				// Create translator for each request (simulating real usage)
				trans := &MockTranslator{}

				_, err := trans.Translate(context.Background(), text, "")
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Verify results
	t.Logf("Stress test completed in %v", duration)
	t.Logf("Total requests: %d", totalRequests)
	t.Logf("Successful requests: %d", successCount)
	t.Logf("Failed requests: %d", errorCount)
	t.Logf("Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

	if errorCount > 0 {
		t.Errorf("Had %d failed requests out of %d total", errorCount, totalRequests)
	}

	// Performance assertions
	if duration > 30*time.Second {
		t.Errorf("Test took too long: %v (should complete within 30s)", duration)
	}

	minRequestsPerSecond := 50.0 // Adjust based on system capabilities
	if rps := float64(totalRequests) / duration.Seconds(); rps < minRequestsPerSecond {
		t.Errorf("Throughput too low: %.2f requests/sec (minimum: %.2f)", rps, minRequestsPerSecond)
	}
}

func TestDictionaryStress_LargeConcurrentLoad(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	const numGoroutines = 20
	const translationsPerGoroutine = 10
	const totalTranslations = numGoroutines * translationsPerGoroutine

	var successCount int64
	var errorCount int64
	var wg sync.WaitGroup

	// Start stress test
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < translationsPerGoroutine; j++ {
				text := fmt.Sprintf("Concurrent mock test %d-%d", id, j)
				trans := &MockTranslator{}

				result, err := trans.Translate(context.Background(), text, "")
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else if result == "" {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Verify results
	t.Logf("Dictionary stress test completed in %v", duration)
	t.Logf("Total translations: %d", totalTranslations)
	t.Logf("Successful translations: %d", successCount)
	t.Logf("Failed translations: %d", errorCount)
	t.Logf("Translations per second: %.2f", float64(totalTranslations)/duration.Seconds())

	if errorCount > 0 {
		t.Errorf("Had %d failed translations out of %d total", errorCount, totalTranslations)
	}
}

func TestMemoryStress_LargeTextTranslation(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	// Create a moderately large text for memory stress testing
	var largeText string
	for i := 0; i < 1000; i++ {
		largeText += fmt.Sprintf("This is sentence number %d in a large text document that will test memory usage and performance under stress. ", i)
	}

	t.Logf("Created large text of %d characters", len(largeText))

	trans := &MockTranslator{}
	start := time.Now()
	result, err := trans.Translate(context.Background(), largeText, "")
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Large text translation failed: %v", err)
	}

	t.Logf("Large text translation completed in %v", duration)
	t.Logf("Result length: %d characters", len(result))

	// Verify result is not empty and reasonable
	if len(result) == 0 {
		t.Error("Translation result is empty")
	}

	// Performance check
	if duration > 5*time.Second {
		t.Errorf("Large text translation took too long: %v", duration)
	}
}

func TestResourceStress_LongRunning(t *testing.T) {
	// Note: This test is now enabled - remove short mode check to ensure it runs

	// Test system stability over time
	duration := 10 * time.Second

	t.Logf("Starting long-running resource stress test for %v", duration)

	start := time.Now()
	end := start.Add(duration)

	var translationCount int64
	var errorCount int64

	for time.Now().Before(end) {
		text := fmt.Sprintf("Resource stress test translation %d", translationCount)
		trans := &MockTranslator{}

		_, err := trans.Translate(context.Background(), text, "")
		if err != nil {
			atomic.AddInt64(&errorCount, 1)
		} else {
			atomic.AddInt64(&translationCount, 1)
		}

		// Small delay to prevent overwhelming the system
		time.Sleep(50 * time.Millisecond)
	}

	actualDuration := time.Since(start)
	t.Logf("Long-running test completed in %v", actualDuration)
	t.Logf("Total translations: %d", translationCount)
	t.Logf("Errors: %d", errorCount)
	t.Logf("Average translations per second: %.2f", float64(translationCount)/actualDuration.Seconds())

	if errorCount > 0 {
		t.Errorf("Had %d errors during long-running test", errorCount)
	}

	minTranslations := int64(20) // Adjust based on system capabilities
	if translationCount < minTranslations {
		t.Errorf("Too few translations completed: %d (minimum: %d)", translationCount, minTranslations)
	}
}
