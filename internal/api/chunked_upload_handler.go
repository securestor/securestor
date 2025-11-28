package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
	"github.com/securestor/securestor/internal/tenant"
)

// ChunkedUploadSession tracks multi-part upload progress for enterprise-scale resumable uploads
type ChunkedUploadSession struct {
	UploadID     string                 `json:"upload_id"`
	TenantID     uuid.UUID              `json:"tenant_id"`
	RepositoryID uuid.UUID              `json:"repository_id"`
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	TotalSize    int64                  `json:"total_size"`
	TotalChunks  int                    `json:"total_chunks"`
	ChunkSize    int64                  `json:"chunk_size"`
	Uploaded     map[int]bool           `json:"uploaded_chunks"` // Track which chunks received
	Checksums    map[int]string         `json:"checksums"`       // Per-chunk checksums
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	mu           sync.RWMutex           // Protect concurrent chunk uploads
}

// In-memory session storage (in production, use Redis for distributed HA)
var (
	uploadSessions   = make(map[string]*ChunkedUploadSession)
	uploadSessionsMu sync.RWMutex
)

// InitiateChunkedUploadRequest for starting a resumable upload
type InitiateChunkedUploadRequest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	RepositoryID string                 `json:"repository_id"`
	TotalSize    int64                  `json:"total_size"`
	ChunkSize    int64                  `json:"chunk_size,omitempty"` // Optional, default 32MB
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// InitiateChunkedUploadResponse returns upload session details
type InitiateChunkedUploadResponse struct {
	UploadID    string    `json:"upload_id"`
	ChunkSize   int64     `json:"chunk_size"`
	TotalChunks int       `json:"total_chunks"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// ChunkUploadResponse tracks progress after each chunk upload
type ChunkUploadResponse struct {
	ChunkNumber     int     `json:"chunk_number"`
	Received        bool    `json:"received"`
	TotalReceived   int     `json:"total_received"`
	TotalChunks     int     `json:"total_chunks"`
	ProgressPercent float64 `json:"progress_percent"`
}

// CompleteChunkedUploadRequest finalizes the upload
type CompleteChunkedUploadRequest struct {
	FinalChecksum string `json:"final_checksum,omitempty"` // Optional verification
}

const (
	DefaultChunkSize        = 32 << 20       // 32 MB per chunk
	MaxChunkSize            = 128 << 20      // 128 MB max
	ChunkedUploadExpiration = 24 * time.Hour // 24 hours to complete
)

// handleInitiateChunkedUpload creates a new upload session for resumable uploads
// POST /api/v1/artifacts/upload/initiate
func (s *Server) handleInitiateChunkedUpload(c *gin.Context) {
	var req InitiateChunkedUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate required fields
	if req.Name == "" || req.Version == "" || req.RepositoryID == "" || req.TotalSize <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name, version, repository_id, and total_size are required"})
		return
	}

	// Parse repository ID
	repoID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid repository ID format"})
		return
	}

	// Verify repository exists
	repo, err := s.repositoryService.GetByID(repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found"})
		return
	}

	// Get tenant ID from context
	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No tenant context found"})
		return
	}

	// Set chunk size (default 32MB, max 128MB)
	chunkSize := req.ChunkSize
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
	}
	if chunkSize > MaxChunkSize {
		chunkSize = MaxChunkSize
	}

	// Calculate total chunks
	totalChunks := int((req.TotalSize + chunkSize - 1) / chunkSize)

	// Create upload session
	uploadID := uuid.New().String()
	session := &ChunkedUploadSession{
		UploadID:     uploadID,
		TenantID:     tenantID,
		RepositoryID: repoID,
		Name:         req.Name,
		Version:      req.Version,
		TotalSize:    req.TotalSize,
		TotalChunks:  totalChunks,
		ChunkSize:    chunkSize,
		Uploaded:     make(map[int]bool),
		Checksums:    make(map[int]string),
		Metadata:     req.Metadata,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(ChunkedUploadExpiration),
	}

	// Store session (in production, use Redis with TTL)
	uploadSessionsMu.Lock()
	uploadSessions[uploadID] = session
	uploadSessionsMu.Unlock()

	s.logger.Printf("[CHUNKED_UPLOAD] Initiated upload session %s for %s:%s in repo %s (tenant %s) - %d chunks of %d bytes",
		uploadID, req.Name, req.Version, repo.Name, tenantID, totalChunks, chunkSize)

	// Return session details
	c.JSON(http.StatusCreated, &InitiateChunkedUploadResponse{
		UploadID:    uploadID,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
		ExpiresAt:   session.ExpiresAt,
	})
}

// handleUploadChunk handles individual chunk upload
// POST /api/v1/artifacts/upload/{upload_id}/parts?chunk={chunk_number}
func (s *Server) handleUploadChunk(c *gin.Context) {
	uploadID := c.Param("upload_id")

	// Parse chunk number from query params
	chunkNumStr := c.Query("chunk")
	if chunkNumStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chunk number is required"})
		return
	}

	chunkNum, err := strconv.Atoi(chunkNumStr)
	if err != nil || chunkNum < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chunk number"})
		return
	}

	// Get upload session
	uploadSessionsMu.RLock()
	session, exists := uploadSessions[uploadID]
	uploadSessionsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload session not found or expired"})
		return
	}

	// Check if session expired
	if time.Now().After(session.ExpiresAt) {
		uploadSessionsMu.Lock()
		delete(uploadSessions, uploadID)
		uploadSessionsMu.Unlock()
		c.JSON(http.StatusGone, gin.H{"error": "Upload session expired"})
		return
	}

	// Validate chunk number
	if chunkNum >= session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid chunk number (max: %d)", session.TotalChunks-1)})
		return
	}

	// Get tenant ID from context and verify it matches session
	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil || tenantID != session.TenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tenant mismatch"})
		return
	}

	// Read chunk data with size limit
	maxChunkSize := session.ChunkSize
	if int64(chunkNum) == (session.TotalSize / session.ChunkSize) {
		// Last chunk might be smaller
		maxChunkSize = session.TotalSize % session.ChunkSize
		if maxChunkSize == 0 {
			maxChunkSize = session.ChunkSize
		}
	}

	chunkData, err := io.ReadAll(io.LimitReader(c.Request.Body, maxChunkSize+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read chunk data"})
		return
	}

	if int64(len(chunkData)) > maxChunkSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chunk size exceeds limit"})
		return
	}

	// Calculate chunk checksum for integrity verification
	hasher := sha256.New()
	hasher.Write(chunkData)
	chunkChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Store chunk data (in production, stream directly to blob storage)
	// For now, store in temp location with chunk metadata
	ctx := context.Background()
	chunkStoragePath := fmt.Sprintf("%s/chunk-%d", uploadID, chunkNum)

	// Upload chunk to blob storage
	if err := s.blobStorage.UploadArtifact(ctx,
		session.TenantID.String(),
		session.RepositoryID.String(),
		chunkStoragePath,
		bytes.NewReader(chunkData),
		int64(len(chunkData))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store chunk"})
		return
	}

	// Mark chunk as uploaded
	session.mu.Lock()
	session.Uploaded[chunkNum] = true
	session.Checksums[chunkNum] = chunkChecksum
	uploadedCount := len(session.Uploaded)
	session.mu.Unlock()

	progress := float64(uploadedCount) / float64(session.TotalChunks) * 100

	s.logger.Printf("[CHUNKED_UPLOAD] Received chunk %d/%d for upload %s (%.1f%% complete)",
		chunkNum+1, session.TotalChunks, uploadID, progress)

	// Return progress
	c.JSON(http.StatusOK, &ChunkUploadResponse{
		ChunkNumber:     chunkNum,
		Received:        true,
		TotalReceived:   uploadedCount,
		TotalChunks:     session.TotalChunks,
		ProgressPercent: progress,
	})
}

// handleCompleteChunkedUpload finalizes the upload by assembling chunks
// POST /api/v1/artifacts/upload/{upload_id}/complete
func (s *Server) handleCompleteChunkedUpload(c *gin.Context) {
	uploadID := c.Param("upload_id")

	// Get upload session
	uploadSessionsMu.Lock()
	session, exists := uploadSessions[uploadID]
	if exists {
		delete(uploadSessions, uploadID) // Remove from active sessions
	}
	uploadSessionsMu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload session not found"})
		return
	}

	// Get tenant ID from context and verify
	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil || tenantID != session.TenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tenant mismatch"})
		return
	}

	// Verify all chunks received
	session.mu.RLock()
	uploadedCount := len(session.Uploaded)
	session.mu.RUnlock()

	if uploadedCount != session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Incomplete upload: %d/%d chunks received", uploadedCount, session.TotalChunks)})
		return
	}

	s.logger.Printf("[CHUNKED_UPLOAD] Completing upload %s - assembling %d chunks", uploadID, session.TotalChunks)

	// Assemble chunks into final artifact
	ctx := context.Background()
	var assembledData bytes.Buffer
	globalHasher := sha256.New()

	for i := 0; i < session.TotalChunks; i++ {
		chunkStoragePath := fmt.Sprintf("%s/chunk-%d", uploadID, i)

		// Download chunk from blob storage (returns []byte)
		chunkData, err := s.blobStorage.DownloadArtifact(ctx,
			session.TenantID.String(),
			session.RepositoryID.String(),
			chunkStoragePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to retrieve chunk %d", i)})
			return
		}

		// Write chunk to assembled data and calculate global checksum
		if _, err := assembledData.Write(chunkData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assemble chunks"})
			return
		}
		if _, err := globalHasher.Write(chunkData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate checksum"})
			return
		}

		// Clean up chunk
		s.blobStorage.DeleteArtifact(ctx, session.TenantID.String(), session.RepositoryID.String(), chunkStoragePath)
	}

	finalChecksum := "sha256:" + hex.EncodeToString(globalHasher.Sum(nil))
	finalData := assembledData.Bytes()

	s.logger.Printf("[CHUNKED_UPLOAD] Assembled artifact %s:%s - %d bytes, checksum: %s",
		session.Name, session.Version, len(finalData), finalChecksum)

	// Create final artifact
	artifactID := fmt.Sprintf("%s-%s-%d", session.Name, session.Version, time.Now().Unix())

	// Upload final artifact to blob storage
	if err := s.blobStorage.UploadArtifact(ctx,
		session.TenantID.String(),
		session.RepositoryID.String(),
		artifactID,
		bytes.NewReader(finalData),
		int64(len(finalData))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store final artifact"})
		return
	}

	// Get repository info
	repo, _ := s.repositoryService.GetByID(session.RepositoryID)

	// Create artifact record
	artifact := &models.Artifact{
		Name:         session.Name,
		Version:      session.Version,
		Type:         repo.Type,
		RepositoryID: session.RepositoryID,
		TenantID:     session.TenantID,
		Size:         int64(len(finalData)),
		Checksum:     finalChecksum,
		UploadedBy:   nil,
		Metadata:     session.Metadata,
	}

	if artifact.Metadata == nil {
		artifact.Metadata = make(map[string]interface{})
	}
	artifact.Metadata["upload_type"] = "chunked"
	artifact.Metadata["total_chunks"] = session.TotalChunks
	artifact.Metadata["chunk_size"] = session.ChunkSize
	artifact.Metadata["storage_id"] = artifactID

	if err := s.artifactService.Create(artifact); err != nil {
		s.blobStorage.DeleteArtifact(ctx, session.TenantID.String(), session.RepositoryID.String(), artifactID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create artifact record"})
		return
	}

	// Initialize security scans asynchronously
	go s.initializeArtifactScans(artifact.ID.String())

	s.logger.Printf("[CHUNKED_UPLOAD] ✅ Successfully completed chunked upload %s for artifact %s",
		uploadID, artifact.ID)

	// Return artifact details
	c.JSON(http.StatusCreated, artifact)
}

// handleGetUploadProgress returns current upload progress
// GET /api/v1/artifacts/upload/{upload_id}/progress
func (s *Server) handleGetUploadProgress(c *gin.Context) {
	uploadID := c.Param("upload_id")

	uploadSessionsMu.RLock()
	session, exists := uploadSessions[uploadID]
	uploadSessionsMu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Upload session not found"})
		return
	}

	// Get tenant ID and verify
	tenantID, err := tenant.GetTenantID(c.Request.Context())
	if err != nil || tenantID != session.TenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Tenant mismatch"})
		return
	}

	session.mu.RLock()
	uploadedCount := len(session.Uploaded)
	uploadedChunks := make([]int, 0, uploadedCount)
	for chunkNum := range session.Uploaded {
		uploadedChunks = append(uploadedChunks, chunkNum)
	}
	session.mu.RUnlock()

	progress := float64(uploadedCount) / float64(session.TotalChunks) * 100

	c.JSON(http.StatusOK, gin.H{
		"upload_id":        session.UploadID,
		"total_chunks":     session.TotalChunks,
		"uploaded_chunks":  uploadedCount,
		"progress_percent": progress,
		"uploaded_list":    uploadedChunks,
		"expires_at":       session.ExpiresAt,
	})
}
