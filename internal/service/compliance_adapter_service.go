package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/securestor/securestor/internal/adapters"
)

// ComplianceAdapterService integrates the database and storage adapters with compliance workflows
type ComplianceAdapterService struct {
	dbAdapter      adapters.DBAdapter
	storageAdapter adapters.StorageAdapter
	logger         *log.Logger
}

// NewComplianceAdapterService creates a new compliance adapter service
func NewComplianceAdapterService(dbAdapter adapters.DBAdapter, storageAdapter adapters.StorageAdapter, logger *log.Logger) *ComplianceAdapterService {
	return &ComplianceAdapterService{
		dbAdapter:      dbAdapter,
		storageAdapter: storageAdapter,
		logger:         logger,
	}
}

// RecordComplianceAudit records a comprehensive compliance audit for an artifact
func (c *ComplianceAdapterService) RecordComplianceAudit(ctx context.Context, artifactID int64, policyID int64, result ComplianceAuditResult) error {
	// Create compliance log entry
	logEntry := &adapters.ComplianceLogEntry{
		ArtifactID:  artifactID,
		PolicyID:    policyID,
		Status:      result.Status,
		Details:     result.Details,
		Violations:  result.Violations,
		Remediation: result.RemediationSteps,
		CheckedAt:   time.Now(),
		NextCheck:   result.NextCheck,
		CreatedBy:   result.AuditorID,
	}

	if err := c.dbAdapter.CreateComplianceLog(ctx, logEntry); err != nil {
		c.logger.Printf("Failed to create compliance log entry: %v", err)
		return err
	}

	// Update overall compliance status
	status := adapters.ComplianceStatus{
		ArtifactID:        artifactID,
		OverallStatus:     result.OverallStatus,
		GDPRStatus:        result.GDPRStatus,
		RetentionStatus:   result.RetentionStatus,
		EncryptionStatus:  result.EncryptionStatus,
		DataLocality:      result.DataLocality,
		LegalHold:         result.LegalHold,
		LastComplianceRun: time.Now(),
		Score:             result.Score,
	}

	if err := c.dbAdapter.UpdateComplianceStatus(ctx, artifactID, status); err != nil {
		c.logger.Printf("Failed to update compliance status: %v", err)
		return err
	}

	c.logger.Printf("Recorded compliance audit for artifact %d with status: %s", artifactID, result.Status)
	return nil
}

// CreateRetentionRecord creates a retention record for an artifact
func (c *ComplianceAdapterService) CreateRetentionRecord(ctx context.Context, artifactID int64, policyID int64, retentionDays int) error {
	now := time.Now()
	expiresAt := now.AddDate(0, 0, retentionDays)

	record := &adapters.RetentionRecord{
		ArtifactID:       artifactID,
		PolicyID:         policyID,
		RetentionDays:    retentionDays,
		CreatedAt:        now,
		ExpiresAt:        expiresAt,
		GracePeriodDays:  30, // Default grace period
		Status:           "active",
		NotificationSent: false,
	}

	if err := c.dbAdapter.CreateRetentionRecord(ctx, record); err != nil {
		c.logger.Printf("Failed to create retention record: %v", err)
		return err
	}

	c.logger.Printf("Created retention record for artifact %d, expires at: %v", artifactID, expiresAt)
	return nil
}

// RecordStorageMetadata records storage metadata for an artifact
func (c *ComplianceAdapterService) RecordStorageMetadata(ctx context.Context, artifactID int64, metadata StorageInfo) error {
	storageMetadata := &adapters.StorageMetadata{
		ArtifactID:       artifactID,
		StorageBackend:   metadata.Backend,
		StorageLocation:  metadata.Location,
		EncryptionStatus: metadata.EncryptionStatus,
		ErasureCoding:    metadata.ErasureConfig,
		Checksum:         metadata.Checksum,
		Size:             metadata.Size,
		CreatedAt:        time.Now(),
	}

	if err := c.dbAdapter.CreateStorageMetadata(ctx, storageMetadata); err != nil {
		c.logger.Printf("Failed to record storage metadata: %v", err)
		return err
	}

	c.logger.Printf("Recorded storage metadata for artifact %d on backend: %s", artifactID, metadata.Backend)
	return nil
}

// PerformIntegrityCheck performs a comprehensive integrity check on an artifact
func (c *ComplianceAdapterService) PerformIntegrityCheck(ctx context.Context, artifactID int64, storageKey string) (*IntegrityCheckResult, error) {
	// Verify integrity using storage adapter
	report, err := c.storageAdapter.VerifyIntegrity(ctx, storageKey)
	if err != nil {
		c.logger.Printf("Failed to verify integrity for artifact %d: %v", artifactID, err)
		return nil, err
	}

	// Convert to our result format
	result := &IntegrityCheckResult{
		ArtifactID:        artifactID,
		StorageKey:        storageKey,
		Status:            string(report.Status),
		ChecksumValid:     report.ChecksumValid,
		ErasureCodeValid:  report.ErasureCodeValid,
		CorruptedShards:   report.CorruptedShards,
		RecoverableShards: report.RecoverableShards,
		RequiredShards:    report.RequiredShards,
		LastVerified:      report.LastVerified,
		Recommendation:    string(report.RepairRecommendation),
	}

	c.logger.Printf("Integrity check for artifact %d: %s", artifactID, result.Status)
	return result, nil
}

// EnforceRetentionPolicies finds and processes artifacts that have expired according to retention policies
func (c *ComplianceAdapterService) EnforceRetentionPolicies(ctx context.Context) error {
	// Get expiring artifacts
	cutoffDate := time.Now()
	records, err := c.dbAdapter.GetExpiringArtifacts(ctx, cutoffDate)
	if err != nil {
		c.logger.Printf("Failed to get expiring artifacts: %v", err)
		return err
	}

	c.logger.Printf("Found %d artifacts for retention enforcement", len(records))

	for _, record := range records {
		// Check if artifact is under legal hold
		legalHolds, err := c.dbAdapter.GetActiveLegalHolds(ctx, record.ArtifactID)
		if err != nil {
			c.logger.Printf("Failed to check legal holds for artifact %d: %v", record.ArtifactID, err)
			continue
		}

		if len(legalHolds) > 0 {
			c.logger.Printf("Artifact %d is under legal hold, skipping deletion", record.ArtifactID)
			continue
		}

		// Get storage metadata to determine storage key
		storageMetadata, err := c.dbAdapter.GetStorageMetadata(ctx, record.ArtifactID)
		if err != nil {
			c.logger.Printf("Failed to get storage metadata for artifact %d: %v", record.ArtifactID, err)
			continue
		}

		// Determine storage key from metadata (this would need to be implemented based on your storage key format)
		storageKey := c.generateStorageKey(record.ArtifactID, storageMetadata)

		// Delete from storage
		if err := c.storageAdapter.Delete(ctx, storageKey); err != nil {
			c.logger.Printf("Failed to delete artifact %d from storage: %v", record.ArtifactID, err)
			continue
		}

		// Update retention record status
		if err := c.dbAdapter.UpdateRetentionStatus(ctx, record.ArtifactID, "deleted"); err != nil {
			c.logger.Printf("Failed to update retention status for artifact %d: %v", record.ArtifactID, err)
		}

		c.logger.Printf("Successfully deleted artifact %d due to retention policy", record.ArtifactID)
	}

	return nil
}

// GetComplianceReports retrieves compliance reports with filtering
func (c *ComplianceAdapterService) GetComplianceReports(ctx context.Context, filters adapters.ComplianceLogFilters) ([]*adapters.ComplianceLogEntry, error) {
	return c.dbAdapter.GetComplianceLogs(ctx, filters)
}

// GetStorageHealth retrieves storage backend health information
func (c *ComplianceAdapterService) GetStorageHealth(ctx context.Context) (*StorageHealthInfo, error) {
	health, err := c.storageAdapter.GetHealth(ctx)
	if err != nil {
		c.logger.Printf("Failed to get storage health: %v", err)
		return nil, err
	}

	stats, err := c.storageAdapter.GetStorageStats(ctx)
	if err != nil {
		c.logger.Printf("Failed to get storage stats: %v", err)
		return nil, err
	}

	return &StorageHealthInfo{
		Backend:            health.BackendType,
		Status:             string(health.Status),
		ResponseTime:       health.ResponseTime,
		AvailableSpace:     health.AvailableSpace,
		UsedSpace:          health.UsedSpace,
		TotalSpace:         health.TotalSpace,
		HealthyShards:      health.HealthyShards,
		DamagedShards:      health.DamagedShards,
		TotalArtifacts:     stats.TotalArtifacts,
		StorageUtilization: stats.StorageUtilization,
		CompressionRatio:   stats.CompressionRatio,
		ErasureOverhead:    stats.ErasureOverhead,
		LastCheck:          health.LastHealthCheck,
		Issues:             health.Issues,
	}, nil
}

// generateStorageKey generates a storage key for an artifact based on its metadata
func (c *ComplianceAdapterService) generateStorageKey(artifactID int64, metadata *adapters.StorageMetadata) string {
	// This would need to be implemented based on your storage key format
	// For now, return a simple format
	return fmt.Sprintf("artifact-%d", artifactID)
}

// Data structures for service inputs/outputs
type ComplianceAuditResult struct {
	Status           string                 `json:"status"`
	Details          string                 `json:"details"`
	Violations       []string               `json:"violations"`
	RemediationSteps map[string]interface{} `json:"remediation_steps"`
	NextCheck        *time.Time             `json:"next_check"`
	AuditorID        string                 `json:"auditor_id"`
	OverallStatus    string                 `json:"overall_status"`
	GDPRStatus       string                 `json:"gdpr_status"`
	RetentionStatus  string                 `json:"retention_status"`
	EncryptionStatus string                 `json:"encryption_status"`
	DataLocality     string                 `json:"data_locality"`
	LegalHold        bool                   `json:"legal_hold"`
	Score            int                    `json:"score"`
}

type StorageInfo struct {
	Backend          string                   `json:"backend"`
	Location         adapters.StorageLocation `json:"location"`
	EncryptionStatus string                   `json:"encryption_status"`
	ErasureConfig    adapters.ErasureConfig   `json:"erasure_config"`
	Checksum         string                   `json:"checksum"`
	Size             int64                    `json:"size"`
}

type IntegrityCheckResult struct {
	ArtifactID        int64     `json:"artifact_id"`
	StorageKey        string    `json:"storage_key"`
	Status            string    `json:"status"`
	ChecksumValid     bool      `json:"checksum_valid"`
	ErasureCodeValid  bool      `json:"erasure_code_valid"`
	CorruptedShards   []int     `json:"corrupted_shards"`
	RecoverableShards int       `json:"recoverable_shards"`
	RequiredShards    int       `json:"required_shards"`
	LastVerified      time.Time `json:"last_verified"`
	Recommendation    string    `json:"recommendation"`
}

type StorageHealthInfo struct {
	Backend            string                 `json:"backend"`
	Status             string                 `json:"status"`
	ResponseTime       time.Duration          `json:"response_time"`
	AvailableSpace     int64                  `json:"available_space"`
	UsedSpace          int64                  `json:"used_space"`
	TotalSpace         int64                  `json:"total_space"`
	HealthyShards      int                    `json:"healthy_shards"`
	DamagedShards      int                    `json:"damaged_shards"`
	TotalArtifacts     int64                  `json:"total_artifacts"`
	StorageUtilization float64                `json:"storage_utilization"`
	CompressionRatio   float64                `json:"compression_ratio"`
	ErasureOverhead    float64                `json:"erasure_overhead"`
	LastCheck          time.Time              `json:"last_check"`
	Issues             []adapters.HealthIssue `json:"issues"`
}
