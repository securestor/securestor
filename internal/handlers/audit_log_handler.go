package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/service"
)

// AuditLogHandler handles audit log API requests
type AuditLogHandler struct {
	auditLogService *service.AuditLogService
}

// NewAuditLogHandler creates a new audit log handler
func NewAuditLogHandler(auditLogService *service.AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{
		auditLogService: auditLogService,
	}
}

// RegisterRoutes registers audit log routes
func (h *AuditLogHandler) RegisterRoutes(rg *gin.RouterGroup) {
	audit := rg.Group("/audit")
	{
		audit.GET("/logs", h.GetAuditLogs)
		audit.GET("/stats", h.GetAuditStats)
	}
}

// GetAuditLogs handles GET /audit/logs - retrieve audit logs with filtering
func (h *AuditLogHandler) GetAuditLogs(c *gin.Context) {
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	eventType := c.Query("event_type")
	userID := c.Query("user_id")
	action := c.Query("action")
	resourceType := c.Query("resource_type")
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 50
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Parse date filters
	var startDate, endDate time.Time
	if startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = t
		}
	}
	if endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = t
		}
	}

	// Build filters
	filters := &service.AuditLogFilters{
		TenantID:     tenantID.String(),
		EventType:    eventType,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		StartTime:    startDate,
		EndTime:      endDate,
		Limit:        limit,
		Offset:       offset,
	}

	// Get audit logs from service
	logs, total, err := h.auditLogService.GetAuditLogs(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetAuditStats handles GET /audit/stats - retrieve audit statistics
func (h *AuditLogHandler) GetAuditStats(c *gin.Context) {
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

	// Parse time range parameter (default: last 7 days)
	daysStr := c.DefaultQuery("days", "7")
	days, _ := strconv.Atoi(daysStr)
	if days < 1 || days > 90 {
		days = 7
	}

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	// Get audit stats from service with tenant filtering
	stats, err := h.auditLogService.GetAuditStats(c.Request.Context(), startTime, endTime, tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit stats: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
