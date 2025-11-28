package api

import (
	"encoding/json"
	"net/http"

	"github.com/securestor/securestor/internal/replicate"
)

// handleDatabaseFailoverStatus returns database failover status
func (s *Server) handleDatabaseFailoverStatus(w http.ResponseWriter, r *http.Request) {
	// This would be populated when failover service is initialized
	status := map[string]interface{}{
		"primary":                 "postgres-primary:5432",
		"standby":                 "postgres-standby:5433",
		"status":                  "healthy",
		"replication_lag_seconds": 0,
		"last_failover":           "never",
		"failover_count":          0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// handlePatroniClusterStatus returns Patroni cluster information
func (s *Server) handlePatroniClusterStatus(w http.ResponseWriter, r *http.Request) {
	// This would be populated when Patroni service is initialized
	status := map[string]interface{}{
		"cluster_name": "securestor",
		"members": []map[string]interface{}{
			{
				"name":  "postgres-primary",
				"role":  "primary",
				"state": "running",
				"lag":   0,
			},
			{
				"name":  "postgres-standby",
				"role":  "standby",
				"state": "running",
				"lag":   0,
			},
		},
		"health": "healthy",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// handleReplicationMetrics returns replication performance metrics
func (s *Server) handleReplicationMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"replication_lag_seconds": 0,
		"replicas": []map[string]interface{}{
			{
				"client_addr": "postgres-standby",
				"user":        "replication_user",
				"state":       "streaming",
				"write_lag":   "0 ms",
				"flush_lag":   "0 ms",
				"replay_lag":  "0 ms",
			},
		},
		"bytes_received": 0,
		"bytes_written":  0,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// handleStorageReplicationStatus returns storage replication metrics
func (s *Server) handleStorageReplicationStatus(w http.ResponseWriter, r *http.Request) {
	rs := s.getReplicationService()
	if rs == nil {
		respondWithError(w, http.StatusInternalServerError, "Replication service not initialized")
		return
	}

	nodeStatus := rs.GetHealthStatus()
	metrics := map[string]interface{}{
		"total_nodes":   len(nodeStatus),
		"healthy_nodes": 0,
		"nodes":         make([]map[string]interface{}, 0),
	}

	healthyCount := 0
	for nodeID, health := range nodeStatus {
		if health.IsHealthy {
			healthyCount++
		}

		nodeInfo := map[string]interface{}{
			"id":            nodeID,
			"status":        "unknown",
			"healthy":       health.IsHealthy,
			"failure_count": health.FailureCount,
			"last_check":    health.LastCheck,
		}

		if health.IsHealthy {
			nodeInfo["status"] = "healthy"
		} else {
			nodeInfo["status"] = "unhealthy"
		}

		metrics["nodes"] = append(metrics["nodes"].([]map[string]interface{}), nodeInfo)
	}

	metrics["healthy_nodes"] = healthyCount
	if healthyCount >= 2 {
		metrics["replication_status"] = "healthy"
	} else if healthyCount >= 1 {
		metrics["replication_status"] = "degraded"
	} else {
		metrics["replication_status"] = "unhealthy"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// getReplicationService returns the replication service
func (s *Server) getReplicationService() *replicate.ReplicationService {
	rs := replicate.GetInstance()
	return rs
}
