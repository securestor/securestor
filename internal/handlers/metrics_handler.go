package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/securestor/securestor/internal/health"
)

// MetricsHandler handles metrics and health endpoints using Gin framework
type MetricsHandler struct {
	db            *sql.DB
	healthChecker *health.HealthChecker
}

// NewMetricsHandler creates a new Gin-based metrics handler
func NewMetricsHandler(db *sql.DB, healthChecker *health.HealthChecker) *MetricsHandler {
	return &MetricsHandler{
		db:            db,
		healthChecker: healthChecker,
	}
}

// GetCacheMetrics returns cache performance metrics
// GET /api/v1/gin/metrics/cache
func (h *MetricsHandler) GetCacheMetrics(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "1h")

	// Parse time range
	duration := parseMetricsDuration(timeRange)
	startTime := time.Now().Add(-duration)

	// Query cache statistics
	query := `
		SELECT 
			SUM(l1_hits) as l1_hits,
			SUM(l1_misses) as l1_misses,
			SUM(l2_hits) as l2_hits,
			SUM(l2_misses) as l2_misses,
			SUM(l3_hits) as l3_hits,
			SUM(l3_misses) as l3_misses,
			AVG(l1_size_mb) as l1_avg_size,
			AVG(l2_size_gb) as l2_avg_size,
			AVG(l3_size_gb) as l3_avg_size,
			AVG(l1_latency_ms) as l1_avg_latency,
			AVG(l2_latency_ms) as l2_avg_latency,
			AVG(l3_latency_ms) as l3_avg_latency
		FROM cache_statistics
		WHERE timestamp >= $1
	`

	var l1Hits, l1Misses, l2Hits, l2Misses, l3Hits, l3Misses sql.NullInt64
	var l1Size, l2Size, l3Size, l1Latency, l2Latency, l3Latency sql.NullFloat64

	err := h.db.QueryRow(query, startTime).Scan(
		&l1Hits, &l1Misses, &l2Hits, &l2Misses, &l3Hits, &l3Misses,
		&l1Size, &l2Size, &l3Size, &l1Latency, &l2Latency, &l3Latency,
	)

	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get values with proper null handling
	l1H := int64(0)
	l1M := int64(0)
	l2H := int64(0)
	l2M := int64(0)
	l3H := int64(0)
	l3M := int64(0)

	if l1Hits.Valid {
		l1H = l1Hits.Int64
	}
	if l1Misses.Valid {
		l1M = l1Misses.Int64
	}
	if l2Hits.Valid {
		l2H = l2Hits.Int64
	}
	if l2Misses.Valid {
		l2M = l2Misses.Int64
	}
	if l3Hits.Valid {
		l3H = l3Hits.Int64
	}
	if l3Misses.Valid {
		l3M = l3Misses.Int64
	}

	// Calculate hit rates
	l1Total := l1H + l1M
	l2Total := l2H + l2M
	l3Total := l3H + l3M

	var l1HitRate, l2HitRate, l3HitRate float64
	if l1Total > 0 {
		l1HitRate = float64(l1H) / float64(l1Total)
	}
	if l2Total > 0 {
		l2HitRate = float64(l2H) / float64(l2Total)
	}
	if l3Total > 0 {
		l3HitRate = float64(l3H) / float64(l3Total)
	}

	// Get latency values
	l1Lat := 0.0
	l2Lat := 0.0
	l3Lat := 0.0
	if l1Latency.Valid {
		l1Lat = l1Latency.Float64
	}
	if l2Latency.Valid {
		l2Lat = l2Latency.Float64
	}
	if l3Latency.Valid {
		l3Lat = l3Latency.Float64
	}

	c.JSON(http.StatusOK, gin.H{
		"time_range": timeRange,
		"cache_metrics": gin.H{
			"l1_hit_rate":       l1HitRate,
			"l2_hit_rate":       l2HitRate,
			"l3_hit_rate":       l3HitRate,
			"l1_hits":           l1H,
			"l1_misses":         l1M,
			"l2_hits":           l2H,
			"l2_misses":         l2M,
			"l3_hits":           l3H,
			"l3_misses":         l3M,
			"l1_avg_latency_ms": l1Lat,
			"l2_avg_latency_ms": l2Lat,
			"l3_avg_latency_ms": l3Lat,
			"total_requests":    l1Total + l2Total + l3Total,
		},
	})
}

// GetPerformanceMetrics returns system performance metrics
// GET /api/v1/gin/metrics/performance
func (h *MetricsHandler) GetPerformanceMetrics(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "1h")

	duration := parseMetricsDuration(timeRange)
	startTime := time.Now().Add(-duration)

	// Query performance metrics with proper percentile calculations
	query := `
		SELECT 
			AVG(request_count) as avg_requests,
			AVG(error_count) as avg_errors,
			PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY response_time_ms) as p50_response_time,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time_ms) as p95_response_time,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY response_time_ms) as p99_response_time,
			MAX(response_time_ms) as max_response_time,
			SUM(bandwidth_mb) as total_bandwidth
		FROM performance_metrics
		WHERE timestamp >= $1
	`

	var avgRequests, avgErrors, p50, p95, p99, maxResponseTime, totalBandwidth sql.NullFloat64

	err := h.db.QueryRow(query, startTime).Scan(
		&avgRequests, &avgErrors, &p50, &p95, &p99, &maxResponseTime, &totalBandwidth,
	)

	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get values with null handling
	avgReq := 0.0
	avgErr := 0.0
	p50Val := 0.0
	p95Val := 0.0
	p99Val := 0.0
	maxRT := 0.0
	totalBW := 0.0

	if avgRequests.Valid {
		avgReq = avgRequests.Float64
	}
	if avgErrors.Valid {
		avgErr = avgErrors.Float64
	}
	if p50.Valid {
		p50Val = p50.Float64
	}
	if p95.Valid {
		p95Val = p95.Float64
	}
	if p99.Valid {
		p99Val = p99.Float64
	}
	if maxResponseTime.Valid {
		maxRT = maxResponseTime.Float64
	}
	if totalBandwidth.Valid {
		totalBW = totalBandwidth.Float64
	}

	// Calculate error rate
	var errorRate float64
	if avgReq > 0 {
		errorRate = (avgErr / avgReq)
	}

	c.JSON(http.StatusOK, gin.H{
		"time_range": timeRange,
		"response_time_metrics": gin.H{
			"p50_ms": p50Val,
			"p95_ms": p95Val,
			"p99_ms": p99Val,
			"max_ms": maxRT,
		},
		"throughput": gin.H{
			"requests_per_sec":  avgReq / 3600, // Convert hourly to per second
			"megabytes_per_sec": totalBW / 3600,
		},
		"error_rate": errorRate,
	})
}

// GetRepositoryHealth returns health status of all repositories
// GET /api/v1/gin/health/repositories
func (h *MetricsHandler) GetRepositoryHealth(c *gin.Context) {
	query := `
		SELECT 
			r.repository_id,
			r.name,
			r.type,
			r.status as status,
			NOW() as last_check,
			0 as response_time,
			0 as error_count
		FROM repositories r
		ORDER BY r.name
		LIMIT 100
	`

	rows, err := h.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var repositories []gin.H
	for rows.Next() {
		var id, name, repoType, status string
		var lastCheck time.Time
		var responseTime, errorCount int

		err := rows.Scan(&id, &name, &repoType, &status, &lastCheck, &responseTime, &errorCount)
		if err != nil {
			continue
		}

		repositories = append(repositories, gin.H{
			"id":            id,
			"name":          name,
			"type":          repoType,
			"status":        status,
			"url":           "", // Will be populated if needed
			"response_time": responseTime,
			"last_check":    lastCheck,
			"error_count":   errorCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": repositories,
		"total":        len(repositories),
	})
}

// GetAlerts returns active system alerts
// GET /api/v1/gin/alerts
func (h *MetricsHandler) GetAlerts(c *gin.Context) {
	status := c.DefaultQuery("status", "active")

	query := `
		SELECT 
			id,
			alert_type,
			severity,
			message,
			repository_id,
			created_at,
			resolved_at,
			metadata
		FROM system_alerts
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := h.db.Query(query, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var alerts []gin.H
	for rows.Next() {
		var id, alertType, severity, message string
		var repositoryID sql.NullString
		var createdAt time.Time
		var resolvedAt sql.NullTime
		var metadata sql.NullString

		err := rows.Scan(&id, &alertType, &severity, &message, &repositoryID, &createdAt, &resolvedAt, &metadata)
		if err != nil {
			continue
		}

		// Default values
		metricValue := "N/A"
		threshold := "N/A"

		// Parse metadata JSON if present
		if metadata.Valid && metadata.String != "" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(metadata.String), &meta); err == nil {
				if val, ok := meta["metric_value"]; ok {
					metricValue = fmt.Sprintf("%.1f", val)
				}
				if thr, ok := meta["threshold"]; ok {
					threshold = fmt.Sprintf("%.1f", thr)
				}
				// Add unit if present
				if unit, ok := meta["unit"]; ok {
					unitStr := unit.(string)
					if metricValue != "N/A" {
						metricValue = fmt.Sprintf("%s %s", metricValue, unitStr)
					}
					if threshold != "N/A" {
						threshold = fmt.Sprintf("%s %s", threshold, unitStr)
					}
				}
			}
		}

		alert := gin.H{
			"id":           id,
			"rule_name":    alertType,
			"severity":     severity,
			"message":      message,
			"created_at":   createdAt,
			"metric_value": metricValue,
			"threshold":    threshold,
		}

		if repositoryID.Valid {
			alert["repository_id"] = repositoryID.String
		}
		if resolvedAt.Valid {
			alert["resolved_at"] = resolvedAt.Time
		}

		alerts = append(alerts, alert)
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": alerts,
		"total":  len(alerts),
		"status": status,
	})
}

// GetLiveness returns liveness status (always returns OK if service is running)
// GET /api/v1/gin/health/live
func (h *MetricsHandler) GetLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"alive": true,
	})
}

// GetReadiness returns readiness status checking all dependencies
// GET /api/v1/gin/health/ready
func (h *MetricsHandler) GetReadiness(c *gin.Context) {
	if h.healthChecker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not initialized",
		})
		return
	}

	checks := h.healthChecker.VerifyReadiness()

	// All checks must pass
	allReady := true
	for _, ready := range checks {
		if !ready {
			allReady = false
			break
		}
	}

	if allReady {
		c.JSON(http.StatusOK, checks)
	} else {
		c.JSON(http.StatusServiceUnavailable, checks)
	}
}

// GetHealth returns comprehensive health status (requires authentication)
// GET /api/v1/gin/health
func (h *MetricsHandler) GetHealth(c *gin.Context) {
	if h.healthChecker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Health checker not initialized",
		})
		return
	}

	// Perform comprehensive health check
	status := h.healthChecker.GetHealthStatus()

	// Determine HTTP status code based on health
	httpStatus := http.StatusOK
	if status == nil {
		httpStatus = http.StatusServiceUnavailable
		c.JSON(httpStatus, gin.H{
			"status": "unhealthy",
		})
		return
	}

	c.JSON(httpStatus, status)
}

// parseMetricsDuration parses a time range string (e.g., "1h", "24h", "7d") into a duration
func parseMetricsDuration(timeRange string) time.Duration {
	// Default to 1 hour if parsing fails
	duration, err := time.ParseDuration(timeRange)
	if err != nil {
		// Try parsing as days (e.g., "7d")
		if len(timeRange) > 1 && timeRange[len(timeRange)-1] == 'd' {
			days := timeRange[:len(timeRange)-1]
			if d, err := time.ParseDuration(days + "h"); err == nil {
				return d * 24
			}
		}
		// Default to 1 hour
		return time.Hour
	}
	return duration
}
