package handlers

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/middleware"
	"github.com/securestor/securestor/internal/service"
)

// AuthBasicHandler handles basic authentication endpoints using Gin framework
type AuthBasicHandler struct {
	userService        *service.UserManagementService
	passwordService    *service.PasswordService
	jwtAuth            *middleware.JWTAuth
	auditLogService    *service.AuditLogService
	userSessionService *service.UserSessionService
}

// NewAuthBasicHandler creates a new basic auth handler for Gin
func NewAuthBasicHandler(
	userService *service.UserManagementService,
	passwordService *service.PasswordService,
	jwtAuth *middleware.JWTAuth,
	auditLogService *service.AuditLogService,
	userSessionService *service.UserSessionService,
) *AuthBasicHandler {
	return &AuthBasicHandler{
		userService:        userService,
		passwordService:    passwordService,
		jwtAuth:            jwtAuth,
		auditLogService:    auditLogService,
		userSessionService: userSessionService,
	}
}

// BasicLoginRequest represents the basic login request payload
type BasicLoginRequestGin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// BasicLoginResponse represents the basic login response payload
type BasicLoginResponseGin struct {
	Token     string      `json:"token"`
	User      interface{} `json:"user"`
	ExpiresAt int64       `json:"expires_at"`
}

// ChangePasswordRequestGin represents the change password request payload
type ChangePasswordRequestGin struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// Login handles POST /api/gin/auth/login
// @Summary User login
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body BasicLoginRequestGin true "Login credentials"
// @Success 200 {object} BasicLoginResponseGin
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /gin/auth/login [post]
func (h *AuthBasicHandler) Login(c *gin.Context) {
	var req BasicLoginRequestGin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	fmt.Printf("DEBUG: Login attempt for user: %s\n", req.Username)

	// Helper function to log failed login attempts
	logFailedLogin := func(username, reason string, tenantID *string) {
		if h.auditLogService != nil {
			xForwardedFor := c.GetHeader("X-Forwarded-For")
			xRealIP := c.GetHeader("X-Real-IP")
			remoteIP := c.ClientIP()
			if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
				remoteIP = host
			}
			ipAddress := service.ExtractIPFromRequest(remoteIP, xForwardedFor, xRealIP)
			userAgent := c.GetHeader("User-Agent")

			tid := "00000000-0000-0000-0000-000000000000" // Default tenant
			if tenantID != nil {
				tid = *tenantID
			}

			h.auditLogService.LogUserSession(c.Request.Context(), tid, username, "login", ipAddress, userAgent, "", map[string]interface{}{
				"login_successful": false,
				"login_method":     "basic_auth_gin",
				"failure_reason":   reason,
				"username":         username,
			})
		}
	}

	// Get user by username
	user, err := h.userService.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		fmt.Printf("DEBUG: User not found or error: %v\n", err)
		if strings.Contains(err.Error(), "not found") {
			logFailedLogin(req.Username, "user_not_found", nil)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		logFailedLogin(req.Username, "authentication_error", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication failed"})
		return
	}

	fmt.Printf("DEBUG: User found: %s (ID: %s)\n", user.Username, user.ID.String())
	tenantIDStr := user.TenantID.String()

	// Verify password
	if user.PasswordHash == nil {
		fmt.Printf("DEBUG: No password hash for user\n")
		logFailedLogin(req.Username, "no_password_configured", &tenantIDStr)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Password authentication not available for this user"})
		return
	}

	if err := h.passwordService.VerifyPassword(req.Password, *user.PasswordHash); err != nil {
		fmt.Printf("DEBUG: Password verification failed: %v\n", err)
		logFailedLogin(req.Username, "invalid_password", &tenantIDStr)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	fmt.Printf("DEBUG: Password verified successfully\n")

	// Check if user is active
	if !user.IsActive {
		fmt.Printf("DEBUG: User is not active\n")
		logFailedLogin(req.Username, "account_disabled", &tenantIDStr)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	fmt.Printf("DEBUG: User is active, generating token. User TenantID: %s\n", user.TenantID.String())

	// Generate JWT token (with tenantID for multi-tenancy)
	token, err := h.jwtAuth.GenerateToken(user.ID, user.TenantID, user.Username, user.Email, user.IsAdmin())
	if err != nil {
		fmt.Printf("DEBUG: Token generation failed: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Log user session if audit services are available
	if h.auditLogService != nil && h.userSessionService != nil {
		xForwardedFor := c.GetHeader("X-Forwarded-For")
		xRealIP := c.GetHeader("X-Real-IP")

		remoteIP := c.ClientIP()
		if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
			remoteIP = host
		}

		ipAddress := service.ExtractIPFromRequest(remoteIP, xForwardedFor, xRealIP)
		userAgent := c.GetHeader("User-Agent")

		userIDStr := user.ID.String()
		session, err := h.userSessionService.CreateSession(c.Request.Context(), user.TenantID.String(), userIDStr, ipAddress, userAgent)
		if err == nil && session != nil {
			h.auditLogService.LogUserSession(c.Request.Context(), user.TenantID.String(), userIDStr, "login", ipAddress, userAgent, session.ID, map[string]interface{}{
				"login_successful": true,
				"login_method":     "basic_auth_gin",
			})
		}
	}

	// Prepare user data for response (remove sensitive fields)
	userResponse := gin.H{
		"id":                user.ID,
		"username":          user.Username,
		"email":             user.Email,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"is_active":         user.IsActive,
		"is_email_verified": user.IsEmailVerified,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
		"tenant_id":         user.TenantID,
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"user":       userResponse,
		"expires_at": 0, // TODO: calculate actual expiry time
	})
}

// Logout handles POST /api/gin/auth/logout
// @Summary User logout
// @Tags Authentication
// @Produce json
// @Success 200 {object} map[string]string
// @Router /gin/auth/logout [post]
func (h *AuthBasicHandler) Logout(c *gin.Context) {
	// In a stateless JWT implementation, logout is handled client-side
	// The client should remove the token from storage
	// For enhanced security, you could implement token blacklisting here

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// GetCurrentUser handles GET /api/gin/auth/me
// @Summary Get current user
// @Tags Authentication
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /gin/auth/me [get]
func (h *AuthBasicHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Prepare user data for response (remove sensitive fields)
	c.JSON(http.StatusOK, gin.H{
		"id":                user.ID,
		"username":          user.Username,
		"email":             user.Email,
		"first_name":        user.FirstName,
		"last_name":         user.LastName,
		"is_active":         user.IsActive,
		"is_email_verified": user.IsEmailVerified,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
		"tenant_id":         user.TenantID,
	})
}

// ChangePassword handles POST /api/gin/auth/change-password
// @Summary Change password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body ChangePasswordRequestGin true "Password change request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /gin/auth/change-password [post]
func (h *AuthBasicHandler) ChangePassword(c *gin.Context) {
	// Get user ID from context (set by JWT middleware)
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	var req ChangePasswordRequestGin
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Get current user
	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Verify current password
	if user.PasswordHash == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password authentication not available for this user"})
		return
	}

	if err := h.passwordService.VerifyPassword(req.CurrentPassword, *user.PasswordHash); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Validate new password strength
	if err := h.passwordService.ValidatePassword(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("New password validation failed: %v", err)})
		return
	}

	// Hash new password
	hashedPassword, err := h.passwordService.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password"})
		return
	}

	// Update password in database
	updates := map[string]interface{}{
		"password_hash": hashedPassword,
	}

	_, err = h.userService.UpdateUser(c.Request.Context(), userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// RegisterRoutes registers authentication routes with Gin router
func (h *AuthBasicHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Public routes (no authentication required)
	router.POST("/login", h.Login)
	router.POST("/logout", h.Logout)

	// Protected routes will be registered separately with auth middleware
}

// RegisterProtectedRoutes registers authentication routes that require authentication
func (h *AuthBasicHandler) RegisterProtectedRoutes(router *gin.RouterGroup) {
	router.GET("/me", h.GetCurrentUser)
	router.POST("/change-password", h.ChangePassword)
}
