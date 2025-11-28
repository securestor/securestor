package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/securestor/securestor/internal/auth"
	"github.com/securestor/securestor/internal/models"
)

// RefreshTokenHandler handles refresh token management endpoints
type RefreshTokenHandler struct {
	refreshTokenService *auth.RefreshTokenService
}

// NewRefreshTokenHandler creates a new refresh token handler
func NewRefreshTokenHandler(refreshTokenService *auth.RefreshTokenService) *RefreshTokenHandler {
	return &RefreshTokenHandler{
		refreshTokenService: refreshTokenService,
	}
}

// GetUserTokens returns active refresh tokens for the current user
func (h *RefreshTokenHandler) GetUserTokens(c *gin.Context) {
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

	tokens, err := h.refreshTokenService.GetUserTokens(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user tokens"})
		return
	}

	// Remove sensitive token hashes from response
	sanitizedTokens := make([]map[string]interface{}, len(tokens))
	for i, token := range tokens {
		sanitizedTokens[i] = map[string]interface{}{
			"id":         token.ID,
			"session_id": token.SessionID,
			"client_id":  token.ClientID,
			"expires_at": token.ExpiresAt,
			"created_at": token.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"tokens": sanitizedTokens})
}

// RevokeUserTokens revokes all refresh tokens for the current user
func (h *RefreshTokenHandler) RevokeUserTokens(c *gin.Context) {
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

	err = h.refreshTokenService.RevokeAllUserTokens(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke user tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All refresh tokens revoked successfully"})
}

// RevokeSessionTokens revokes refresh tokens for a specific session
func (h *RefreshTokenHandler) RevokeSessionTokens(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID required"})
		return
	}

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

	// TODO: Add authorization check to ensure user can only revoke their own session tokens

	err := h.refreshTokenService.RevokeSessionTokens(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke session tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session refresh tokens revoked successfully"})
}

// GetTokenStats returns refresh token statistics (admin only)
func (h *RefreshTokenHandler) GetTokenStats(c *gin.Context) {
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

	stats, err := h.refreshTokenService.GetTokenStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get token stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// CleanupExpiredTokens removes expired refresh tokens (admin only)
func (h *RefreshTokenHandler) CleanupExpiredTokens(c *gin.Context) {
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

	deletedCount, err := h.refreshTokenService.CleanupExpiredTokens()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup expired tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Cleanup completed successfully",
		"deleted_count": deletedCount,
	})
}
