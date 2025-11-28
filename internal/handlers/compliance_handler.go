package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// ComplianceReportInfo represents a compliance report metadata
type ComplianceReportInfo struct {
	ID           int64     `json:"id"`
	ReportFormat string    `json:"report_format"`
	ReportType   string    `json:"report_type"`
	GeneratedAt  time.Time `json:"generated_at"`
}

// ComplianceHandler handles compliance dashboard and reporting using Gin
type ComplianceHandler struct {
	auditSvc      *service.AuditService
	complianceSvc *service.CompliancePolicyService
	db            *sql.DB
}

// NewComplianceHandler creates a new compliance handler for Gin
func NewComplianceHandler(auditSvc *service.AuditService, complianceSvc *service.CompliancePolicyService, db *sql.DB) *ComplianceHandler {
	return &ComplianceHandler{
		auditSvc:      auditSvc,
		complianceSvc: complianceSvc,
		db:            db,
	}
}

// RegisterRoutes registers compliance routes
func (h *ComplianceHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Dashboard and Reports
	rg.GET("/compliance/dashboard", h.GetDashboardData)
	rg.POST("/compliance/generate-report", h.GenerateReport)
	rg.GET("/compliance/reports", h.GetReports)
	rg.GET("/compliance/reports/:id/download", h.DownloadReport)

	// Policy Management
	rg.GET("/compliance/policies", h.GetCompliancePolicies)
	rg.POST("/compliance/policies", h.CreateCompliancePolicy)
	rg.GET("/compliance/policies/artifact-type/:type", h.GetPoliciesByArtifactType)
	rg.POST("/compliance/retention/apply", h.ApplyRetentionPolicies)

	// Stats
	rg.GET("/compliance/stats", h.GetComplianceStats)

	// Audit Logs
	rg.GET("/compliance/audit-logs", h.GetComplianceAuditLogs)

	// Data Erasure Requests
	rg.POST("/compliance/erase-request", h.CreateDataErasureRequest)
	rg.GET("/compliance/erasure-requests", h.GetDataErasureRequests)
	rg.POST("/compliance/erasure-requests/:id/approve", h.ApproveDataErasureRequest)
	rg.POST("/compliance/erasure-requests/:id/reject", h.RejectDataErasureRequest)

	// Legal Holds
	rg.GET("/compliance/legal-holds", h.GetLegalHolds)
	rg.POST("/compliance/legal-holds", h.CreateLegalHold)
	rg.PUT("/compliance/legal-holds/:id", h.UpdateLegalHold)
	rg.DELETE("/compliance/legal-holds/:id", h.DeleteLegalHold)

	// Scheduler Management
	rg.GET("/compliance/scheduler/status", h.GetSchedulerStatus)
	rg.POST("/compliance/scheduler/trigger", h.TriggerSchedulerJob)
}

// GetDashboardData handles GET /compliance/dashboard
func (h *ComplianceHandler) GetDashboardData(c *gin.Context) {
	// Parse time range parameter
	timeRange := c.DefaultQuery("timeRange", "7d")

	// Calculate start and end dates
	endDate := time.Now()
	var startDate time.Time

	switch timeRange {
	case "1d":
		startDate = endDate.Add(-24 * time.Hour)
	case "7d":
		startDate = endDate.Add(-7 * 24 * time.Hour)
	case "30d":
		startDate = endDate.Add(-30 * 24 * time.Hour)
	case "90d":
		startDate = endDate.Add(-90 * 24 * time.Hour)
	default:
		startDate = endDate.Add(-7 * 24 * time.Hour)
	}

	// Get policy decision statistics
	policyStats, err := h.getPolicyDecisionStats(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get policy statistics"})
		return
	}

	// Get security violation statistics
	violationStats, err := h.getViolationStats(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get violation statistics"})
		return
	}

	// Calculate compliance score
	complianceScore := h.calculateComplianceScore(policyStats, violationStats)

	// Get compliance metrics
	complianceMetrics, err := h.getComplianceMetrics(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get compliance metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"compliance_score":    complianceScore,
		"policy_decisions":    policyStats,
		"security_violations": violationStats,
		"compliance_metrics":  complianceMetrics,
		"time_range":          timeRange,
		"start_date":          startDate,
		"end_date":            endDate,
		"last_updated":        time.Now(),
	})
}

// GenerateReport handles POST /compliance/generate-report
func (h *ComplianceHandler) GenerateReport(c *gin.Context) {
	var request struct {
		StartDate   time.Time `json:"start_date"`
		EndDate     time.Time `json:"end_date"`
		GeneratedBy int64     `json:"generated_by"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Generate compliance report
	report, err := h.auditSvc.GenerateComplianceReport(c.Request.Context(), request.StartDate, request.EndDate, request.GeneratedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate compliance report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"report": gin.H{
			"id":            report.ID,
			"report_type":   report.ReportType,
			"report_format": report.ReportFormat,
			"generated_at":  report.GeneratedAt,
			"generated_by":  report.GeneratedBy,
			"parameters":    report.Parameters,
		},
	})
}

// GetReports handles GET /compliance/reports
func (h *ComplianceHandler) GetReports(c *gin.Context) {
	// Get pagination parameters
	page := 1
	limit := 20

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := (page - 1) * limit

	// Fetch reports from database
	reports, err := h.getComplianceReports(offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reports"})
		return
	}

	// Get total count
	totalCount, err := h.getReportCount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get report count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reports": reports,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": (totalCount + limit - 1) / limit,
		},
	})
}

// DownloadReport handles GET /compliance/reports/:id/download
func (h *ComplianceHandler) DownloadReport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report ID"})
		return
	}

	// Get report info
	report, err := h.getComplianceReportByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get report"})
		return
	}

	// Set headers for file download
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="compliance-report-%d.%s"`, id, report.ReportFormat))
	c.Header("Content-Type", getContentType(report.ReportFormat))

	// Return the report data - would need actual file content from DB
	c.String(http.StatusOK, "Report data would be here")
}

// GetCompliancePolicies handles GET /compliance/policies
func (h *ComplianceHandler) GetCompliancePolicies(c *gin.Context) {
	// Parse query parameters for filtering
	policyType := c.Query("type")

	policies, err := h.complianceSvc.GetPolicies(c.Request.Context(), policyType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get policies: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"policies": policies,
		"total":    len(policies),
	})
}

// CreateCompliancePolicy handles POST /compliance/policies
func (h *ComplianceHandler) CreateCompliancePolicy(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Type        string `json:"type"`
		Status      string `json:"status"`
		Rules       string `json:"rules"`
		Region      string `json:"region"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.Name == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name and type are required"})
		return
	}

	// Set defaults
	if req.Status == "" {
		req.Status = "draft"
	}
	if req.Region == "" {
		req.Region = "GLOBAL"
	}

	policy := &models.CompliancePolicy{
		Name:        req.Name,
		Type:        req.Type,
		Status:      req.Status,
		Rules:       req.Rules,
		Region:      req.Region,
		Description: req.Description,
	}

	// Get user ID from context
	userIDVal, exists := c.Get("user_id")
	var userID string
	if exists {
		if uid, ok := userIDVal.(uuid.UUID); ok {
			userID = uid.String()
		}
	}

	err := h.complianceSvc.CreatePolicy(c.Request.Context(), policy, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create policy: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, policy)
}

// GetPoliciesByArtifactType handles GET /compliance/policies/artifact-type/:type
func (h *ComplianceHandler) GetPoliciesByArtifactType(c *gin.Context) {
	artifactType := c.Param("type")
	if artifactType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Artifact type is required"})
		return
	}

	// Mock response - replace with actual database query
	policies := []map[string]interface{}{
		{
			"id":               1,
			"name":             fmt.Sprintf("Retention Policy for %s", artifactType),
			"type":             "retention",
			"artifact_type":    artifactType,
			"retention_period": "90d",
			"status":           "active",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"policies":      policies,
		"artifact_type": artifactType,
		"total":         len(policies),
	})
}

// ApplyRetentionPolicies handles POST /compliance/retention/apply
func (h *ComplianceHandler) ApplyRetentionPolicies(c *gin.Context) {
	err := h.complianceSvc.EnforceDataRetention(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to apply retention policies: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Data retention policies applied successfully",
		"processed": 0, // Would need proper tracking
	})
}

// GetComplianceStats handles GET /compliance/stats
func (h *ComplianceHandler) GetComplianceStats(c *gin.Context) {
	// Mock stats - replace with actual database queries
	stats := gin.H{
		"total_policies":       7,
		"active_policies":      5,
		"total_violations":     3,
		"pending_erasures":     2,
		"active_legal_holds":   1,
		"compliance_score":     87.5,
		"last_audit":           time.Now().Add(-2 * time.Hour),
		"next_scheduled_audit": time.Now().Add(22 * time.Hour),
	}

	c.JSON(http.StatusOK, stats)
}

// GetComplianceAuditLogs handles GET /compliance/audit-logs
func (h *ComplianceHandler) GetComplianceAuditLogs(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	// Mock logs - replace with actual audit log queries
	logs := []map[string]interface{}{}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": len(logs),
		"limit": limit,
	})
}

// CreateDataErasureRequest handles POST /compliance/erase-request
func (h *ComplianceHandler) CreateDataErasureRequest(c *gin.Context) {
	var req struct {
		DataSubject   string  `json:"data_subject"`
		ContactEmail  string  `json:"contact_email"`
		ArtifactIDs   []int64 `json:"artifact_ids"`
		Reason        string  `json:"reason"`
		RequestSource string  `json:"request_source"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.DataSubject == "" || len(req.ArtifactIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data subject and artifact IDs are required"})
		return
	}

	// Get user ID from context
	userIDVal, exists := c.Get("user_id")
	var userID string
	if exists {
		if uid, ok := userIDVal.(uuid.UUID); ok {
			userID = uid.String()
		}
	}

	err := h.complianceSvc.RequestDataErasure(c.Request.Context(), userID, req.ArtifactIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create erasure request: %v", err)})
		return
	}

	request := gin.H{
		"id":           1,
		"data_subject": req.DataSubject,
		"artifact_ids": req.ArtifactIDs,
		"status":       "pending",
		"created_at":   time.Now(),
	}

	c.JSON(http.StatusCreated, request)
}

// GetDataErasureRequests handles GET /compliance/erasure-requests
func (h *ComplianceHandler) GetDataErasureRequests(c *gin.Context) {
	status := c.Query("status")
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	// Mock data - replace with actual database queries
	allRequests := []map[string]interface{}{
		{
			"id":           1,
			"request_type": "gdpr_erasure",
			"subject_id":   "user123",
			"reason":       "User requested account deletion",
			"status":       "pending",
			"created_at":   "2024-10-15T10:00:00Z",
			"artifacts":    []string{"artifact-456", "artifact-789"},
		},
		{
			"id":           2,
			"request_type": "retention_expired",
			"subject_id":   "auto-system",
			"reason":       "Automatic cleanup - retention period expired",
			"status":       "approved",
			"created_at":   "2024-10-10T08:00:00Z",
			"processed_at": "2024-10-12T09:15:00Z",
			"artifacts":    []string{"artifact-123"},
		},
	}

	// Filter by status
	var filteredRequests []map[string]interface{}
	for _, req := range allRequests {
		if status == "" || req["status"] == status {
			filteredRequests = append(filteredRequests, req)
		}
	}

	// Apply pagination
	total := len(filteredRequests)
	end := offset + limit
	if end > total {
		end = total
	}

	var paginatedRequests []map[string]interface{}
	if offset < total {
		paginatedRequests = filteredRequests[offset:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"erasure_requests": paginatedRequests,
		"total":            total,
		"limit":            limit,
		"offset":           offset,
	})
}

// ApproveDataErasureRequest handles POST /compliance/erasure-requests/:id/approve
func (h *ComplianceHandler) ApproveDataErasureRequest(c *gin.Context) {
	requestID := c.Param("id")

	// Get user ID from context
	userIDVal, _ := c.Get("user_id")
	userID := fmt.Sprintf("%v", userIDVal)

	// Mock approval - replace with actual logic
	c.JSON(http.StatusOK, gin.H{
		"message":     "Data erasure request approved successfully",
		"request_id":  requestID,
		"status":      "approved",
		"approved_at": time.Now().Format(time.RFC3339),
		"approved_by": userID,
	})
}

// RejectDataErasureRequest handles POST /compliance/erasure-requests/:id/reject
func (h *ComplianceHandler) RejectDataErasureRequest(c *gin.Context) {
	requestID := c.Param("id")

	var rejectionData struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&rejectionData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection reason is required"})
		return
	}

	// Get user ID from context
	userIDVal, _ := c.Get("user_id")
	userID := fmt.Sprintf("%v", userIDVal)

	// Mock rejection - replace with actual logic
	c.JSON(http.StatusOK, gin.H{
		"message":     "Data erasure request rejected successfully",
		"request_id":  requestID,
		"status":      "rejected",
		"rejected_at": time.Now().Format(time.RFC3339),
		"rejected_by": userID,
		"reason":      rejectionData.Reason,
	})
}

// GetLegalHolds handles GET /compliance/legal-holds
func (h *ComplianceHandler) GetLegalHolds(c *gin.Context) {
	status := c.Query("status")
	limit := 10
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	// Mock data - replace with actual database queries
	allHolds := []map[string]interface{}{
		{
			"id":          "hold-001",
			"case_number": "CASE-2024-001",
			"description": "SEC Investigation",
			"status":      "active",
			"created_at":  "2024-09-01T00:00:00Z",
			"artifacts":   []string{"artifact-123", "artifact-456"},
		},
		{
			"id":          "hold-002",
			"case_number": "CASE-2024-002",
			"description": "Litigation Hold",
			"status":      "released",
			"created_at":  "2024-08-15T00:00:00Z",
			"released_at": "2024-10-01T00:00:00Z",
			"artifacts":   []string{"artifact-789"},
		},
	}

	// Filter by status
	var filteredHolds []map[string]interface{}
	for _, hold := range allHolds {
		if status == "" || hold["status"] == status {
			filteredHolds = append(filteredHolds, hold)
		}
	}

	// Apply pagination
	total := len(filteredHolds)
	end := offset + limit
	if end > total {
		end = total
	}

	var paginatedHolds []map[string]interface{}
	if offset < total {
		paginatedHolds = filteredHolds[offset:end]
	}

	c.JSON(http.StatusOK, gin.H{
		"legal_holds": paginatedHolds,
		"total":       total,
		"limit":       limit,
		"offset":      offset,
	})
}

// CreateLegalHold handles POST /compliance/legal-holds
func (h *ComplianceHandler) CreateLegalHold(c *gin.Context) {
	var req struct {
		CaseNumber  string   `json:"case_number" binding:"required"`
		Description string   `json:"description"`
		ArtifactIDs []string `json:"artifact_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Mock creation - replace with actual logic
	hold := gin.H{
		"id":          fmt.Sprintf("hold-%d", time.Now().Unix()),
		"case_number": req.CaseNumber,
		"description": req.Description,
		"status":      "active",
		"created_at":  time.Now(),
		"artifacts":   req.ArtifactIDs,
	}

	c.JSON(http.StatusCreated, hold)
}

// UpdateLegalHold handles PUT /compliance/legal-holds/:id
func (h *ComplianceHandler) UpdateLegalHold(c *gin.Context) {
	holdID := c.Param("id")

	var req struct {
		Description string   `json:"description"`
		Status      string   `json:"status"`
		ArtifactIDs []string `json:"artifact_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Mock update - replace with actual logic
	hold := gin.H{
		"id":          holdID,
		"description": req.Description,
		"status":      req.Status,
		"updated_at":  time.Now(),
		"artifacts":   req.ArtifactIDs,
	}

	c.JSON(http.StatusOK, hold)
}

// DeleteLegalHold handles DELETE /compliance/legal-holds/:id
func (h *ComplianceHandler) DeleteLegalHold(c *gin.Context) {
	holdID := c.Param("id")

	// Mock deletion - replace with actual logic
	c.JSON(http.StatusOK, gin.H{
		"message": "Legal hold deleted successfully",
		"id":      holdID,
	})
}

// GetSchedulerStatus handles GET /compliance/scheduler/status
func (h *ComplianceHandler) GetSchedulerStatus(c *gin.Context) {
	// Mock status - would need actual scheduler reference
	status := gin.H{
		"scheduler_available": true,
		"is_running":          true,
		"jobs": gin.H{
			"retention_enforcement": gin.H{
				"enabled":  true,
				"last_run": time.Now().Add(-2 * time.Hour),
				"next_run": time.Now().Add(22 * time.Hour),
				"interval": "24h",
			},
			"erasure_processing": gin.H{
				"enabled":  true,
				"last_run": time.Now().Add(-30 * time.Minute),
				"next_run": time.Now().Add(5*time.Hour + 30*time.Minute),
				"interval": "6h",
			},
		},
	}

	c.JSON(http.StatusOK, status)
}

// TriggerSchedulerJob handles POST /compliance/scheduler/trigger
func (h *ComplianceHandler) TriggerSchedulerJob(c *gin.Context) {
	var request struct {
		JobType string `json:"job_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job_type is required"})
		return
	}

	// Mock trigger - would need actual scheduler reference
	c.JSON(http.StatusOK, gin.H{
		"message":  "Job triggered successfully",
		"job_type": request.JobType,
		"status":   "triggered",
	})
}

// Helper methods (internal)

func (h *ComplianceHandler) getPolicyDecisionStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	// Mock stats - replace with actual database queries
	return map[string]interface{}{
		"total":   100,
		"allowed": 85,
		"blocked": 15,
		"by_type": map[string]int{
			"retention": 50,
			"access":    30,
			"erasure":   20,
		},
	}, nil
}

func (h *ComplianceHandler) getViolationStats(startDate, endDate time.Time) (map[string]interface{}, error) {
	// Mock stats
	return map[string]interface{}{
		"total":    5,
		"critical": 1,
		"high":     2,
		"medium":   2,
		"low":      0,
		"by_category": map[string]int{
			"data_retention": 2,
			"access_control": 2,
			"encryption":     1,
		},
	}, nil
}

func (h *ComplianceHandler) calculateComplianceScore(policyStats, violationStats map[string]interface{}) float64 {
	// Simple mock calculation
	return 87.5
}

func (h *ComplianceHandler) getComplianceMetrics(startDate, endDate time.Time) (map[string]interface{}, error) {
	// Mock metrics
	return map[string]interface{}{
		"total_artifacts":     1000,
		"compliant_artifacts": 875,
		"pending_review":      100,
		"violations":          25,
	}, nil
}

func (h *ComplianceHandler) getComplianceReports(offset, limit int) ([]map[string]interface{}, error) {
	// Mock reports
	return []map[string]interface{}{
		{
			"id":            1,
			"report_type":   "compliance_summary",
			"report_format": "pdf",
			"generated_at":  time.Now().Add(-24 * time.Hour),
			"status":        "completed",
		},
	}, nil
}

func (h *ComplianceHandler) getReportCount() (int, error) {
	return 1, nil
}

func (h *ComplianceHandler) getComplianceReportByID(id int64) (*ComplianceReportInfo, error) {
	// Mock report - use the existing ComplianceReportInfo from compliance_handler.go
	if id == 1 {
		return &ComplianceReportInfo{
			ID:           1,
			ReportFormat: "pdf",
			ReportType:   "compliance_summary",
			GeneratedAt:  time.Now(),
		}, nil
	}
	return nil, sql.ErrNoRows
}

func getContentType(format string) string {
	switch strings.ToLower(format) {
	case "pdf":
		return "application/pdf"
	case "csv":
		return "text/csv"
	case "json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}
