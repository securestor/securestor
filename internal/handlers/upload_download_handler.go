package handlers

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/encrypt"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/service"
	"github.com/securestor/securestor/internal/storage"
)

const MaxUploadSize = 10 << 30    // 10 GB
const StreamBufferSize = 32 << 20 // 32 MB streaming buffer for memory efficiency

type UploadDownloadHandler struct {
	artifactService   *service.ArtifactService
	repositoryService *service.RepositoryService
	encryptionService *encrypt.EncryptionService
	tmkService        *encrypt.TMKService
	policyService     interface{} // Generic policy service to avoid circular dependency
	auditLogService   *service.AuditLogService
	scanService       *service.ScanService
	blobStorage       *storage.BlobStorage
	config            *UploadDownloadConfig
}

type UploadDownloadConfig struct {
	EncryptionEnabled   bool
	EncryptionEnforced  bool
	EncryptionMode      string
	AWSKMSKeyID         string
	ErasureDataShards   int
	ErasureParityShards int
}

func NewUploadDownloadHandler(
	artifactService *service.ArtifactService,
	repositoryService *service.RepositoryService,
	encryptionService *encrypt.EncryptionService,
	tmkService *encrypt.TMKService,
	policyService interface{},
	auditLogService *service.AuditLogService,
	scanService *service.ScanService,
	blobStorage *storage.BlobStorage,
	config *UploadDownloadConfig,
) *UploadDownloadHandler {
	return &UploadDownloadHandler{
		artifactService:   artifactService,
		repositoryService: repositoryService,
		encryptionService: encryptionService,
		tmkService:        tmkService,
		policyService:     policyService,
		auditLogService:   auditLogService,
		scanService:       scanService,
		blobStorage:       blobStorage,
		config:            config,
	}
}

// RegisterRoutes registers upload/download routes
func (h *UploadDownloadHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/artifacts/upload", h.UploadArtifact)
	rg.GET("/artifacts/:id/download", h.DownloadArtifact)
}

// UploadArtifact handles artifact upload with streaming and encryption
func (h *UploadDownloadHandler) UploadArtifact(c *gin.Context) {
	// Get tenant ID from context
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

	// Parse multipart form with 64MB max memory
	if err := c.Request.ParseMultipartForm(64 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large or invalid form data"})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	// Get form data
	name := c.Request.FormValue("name")
	version := c.Request.FormValue("version")
	repositoryID := c.Request.FormValue("repository_id")
	license := c.Request.FormValue("license")
	tagsJSON := c.Request.FormValue("tags")
	metadataJSON := c.Request.FormValue("metadata")
	artifactType := c.Request.FormValue("artifact_type")

	// Validate required fields
	if name == "" || version == "" || repositoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name, version, and repository are required"})
		return
	}

	// Parse repository ID
	repoID, err := uuid.Parse(repositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID format"})
		return
	}

	// Get repository to determine type and tenant
	repo, err := h.repositoryService.GetByID(repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	// Parse tags
	var tags []string
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tags JSON format"})
			return
		}
	}

	// Parse metadata
	metadata := make(map[string]interface{})
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":         "Invalid metadata JSON format",
				"details":       err.Error(),
				"received_json": metadataJSON,
			})
			return
		}
	}

	// Stream file with checksum calculation
	hasher := sha256.New()
	var fileSize int64
	var checksum string
	var fileData []byte

	if repo.Type == "docker" && (artifactType == "manifest" || artifactType == "blob") {
		// Docker requires full data - use efficient buffered reading
		fileData, err = io.ReadAll(io.LimitReader(file, MaxUploadSize))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
			return
		}
		fileSize = int64(len(fileData))
		hasher.Write(fileData)
		checksum = "sha256:" + hex.EncodeToString(hasher.Sum(nil))

		// Handle Docker-specific uploads
		if artifactType == "manifest" || artifactType == "blob" {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "Docker manifest/blob uploads not yet implemented in Gin handler"})
			return
		}
	} else {
		// Streaming optimization for non-Docker artifacts
		var buf bytes.Buffer
		multiWriter := io.MultiWriter(&buf, hasher)

		// Read with buffering
		bufferedReader := bufio.NewReaderSize(file, StreamBufferSize)
		written, err := io.Copy(multiWriter, bufferedReader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process file"})
			return
		}

		fileSize = written
		checksum = "sha256:" + hex.EncodeToString(hasher.Sum(nil))
		fileData = buf.Bytes()
	}

	// Generate artifact ID
	artifactID := fmt.Sprintf("%s-%s-%d", name, version, time.Now().Unix())

	ctx := context.Background()

	// Enterprise-Grade Encryption
	var encryptedData *encrypt.EncryptedData
	var encryptionEnabled bool

	if h.config.EncryptionEnabled {
		// Get or create TMK for tenant
		_, err := h.tmkService.GetActiveTMK(ctx, tenantID)
		if err != nil {
			// Create TMK if it doesn't exist
			// Try to get user ID from context, use nil if not available (system-created)
			var createdBy uuid.UUID
			if userIDVal, exists := c.Get("user_id"); exists {
				if userID, ok := userIDVal.(uuid.UUID); ok {
					createdBy = userID
				}
			}
			_, err := h.tmkService.CreateTMK(ctx, tenantID, createdBy, h.config.AWSKMSKeyID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize encryption for tenant: " + err.Error()})
				return
			}
		}

		// Encrypt artifact using envelope encryption
		encryptedData, err = h.encryptionService.EncryptArtifact(tenantID.String(), fileData, h.config.AWSKMSKeyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt artifact: " + err.Error()})
			return
		}

		// Replace fileData with encrypted ciphertext
		fileData = encryptedData.Ciphertext
		encryptionEnabled = true
	}

	// Upload to blob storage with erasure coding
	if err := h.blobStorage.UploadArtifact(ctx, tenantID.String(), repoID.String(), artifactID, bytes.NewReader(fileData), int64(len(fileData))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload artifact: " + err.Error()})
		return
	}

	// Create artifact record
	licensePtr := (*string)(nil)
	if license != "" {
		licensePtr = &license
	}

	artifact := &models.Artifact{
		Name:         name,
		Version:      version,
		Type:         repo.Type,
		RepositoryID: repoID,
		TenantID:     tenantID,
		Size:         fileSize,
		Checksum:     checksum,
		UploadedBy:   nil, // TODO: Get from auth context
		License:      licensePtr,
		Metadata:     metadata,
		Tags:         tags,
	}

	// Add storage metadata
	if artifact.Metadata == nil {
		artifact.Metadata = make(map[string]interface{})
	}
	artifact.Metadata["storage_type"] = "erasure_coded"
	artifact.Metadata["storage_id"] = artifactID
	artifact.Metadata["original_filename"] = header.Filename
	artifact.Metadata["file_extension"] = filepath.Ext(header.Filename)
	artifact.Metadata["data_shards"] = h.config.ErasureDataShards
	artifact.Metadata["parity_shards"] = h.config.ErasureParityShards

	// Add encryption metadata if enabled
	if encryptionEnabled && encryptedData != nil {
		artifact.Encrypted = true
		artifact.EncryptionVersion = encryptedData.KeyVersion
		artifact.EncryptedDEK = encryptedData.EncryptedDEK
		artifact.EncryptionAlgorithm = encryptedData.Algorithm

		// Store encryption-specific metadata in EncryptionMetadata field
		artifact.EncryptionMetadata = map[string]interface{}{
			"nonce":        base64.StdEncoding.EncodeToString(encryptedData.Nonce),
			"encrypted_at": encryptedData.EncryptedAt,
			"tenant_id":    encryptedData.TenantID,
			"key_version":  encryptedData.KeyVersion,
			"algorithm":    encryptedData.Algorithm,
		}

		// Also add to general Metadata for backward compatibility and display
		artifact.Metadata["encryption_nonce"] = base64.StdEncoding.EncodeToString(encryptedData.Nonce)
		artifact.Metadata["encrypted_at"] = encryptedData.EncryptedAt
		artifact.Metadata["encryption_tenant_id"] = encryptedData.TenantID
	}

	// Log metadata before saving for debugging
	metadataDebug, _ := json.Marshal(artifact.Metadata)
	fmt.Printf("[DEBUG] Artifact metadata before save: %s\n", string(metadataDebug))

	// Policy evaluation placeholder - skipped to avoid circular dependency
	// In production, proper policy integration would be implemented
	_ = h.policyService

	if err := h.artifactService.Create(artifact); err != nil {
		// Rollback: delete uploaded shards
		h.blobStorage.DeleteArtifact(ctx, tenantID.String(), repoID.String(), artifactID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create artifact record: " + err.Error()})
		return
	}

	// Trigger automatic security scan for the uploaded artifact
	go h.triggerAutomaticScan(tenantID, artifact)

	// Prepare response
	response := gin.H{
		"id":             artifact.ID,
		"name":           artifact.Name,
		"version":        artifact.Version,
		"type":           artifact.Type,
		"repository":     repo.Name,
		"size":           artifact.Size,
		"size_formatted": formatSize(artifact.Size),
		"checksum":       artifact.Checksum,
		"uploaded_by": func() string {
			if artifact.UploadedBy != nil {
				return *artifact.UploadedBy
			}
			return ""
		}(),
		"uploaded_at":  artifact.UploadedAt.Format(time.RFC3339),
		"storage_type": "erasure_coded",
		"shard_count":  h.config.ErasureDataShards + h.config.ErasureParityShards,
		"message":      "Artifact deployed successfully with erasure coding",
	}

	c.JSON(http.StatusCreated, response)
}

// DownloadArtifact handles artifact download with decryption
func (h *UploadDownloadHandler) DownloadArtifact(c *gin.Context) {
	// Parse artifact ID
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact from database
	artifact, err := h.artifactService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Check repository public access
	var publicAccess bool
	// This would need database access - for now, assume private
	publicAccess = false

	// If repository is not public, verify tenant access
	if !publicAccess {
		tenantIDVal, exists := c.Get("tenant_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required for private repository"})
			return
		}
		tenantID, ok := tenantIDVal.(uuid.UUID)
		if !ok || tenantID != artifact.TenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this artifact"})
			return
		}
	}

	// Get storage ID from metadata
	storageID, ok := artifact.Metadata["storage_id"].(string)
	if !ok || storageID == "" {
		// For legacy artifacts, generate fallback ID
		storageID = fmt.Sprintf("%s-%s-%d", artifact.Name, artifact.Version, artifact.CreatedAt.Unix())
	}

	// Download from blob storage
	ctx := context.Background()
	data, err := h.blobStorage.DownloadArtifact(ctx, artifact.TenantID.String(), artifact.RepositoryID.String(), storageID)
	if err != nil {
		if strings.Contains(err.Error(), "failed to read metadata") || strings.Contains(err.Error(), "no such file or directory") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artifact data not found. This artifact may need to be re-uploaded."})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download artifact: " + err.Error()})
		}
		return
	}

	// Decrypt artifact if encrypted
	if artifact.Encrypted {
		// Validate encryption metadata
		if len(artifact.EncryptedDEK) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Artifact encryption metadata is corrupted"})
			return
		}

		if artifact.EncryptionMetadata == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Artifact encryption metadata is missing"})
			return
		}

		// Extract nonce from metadata
		nonceInterface, ok := artifact.EncryptionMetadata["nonce"]
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Encryption nonce not found"})
			return
		}

		// Convert nonce to []byte
		var nonce []byte
		switch v := nonceInterface.(type) {
		case []byte:
			nonce = v
		case []interface{}:
			nonce = make([]byte, len(v))
			for i, val := range v {
				if byteVal, ok := val.(float64); ok {
					nonce[i] = byte(byteVal)
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid nonce format"})
					return
				}
			}
		case string:
			// Decode base64-encoded nonce string
			nonce, err = base64.StdEncoding.DecodeString(v)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid nonce encoding"})
				return
			}
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid nonce data type"})
			return
		}

		// Reconstruct EncryptedData for decryption
		encryptedData := &encrypt.EncryptedData{
			Ciphertext:   data,
			EncryptedDEK: artifact.EncryptedDEK,
			Nonce:        nonce,
			Algorithm:    artifact.EncryptionAlgorithm,
			KeyVersion:   artifact.EncryptionVersion,
			TenantID:     artifact.TenantID.String(),
		}

		// Decrypt artifact
		plaintext, err := h.encryptionService.DecryptArtifact(encryptedData, h.config.AWSKMSKeyID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt artifact: " + err.Error()})
			return
		}

		// Replace encrypted data with plaintext
		data = plaintext

		// Audit log: Track plaintext access
		if h.auditLogService != nil {
			userID := c.GetString("user_id")
			if userID == "" {
				userID = "anonymous"
			}
			sessionID := c.GetString("session_id")
			ipAddress := c.GetHeader("X-Forwarded-For")
			if ipAddress == "" {
				ipAddress = c.GetHeader("X-Real-IP")
			}
			if ipAddress == "" {
				if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
					ipAddress = host
				} else {
					ipAddress = c.Request.RemoteAddr
				}
			}

			go func() {
				auditCtx := context.Background()
				auditEntry := &service.AuditLogEntry{
					TenantID:     artifact.TenantID.String(),
					EventType:    "artifact_decryption",
					ResourceID:   id.String(),
					ResourceType: "artifact",
					UserID:       &userID,
					Action:       "decrypt_download",
					IPAddress:    ipAddress,
					SessionID:    sessionID,
					Success:      true,
					Metadata: map[string]interface{}{
						"artifact_name":      artifact.Name,
						"artifact_version":   artifact.Version,
						"encryption_version": artifact.EncryptionVersion,
						"encryption_algo":    artifact.EncryptionAlgorithm,
						"plaintext_size":     len(data),
						"purpose":            "download",
					},
					Timestamp: time.Now(),
				}
				if err := h.auditLogService.LogEvent(auditCtx, auditEntry); err != nil {
					// Log error but don't fail the download
				}
			}()
		}
	}

	// Increment download counter
	h.artifactService.IncrementDownloads(id)

	// Log download event
	if h.auditLogService != nil {
		userID := c.GetString("user_id")
		if userID == "" {
			userID = "anonymous"
		}
		sessionID := c.GetString("session_id")
		ipAddress := c.GetHeader("X-Forwarded-For")
		if ipAddress == "" {
			ipAddress = c.GetHeader("X-Real-IP")
		}
		if ipAddress == "" {
			if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
				ipAddress = host
			} else {
				ipAddress = c.Request.RemoteAddr
			}
		}
		userAgent := c.GetHeader("User-Agent")

		go func() {
			ctx := context.Background()
			h.auditLogService.LogDownload(ctx, artifact.TenantID.String(), id, userID, ipAddress, userAgent, sessionID)
		}()
	}

	// Determine filename
	filename := ""
	if originalFilename, ok := artifact.Metadata["original_filename"].(string); ok && originalFilename != "" {
		filename = originalFilename
	} else {
		// Generate filename from artifact name and version
		filename = fmt.Sprintf("%s-%s", artifact.Name, artifact.Version)

		// Try to get stored file extension
		if ext, ok := artifact.Metadata["file_extension"].(string); ok && ext != "" {
			if len(ext) > 1 && ext[0] == '.' && !containsOnlyDigits(ext[1:]) {
				filename += ext
			}
		} else {
			// Add extension based on artifact type
			switch artifact.Type {
			case "npm":
				filename += ".tgz"
			case "maven":
				filename += ".jar"
			case "pypi":
				filename += ".whl"
			case "docker":
				filename += ".json"
			case "helm":
				filename += ".tgz"
			default:
				// Detect file type from magic bytes
				if len(data) > 4 {
					if data[0] == 0x1f && data[1] == 0x8b {
						filename += ".tar.gz"
					} else if data[0] == 0x50 && data[1] == 0x4b && data[2] == 0x03 && data[3] == 0x04 {
						filename += ".zip"
					} else if data[0] == '{' || data[0] == '[' {
						filename += ".json"
					} else {
						filename += ".bin"
					}
				} else {
					filename += ".bin"
				}
			}
		}
	}

	// Detect content type
	contentType := "application/octet-stream"
	ext := filepath.Ext(filename)
	switch ext {
	case ".tar", ".tar.gz", ".tgz":
		contentType = "application/gzip"
	case ".zip":
		contentType = "application/zip"
	case ".json":
		contentType = "application/json"
	case ".jar":
		contentType = "application/java-archive"
	case ".war":
		contentType = "application/x-webarchive"
	case ".whl":
		contentType = "application/zip"
	case ".exe":
		contentType = "application/x-msdownload"
	case ".deb":
		contentType = "application/x-debian-package"
	case ".rpm":
		contentType = "application/x-rpm"
	default:
		// Use artifact type as hint
		switch artifact.Type {
		case "docker":
			contentType = "application/vnd.docker.distribution.manifest.v2+json"
		case "npm":
			contentType = "application/gzip"
		case "maven":
			contentType = "application/java-archive"
		case "pypi":
			contentType = "application/zip"
		}
	}

	// Set response headers
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Header("X-Checksum", artifact.Checksum)
	c.Header("X-Artifact-Type", artifact.Type)

	c.Data(http.StatusOK, contentType, data)
}

// Helper functions

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func containsOnlyDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// triggerAutomaticScan initiates a security scan for a newly uploaded artifact
func (h *UploadDownloadHandler) triggerAutomaticScan(tenantID uuid.UUID, artifact *models.Artifact) {
	// Skip scan if scanService is not available
	if h.scanService == nil {
		return
	}

	// Check if a scan is already in progress for this artifact
	if existingScan, err := h.scanService.GetActiveScan(artifact.ID); err == nil && existingScan != nil {
		fmt.Printf("[INFO] Scan already in progress for artifact %s, skipping automatic scan\n", artifact.ID)
		return
	}

	// Create a full security scan with all scan types enabled
	scan := &models.SecurityScan{
		TenantID:          tenantID,
		ArtifactID:        artifact.ID,
		Status:            "initiated",
		ScanType:          "full",
		Priority:          "normal",
		VulnerabilityScan: true,
		MalwareScan:       true,
		LicenseScan:       true,
		DependencyScan:    true,
		InitiatedBy:       nil, // System-initiated scan
		StartedAt:         time.Now(),
	}

	// Create scan record
	if err := h.scanService.CreateScan(scan); err != nil {
		fmt.Printf("[ERROR] Failed to create automatic scan for artifact %s: %v\n", artifact.ID, err)
		return
	}

	fmt.Printf("[INFO] Automatic security scan initiated for artifact %s:%s (scan ID: %s)\n",
		artifact.Name, artifact.Version, scan.ID)

	// Start scan asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := h.scanService.ScanArtifact(ctx, artifact, scan); err != nil {
			fmt.Printf("[ERROR] Automatic scan failed for artifact %s: %v\n", artifact.ID, err)
			scan.Status = "failed"
			errMsg := err.Error()
			scan.ErrorMessage = &errMsg
			now := time.Now()
			scan.CompletedAt = &now
			h.scanService.UpdateScan(scan)
		} else {
			fmt.Printf("[INFO] Automatic scan completed successfully for artifact %s:%s\n",
				artifact.Name, artifact.Version)
		}
	}()
}
