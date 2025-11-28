package api

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/securestor/securestor/internal/service"
)

// BackupServiceManager provides backup operations (to be initialized in Server)
var backupServiceInstance *service.BackupService

// handleCreateBackup triggers an immediate backup
func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if backupServiceInstance == nil {
		respondWithError(w, http.StatusInternalServerError, "Backup service not initialized")
		return
	}

	ctx := r.Context()
	if err := backupServiceInstance.CreateBackup(ctx); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Backup created successfully",
	})
}

// handleGetBackupHistory returns backup history
func (s *Server) handleGetBackupHistory(w http.ResponseWriter, r *http.Request) {
	if backupServiceInstance == nil {
		respondWithError(w, http.StatusInternalServerError, "Backup service not initialized")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	history := backupServiceInstance.GetBackupHistory(limit)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"backups": history,
		"count":   len(history),
	})
}

// handleGetBackupStats returns backup statistics
func (s *Server) handleGetBackupStats(w http.ResponseWriter, r *http.Request) {
	if backupServiceInstance == nil {
		respondWithError(w, http.StatusInternalServerError, "Backup service not initialized")
		return
	}

	stats := backupServiceInstance.GetBackupStats()
	respondWithJSON(w, http.StatusOK, stats)
}

// handleVerifyBackup verifies a specific backup
func (s *Server) handleVerifyBackup(w http.ResponseWriter, r *http.Request) {
	if backupServiceInstance == nil {
		respondWithError(w, http.StatusInternalServerError, "Backup service not initialized")
		return
	}

	backupFile := r.URL.Query().Get("file")
	if backupFile == "" {
		respondWithError(w, http.StatusBadRequest, "file parameter required")
		return
	}

	ctx := r.Context()
	backupPath := filepath.Join(s.config.StoragePath, "backups", filepath.Base(backupFile))

	if err := backupServiceInstance.VerifyBackup(ctx, backupPath); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"status": "verified",
		"file":   backupFile,
	})
}

// handleRestoreBackup initiates a backup restore
func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if backupServiceInstance == nil {
		respondWithError(w, http.StatusInternalServerError, "Backup service not initialized")
		return
	}

	backupFile := r.URL.Query().Get("file")
	if backupFile == "" {
		respondWithError(w, http.StatusBadRequest, "file parameter required")
		return
	}

	ctx := r.Context()
	backupPath := filepath.Join(s.config.StoragePath, "backups", filepath.Base(backupFile))

	if err := backupServiceInstance.RestoreBackup(ctx, backupPath); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondWithJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":  "restore_initiated",
		"file":    backupFile,
		"message": "Restore operation in progress",
	})
}

// SetBackupServiceInstance sets the backup service instance
func SetBackupServiceInstance(bs *service.BackupService) {
	backupServiceInstance = bs
}
