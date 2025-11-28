package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"github.com/securestor/securestor/internal/handlers"
	securelogger "github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/replicate"
	"github.com/securestor/securestor/internal/repository"
	"github.com/securestor/securestor/internal/service"
	"github.com/securestor/securestor/internal/storage"
)

// HelmHandler handles Helm chart repository operations with enterprise replication
type HelmHandler struct {
	helmHandler       *storage.HelmOCIHandler
	storagePath       string
	artifactService   *service.ArtifactService
	repositoryService *service.RepositoryService
	artifactRepo      *repository.ArtifactRepository
	replicationMixin  *handlers.ReplicationMixin
}

// HelmChartMetadata represents the Chart.yaml structure
type HelmChartMetadata struct {
	APIVersion   string            `yaml:"apiVersion" json:"apiVersion"`
	Name         string            `yaml:"name" json:"name"`
	Version      string            `yaml:"version" json:"version"`
	KubeVersion  string            `yaml:"kubeVersion,omitempty" json:"kubeVersion,omitempty"`
	Description  string            `yaml:"description,omitempty" json:"description,omitempty"`
	Type         string            `yaml:"type,omitempty" json:"type,omitempty"`
	Keywords     []string          `yaml:"keywords,omitempty" json:"keywords,omitempty"`
	Home         string            `yaml:"home,omitempty" json:"home,omitempty"`
	Sources      []string          `yaml:"sources,omitempty" json:"sources,omitempty"`
	Dependencies []HelmDependency  `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Maintainers  []HelmMaintainer  `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	Icon         string            `yaml:"icon,omitempty" json:"icon,omitempty"`
	AppVersion   string            `yaml:"appVersion,omitempty" json:"appVersion,omitempty"`
	Deprecated   bool              `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Annotations  map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

// HelmDependency represents a chart dependency
type HelmDependency struct {
	Name       string `yaml:"name" json:"name"`
	Version    string `yaml:"version" json:"version"`
	Repository string `yaml:"repository" json:"repository"`
	Condition  string `yaml:"condition,omitempty" json:"condition,omitempty"`
}

// HelmMaintainer represents a chart maintainer
type HelmMaintainer struct {
	Name  string `yaml:"name" json:"name"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
}

// HelmIndexEntry represents an entry in index.yaml
type HelmIndexEntry struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	AppVersion  string            `json:"appVersion,omitempty"`
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Home        string            `json:"home,omitempty"`
	Sources     []string          `json:"sources,omitempty"`
	Maintainers []HelmMaintainer  `json:"maintainers,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	APIVersion  string            `json:"apiVersion"`
	URLs        []string          `json:"urls"`
	Created     time.Time         `json:"created"`
	Digest      string            `json:"digest"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// HelmIndex represents the index.yaml file
type HelmIndex struct {
	APIVersion string                      `json:"apiVersion"`
	Generated  time.Time                   `json:"generated"`
	Entries    map[string][]HelmIndexEntry `json:"entries"`
}

// NewHelmHandler creates a new Helm handler with replication support
func NewHelmHandler(storagePath string, artifactService *service.ArtifactService, repositoryService *service.RepositoryService, artifactRepo *repository.ArtifactRepository, replicationMixin *handlers.ReplicationMixin) *HelmHandler {
	return &HelmHandler{
		helmHandler:       storage.NewHelmOCIHandler(storagePath),
		storagePath:       storagePath,
		artifactService:   artifactService,
		repositoryService: repositoryService,
		artifactRepo:      artifactRepo,
		replicationMixin:  replicationMixin,
	}
}

// RegisterRoutes sets up Helm chart repository routes on a Gin router
func (h *HelmHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Helm chart repository routes

	// Index file (required by Helm clients)
	r.GET("/index.yaml", h.ginHandleGetIndex)
	r.GET("/:repo/index.yaml", h.ginHandleGetIndex)

	// Chart upload
	r.POST("/:repo/api/charts", h.ginHandleUploadChart)
	r.POST("/api/charts", h.ginHandleUploadChart)

	// Chart download
	r.GET("/charts/:filename", h.ginHandleDownloadChart)
	r.GET("/:repo/charts/:filename", h.ginHandleDownloadChart)

	// Chart deletion
	r.DELETE("/api/charts/:name/:version", h.ginHandleDeleteChart)
}

// ginHandleGetIndex returns the index.yaml file listing all available charts (Gin version)
func (h *HelmHandler) ginHandleGetIndex(c *gin.Context) {
	repoName := c.Param("repo")
	if repoName == "" {
		repoName = "default"
	}

	// Get all Helm repositories
	repos, err := h.repositoryService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list repositories"})
		return
	}

	// Find the requested repository or use first Helm repo
	var targetRepo *models.Repository
	for _, repo := range repos {
		if repo.Type == "helm" && (repo.Name == repoName || repoName == "default") {
			targetRepo = &repo
			break
		}
	}

	if targetRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Helm repository not found"})
		return
	}

	// Get all artifacts from this repository
	// Build index entries
	index := &HelmIndex{
		APIVersion: "v1",
		Entries:    make(map[string][]HelmIndexEntry),
		Generated:  time.Now(),
	}

	// Query artifacts for this repository
	filter := &models.ArtifactFilter{
		RepositoryID: &targetRepo.ID,
	}
	artifacts, _, err := h.artifactRepo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query artifacts"})
		return
	}

	// Build index entries from artifacts
	for _, artifact := range artifacts {
		if artifact.Type != "helm" {
			continue
		}

		// Extract metadata
		chartName := artifact.Name
		entry := HelmIndexEntry{
			Name:    chartName,
			Version: artifact.Version,
			Created: artifact.CreatedAt,
			Digest:  artifact.Checksum,
		}

		// Extract additional metadata from artifact.Metadata if available
		if artifact.Metadata != nil {
			if desc, ok := artifact.Metadata["description"].(string); ok {
				entry.Description = desc
			}
			if appVer, ok := artifact.Metadata["appVersion"].(string); ok {
				entry.AppVersion = appVer
			}
		}

		// Add to index
		if index.Entries[chartName] == nil {
			index.Entries[chartName] = make([]HelmIndexEntry, 0)
		}
		index.Entries[chartName] = append(index.Entries[chartName], entry)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(index)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate index"})
		return
	}

	c.Data(http.StatusOK, "application/x-yaml", yamlData)
}

// ginHandleUploadChart handles Helm chart upload (Gin version)
func (h *HelmHandler) ginHandleUploadChart(c *gin.Context) {
	repoName := c.Param("repo")
	if repoName == "" {
		repoName = "default"
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("chart")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to get chart file: %v", err)})
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(header.Filename, ".tgz") && !strings.HasSuffix(header.Filename, ".tar.gz") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chart file must be a .tgz or .tar.gz archive"})
		return
	}

	// Read file content
	chartData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read chart data"})
		return
	}

	// Extract chart metadata
	metadata, err := h.extractChartMetadata(chartData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to extract chart metadata: %v", err)})
		return
	}

	// Get or create Helm repository
	repos, err := h.repositoryService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list repositories"})
		return
	}

	var targetRepo *models.Repository
	for _, repo := range repos {
		if repo.Type == "helm" && repo.Name == repoName {
			targetRepo = &repo
			break
		}
	}

	if targetRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Helm repository not found"})
		return
	}

	// Calculate checksum
	hash := sha256.Sum256(chartData)
	checksum := fmt.Sprintf("%x", hash)

	// Generate filename
	filename := fmt.Sprintf("%s-%s.tgz", metadata.Name, metadata.Version)

	// Save chart to storage
	chartPath := filepath.Join(h.storagePath, targetRepo.ID.String(), filename)
	if err := os.MkdirAll(filepath.Dir(chartPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create storage directory"})
		return
	}

	if err := os.WriteFile(chartPath, chartData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write chart file"})
		return
	}

	// Create artifact record
	artifact := &models.Artifact{
		Name:         metadata.Name,
		Version:      metadata.Version,
		Type:         "helm",
		RepositoryID: targetRepo.ID,
		Size:         int64(len(chartData)),
		Checksum:     checksum,
		Metadata: map[string]interface{}{
			"description": metadata.Description,
			"appVersion":  metadata.AppVersion,
			"apiVersion":  metadata.APIVersion,
			"kubeVersion": metadata.KubeVersion,
			"keywords":    metadata.Keywords,
			"home":        metadata.Home,
			"sources":     metadata.Sources,
			"maintainers": metadata.Maintainers,
			"deprecated":  metadata.Deprecated,
			"file_path":   chartPath,
			"filename":    filename,
		},
	}

	if err := h.artifactRepo.Create(artifact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create artifact: %v", err)})
		return
	}

	// Trigger replication asynchronously
	go h.replicateHelmChart(targetRepo.TenantID.String(), targetRepo.ID.String(), artifact.ID.String(), chartData)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Chart uploaded successfully",
		"chart": gin.H{
			"name":    metadata.Name,
			"version": metadata.Version,
			"id":      artifact.ID,
		},
	})
}

// ginHandleDownloadChart handles chart file downloads (Gin version)
func (h *HelmHandler) ginHandleDownloadChart(c *gin.Context) {
	filename := c.Param("filename")
	repoName := c.Param("repo")

	if repoName == "" {
		repoName = "default"
	}

	// Get repository
	repos, err := h.repositoryService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list repositories"})
		return
	}

	var targetRepo *models.Repository
	for _, repo := range repos {
		if repo.Type == "helm" && repo.Name == repoName {
			targetRepo = &repo
			break
		}
	}

	if targetRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	// Build file path
	chartPath := filepath.Join(h.storagePath, targetRepo.ID.String(), filename)

	// Check if file exists
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chart not found"})
		return
	}

	// Serve file
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.File(chartPath)
}

// ginHandleDeleteChart handles chart deletion (Gin version)
func (h *HelmHandler) ginHandleDeleteChart(c *gin.Context) {
	chartName := c.Param("name")
	version := c.Param("version")

	// Find the artifact
	filter := &models.ArtifactFilter{}
	artifacts, _, err := h.artifactRepo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query artifacts"})
		return
	}

	var targetArtifact *models.Artifact
	for _, artifact := range artifacts {
		if artifact.Type == "helm" && artifact.Name == chartName && artifact.Version == version {
			targetArtifact = &artifact
			break
		}
	}

	if targetArtifact == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chart not found"})
		return
	}

	// Get file path from metadata
	var filePath string
	if targetArtifact.Metadata != nil {
		if fp, ok := targetArtifact.Metadata["file_path"].(string); ok {
			filePath = fp
		}
	}

	// Delete file if exists
	if filePath != "" {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete chart file"})
			return
		}
	}

	// Delete artifact from database
	if err := h.artifactRepo.Delete(targetArtifact.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete artifact"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chart deleted successfully"})
}

// Legacy mux.Router methods for backward compatibility
// SetupHelmRoutes sets up Helm chart repository routes (mux version - deprecated)
func (s *Server) SetupHelmRoutes(r *mux.Router) {
	handler := NewHelmHandler(s.config.StoragePath, s.artifactService, s.repositoryService, s.artifactRepo, CreateReplicationMixin(securelogger.NewLogger("helm")))

	// Helm chart repository routes
	helmRouter := r.PathPrefix("/helm").Subrouter()

	// Index file (required by Helm clients)
	helmRouter.HandleFunc("/index.yaml", handler.handleGetIndex).Methods("GET")
	helmRouter.HandleFunc("/{repo}/index.yaml", handler.handleGetIndex).Methods("GET")

	// Chart upload
	helmRouter.HandleFunc("/{repo}/api/charts", handler.handleUploadChart).Methods("POST")
	helmRouter.HandleFunc("/api/charts", handler.handleUploadChart).Methods("POST")

	// Chart download
	helmRouter.HandleFunc("/charts/{filename}", handler.handleDownloadChart).Methods("GET")
	helmRouter.HandleFunc("/{repo}/charts/{filename}", handler.handleDownloadChart).Methods("GET")

	// Chart deletion
	helmRouter.HandleFunc("/api/charts/{name}/{version}", handler.handleDeleteChart).Methods("DELETE")
}

// handleGetIndex returns the index.yaml file listing all available charts (mux version)
func (h *HelmHandler) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	if repoName == "" {
		repoName = "default"
	}

	// Get all Helm repositories
	repos, err := h.repositoryService.List()
	if err != nil {
		http.Error(w, "Failed to list repositories", http.StatusInternalServerError)
		return
	}

	// Find the requested repository or use first Helm repo
	var targetRepo *models.Repository
	for _, repo := range repos {
		if repo.Type == "helm" && (repo.Name == repoName || repoName == "default") {
			targetRepo = &repo
			break
		}
	}

	if targetRepo == nil {
		http.Error(w, "Helm repository not found", http.StatusNotFound)
		return
	}

	// Get all artifacts in this repository
	filter := &models.ArtifactFilter{
		RepositoryID: &targetRepo.ID,
		Types:        []string{"helm"},
		Limit:        1000,
	}
	artifacts, _, err := h.artifactService.List(filter)
	if err != nil {
		http.Error(w, "Failed to list charts", http.StatusInternalServerError)
		return
	}

	// Build index
	index := HelmIndex{
		APIVersion: "v1",
		Generated:  time.Now(),
		Entries:    make(map[string][]HelmIndexEntry),
	}

	for _, artifact := range artifacts {
		// Parse metadata from artifact
		var chartMeta HelmChartMetadata
		if metaBytes, err := json.Marshal(artifact.Metadata); err == nil {
			json.Unmarshal(metaBytes, &chartMeta)
		}

		entry := HelmIndexEntry{
			Name:        artifact.Name,
			Version:     artifact.Version,
			AppVersion:  chartMeta.AppVersion,
			Description: chartMeta.Description,
			Keywords:    chartMeta.Keywords,
			Home:        chartMeta.Home,
			Sources:     chartMeta.Sources,
			Maintainers: chartMeta.Maintainers,
			Icon:        chartMeta.Icon,
			APIVersion:  chartMeta.APIVersion,
			URLs:        []string{fmt.Sprintf("/helm/charts/%s-%s.tgz", artifact.Name, artifact.Version)},
			Created:     artifact.UploadedAt,
			Digest:      artifact.Checksum,
			Annotations: chartMeta.Annotations,
		}

		index.Entries[artifact.Name] = append(index.Entries[artifact.Name], entry)
	}

	// Return as YAML
	w.Header().Set("Content-Type", "application/x-yaml")
	yamlData, err := yaml.Marshal(index)
	if err != nil {
		http.Error(w, "Failed to generate index", http.StatusInternalServerError)
		return
	}

	w.Write(yamlData)
}

// handleUploadChart handles Helm chart upload
func (h *HelmHandler) handleUploadChart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoName := vars["repo"]
	if repoName == "" {
		repoName = "default"
	}

	// Parse multipart form
	err := r.ParseMultipartForm(100 << 20) // 100MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("chart")
	if err != nil {
		http.Error(w, "Chart file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read chart data
	chartData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read chart file", http.StatusInternalServerError)
		return
	}

	// Extract Chart.yaml metadata
	chartMeta, err := h.extractChartMetadata(chartData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid chart: %v", err), http.StatusBadRequest)
		return
	}

	// Get repository
	repos, err := h.repositoryService.List()
	if err != nil {
		http.Error(w, "Failed to list repositories", http.StatusInternalServerError)
		return
	}

	var helmRepo *models.Repository
	for _, repo := range repos {
		if repo.Type == "helm" && (repo.Name == repoName || repoName == "default") {
			helmRepo = &repo
			break
		}
	}

	if helmRepo == nil {
		http.Error(w, "Helm repository not found", http.StatusNotFound)
		return
	}

	// Calculate checksum
	checksum := fmt.Sprintf("sha256:%x", sha256.Sum256(chartData))

	// Create artifact record
	artifact := &models.Artifact{
		TenantID:     helmRepo.TenantID,
		RepositoryID: helmRepo.ID,
		Name:         chartMeta.Name,
		Version:      chartMeta.Version,
		Type:         "helm",
		Size:         int64(len(chartData)),
		Checksum:     checksum,
		Metadata: map[string]interface{}{
			"apiVersion":  chartMeta.APIVersion,
			"appVersion":  chartMeta.AppVersion,
			"description": chartMeta.Description,
			"keywords":    chartMeta.Keywords,
			"home":        chartMeta.Home,
			"sources":     chartMeta.Sources,
			"maintainers": chartMeta.Maintainers,
			"icon":        chartMeta.Icon,
			"filename":    fileHeader.Filename,
		},
	}

	// Save artifact to database
	err = h.artifactService.Create(artifact)
	if err != nil {
		http.Error(w, "Failed to save artifact", http.StatusInternalServerError)
		return
	}

	// Save chart file to storage
	chartPath := filepath.Join(h.storagePath, "helm", helmRepo.TenantID.String(), helmRepo.ID.String(),
		fmt.Sprintf("%s-%s.tgz", chartMeta.Name, chartMeta.Version))

	if err := os.MkdirAll(filepath.Dir(chartPath), 0755); err != nil {
		http.Error(w, "Failed to create storage directory", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(chartPath, chartData, 0644); err != nil {
		http.Error(w, "Failed to save chart file", http.StatusInternalServerError)
		return
	}

	// Trigger replication
	go h.replicateHelmChart(helmRepo.TenantID.String(), helmRepo.ID.String(),
		fmt.Sprintf("%s-%s", chartMeta.Name, chartMeta.Version), chartData)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"saved":   true,
		"name":    chartMeta.Name,
		"version": chartMeta.Version,
		"id":      artifact.ID,
	})
}

// handleDownloadChart handles chart download
func (h *HelmHandler) handleDownloadChart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filename := vars["filename"]

	if !strings.HasSuffix(filename, ".tgz") {
		http.Error(w, "Invalid chart filename", http.StatusBadRequest)
		return
	}

	// Extract name and version from filename (format: name-version.tgz)
	parts := strings.TrimSuffix(filename, ".tgz")
	lastDash := strings.LastIndex(parts, "-")
	if lastDash == -1 {
		http.Error(w, "Invalid chart filename format", http.StatusBadRequest)
		return
	}

	name := parts[:lastDash]
	version := parts[lastDash+1:]

	// Find artifact
	filter := &models.ArtifactFilter{
		Types:  []string{"helm"},
		Search: name,
		Limit:  100,
	}
	artifacts, _, err := h.artifactService.List(filter)
	if err != nil {
		http.Error(w, "Failed to find chart", http.StatusInternalServerError)
		return
	}

	var targetArtifact *models.Artifact
	for _, artifact := range artifacts {
		if artifact.Name == name && artifact.Version == version {
			targetArtifact = &artifact
			break
		}
	}

	if targetArtifact == nil {
		http.NotFound(w, r)
		return
	}

	// Read chart file
	chartPath := filepath.Join(h.storagePath, "helm", targetArtifact.TenantID.String(),
		targetArtifact.RepositoryID.String(), filename)

	chartData, err := os.ReadFile(chartPath)
	if err != nil {
		http.Error(w, "Chart file not found", http.StatusNotFound)
		return
	}

	// Return chart
	w.Header().Set("Content-Type", "application/x-gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Write(chartData)
}

// handleDeleteChart handles chart deletion
func (h *HelmHandler) handleDeleteChart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	version := vars["version"]

	// Find and delete artifact
	filter := &models.ArtifactFilter{
		Types:  []string{"helm"},
		Search: name,
		Limit:  100,
	}
	artifacts, _, err := h.artifactService.List(filter)
	if err != nil {
		http.Error(w, "Failed to find chart", http.StatusInternalServerError)
		return
	}

	var targetArtifact *models.Artifact
	for _, artifact := range artifacts {
		if artifact.Name == name && artifact.Version == version {
			targetArtifact = &artifact
			break
		}
	}

	if targetArtifact == nil {
		http.NotFound(w, r)
		return
	}

	// Delete from database
	err = h.artifactService.Delete(targetArtifact.ID)
	if err != nil {
		http.Error(w, "Failed to delete chart", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// extractChartMetadata extracts Chart.yaml from the chart tarball
func (h *HelmHandler) extractChartMetadata(chartData []byte) (*HelmChartMetadata, error) {
	// Open gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(chartData))
	if err != nil {
		return nil, fmt.Errorf("invalid gzip: %w", err)
	}
	defer gzReader.Close()

	// Open tar reader
	tarReader := tar.NewReader(gzReader)

	// Find Chart.yaml
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar error: %w", err)
		}

		// Chart.yaml is typically at <chart-name>/Chart.yaml
		if strings.HasSuffix(header.Name, "/Chart.yaml") || header.Name == "Chart.yaml" {
			chartYaml, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("failed to read Chart.yaml: %w", err)
			}

			var meta HelmChartMetadata
			if err := yaml.Unmarshal(chartYaml, &meta); err != nil {
				return nil, fmt.Errorf("invalid Chart.yaml: %w", err)
			}

			return &meta, nil
		}
	}

	return nil, fmt.Errorf("Chart.yaml not found in chart")
}

// replicateHelmChart triggers replication for a Helm chart using the replication mixin
func (h *HelmHandler) replicateHelmChart(tenantID, repositoryID, chartID string, data []byte) {
	if h.replicationMixin == nil || !h.replicationMixin.IsEnabled() {
		return
	}

	h.replicationMixin.ReplicateAsync(&replicate.ReplicationRequest{
		TenantID:     tenantID,
		RepositoryID: repositoryID,
		ArtifactID:   chartID,
		ArtifactType: "helm",
		Data:         data,
		Metadata: map[string]string{
			"artifact_type": "chart",
		},
		BucketName: fmt.Sprintf("%s/helm", repositoryID),
		FileName:   fmt.Sprintf("%s.tgz", chartID),
	})
}
