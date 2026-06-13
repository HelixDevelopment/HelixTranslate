package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// CacheEntry represents a cached translation
type CacheEntry struct {
	Value      string
	Expiration time.Time
}

// Cache implements thread-safe translation caching
type Cache struct {
	mu        sync.RWMutex
	entries   map[string]CacheEntry
	ttl       time.Duration
	enabled   bool
	stop      chan struct{}
	stopOnce  sync.Once
	cleanupWG sync.WaitGroup
}

// NewCache creates a new cache
func NewCache(ttl time.Duration, enabled bool) *Cache {
	c := &Cache{
		entries: make(map[string]CacheEntry),
		ttl:     ttl,
		enabled: enabled,
		stop:    make(chan struct{}),
	}

	// Start cleanup goroutine
	if enabled {
		c.cleanupWG.Add(1)
		go c.cleanup()
	}

	return c
}

// Close stops the background cleanup goroutine and releases its resources.
// It is safe to call multiple times and from multiple goroutines. After Close
// the cache may still be read/written, but expired entries are no longer
// reaped in the background. Callers that create many Cache instances MUST call
// Close to avoid leaking one goroutine (and keeping the Cache alive) per
// instance.
func (c *Cache) Close() {
	c.stopOnce.Do(func() {
		close(c.stop)
	})
	c.cleanupWG.Wait()
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (string, bool) {
	if !c.enabled {
		return "", false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[c.hashKey(key)]
	if !ok {
		return "", false
	}

	// Check expiration
	if time.Now().After(entry.Expiration) {
		return "", false
	}

	return entry.Value, true
}

// Set stores a value in cache
func (c *Cache) Set(key, value string) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[c.hashKey(key)] = CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, c.hashKey(key))
}

// Clear removes all entries from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]CacheEntry)
}

// Size returns the number of entries in cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	validCount := 0
	expiredCount := 0
	now := time.Now()

	for _, entry := range c.entries {
		if now.After(entry.Expiration) {
			expiredCount++
		} else {
			validCount++
		}
	}

	return CacheStats{
		TotalEntries:   len(c.entries),
		ValidEntries:   validCount,
		ExpiredEntries: expiredCount,
		Enabled:        c.enabled,
		TTL:            c.ttl,
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries   int
	ValidEntries   int
	ExpiredEntries int
	Enabled        bool
	TTL            time.Duration
}

// hashKey creates a hash of the cache key
func (c *Cache) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// cleanup periodically removes expired entries until the cache is closed.
func (c *Cache) cleanup() {
	defer c.cleanupWG.Done()

	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.removeExpired()
		}
	}
}

// removeExpired removes expired entries from cache
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.Expiration) {
			delete(c.entries, key)
		}
	}
}
