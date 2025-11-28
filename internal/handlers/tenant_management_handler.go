package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/service"
)

// TenantManagementHandler handles tenant management endpoints using Gin
type TenantManagementHandler struct {
	tenantService *service.TenantManagementService
}

// NewTenantManagementHandler creates a new Gin-based tenant management handler
func NewTenantManagementHandler(tenantService *service.TenantManagementService) *TenantManagementHandler {
	return &TenantManagementHandler{
		tenantService: tenantService,
	}
}

// RegisterRoutes registers all tenant management routes (protected, requires JWT)
// Note: Public routes (validate, resolve) should be registered separately without JWT middleware
func (h *TenantManagementHandler) RegisterRoutes(rg *gin.RouterGroup) {
	tenants := rg.Group("/tenants")
	{
		// General tenant routes (all require JWT)
		tenants.GET("", h.GetTenants)
		tenants.POST("", h.CreateTenant)

		// Parameterized routes
		tenants.GET("/:tenantId", h.GetTenant)
		tenants.PUT("/:tenantId", h.UpdateTenant)
		tenants.GET("/:tenantId/settings", h.GetTenantSettings)
		tenants.PUT("/:tenantId/settings", h.UpdateTenantSettings)
		tenants.PATCH("/:tenantId/settings", h.PartialUpdateTenantSettings)
		tenants.GET("/:tenantId/usage", h.GetTenantUsage)
	}

	// Admin routes
	admin := rg.Group("/admin/tenants")
	{
		admin.GET("/stats", h.GetTenantStats)
	}
}

// GetTenants handles GET /tenants - List tenants with filtering
func (h *TenantManagementHandler) GetTenants(c *gin.Context) {
	filter := &service.TenantFilter{
		Search:    c.Query("search"),
		Plan:      c.Query("plan"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	// Parse boolean parameters
	if activeStr := c.Query("is_active"); activeStr != "" {
		if active, err := strconv.ParseBool(activeStr); err == nil {
			filter.IsActive = &active
		}
	}

	// Parse pagination
	filter.Limit = 50 // Default
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	filter.Offset = 0 // Default
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	tenants, totalCount, err := h.tenantService.GetTenants(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get tenants: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants":     tenants,
		"total_count": totalCount,
		"limit":       filter.Limit,
		"offset":      filter.Offset,
	})
}

// GetTenant handles GET /tenants/:tenantId - Get specific tenant
func (h *TenantManagementHandler) GetTenant(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	tenant, err := h.tenantService.GetTenantByID(c.Request.Context(), tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get tenant: %v", err)})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// GetTenantBySlug handles GET /tenants/resolve/:slug - Resolve tenant by slug
func (h *TenantManagementHandler) GetTenantBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant slug is required"})
		return
	}

	tenant, err := h.tenantService.GetTenantBySlug(c.Request.Context(), slug)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to resolve tenant: %v", err)})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// ValidateTenant handles GET /tenants/validate/:slug - PUBLIC endpoint for frontend tenant validation
func (h *TenantManagementHandler) ValidateTenant(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusOK, gin.H{
			"exists": false,
			"error":  "Tenant slug is required",
		})
		return
	}

	tenant, err := h.tenantService.GetTenantBySlug(c.Request.Context(), slug)

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusOK, gin.H{
				"exists":  false,
				"message": "Tenant not found",
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"exists": false,
				"error":  fmt.Sprintf("Validation error: %v", err),
			})
		}
		return
	}

	if tenant != nil {
		c.JSON(http.StatusOK, gin.H{
			"exists":      tenant.IsActive,
			"slug":        tenant.Slug,
			"tenant_id":   tenant.ID,
			"tenant_name": tenant.Name,
		})
		if !tenant.IsActive {
			c.JSON(http.StatusOK, gin.H{
				"exists":  false,
				"message": "Tenant exists but is not active",
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"exists":  false,
			"message": "Tenant not found",
		})
	}
}

// CreateTenant handles POST /tenants - Create new tenant
func (h *TenantManagementHandler) CreateTenant(c *gin.Context) {
	var tenant service.Tenant
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if tenant.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant name is required"})
		return
	}

	// Set defaults
	if tenant.Plan == "" {
		tenant.Plan = "basic"
	}
	if tenant.MaxUsers == 0 {
		tenant.MaxUsers = 100
	}
	if tenant.Settings == nil {
		tenant.Settings = make(map[string]interface{})
	}
	if tenant.Features == nil {
		tenant.Features = []string{}
	}
	tenant.IsActive = true

	createdTenant, err := h.tenantService.CreateTenant(c.Request.Context(), &tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create tenant: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, createdTenant)
}

// UpdateTenant handles PUT /tenants/:tenantId - Update tenant
func (h *TenantManagementHandler) UpdateTenant(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Remove read-only fields
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "current_users")
	delete(updates, "usage_percent")

	updatedTenant, err := h.tenantService.UpdateTenant(c.Request.Context(), tenantID, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update tenant: %v", err)})
		return
	}

	c.JSON(http.StatusOK, updatedTenant)
}

// GetTenantSettings handles GET /tenants/:tenantId/settings - Get tenant settings
func (h *TenantManagementHandler) GetTenantSettings(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	settings, err := h.tenantService.GetTenantSettings(c.Request.Context(), tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get tenant settings: %v", err)})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateTenantSettings handles PUT /tenants/:tenantId/settings - Update tenant settings
func (h *TenantManagementHandler) UpdateTenantSettings(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	var settings service.TenantSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	settings.TenantID = tenantID

	err = h.tenantService.UpdateTenantSettings(c.Request.Context(), tenantID, &settings)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update tenant settings: %v", err)})
		return
	}

	// Return updated settings
	updatedSettings, err := h.tenantService.GetTenantSettings(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get updated settings: %v", err)})
		return
	}

	c.JSON(http.StatusOK, updatedSettings)
}

// PartialUpdateTenantSettings handles PATCH /tenants/:tenantId/settings - Partial update
func (h *TenantManagementHandler) PartialUpdateTenantSettings(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	type TenantSettingsRequest struct {
		Security       *service.SecuritySettings     `json:"security,omitempty"`
		UserManagement *service.UserSettings         `json:"user_management,omitempty"`
		Storage        *service.StorageSettings      `json:"storage,omitempty"`
		Notifications  *service.NotificationSettings `json:"notifications,omitempty"`
		Integrations   *service.IntegrationSettings  `json:"integrations,omitempty"`
		Compliance     *service.ComplianceSettings   `json:"compliance,omitempty"`
	}

	var request TenantSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get current settings
	currentSettings, err := h.tenantService.GetTenantSettings(c.Request.Context(), tenantID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get current settings: %v", err)})
		return
	}

	// Apply partial updates
	if request.Security != nil {
		currentSettings.SecuritySettings = *request.Security
	}
	if request.UserManagement != nil {
		currentSettings.UserSettings = *request.UserManagement
	}
	if request.Storage != nil {
		currentSettings.StorageSettings = *request.Storage
	}
	if request.Notifications != nil {
		currentSettings.NotificationSettings = *request.Notifications
	}
	if request.Integrations != nil {
		currentSettings.IntegrationSettings = *request.Integrations
	}
	if request.Compliance != nil {
		currentSettings.ComplianceSettings = *request.Compliance
	}

	// Update settings
	err = h.tenantService.UpdateTenantSettings(c.Request.Context(), tenantID, currentSettings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update tenant settings: %v", err)})
		return
	}

	// Return updated settings
	updatedSettings, err := h.tenantService.GetTenantSettings(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get updated settings: %v", err)})
		return
	}

	c.JSON(http.StatusOK, updatedSettings)
}

// GetTenantUsage handles GET /tenants/:tenantId/usage - Get tenant usage statistics
func (h *TenantManagementHandler) GetTenantUsage(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("tenantId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	usage, err := h.tenantService.GetTenantUsage(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get tenant usage: %v", err)})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetTenantStats handles GET /admin/tenants/stats - Get overall tenant statistics
func (h *TenantManagementHandler) GetTenantStats(c *gin.Context) {
	stats, err := h.tenantService.GetTenantStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get tenant stats: %v", err)})
		return
	}

	c.JSON(http.StatusOK, stats)
}
