//go:build performance

package performance

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier/scoring"
)

// BenchmarkScoringEngine_ConcurrentLoad measures scoring under concurrent load.
func BenchmarkScoringEngine_ConcurrentLoad(b *testing.B) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			modelID := fmt.Sprintf("model-%d", i%1000)
			_, _ = engine.CalculateScore(modelID, 0.9, 0.8, 0.7, 0.6, 0.5)
			i++
		}
	})
}

// BenchmarkScoringEngine_Throughput measures tokens of throughput.
func BenchmarkScoringEngine_Throughput(b *testing.B) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	b.ResetTimer()
	start := time.Now()
	for i := 0; i < b.N; i++ {
		_, _ = engine.CalculateScore("throughput-model", 0.9, 0.8, 0.7, 0.6, 0.5)
	}
	elapsed := time.Since(start)
	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "scores/sec")
}

// TestScoringEngine_MemoryLeak performs a sustained load test.
func TestScoringEngine_MemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Run 100k score calculations
	for i := 0; i < 100000; i++ {
		modelID := fmt.Sprintf("model-%d", i)
		_, err := engine.CalculateScore(modelID, 0.9, 0.8, 0.7, 0.6, 0.5)
		if err != nil {
			t.Fatalf("scoring failed: %v", err)
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Memory growth should be bounded (allow 50MB for 100k entries)
	growthBytes := int64(m2.Alloc) - int64(m1.Alloc)
	maxGrowth := int64(50 * 1024 * 1024)
	if growthBytes > maxGrowth {
		t.Errorf("Memory growth too high: %d bytes (max %d)", growthBytes, maxGrowth)
	}

	t.Logf("Memory growth: %d bytes for 100k scores", growthBytes)
}

// TestScoringEngine_ConcurrentSafety verifies thread-safety.
func TestScoringEngine_ConcurrentSafety(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed: 1.0,
	})

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			modelID := fmt.Sprintf("concurrent-model-%d", id)
			_, err := engine.CalculateScore(modelID, float64(id)/100.0, 0, 0, 0, 0)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errCount := 0
	for err := range errors {
		if err != nil {
			t.Logf("Concurrent scoring error: %v", err)
			errCount++
		}
	}
	if errCount > 0 {
		t.Errorf("Got %d errors during concurrent scoring", errCount)
	}

	// Verify all scores are retrievable
	for i := 0; i < 100; i++ {
		modelID := fmt.Sprintf("concurrent-model-%d", i)
		_, ok := engine.GetScore(modelID)
		if !ok {
			t.Errorf("Score missing for %s after concurrent write", modelID)
		}
	}
}
