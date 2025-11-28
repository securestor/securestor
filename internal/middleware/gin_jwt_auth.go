package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GinJWTAuth is a Gin middleware adapter for JWT authentication
type GinJWTAuth struct {
	jwtAuth *JWTAuth
}

// NewGinJWTAuth creates a new Gin JWT authentication middleware
func NewGinJWTAuth(jwtAuth *JWTAuth) *GinJWTAuth {
	return &GinJWTAuth{
		jwtAuth: jwtAuth,
	}
}

// RequireAuth is Gin middleware that requires valid JWT authentication
func (m *GinJWTAuth) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check if header starts with "Bearer "
		tokenString := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Parse and validate token
		claims, err := m.jwtAuth.validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			c.Abort()
			return
		}

		// Parse UserID from string to UUID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		// Parse TenantID from string to UUID
		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
			c.Abort()
			return
		}

		// Set auth context in Gin context
		c.Set("user_id", userID)
		c.Set("tenant_id", tenantID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}

// RequireAdmin is Gin middleware that requires admin privileges
func (m *GinJWTAuth) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check authentication
		m.RequireAuth()(c)
		if c.IsAborted() {
			return
		}

		// Then check if user is admin
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth is Gin middleware that optionally validates authentication
// Continues even if authentication fails, but sets context if token is valid
func (m *GinJWTAuth) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := authHeader[7:]
			claims, err := m.jwtAuth.validateToken(tokenString)
			if err == nil {
				userID, _ := uuid.Parse(claims.UserID)
				tenantID, _ := uuid.Parse(claims.TenantID)

				c.Set("user_id", userID)
				c.Set("tenant_id", tenantID)
				c.Set("username", claims.Username)
				c.Set("email", claims.Email)
				c.Set("is_admin", claims.IsAdmin)
			}
		}

		c.Next()
	}
}
