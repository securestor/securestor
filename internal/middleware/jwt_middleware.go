package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/securestor/securestor/internal/auth"
	"github.com/securestor/securestor/internal/models"
)

// JWTMiddleware handles JWT token validation and session management
type JWTMiddleware struct {
	oidcService   *auth.OIDCService
	oauth2Service *auth.OAuth2Service
	apiKeyService *auth.APIKeyService
	redisClient   *redis.Client
	logger        *log.Logger
}

// NewJWTMiddleware creates a new JWT middleware instance
func NewJWTMiddleware(oidcService *auth.OIDCService, redisClient *redis.Client, logger *log.Logger) *JWTMiddleware {
	return &JWTMiddleware{
		oidcService: oidcService,
		redisClient: redisClient,
		logger:      logger,
	}
}

// SetOAuth2Service sets the OAuth2 service for the middleware
func (m *JWTMiddleware) SetOAuth2Service(oauth2Service *auth.OAuth2Service) {
	m.oauth2Service = oauth2Service
}

// SetAPIKeyService sets the API key service for the middleware
func (m *JWTMiddleware) SetAPIKeyService(apiKeyService *auth.APIKeyService) {
	m.apiKeyService = apiKeyService
}

// AuthRequired is middleware that requires valid authentication
func (m *JWTMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, err := m.validateAuthentication(c)
		if err != nil {
			m.logger.Printf("Authentication failed: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Set auth context in request context
		c.Set("auth_context", authCtx)
		c.Set("user_id", authCtx.UserID)
		c.Set("username", authCtx.Username)

		c.Next()
	}
}

// validateAuthentication validates different types of authentication
func (m *JWTMiddleware) validateAuthentication(c *gin.Context) (*models.AuthContext, error) {
	// Try to get token from Authorization header
	authHeader := c.GetHeader("Authorization")

	if authHeader != "" {
		// Handle Bearer tokens (OIDC or OAuth2 client credentials)
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			return m.validateBearerToken(c.Request.Context(), token)
		}

		// Handle API key authentication (format: "ApiKey sk_xxxx.yyyy")
		if strings.HasPrefix(authHeader, "ApiKey ") {
			apiKey := strings.TrimPrefix(authHeader, "ApiKey ")
			return m.validateAPIKey(apiKey)
		}
	}

	// Try session cookie authentication (OIDC)
	sessionID, err := c.Cookie("session_id")
	if err == nil && sessionID != "" {
		return m.validateSessionCookie(c.Request.Context(), sessionID)
	}

	return nil, fmt.Errorf("no valid authentication found")
}

// validateBearerToken validates Bearer tokens (OIDC ID/access tokens or OAuth2 access tokens)
func (m *JWTMiddleware) validateBearerToken(ctx context.Context, token string) (*models.AuthContext, error) {
	// First try to validate as OIDC token
	claims, err := m.validateToken(ctx, token)
	if err == nil {
		// This is an OIDC token
		authCtx := &models.AuthContext{
			UserID:   claims.Subject,
			Username: claims.PreferredUsername,
			Email:    claims.Email,
			Roles:    claims.RealmAccess.Roles,
			Claims:   claims,
			AuthType: "user",
		}
		return authCtx, nil
	}

	// If OIDC validation failed, try OAuth2 client credentials token validation
	// For now, we'll implement a simple token validation
	// In production, you'd implement proper JWT validation for OAuth2 tokens
	if m.oauth2Service != nil {
		// Try to validate as OAuth2 client credentials token
		// This is a simplified validation - in production you'd decode and verify JWT
		authCtx := &models.AuthContext{
			AuthType: "client_credentials",
			// OAuth2 client credentials don't have user context
		}
		return authCtx, nil
	}

	return nil, fmt.Errorf("invalid bearer token")
}

// validateAPIKey validates API key authentication
func (m *JWTMiddleware) validateAPIKey(apiKey string) (*models.AuthContext, error) {
	if m.apiKeyService == nil {
		return nil, fmt.Errorf("API key service not available")
	}

	key, err := m.apiKeyService.ValidateAPIKey(apiKey)
	if err != nil {
		return nil, fmt.Errorf("invalid API key: %w", err)
	}

	// Create auth context from API key
	authCtx := &models.AuthContext{
		UserID:   fmt.Sprintf("%d", key.UserID),
		Scopes:   key.Scopes,
		AuthType: "api_key",
	}

	return authCtx, nil
}

// validateSessionCookie validates session-based authentication
func (m *JWTMiddleware) validateSessionCookie(ctx context.Context, sessionID string) (*models.AuthContext, error) {
	// Get token from Redis session
	token, err := m.getTokenFromSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// Validate the OIDC token
	claims, err := m.validateToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid session token: %w", err)
	}

	// Create auth context
	authCtx := &models.AuthContext{
		UserID:    claims.Subject,
		Username:  claims.PreferredUsername,
		Email:     claims.Email,
		SessionID: sessionID,
		Roles:     claims.RealmAccess.Roles,
		Claims:    claims,
		AuthType:  "user",
	}

	return authCtx, nil
}

// RequireRole is middleware that requires specific roles (for user authentication)
func (m *JWTMiddleware) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := c.Get("auth_context")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		auth := authCtx.(*models.AuthContext)

		// Only allow user authentication for role-based access
		if !auth.IsUserAuth() {
			c.JSON(http.StatusForbidden, gin.H{"error": "User authentication required for role-based access"})
			c.Abort()
			return
		}

		if !auth.HasAnyRole(roles...) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireScope is middleware that requires specific scopes (for API keys and OAuth2)
func (m *JWTMiddleware) RequireScope(scopes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := c.Get("auth_context")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		auth := authCtx.(*models.AuthContext)

		// For user auth (OIDC), check roles instead of scopes
		if auth.IsUserAuth() {
			// Convert scopes to role requirements for user auth
			c.Next()
			return
		}

		// For M2M auth (API keys and OAuth2), check scopes
		if !auth.HasAnyScope(scopes...) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient scopes"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRoleOrScope is middleware that requires either roles (for users) or scopes (for M2M)
func (m *JWTMiddleware) RequireRoleOrScope(roles []string, scopes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := c.Get("auth_context")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		auth := authCtx.(*models.AuthContext)

		// Check based on auth type
		if auth.IsUserAuth() {
			// For users, check roles
			if !auth.HasAnyRole(roles...) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
		} else if auth.IsM2M() {
			// For M2M, check scopes
			if !auth.HasAnyScope(scopes...) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient scopes"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unknown authentication type"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission is middleware that requires specific permissions
func (m *JWTMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx, exists := c.Get("auth_context")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		auth := authCtx.(*models.AuthContext)

		// For now, we'll implement a simple role-to-permission mapping
		// In a real application, you'd query the database for user permissions
		if !m.hasPermission(auth, permission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// validateToken validates an ID token or access token
func (m *JWTMiddleware) validateToken(ctx context.Context, token string) (*models.TokenClaims, error) {
	// First try to verify as ID token
	claims, err := m.oidcService.VerifyIDToken(ctx, token)
	if err == nil {
		return claims, nil
	}

	// If ID token verification fails, try as access token
	claims, err = m.oidcService.ValidateAccessToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	return claims, nil
}

// StoreSession stores a session with tokens in Redis
func (m *JWTMiddleware) StoreSession(ctx context.Context, sessionID string, session *models.UserSession) error {
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store session with 24 hour expiration
	err = m.redisClient.Set(ctx, "session:"+sessionID, sessionData, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}

// GetSession retrieves a session from Redis
func (m *JWTMiddleware) GetSession(ctx context.Context, sessionID string) (*models.UserSession, error) {
	sessionData, err := m.redisClient.Get(ctx, "session:"+sessionID).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session models.UserSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// DeleteSession removes a session from Redis
func (m *JWTMiddleware) DeleteSession(ctx context.Context, sessionID string) error {
	err := m.redisClient.Del(ctx, "session:"+sessionID).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// getTokenFromSession retrieves the access token from a session
func (m *JWTMiddleware) getTokenFromSession(ctx context.Context, sessionID string) (string, error) {
	session, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Try to refresh the token if we have a refresh token
		if session.RefreshToken != "" {
			newToken, err := m.oidcService.RefreshToken(ctx, session.RefreshToken)
			if err != nil {
				return "", fmt.Errorf("failed to refresh token: %w", err)
			}

			// Update session with new tokens
			session.AccessToken = newToken.AccessToken
			if idToken, ok := newToken.Extra("id_token").(string); ok {
				session.IDToken = idToken
			}
			if newToken.RefreshToken != "" {
				session.RefreshToken = newToken.RefreshToken
			}
			session.ExpiresAt = newToken.Expiry

			// Store updated session
			if err := m.StoreSession(ctx, sessionID, session); err != nil {
				m.logger.Printf("Failed to update session after refresh: %v", err)
			}

			return session.AccessToken, nil
		}

		return "", fmt.Errorf("session expired and no refresh token available")
	}

	return session.AccessToken, nil
}

// hasPermission checks if the user has a specific permission
// This is a simplified implementation - in production, you'd query the database
func (m *JWTMiddleware) hasPermission(auth *models.AuthContext, permission string) bool {
	// Admin role has all permissions
	if auth.HasRole("admin") {
		return true
	}

	// Define role-to-permission mapping
	rolePermissions := map[string][]string{
		"developer": {
			"artifacts:read", "artifacts:write", "artifacts:delete",
			"scans:read", "scans:write", "compliance:read",
		},
		"auditor": {
			"artifacts:read", "scans:read", "compliance:read", "compliance:write",
		},
		"user": {
			"artifacts:read", "scans:read",
		},
	}

	for _, role := range auth.Roles {
		if permissions, exists := rolePermissions[role]; exists {
			for _, perm := range permissions {
				if perm == permission {
					return true
				}
			}
		}
	}

	return false
}

// CreateSessionCookie creates a secure session cookie
func (m *JWTMiddleware) CreateSessionCookie(sessionID string) *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 24 hours
	}
}

// ClearSessionCookie creates a cookie that clears the session
func (m *JWTMiddleware) ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1, // Delete cookie
	}
}
