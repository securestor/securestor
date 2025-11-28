package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// respondWithJSON is a helper to send JSON responses
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// MetricsAPIHandler handles metrics API requests for the monitoring dashboard
type MetricsAPIHandler struct {
	db *sql.DB
}

func NewMetricsAPIHandler(db *sql.DB) *MetricsAPIHandler {
	return &MetricsAPIHandler{db: db}
}

// GetCacheMetrics returns cache performance metrics
func (h *MetricsAPIHandler) GetCacheMetrics(w http.ResponseWriter, r *http.Request) {
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}

	// Parse time range
	duration := parseDuration(timeRange)
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

	var l1Hits, l1Misses, l2Hits, l2Misses, l3Hits, l3Misses int64
	var l1Size, l2Size, l3Size, l1Latency, l2Latency, l3Latency float64

	err := h.db.QueryRow(query, startTime).Scan(
		&l1Hits, &l1Misses, &l2Hits, &l2Misses, &l3Hits, &l3Misses,
		&l1Size, &l2Size, &l3Size, &l1Latency, &l2Latency, &l3Latency,
	)

	if err != nil && err != sql.ErrNoRows {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Calculate hit rates
	l1Total := l1Hits + l1Misses
	l2Total := l2Hits + l2Misses
	l3Total := l3Hits + l3Misses

	var l1HitRate, l2HitRate, l3HitRate float64
	if l1Total > 0 {
		l1HitRate = float64(l1Hits) / float64(l1Total)
	}
	if l2Total > 0 {
		l2HitRate = float64(l2Hits) / float64(l2Total)
	}
	if l3Total > 0 {
		l3HitRate = float64(l3Hits) / float64(l3Total)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"time_range": timeRange,
		"cache_metrics": map[string]interface{}{
			"l1_hit_rate":       l1HitRate,
			"l2_hit_rate":       l2HitRate,
			"l3_hit_rate":       l3HitRate,
			"l1_hits":           l1Hits,
			"l1_misses":         l1Misses,
			"l2_hits":           l2Hits,
			"l2_misses":         l2Misses,
			"l3_hits":           l3Hits,
			"l3_misses":         l3Misses,
			"l1_avg_latency_ms": l1Latency,
			"l2_avg_latency_ms": l2Latency,
			"l3_avg_latency_ms": l3Latency,
			"total_requests":    l1Total + l2Total + l3Total,
		},
	})
}

// GetPerformanceMetrics returns system performance metrics
func (h *MetricsAPIHandler) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1h"
	}

	duration := parseDuration(timeRange)
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

	var avgRequests, avgErrors, p50, p95, p99, maxResponseTime, totalBandwidth float64

	err := h.db.QueryRow(query, startTime).Scan(
		&avgRequests, &avgErrors, &p50, &p95, &p99, &maxResponseTime, &totalBandwidth,
	)

	if err != nil && err != sql.ErrNoRows {
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Calculate error rate
	var errorRate float64
	if avgRequests > 0 {
		errorRate = (avgErrors / avgRequests)
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"time_range": timeRange,
		"response_time_metrics": map[string]interface{}{
			"p50_ms": p50,
			"p95_ms": p95,
			"p99_ms": p99,
			"max_ms": maxResponseTime,
		},
		"throughput": map[string]interface{}{
			"requests_per_sec":  avgRequests / 3600, // Convert hourly to per second
			"megabytes_per_sec": totalBandwidth / 3600,
		},
		"error_rate": errorRate,
	})
}

// GetRepositoryHealth returns health status of all repositories
func (h *MetricsAPIHandler) GetRepositoryHealth(w http.ResponseWriter, r *http.Request) {
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
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var repositories []map[string]interface{}
	for rows.Next() {
		var id, name, repoType, status string
		var lastCheck time.Time
		var responseTime, errorCount int

		err := rows.Scan(&id, &name, &repoType, &status, &lastCheck, &responseTime, &errorCount)
		if err != nil {
			continue
		}

		repositories = append(repositories, map[string]interface{}{
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

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"repositories": repositories,
		"total":        len(repositories),
	})
}

// GetAlerts returns active system alerts
func (h *MetricsAPIHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}

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
		respondWithJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var alerts []map[string]interface{}
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

		alert := map[string]interface{}{
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

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"total":  len(alerts),
		"status": status,
	})
}

// Helper function to parse duration strings
func parseDuration(timeRange string) time.Duration {
	switch timeRange {
	case "1h":
		return time.Hour
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}
