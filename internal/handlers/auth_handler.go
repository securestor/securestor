package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/auth"
	"github.com/securestor/securestor/internal/middleware"
	"github.com/securestor/securestor/internal/models"
)

// AuthHandler handles OIDC authentication endpoints
type AuthHandler struct {
	oidcService         *auth.OIDCService
	jwtMiddleware       *middleware.JWTMiddleware
	refreshTokenService *auth.RefreshTokenService
	scimService         *auth.SCIMService
	redisClient         *redis.Client
	db                  *sql.DB
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(oidcService *auth.OIDCService, jwtMiddleware *middleware.JWTMiddleware, scimService *auth.SCIMService, redisClient *redis.Client, db *sql.DB) *AuthHandler {
	return &AuthHandler{
		oidcService:         oidcService,
		jwtMiddleware:       jwtMiddleware,
		refreshTokenService: auth.NewRefreshTokenService(db, nil), // Will set logger later
		scimService:         scimService,
		redisClient:         redisClient,
		db:                  db,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	RedirectURI string `json:"redirect_uri,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

// CallbackRequest represents the callback from OIDC provider
type CallbackRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

// CallbackResponse represents the callback response
type CallbackResponse struct {
	Success     bool         `json:"success"`
	RedirectURL string       `json:"redirect_url,omitempty"`
	User        *models.User `json:"user,omitempty"`
}

// Login initiates the OIDC login flow
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no JSON body, this is fine - we'll use defaults
	}

	// Generate state for CSRF protection
	state, err := h.oidcService.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	// Store state in Redis with 10 minute expiration
	err = h.redisClient.Set(c.Request.Context(), "oauth_state:"+state, req.RedirectURI, 10*time.Minute).Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store state"})
		return
	}

	// Generate authorization URL
	authURL := h.oidcService.GetAuthCodeURL(state)

	c.JSON(http.StatusOK, LoginResponse{
		AuthURL: authURL,
		State:   state,
	})
}

// Callback handles the OIDC callback
func (h *AuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing code or state parameter"})
		return
	}

	// Verify state to prevent CSRF attacks
	storedRedirectURI, err := h.redisClient.Get(c.Request.Context(), "oauth_state:"+state).Result()
	if err != nil {
		if err == redis.Nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired state"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify state"})
		return
	}

	// Delete the used state
	h.redisClient.Del(c.Request.Context(), "oauth_state:"+state)

	// Exchange authorization code for tokens
	token, err := h.oidcService.ExchangeToken(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Verify ID token and extract claims
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No ID token received"})
		return
	}

	claims, err := h.oidcService.VerifyIDToken(c.Request.Context(), rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ID token"})
		return
	}

	// Find or create user in database
	user, err := h.findOrCreateUser(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate session ID
	sessionID, err := h.generateSessionID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session"})
		return
	}

	// Create user session
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()
	session := &models.UserSession{
		ID:             sessionID,
		UserID:         user.ID,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		IDToken:        rawIDToken,
		ExpiresAt:      token.Expiry,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		IPAddress:      &clientIP,
		UserAgent:      &userAgent,
	}

	// Store session in Redis
	err = h.jwtMiddleware.StoreSession(c.Request.Context(), sessionID, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store session"})
		return
	}

	// Store session in database
	err = h.storeSessionInDB(session)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to store session in database: %v\n", err)
	}

	// Set session cookie
	cookie := h.jwtMiddleware.CreateSessionCookie(sessionID)
	http.SetCookie(c.Writer, cookie)

	// Determine redirect URL
	redirectURL := "/dashboard" // Default redirect
	if storedRedirectURI != "" {
		redirectURL = storedRedirectURI
	}

	c.JSON(http.StatusOK, CallbackResponse{
		Success:     true,
		RedirectURL: redirectURL,
		User:        user,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get session ID from cookie
	sessionID, err := c.Cookie("session_id")
	if err == nil && sessionID != "" {
		// Get session to retrieve ID token for logout
		session, err := h.jwtMiddleware.GetSession(c.Request.Context(), sessionID)
		if err == nil && session != nil {
			// Delete session from Redis
			h.jwtMiddleware.DeleteSession(c.Request.Context(), sessionID)

			// Delete session from database
			h.deleteSessionFromDB(sessionID)

			// Get Keycloak logout URL
			logoutURL := h.oidcService.GetLogoutURL(session.IDToken)

			// Clear session cookie
			cookie := h.jwtMiddleware.ClearSessionCookie()
			http.SetCookie(c.Writer, cookie)

			c.JSON(http.StatusOK, gin.H{
				"success":    true,
				"logout_url": logoutURL,
			})
			return
		}
	}

	// Clear session cookie even if session lookup failed
	cookie := h.jwtMiddleware.ClearSessionCookie()
	http.SetCookie(c.Writer, cookie)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// Me returns the current user info
func (h *AuthHandler) Me(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)

	// Get full user details from database
	user, err := h.getUserBySubject(auth.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":        user,
		"roles":       auth.Roles,
		"permissions": auth.Permissions,
		"session_id":  auth.SessionID,
		"expires_at":  auth.ExpiresAt,
	})
}

// RefreshToken refreshes the access token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No session found"})
		return
	}

	session, err := h.jwtMiddleware.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
		return
	}

	if session.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No refresh token available"})
		return
	}

	// Refresh the token with rotation
	newToken, err := h.oidcService.RefreshToken(c.Request.Context(), session.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to refresh token"})
		return
	}

	// Update session with new tokens (using OIDC provider's refresh token rotation)
	session.AccessToken = newToken.AccessToken
	if newToken.RefreshToken != "" {
		session.RefreshToken = newToken.RefreshToken
	}
	if idToken, ok := newToken.Extra("id_token").(string); ok {
		session.IDToken = idToken
	}
	session.ExpiresAt = newToken.Expiry
	session.LastAccessedAt = time.Now()

	// Store updated session
	err = h.jwtMiddleware.StoreSession(c.Request.Context(), sessionID, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"expires_at": newToken.Expiry,
	})
}

// Helper functions

// generateSessionID generates a random UUID session ID
func (h *AuthHandler) generateSessionID() (string, error) {
	id := uuid.New()
	return id.String(), nil
}

// findOrCreateUser finds an existing user or creates a new one
func (h *AuthHandler) findOrCreateUser(claims *models.TokenClaims) (*models.User, error) {
	// First try to find existing user
	user, err := h.getUserBySubject(claims.Subject)
	if err == nil {
		// Update user info from claims
		user.Email = claims.Email
		user.Username = claims.PreferredUsername
		if claims.GivenName != "" {
			user.FirstName = &claims.GivenName
		}
		if claims.FamilyName != "" {
			user.LastName = &claims.FamilyName
		}
		user.IsEmailVerified = claims.EmailVerified
		user.LastLoginAt = &time.Time{}
		*user.LastLoginAt = time.Now()
		user.UpdatedAt = time.Now()

		// Update user in database
		h.updateUser(user)

		// Process SCIM group mapping for existing user
		h.processSCIMGroupMapping(claims, user.ID)

		return user, nil
	}

	// Create new user
	user = &models.User{
		Sub:             &claims.Subject,
		Username:        claims.PreferredUsername,
		Email:           claims.Email,
		IsActive:        true,
		IsEmailVerified: claims.EmailVerified,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if claims.GivenName != "" {
		user.FirstName = &claims.GivenName
	}
	if claims.FamilyName != "" {
		user.LastName = &claims.FamilyName
	}
	if claims.Name != "" {
		user.DisplayName = &claims.Name
	}

	now := time.Now()
	user.LastLoginAt = &now

	// Insert user into database
	err = h.createUser(user)
	if err != nil {
		return nil, err
	}

	// Process SCIM group mapping for new user
	h.processSCIMGroupMapping(claims, user.ID)

	return user, nil
}

// Database helper functions (these would be implemented based on your database layer)

func (h *AuthHandler) getUserBySubject(subject string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT user_id, sub, username, email, first_name, last_name, display_name, 
		       is_active, is_email_verified, last_login_at, created_at, updated_at, tenant_id
		FROM users WHERE sub = $1`

	err := h.db.QueryRow(query, subject).Scan(
		&user.ID, &user.Sub, &user.Username, &user.Email,
		&user.FirstName, &user.LastName, &user.DisplayName,
		&user.IsActive, &user.IsEmailVerified, &user.LastLoginAt,
		&user.CreatedAt, &user.UpdatedAt, &user.TenantID,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *AuthHandler) createUser(user *models.User) error {
	query := `
		INSERT INTO users (tenant_id, sub, username, email, first_name, last_name, display_name, 
		                  is_active, is_email_verified, last_login_at, created_at, updated_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING user_id`

	err := h.db.QueryRow(query,
		user.Sub, user.Username, user.Email, user.FirstName, user.LastName,
		user.DisplayName, user.IsActive, user.IsEmailVerified, user.LastLoginAt,
		user.CreatedAt, user.UpdatedAt,
	).Scan(&user.ID)

	return err
}

func (h *AuthHandler) updateUser(user *models.User) error {
	query := `
		UPDATE users 
		SET username = $2, email = $3, first_name = $4, last_name = $5, 
		    display_name = $6, is_email_verified = $7, last_login_at = $8, updated_at = $9
		WHERE sub = $1`

	_, err := h.db.Exec(query,
		user.Sub, user.Username, user.Email, user.FirstName, user.LastName,
		user.DisplayName, user.IsEmailVerified, user.LastLoginAt, user.UpdatedAt,
	)

	return err
}

func (h *AuthHandler) storeSessionInDB(session *models.UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, expires_at, created_at, last_accessed_at, 
		                          ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := h.db.Exec(query,
		session.ID, session.UserID.String(), session.ExpiresAt, session.CreatedAt,
		session.LastAccessedAt, session.IPAddress, session.UserAgent,
	)

	return err
}

// processSCIMGroupMapping processes SCIM group mapping for a user
func (h *AuthHandler) processSCIMGroupMapping(claims *models.TokenClaims, userID uuid.UUID) {
	if h.scimService == nil {
		return // SCIM service not configured
	}

	// Convert token claims to interface map for SCIM processing
	claimsMap := make(map[string]interface{})
	claimsMap["sub"] = claims.Subject
	claimsMap["email"] = claims.Email
	claimsMap["preferred_username"] = claims.PreferredUsername
	claimsMap["name"] = claims.Name
	claimsMap["given_name"] = claims.GivenName
	claimsMap["family_name"] = claims.FamilyName
	claimsMap["email_verified"] = claims.EmailVerified

	// Add realm access roles if present
	if len(claims.RealmAccess.Roles) > 0 {
		claimsMap["realm_access"] = map[string]interface{}{
			"roles": claims.RealmAccess.Roles,
		}
	}

	// Add direct roles if present
	if len(claims.Roles) > 0 {
		claimsMap["roles"] = claims.Roles
		claimsMap["groups"] = claims.Roles // Some IdPs use 'roles' for groups
	}

	// Process group mapping (this runs asynchronously and won't fail authentication)
	_, err := h.scimService.ProcessUserClaims(claimsMap, userID)
	if err != nil {
		// Log error but don't fail authentication
		fmt.Printf("Failed to process SCIM group mapping for user %s: %v\n", userID.String(), err)
	}
}

func (h *AuthHandler) deleteSessionFromDB(sessionID string) error {
	query := `DELETE FROM user_sessions WHERE id = $1`
	_, err := h.db.Exec(query, sessionID)
	return err
}
