package scanner

import (
	"context"
	"sync"
	"time"
)

// HealthChecker monitors plugin health status
type HealthChecker struct {
	plugins     map[string]ScannerPlugin
	healthCache map[string]*HealthStatus
	checkPeriod time.Duration
	mu          sync.RWMutex
	stopCh      chan struct{}
	logger      Logger
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		plugins:     make(map[string]ScannerPlugin),
		healthCache: make(map[string]*HealthStatus),
		checkPeriod: 30 * time.Second,
		stopCh:      make(chan struct{}),
	}
}

// SetLogger sets the logger for the health checker
func (hc *HealthChecker) SetLogger(logger Logger) {
	hc.logger = logger
}

// RegisterPlugin registers a plugin for health monitoring
func (hc *HealthChecker) RegisterPlugin(plugin ScannerPlugin) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	id := plugin.GetMetadata().ID
	hc.plugins[id] = plugin

	// Initialize health status
	health := plugin.HealthCheck()
	hc.healthCache[id] = &health
}

// UnregisterPlugin removes a plugin from health monitoring
func (hc *HealthChecker) UnregisterPlugin(pluginID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	delete(hc.plugins, pluginID)
	delete(hc.healthCache, pluginID)
}

// GetHealth returns the cached health status for a plugin
func (hc *HealthChecker) GetHealth(pluginID string) (*HealthStatus, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	health, exists := hc.healthCache[pluginID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	healthCopy := *health
	return &healthCopy, true
}

// GetAllHealth returns health status for all registered plugins
func (hc *HealthChecker) GetAllHealth() map[string]*HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result := make(map[string]*HealthStatus)
	for id, health := range hc.healthCache {
		healthCopy := *health
		result[id] = &healthCopy
	}

	return result
}

// GetHealthyPlugins returns IDs of all healthy plugins
func (hc *HealthChecker) GetHealthyPlugins() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var healthy []string
	for id, health := range hc.healthCache {
		if health.Status == HealthHealthy {
			healthy = append(healthy, id)
		}
	}

	return healthy
}

// StartMonitoring starts the health monitoring routine
func (hc *HealthChecker) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(hc.checkPeriod)
	defer ticker.Stop()

	if hc.logger != nil {
		hc.logger.Printf("[HEALTH_CHECKER] Starting health monitoring with %d second intervals", int(hc.checkPeriod.Seconds()))
	}

	for {
		select {
		case <-ctx.Done():
			if hc.logger != nil {
				hc.logger.Printf("[HEALTH_CHECKER] Stopping health monitoring")
			}
			return
		case <-hc.stopCh:
			if hc.logger != nil {
				hc.logger.Printf("[HEALTH_CHECKER] Health monitoring stopped")
			}
			return
		case <-ticker.C:
			hc.performHealthChecks()
		}
	}
}

// StopMonitoring stops the health monitoring routine
func (hc *HealthChecker) StopMonitoring() {
	select {
	case hc.stopCh <- struct{}{}:
	default:
		// Channel already closed or full
	}
}

// performHealthChecks checks the health of all registered plugins
func (hc *HealthChecker) performHealthChecks() {
	hc.mu.RLock()
	plugins := make(map[string]ScannerPlugin)
	for id, plugin := range hc.plugins {
		plugins[id] = plugin
	}
	hc.mu.RUnlock()

	for id, plugin := range plugins {
		go hc.checkPluginHealth(id, plugin)
	}
}

// checkPluginHealth performs a health check for a specific plugin
func (hc *HealthChecker) checkPluginHealth(pluginID string, plugin ScannerPlugin) {
	defer func() {
		if r := recover(); r != nil {
			// Handle panic during health check
			hc.mu.Lock()
			hc.healthCache[pluginID] = &HealthStatus{
				Status:    HealthUnhealthy,
				Message:   "Health check panicked",
				CheckedAt: time.Now(),
				Details:   map[string]interface{}{"panic": r},
			}
			hc.mu.Unlock()

			if hc.logger != nil {
				hc.logger.Printf("[HEALTH_CHECKER] Plugin %s health check panicked: %v", pluginID, r)
			}
		}
	}()

	// Perform health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	healthCh := make(chan HealthStatus, 1)
	go func() {
		health := plugin.HealthCheck()
		select {
		case healthCh <- health:
		case <-ctx.Done():
		}
	}()

	var health HealthStatus
	select {
	case health = <-healthCh:
		// Health check completed successfully
	case <-ctx.Done():
		// Health check timed out
		health = HealthStatus{
			Status:    HealthUnhealthy,
			Message:   "Health check timed out",
			CheckedAt: time.Now(),
			Details:   map[string]interface{}{"timeout": "10s"},
		}
	}

	// Update cached health status
	hc.mu.Lock()
	hc.healthCache[pluginID] = &health
	hc.mu.Unlock()

	// Log status changes
	if hc.logger != nil {
		if health.Status != HealthHealthy {
			hc.logger.Printf("[HEALTH_CHECKER] Plugin %s is %s: %s", pluginID, health.Status, health.Message)
		}
	}
}

// SetCheckPeriod sets the health check interval
func (hc *HealthChecker) SetCheckPeriod(period time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checkPeriod = period
}

// GetStatistics returns health monitoring statistics
func (hc *HealthChecker) GetStatistics() HealthStatistics {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	stats := HealthStatistics{
		TotalPlugins: len(hc.plugins),
		CheckPeriod:  hc.checkPeriod,
		LastCheck:    time.Now(),
	}

	for _, health := range hc.healthCache {
		switch health.Status {
		case HealthHealthy:
			stats.HealthyCount++
		case HealthUnhealthy:
			stats.UnhealthyCount++
		case HealthDegraded:
			stats.DegradedCount++
		default:
			stats.UnknownCount++
		}
	}

	return stats
}

// HealthStatistics provides monitoring statistics
type HealthStatistics struct {
	TotalPlugins   int           `json:"total_plugins"`
	HealthyCount   int           `json:"healthy_count"`
	UnhealthyCount int           `json:"unhealthy_count"`
	DegradedCount  int           `json:"degraded_count"`
	UnknownCount   int           `json:"unknown_count"`
	CheckPeriod    time.Duration `json:"check_period"`
	LastCheck      time.Time     `json:"last_check"`
}
