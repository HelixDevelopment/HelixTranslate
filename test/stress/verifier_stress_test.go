//go:build stress

package stress

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"digital.vasic.translator/internal/verifier"
	"digital.vasic.translator/internal/verifier/discovery"
	"digital.vasic.translator/internal/verifier/scoring"
	"digital.vasic.translator/internal/verifier/selection"
)

// TestStress_ScoringEngine_HighConcurrency stress-tests scoring under load.
func TestStress_ScoringEngine_HighConcurrency(t *testing.T) {
	engine := scoring.NewEngine(scoring.ScoreWeights{
		ResponseSpeed:     0.20,
		CostEffectiveness: 0.30,
		ModelEfficiency:   0.25,
		Capability:        0.20,
		Recency:           0.05,
	})

	const numGoroutines = 500
	const scoresPerGoroutine = 200

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			for j := 0; j < scoresPerGoroutine; j++ {
				modelID := fmt.Sprintf("stress-g%d-m%d", gID, j)
				_, err := engine.CalculateScore(modelID, 0.9, 0.8, 0.7, 0.6, 0.5)
				if err != nil {
					t.Errorf("scoring failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	totalScores := numGoroutines * scoresPerGoroutine

	t.Logf("Calculated %d scores in %v (%.0f scores/sec)", totalScores, elapsed, float64(totalScores)/elapsed.Seconds())

	// Verify all scores are retrievable
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < scoresPerGoroutine; j++ {
			modelID := fmt.Sprintf("stress-g%d-m%d", i, j)
			_, ok := engine.GetScore(modelID)
			if !ok {
				t.Errorf("Score missing for %s", modelID)
			}
		}
	}
}

// TestStress_SelectionEngine_ManyModels tests selection with large model catalogs.
func TestStress_SelectionEngine_ManyModels(t *testing.T) {
	reg := verifier.NewRegistry()
	scoreEngine := scoring.NewEngine(scoring.ScoreWeights{ResponseSpeed: 1.0})

	// Register 10,000 models
	for i := 0; i < 10000; i++ {
		modelID := fmt.Sprintf("model-%d", i)
		reg.AddModel(verifier.Model{
			ID:                  modelID,
			ProviderID:          fmt.Sprintf("provider-%d", i%50),
			VerificationStatus:  "verified",
			CanSeeCode:          true,
			AffirmativeResponse: true,
			OverallScore:        float64(i%100) / 100.0,
			Capabilities:        map[string]bool{"streaming": true},
		})
		_, _ = scoreEngine.CalculateScore(modelID, float64(i%100)/100.0, 0, 0, 0, 0)
	}

	cfg := verifier.DefaultConfig()
	selEngine := selection.NewEngine(reg, scoreEngine, cfg)

	// Perform 1000 selections
	start := time.Now()
	for i := 0; i < 1000; i++ {
		_, err := selEngine.SelectModel(context.Background(), selection.TaskRequirements{
			RequireStreaming: true,
		})
		if err != nil {
			t.Fatalf("Selection failed: %v", err)
		}
	}
	elapsed := time.Since(start)
	t.Logf("1000 selections from 10k models took %v (%.0f selections/sec)", elapsed, 1000.0/elapsed.Seconds())
}

// TestStress_DiscoveryService_RapidSync tests rapid discovery cycles.
func TestStress_DiscoveryService_RapidSync(t *testing.T) {
	reg := verifier.NewRegistry()
	cfg := verifier.DefaultConfig()
	svc := discovery.NewService(cfg, reg)

	// Register many providers
	for i := 0; i < 100; i++ {
		svc.RegisterProvider(verifier.ProviderConfig{
			ID:      fmt.Sprintf("provider-%d", i),
			APIKey:  "key",
			BaseURL: fmt.Sprintf("https://api%d.example.com", i),
		})
	}

	// Run 100 discovery cycles
	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := svc.Discover(context.Background()); err != nil {
			t.Fatalf("Discovery failed: %v", err)
		}
	}
	elapsed := time.Since(start)
	t.Logf("100 discovery cycles with 100 providers took %v", elapsed)
}
