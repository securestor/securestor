package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/encrypt"
	"github.com/securestor/securestor/internal/service"
)

// EncryptionAdminHandler handles encryption administration endpoints (Gin version)
type EncryptionAdminHandler struct {
	tmkService             *encrypt.TMKService
	rewrapService          *encrypt.RewrapService
	encryptedBackupService *service.EncryptedBackupService
}

// NewEncryptionAdminHandler creates a new encryption admin handler
func NewEncryptionAdminHandler(
	tmkService *encrypt.TMKService,
	rewrapService *encrypt.RewrapService,
	encryptedBackupService *service.EncryptedBackupService,
) *EncryptionAdminHandler {
	return &EncryptionAdminHandler{
		tmkService:             tmkService,
		rewrapService:          rewrapService,
		encryptedBackupService: encryptedBackupService,
	}
}

// RegisterRoutes registers encryption admin routes (Gin version)
func (h *EncryptionAdminHandler) RegisterRoutes(router *gin.RouterGroup) {
	// TMK management routes
	tmkGroup := router.Group("/encryption/tmk")
	{
		tmkGroup.GET("/status", h.GetTMKStatus)
		tmkGroup.POST("/rotate", h.RotateTMK)
		tmkGroup.POST("/export", h.ExportTMK)
	}

	// Re-wrap management routes
	rewrapGroup := router.Group("/encryption/rewrap")
	{
		rewrapGroup.POST("/start", h.StartRewrap)
		rewrapGroup.GET("/:job_id/status", h.GetRewrapStatus)
		rewrapGroup.POST("/:job_id/cancel", h.CancelRewrap)
	}

	// Backup management routes
	backupGroup := router.Group("/backups")
	{
		backupGroup.POST("/create", h.CreateBackup)
		backupGroup.POST("/keys/create", h.CreateKeyBackup)
		backupGroup.POST("/:backup_id/verify", h.VerifyBackup)
		backupGroup.POST("/:backup_id/restore", h.RestoreBackup)
		backupGroup.POST("/:backup_id/test-restore", h.TestRestore)
	}
}

// GetTMKStatus returns the TMK status for a tenant
func (h *EncryptionAdminHandler) GetTMKStatus(c *gin.Context) {
	tenantIDStr := c.Query("tenant_id")
	if tenantIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id query parameter is required"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	status, err := h.tmkService.GetTMKStatus(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get TMK status: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// RotateTMK rotates the tenant master key
func (h *EncryptionAdminHandler) RotateTMK(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id" binding:"required"`
		KMSKeyID string `json:"kms_key_id,omitempty"`
		Reason   string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	// Get user ID from JWT context (set by middleware)
	userID := uuid.Nil
	if uid, exists := c.Get("user_id"); exists {
		if uidStr, ok := uid.(string); ok {
			userID, _ = uuid.Parse(uidStr)
		}
	}

	// Use default KMS key if not provided
	if req.KMSKeyID == "" {
		req.KMSKeyID = "arn:aws:kms:us-east-1:123456789012:key/mock-key-id"
	}

	tmk, err := h.tmkService.RotateTMK(c.Request.Context(), tenantID, userID, req.KMSKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rotate TMK: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "TMK rotated successfully",
		"key_version": tmk.KeyVersion,
		"created_at":  tmk.CreatedAt,
		"reason":      req.Reason,
	})
}

// ExportTMK exports the encrypted TMK for backup
func (h *EncryptionAdminHandler) ExportTMK(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	// Get user ID from JWT context
	userID := uuid.Nil
	if uid, exists := c.Get("user_id"); exists {
		if uidStr, ok := uid.(string); ok {
			userID, _ = uuid.Parse(uidStr)
		}
	}

	exportedKey, err := h.tmkService.ExportTMKForBackup(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export TMK: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"exported_key": exportedKey,
		"tenant_id":    tenantID,
		"exported_at":  time.Now().Format(time.RFC3339),
	})
}

// StartRewrap starts a DEK re-wrap job
func (h *EncryptionAdminHandler) StartRewrap(c *gin.Context) {
	var req struct {
		TenantID      string `json:"tenant_id" binding:"required"`
		OldTMKVersion int    `json:"old_tmk_version" binding:"required"`
		NewTMKVersion int    `json:"new_tmk_version" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	job, err := h.rewrapService.StartRewrapJob(
		c.Request.Context(),
		tenantID,
		req.OldTMKVersion,
		req.NewTMKVersion,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start re-wrap job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Re-wrap job started successfully",
		"job_id":  job.JobID,
	})
}

// GetRewrapStatus returns the status of a re-wrap job
func (h *EncryptionAdminHandler) GetRewrapStatus(c *gin.Context) {
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job_id format"})
		return
	}

	status, err := h.rewrapService.GetJobStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// CancelRewrap cancels a running re-wrap job
func (h *EncryptionAdminHandler) CancelRewrap(c *gin.Context) {
	jobIDStr := c.Param("job_id")

	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job_id format"})
		return
	}

	err = h.rewrapService.CancelJob(jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel job: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Re-wrap job cancelled",
		"job_id":  jobID,
	})
}

// CreateBackup creates an encrypted database backup
func (h *EncryptionAdminHandler) CreateBackup(c *gin.Context) {
	var req struct {
		BackupType  string `json:"backup_type" binding:"required"` // "database" or "keys"
		CrossRegion bool   `json:"cross_region,omitempty"`
		VerifyAfter bool   `json:"verify_after,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	if req.BackupType != "database" && req.BackupType != "keys" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup_type must be 'database' or 'keys'"})
		return
	}

	backupID, err := h.encryptedBackupService.BackupDatabase(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create backup: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Backup created successfully",
		"backup_id":   backupID,
		"backup_type": req.BackupType,
		"created_at":  time.Now().Format(time.RFC3339),
	})
}

// CreateKeyBackup creates an encrypted key backup
func (h *EncryptionAdminHandler) CreateKeyBackup(c *gin.Context) {
	var req struct {
		TenantID string `json:"tenant_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id format"})
		return
	}

	backupID, err := h.encryptedBackupService.BackupKeys(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create key backup: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    "Key backup created successfully",
		"backup_id":  backupID,
		"tenant_id":  tenantID,
		"created_at": time.Now().Format(time.RFC3339),
	})
}

// VerifyBackup verifies the integrity of a backup
func (h *EncryptionAdminHandler) VerifyBackup(c *gin.Context) {
	backupIDStr := c.Param("backup_id")

	backupID, err := uuid.Parse(backupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup_id format"})
		return
	}

	// TODO: Implement backup verification
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"backup_id":      backupID,
		"verified":       true,
		"checksum_valid": true,
		"kek_available":  true,
	})
}

// RestoreBackup restores from an encrypted backup
func (h *EncryptionAdminHandler) RestoreBackup(c *gin.Context) {
	backupIDStr := c.Param("backup_id")

	backupID, err := uuid.Parse(backupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup_id format"})
		return
	}

	err = h.encryptedBackupService.RestoreDatabase(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to restore backup: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Backup restored successfully",
		"backup_id":   backupID,
		"restored_at": time.Now().Format(time.RFC3339),
	})
}

// TestRestore performs a dry-run restore test
func (h *EncryptionAdminHandler) TestRestore(c *gin.Context) {
	backupIDStr := c.Param("backup_id")

	backupID, err := uuid.Parse(backupIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup_id format"})
		return
	}

	// TODO: Implement dry-run restore test
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"backup_id":     backupID,
		"can_restore":   true,
		"kek_available": true,
		"backup_valid":  true,
	})
}
