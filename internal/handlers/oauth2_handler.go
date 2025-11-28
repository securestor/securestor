package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/auth"
	"github.com/securestor/securestor/internal/models"
)

// OAuth2Handler handles OAuth2 client credentials and API key endpoints
type OAuth2Handler struct {
	oauth2Service *auth.OAuth2Service
	apiKeyService *auth.APIKeyService
}

// NewOAuth2Handler creates a new OAuth2 handler
func NewOAuth2Handler(oauth2Service *auth.OAuth2Service, apiKeyService *auth.APIKeyService) *OAuth2Handler {
	return &OAuth2Handler{
		oauth2Service: oauth2Service,
		apiKeyService: apiKeyService,
	}
}

// OAuth2 Client Credentials Token Endpoint
func (h *OAuth2Handler) Token(c *gin.Context) {
	var req auth.ClientCredentialsRequest

	// Support both JSON and form-encoded requests
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		req.GrantType = c.PostForm("grant_type")
		req.ClientID = c.PostForm("client_id")
		req.ClientSecret = c.PostForm("client_secret")
		req.Scope = c.PostForm("scope")
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": err.Error()})
			return
		}
	}

	// Validate grant type
	if req.GrantType != "client_credentials" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	// Validate client credentials
	client, err := h.oauth2Service.ValidateClientCredentials(req.ClientID, req.ClientSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client", "error_description": err.Error()})
		return
	}

	// Parse requested scopes
	var scopes []string
	if req.Scope != "" {
		scopes = strings.Fields(req.Scope)
	}

	// Generate access token
	accessToken, expiresIn, err := h.oauth2Service.GenerateAccessToken(client, scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": "Failed to generate token"})
		return
	}

	response := auth.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       strings.Join(scopes, " "),
	}

	c.JSON(http.StatusOK, response)
}

// CreateOAuth2Client creates a new OAuth2 client
func (h *OAuth2Handler) CreateOAuth2Client(c *gin.Context) {
	// Get user from context
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := strconv.ParseInt(auth.UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Scopes      []string `json:"scopes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create OAuth2 client
	client, clientSecret, err := h.oauth2Service.CreateOAuth2Client(req.Name, req.Description, req.Scopes, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OAuth2 client"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"client":        client,
		"client_secret": clientSecret, // Only returned once during creation
	})
}

// ListOAuth2Clients lists OAuth2 clients for the current user
func (h *OAuth2Handler) ListOAuth2Clients(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := strconv.ParseInt(auth.UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	clients, err := h.oauth2Service.ListOAuth2Clients(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list OAuth2 clients"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"clients": clients})
}

// RevokeOAuth2Client revokes an OAuth2 client
func (h *OAuth2Handler) RevokeOAuth2Client(c *gin.Context) {
	clientID := c.Param("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Client ID required"})
		return
	}

	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := strconv.ParseInt(auth.UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.oauth2Service.RevokeOAuth2Client(clientID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OAuth2 client revoked successfully"})
}

// CreateAPIKey creates a new API key
func (h *OAuth2Handler) CreateAPIKey(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := uuid.Parse(auth.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Name        string     `json:"name" binding:"required"`
		Description string     `json:"description"`
		Scopes      []string   `json:"scopes" binding:"required"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call the service directly - we'll fix the type interface later
	apiKey, fullKey, err := h.createAPIKeyForUser(userID, req.Name, req.Description, req.Scopes, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"api_key": apiKey,
		"key":     fullKey, // Only returned once during creation
	})
}

// ListAPIKeys lists API keys for the current user
func (h *OAuth2Handler) ListAPIKeys(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := uuid.Parse(auth.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	keys, err := h.apiKeyService.ListAPIKeys(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

// UpdateAPIKey updates an API key
func (h *OAuth2Handler) UpdateAPIKey(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key ID required"})
		return
	}

	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := uuid.Parse(auth.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Name        string   `json:"name" binding:"required"`
		Description string   `json:"description"`
		Scopes      []string `json:"scopes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.apiKeyService.UpdateAPIKey(keyID, userID, req.Name, req.Description, req.Scopes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key updated successfully"})
}

// RevokeAPIKey revokes an API key
func (h *OAuth2Handler) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("keyId")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key ID required"})
		return
	}

	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := uuid.Parse(auth.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.apiKeyService.RevokeAPIKey(keyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key revoked successfully"})
}

// GetAPIKeyStats returns API key statistics
func (h *OAuth2Handler) GetAPIKeyStats(c *gin.Context) {
	authCtx, exists := c.Get("auth_context")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	auth := authCtx.(*models.AuthContext)
	userID, err := strconv.ParseInt(auth.UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	stats, err := h.apiKeyService.GetAPIKeyStats(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get API key stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"stats": stats})
}

// GetAvailableScopes returns available OAuth2/API key scopes
func (h *OAuth2Handler) GetAvailableScopes(c *gin.Context) {
	scopes, err := h.oauth2Service.GetAvailableScopes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get available scopes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"scopes": scopes})
}

// Helper method to create API key with individual parameters
func (h *OAuth2Handler) createAPIKeyForUser(userID uuid.UUID, name, description string, scopes []string, expiresAt *time.Time) (*models.APIKey, string, error) {
	return h.apiKeyService.CreateAPIKeyDirect(userID, name, description, scopes, expiresAt)
}
