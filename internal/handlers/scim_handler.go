package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/auth"
	"github.com/securestor/securestor/internal/models"
)

// SCIMHandler handles SCIM group mapping endpoints
type SCIMHandler struct {
	scimService *auth.SCIMService
}

// NewSCIMHandler creates a new SCIM handler
func NewSCIMHandler(scimService *auth.SCIMService) *SCIMHandler {
	return &SCIMHandler{
		scimService: scimService,
	}
}

// GetGroupMappingConfig returns the current group mapping configuration
func (h *SCIMHandler) GetGroupMappingConfig(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)

	// Require admin role
	if !auth.HasRole("admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	err := h.scimService.LoadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load SCIM settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"group_role_mapping":       h.scimService.GetSettings().GroupRoleMapping,
		"enable_user_provisioning": h.scimService.GetSettings().EnableUserProvisioning,
		"enable_group_mapping":     h.scimService.GetSettings().EnableGroupMapping,
		"group_claim_name":         h.scimService.GetSettings().GroupClaimName,
		"group_format":             h.scimService.GetSettings().GroupFormat,
	})
}

// UpdateGroupMappingConfig updates the group mapping configuration
func (h *SCIMHandler) UpdateGroupMappingConfig(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)

	// Require admin role
	if !auth.HasRole("admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	var req struct {
		GroupRoleMapping       map[string][]string `json:"group_role_mapping"`
		EnableUserProvisioning *bool               `json:"enable_user_provisioning,omitempty"`
		EnableGroupMapping     *bool               `json:"enable_group_mapping,omitempty"`
		GroupClaimName         string              `json:"group_claim_name,omitempty"`
		GroupFormat            string              `json:"group_format,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate group mapping
	if req.GroupRoleMapping != nil {
		err := h.scimService.ValidateGroupMapping(req.GroupRoleMapping)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Load current settings
	err := h.scimService.LoadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load current settings"})
		return
	}

	settings := h.scimService.GetSettings()

	// Update settings
	if req.GroupRoleMapping != nil {
		settings.GroupRoleMapping = req.GroupRoleMapping
	}
	if req.EnableUserProvisioning != nil {
		settings.EnableUserProvisioning = *req.EnableUserProvisioning
	}
	if req.EnableGroupMapping != nil {
		settings.EnableGroupMapping = *req.EnableGroupMapping
	}
	if req.GroupClaimName != "" {
		settings.GroupClaimName = req.GroupClaimName
	}
	if req.GroupFormat != "" {
		settings.GroupFormat = req.GroupFormat
	}

	// Save updated settings
	err = h.scimService.SaveSettings(settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save SCIM settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SCIM settings updated successfully"})
}

// GetUserGroups returns the group mappings for the current user
func (h *SCIMHandler) GetUserGroups(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)

	// Only allow user authentication for this endpoint
	if !auth.IsUserAuth() {
		c.JSON(http.StatusForbidden, gin.H{"error": "User authentication required"})
		return
	}

	userID, err := strconv.ParseInt(auth.UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	groups, err := h.scimService.GetUserGroups(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user groups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"groups":  groups,
		"roles":   auth.Roles,
	})
}

// GetGroupStatistics returns SCIM group mapping statistics
func (h *SCIMHandler) GetGroupStatistics(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)

	// Require admin role
	if !auth.HasRole("admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	stats, err := h.scimService.GetGroupStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get group statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"statistics": stats})
}

// TestGroupMapping tests group mapping for provided claims
func (h *SCIMHandler) TestGroupMapping(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)

	// Require admin role
	if !auth.HasRole("admin") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	var req struct {
		Claims map[string]interface{} `json:"claims"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Process claims without saving to database
	userGroupInfo, err := h.scimService.ProcessUserClaims(req.Claims, uuid.UUID{}) // Use empty UUID as dummy user ID
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process claims"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"extracted_groups": userGroupInfo.Groups,
		"mapped_roles":     userGroupInfo.Roles,
	})
}
