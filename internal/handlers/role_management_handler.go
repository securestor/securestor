package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// RoleManagementHandler handles role and permission management using Gin
type RoleManagementHandler struct {
	roleService *service.RoleManagementService
}

// NewRoleManagementHandler creates a new role management handler for Gin
func NewRoleManagementHandler(roleService *service.RoleManagementService) *RoleManagementHandler {
	return &RoleManagementHandler{
		roleService: roleService,
	}
}

// RegisterRoutes registers role management routes (all require admin privileges)
func (h *RoleManagementHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Role CRUD operations
	rg.GET("/roles", h.GetRoles)
	rg.POST("/roles", h.CreateRole)
	rg.GET("/roles/:id", h.GetRole)
	rg.PUT("/roles/:id", h.UpdateRole)
	rg.PATCH("/roles/:id", h.UpdateRole)
	rg.DELETE("/roles/:id", h.DeleteRole)

	// Role permission management
	rg.GET("/roles/:id/permissions", h.GetRolePermissions)
	rg.PUT("/roles/:id/permissions", h.AssignRolePermissions)
	rg.POST("/roles/:id/permissions/:permissionId", h.AddRolePermission)
	rg.DELETE("/roles/:id/permissions/:permissionId", h.RemoveRolePermission)

	// Role user management
	rg.GET("/roles/:id/users", h.GetRoleUsers)

	// Permission management
	rg.GET("/permissions", h.GetPermissions)
	rg.POST("/permissions", h.CreatePermission)
	rg.GET("/permissions/resources", h.GetResourcesAndActions)
}

// GetRoles handles GET /roles - List roles with filtering
func (h *RoleManagementHandler) GetRoles(c *gin.Context) {
	// Parse query parameters for filtering
	filter := &service.RoleFilter{
		Search:        c.Query("search"),
		IncludeSystem: c.Query("include_system") == "true",
		SortBy:        c.Query("sort_by"),
		SortOrder:     c.Query("sort_order"),
	}

	// Parse pagination
	filter.Limit = 50 // Default
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	filter.Offset = 0 // Default
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	// Parse boolean filters
	if isSystemRoleStr := c.Query("is_system_role"); isSystemRoleStr != "" {
		if isSystemRole, err := strconv.ParseBool(isSystemRoleStr); err == nil {
			filter.IsSystemRole = &isSystemRole
		}
	}

	// Get roles
	roles, totalCount, err := h.roleService.GetRoles(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get roles: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles":       roles,
		"total_count": totalCount,
		"limit":       filter.Limit,
		"offset":      filter.Offset,
	})
}

// GetRole handles GET /roles/:id - Get specific role
func (h *RoleManagementHandler) GetRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	role, err := h.roleService.GetRoleByID(c.Request.Context(), roleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, role)
}

// CreateRole handles POST /roles - Create new role
func (h *RoleManagementHandler) CreateRole(c *gin.Context) {
	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Basic validation
	if role.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role name is required"})
		return
	}

	// Get tenant ID from context
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context not found"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}
	role.TenantID = tenantID

	// Ensure it's not a system role (only system can create system roles)
	role.IsSystemRole = false

	createdRole, err := h.roleService.CreateRole(c.Request.Context(), &role)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Role name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create role: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, createdRole)
}

// UpdateRole handles PUT/PATCH /roles/:id - Update role
func (h *RoleManagementHandler) UpdateRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Remove fields that shouldn't be updated directly
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "is_system_role") // Prevent changing system role flag

	updatedRole, err := h.roleService.UpdateRole(c.Request.Context(), roleID, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		if strings.Contains(err.Error(), "system role") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update system role"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, updatedRole)
}

// DeleteRole handles DELETE /roles/:id - Delete role
func (h *RoleManagementHandler) DeleteRole(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	err = h.roleService.DeleteRole(c.Request.Context(), roleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		if strings.Contains(err.Error(), "system role") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete system role"})
			return
		}
		if strings.Contains(err.Error(), "assigned users") {
			c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete role with assigned users"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete role: %v", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRolePermissions handles GET /roles/:id/permissions - Get role's permissions
func (h *RoleManagementHandler) GetRolePermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	role, err := h.roleService.GetRoleByID(c.Request.Context(), roleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role_id":     roleID,
		"role_name":   role.Name,
		"permissions": role.Permissions,
	})
}

// AssignRolePermissions handles PUT /roles/:id/permissions - Replace all permissions
func (h *RoleManagementHandler) AssignRolePermissions(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	type AssignPermissionsRequest struct {
		PermissionIDs []uuid.UUID `json:"permission_ids" binding:"required"`
	}

	var req AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err = h.roleService.AssignRolePermissions(c.Request.Context(), roleID, req.PermissionIDs)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		if strings.Contains(err.Error(), "system role") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system role permissions"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to assign permissions: %v", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

// AddRolePermission handles POST /roles/:id/permissions/:permissionId - Add single permission
func (h *RoleManagementHandler) AddRolePermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	permissionID, err := uuid.Parse(c.Param("permissionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	err = h.roleService.AddRolePermission(c.Request.Context(), roleID, permissionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role or permission not found"})
			return
		}
		if strings.Contains(err.Error(), "system role") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system role permissions"})
			return
		}
		if strings.Contains(err.Error(), "already has") {
			c.JSON(http.StatusConflict, gin.H{"error": "Role already has this permission"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to add permission: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission added successfully"})
}

// RemoveRolePermission handles DELETE /roles/:id/permissions/:permissionId - Remove permission
func (h *RoleManagementHandler) RemoveRolePermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	permissionID, err := uuid.Parse(c.Param("permissionId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission ID"})
		return
	}

	err = h.roleService.RemoveRolePermission(c.Request.Context(), roleID, permissionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role or permission not found"})
			return
		}
		if strings.Contains(err.Error(), "system role") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify system role permissions"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to remove permission: %v", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRoleUsers handles GET /roles/:id/users - Get users assigned to role
func (h *RoleManagementHandler) GetRoleUsers(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	users, err := h.roleService.GetRoleUsers(c.Request.Context(), roleID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get role users: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role_id": roleID,
		"users":   users,
	})
}

// GetPermissions handles GET /permissions - List permissions with filtering
func (h *RoleManagementHandler) GetPermissions(c *gin.Context) {
	// Parse query parameters for filtering
	filter := &service.PermissionFilter{
		Search:    c.Query("search"),
		Resource:  c.Query("resource"),
		Action:    c.Query("action"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	// Parse pagination
	filter.Limit = 50 // Default
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}

	filter.Offset = 0 // Default
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	// Get permissions
	permissions, totalCount, err := h.roleService.GetPermissions(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get permissions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"permissions": permissions,
		"total_count": totalCount,
		"limit":       filter.Limit,
		"offset":      filter.Offset,
	})
}

// CreatePermission handles POST /permissions - Create new permission
func (h *RoleManagementHandler) CreatePermission(c *gin.Context) {
	var permission models.Permission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Basic validation
	if permission.Name == "" || permission.Resource == "" || permission.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name, resource, and action are required"})
		return
	}

	createdPermission, err := h.roleService.CreatePermission(c.Request.Context(), &permission)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Permission already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create permission: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, createdPermission)
}

// GetResourcesAndActions handles GET /permissions/resources - Get available resources and actions
func (h *RoleManagementHandler) GetResourcesAndActions(c *gin.Context) {
	resourceActions, err := h.roleService.GetResourcesAndActions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get resources and actions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resources": resourceActions,
	})
}
