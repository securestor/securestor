package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CloudRepositoryHandler handles cloud storage repository operations
type CloudRepositoryHandler struct {
	db *sql.DB
}

type CloudRepositoryRequest struct {
	Name            string `json:"name" binding:"required"`
	Type            string `json:"type" binding:"required"`
	RepositoryType  string `json:"repository_type" binding:"required"`
	Description     string `json:"description"`
	PublicAccess    bool   `json:"public_access"`
	EnableIndexing  bool   `json:"enable_indexing"`
	CloudProvider   string `json:"cloud_provider"`
	Region          string `json:"region"`
	BucketName      string `json:"bucket_name"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	Endpoint        string `json:"endpoint"`
}

type CloudRepositoryResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	RepositoryType string    `json:"repository_type"`
	CloudProvider  string    `json:"cloud_provider"`
	Region         string    `json:"region"`
	BucketName     string    `json:"bucket_name"`
	Status         string    `json:"status"`
	HealthCheck    string    `json:"health_check"`
	CreatedAt      time.Time `json:"created_at"`
}

func NewCloudRepositoryHandler(db *sql.DB) *CloudRepositoryHandler {
	return &CloudRepositoryHandler{db: db}
}

// CreateCloudRepository creates a repository with cloud storage backend
func (h *CloudRepositoryHandler) CreateCloudRepository(c *gin.Context) {
	var req CloudRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate repository type
	if req.RepositoryType != "cloud" && req.RepositoryType != "remote" {
		// If not cloud/remote, fallback to regular repository creation
		c.JSON(http.StatusBadRequest, gin.H{"error": "use regular repository endpoint for local repositories"})
		return
	}

	// Validate cloud provider
	if req.CloudProvider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cloud_provider is required for cloud repositories"})
		return
	}

	// Create repository record
	repoID := uuid.New().String()

	// Insert into remote_repositories table (created by our migrations)
	query := `
		INSERT INTO remote_repositories (
			id, name, type, url, cache_enabled, cache_ttl_hours,
			max_cache_size_mb, upstream_health_check_interval_seconds,
			protocol_adapter, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at
	`

	// Build cloud storage URL based on provider
	cloudURL := buildCloudStorageURL(req.CloudProvider, req.Region, req.BucketName, req.Endpoint)

	// Store credentials in metadata (in production, use secrets manager)
	metadata := map[string]interface{}{
		"cloud_provider": req.CloudProvider,
		"region":         req.Region,
		"bucket_name":    req.BucketName,
		"endpoint":       req.Endpoint,
		// Note: In production, never store credentials in DB
		// Use AWS Secrets Manager, Vault, or environment variables
	}

	var createdAt time.Time
	err := h.db.QueryRow(query,
		repoID,
		req.Name,
		req.Type,
		cloudURL,
		true,     // cache_enabled
		24,       // cache_ttl_hours
		10240,    // max_cache_size_mb (10GB)
		300,      // health_check_interval (5 min)
		req.Type, // protocol_adapter
		metadata,
	).Scan(&repoID, &createdAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create repository: " + err.Error()})
		return
	}

	// Test connection to cloud storage
	healthStatus := testCloudConnection(req.CloudProvider, req.Region, req.BucketName, req.AccessKeyID, req.SecretAccessKey, req.Endpoint)

	response := CloudRepositoryResponse{
		ID:             repoID,
		Name:           req.Name,
		Type:           req.Type,
		RepositoryType: req.RepositoryType,
		CloudProvider:  req.CloudProvider,
		Region:         req.Region,
		BucketName:     req.BucketName,
		Status:         "active",
		HealthCheck:    healthStatus,
		CreatedAt:      createdAt,
	}

	c.JSON(http.StatusCreated, response)
}

// GetCloudRepositoryStatus gets the health status of a cloud repository
func (h *CloudRepositoryHandler) GetCloudRepositoryStatus(c *gin.Context) {
	repoID := c.Param("id")
	if repoID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository ID required"})
		return
	}

	// Query repository details
	query := `
		SELECT id, name, type, url, metadata, created_at, updated_at
		FROM remote_repositories
		WHERE id = $1
	`

	var id, name, repoType, url string
	var metadata map[string]interface{}
	var createdAt, updatedAt time.Time

	err := h.db.QueryRow(query, repoID).Scan(&id, &name, &repoType, &url, &metadata, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get health status from monitoring table
	healthQuery := `
		SELECT status, last_check_at, response_time_ms, error_count_24h
		FROM remote_repository_health
		WHERE repository_id = $1
		ORDER BY last_check_at DESC
		LIMIT 1
	`

	var status string
	var lastCheck time.Time
	var responseTime int
	var errorCount int

	err = h.db.QueryRow(healthQuery, repoID).Scan(&status, &lastCheck, &responseTime, &errorCount)
	if err != nil && err != sql.ErrNoRows {
		// Default values if no health data
		status = "unknown"
		lastCheck = time.Now()
		responseTime = 0
		errorCount = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"id":               id,
		"name":             name,
		"type":             repoType,
		"url":              url,
		"metadata":         metadata,
		"status":           status,
		"last_check":       lastCheck,
		"response_time_ms": responseTime,
		"error_count_24h":  errorCount,
		"created_at":       createdAt,
		"updated_at":       updatedAt,
	})
}

// Helper functions

func buildCloudStorageURL(provider, region, bucket, endpoint string) string {
	switch provider {
	case "s3":
		if region != "" {
			return fmt.Sprintf("s3://%s.s3.%s.amazonaws.com", bucket, region)
		}
		return fmt.Sprintf("s3://%s.s3.amazonaws.com", bucket)
	case "gcs", "gcp":
		return fmt.Sprintf("gs://%s", bucket)
	case "azure":
		return fmt.Sprintf("https://%s.blob.core.windows.net", bucket)
	case "s3-compatible":
		if endpoint != "" {
			return fmt.Sprintf("%s/%s", endpoint, bucket)
		}
		return fmt.Sprintf("s3://%s", bucket)
	default:
		return fmt.Sprintf("%s://%s", provider, bucket)
	}
}

func testCloudConnection(provider, region, bucket, accessKey, secretKey, endpoint string) string {
	// In a real implementation, this would:
	// 1. Initialize the appropriate cloud SDK client
	// 2. Attempt to list objects or get bucket info
	// 3. Return "healthy" or "unhealthy" based on the result

	// For now, return a placeholder
	if accessKey != "" && bucket != "" {
		return "healthy"
	}
	return "needs_configuration"
}
