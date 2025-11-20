package security

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rps      int
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed for a given key
func (rl *RateLimiter) Allow(key string) bool {
	limiter := rl.getLimiter(key)
	return limiter.Allow()
}

// Wait waits until a request is allowed
func (rl *RateLimiter) Wait(key string) {
	limiter := rl.getLimiter(key)
	limiter.Wait(nil)
}

// getLimiter gets or creates a limiter for a key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := rl.limiters[key]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters[key] = limiter
	return limiter
}

// cleanup removes old limiters
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		// In a real implementation, you'd track last access time
		// For now, we keep all limiters
		rl.mu.Unlock()
	}
}

// Reset resets the limiter for a key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, key)
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"total_limiters": len(rl.limiters),
		"rps":            rl.rps,
		"burst":          rl.burst,
	}
}
