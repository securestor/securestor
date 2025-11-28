package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/tenant"
)

func (s *Server) handleGetCompliance(c *gin.Context) {
	// Extract tenant context
	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found. Ensure tenant middleware is applied."})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Verify artifact belongs to tenant
	_, err = s.artifactService.GetByIDAndTenant(id, tenantID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: artifact not found or belongs to different tenant"})
		return
	}

	compliance, err := s.complianceService.GetByArtifactID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Enhance response with scan status if available
	response := map[string]interface{}{
		"compliance": compliance,
		"scan_info":  nil,
	}

	// Add current scan status from the new scan system
	if s.scanService != nil {
		if latestScan, err := s.scanService.GetLatestScanResult(id); err == nil && latestScan != nil {
			response["scan_info"] = map[string]interface{}{
				"last_scan_id":     latestScan.ID,
				"last_scan_status": latestScan.Status,
				"last_scan_date":   latestScan.StartedAt,
				"overall_score":    nil,
				"risk_level":       nil,
			}

			if latestScan.Results != nil {
				response["scan_info"].(map[string]interface{})["overall_score"] = latestScan.Results.OverallScore
				response["scan_info"].(map[string]interface{})["risk_level"] = latestScan.Results.RiskLevel
			}
		}

		// Check if there's an active scan
		if activeScan, err := s.scanService.GetActiveScan(id); err == nil && activeScan != nil {
			response["active_scan"] = map[string]interface{}{
				"scan_id":    activeScan.ID,
				"status":     activeScan.Status,
				"started_at": activeScan.StartedAt,
				"scan_type":  activeScan.ScanType,
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleCreateCompliance(c *gin.Context) {
	// Extract tenant context
	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found. Ensure tenant middleware is applied."})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Verify artifact belongs to tenant
	artifact, err := s.artifactService.GetByIDAndTenant(id, tenantID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: artifact not found or belongs to different tenant"})
		return
	}

	var request struct {
		models.ComplianceAudit
		TriggerScan bool `json:"trigger_scan,omitempty"` // Optional flag to trigger security scan
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	compliance := request.ComplianceAudit
	compliance.ArtifactID = id

	if err := s.complianceService.CreateAudit(&compliance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Optionally trigger a comprehensive security scan
	if request.TriggerScan && s.scanService != nil {
		// Check if scan is already in progress
		if existingScan, err := s.scanService.GetActiveScan(id); err != nil || existingScan == nil {
			// Create and start security scan (artifact already fetched above)
			scan := &models.SecurityScan{
				ArtifactID:        id,
				Status:            "initiated",
				ScanType:          "compliance",
				Priority:          "normal",
				VulnerabilityScan: true,
				MalwareScan:       true,
				LicenseScan:       true,
				DependencyScan:    true,
				InitiatedBy:       nil, // Compliance-triggered scan
				StartedAt:         time.Now(),
			}

			if err := s.scanService.CreateScan(scan); err == nil {
				// Start scan asynchronously
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
					defer cancel()

					if err := s.scanService.ScanArtifact(ctx, artifact, scan); err != nil {
						s.logger.Printf("Compliance-triggered scan failed: %v", err)
					}
				}()
			}
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"compliance":     compliance,
		"scan_triggered": request.TriggerScan,
	})
}

func (s *Server) handleUpdateCompliance(c *gin.Context) {
	// Compliance is now auto-calculated from security scans and cannot be manually updated
	c.JSON(http.StatusForbidden, gin.H{
		"error":   "Manual compliance updates are disabled",
		"message": "Compliance status is automatically calculated from security scan results. Run a security scan to update compliance.",
		"hint":    "POST /api/v1/scans with artifact_id to trigger a new security scan",
	})
}

func (s *Server) handleGetVulnerabilities(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Parse query parameters for filtering (for new scan system compatibility)
	severity := c.Query("severity")
	status := c.Query("status")

	// Try to get detailed vulnerabilities from new scan system first
	if s.scanService != nil {
		detailedVulns, err := s.scanService.GetVulnerabilities(id, severity, status)
		if err == nil && len(detailedVulns) > 0 {
			// Group vulnerabilities by severity for compatibility
			severityCount := make(map[string]int)
			for _, vuln := range detailedVulns {
				severityCount[vuln.Severity]++
			}

			// Return new format with detailed data
			c.JSON(http.StatusOK, gin.H{
				"vulnerabilities": detailedVulns,
				"total_count":     len(detailedVulns),
				"severity_count":  severityCount,
				"source":          "scan_system", // Indicate source
				"has_details":     true,
			})
			return
		}
	}

	// Fallback to legacy vulnerability system
	vulnerabilities, err := s.complianceService.GetVulnerabilities(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Return legacy format
	c.JSON(http.StatusOK, gin.H{
		"vulnerabilities": vulnerabilities,
		"source":          "legacy_system",
		"has_details":     false,
	})
}

func (s *Server) handleComplianceReport(c *gin.Context) {
	report, err := s.complianceService.GenerateReport()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, report)
}
