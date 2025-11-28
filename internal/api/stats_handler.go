package api

import (
	"encoding/json"
	"net/http"

	"github.com/securestor/securestor/internal/health"
	"github.com/securestor/securestor/internal/service"
)

func (s *Server) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Origin, X-Requested-With")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Try to use audit log service for more accurate stats if available
	if s.auditLogService != nil {
		// Get today's data from audit service
		ctx := r.Context()
		activeUsers, err := s.auditLogService.GetActiveUsersToday(ctx)
		if err != nil {
			s.logger.Printf("Error getting active users from audit service: %v", err)
		}

		downloadsToday, err := s.auditLogService.GetDownloadsToday(ctx)
		if err != nil {
			s.logger.Printf("Error getting downloads from audit service: %v", err)
		}

		// If we got data from audit service, use enhanced stats
		if activeUsers >= 0 || downloadsToday >= 0 {
			// Get other stats from dashboard service
			dashboardStatsService := service.NewDashboardStatsService(s.db)
			stats, err := dashboardStatsService.GetBasicStats()
			if err != nil {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}

			// Override with audit service data
			if activeUsers >= 0 {
				stats.ActiveUsers = activeUsers
			}
			if downloadsToday >= 0 {
				stats.DownloadsToday = downloadsToday
			}

			respondWithJSON(w, http.StatusOK, stats)
			return
		}
	}

	// Fallback to standard dashboard stats service
	dashboardStatsService := service.NewDashboardStatsService(s.db)

	// Get basic stats for dashboard
	stats, err := dashboardStatsService.GetBasicStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

func (s *Server) handleDetailedStats(w http.ResponseWriter, r *http.Request) {
	// Create dashboard stats service
	dashboardStatsService := service.NewDashboardStatsService(s.db)

	// Get detailed stats
	stats, err := dashboardStatsService.GetDetailedStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}

func (s *Server) handleRealtimeStats(w http.ResponseWriter, r *http.Request) {
	// Set headers for Server-Sent Events
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	dashboardStatsService := service.NewDashboardStatsService(s.db)

	// Send initial stats
	stats, err := dashboardStatsService.GetBasicStats()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Send the data as Server-Sent Event
	_, err = w.Write([]byte("data: "))
	if err != nil {
		return
	}

	respondWithJSON(w, http.StatusOK, stats)

	// Note: In a production system, you'd implement a proper SSE loop here
	// For now, we'll just send one response
}

func (s *Server) handleStatsMetrics(w http.ResponseWriter, r *http.Request) {
	dashboardStatsService := service.NewDashboardStatsService(s.db)

	// Get the metric type from query params
	metricType := r.URL.Query().Get("type")

	switch metricType {
	case "storage":
		stats, err := dashboardStatsService.GetDetailedStats()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, stats.Storage)

	case "artifacts":
		stats, err := dashboardStatsService.GetDetailedStats()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, stats.Artifacts)

	case "downloads":
		stats, err := dashboardStatsService.GetDetailedStats()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, stats.Downloads)

	case "users":
		stats, err := dashboardStatsService.GetDetailedStats()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, stats.Users)

	case "compliance":
		stats, err := dashboardStatsService.GetDetailedStats()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, stats.Compliance)

	case "security":
		stats, err := dashboardStatsService.GetDetailedStats()
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondWithJSON(w, http.StatusOK, stats.Security)

	default:
		respondWithError(w, http.StatusBadRequest, "Invalid metric type. Use: storage, artifacts, downloads, users, compliance, security")
	}
}

func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	hc := health.GetInstance()
	if hc == nil {
		respondWithError(w, http.StatusInternalServerError, "Health checker not initialized")
		return
	}

	status := hc.GetHealthStatus()
	w.Header().Set("Content-Type", "application/json")

	// Set appropriate HTTP status based on overall health
	statusCode := http.StatusOK
	if status.Overall == "degraded" {
		statusCode = http.StatusOK // Still OK, but degraded
	} else if status.Overall == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(status)
}

// handleReadinessCheck returns readiness status for Kubernetes probes
func (s *Server) handleReadinessCheck(w http.ResponseWriter, r *http.Request) {
	hc := health.GetInstance()
	if hc == nil {
		respondWithError(w, http.StatusServiceUnavailable, "Health checker not initialized")
		return
	}

	checks := hc.VerifyReadiness()

	// All checks must pass
	allReady := true
	for _, ready := range checks {
		if !ready {
			allReady = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if allReady {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(checks)
}

// handleLivenessCheck returns liveness status (always returns OK if service is running)
func (s *Server) handleLivenessCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alive": true,
	})
}
