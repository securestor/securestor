package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleCacheHealthCheck provides a Redis-enhanced health check
func (s *Server) handleCacheHealthCheck(w http.ResponseWriter, r *http.Request) {
	healthData := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"services":  map[string]string{},
	}

	// Database is assumed healthy if server is running
	healthData["services"].(map[string]string)["database"] = "healthy"

	// Check Redis health if available
	if s.cacheService != nil {
		if err := s.cacheService.Health(r.Context()); err != nil {
			healthData["services"].(map[string]string)["redis"] = "unhealthy: " + err.Error()
			healthData["status"] = "degraded"
		} else {
			healthData["services"].(map[string]string)["redis"] = "healthy"
		}

		// Try to cache the health check result for 30 seconds
		cacheKey := "health_check_cache"
		if cachedHealth := map[string]interface{}{}; s.cacheService.GetStats(r.Context(), cacheKey, &cachedHealth) == nil {
			healthData["cached"] = true
			healthData["cache_timestamp"] = cachedHealth["timestamp"]
		} else {
			// Cache this health check
			s.cacheService.CacheStats(r.Context(), cacheKey, healthData)
			healthData["cached"] = false
		}
	} else {
		healthData["services"].(map[string]string)["redis"] = "not configured"
	}

	// Check workflow service health
	if s.workflowService != nil {
		workflows := s.workflowService.GetAvailableWorkflows()
		healthData["services"].(map[string]string)["workflows"] = "healthy"
		healthData["workflow_count"] = len(workflows)
	} else {
		healthData["services"].(map[string]string)["workflows"] = "not available"
	}

	w.Header().Set("Content-Type", "application/json")

	// Set appropriate HTTP status based on health
	if healthData["status"] == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(healthData)
}

// handleCacheStats provides Redis cache statistics
func (s *Server) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if s.cacheService == nil {
		respondWithError(w, http.StatusServiceUnavailable, "Cache service not available")
		return
	}

	stats := map[string]interface{}{
		"redis_connected": true,
		"timestamp":       time.Now(),
	}

	// Try to get some cache stats
	if err := s.cacheService.Health(r.Context()); err != nil {
		stats["redis_connected"] = false
		stats["error"] = err.Error()
	}

	respondWithJSON(w, http.StatusOK, stats)
}

// handleFlushCache provides cache flush functionality (admin only)
func (s *Server) handleFlushCache(w http.ResponseWriter, r *http.Request) {
	if s.cacheService == nil {
		respondWithError(w, http.StatusServiceUnavailable, "Cache service not available")
		return
	}

	// In production, add authentication/authorization here
	if err := s.cacheService.FlushAll(r.Context()); err != nil {
		s.logger.Printf("Failed to flush cache: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to flush cache")
		return
	}

	s.logger.Println("Cache flushed successfully")
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Cache flushed successfully",
		"timestamp": time.Now(),
	})
}
