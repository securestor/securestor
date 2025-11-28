package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// MetricsCollector collects and stores proxy metrics for monitoring
type MetricsCollector struct {
	db *sql.DB
}

// HealthCheckResult represents health check status
type HealthCheckResult struct {
	RemoteRepositoryID  string    `json:"remote_repository_id"`
	Status              string    `json:"status"` // healthy, degraded, unhealthy
	LastCheckAt         time.Time `json:"last_check_at"`
	ResponseTimeMs      int       `json:"response_time_ms"`
	ErrorCount          int       `json:"error_count"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	SuccessRate         float64   `json:"success_rate"` // 0.0 - 100.0
	LastErrorMessage    string    `json:"last_error_message"`
}

// CacheMetrics represents cache performance metrics
type CacheMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	TotalRequests     int64     `json:"total_requests"`
	CacheHits         int64     `json:"cache_hits"`
	CacheMisses       int64     `json:"cache_misses"`
	HitRate           float64   `json:"hit_rate"` // 0.0 - 100.0
	L1HitRate         float64   `json:"l1_hit_rate"`
	L2HitRate         float64   `json:"l2_hit_rate"`
	L3HitRate         float64   `json:"l3_hit_rate"`
	AvgResponseTimeMs float64   `json:"avg_response_time_ms"`
	P99ResponseTimeMs float64   `json:"p99_response_time_ms"`
	P95ResponseTimeMs float64   `json:"p95_response_time_ms"`
	EvictionsCount    int64     `json:"evictions_count"`
	BytesCached       int64     `json:"bytes_cached"`
}

// ProxyMetric represents a single proxy metric record
type ProxyMetric struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	RemoteRepoID  string    `json:"remote_repository_id"`
	ArtifactID    string    `json:"artifact_id"`
	RequestTime   time.Time `json:"request_time"`
	CacheTierHit  string    `json:"cache_tier_hit"` // L1, L2, L3, MISS
	CacheHit      bool      `json:"cache_hit"`
	TotalTimeMs   int       `json:"total_time_ms"`
	NetworkTimeMs int       `json:"network_time_ms"`
	CacheLookupMs int       `json:"cache_lookup_time_ms"`
	RequestMethod string    `json:"request_method"`
	ResponseCode  int       `json:"response_code"`
	ResponseSizeB int64     `json:"response_size_bytes"`
	RetryCount    int       `json:"retry_count"`
	FallbackUsed  bool      `json:"fallback_used"`
}

// AlertRule defines alerting conditions
type AlertRule struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Metric      string    `json:"metric"`   // e.g., "cache_hit_rate", "response_time_p99"
	Operator    string    `json:"operator"` // <, >, <=, >=, ==
	Threshold   float64   `json:"threshold"`
	DurationSec int       `json:"duration_seconds"` // Alert if condition persists for this duration
	Enabled     bool      `json:"enabled"`
	NotifyEmail string    `json:"notify_email"`
	Severity    string    `json:"severity"` // INFO, WARNING, CRITICAL
	CreatedAt   time.Time `json:"created_at"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(db *sql.DB) *MetricsCollector {
	return &MetricsCollector{db: db}
}

// RecordProxyMetric records a proxy request metric
func (mc *MetricsCollector) RecordProxyMetric(ctx context.Context, metric *ProxyMetric) error {
	query := `
		INSERT INTO proxy_metrics (
			tenant_id, remote_repository_id, artifact_id, request_timestamp,
			cache_tier_hit, cache_hit, total_time_ms, network_time_ms,
			cache_lookup_time_ms, request_method, response_code, response_size_bytes,
			retry_count, fallback_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	err := mc.db.QueryRowContext(ctx, query,
		metric.TenantID,
		metric.RemoteRepoID,
		metric.ArtifactID,
		metric.RequestTime,
		metric.CacheTierHit,
		metric.CacheHit,
		metric.TotalTimeMs,
		metric.NetworkTimeMs,
		metric.CacheLookupMs,
		metric.RequestMethod,
		metric.ResponseCode,
		metric.ResponseSizeB,
		metric.RetryCount,
		metric.FallbackUsed,
	).Scan(&metric.ID)

	if err != nil {
		return fmt.Errorf("failed to record proxy metric: %w", err)
	}

	return nil
}

// GetCacheMetrics calculates cache performance metrics for the last duration
func (mc *MetricsCollector) GetCacheMetrics(ctx context.Context, tenantID string, duration time.Duration) (*CacheMetrics, error) {
	metrics := &CacheMetrics{
		Timestamp: time.Now(),
	}

	query := `
		SELECT 
			COUNT(*) as total_requests,
			SUM(CASE WHEN cache_hit = true THEN 1 ELSE 0 END) as cache_hits,
			SUM(CASE WHEN cache_hit = false THEN 1 ELSE 0 END) as cache_misses,
			AVG(total_time_ms) as avg_response_time_ms,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY total_time_ms) as p99_response_time_ms,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY total_time_ms) as p95_response_time_ms,
			SUM(response_size_bytes) as bytes_cached
		FROM proxy_metrics
		WHERE tenant_id = $1 AND request_timestamp > NOW() - INTERVAL '1' MINUTE * $2
	`

	var totalRequests, cacheHits, cacheMisses int64
	var avgResponseTime, p99ResponseTime, p95ResponseTime float64
	var bytesCached int64

	err := mc.db.QueryRowContext(ctx, query, tenantID, int(duration.Minutes())).Scan(
		&totalRequests,
		&cacheHits,
		&cacheMisses,
		&avgResponseTime,
		&p99ResponseTime,
		&p95ResponseTime,
		&bytesCached,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get cache metrics: %w", err)
	}

	metrics.TotalRequests = totalRequests
	metrics.CacheHits = cacheHits
	metrics.CacheMisses = cacheMisses
	metrics.BytesCached = bytesCached
	metrics.AvgResponseTimeMs = avgResponseTime
	metrics.P99ResponseTimeMs = p99ResponseTime
	metrics.P95ResponseTimeMs = p95ResponseTime

	if totalRequests > 0 {
		metrics.HitRate = (float64(cacheHits) / float64(totalRequests)) * 100.0
	}

	return metrics, nil
}

// RecordHealthCheck records health check result for a remote repository
func (mc *MetricsCollector) RecordHealthCheck(ctx context.Context, result *HealthCheckResult) error {
	query := `
		INSERT INTO remote_repository_health (
			remote_repository_id, status, last_check_at, response_time_ms,
			error_count, consecutive_failures, success_rate, last_error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (remote_repository_id) DO UPDATE SET
			status = $2,
			last_check_at = $3,
			response_time_ms = $4,
			error_count = $5,
			consecutive_failures = $6,
			success_rate = $7,
			last_error_message = $8,
			updated_at = NOW()
	`

	_, err := mc.db.ExecContext(ctx, query,
		result.RemoteRepositoryID,
		result.Status,
		result.LastCheckAt,
		result.ResponseTimeMs,
		result.ErrorCount,
		result.ConsecutiveFailures,
		result.SuccessRate,
		result.LastErrorMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to record health check: %w", err)
	}

	log.Printf("✅ Health check recorded: %s - %s", result.RemoteRepositoryID, result.Status)
	return nil
}

// GetHealthStatus gets current health status of all remote repositories
func (mc *MetricsCollector) GetHealthStatus(ctx context.Context, tenantID string) (map[string]*HealthCheckResult, error) {
	query := `
		SELECT 
			remote_repository_id, status, last_check_at, response_time_ms,
			error_count, consecutive_failures, success_rate, last_error_message
		FROM remote_repository_health
		WHERE remote_repository_id IN (
			SELECT id FROM remote_repositories WHERE tenant_id = $1
		)
		ORDER BY last_check_at DESC
	`

	rows, err := mc.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get health status: %w", err)
	}
	defer rows.Close()

	results := make(map[string]*HealthCheckResult)
	for rows.Next() {
		result := &HealthCheckResult{}
		err := rows.Scan(
			&result.RemoteRepositoryID,
			&result.Status,
			&result.LastCheckAt,
			&result.ResponseTimeMs,
			&result.ErrorCount,
			&result.ConsecutiveFailures,
			&result.SuccessRate,
			&result.LastErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health status: %w", err)
		}
		results[result.RemoteRepositoryID] = result
	}

	return results, rows.Err()
}

// CreateAlertRule creates a new alerting rule
func (mc *MetricsCollector) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	query := `
		INSERT INTO proxy_alert_rules (
			tenant_id, name, description, metric, operator, threshold,
			duration_seconds, enabled, notify_email, severity, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id
	`

	err := mc.db.QueryRowContext(ctx, query,
		rule.TenantID,
		rule.Name,
		rule.Description,
		rule.Metric,
		rule.Operator,
		rule.Threshold,
		rule.DurationSec,
		rule.Enabled,
		rule.NotifyEmail,
		rule.Severity,
	).Scan(&rule.ID)

	if err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}

	log.Printf("✅ Alert rule created: %s", rule.Name)
	return nil
}

// EvaluateAlertRules evaluates all active alert rules
func (mc *MetricsCollector) EvaluateAlertRules(ctx context.Context, tenantID string) ([]string, error) {
	var alerts []string

	query := `
		SELECT id, name, metric, operator, threshold, duration_seconds
		FROM proxy_alert_rules
		WHERE tenant_id = $1 AND enabled = true
	`

	rows, err := mc.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate alert rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ruleID, name, metric, operator string
		var threshold float64
		var duration int

		if err := rows.Scan(&ruleID, &name, &metric, &operator, &threshold, &duration); err != nil {
			return nil, fmt.Errorf("failed to scan alert rule: %w", err)
		}

		// Evaluate based on metric type
		alert, err := mc.evaluateMetric(ctx, tenantID, metric, operator, threshold, duration)
		if err != nil {
			log.Printf("⚠️  Error evaluating alert rule %s: %v", ruleID, err)
			continue
		}

		if alert != "" {
			alerts = append(alerts, fmt.Sprintf("[%s] %s - %s", name, metric, alert))
		}
	}

	return alerts, rows.Err()
}

// evaluateMetric evaluates a specific metric against a threshold
func (mc *MetricsCollector) evaluateMetric(ctx context.Context, tenantID, metric, operator string, threshold float64, duration int) (string, error) {
	switch metric {
	case "cache_hit_rate":
		metrics, err := mc.GetCacheMetrics(ctx, tenantID, time.Duration(duration)*time.Second)
		if err != nil {
			return "", err
		}
		if shouldAlert(metrics.HitRate, operator, threshold) {
			return fmt.Sprintf("Hit rate %.2f%% violates threshold %.2f%%", metrics.HitRate, threshold), nil
		}

	case "response_time_p99":
		metrics, err := mc.GetCacheMetrics(ctx, tenantID, time.Duration(duration)*time.Second)
		if err != nil {
			return "", err
		}
		if shouldAlert(metrics.P99ResponseTimeMs, operator, threshold) {
			return fmt.Sprintf("P99 response time %.0fms violates threshold %.0fms", metrics.P99ResponseTimeMs, threshold), nil
		}

	case "error_rate":
		// Calculate error rate from proxy_metrics
		query := `
			SELECT 
				(SUM(CASE WHEN response_code >= 400 THEN 1 ELSE 0 END)::float / COUNT(*) * 100)
			FROM proxy_metrics
			WHERE tenant_id = $1 AND request_timestamp > NOW() - INTERVAL '1' SECOND * $2
		`
		var errorRate float64
		err := mc.db.QueryRowContext(ctx, query, tenantID, duration).Scan(&errorRate)
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
		if shouldAlert(errorRate, operator, threshold) {
			return fmt.Sprintf("Error rate %.2f%% violates threshold %.2f%%", errorRate, threshold), nil
		}
	}

	return "", nil
}

// shouldAlert determines if threshold is breached based on operator
func shouldAlert(actual float64, operator string, threshold float64) bool {
	switch operator {
	case "<":
		return actual < threshold
	case ">":
		return actual > threshold
	case "<=":
		return actual <= threshold
	case ">=":
		return actual >= threshold
	case "==":
		return actual == threshold
	default:
		return false
	}
}

// GetMetricsTimeSeries retrieves metrics for time-series graphing
func (mc *MetricsCollector) GetMetricsTimeSeries(ctx context.Context, tenantID string, metricType string, duration time.Duration) ([]map[string]interface{}, error) {
	var query string

	switch metricType {
	case "cache_hits":
		query = `
			SELECT 
				DATE_TRUNC('minute', request_timestamp) as timestamp,
				SUM(CASE WHEN cache_hit = true THEN 1 ELSE 0 END) as value
			FROM proxy_metrics
			WHERE tenant_id = $1 AND request_timestamp > NOW() - $2::INTERVAL
			GROUP BY DATE_TRUNC('minute', request_timestamp)
			ORDER BY timestamp ASC
		`

	case "response_time":
		query = `
			SELECT 
				DATE_TRUNC('minute', request_timestamp) as timestamp,
				AVG(total_time_ms) as value
			FROM proxy_metrics
			WHERE tenant_id = $1 AND request_timestamp > NOW() - $2::INTERVAL
			GROUP BY DATE_TRUNC('minute', request_timestamp)
			ORDER BY timestamp ASC
		`

	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricType)
	}

	rows, err := mc.db.QueryContext(ctx, query, tenantID, fmt.Sprintf("%d minutes", int(duration.Minutes())))
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timestamp time.Time
		var value float64

		if err := rows.Scan(&timestamp, &value); err != nil {
			return nil, fmt.Errorf("failed to scan time series data: %w", err)
		}

		results = append(results, map[string]interface{}{
			"timestamp": timestamp,
			"value":     value,
		})
	}

	return results, rows.Err()
}
