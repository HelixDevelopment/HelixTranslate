package security

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRateLimiter_WithinLimit tests requests within rate limit
func TestRateLimiter_WithinLimit(t *testing.T) {
	rl := NewRateLimiter(10, 10) // 10 RPS, burst 10
	key := "test-client"

	// Should allow 10 requests immediately (burst)
	for i := 0; i < 10; i++ {
		allowed := rl.Allow(key)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}
}

// TestRateLimiter_ExceedLimit tests requests exceeding rate limit
func TestRateLimiter_ExceedLimit(t *testing.T) {
	rl := NewRateLimiter(5, 5) // 5 RPS, burst 5
	key := "test-client"

	// Use up burst
	for i := 0; i < 5; i++ {
		rl.Allow(key)
	}

	// Next request should be denied (burst exhausted, no time passed)
	allowed := rl.Allow(key)
	assert.False(t, allowed, "Request should be denied after burst exhausted")
}

// TestRateLimiter_Reset tests rate limit reset for a key
func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(5, 5)
	key := "test-client"

	// Exhaust burst
	for i := 0; i < 5; i++ {
		rl.Allow(key)
	}

	// Should be denied
	assert.False(t, rl.Allow(key))

	// Reset limiter
	rl.Reset(key)

	// Should be allowed again
	assert.True(t, rl.Allow(key))
}

// TestRateLimiter_MultipleClients tests rate limiting with multiple clients
func TestRateLimiter_MultipleClients(t *testing.T) {
	rl := NewRateLimiter(5, 5)

	clients := []string{"client1", "client2", "client3"}

	// Each client should have independent rate limits
	for _, client := range clients {
		for i := 0; i < 5; i++ {
			allowed := rl.Allow(client)
			assert.True(t, allowed, "Client %s request %d should be allowed", client, i+1)
		}

		// Each client should be limited independently
		allowed := rl.Allow(client)
		assert.False(t, allowed, "Client %s should be rate limited", client)
	}
}

// TestRateLimiter_ConcurrentRequests tests concurrent requests from same client
func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	rl := NewRateLimiter(100, 100) // Higher limits for concurrency test
	key := "test-client"

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Launch 100 concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow(key) {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// All requests should be allowed (within burst)
	assert.Equal(t, 100, allowedCount)

	// Next request should be denied
	assert.False(t, rl.Allow(key))
}

// TestRateLimiter_TokenRefill tests that tokens refill over time
func TestRateLimiter_TokenRefill(t *testing.T) {
	rl := NewRateLimiter(10, 5) // 10 RPS, burst 5
	key := "test-client"

	// Exhaust burst
	for i := 0; i < 5; i++ {
		rl.Allow(key)
	}

	// Should be denied
	assert.False(t, rl.Allow(key))

	// Wait for tokens to refill (0.1 second = 1 token at 10 RPS)
	time.Sleep(time.Millisecond * 150)

	// Should be allowed now
	assert.True(t, rl.Allow(key))
}

// TestRateLimiter_GetStats tests statistics retrieval
func TestRateLimiter_GetStats(t *testing.T) {
	rl := NewRateLimiter(10, 20)

	stats := rl.GetStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["total_limiters"]) // No clients yet
	assert.Equal(t, 10, stats["rps"])
	assert.Equal(t, 20, stats["burst"])

	// Add a client
	rl.Allow("client1")

	stats = rl.GetStats()
	assert.Equal(t, 1, stats["total_limiters"])
}

// TestRateLimiter_DifferentRates tests limiters with different rates
func TestRateLimiter_DifferentRates(t *testing.T) {
	tests := []struct {
		name  string
		rps   int
		burst int
	}{
		{"low rate", 1, 1},
		{"medium rate", 10, 10},
		{"high rate", 100, 100},
		{"high burst", 10, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.rps, tt.burst)
			key := "test-client"

			// Should allow burst number of requests
			for i := 0; i < tt.burst; i++ {
				allowed := rl.Allow(key)
				assert.True(t, allowed, "Request %d should be allowed", i+1)
			}

			// Next should be denied
			allowed := rl.Allow(key)
			assert.False(t, allowed, "Request should be denied after burst")
		})
	}
}

// Performance Test: TestRateLimiter_HighConcurrency tests high concurrent load
func TestRateLimiter_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")  // SKIP-OK: #short-mode
	}

	rl := NewRateLimiter(1000, 1000)
	key := "test-client"

	var wg sync.WaitGroup
	requests := 1000

	start := time.Now()

	// Launch many concurrent requests
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rl.Allow(key)
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	// Should complete quickly (< 100ms)
	assert.Less(t, duration.Milliseconds(), int64(100), "High concurrency should be handled quickly")
}

// Performance Test: TestRateLimiter_ManyClients tests many different clients
func TestRateLimiter_ManyClients(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping many clients test in short mode")  // SKIP-OK: #short-mode
	}

	rl := NewRateLimiter(10, 10)
	numClients := 100

	var wg sync.WaitGroup

	// Simulate many different clients
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			key := "client-" + string(rune('0'+clientID%10))
			for j := 0; j < 5; j++ {
				rl.Allow(key)
			}
		}(i)
	}

	wg.Wait()

	stats := rl.GetStats()
	// Should have limiters for different clients
	assert.Greater(t, stats["total_limiters"], 0)
}

// Stress Test: TestRateLimiter_ContinuousLoad tests continuous load
func TestRateLimiter_ContinuousLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping continuous load test in short mode")  // SKIP-OK: #short-mode
	}

	rl := NewRateLimiter(100, 100)
	key := "stress-client"

	duration := time.Second * 2
	deadline := time.Now().Add(duration)
	requestCount := 0

	// Send requests continuously for 2 seconds
	for time.Now().Before(deadline) {
		rl.Allow(key)
		requestCount++
	}

	assert.Greater(t, requestCount, 100, "Should handle continuous load")
}

// BenchmarkRateLimiter_Sequential benchmarks sequential requests
func BenchmarkRateLimiter_Sequential(b *testing.B) {
	rl := NewRateLimiter(1000, 1000)
	key := "bench-client"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow(key)
	}
}

// BenchmarkRateLimiter_Concurrent benchmarks concurrent requests
func BenchmarkRateLimiter_Concurrent(b *testing.B) {
	rl := NewRateLimiter(10000, 10000)
	key := "bench-client"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.Allow(key)
		}
	})
}

// BenchmarkRateLimiter_MultipleClients benchmarks multiple clients
func BenchmarkRateLimiter_MultipleClients(b *testing.B) {
	rl := NewRateLimiter(1000, 1000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		clientID := 0
		for pb.Next() {
			key := "client-" + string(rune('A'+clientID%26))
			rl.Allow(key)
			clientID++
		}
	})
}

// BenchmarkRateLimiter_GetStats benchmarks statistics retrieval
func BenchmarkRateLimiter_GetStats(b *testing.B) {
	rl := NewRateLimiter(100, 100)

	// Add some clients
	for i := 0; i < 10; i++ {
		rl.Allow("client-" + string(rune('0'+i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.GetStats()
	}
}

// BenchmarkRateLimiter_Reset benchmarks reset operation
func BenchmarkRateLimiter_Reset(b *testing.B) {
	rl := NewRateLimiter(100, 100)
	key := "bench-client"

	// Create limiter
	rl.Allow(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Reset(key)
	}
}

// Stress Test: TestRateLimiter_MemoryUsage tests memory under load
func TestRateLimiter_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")  // SKIP-OK: #short-mode
	}

	rl := NewRateLimiter(1000, 1000)

	// Create many limiters
	for i := 0; i < 1000; i++ {
		key := "client-" + string(rune('0'+i%100))
		rl.Allow(key)
	}

	stats := rl.GetStats()
	// Should not create too many limiters (at most 100 unique keys)
	assert.LessOrEqual(t, stats["total_limiters"], 100)
}
