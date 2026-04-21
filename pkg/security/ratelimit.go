package security

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements rate limiting
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	lastUsed map[string]time.Time
	rps      int
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		lastUsed: make(map[string]time.Time),
		rps:      rps,
		burst:    burst,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed for a given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Update last used time
	rl.lastUsed[key] = time.Now()

	limiter := rl.getLimiterUnsafe(key)
	return limiter.Allow()
}

// Wait waits until a request is allowed
func (rl *RateLimiter) Wait(key string) {
	rl.mu.Lock()

	// Update last used time
	rl.lastUsed[key] = time.Now()

	limiter := rl.getLimiterUnsafe(key)
	rl.mu.Unlock()

	limiter.Wait(context.Background())
}

// getLimiterUnsafe gets or creates a limiter for a key (caller must hold lock)
func (rl *RateLimiter) getLimiterUnsafe(key string) *rate.Limiter {
	limiter, exists := rl.limiters[key]
	if exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters[key] = limiter
	return limiter
}

// getLimiter gets or creates a limiter for a key
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Update last used time
	rl.lastUsed[key] = time.Now()

	return rl.getLimiterUnsafe(key)
}

// cleanup removes old limiters
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()

		// Remove limiters not used in the last hour
		for key, lastUsed := range rl.lastUsed {
			if now.Sub(lastUsed) > time.Hour {
				delete(rl.limiters, key)
				delete(rl.lastUsed, key)
			}
		}

		rl.mu.Unlock()
	}
}

// Reset resets the limiter for a key
func (rl *RateLimiter) Reset(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, key)
	delete(rl.lastUsed, key)
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
