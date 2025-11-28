package health

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/securestor/securestor/internal/config"
	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/replicate"
)

// HealthStatus represents overall system health
type HealthStatus struct {
	Timestamp        time.Time                 `json:"timestamp"`
	Overall          string                    `json:"overall"` // "healthy", "degraded", "unhealthy"
	Database         ComponentHealth           `json:"database"`
	Cache            ComponentHealth           `json:"cache"`
	Storage          ComponentHealth           `json:"storage"`
	Replication      ComponentHealth           `json:"replication"`
	ReplicationNodes map[string]NodeHealthInfo `json:"replication_nodes,omitempty"`
	ReadinessChecks  map[string]bool           `json:"readiness_checks,omitempty"`
	ResponseTime     string                    `json:"response_time"`
}

// ComponentHealth represents health of a system component
type ComponentHealth struct {
	Status    string    `json:"status"` // "healthy", "degraded", "unhealthy"
	Message   string    `json:"message"`
	LastCheck time.Time `json:"last_check"`
}

// NodeHealthInfo represents health of a storage node
type NodeHealthInfo struct {
	Status       string    `json:"status"`
	FailureCount int       `json:"failure_count"`
	LastCheck    time.Time `json:"last_check"`
}

// HealthChecker performs comprehensive health checks
type HealthChecker struct {
	db     *sql.DB
	logger *logger.Logger
	mutex  sync.RWMutex
	cache  *HealthStatus
}

var (
	healthChecker *HealthChecker
	once          sync.Once
	log           *logger.Logger
)

// InitHealthChecker initializes the health checker
func InitHealthChecker(database *sql.DB, l *logger.Logger) *HealthChecker {
	once.Do(func() {
		log = l
		healthChecker = &HealthChecker{
			db:     database,
			logger: l,
		}

		// Start periodic health check
		go healthChecker.startPeriodicCheck()
	})

	return healthChecker
}

// GetInstance returns the singleton health checker
func GetInstance() *HealthChecker {
	return healthChecker
}

// GetHealthStatus returns current health status
func (hc *HealthChecker) GetHealthStatus() *HealthStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return hc.checkHealth(ctx)
}

// checkHealth performs comprehensive health checks
func (hc *HealthChecker) checkHealth(ctx context.Context) *HealthStatus {
	start := time.Now()

	status := &HealthStatus{
		Timestamp:        time.Now(),
		ReplicationNodes: make(map[string]NodeHealthInfo),
		ReadinessChecks:  make(map[string]bool),
	}

	var wg sync.WaitGroup

	// Check database
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.Database = hc.checkDatabase(ctx)
	}()

	// Check cache
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.Cache = hc.checkCache(ctx)
	}()

	// Check storage replication
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.Replication, status.ReplicationNodes = hc.checkReplication(ctx)
	}()

	wg.Wait()

	// Determine overall status
	status.Overall = hc.determineOverallStatus(status)
	status.ResponseTime = time.Since(start).String()

	// Cache the status
	hc.mutex.Lock()
	hc.cache = status
	hc.mutex.Unlock()

	return status
}

// checkDatabase checks database connectivity and replication lag
func (hc *HealthChecker) checkDatabase(ctx context.Context) ComponentHealth {
	health := ComponentHealth{
		LastCheck: time.Now(),
	}

	// Check connectivity
	err := hc.db.PingContext(ctx)
	if err != nil {
		health.Status = "unhealthy"
		health.Message = fmt.Sprintf("Database unreachable: %v", err)
		return health
	}

	// Check replication lag
	var lag sql.NullInt64
	err = hc.db.QueryRowContext(ctx,
		`SELECT EXTRACT(EPOCH FROM (NOW() - pg_last_xact_replay_timestamp()))::bigint`).Scan(&lag)

	if err != nil {
		// Might not have standby, that's OK for single instance
		health.Status = "healthy"
		health.Message = "Database healthy (single instance)"
		return health
	}

	if lag.Valid && lag.Int64 > 30 {
		health.Status = "degraded"
		health.Message = fmt.Sprintf("Replication lag: %d seconds", lag.Int64)
	} else {
		health.Status = "healthy"
		health.Message = "Database healthy"
	}

	return health
}

// checkCache checks Redis/cache connectivity
func (hc *HealthChecker) checkCache(ctx context.Context) ComponentHealth {
	health := ComponentHealth{
		LastCheck: time.Now(),
	}

	// TODO: Implement Redis health check when available
	// For now, assume healthy
	health.Status = "healthy"
	health.Message = "Cache configured"

	return health
}

// checkReplication checks storage replication health
func (hc *HealthChecker) checkReplication(ctx context.Context) (ComponentHealth, map[string]NodeHealthInfo) {
	health := ComponentHealth{
		LastCheck: time.Now(),
	}

	nodeHealth := make(map[string]NodeHealthInfo)

	rs := replicate.GetInstance()
	if rs == nil {
		health.Status = "unhealthy"
		health.Message = "Replication service not initialized"
		return health, nodeHealth
	}

	// Get node health status
	nodeStatus := rs.GetHealthStatus()

	healthyCount := 0
	for nodeID, status := range nodeStatus {
		info := NodeHealthInfo{
			FailureCount: status.FailureCount,
			LastCheck:    status.LastCheck,
		}

		if status.IsHealthy {
			info.Status = "healthy"
			healthyCount++
		} else {
			info.Status = "unhealthy"
		}

		nodeHealth[nodeID] = info
	}

	// Determine replication health
	totalNodes := len(nodeStatus)
	requiredHealthyNodes := 2

	if healthyCount < requiredHealthyNodes {
		health.Status = "unhealthy"
		health.Message = fmt.Sprintf("Insufficient replicas: %d/%d healthy", healthyCount, totalNodes)
	} else if healthyCount < totalNodes {
		health.Status = "degraded"
		health.Message = fmt.Sprintf("Replication degraded: %d/%d nodes healthy", healthyCount, totalNodes)
	} else {
		health.Status = "healthy"
		health.Message = fmt.Sprintf("All %d nodes healthy", totalNodes)
	}

	return health, nodeHealth
}

// determineOverallStatus determines overall system health
func (hc *HealthChecker) determineOverallStatus(status *HealthStatus) string {
	// If any critical component is unhealthy, system is unhealthy
	if status.Database.Status == "unhealthy" ||
		status.Replication.Status == "unhealthy" {
		return "unhealthy"
	}

	// If any component is degraded, system is degraded
	if status.Database.Status == "degraded" ||
		status.Cache.Status == "degraded" ||
		status.Replication.Status == "degraded" {
		return "degraded"
	}

	return "healthy"
}

// startPeriodicCheck runs health checks periodically
func (hc *HealthChecker) startPeriodicCheck() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = hc.checkHealth(ctx)
		cancel()
	}
}

// GetCachedStatus returns the last cached health status
func (hc *HealthChecker) GetCachedStatus() *HealthStatus {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	if hc.cache == nil {
		return &HealthStatus{
			Overall:   "unknown",
			Timestamp: time.Now(),
		}
	}

	return hc.cache
}

// VerifyReadiness performs readiness checks
func (hc *HealthChecker) VerifyReadiness() map[string]bool {
	checks := make(map[string]bool)

	// Check database
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err := hc.db.PingContext(ctx)
	cancel()
	checks["database_ready"] = err == nil

	// Check replication
	rs := replicate.GetInstance()
	nodeStatus := rs.GetHealthStatus()
	healthyNodes := 0
	for _, status := range nodeStatus {
		if status.IsHealthy {
			healthyNodes++
		}
	}
	checks["replication_ready"] = healthyNodes >= 2

	// Check config
	config.LoadEnvOnce()
	checks["config_ready"] = true

	return checks
}
