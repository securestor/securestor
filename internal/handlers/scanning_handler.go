package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// PolicyEvaluator is an interface for policy evaluation to avoid import cycles
// It wraps the actual PolicyService from the api package
type PolicyEvaluator interface {
	EvaluateArtifactPolicyFromMap(ctx context.Context, artifact *models.Artifact, repo *models.Repository, userID string) (*PolicyDecisionResult, error)
}

// PolicyDecisionResult represents a policy decision (mirrors models.PolicyDecision)
type PolicyDecisionResult struct {
	Allow     bool   `json:"allow"`
	Action    string `json:"action"`
	RiskScore int    `json:"risk_score"`
	RiskLevel string `json:"risk_level"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

// ScanningHandler handles security scanning operations using Gin
type ScanningHandler struct {
	scanService       *service.ScanService
	artifactService   *service.ArtifactService
	repositoryService *service.RepositoryService
	policyEvaluator   PolicyEvaluator
	db                *sql.DB
	logger            *log.Logger
}

// NewScanningHandler creates a new ScanningHandler
func NewScanningHandler(
	scanService *service.ScanService,
	artifactService *service.ArtifactService,
	repositoryService *service.RepositoryService,
	policyEvaluator PolicyEvaluator,
	db *sql.DB,
	logger *log.Logger,
) *ScanningHandler {
	return &ScanningHandler{
		scanService:       scanService,
		artifactService:   artifactService,
		repositoryService: repositoryService,
		policyEvaluator:   policyEvaluator,
		db:                db,
		logger:            logger,
	}
}

// RegisterRoutes registers all scanning routes
func (h *ScanningHandler) RegisterRoutes(rg *gin.RouterGroup) {
	scanning := rg.Group("/scanning")
	{
		// Scan management
		scanning.GET("", h.GetAllScans)
		scanning.POST("", h.CreateScan)
		scanning.POST("/bulk", h.BulkScan)

		// Scan operations
		scanning.GET("/:scanId", h.GetScanByID)
		scanning.POST("/:scanId/cancel", h.CancelScan)

		// Artifact scanning
		scanning.POST("/artifacts/:id/scan", h.ScanArtifact)
		scanning.GET("/artifacts/:id/results", h.GetScanResult)
		scanning.GET("/artifacts/:id/history", h.GetScanHistory)

		// Scanner management
		scanning.GET("/scanners", h.GetAvailableScanners)
		scanning.GET("/scanners/health", h.ScannerHealth)

		// Security dashboard & reports
		scanning.GET("/dashboard", h.GetSecurityDashboard)
		scanning.GET("/vulnerable-artifacts", h.ListVulnerableArtifacts)
		scanning.GET("/export-report", h.ExportSecurityReport)
	}
}

// ========== Category 1: Scan Management ==========

// GetAllScans retrieves all scans with optional filtering
func (h *ScanningHandler) GetAllScans(c *gin.Context) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	// Parse query parameters
	status := c.Query("status")
	scanType := c.Query("type")
	priority := c.Query("priority")

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	scans, total, err := h.scanService.GetAllScansByTenant(tenantID, status, scanType, priority, limit, offset)
	if err != nil {
		h.logger.Printf("ERROR: Failed to retrieve scans for tenant %s: %v", tenantID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scans"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scans":       scans,
		"total_count": total,
		"limit":       limit,
		"offset":      offset,
	})
}

// CreateScan creates a new security scan
func (h *ScanningHandler) CreateScan(c *gin.Context) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	var request struct {
		ArtifactID        uuid.UUID `json:"artifact_id"`
		ScanType          string    `json:"scan_type"`
		Priority          string    `json:"priority"`
		VulnerabilityScan bool      `json:"vulnerability_scan"`
		MalwareScan       bool      `json:"malware_scan"`
		LicenseScan       bool      `json:"license_scan"`
		DependencyScan    bool      `json:"dependency_scan"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	emptyUUID := uuid.UUID{}
	if request.ArtifactID == emptyUUID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact - verify it belongs to tenant
	artifact, err := h.artifactService.GetByIDAndTenant(request.ArtifactID, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Check if scan is already in progress
	if existingScan, err := h.scanService.GetActiveScan(request.ArtifactID); err == nil && existingScan != nil {
		c.JSON(http.StatusConflict, gin.H{
			"message":    "Scan already in progress",
			"scan_id":    existingScan.ID,
			"status":     existingScan.Status,
			"started_at": existingScan.StartedAt,
		})
		return
	}

	// Set defaults if not provided
	if request.ScanType == "" {
		request.ScanType = "full"
	}
	if request.Priority == "" {
		request.Priority = "normal"
	}

	// Create scan record
	scan := &models.SecurityScan{
		TenantID:          tenantID,
		ArtifactID:        request.ArtifactID,
		Status:            "initiated",
		ScanType:          request.ScanType,
		Priority:          request.Priority,
		VulnerabilityScan: request.VulnerabilityScan,
		MalwareScan:       request.MalwareScan,
		LicenseScan:       request.LicenseScan,
		DependencyScan:    request.DependencyScan,
		InitiatedBy:       nil, // TODO: Get from auth context
		StartedAt:         time.Now(),
	}

	if err := h.scanService.CreateScan(scan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scan record"})
		return
	}

	// Start scan asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := h.scanService.ScanArtifact(ctx, artifact, scan); err != nil {
			h.logger.Printf("Scan failed: %v", err)
			scan.Status = "failed"
			scan.ErrorMessage = stringPtr(err.Error())
			now := time.Now()
			scan.CompletedAt = &now
			h.scanService.UpdateScan(scan)
		}
	}()

	c.JSON(http.StatusCreated, scan)
}

// BulkScan initiates scans for multiple artifacts
func (h *ScanningHandler) BulkScan(c *gin.Context) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	var request struct {
		ArtifactIDs []uuid.UUID       `json:"artifact_ids"`
		Config      models.ScanConfig `json:"config"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(request.ArtifactIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No artifact IDs provided"})
		return
	}

	if len(request.ArtifactIDs) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Too many artifacts (max 50)"})
		return
	}

	results := make([]map[string]interface{}, 0, len(request.ArtifactIDs))

	for _, artifactID := range request.ArtifactIDs {
		// Check if artifact exists and belongs to tenant
		artifact, err := h.artifactService.GetByIDAndTenant(artifactID, tenantID)
		if err != nil {
			results = append(results, map[string]interface{}{
				"artifact_id": artifactID,
				"status":      "failed",
				"error":       "Artifact not found",
			})
			continue
		}

		// Check if scan is already in progress
		if existingScan, err := h.scanService.GetActiveScan(artifactID); err == nil && existingScan != nil {
			results = append(results, map[string]interface{}{
				"artifact_id": artifactID,
				"status":      "skipped",
				"message":     "Scan already in progress",
				"scan_id":     existingScan.ID,
			})
			continue
		}

		// Create scan record
		scan := &models.SecurityScan{
			TenantID:          tenantID,
			ArtifactID:        artifactID,
			Status:            "initiated",
			ScanType:          "bulk",
			Priority:          request.Config.Priority,
			VulnerabilityScan: request.Config.VulnerabilityScan,
			MalwareScan:       request.Config.MalwareScan,
			LicenseScan:       request.Config.LicenseScan,
			DependencyScan:    request.Config.DependencyScan,
			InitiatedBy:       nil, // TODO: Get from auth context
			StartedAt:         time.Now(),
		}

		if err := h.scanService.CreateScan(scan); err != nil {
			results = append(results, map[string]interface{}{
				"artifact_id": artifactID,
				"status":      "failed",
				"error":       "Failed to create scan record",
			})
			continue
		}

		// Start scan asynchronously
		go func(art *models.Artifact, sc *models.SecurityScan) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()

			if err := h.scanService.ScanArtifact(ctx, art, sc); err != nil {
				h.logger.Printf("Bulk scan failed: %v", err)
				sc.Status = "failed"
				sc.ErrorMessage = stringPtr(err.Error())
				now := time.Now()
				sc.CompletedAt = &now
				h.scanService.UpdateScan(sc)
			}
		}(artifact, scan)

		results = append(results, map[string]interface{}{
			"artifact_id": artifactID,
			"status":      "initiated",
			"scan_id":     scan.ID,
		})
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Bulk scan initiated",
		"results": results,
		"total":   len(results),
	})
}

// ========== Category 2: Scan Operations ==========

// GetScanByID retrieves a specific scan by its ID
func (h *ScanningHandler) GetScanByID(c *gin.Context) {
	scanIDStr := c.Param("scanId")

	// Try to parse as UUID
	scanID, err := uuid.Parse(scanIDStr)
	if err != nil {
		// Fall back to numeric ID for backward compatibility
		_, errInt := strconv.ParseInt(scanIDStr, 10, 64)
		if errInt != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan ID"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}

	scan, err := h.scanService.GetScanByUUID(scanID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}

	c.JSON(http.StatusOK, scan)
}

// CancelScan cancels an active scan
func (h *ScanningHandler) CancelScan(c *gin.Context) {
	scanIDStr := c.Param("scanId")

	// Try to parse as UUID
	scanID, err := uuid.Parse(scanIDStr)
	if err != nil {
		// Fall back to numeric ID for backward compatibility
		_, errInt := strconv.ParseInt(scanIDStr, 10, 64)
		if errInt != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan ID"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan not found"})
		return
	}

	if err := h.scanService.CancelScanByUUID(scanID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel scan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scan cancelled successfully",
		"scan_id": scanID,
	})
}

// ========== Category 3: Artifact Scanning ==========

// ScanArtifact initiates a security scan for an artifact
func (h *ScanningHandler) ScanArtifact(c *gin.Context) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Parse scan configuration from request body
	var scanConfig models.ScanConfig
	if err := c.ShouldBindJSON(&scanConfig); err != nil {
		// Use default config if parsing fails
		scanConfig = models.ScanConfig{
			VulnerabilityScan: true,
			MalwareScan:       true,
			LicenseScan:       true,
			DependencyScan:    true,
			Priority:          "normal",
		}
	}

	// Get artifact - verify it belongs to tenant
	artifact, err := h.artifactService.GetByIDAndTenant(id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Check if scan is already in progress
	if existingScan, err := h.scanService.GetActiveScan(id); err == nil && existingScan != nil {
		c.JSON(http.StatusConflict, gin.H{
			"message":    "Scan already in progress",
			"scan_id":    existingScan.ID,
			"status":     existingScan.Status,
			"started_at": existingScan.StartedAt,
		})
		return
	}

	// Create scan record
	scan := &models.SecurityScan{
		TenantID:          tenantID,
		ArtifactID:        id,
		Status:            "initiated",
		ScanType:          "full",
		Priority:          scanConfig.Priority,
		VulnerabilityScan: scanConfig.VulnerabilityScan,
		MalwareScan:       scanConfig.MalwareScan,
		LicenseScan:       scanConfig.LicenseScan,
		DependencyScan:    scanConfig.DependencyScan,
		InitiatedBy:       nil, // TODO: Get from auth context
		StartedAt:         time.Now(),
	}

	if err := h.scanService.CreateScan(scan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scan record"})
		return
	}

	// Start scan asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := h.scanService.ScanArtifact(ctx, artifact, scan); err != nil {
			h.logger.Printf("Scan failed: %v", err)
			scan.Status = "failed"
			scan.ErrorMessage = stringPtr(err.Error())
			now := time.Now()
			scan.CompletedAt = &now
			h.scanService.UpdateScan(scan)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Scan initiated successfully",
		"scan_id":     scan.ID,
		"artifact_id": id,
		"status":      scan.Status,
		"started_at":  scan.StartedAt,
	})
}

// GetScanResult retrieves scan results for an artifact
func (h *ScanningHandler) GetScanResult(c *gin.Context) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	result, err := h.scanService.GetLatestScanResultByTenant(id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan result not found"})
		return
	}

	// Enhance scan result with OPA policy evaluation
	if result != nil && result.Results != nil {
		// Get artifact and repository for policy context
		artifact, err := h.artifactService.GetByIDAndTenant(id, tenantID)
		if err == nil {
			repo, err := h.repositoryService.GetByID(artifact.RepositoryID)
			if err == nil {
				// Update artifact metadata with latest scan results
				if artifact.Metadata == nil {
					artifact.Metadata = make(map[string]interface{})
				}

				// Add vulnerability info to metadata for policy evaluation
				if result.Results.VulnerabilityResults != nil {
					artifact.Metadata["vulnerabilities"] = map[string]interface{}{
						"critical": result.Results.VulnerabilityResults.Critical,
						"high":     result.Results.VulnerabilityResults.High,
						"medium":   result.Results.VulnerabilityResults.Medium,
						"low":      result.Results.VulnerabilityResults.Low,
					}
				}

				// Add scan results to metadata
				malwareDetected := false
				licenseCompliant := true
				dependencyIssues := []string{}

				if result.Results.MalwareResults != nil {
					malwareDetected = result.Results.MalwareResults.ThreatsFound > 0
				}
				if result.Results.LicenseResults != nil {
					licenseCompliant = result.Results.LicenseResults.NonCompliantLicenses == 0
				}
				if result.Results.DependencyResults != nil {
					for _, advisory := range result.Results.DependencyResults.SecurityAdvisories {
						dependencyIssues = append(dependencyIssues, advisory.Title)
					}
				}

				artifact.Metadata["scan_results"] = map[string]interface{}{
					"malware_detected":  malwareDetected,
					"license_compliant": licenseCompliant,
					"dependency_issues": dependencyIssues,
					"last_scanned":      result.CompletedAt.Format(time.RFC3339),
				}

				// Evaluate OPA policy with updated scan data
				userID := "admin" // TODO: Get from auth context

				ctx := context.Background()
				policyDecision, err := h.policyEvaluator.EvaluateArtifactPolicyFromMap(ctx, artifact, repo, userID)
				if err != nil {
					h.logger.Printf("Policy evaluation failed for scan result %s: %v", id, err)
				} else {
					// Add policy decision to scan result
					result.Results.PolicyDecision = &models.PolicyDecision{
						Allow:     policyDecision.Allow,
						Action:    policyDecision.Action,
						RiskScore: policyDecision.RiskScore,
						RiskLevel: policyDecision.RiskLevel,
						Reason:    policyDecision.Reason,
						Timestamp: policyDecision.Timestamp,
					}

					h.logger.Printf("Policy evaluation for scan %s: Allow=%t, Action=%s, RiskScore=%d",
						id, policyDecision.Allow, policyDecision.Action, policyDecision.RiskScore)
				}
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetScanHistory retrieves scan history for an artifact
func (h *ScanningHandler) GetScanHistory(c *gin.Context) {
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}
	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant context"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Verify artifact belongs to tenant
	_, err = h.artifactService.GetByIDAndTenant(id, tenantID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: artifact not found or belongs to different tenant"})
		return
	}

	// Parse query parameters
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	history, total, err := h.scanService.GetScanHistory(id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve scan history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scans":       history,
		"total_count": total,
		"limit":       limit,
		"offset":      offset,
	})
}

// ========== Category 4: Scanner Management ==========

// GetAvailableScanners returns information about available security scanners
func (h *ScanningHandler) GetAvailableScanners(c *gin.Context) {
	scanners := h.scanService.GetAvailableScanners()
	c.JSON(http.StatusOK, gin.H{
		"scanners": scanners,
		"total":    len(scanners),
	})
}

// ScannerHealth checks the health status of all security scanners
func (h *ScanningHandler) ScannerHealth(c *gin.Context) {
	health := h.scanService.CheckScannerHealth()

	allHealthy := true
	for _, status := range health {
		if status.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	statusCode := http.StatusOK
	if !allHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"overall_status": gin.H{"healthy": allHealthy},
		"scanners":       health,
		"timestamp":      time.Now(),
	})
}

// ========== Category 5: Security Dashboard & Reports ==========

// GetSecurityDashboard returns security dashboard statistics
func (h *ScanningHandler) GetSecurityDashboard(c *gin.Context) {
	stats, err := h.scanService.GetSecurityDashboard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ListVulnerableArtifacts returns all artifacts with vulnerabilities
func (h *ScanningHandler) ListVulnerableArtifacts(c *gin.Context) {
	severity := c.Query("severity")
	minCVSS := c.Query("min_cvss")

	artifacts, err := h.scanService.ListVulnerableArtifacts(severity, minCVSS)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifacts,
		"total":     len(artifacts),
	})
}

// ExportSecurityReport exports security report
func (h *ScanningHandler) ExportSecurityReport(c *gin.Context) {
	format := c.Query("format") // json, csv, pdf
	if format == "" {
		format = "json"
	}

	report, err := h.scanService.GenerateSecurityReport(format)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Set appropriate content type
	contentTypes := map[string]string{
		"json": "application/json",
		"csv":  "text/csv",
		"pdf":  "application/pdf",
	}

	c.Header("Content-Type", contentTypes[format])
	c.Header("Content-Disposition", "attachment; filename=security-report."+format)
	c.Data(http.StatusOK, contentTypes[format], report)
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
