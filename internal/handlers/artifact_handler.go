package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
)

// ArtifactHandler handles artifact-related endpoints using Gin framework
type ArtifactHandler struct {
	artifactService *service.ArtifactService
	scanService     *service.ScanService
}

// NewArtifactHandler creates a new artifact handler for Gin
func NewArtifactHandler(
	artifactService *service.ArtifactService,
	scanService *service.ScanService,
) *ArtifactHandler {
	return &ArtifactHandler{
		artifactService: artifactService,
		scanService:     scanService,
	}
}

// ListArtifacts handles GET /artifacts
// @Summary List artifacts
// @Tags Artifacts
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param search query string false "Search term"
// @Param type query string false "Artifact type"
// @Param repository_id query string false "Repository ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /gin/artifacts [get]
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	search := c.Query("search")
	artifactType := c.Query("type")
	repositoryID := c.Query("repository_id")

	filter := &models.ArtifactFilter{
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	if artifactType != "" {
		filter.Types = []string{artifactType}
	}

	if repositoryID != "" {
		if repoUUID, err := uuid.Parse(repositoryID); err == nil {
			filter.RepositoryID = &repoUUID
		}
	}

	// List artifacts for the tenant
	artifacts, total, err := h.artifactService.ListByTenant(tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list artifacts: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifacts,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetArtifact handles GET /artifacts/:id
// @Summary Get artifact by ID
// @Tags Artifacts
// @Produce json
// @Param id path string true "Artifact ID"
// @Success 200 {object} models.Artifact
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /gin/artifacts/{id} [get]
func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
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

	// Parse artifact ID
	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact
	artifact, err := h.artifactService.GetByID(artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Verify tenant access
	if artifact.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this artifact"})
		return
	}

	c.JSON(http.StatusOK, artifact)
}

// CreateArtifact handles POST /artifacts
// @Summary Create artifact metadata
// @Tags Artifacts
// @Accept json
// @Produce json
// @Param artifact body models.CreateArtifactRequest true "Artifact data"
// @Success 201 {object} models.Artifact
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /gin/artifacts [post]
func (h *ArtifactHandler) CreateArtifact(c *gin.Context) {
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

	// Parse request body
	var req struct {
		Name         string    `json:"name" binding:"required"`
		Version      string    `json:"version"`
		Type         string    `json:"type" binding:"required"`
		RepositoryID uuid.UUID `json:"repository_id" binding:"required"`
		Description  string    `json:"description"`
		Tags         []string  `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Create artifact
	artifact := &models.Artifact{
		Name:         req.Name,
		Version:      req.Version,
		Type:         req.Type,
		RepositoryID: req.RepositoryID,
		TenantID:     tenantID,
		Tags:         req.Tags,
	}

	if err := h.artifactService.Create(artifact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create artifact: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, artifact)
}

// DeleteArtifact handles DELETE /artifacts/:id
// @Summary Delete artifact
// @Tags Artifacts
// @Param id path string true "Artifact ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /gin/artifacts/{id} [delete]
func (h *ArtifactHandler) DeleteArtifact(c *gin.Context) {
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

	// Parse artifact ID
	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact to verify ownership
	artifact, err := h.artifactService.GetByID(artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Verify tenant access
	if artifact.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this artifact"})
		return
	}

	// Delete artifact
	if err := h.artifactService.Delete(artifactID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete artifact: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// SearchArtifacts handles POST /artifacts/search
// @Summary Search artifacts
// @Tags Artifacts
// @Accept json
// @Produce json
// @Param search body models.ArtifactSearchRequest true "Search criteria"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /gin/artifacts/search [post]
func (h *ArtifactHandler) SearchArtifacts(c *gin.Context) {
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

	// Parse search request
	var searchReq struct {
		Query     string   `json:"query"`
		Types     []string `json:"types"`
		Tags      []string `json:"tags"`
		Limit     int      `json:"limit"`
		Offset    int      `json:"offset"`
		SortBy    string   `json:"sort_by"`
		SortOrder string   `json:"sort_order"`
	}

	if err := c.ShouldBindJSON(&searchReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid search request: " + err.Error()})
		return
	}

	// Build filter
	filter := &models.ArtifactFilter{
		Search: searchReq.Query,
		Types:  searchReq.Types,
		Tags:   searchReq.Tags,
		Limit:  searchReq.Limit,
		Offset: searchReq.Offset,
	}

	if filter.Limit == 0 {
		filter.Limit = 50
	}

	// Search artifacts
	artifacts, total, err := h.artifactService.ListByTenant(tenantID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search artifacts: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifacts,
		"total":     total,
		"query":     searchReq.Query,
	})
}

// ScanArtifact handles POST /artifacts/:id/scan
// @Summary Scan artifact for vulnerabilities
// @Tags Artifacts
// @Param id path string true "Artifact ID"
// @Success 202 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /gin/artifacts/{id}/scan [post]
func (h *ArtifactHandler) ScanArtifact(c *gin.Context) {
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

	// Parse artifact ID
	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact to verify ownership
	artifact, err := h.artifactService.GetByID(artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Verify tenant access
	if artifact.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this artifact"})
		return
	}

	// Parse scan options
	var scanReq struct {
		ScanType string `json:"scan_type"` // full, quick, vulnerability, malware, license, dependency
		Priority string `json:"priority"`  // low, normal, high, critical
	}
	c.ShouldBindJSON(&scanReq)

	// Default values
	if scanReq.ScanType == "" {
		scanReq.ScanType = "full"
	}
	if scanReq.Priority == "" {
		scanReq.Priority = "normal"
	}

	// Initiate scan using SecurityScan model
	scan := &models.SecurityScan{
		ArtifactID:        artifactID,
		TenantID:          tenantID,
		Status:            "initiated",
		ScanType:          scanReq.ScanType,
		Priority:          scanReq.Priority,
		VulnerabilityScan: scanReq.ScanType == "full" || scanReq.ScanType == "vulnerability",
		MalwareScan:       scanReq.ScanType == "full" || scanReq.ScanType == "malware",
		LicenseScan:       scanReq.ScanType == "full" || scanReq.ScanType == "license",
		DependencyScan:    scanReq.ScanType == "full" || scanReq.ScanType == "dependency",
	}

	if err := h.scanService.CreateScan(scan); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate scan: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Scan initiated successfully",
		"scan_id":  scan.ID,
		"status":   scan.Status,
		"artifact": artifact.Name,
	})
}

// GetScanResults handles GET /artifacts/:id/scan/results
// @Summary Get scan results for artifact
// @Tags Artifacts
// @Param id path string true "Artifact ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /gin/artifacts/{id}/scan/results [get]
func (h *ArtifactHandler) GetScanResults(c *gin.Context) {
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

	// Parse artifact ID
	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact to verify ownership
	artifact, err := h.artifactService.GetByID(artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Verify tenant access
	if artifact.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this artifact"})
		return
	}

	// Get scan results
	scans, _, err := h.scanService.GetScanHistory(artifactID, 1, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scan results: " + err.Error()})
		return
	}

	if len(scans) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No scan results found for this artifact"})
		return
	}

	// Return the most recent scan
	latestScan := scans[0]
	c.JSON(http.StatusOK, gin.H{
		"scan":     latestScan,
		"artifact": artifact,
	})
}

// GetScanHistory handles GET /artifacts/:id/scan/history
// @Summary Get scan history for artifact
// @Tags Artifacts
// @Param id path string true "Artifact ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /gin/artifacts/{id}/scan/history [get]
func (h *ArtifactHandler) GetScanHistory(c *gin.Context) {
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

	// Parse artifact ID
	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact to verify ownership
	artifact, err := h.artifactService.GetByID(artifactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Verify tenant access
	if artifact.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this artifact"})
		return
	}

	// Get all scans for this artifact
	limit := 100 // default limit
	offset := 0
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	scans, total, err := h.scanService.GetScanHistory(artifactID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scan history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scans":    scans,
		"total":    total,
		"artifact": artifact.Name,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetAvailableScanners handles GET /artifacts/scanners
// @Summary Get available scanners
// @Tags Artifacts
// @Success 200 {object} map[string]interface{}
// @Router /gin/artifacts/scanners [get]
func (h *ArtifactHandler) GetAvailableScanners(c *gin.Context) {
	scanners := h.scanService.GetAvailableScanners()

	c.JSON(http.StatusOK, gin.H{
		"scanners": scanners,
		"total":    len(scanners),
	})
}

// RegisterRoutes registers all artifact routes
func (h *ArtifactHandler) RegisterRoutes(router *gin.RouterGroup) {
	// List and create artifacts
	router.GET("/artifacts", h.ListArtifacts)
	router.POST("/artifacts", h.CreateArtifact)

	// Search artifacts
	router.POST("/artifacts/search", h.SearchArtifacts)

	// Scanner information
	router.GET("/artifacts/scanners", h.GetAvailableScanners)

	// Specific artifact operations
	router.GET("/artifacts/:id", h.GetArtifact)
	router.DELETE("/artifacts/:id", h.DeleteArtifact)

	// Scanning operations
	router.POST("/artifacts/:id/scan", h.ScanArtifact)
	router.GET("/artifacts/:id/scan/results", h.GetScanResults)
	router.GET("/artifacts/:id/scan/history", h.GetScanHistory)
}
