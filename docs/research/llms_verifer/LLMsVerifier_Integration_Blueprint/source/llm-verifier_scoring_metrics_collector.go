package scoring

import (
	"sync"
	"time"
)

// MetricsCollector collects and manages system metrics
type MetricsCollector struct {
	config MonitoringConfig
	
	// Score metrics
	scoreCalculationsTotal int64
	scoreCalculationErrors int64
	scoreChanges           []ScoreChangeMetric
	
	// API metrics
	apiRequestsTotal  int64
	apiRequestsFailed int64
	apiResponseTimes  []time.Duration
	
	// Database metrics
	dbOperationsTotal  int64
	dbOperationsFailed int64
	dbLatencies        []time.Duration
	
	// Performance metrics
	memoryUsage    []MemoryMetric
	cpuUsage       []CPUMetric
	diskUsage      []DiskMetric
	
	mu sync.RWMutex
}

// ScoreChangeMetric represents a score change event
type ScoreChangeMetric struct {
	ModelID   string    `json:"model_id"`
	Change    float64   `json:"change"`
	Timestamp time.Time `json:"timestamp"`
}

// MemoryMetric represents memory usage
type MemoryMetric struct {
	Used      uint64    `json:"used"`
	Total     uint64    `json:"total"`
	Timestamp time.Time `json:"timestamp"`
}

// CPUMetric represents CPU usage
type CPUMetric struct {
	Usage     float64   `json:"usage"`
	Timestamp time.Time `json:"timestamp"`
}

// DiskMetric represents disk usage
type DiskMetric struct {
	Used      uint64    `json:"used"`
	Total     uint64    `json:"total"`
	Timestamp time.Time `json:"timestamp"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config MonitoringConfig) MetricsCollector {
	return MetricsCollector{
		config:         config,
		scoreChanges:   make([]ScoreChangeMetric, 0, 1000),
		apiResponseTimes: make([]time.Duration, 0, 1000),
		dbLatencies:    make([]time.Duration, 0, 1000),
		memoryUsage:    make([]MemoryMetric, 0, 1000),
		cpuUsage:       make([]CPUMetric, 0, 1000),
		diskUsage:      make([]DiskMetric, 0, 1000),
	}
}

// RecordScoreCalculation records a score calculation event
func (mc *MetricsCollector) RecordScoreCalculation(success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.scoreCalculationsTotal++
	if !success {
		mc.scoreCalculationErrors++
	}
}

// RecordScoreChange records a score change event
func (mc *MetricsCollector) RecordScoreChange(modelID string, change float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metric := ScoreChangeMetric{
		ModelID:   modelID,
		Change:    change,
		Timestamp: time.Now(),
	}

	mc.scoreChanges = append(mc.scoreChanges, metric)
	
	// Keep only recent metrics
	mc.trimScoreChanges()
}

// RecordAPIPerformance records API performance metrics
func (mc *MetricsCollector) RecordAPIPerformance(apiName string, responseTime time.Duration, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.apiRequestsTotal++
	if !success {
		mc.apiRequestsFailed++
	}

	mc.apiResponseTimes = append(mc.apiResponseTimes, responseTime)
	
	// Keep only recent metrics
	mc.trimAPIResponseTimes()
}

// RecordDatabasePerformance records database performance metrics
func (mc *MetricsCollector) RecordDatabasePerformance(operation string, latency time.Duration, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.dbOperationsTotal++
	if !success {
		mc.dbOperationsFailed++
	}

	mc.dbLatencies = append(mc.dbLatencies, latency)
	
	// Keep only recent metrics
	mc.trimDatabaseLatencies()
}

// RecordSystemPerformance records system performance metrics
func (mc *MetricsCollector) RecordSystemPerformance(memoryUsed, memoryTotal uint64, cpuUsage float64, diskUsed, diskTotal uint64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()

	if memoryTotal > 0 {
		mc.memoryUsage = append(mc.memoryUsage, MemoryMetric{
			Used:      memoryUsed,
			Total:     memoryTotal,
			Timestamp: now,
		})
		mc.trimMemoryMetrics()
	}

	mc.cpuUsage = append(mc.cpuUsage, CPUMetric{
		Usage:     cpuUsage,
		Timestamp: now,
	})
	mc.trimCPUMetrics()

	if diskTotal > 0 {
		mc.diskUsage = append(mc.diskUsage, DiskMetric{
			Used:      diskUsed,
			Total:     diskTotal,
			Timestamp: now,
		})
		mc.trimDiskMetrics()
	}
}

// GetCurrentMetrics returns current system metrics
func (mc *MetricsCollector) GetCurrentMetrics() CurrentMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return CurrentMetrics{
		ScoreCalculationsTotal:   mc.scoreCalculationsTotal,
		ScoreCalculationErrors:   mc.scoreCalculationErrors,
		AverageScoreChange:       mc.calculateAverageScoreChange(),
		APIRequestsTotal:         mc.apiRequestsTotal,
		APIRequestsFailed:        mc.apiRequestsFailed,
		AverageAPIResponseTime:   mc.calculateAverageAPIResponseTime(),
		DatabaseOperationsTotal:  mc.dbOperationsTotal,
		DatabaseOperationsFailed: mc.dbOperationsFailed,
		AverageDatabaseLatency:   mc.calculateAverageDatabaseLatency(),
		CurrentMemoryUsage:       mc.getCurrentMemoryUsage(),
		CurrentCPUUsage:          mc.getCurrentCPUUsage(),
		CurrentDiskUsage:         mc.getCurrentDiskUsage(),
		LastUpdated:              time.Now(),
	}
}

// GetSummary returns a summary of metrics over a time period
func (mc *MetricsCollector) GetSummary(duration time.Duration) MetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	cutoff := time.Now().Add(-duration)

	// Calculate score change statistics
	scoreChanges := mc.filterScoreChanges(cutoff)
	avgScoreChange := 0.0
	if len(scoreChanges) > 0 {
		sum := 0.0
		for _, sc := range scoreChanges {
			sum += sc.Change
		}
		avgScoreChange = sum / float64(len(scoreChanges))
	}

	// Calculate API performance statistics
	apiResponseTimes := mc.filterAPIResponseTimes(cutoff)
	avgAPIResponseTime := time.Duration(0)
	if len(apiResponseTimes) > 0 {
		sum := time.Duration(0)
		for _, rt := range apiResponseTimes {
			sum += rt
		}
		avgAPIResponseTime = sum / time.Duration(len(apiResponseTimes))
	}

	// Calculate database performance statistics
	dbLatencies := mc.filterDatabaseLatencies(cutoff)
	avgDBLatency := time.Duration(0)
	if len(dbLatencies) > 0 {
		sum := time.Duration(0)
		for _, lat := range dbLatencies {
			sum += lat
		}
		avgDBLatency = sum / time.Duration(len(dbLatencies))
	}

	return MetricsSummary{
		TotalScoreCalculations:   mc.scoreCalculationsTotal,
		AverageScoreChange:       avgScoreChange,
		APIRequestsTotal:         mc.apiRequestsTotal,
		APIRequestsFailed:        mc.apiRequestsFailed,
		AverageAPIResponseTime:   avgAPIResponseTime,
		DatabaseOperationsTotal:  mc.dbOperationsTotal,
		DatabaseOperationsFailed: mc.dbOperationsFailed,
		AverageDatabaseLatency:   avgDBLatency,
		LastUpdated:              time.Now(),
	}
}

// CleanupOldMetrics removes metrics older than the specified duration
func (mc *MetricsCollector) CleanupOldMetrics(maxAge time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	// Cleanup score changes
	mc.scoreChanges = mc.filterScoreChanges(cutoff)

	// Cleanup API response times
	mc.apiResponseTimes = mc.filterAPIResponseTimes(cutoff)

	// Cleanup database latencies
	mc.dbLatencies = mc.filterDatabaseLatencies(cutoff)

	// Cleanup system metrics
	mc.memoryUsage = mc.filterMemoryMetrics(cutoff)
	mc.cpuUsage = mc.filterCPUMetrics(cutoff)
	mc.diskUsage = mc.filterDiskMetrics(cutoff)
}

// Internal calculation methods

func (mc *MetricsCollector) calculateAverageScoreChange() float64 {
	if len(mc.scoreChanges) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, sc := range mc.scoreChanges {
		sum += sc.Change
	}
	return sum / float64(len(mc.scoreChanges))
}

func (mc *MetricsCollector) calculateAverageAPIResponseTime() time.Duration {
	if len(mc.apiResponseTimes) == 0 {
		return 0
	}

	sum := time.Duration(0)
	for _, rt := range mc.apiResponseTimes {
		sum += rt
	}
	return sum / time.Duration(len(mc.apiResponseTimes))
}

func (mc *MetricsCollector) calculateAverageDatabaseLatency() time.Duration {
	if len(mc.dbLatencies) == 0 {
		return 0
	}

	sum := time.Duration(0)
	for _, lat := range mc.dbLatencies {
		sum += lat
	}
	return sum / time.Duration(len(mc.dbLatencies))
}

func (mc *MetricsCollector) getCurrentMemoryUsage() float64 {
	if len(mc.memoryUsage) == 0 {
		return 0.0
	}

	latest := mc.memoryUsage[len(mc.memoryUsage)-1]
	if latest.Total == 0 {
		return 0.0
	}

	return float64(latest.Used) / float64(latest.Total) * 100.0
}

func (mc *MetricsCollector) getCurrentCPUUsage() float64 {
	if len(mc.cpuUsage) == 0 {
		return 0.0
	}

	return mc.cpuUsage[len(mc.cpuUsage)-1].Usage
}

func (mc *MetricsCollector) getCurrentDiskUsage() float64 {
	if len(mc.diskUsage) == 0 {
		return 0.0
	}

	latest := mc.diskUsage[len(mc.diskUsage)-1]
	if latest.Total == 0 {
		return 0.0
	}

	return float64(latest.Used) / float64(latest.Total) * 100.0
}

// Filtering methods

func (mc *MetricsCollector) filterScoreChanges(cutoff time.Time) []ScoreChangeMetric {
	filtered := make([]ScoreChangeMetric, 0)
	for _, sc := range mc.scoreChanges {
		if sc.Timestamp.After(cutoff) {
			filtered = append(filtered, sc)
		}
	}
	return filtered
}

func (mc *MetricsCollector) filterAPIResponseTimes(cutoff time.Time) []time.Duration {
	// Note: We don't have timestamps for API response times in this simple implementation
	// In a real implementation, you would store them with timestamps
	return mc.apiResponseTimes
}

func (mc *MetricsCollector) filterDatabaseLatencies(cutoff time.Time) []time.Duration {
	// Note: We don't have timestamps for database latencies in this simple implementation
	// In a real implementation, you would store them with timestamps
	return mc.dbLatencies
}

func (mc *MetricsCollector) filterMemoryMetrics(cutoff time.Time) []MemoryMetric {
	filtered := make([]MemoryMetric, 0)
	for _, mm := range mc.memoryUsage {
		if mm.Timestamp.After(cutoff) {
			filtered = append(filtered, mm)
		}
	}
	return filtered
}

func (mc *MetricsCollector) filterCPUMetrics(cutoff time.Time) []CPUMetric {
	filtered := make([]CPUMetric, 0)
	for _, cm := range mc.cpuUsage {
		if cm.Timestamp.After(cutoff) {
			filtered = append(filtered, cm)
		}
	}
	return filtered
}

func (mc *MetricsCollector) filterDiskMetrics(cutoff time.Time) []DiskMetric {
	filtered := make([]DiskMetric, 0)
	for _, dm := range mc.diskUsage {
		if dm.Timestamp.After(cutoff) {
			filtered = append(filtered, dm)
		}
	}
	return filtered
}

// Trimming methods to keep memory usage reasonable

func (mc *MetricsCollector) trimScoreChanges() {
	maxMetrics := 10000 // Keep last 10,000 score changes
	if len(mc.scoreChanges) > maxMetrics {
		mc.scoreChanges = mc.scoreChanges[len(mc.scoreChanges)-maxMetrics:]
	}
}

func (mc *MetricsCollector) trimAPIResponseTimes() {
	maxMetrics := 5000 // Keep last 5,000 API response times
	if len(mc.apiResponseTimes) > maxMetrics {
		mc.apiResponseTimes = mc.apiResponseTimes[len(mc.apiResponseTimes)-maxMetrics:]
	}
}

func (mc *MetricsCollector) trimDatabaseLatencies() {
	maxMetrics := 5000 // Keep last 5,000 database latencies
	if len(mc.dbLatencies) > maxMetrics {
		mc.dbLatencies = mc.dbLatencies[len(mc.dbLatencies)-maxMetrics:]
	}
}

func (mc *MetricsCollector) trimMemoryMetrics() {
	maxMetrics := 1000 // Keep last 1,000 memory metrics
	if len(mc.memoryUsage) > maxMetrics {
		mc.memoryUsage = mc.memoryUsage[len(mc.memoryUsage)-maxMetrics:]
	}
}

func (mc *MetricsCollector) trimCPUMetrics() {
	maxMetrics := 1000 // Keep last 1,000 CPU metrics
	if len(mc.cpuUsage) > maxMetrics {
		mc.cpuUsage = mc.cpuUsage[len(mc.cpuUsage)-maxMetrics:]
	}
}

func (mc *MetricsCollector) trimDiskMetrics() {
	maxMetrics := 1000 // Keep last 1,000 disk metrics
	if len(mc.diskUsage) > maxMetrics {
		mc.diskUsage = mc.diskUsage[len(mc.diskUsage)-maxMetrics:]
	}
}

// CurrentMetrics represents current system metrics
type CurrentMetrics struct {
	ScoreCalculationsTotal   int64         `json:"score_calculations_total"`
	ScoreCalculationErrors   int64         `json:"score_calculation_errors"`
	AverageScoreChange       float64       `json:"average_score_change"`
	APIRequestsTotal         int64         `json:"api_requests_total"`
	APIRequestsFailed        int64         `json:"api_requests_failed"`
	AverageAPIResponseTime   time.Duration `json:"average_api_response_time"`
	DatabaseOperationsTotal  int64         `json:"database_operations_total"`
	DatabaseOperationsFailed int64         `json:"database_operations_failed"`
	AverageDatabaseLatency   time.Duration `json:"average_database_latency"`
	CurrentMemoryUsage       float64       `json:"current_memory_usage"`
	CurrentCPUUsage          float64       `json:"current_cpu_usage"`
	CurrentDiskUsage         float64       `json:"current_disk_usage"`
	LastUpdated              time.Time     `json:"last_updated"`
}

// GetScoreCalculationRate returns the score calculation rate per minute
func (mc *MetricsCollector) GetScoreCalculationRate() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// This is a simplified calculation
	// In a real implementation, you would track this over time
	return float64(mc.scoreCalculationsTotal) / (24 * 60) // Per day average
}

// GetAPIErrorRate returns the API error rate as a percentage
func (mc *MetricsCollector) GetAPIErrorRate() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.apiRequestsTotal == 0 {
		return 0.0
	}

	return float64(mc.apiRequestsFailed) / float64(mc.apiRequestsTotal) * 100.0
}

// GetDatabaseErrorRate returns the database error rate as a percentage
func (mc *MetricsCollector) GetDatabaseErrorRate() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.dbOperationsTotal == 0 {
		return 0.0
	}

	return float64(mc.dbOperationsFailed) / float64(mc.dbOperationsTotal) * 100.0
}

// Reset resets all metrics (use with caution)
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.scoreCalculationsTotal = 0
	mc.scoreCalculationErrors = 0
	mc.scoreChanges = mc.scoreChanges[:0]
	mc.apiRequestsTotal = 0
	mc.apiRequestsFailed = 0
	mc.apiResponseTimes = mc.apiResponseTimes[:0]
	mc.dbOperationsTotal = 0
	mc.dbOperationsFailed = 0
	mc.dbLatencies = mc.dbLatencies[:0]
	mc.memoryUsage = mc.memoryUsage[:0]
	mc.cpuUsage = mc.cpuUsage[:0]
	mc.diskUsage = mc.diskUsage[:0]
}