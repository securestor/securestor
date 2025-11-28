package scanner

import (
	"runtime"
	"sync"
	"time"
)

// PerformanceConfig holds performance optimization settings
type PerformanceConfig struct {
	// Parallel processing
	MaxConcurrentScans int           `json:"max_concurrent_scans"`
	ScanTimeout        time.Duration `json:"scan_timeout"`

	// Caching
	EnableResultCache bool          `json:"enable_result_cache"`
	CacheTTL          time.Duration `json:"cache_ttl"`

	// Circuit breaker
	CircuitBreakerEnabled bool `json:"circuit_breaker_enabled"`
	FailureThreshold      int  `json:"failure_threshold"`

	// Memory management
	EnableGCOptimization bool  `json:"enable_gc_optimization"`
	GCThresholdPercent   int   `json:"gc_threshold_percent"`
	MaxMemoryMB          int64 `json:"max_memory_mb"`

	// Retry logic
	MaxRetries   int           `json:"max_retries"`
	RetryBackoff time.Duration `json:"retry_backoff"`
}

// DefaultPerformanceConfig returns optimized default settings
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		MaxConcurrentScans:    runtime.NumCPU() * 2,
		ScanTimeout:           10 * time.Minute,
		EnableResultCache:     true,
		CacheTTL:              30 * time.Minute,
		CircuitBreakerEnabled: true,
		FailureThreshold:      5,
		EnableGCOptimization:  true,
		GCThresholdPercent:    80,
		MaxMemoryMB:           1024,
		MaxRetries:            3,
		RetryBackoff:          2 * time.Second,
	}
}

// PerformanceEnhancer adds scalable performance features to existing orchestrator
type PerformanceEnhancer struct {
	config          *PerformanceConfig
	resultCache     *sync.Map     // cacheKey -> *CachedResult
	circuitBreakers *sync.Map     // scannerName -> *CircuitBreaker
	scanSemaphore   chan struct{} // Limits concurrent scans
	metrics         *PerformanceMetrics
	mu              sync.RWMutex
}

// CachedResult holds cached scan results with TTL
type CachedResult struct {
	Result   *ScanResult
	CachedAt time.Time
	TTL      time.Duration
}

// IsExpired checks if cached result has expired
func (cr *CachedResult) IsExpired() bool {
	return time.Since(cr.CachedAt) > cr.TTL
}

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	TotalScans         int64         `json:"total_scans"`
	CacheHits          int64         `json:"cache_hits"`
	CacheMisses        int64         `json:"cache_misses"`
	FailedScans        int64         `json:"failed_scans"`
	AverageDuration    time.Duration `json:"average_duration"`
	ParallelScans      int64         `json:"parallel_scans"`
	CircuitBreakEvents int64         `json:"circuit_break_events"`
	mu                 sync.RWMutex
}

// NewPerformanceEnhancer creates performance enhancement layer
func NewPerformanceEnhancer(config *PerformanceConfig) *PerformanceEnhancer {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	return &PerformanceEnhancer{
		config:          config,
		resultCache:     &sync.Map{},
		circuitBreakers: &sync.Map{},
		scanSemaphore:   make(chan struct{}, config.MaxConcurrentScans),
		metrics:         &PerformanceMetrics{},
	}
}

// CircuitBreakerState represents circuit breaker states
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	threshold    int
	timeout      time.Duration
	mu           sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:     CircuitBreakerClosed,
		threshold: threshold,
		timeout:   timeout,
	}
}

// CanExecute checks if execution is allowed
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		if time.Since(cb.lastFailTime) > cb.timeout {
			cb.state = CircuitBreakerHalfOpen
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records successful execution
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitBreakerClosed
}

// RecordFailure records failed execution
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = CircuitBreakerOpen
	}
}
