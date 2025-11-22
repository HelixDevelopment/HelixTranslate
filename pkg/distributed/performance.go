package distributed

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// PerformanceConfig holds performance-related configuration
type PerformanceConfig struct {
	// Connection Pooling
	MaxConnectionsPerWorker int
	ConnectionIdleTimeout   time.Duration
	ConnectionMaxLifetime   time.Duration

	// Request Batching
	EnableBatching bool
	BatchSize      int
	BatchTimeout   time.Duration

	// Caching
	EnableResultCaching  bool
	CacheTTL             time.Duration
	CacheCleanupInterval time.Duration
	MaxCacheSize         int

	// Load Balancing
	LoadBalancingStrategy string // "round_robin", "least_loaded", "weighted"
	HealthCheckInterval   time.Duration

	// Circuit Breaker
	EnableCircuitBreaker bool
	FailureThreshold     int
	RecoveryTimeout      time.Duration
	SuccessThreshold     int

	// Metrics
	EnableMetrics   bool
	MetricsInterval time.Duration
}

// DefaultPerformanceConfig returns optimized default configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		MaxConnectionsPerWorker: 10,
		ConnectionIdleTimeout:   5 * time.Minute,
		ConnectionMaxLifetime:   30 * time.Minute,
		EnableBatching:          true,
		BatchSize:               10,
		BatchTimeout:            100 * time.Millisecond,
		EnableResultCaching:     true,
		CacheTTL:                10 * time.Minute,
		CacheCleanupInterval:    5 * time.Minute,
		MaxCacheSize:            10000,
		LoadBalancingStrategy:   "least_loaded",
		HealthCheckInterval:     30 * time.Second,
		EnableCircuitBreaker:    true,
		FailureThreshold:        5,
		RecoveryTimeout:         60 * time.Second,
		SuccessThreshold:        3,
		EnableMetrics:           true,
		MetricsInterval:         10 * time.Second,
	}
}

// ConnectionPool manages a pool of connections with performance optimizations
type ConnectionPool struct {
	connections map[string]*ConnectionPoolEntry
	mu          sync.RWMutex
	config      *PerformanceConfig
	security    *SecurityConfig
	auditor     *SecurityAuditor
}

// ConnectionPoolEntry represents a pooled connection
type ConnectionPoolEntry struct {
	Connection *SSHConnection
	LastUsed   time.Time
	CreatedAt  time.Time
	InUse      bool
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PerformanceConfig, security *SecurityConfig, auditor *SecurityAuditor) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(map[string]*ConnectionPoolEntry),
		config:      config,
		security:    security,
		auditor:     auditor,
	}

	// Start cleanup goroutine
	go pool.cleanup()

	return pool
}

// GetConnection gets a connection from the pool or creates a new one
func (cp *ConnectionPool) GetConnection(workerID string, worker *WorkerConfig) (*SSHConnection, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	key := cp.getConnectionKey(workerID)

	// Check if we have an available connection
	if entry, exists := cp.connections[key]; exists && !entry.InUse {
		// Check if connection is still valid
		if time.Since(entry.CreatedAt) < cp.config.ConnectionMaxLifetime &&
			time.Since(entry.LastUsed) < cp.config.ConnectionIdleTimeout {
			entry.InUse = true
			entry.LastUsed = time.Now()
			return entry.Connection, nil
		}
		// Connection is stale, remove it
		delete(cp.connections, key)
	}

	// Create new connection
	conn, err := cp.createConnection(worker)
	if err != nil {
		cp.auditor.LogConnectionAttempt(workerID, fmt.Sprintf("%s:%d", worker.SSH.Host, worker.SSH.Port), false, err.Error())
		return nil, err
	}

	entry := &ConnectionPoolEntry{
		Connection: conn,
		LastUsed:   time.Now(),
		CreatedAt:  time.Now(),
		InUse:      true,
	}

	cp.connections[key] = entry
	cp.auditor.LogConnectionAttempt(workerID, fmt.Sprintf("%s:%d", worker.SSH.Host, worker.SSH.Port), true, "")

	return conn, nil
}

// ReturnConnection returns a connection to the pool
func (cp *ConnectionPool) ReturnConnection(workerID string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	key := cp.getConnectionKey(workerID)
	if entry, exists := cp.connections[key]; exists {
		entry.InUse = false
		entry.LastUsed = time.Now()
	}
}

// RemoveConnection removes a connection from the pool
func (cp *ConnectionPool) RemoveConnection(workerID string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	key := cp.getConnectionKey(workerID)
	delete(cp.connections, key)
}

// createConnection creates a new SSH connection with security hardening
func (cp *ConnectionPool) createConnection(worker *WorkerConfig) (*SSHConnection, error) {
	// Validate network access
	address := fmt.Sprintf("%s:%d", worker.SSH.Host, worker.SSH.Port)
	if err := cp.security.ValidateNetworkAccess(address); err != nil {
		cp.auditor.LogNetworkAccess(address, false)
		return nil, fmt.Errorf("network access denied: %w", err)
	}
	cp.auditor.LogNetworkAccess(address, true)

	// Create SSH config with security hardening
	authMethods := []ssh.AuthMethod{}

	if worker.SSH.KeyFile != "" {
		key, err := ssh.ParsePrivateKey([]byte(worker.SSH.KeyFile))
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
		cp.auditor.LogAuthAttempt(worker.ID, worker.SSH.User, "public_key", true)
	}

	if worker.SSH.Password != "" {
		authMethods = append(authMethods, ssh.Password(worker.SSH.Password))
		cp.auditor.LogAuthAttempt(worker.ID, worker.SSH.User, "password", true)
	}

	sshConfig, err := cp.security.SecureSSHConfig(worker.SSH.User, authMethods)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure SSH config: %w", err)
	}

	// Create connection with timeout
	conn, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH dial failed: %w", err)
	}

	return &SSHConnection{
		Config:    worker,
		Client:    conn,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}, nil
}

// getConnectionKey generates a unique key for connection pooling
func (cp *ConnectionPool) getConnectionKey(workerID string) string {
	return fmt.Sprintf("worker:%s", workerID)
}

// cleanup periodically removes idle and expired connections
func (cp *ConnectionPool) cleanup() {
	ticker := time.NewTicker(cp.config.CacheCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.mu.Lock()
			now := time.Now()

			for key, entry := range cp.connections {
				// Remove connections that are:
				// 1. Not in use and idle for too long
				// 2. Exceeded maximum lifetime
				if (!entry.InUse && now.Sub(entry.LastUsed) > cp.config.ConnectionIdleTimeout) ||
					now.Sub(entry.CreatedAt) > cp.config.ConnectionMaxLifetime {
					entry.Connection.Close()
					delete(cp.connections, key)
				}
			}
			cp.mu.Unlock()
		}
	}
}

// GetPoolStats returns connection pool statistics
func (cp *ConnectionPool) GetPoolStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	total := len(cp.connections)
	inUse := 0
	idle := 0

	for _, entry := range cp.connections {
		if entry.InUse {
			inUse++
		} else {
			idle++
		}
	}

	return map[string]interface{}{
		"total_connections":  total,
		"active_connections": inUse,
		"idle_connections":   idle,
		"max_per_worker":     cp.config.MaxConnectionsPerWorker,
	}
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Value     string
	ExpiresAt time.Time
}

// ResultCache provides caching for translation results
type ResultCache struct {
	cache   map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
}

// NewResultCache creates a new result cache
func NewResultCache(config *PerformanceConfig) *ResultCache {
	rc := &ResultCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: config.MaxCacheSize,
		ttl:     config.CacheTTL,
	}

	// Start cleanup goroutine
	go rc.cleanup(config.CacheCleanupInterval)

	return rc
}

// Get retrieves a cached result
func (rc *ResultCache) Get(key string) (string, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, found := rc.cache[key]
	if !found {
		return "", false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return "", false
	}

	return entry.Value, true
}

// Set stores a result in the cache
func (rc *ResultCache) Set(key, value string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Check cache size limit
	if len(rc.cache) >= rc.maxSize {
		// Remove expired entries first
		rc.removeExpired()

		// If still at limit, remove oldest entry
		if len(rc.cache) >= rc.maxSize {
			rc.removeOldest()
		}
	}

	rc.cache[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(rc.ttl),
	}
}

// removeExpired removes expired cache entries
func (rc *ResultCache) removeExpired() {
	now := time.Now()
	for key, entry := range rc.cache {
		if now.After(entry.ExpiresAt) {
			delete(rc.cache, key)
		}
	}
}

// removeOldest removes the oldest cache entry
func (rc *ResultCache) removeOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range rc.cache {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(rc.cache, oldestKey)
	}
}

// cleanup periodically removes expired entries
func (rc *ResultCache) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rc.mu.Lock()
			rc.removeExpired()
			rc.mu.Unlock()
		}
	}
}

// generateCacheKey generates a cache key for translation requests
func (rc *ResultCache) GenerateCacheKey(text, contextHint, provider, model string) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", text, contextHint, provider, model)))
	return fmt.Sprintf("%x", hash)
}

// CircuitBreaker implements circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	failureThreshold int
	recoveryTimeout  time.Duration
	successThreshold int

	failures    int
	lastFailure time.Time
	successes   int
	state       CircuitState
	mu          sync.RWMutex
}

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, recoveryTimeout time.Duration, successThreshold int) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
		successThreshold: successThreshold,
		state:            StateClosed,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateOpen:
		if time.Since(cb.lastFailure) < cb.recoveryTimeout {
			return fmt.Errorf("circuit breaker is open")
		}
		cb.state = StateHalfOpen
		cb.successes = 0
		fallthrough

	case StateHalfOpen:
		err := fn()
		if err != nil {
			cb.failures++
			cb.lastFailure = time.Now()
			cb.state = StateOpen
			return err
		}

		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = StateClosed
			cb.failures = 0
		}
		return nil

	case StateClosed:
		err := fn()
		if err != nil {
			cb.failures++
			if cb.failures >= cb.failureThreshold {
				cb.state = StateOpen
				cb.lastFailure = time.Now()
			}
			return err
		}
		cb.failures = 0
		return nil
	}

	return fmt.Errorf("invalid circuit breaker state")
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// BatchProcessor handles request batching for improved performance
type BatchProcessor struct {
	batchSize int
	timeout   time.Duration
	processFn func([]interface{}) error
	batches   map[string]*Batch
	mu        sync.RWMutex
}

// Batch represents a batch of requests
type Batch struct {
	ID        string
	Requests  []interface{}
	CreatedAt time.Time
	Timer     *time.Timer
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, timeout time.Duration, processFn func([]interface{}) error) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		timeout:   timeout,
		processFn: processFn,
		batches:   make(map[string]*Batch),
	}
}

// AddRequest adds a request to be batched
func (bp *BatchProcessor) AddRequest(batchID string, request interface{}) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	batch, exists := bp.batches[batchID]
	if !exists {
		batch = &Batch{
			ID:        batchID,
			Requests:  make([]interface{}, 0, bp.batchSize),
			CreatedAt: time.Now(),
		}
		bp.batches[batchID] = batch
	}

	batch.Requests = append(batch.Requests, request)

	// If batch is full, process it immediately
	if len(batch.Requests) >= bp.batchSize {
		return bp.processBatch(batchID)
	}

	// Set timeout if not already set
	if batch.Timer == nil {
		batch.Timer = time.AfterFunc(bp.timeout, func() {
			bp.mu.Lock()
			defer bp.mu.Unlock()
			bp.processBatch(batchID)
		})
	}

	return nil
}

// processBatch processes a batch of requests
func (bp *BatchProcessor) processBatch(batchID string) error {
	batch, exists := bp.batches[batchID]
	if !exists {
		return nil
	}

	// Cancel timer if it exists
	if batch.Timer != nil {
		batch.Timer.Stop()
	}

	// Process the batch
	err := bp.processFn(batch.Requests)

	// Remove the batch
	delete(bp.batches, batchID)

	return err
}

// FlushAll flushes all pending batches
func (bp *BatchProcessor) FlushAll() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	var lastErr error
	for batchID := range bp.batches {
		if err := bp.processBatch(batchID); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
