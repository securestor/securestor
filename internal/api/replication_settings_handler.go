package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// ReplicationSettingsHandler handles replication configuration endpoints
type ReplicationSettingsHandler struct {
	replicationConfigService *service.ReplicationConfigService
	tenantService            *service.TenantManagementService
	logger                   *log.Logger
}

// NewReplicationSettingsHandler creates a new handler instance
func NewReplicationSettingsHandler(configService *service.ReplicationConfigService, tenantService *service.TenantManagementService, logger *log.Logger) *ReplicationSettingsHandler {
	return &ReplicationSettingsHandler{
		replicationConfigService: configService,
		tenantService:            tenantService,
		logger:                   logger,
	}
}

// RegisterRoutes registers replication settings routes (Gin version)
func (h *ReplicationSettingsHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Root endpoint returns global config (for frontend compatibility)
	router.GET("", h.HandleGetGlobalReplicationConfig)

	// Global replication configuration routes
	router.GET("/config", h.HandleGetGlobalReplicationConfig)
	router.PUT("/config", h.HandleUpdateGlobalReplicationConfig)
	router.GET("/config/history", h.HandleGetConfigHistory)

	// Replication nodes routes
	router.GET("/nodes", h.HandleListReplicationNodes)
	router.POST("/nodes", h.HandleCreateReplicationNode)
	router.GET("/nodes/:id", h.HandleGetReplicationNode)
	router.PUT("/nodes/:id", h.HandleUpdateReplicationNode)
	router.DELETE("/nodes/:id", h.HandleDeleteReplicationNode)
	router.GET("/nodes/:id/health", h.HandleGetNodeHealth)

	// Repository-specific replication settings
	router.GET("/repositories/:id", h.HandleGetRepositoryReplicationSettings)
	router.PUT("/repositories/:id", h.HandleUpdateRepositoryReplicationSettings)
	router.GET("/repositories/:id/effective", h.HandleGetEffectiveReplicationConfig)
}

// HandleGetGlobalReplicationConfig returns the global replication configuration
func (h *ReplicationSettingsHandler) HandleGetGlobalReplicationConfig(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	config, err := h.replicationConfigService.GetGlobalConfig(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// HandleUpdateGlobalReplicationConfig updates the global replication configuration
func (h *ReplicationSettingsHandler) HandleUpdateGlobalReplicationConfig(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	var req models.CreateTenantReplicationConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	config, err := h.replicationConfigService.UpdateGlobalConfig(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// HandleListReplicationNodes returns all replication nodes
func (h *ReplicationSettingsHandler) HandleListReplicationNodes(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	nodes, err := h.replicationConfigService.GetReplicationNodes(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, nodes)
}

// HandleCreateReplicationNode creates a new replication node
func (h *ReplicationSettingsHandler) HandleCreateReplicationNode(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	var req models.CreateReplicationNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	node, err := h.replicationConfigService.CreateReplicationNode(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, node)
}

// HandleGetReplicationNode returns a specific replication node
func (h *ReplicationSettingsHandler) HandleGetReplicationNode(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	node, err := h.replicationConfigService.GetReplicationNodeByID(tenantID, nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, node)
}

// HandleUpdateReplicationNode updates a replication node
func (h *ReplicationSettingsHandler) HandleUpdateReplicationNode(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	var req models.UpdateReplicationNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	node, err := h.replicationConfigService.UpdateReplicationNode(tenantID, nodeID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, node)
}

// HandleDeleteReplicationNode deletes a replication node
func (h *ReplicationSettingsHandler) HandleDeleteReplicationNode(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	err = h.replicationConfigService.DeleteReplicationNode(tenantID, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Replication node deleted successfully"})
}

// HandleGetNodeHealth returns the health status of a specific node
func (h *ReplicationSettingsHandler) HandleGetNodeHealth(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	nodeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	node, err := h.replicationConfigService.CheckNodeHealth(tenantID, nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	health := models.ReplicationNodeHealthResponse{
		ID:                 node.ID,
		Name:               node.NodeName,
		IsHealthy:          node.IsHealthy,
		HealthStatus:       node.HealthStatus,
		LastCheck:          node.LastHealthCheck,
		StorageAvailableGB: node.StorageAvailableGB,
		StorageTotalGB:     node.StorageTotalGB,
		ErrorCount:         node.ErrorCount,
		ResponseTimeMs:     node.ResponseTimeMs,
	}

	c.JSON(http.StatusOK, health)
}

// HandleGetConfigHistory returns the audit trail of configuration changes
func (h *ReplicationSettingsHandler) HandleGetConfigHistory(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	// Parse pagination parameters
	limit := 50
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if limitVal, err := strconv.Atoi(limitStr); err == nil && limitVal > 0 && limitVal <= 100 {
			limit = limitVal
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offsetVal, err := strconv.Atoi(offsetStr); err == nil && offsetVal >= 0 {
			offset = offsetVal
		}
	}

	history, err := h.replicationConfigService.GetConfigHistory(tenantID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"limit":   limit,
		"offset":  offset,
	})
}

// HandleGetRepositoryReplicationSettings returns replication settings for a specific repository
func (h *ReplicationSettingsHandler) HandleGetRepositoryReplicationSettings(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	settings, err := h.replicationConfigService.GetRepositoryReplicationSettings(tenantID, repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// HandleUpdateRepositoryReplicationSettings updates replication settings for a specific repository
func (h *ReplicationSettingsHandler) HandleUpdateRepositoryReplicationSettings(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	var req models.UpdateRepositoryReplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	err = h.replicationConfigService.UpdateRepositoryReplicationSettings(tenantID, repoID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return updated settings
	settings, err := h.replicationConfigService.GetRepositoryReplicationSettings(tenantID, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// HandleGetEffectiveReplicationConfig returns the effective (global or override) replication config for a repo
func (h *ReplicationSettingsHandler) HandleGetEffectiveReplicationConfig(c *gin.Context) {
	tenantID := h.getTenantIDFromContext(c)

	repoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	effectiveConfig, err := h.replicationConfigService.GetEffectiveReplicationConfig(tenantID, repoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, effectiveConfig)
}

// Helper function to get tenant ID from context (Gin version)
func (h *ReplicationSettingsHandler) getTenantIDFromContext(c *gin.Context) uuid.UUID {
	// First try to get tenant ID from context (set by middleware)
	if tenantID, exists := c.Get("tenant_id"); exists {
		if id, ok := tenantID.(uuid.UUID); ok {
			return id
		}
	}

	// Try X-Tenant-ID header
	tenantIDStr := c.GetHeader("X-Tenant-ID")
	if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
		return tenantID
	}

	// Try to resolve from X-Tenant-Slug header
	tenantSlug := c.GetHeader("X-Tenant-Slug")
	if tenantSlug != "" && h.tenantService != nil {
		tenant, err := h.tenantService.GetTenantBySlug(c.Request.Context(), tenantSlug)
		if err == nil && tenant != nil {
			return tenant.ID
		}
	}

	// Fallback to default tenant (nil UUID)
	return uuid.Nil
}
