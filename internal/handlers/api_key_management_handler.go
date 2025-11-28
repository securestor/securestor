package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/securestor/securestor/internal/service"
)

// APIKeyManagementHandler handles API key management endpoints using Gin
type APIKeyManagementHandler struct {
	apiKeyService *service.APIKeyManagementService
}

// NewAPIKeyManagementHandler creates a new API key management handler for Gin
func NewAPIKeyManagementHandler(apiKeyService *service.APIKeyManagementService) *APIKeyManagementHandler {
	return &APIKeyManagementHandler{
		apiKeyService: apiKeyService,
	}
}

// RegisterRoutes registers API key management routes
func (h *APIKeyManagementHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// API key CRUD operations
	rg.GET("/keys", h.GetAPIKeys)
	rg.POST("/keys", h.CreateAPIKey)
	rg.GET("/keys/:id", h.GetAPIKey)
	rg.DELETE("/keys/:id", h.RevokeAPIKey)

	// API key analytics and monitoring
	rg.GET("/keys/:id/analytics", h.GetAPIKeyAnalytics)
	rg.GET("/keys/:id/usage", h.GetAPIKeyUsage)

	// API scopes management
	rg.GET("/scopes", h.GetAPIScopes)

	// Validation endpoint (for middleware use)
	rg.POST("/keys/validate", h.ValidateAPIKey)
}

// GetAPIKeys handles GET /keys - List API keys with filtering
func (h *APIKeyManagementHandler) GetAPIKeys(c *gin.Context) {
	// Extract tenant context - ENFORCE tenant isolation
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	// Parse query parameters for filtering
	filter := &service.APIKeyFilter{
		Search:    c.Query("search"),
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

	// Parse filters
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userUUID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userUUID
		}
	}

	// ENFORCE: Use tenant UUID directly for filtering
	filter.TenantID = &tenantID

	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			filter.IsActive = &isActive
		}
	}

	if scopesStr := c.Query("scopes"); scopesStr != "" {
		filter.Scopes = strings.Split(scopesStr, ",")
	}

	// Get API keys
	apiKeys, totalCount, err := h.apiKeyService.GetAPIKeys(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get API keys: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys":    apiKeys,
		"total_count": totalCount,
		"limit":       filter.Limit,
		"offset":      filter.Offset,
	})
}

// GetAPIKey handles GET /keys/:id - Get specific API key
func (h *APIKeyManagementHandler) GetAPIKey(c *gin.Context) {
	// ENFORCE: Extract tenant context for isolation
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(c.Request.Context(), keyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get API key: %v", err)})
		return
	}

	// ENFORCE: Verify API key belongs to current tenant (tenant isolation)
	if apiKey.TenantID != tenantID {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"}) // Don't reveal existence
		return
	}

	c.JSON(http.StatusOK, apiKey)
}

// CreateAPIKey handles POST /keys - Create new API key
func (h *APIKeyManagementHandler) CreateAPIKey(c *gin.Context) {
	type CreateAPIKeyRequest struct {
		Name             string   `json:"name" binding:"required"`
		Description      string   `json:"description"`
		Scopes           []string `json:"scopes" binding:"required,min=1"`
		ExpiresAt        *string  `json:"expires_at,omitempty"`
		RateLimitPerHour int      `json:"rate_limit_per_hour"`
		RateLimitPerDay  int      `json:"rate_limit_per_day"`
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Set defaults for rate limits
	if req.RateLimitPerHour <= 0 {
		req.RateLimitPerHour = 1000
	}
	if req.RateLimitPerDay <= 0 {
		req.RateLimitPerDay = 10000
	}

	// Parse expiration date
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expiration date format (use RFC3339)"})
			return
		}
		expiresAt = &parsedTime
	}

	// Get current user ID from auth context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication context not found"})
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID format"})
		return
	}

	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	apiKey, err := h.apiKeyService.CreateAPIKey(
		c.Request.Context(),
		userID,
		tenantID,
		req.Name,
		req.Description,
		req.Scopes,
		expiresAt,
		req.RateLimitPerHour,
		req.RateLimitPerDay,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "API key name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create API key: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, apiKey)
}

// RevokeAPIKey handles DELETE /keys/:id - Revoke API key
func (h *APIKeyManagementHandler) RevokeAPIKey(c *gin.Context) {
	// ENFORCE: Extract tenant context for isolation
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// ENFORCE: Verify API key belongs to tenant before allowing revocation
	apiKey, err := h.apiKeyService.GetAPIKeyByID(c.Request.Context(), keyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get API key: %v", err)})
		return
	}

	if apiKey.TenantID != tenantID {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"}) // Don't reveal existence
		return
	}

	type RevokeAPIKeyRequest struct {
		Reason string `json:"reason"`
	}

	var req RevokeAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Reason is optional, so we can continue without it
		req.Reason = "Revoked by user"
	}

	err = h.apiKeyService.RevokeAPIKey(c.Request.Context(), keyID, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to revoke API key: %v", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAPIKeyAnalytics handles GET /keys/:id/analytics - Get usage analytics
func (h *APIKeyManagementHandler) GetAPIKeyAnalytics(c *gin.Context) {
	// ENFORCE: Extract tenant context for isolation
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// ENFORCE: Verify API key belongs to tenant before showing analytics
	apiKey, err := h.apiKeyService.GetAPIKeyByID(c.Request.Context(), keyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get API key: %v", err)})
		return
	}

	if apiKey.TenantID != tenantID {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"}) // Don't reveal existence
		return
	}

	// Parse days parameter
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	analytics, err := h.apiKeyService.GetUsageAnalytics(c.Request.Context(), keyID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get analytics: %v", err)})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// GetAPIKeyUsage handles GET /keys/:id/usage - Get current usage stats
func (h *APIKeyManagementHandler) GetAPIKeyUsage(c *gin.Context) {
	// ENFORCE: Extract tenant context for isolation
	tenantIDVal, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}

	tenantID, ok := tenantIDVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// Get the API key to verify access
	apiKey, err := h.apiKeyService.GetAPIKeyByID(c.Request.Context(), keyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get API key: %v", err)})
		return
	}

	// ENFORCE: Verify API key belongs to current tenant
	if apiKey.TenantID != tenantID {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"}) // Don't reveal existence
		return
	}

	// Return basic usage information
	c.JSON(http.StatusOK, gin.H{
		"key_id":          apiKey.ID,
		"name":            apiKey.Name,
		"usage_count":     apiKey.UsageCount,
		"last_used_at":    apiKey.LastUsedAt,
		"last_used_ip":    apiKey.LastUsedIP,
		"rate_limit_hour": apiKey.RateLimitPerHour,
		"rate_limit_day":  apiKey.RateLimitPerDay,
		"is_active":       apiKey.IsActive,
		"expires_at":      apiKey.ExpiresAt,
	})
}

// GetAPIScopes handles GET /scopes - Get available API scopes
func (h *APIKeyManagementHandler) GetAPIScopes(c *gin.Context) {
	scopes, err := h.apiKeyService.GetAPIScopes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get API scopes: %v", err)})
		return
	}

	// Group scopes by resource for better organization
	groupedScopes := make(map[string][]service.APIScope)
	for _, scope := range scopes {
		if groupedScopes[scope.Resource] == nil {
			groupedScopes[scope.Resource] = []service.APIScope{}
		}
		groupedScopes[scope.Resource] = append(groupedScopes[scope.Resource], scope)
	}

	c.JSON(http.StatusOK, gin.H{
		"scopes":         scopes,
		"grouped_scopes": groupedScopes,
	})
}

// ValidateAPIKey handles POST /keys/validate - Validate API key
func (h *APIKeyManagementHandler) ValidateAPIKey(c *gin.Context) {
	type ValidateAPIKeyRequest struct {
		APIKey string `json:"api_key" binding:"required"`
	}

	var req ValidateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	apiKey, err := h.apiKeyService.ValidateAPIKey(c.Request.Context(), req.APIKey)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "expired") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired API key"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to validate API key: %v", err)})
		return
	}

	// Check rate limits
	withinLimits, err := h.apiKeyService.CheckRateLimit(c.Request.Context(), apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to check rate limits: %v", err)})
		return
	}

	if !withinLimits {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
		return
	}

	// Update usage tracking
	clientIP := c.ClientIP()

	err = h.apiKeyService.UpdateAPIKeyUsage(c.Request.Context(), apiKey.KeyID, clientIP)
	if err != nil {
		// Log error but don't fail the request
		// logger.Errorf("Failed to update API key usage: %v", err)
	}

	// Return API key info (without sensitive data)
	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"key_id":     apiKey.ID,
		"user_id":    apiKey.UserID,
		"scopes":     apiKey.Scopes,
		"expires_at": apiKey.ExpiresAt,
		"name":       apiKey.Name,
	})
}
