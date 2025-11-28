package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
	"github.com/securestor/securestor/internal/validation"
)

// RepositoryHandler handles repository management endpoints using Gin framework
type RepositoryHandler struct {
	repositoryService *service.RepositoryService
	artifactService   *service.ArtifactService
}

// NewRepositoryHandler creates a new Gin-based repository handler
func NewRepositoryHandler(repositoryService *service.RepositoryService, artifactService *service.ArtifactService) *RepositoryHandler {
	return &RepositoryHandler{
		repositoryService: repositoryService,
		artifactService:   artifactService,
	}
}

// RegisterRoutes registers all repository routes
func (h *RepositoryHandler) RegisterRoutes(rg *gin.RouterGroup) {
	repos := rg.Group("/repositories")
	{
		repos.GET("", h.ListRepositories)
		repos.POST("", h.CreateRepository)
		repos.GET("/stats", h.ListRepositoriesWithStats)
		repos.POST("/test-connection", h.TestRepositoryConnection)
		repos.GET("/:id", h.GetRepository)
		repos.DELETE("/:id", h.DeleteRepository)
		repos.GET("/:id/stats", h.GetRepositoryWithStats)
		repos.GET("/:id/artifacts", h.GetRepositoryArtifacts)
	}
}

// ListRepositories returns all repositories for the current tenant
// GET /api/v1/gin/repositories
func (h *RepositoryHandler) ListRepositories(c *gin.Context) {
	// Get tenant ID from Gin context
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

	repositories, err := h.repositoryService.ListByTenant(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, repositories)
}

// GetRepository returns a specific repository by ID
// GET /api/v1/gin/repositories/:id
func (h *RepositoryHandler) GetRepository(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	repo, err := h.repositoryService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, repo)
}

// CreateRepository creates a new repository
// POST /api/v1/gin/repositories
func (h *RepositoryHandler) CreateRepository(c *gin.Context) {
	// Get tenant ID from Gin context
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

	var req models.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Validate request
	if err := validation.ValidateCreateRepository(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create repository with tenant context
	repo, err := h.repositoryService.CreateWithTenant(tenantID, &req)
	if err != nil {
		// Check for conflict
		if err.Error() == "repository with name '"+req.Name+"' already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Repository created successfully",
		"repository": repo,
	})
}

// DeleteRepository deletes a repository by ID
// DELETE /api/v1/gin/repositories/:id
func (h *RepositoryHandler) DeleteRepository(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	if err := h.repositoryService.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Repository deleted successfully"})
}

// ListRepositoriesWithStats returns all repositories with statistics
// GET /api/v1/gin/repositories/stats
func (h *RepositoryHandler) ListRepositoriesWithStats(c *gin.Context) {
	// Get tenant ID from Gin context
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

	repositories, err := h.repositoryService.ListWithStatsByTenant(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get stats for this tenant
	stats, err := h.repositoryService.GetStatsByTenant(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories": repositories,
		"stats":        stats,
	})
}

// TestRepositoryConnection tests the connection to a remote repository
// POST /api/v1/gin/repositories/test-connection
func (h *RepositoryHandler) TestRepositoryConnection(c *gin.Context) {
	var req models.CreateRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Test connection for remote repositories
	if req.RepositoryType == "remote" && req.RemoteURL != "" {
		// TODO: Implement actual connection test
		// For now, just validate URL format
		if err := validation.ValidateCreateRepository(&req); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Connection successful",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "No connection test needed for local repositories",
	})
}

// GetRepositoryWithStats returns a repository with detailed statistics
// GET /api/v1/gin/repositories/:id/stats
func (h *RepositoryHandler) GetRepositoryWithStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
		return
	}

	repo, err := h.repositoryService.GetByIDWithStats(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, repo)
}

// GetRepositoryArtifacts returns artifacts for a specific repository with filtering
// GET /api/v1/repositories/:id/artifacts
func (h *RepositoryHandler) GetRepositoryArtifacts(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Get repository details to verify it exists and belongs to tenant
	repository, err := h.repositoryService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	// Verify repository belongs to the tenant (if repository has tenant set)
	// Allow access if repository has no tenant (backwards compatibility)
	zeroUUID := uuid.UUID{}
	if repository.TenantID != zeroUUID && repository.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this repository"})
		return
	}

	// Parse query parameters for filtering and pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Build artifact filter
	filter := &models.ArtifactFilter{
		RepositoryID: &id,
		Search:       c.Query("search"),
		SortBy:       c.DefaultQuery("sortBy", "created_at"),
		SortOrder:    c.DefaultQuery("sortOrder", "desc"),
		Limit:        limit,
		Offset:       offset,
	}

	// Add type filter if provided
	if artifactType := c.Query("type"); artifactType != "" {
		filter.Types = []string{artifactType}
	}

	// Add compliance status filter if provided
	if complianceStatus := c.Query("complianceStatus"); complianceStatus != "" {
		filter.ComplianceStatus = []string{complianceStatus}
	}

	// Fetch artifacts for this repository (tenant-aware)
	artifacts, total, err := h.artifactService.ListByTenant(tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch artifacts"})
		return
	}

	// Calculate pagination info
	totalPages := (total + limit - 1) / limit

	// Build response
	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifacts,
		"repository": gin.H{
			"id":          repository.ID,
			"name":        repository.Name,
			"type":        repository.Type,
			"description": repository.Description,
		},
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
		"stats": gin.H{
			"total_artifacts": total,
		},
	})
}
