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

// UserManagementHandler handles user management API endpoints using Gin
type UserManagementHandler struct {
	userService     *service.UserManagementService
	passwordService *service.PasswordService
	emailService    *service.EmailService
}

// NewUserManagementHandler creates a new Gin-based user management handler
func NewUserManagementHandler(
	userService *service.UserManagementService,
	passwordService *service.PasswordService,
	emailService *service.EmailService,
) *UserManagementHandler {
	return &UserManagementHandler{
		userService:     userService,
		passwordService: passwordService,
		emailService:    emailService,
	}
}

// RegisterRoutes registers user management routes
func (h *UserManagementHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Core user CRUD operations
	rg.GET("/users", h.GetUsers)
	rg.POST("/users", h.CreateUser)
	rg.GET("/users/:id", h.GetUser)
	rg.PUT("/users/:id", h.UpdateUser)
	rg.PATCH("/users/:id", h.UpdateUser)
	rg.DELETE("/users/:id", h.DeleteUser)

	// Simple roles endpoint for dropdowns
	rg.GET("/users/roles", h.GetSimpleRoles)

	// User role management
	rg.GET("/users/:id/roles", h.GetUserRoles)
	rg.POST("/users/:id/roles", h.AssignUserRole)
	rg.DELETE("/users/:id/roles/:roleId", h.RemoveUserRole)

	// User invitations
	rg.POST("/users/invite", h.InviteUser)
	rg.GET("/users/invites", h.GetUserInvites)
	rg.POST("/users/invites/:id/resend", h.ResendInvite)
}

// GetUsers handles GET /users - List users with filtering
func (h *UserManagementHandler) GetUsers(c *gin.Context) {
	// Parse query parameters for filtering
	filter := &service.UserFilter{
		Search:    c.Query("search"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	// Parse pagination
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 50 // Default limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	// Parse boolean filters
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	if roleIDStr := c.Query("role_id"); roleIDStr != "" {
		if roleID, err := strconv.ParseInt(roleIDStr, 10, 64); err == nil {
			filter.RoleID = &roleID
		}
	}

	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		if tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64); err == nil {
			filter.TenantID = &tenantID
		}
	}

	// Get users
	users, totalCount, err := h.userService.GetUsers(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get users: %v", err)})
		return
	}

	// Prepare response
	c.JSON(http.StatusOK, gin.H{
		"users":       users,
		"total_count": totalCount,
		"limit":       filter.Limit,
		"offset":      filter.Offset,
	})
}

// GetUser handles GET /users/:id - Get user by ID
func (h *UserManagementHandler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get user: %v", err)})
		return
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser handles POST /users - Create new user
func (h *UserManagementHandler) CreateUser(c *gin.Context) {
	// Get tenant ID from context
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

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Basic validation
	if user.Username == "" || user.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and email are required"})
		return
	}

	// Set tenant ID and defaults
	user.TenantID = tenantID
	user.IsActive = true
	user.IsEmailVerified = false

	createdUser, err := h.userService.CreateUser(c.Request.Context(), &user)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create user: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, createdUser)
}

// UpdateUser handles PUT/PATCH /users/:id - Update user
func (h *UserManagementHandler) UpdateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
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

	updatedUser, err := h.userService.UpdateUser(c.Request.Context(), userID, updates)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update user: %v", err)})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser handles DELETE /users/:id - Delete user
func (h *UserManagementHandler) DeleteUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), userID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete user: %v", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetSimpleRoles handles GET /users/roles - Get roles list for dropdowns
func (h *UserManagementHandler) GetSimpleRoles(c *gin.Context) {
	// Get roles from service
	roles, err := h.userService.GetRoles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get roles: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// GetUserRoles handles GET /users/:id/roles - Get user's assigned roles
func (h *UserManagementHandler) GetUserRoles(c *gin.Context) {
	_, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Note: getUserRoles is unexported, so we need to use an alternative approach
	// For now, return a placeholder response
	c.JSON(http.StatusOK, gin.H{"roles": []interface{}{}, "message": "getUserRoles is unexported - needs service refactoring"})
}

// AssignUserRole handles POST /users/:id/roles - Assign role to user
func (h *UserManagementHandler) AssignUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	type AssignRoleRequest struct {
		RoleID   uuid.UUID `json:"role_id" binding:"required"`
		TenantID uuid.UUID `json:"tenant_id" binding:"required"`
	}

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.userService.AssignUserRole(c.Request.Context(), userID, req.RoleID, req.TenantID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User or role not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to assign role: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role assigned successfully"})
}

// RemoveUserRole handles DELETE /users/:id/roles/:roleId - Remove role from user
func (h *UserManagementHandler) RemoveUserRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	roleID, err := uuid.Parse(c.Param("roleId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role ID"})
		return
	}

	if err := h.userService.RemoveUserRole(c.Request.Context(), userID, roleID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User or role assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to remove role: %v", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

// InviteUser handles POST /users/invite - Send user invitation
func (h *UserManagementHandler) InviteUser(c *gin.Context) {
	type InviteRequest struct {
		Email     string   `json:"email" binding:"required,email"`
		FirstName *string  `json:"first_name,omitempty"`
		LastName  *string  `json:"last_name,omitempty"`
		RoleIDs   []string `json:"role_ids"`
		InvitedBy string   `json:"invited_by"` // UUID string
		TenantID  string   `json:"tenant_id"`  // UUID string
	}

	var req InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Convert role IDs from strings to UUIDs
	var roleUUIDs []uuid.UUID
	for _, roleIDStr := range req.RoleIDs {
		roleID, err := uuid.Parse(roleIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid role ID: %s", roleIDStr)})
			return
		}
		roleUUIDs = append(roleUUIDs, roleID)
	}

	// Parse invitedBy and tenantID
	invitedByUUID, err := uuid.Parse(req.InvitedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invited_by UUID"})
		return
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant_id UUID"})
		return
	}

	// Create user invite
	invite, err := h.userService.InviteUser(c.Request.Context(), req.Email, req.FirstName, req.LastName, roleUUIDs, invitedByUUID, tenantUUID)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists or invite pending"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to invite user: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User invited successfully",
		"invite":  invite,
	})
}

// GetUserInvites handles GET /users/invites - List pending invitations
func (h *UserManagementHandler) GetUserInvites(c *gin.Context) {
	// Parse query parameters
	includeExpired := false
	if expiredStr := c.Query("include_expired"); expiredStr != "" {
		if expired, err := strconv.ParseBool(expiredStr); err == nil {
			includeExpired = expired
		}
	}

	invites, err := h.userService.GetUserInvites(c.Request.Context(), includeExpired)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get invites: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invites":     invites,
		"total_count": len(invites),
	})
}

// ResendInvite handles POST /users/invites/:id/resend - Resend invitation email
func (h *UserManagementHandler) ResendInvite(c *gin.Context) {
	// Note: ResendInvite method not exported in service - needs implementation
	c.JSON(http.StatusNotImplemented, gin.H{"error": "ResendInvite not yet implemented in service"})
}

// Removed duplicate getUserIDFromContext - it already exists in mfa_handler.go
