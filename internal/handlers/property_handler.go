package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// PropertyHandler handles property-related HTTP requests using Gin framework
type PropertyHandler struct {
	propertyService *service.PropertyService
	log             *logger.Logger
}

// NewPropertyHandler creates a new Gin-based property handler
func NewPropertyHandler(propertyService *service.PropertyService, log *logger.Logger) *PropertyHandler {
	return &PropertyHandler{
		propertyService: propertyService,
		log:             log,
	}
}

// RegisterRoutes registers property routes on Gin router
func (h *PropertyHandler) RegisterRoutes(router *gin.RouterGroup) {
	h.log.Info("Registering Gin property routes...")

	// All property routes require authentication (handled by middleware)

	// Property-centric routes under /properties
	properties := router.Group("/properties")
	{
		// Artifact property routes
		properties.POST("/artifacts/:artifact_id", h.CreateArtifactProperty)
		properties.GET("/artifacts/:artifact_id", h.GetArtifactProperties)

		// Property CRUD routes
		properties.GET("/:id", h.GetProperty)
		properties.PUT("/:id", h.UpdateProperty)
		properties.DELETE("/:id", h.DeleteProperty)

		// Property query routes
		properties.POST("/search", h.SearchProperties)
		properties.GET("/statistics", h.GetStatistics)
	}

	// Artifact-centric routes under /artifacts/:id/properties (for frontend compatibility)
	artifacts := router.Group("/artifacts")
	{
		artifacts.POST("/:id/properties", h.CreateArtifactProperty)
		artifacts.GET("/:id/properties", h.GetArtifactProperties)
	}

	h.log.Info("Gin property routes registered successfully")
}

// CreateArtifactProperty creates a new property for an artifact
// @Summary Create artifact property
// @Tags Properties
// @Accept json
// @Produce json
// @Param artifact_id path string true "Artifact ID"
// @Param property body models.CreatePropertyRequest true "Property details"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/artifacts/{artifact_id} [post]
func (h *PropertyHandler) CreateArtifactProperty(c *gin.Context) {
	// Support both :id and :artifact_id parameter names
	artifactID := c.Param("artifact_id")
	if artifactID == "" {
		artifactID = c.Param("id")
	}
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artifact_id is required"})
		return
	}

	// Get tenant ID from context (set by tenant middleware)
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

	// Get user ID from auth context (set by auth middleware)
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		// Try string conversion
		userIDStr, ok := userIDVal.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
			return
		}
		var err error
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
			return
		}
	}

	// Parse request
	var req models.CreatePropertyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Validate required fields
	if req.Key == "" || req.Value == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key and value are required"})
		return
	}

	// For now, use tenant_id as repository_id
	repositoryID := tenantID.String()

	// Create property
	createdProperty, err := h.propertyService.CreateProperty(c.Request.Context(), tenantID, repositoryID, artifactID, &req, userID)
	if err != nil {
		h.log.Error("Failed to create property", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create property: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"property": createdProperty})
}

// GetArtifactProperties lists all properties for an artifact
// @Summary List artifact properties
// @Tags Properties
// @Produce json
// @Param artifact_id path string true "Artifact ID"
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param mask_sensitive query bool false "Mask sensitive values"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/artifacts/{artifact_id} [get]
func (h *PropertyHandler) GetArtifactProperties(c *gin.Context) {
	// Support both :id and :artifact_id parameter names
	artifactID := c.Param("artifact_id")
	if artifactID == "" {
		artifactID = c.Param("id")
	}
	if artifactID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artifact_id is required"})
		return
	}

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

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	maskSensitive, _ := strconv.ParseBool(c.DefaultQuery("mask_sensitive", "true"))

	// Get properties using the correct service method
	properties, total, err := h.propertyService.GetPropertiesByArtifact(
		c.Request.Context(),
		tenantID,
		artifactID,
		limit,
		offset,
		maskSensitive,
	)
	if err != nil {
		h.log.Error("Failed to get properties", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get properties: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"properties": properties,
		"total":      total,
		"count":      len(properties),
	})
}

// GetProperty retrieves a single property by ID
// @Summary Get property
// @Tags Properties
// @Produce json
// @Param id path string true "Property ID"
// @Success 200 {object} models.ArtifactProperty
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/{id} [get]
func (h *PropertyHandler) GetProperty(c *gin.Context) {
	propertyID := c.Param("id")
	if propertyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property ID is required"})
		return
	}

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

	propertyUUID, err := uuid.Parse(propertyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	// Get property
	property, err := h.propertyService.GetProperty(c.Request.Context(), tenantID, propertyUUID)
	if err != nil {
		h.log.Error("Failed to get property", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Property not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, property)
}

// UpdateProperty updates an existing property
// @Summary Update property
// @Tags Properties
// @Accept json
// @Produce json
// @Param id path string true "Property ID"
// @Param updates body models.UpdatePropertyRequest true "Property updates"
// @Success 200 {object} models.ArtifactProperty
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/{id} [put]
func (h *PropertyHandler) UpdateProperty(c *gin.Context) {
	propertyID := c.Param("id")
	if propertyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property ID is required"})
		return
	}

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

	// Get user ID from auth context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		userIDStr, ok := userIDVal.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
			return
		}
		var err error
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
			return
		}
	}

	// Parse request
	var updates models.UpdatePropertyRequest
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	propertyUUID, err := uuid.Parse(propertyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	// Update property
	updatedProperty, err := h.propertyService.UpdateProperty(c.Request.Context(), tenantID, propertyUUID, &updates, userID)
	if err != nil {
		h.log.Error("Failed to update property", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update property: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedProperty)
}

// DeleteProperty deletes a property
// @Summary Delete property
// @Tags Properties
// @Param id path string true "Property ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/{id} [delete]
func (h *PropertyHandler) DeleteProperty(c *gin.Context) {
	propertyID := c.Param("id")
	if propertyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "property ID is required"})
		return
	}

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

	// Get user ID from auth context
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		userIDStr, ok := userIDVal.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
			return
		}
		var err error
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID format"})
			return
		}
	}

	propertyUUID, err := uuid.Parse(propertyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid property ID"})
		return
	}

	// Delete property
	if err := h.propertyService.DeleteProperty(c.Request.Context(), tenantID, propertyUUID, userID); err != nil {
		h.log.Error("Failed to delete property", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete property: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// SearchProperties searches properties with advanced filters
// @Summary Search properties
// @Tags Properties
// @Accept json
// @Produce json
// @Param search body models.PropertySearchRequest true "Search criteria"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/search [post]
func (h *PropertyHandler) SearchProperties(c *gin.Context) {
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

	// Parse request
	var req models.SearchPropertiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Search properties with maskSensitive=true
	properties, total, err := h.propertyService.SearchProperties(c.Request.Context(), tenantID, &req, true)
	if err != nil {
		h.log.Error("Failed to search properties", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search properties: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"properties": properties,
		"total":      total,
		"count":      len(properties),
	})
}

// GetStatistics retrieves property statistics
// @Summary Get property statistics
// @Tags Properties
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /properties/statistics [get]
func (h *PropertyHandler) GetStatistics(c *gin.Context) {
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

	// Get statistics
	stats, err := h.propertyService.GetStatistics(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("Failed to get statistics", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
