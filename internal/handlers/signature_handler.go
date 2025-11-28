package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/securestor/securestor/internal/logger"
	"github.com/securestor/securestor/internal/signature"
	"github.com/securestor/securestor/internal/tenant"
)

// SignatureHandler handles signature-related API requests (Gin version)
type SignatureHandler struct {
	db      *sql.DB
	service *signature.Service
	log     *logger.Logger
}

// NewSignatureHandler creates a new signature handler
func NewSignatureHandler(db *sql.DB, log *logger.Logger) *SignatureHandler {
	return &SignatureHandler{
		db:      db,
		service: signature.NewService(db),
		log:     log,
	}
}

// RegisterRoutes registers signature routes (Gin version)
func (h *SignatureHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Artifact signature routes
	router.POST("/artifacts/:id/signatures", h.UploadSignature)
	router.GET("/artifacts/:id/signatures", h.GetArtifactSignatures)
	router.GET("/artifacts/:id/integrity", h.GetArtifactIntegrity)

	// Signature management routes
	router.GET("/signatures/:signature_id", h.GetSignature)
	router.POST("/signatures/:signature_id/verify", h.VerifySignature)

	// Public key management routes
	router.POST("/keys", h.UploadPublicKey)
	router.GET("/keys", h.GetTrustedKeys)
}

// UploadSignature handles uploading a signature for an artifact
func (h *SignatureHandler) UploadSignature(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Parse request body
	var req signature.UploadSignatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	req.ArtifactID = artifactID

	// Get repository ID from artifact
	var repositoryID uuid.UUID
	err = h.db.QueryRowContext(ctx,
		"SELECT repository_id FROM artifacts WHERE id = $1 AND tenant_id = $2",
		artifactID, tenantID).Scan(&repositoryID)
	if err != nil {
		h.log.Error("Failed to get artifact repository", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Get repository signature policy
	policy, err := h.service.GetRepositorySignaturePolicy(ctx, tenantID, repositoryID)
	if err != nil {
		h.log.Error("Failed to get repository policy", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check repository policy"})
		return
	}

	// Check if signature type is enabled
	if !h.isSignatureTypeEnabled(req.SignatureType, policy) {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Signature type %s not enabled for repository", req.SignatureType)})
		return
	}

	// Create signature record
	sig := &signature.ArtifactSignature{
		SignatureID:        uuid.New(),
		TenantID:           tenantID,
		ArtifactID:         artifactID,
		RepositoryID:       repositoryID,
		SignatureType:      req.SignatureType,
		SignatureFormat:    req.SignatureFormat,
		SignatureData:      req.SignatureData,
		SignatureAlgo:      req.SignatureAlgo,
		PublicKey:          req.PublicKey,
		CosignCertificate:  req.Certificate,
		CosignBundle:       req.CosignBundle,
		SigstoreBundle:     req.SigstoreBundle,
		VerificationStatus: signature.VerificationStatusPending,
	}

	// Store signature
	if err := h.service.StoreSignature(ctx, sig); err != nil {
		h.log.Error("Failed to store signature", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store signature"})
		return
	}

	// Verify immediately if requested
	if req.VerifyImmediately && policy.CheckSignatureVerificationRequired() {
		if err := h.verifySignature(ctx, sig); err != nil {
			h.log.Error("Signature verification failed", err)
			// Don't fail the upload, just log the verification failure
		}
	}

	c.JSON(http.StatusCreated, gin.H{
		"signature_id": sig.SignatureID,
		"artifact_id":  artifactID,
		"status":       sig.VerificationStatus,
		"message":      "Signature uploaded successfully",
	})
}

// GetArtifactSignatures retrieves all signatures for an artifact
func (h *SignatureHandler) GetArtifactSignatures(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	signatures, err := h.service.GetArtifactSignatures(ctx, tenantID, artifactID)
	if err != nil {
		h.log.Error("Failed to get signatures", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve signatures"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"artifact_id": artifactID,
		"count":       len(signatures),
		"signatures":  signatures,
	})
}

// GetSignature retrieves a specific signature
func (h *SignatureHandler) GetSignature(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	signatureID, err := uuid.Parse(c.Param("signature_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature ID"})
		return
	}

	sig, err := h.service.GetSignature(ctx, tenantID, signatureID)
	if err != nil {
		h.log.Error("Failed to get signature", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Signature not found"})
		return
	}

	c.JSON(http.StatusOK, sig)
}

// VerifySignature verifies a signature
func (h *SignatureHandler) VerifySignature(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	signatureID, err := uuid.Parse(c.Param("signature_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signature ID"})
		return
	}

	sig, err := h.service.GetSignature(ctx, tenantID, signatureID)
	if err != nil {
		h.log.Error("Failed to get signature", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Signature not found"})
		return
	}

	if err := h.verifySignature(ctx, sig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Verification failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"signature_id": signatureID,
		"verified":     sig.Verified,
		"status":       sig.VerificationStatus,
		"message":      "Signature verification completed",
	})
}

// UploadPublicKey uploads a trusted public key
func (h *SignatureHandler) UploadPublicKey(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var key signature.PublicKey
	if err := c.ShouldBindJSON(&key); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	key.KeyID = uuid.New()
	key.TenantID = tenantID

	// Extract key information based on type
	if key.KeyType == "pgp" {
		extractedKey, err := signature.ExtractPGPKeyInfo(key.PublicKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to parse PGP key: %v", err)})
			return
		}
		key.KeyFingerprint = extractedKey.KeyFingerprint
		key.KeyIDShort = extractedKey.KeyIDShort
		key.KeyAlgorithm = extractedKey.KeyAlgorithm
		key.OwnerEmail = extractedKey.OwnerEmail
		key.OwnerName = extractedKey.OwnerName
	}

	if err := h.service.StorePublicKey(ctx, &key); err != nil {
		h.log.Error("Failed to store public key", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store public key"})
		return
	}

	c.JSON(http.StatusCreated, key)
}

// GetTrustedKeys retrieves trusted public keys
func (h *SignatureHandler) GetTrustedKeys(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var repositoryID *uuid.UUID
	if repoIDStr := c.Query("repository_id"); repoIDStr != "" {
		if repoID, err := uuid.Parse(repoIDStr); err == nil {
			repositoryID = &repoID
		}
	}

	keys, err := h.service.GetTrustedKeys(ctx, tenantID, repositoryID)
	if err != nil {
		h.log.Error("Failed to get trusted keys", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(keys),
		"keys":  keys,
	})
}

// GetArtifactIntegrity retrieves integrity information for an artifact
func (h *SignatureHandler) GetArtifactIntegrity(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, err := tenant.GetTenantID(ctx)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	artifactID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid artifact ID"})
		return
	}

	// Get artifact with hash information
	var artifact struct {
		SHA256Hash        string `db:"sha256_hash"`
		SHA512Hash        string `db:"sha512_hash"`
		HashAlgorithm     string `db:"hash_algorithm"`
		SignatureRequired bool   `db:"signature_required"`
		SignatureVerified bool   `db:"signature_verified"`
		Checksum          string `db:"checksum"`
	}

	err = h.db.QueryRowContext(ctx,
		`SELECT sha256_hash, sha512_hash, hash_algorithm, 
		        signature_required, signature_verified, checksum
		 FROM artifacts WHERE id = $1 AND tenant_id = $2`,
		artifactID, tenantID).Scan(
		&artifact.SHA256Hash, &artifact.SHA512Hash, &artifact.HashAlgorithm,
		&artifact.SignatureRequired, &artifact.SignatureVerified, &artifact.Checksum)
	if err != nil {
		h.log.Error("Failed to get artifact", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Artifact not found"})
		return
	}

	// Get signature count
	var signatureCount int
	err = h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM artifact_signatures 
		 WHERE artifact_id = $1 AND tenant_id = $2`,
		artifactID, tenantID).Scan(&signatureCount)
	if err != nil {
		signatureCount = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"artifact_id":        artifactID,
		"sha256_hash":        artifact.SHA256Hash,
		"sha512_hash":        artifact.SHA512Hash,
		"hash_algorithm":     artifact.HashAlgorithm,
		"checksum":           artifact.Checksum,
		"signature_required": artifact.SignatureRequired,
		"signature_verified": artifact.SignatureVerified,
		"signature_count":    signatureCount,
	})
}

// verifySignature verifies a signature based on its type
func (h *SignatureHandler) verifySignature(ctx context.Context, sig *signature.ArtifactSignature) error {
	var result *signature.VerificationResult
	var err error

	switch sig.SignatureType {
	case signature.SignatureTypeCosign:
		verifier := signature.NewCosignVerifier("")
		// Get artifact digest
		var artifactDigest string
		err := h.db.QueryRowContext(ctx,
			"SELECT checksum FROM artifacts WHERE id = $1", sig.ArtifactID).Scan(&artifactDigest)
		if err != nil {
			return fmt.Errorf("failed to get artifact digest: %w", err)
		}
		result, err = verifier.VerifyCosignSignature(ctx, sig, artifactDigest)

	case signature.SignatureTypePGP:
		verifier := signature.NewPGPVerifier()
		// Load trusted keys
		keys, _ := h.service.GetTrustedKeys(ctx, sig.TenantID, &sig.RepositoryID)
		verifier.AddPublicKeys(keys)

		// PGP verification requires artifact data
		// In production, fetch artifact from storage and verify
		result = &signature.VerificationResult{
			Verified:           false,
			Status:             signature.VerificationStatusPending,
			ErrorMessage:       "PGP verification requires artifact data",
			VerificationMethod: "pgp",
			VerifiedAt:         time.Now(),
		}
		err = nil

	default:
		return fmt.Errorf("unsupported signature type: %s", sig.SignatureType)
	}

	if err != nil {
		// Log verification failure
		h.service.LogVerification(ctx, &signature.VerificationLog{
			TenantID:           sig.TenantID,
			ArtifactID:         sig.ArtifactID,
			SignatureID:        &sig.SignatureID,
			VerificationType:   "manual",
			VerificationResult: "failure",
			VerificationStatus: string(signature.VerificationStatusInvalid),
			ErrorMessage:       err.Error(),
		})
		return err
	}

	// Update signature verification status
	if err := h.service.UpdateSignatureVerification(ctx, sig.SignatureID, result, nil); err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	// Log successful verification
	h.service.LogVerification(ctx, &signature.VerificationLog{
		TenantID:           sig.TenantID,
		ArtifactID:         sig.ArtifactID,
		SignatureID:        &sig.SignatureID,
		VerificationType:   "manual",
		VerificationResult: "success",
		VerificationStatus: string(result.Status),
		VerificationMethod: result.VerificationMethod,
	})

	return nil
}

// isSignatureTypeEnabled checks if a signature type is enabled for the repository
func (h *SignatureHandler) isSignatureTypeEnabled(sigType signature.SignatureType, policy *signature.RepositorySignaturePolicy) bool {
	switch sigType {
	case signature.SignatureTypeCosign:
		return policy.CosignEnabled
	case signature.SignatureTypePGP:
		return policy.PGPEnabled
	case signature.SignatureTypeSigstore:
		return policy.SigstoreEnabled
	default:
		return false
	}
}

// ComputeArtifactHash computes and stores SHA-256/SHA-512 hashes for an artifact
func ComputeArtifactHash(reader io.Reader, db *sql.DB, tenantID, artifactID uuid.UUID) error {
	hashes, err := signature.ComputeHashes(reader)
	if err != nil {
		return fmt.Errorf("failed to compute hashes: %w", err)
	}

	_, err = db.Exec(
		`UPDATE artifacts 
		 SET sha256_hash = $1, sha512_hash = $2, hash_algorithm = $3, updated_at = CURRENT_TIMESTAMP
		 WHERE tenant_id = $4 AND id = $5`,
		hashes.SHA256, hashes.SHA512, hashes.Algorithm, tenantID, artifactID)

	return err
}
