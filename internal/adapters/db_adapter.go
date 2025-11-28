package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/securestor/securestor/internal/models"
)

// DBAdapter provides a unified interface for database operations across compliance, audit, and storage metadata
type DBAdapter interface {
	// Compliance operations
	CreateComplianceLog(ctx context.Context, log *ComplianceLogEntry) error
	GetComplianceLogs(ctx context.Context, filters ComplianceLogFilters) ([]*ComplianceLogEntry, error)
	UpdateComplianceStatus(ctx context.Context, artifactID int64, status ComplianceStatus) error

	// Retention metadata
	CreateRetentionRecord(ctx context.Context, record *RetentionRecord) error
	GetExpiringArtifacts(ctx context.Context, cutoffDate time.Time) ([]*RetentionRecord, error)
	UpdateRetentionStatus(ctx context.Context, artifactID int64, status string) error

	// Audit operations
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
	GetAuditLogs(ctx context.Context, filters AuditLogFilters) ([]*models.AuditLog, error)

	// Legal hold operations
	CreateLegalHold(ctx context.Context, hold *models.LegalHold) error
	GetActiveLegalHolds(ctx context.Context, artifactID int64) ([]*models.LegalHold, error)
	UpdateLegalHoldStatus(ctx context.Context, holdID int64, status string) error

	// Data locality operations
	CreateDataLocalityRecord(ctx context.Context, record *models.DataLocality) error
	GetDataLocalityRequirements(ctx context.Context, artifactID int64) (*models.DataLocality, error)
	ValidateDataLocality(ctx context.Context, artifactID int64, region string) (bool, error)

	// Storage metadata
	CreateStorageMetadata(ctx context.Context, metadata *StorageMetadata) error
	GetStorageMetadata(ctx context.Context, artifactID int64) (*StorageMetadata, error)
	UpdateStorageLocation(ctx context.Context, artifactID int64, location StorageLocation) error

	// Health and monitoring
	GetDatabaseHealth(ctx context.Context) (*DatabaseHealth, error)
	GetStorageStats(ctx context.Context) (*StorageStats, error)
}

// ComplianceLogEntry represents a compliance audit entry
type ComplianceLogEntry struct {
	ID          int64                  `json:"id"`
	ArtifactID  int64                  `json:"artifact_id"`
	PolicyID    int64                  `json:"policy_id"`
	Status      string                 `json:"status"` // compliant, non_compliant, warning, pending
	Details     string                 `json:"details"`
	Violations  []string               `json:"violations"`
	Remediation map[string]interface{} `json:"remediation"`
	CheckedAt   time.Time              `json:"checked_at"`
	NextCheck   *time.Time             `json:"next_check"`
	CreatedBy   string                 `json:"created_by"`
}

// ComplianceStatus represents overall compliance status
type ComplianceStatus struct {
	ArtifactID        int64     `json:"artifact_id"`
	OverallStatus     string    `json:"overall_status"`
	GDPRStatus        string    `json:"gdpr_status"`
	RetentionStatus   string    `json:"retention_status"`
	EncryptionStatus  string    `json:"encryption_status"`
	DataLocality      string    `json:"data_locality"`
	LegalHold         bool      `json:"legal_hold"`
	LastComplianceRun time.Time `json:"last_compliance_run"`
	Score             int       `json:"score"`
}

// RetentionRecord tracks artifact retention requirements
type RetentionRecord struct {
	ID               int64     `json:"id"`
	ArtifactID       int64     `json:"artifact_id"`
	PolicyID         int64     `json:"policy_id"`
	RetentionDays    int       `json:"retention_days"`
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	GracePeriodDays  int       `json:"grace_period_days"`
	Status           string    `json:"status"` // active, expiring, expired, deleted
	NotificationSent bool      `json:"notification_sent"`
}

// StorageMetadata tracks where and how artifacts are stored
type StorageMetadata struct {
	ID               int64           `json:"id"`
	ArtifactID       int64           `json:"artifact_id"`
	StorageBackend   string          `json:"storage_backend"` // local, s3, gcp, azure, minio
	StorageLocation  StorageLocation `json:"storage_location"`
	EncryptionStatus string          `json:"encryption_status"`
	ErasureCoding    ErasureConfig   `json:"erasure_coding"`
	Checksum         string          `json:"checksum"`
	Size             int64           `json:"size"`
	CreatedAt        time.Time       `json:"created_at"`
	LastVerified     *time.Time      `json:"last_verified"`
}

// StorageLocation represents where data is physically stored
type StorageLocation struct {
	Primary   LocationInfo   `json:"primary"`
	Replicas  []LocationInfo `json:"replicas,omitempty"`
	Region    string         `json:"region"`
	DataClass string         `json:"data_class"` // hot, warm, cold, archive
}

// LocationInfo contains specific storage location details
type LocationInfo struct {
	Backend  string            `json:"backend"`  // local, s3, gcp, azure, minio
	Path     string            `json:"path"`     // file path or object key
	Endpoint string            `json:"endpoint"` // for cloud providers
	Bucket   string            `json:"bucket"`   // bucket/container name
	Region   string            `json:"region"`   // storage region
	Metadata map[string]string `json:"metadata"` // additional metadata
}

// ErasureConfig represents erasure coding configuration
type ErasureConfig struct {
	Enabled      bool `json:"enabled"`
	DataShards   int  `json:"data_shards"`
	ParityShards int  `json:"parity_shards"`
}

// Filter types
type ComplianceLogFilters struct {
	ArtifactID *int64
	PolicyID   *int64
	Status     *string
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int
	Offset     int
}

type AuditLogFilters struct {
	ResourceType *string
	ResourceID   *string
	UserID       *string
	Action       *string
	DateFrom     *time.Time
	DateTo       *time.Time
	Limit        int
	Offset       int
}

// Health monitoring types
type DatabaseHealth struct {
	Status            string        `json:"status"`
	ActiveConnections int           `json:"active_connections"`
	ResponseTime      time.Duration `json:"response_time"`
	LastBackup        *time.Time    `json:"last_backup"`
	DiskUsage         float64       `json:"disk_usage_percent"`
}

type StorageStats struct {
	TotalArtifacts     int64   `json:"total_artifacts"`
	TotalSize          int64   `json:"total_size_bytes"`
	StorageUtilization float64 `json:"storage_utilization_percent"`
	CompressionRatio   float64 `json:"compression_ratio"`
	ErasureOverhead    float64 `json:"erasure_overhead_percent"`
	HealthyShards      int     `json:"healthy_shards"`
	DamagedShards      int     `json:"damaged_shards"`
}

// PostgreSQLAdapter implements DBAdapter for PostgreSQL
type PostgreSQLAdapter struct {
	db     *sql.DB
	logger *log.Logger
}

// NewPostgreSQLAdapter creates a new PostgreSQL database adapter
func NewPostgreSQLAdapter(db *sql.DB, logger *log.Logger) DBAdapter {
	return &PostgreSQLAdapter{
		db:     db,
		logger: logger,
	}
}

// Compliance operations implementation
func (p *PostgreSQLAdapter) CreateComplianceLog(ctx context.Context, log *ComplianceLogEntry) error {
	violationsJSON, _ := json.Marshal(log.Violations)
	remediationJSON, _ := json.Marshal(log.Remediation)

	query := `
		INSERT INTO compliance_log_entries (
			artifact_id, policy_id, status, details, violations, 
			remediation, checked_at, next_check, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return p.db.QueryRowContext(ctx, query,
		log.ArtifactID, log.PolicyID, log.Status, log.Details,
		violationsJSON, remediationJSON, log.CheckedAt, log.NextCheck, log.CreatedBy,
	).Scan(&log.ID)
}

func (p *PostgreSQLAdapter) GetComplianceLogs(ctx context.Context, filters ComplianceLogFilters) ([]*ComplianceLogEntry, error) {
	query := `
		SELECT id, artifact_id, policy_id, status, details, violations,
			   remediation, checked_at, next_check, created_by
		FROM compliance_log_entries
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if filters.ArtifactID != nil {
		argCount++
		query += fmt.Sprintf(" AND artifact_id = $%d", argCount)
		args = append(args, *filters.ArtifactID)
	}

	query += " ORDER BY checked_at DESC"

	if filters.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*ComplianceLogEntry
	for rows.Next() {
		log := &ComplianceLogEntry{}
		var violationsJSON, remediationJSON []byte

		err := rows.Scan(
			&log.ID, &log.ArtifactID, &log.PolicyID, &log.Status,
			&log.Details, &violationsJSON, &remediationJSON,
			&log.CheckedAt, &log.NextCheck, &log.CreatedBy,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(violationsJSON, &log.Violations)
		json.Unmarshal(remediationJSON, &log.Remediation)

		logs = append(logs, log)
	}

	return logs, nil
}

func (p *PostgreSQLAdapter) UpdateComplianceStatus(ctx context.Context, artifactID int64, status ComplianceStatus) error {
	query := `
		INSERT INTO artifact_compliance_status (
			artifact_id, overall_status, gdpr_status, retention_status,
			encryption_status, data_locality, legal_hold, last_compliance_run, score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (artifact_id) DO UPDATE SET
			overall_status = EXCLUDED.overall_status,
			gdpr_status = EXCLUDED.gdpr_status,
			retention_status = EXCLUDED.retention_status,
			encryption_status = EXCLUDED.encryption_status,
			data_locality = EXCLUDED.data_locality,
			legal_hold = EXCLUDED.legal_hold,
			last_compliance_run = EXCLUDED.last_compliance_run,
			score = EXCLUDED.score,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := p.db.ExecContext(ctx, query,
		artifactID, status.OverallStatus, status.GDPRStatus,
		status.RetentionStatus, status.EncryptionStatus, status.DataLocality,
		status.LegalHold, status.LastComplianceRun, status.Score,
	)

	return err
}

// Retention operations implementation
func (p *PostgreSQLAdapter) CreateRetentionRecord(ctx context.Context, record *RetentionRecord) error {
	query := `
		INSERT INTO retention_records (
			artifact_id, policy_id, retention_days, created_at, 
			expires_at, grace_period_days, status, notification_sent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	return p.db.QueryRowContext(ctx, query,
		record.ArtifactID, record.PolicyID, record.RetentionDays,
		record.CreatedAt, record.ExpiresAt, record.GracePeriodDays,
		record.Status, record.NotificationSent,
	).Scan(&record.ID)
}

func (p *PostgreSQLAdapter) GetExpiringArtifacts(ctx context.Context, cutoffDate time.Time) ([]*RetentionRecord, error) {
	query := `
		SELECT id, artifact_id, policy_id, retention_days, created_at,
			   expires_at, grace_period_days, status, notification_sent
		FROM retention_records
		WHERE expires_at <= $1 AND status = 'active'
		ORDER BY expires_at
	`

	rows, err := p.db.QueryContext(ctx, query, cutoffDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*RetentionRecord
	for rows.Next() {
		record := &RetentionRecord{}
		err := rows.Scan(
			&record.ID, &record.ArtifactID, &record.PolicyID, &record.RetentionDays,
			&record.CreatedAt, &record.ExpiresAt, &record.GracePeriodDays,
			&record.Status, &record.NotificationSent,
		)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

func (p *PostgreSQLAdapter) UpdateRetentionStatus(ctx context.Context, artifactID int64, status string) error {
	query := `UPDATE retention_records SET status = $1 WHERE artifact_id = $2`
	_, err := p.db.ExecContext(ctx, query, status, artifactID)
	return err
}

// Audit operations implementation
func (p *PostgreSQLAdapter) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			event_type, resource_id, resource_type, user_id, action,
			old_value, new_value, ip_address, user_agent, success,
			error_msg, timestamp, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

	return p.db.QueryRowContext(ctx, query,
		log.EventType, log.ResourceID, log.ResourceType, log.UserID, log.Action,
		log.OldValue, log.NewValue, log.IPAddress, log.UserAgent, log.Success,
		log.ErrorMsg, log.Timestamp, log.Metadata,
	).Scan(&log.ID)
}

func (p *PostgreSQLAdapter) GetAuditLogs(ctx context.Context, filters AuditLogFilters) ([]*models.AuditLog, error) {
	query := `
		SELECT id, event_type, resource_id, resource_type, user_id, action,
			   COALESCE(old_value, '') as old_value, 
			   COALESCE(new_value, '') as new_value, 
			   COALESCE(ip_address::text, '') as ip_address, 
			   COALESCE(user_agent, '') as user_agent, success,
			   COALESCE(error_msg, '') as error_msg, timestamp, 
			   COALESCE(metadata::text, '{}') as metadata
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if filters.ResourceType != nil {
		argCount++
		query += fmt.Sprintf(" AND resource_type = $%d", argCount)
		args = append(args, *filters.ResourceType)
	}

	query += " ORDER BY timestamp DESC"

	if filters.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filters.Limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}

		err := rows.Scan(
			&log.ID, &log.EventType, &log.ResourceID, &log.ResourceType,
			&log.UserID, &log.Action, &log.OldValue, &log.NewValue,
			&log.IPAddress, &log.UserAgent, &log.Success,
			&log.ErrorMsg, &log.Timestamp, &log.Metadata,
		)
		if err != nil {
			continue
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// Legal hold operations implementation
func (p *PostgreSQLAdapter) CreateLegalHold(ctx context.Context, hold *models.LegalHold) error {
	query := `
		INSERT INTO legal_holds (
			artifact_id, case_number, reason, start_date, end_date,
			status, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return p.db.QueryRowContext(ctx, query,
		hold.ArtifactID, hold.CaseNumber, hold.Reason, hold.StartDate,
		hold.EndDate, hold.Status, hold.CreatedBy, hold.CreatedAt, hold.UpdatedAt,
	).Scan(&hold.ID)
}

func (p *PostgreSQLAdapter) GetActiveLegalHolds(ctx context.Context, artifactID int64) ([]*models.LegalHold, error) {
	query := `
		SELECT id, artifact_id, case_number, reason, start_date, end_date,
			   status, created_by, created_at, updated_at
		FROM legal_holds
		WHERE artifact_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, artifactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holds []*models.LegalHold
	for rows.Next() {
		hold := &models.LegalHold{}
		err := rows.Scan(
			&hold.ID, &hold.ArtifactID, &hold.CaseNumber, &hold.Reason,
			&hold.StartDate, &hold.EndDate, &hold.Status, &hold.CreatedBy,
			&hold.CreatedAt, &hold.UpdatedAt,
		)
		if err != nil {
			continue
		}
		holds = append(holds, hold)
	}

	return holds, nil
}

func (p *PostgreSQLAdapter) UpdateLegalHoldStatus(ctx context.Context, holdID int64, status string) error {
	query := `UPDATE legal_holds SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err := p.db.ExecContext(ctx, query, status, holdID)
	return err
}

// Data locality operations implementation
func (p *PostgreSQLAdapter) CreateDataLocalityRecord(ctx context.Context, record *models.DataLocality) error {
	query := `
		INSERT INTO data_locality (
			artifact_id, required_region, current_region, compliant, last_checked
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	return p.db.QueryRowContext(ctx, query,
		record.ArtifactID, record.RequiredRegion, record.CurrentRegion,
		record.Compliant, record.LastChecked,
	).Scan(&record.ID)
}

func (p *PostgreSQLAdapter) GetDataLocalityRequirements(ctx context.Context, artifactID int64) (*models.DataLocality, error) {
	query := `
		SELECT id, artifact_id, required_region, current_region, compliant, last_checked
		FROM data_locality
		WHERE artifact_id = $1
		ORDER BY last_checked DESC
		LIMIT 1
	`

	locality := &models.DataLocality{}
	err := p.db.QueryRowContext(ctx, query, artifactID).Scan(
		&locality.ID, &locality.ArtifactID, &locality.RequiredRegion,
		&locality.CurrentRegion, &locality.Compliant, &locality.LastChecked,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return locality, err
}

func (p *PostgreSQLAdapter) ValidateDataLocality(ctx context.Context, artifactID int64, region string) (bool, error) {
	query := `
		SELECT required_region FROM data_locality
		WHERE artifact_id = $1 AND required_region != 'GLOBAL'
		ORDER BY created_at DESC
		LIMIT 1
	`

	var requiredRegion string
	err := p.db.QueryRowContext(ctx, query, artifactID).Scan(&requiredRegion)

	if err == sql.ErrNoRows {
		return true, nil // No specific requirement = compliant
	}

	if err != nil {
		return false, err
	}

	return requiredRegion == region, nil
}

// Storage metadata operations implementation
func (p *PostgreSQLAdapter) CreateStorageMetadata(ctx context.Context, metadata *StorageMetadata) error {
	locationJSON, _ := json.Marshal(metadata.StorageLocation)
	erasureJSON, _ := json.Marshal(metadata.ErasureCoding)

	query := `
		INSERT INTO storage_metadata (
			artifact_id, storage_backend, storage_location, encryption_status,
			erasure_coding, checksum, size, created_at, last_verified
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return p.db.QueryRowContext(ctx, query,
		metadata.ArtifactID, metadata.StorageBackend, locationJSON,
		metadata.EncryptionStatus, erasureJSON, metadata.Checksum,
		metadata.Size, metadata.CreatedAt, metadata.LastVerified,
	).Scan(&metadata.ID)
}

func (p *PostgreSQLAdapter) GetStorageMetadata(ctx context.Context, artifactID int64) (*StorageMetadata, error) {
	query := `
		SELECT id, artifact_id, storage_backend, storage_location, encryption_status,
			   erasure_coding, checksum, size, created_at, last_verified
		FROM storage_metadata
		WHERE artifact_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	metadata := &StorageMetadata{}
	var locationJSON, erasureJSON []byte

	err := p.db.QueryRowContext(ctx, query, artifactID).Scan(
		&metadata.ID, &metadata.ArtifactID, &metadata.StorageBackend,
		&locationJSON, &metadata.EncryptionStatus, &erasureJSON,
		&metadata.Checksum, &metadata.Size, &metadata.CreatedAt,
		&metadata.LastVerified,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	json.Unmarshal(locationJSON, &metadata.StorageLocation)
	json.Unmarshal(erasureJSON, &metadata.ErasureCoding)

	return metadata, nil
}

func (p *PostgreSQLAdapter) UpdateStorageLocation(ctx context.Context, artifactID int64, location StorageLocation) error {
	locationJSON, _ := json.Marshal(location)

	query := `
		UPDATE storage_metadata 
		SET storage_location = $1, last_verified = CURRENT_TIMESTAMP
		WHERE artifact_id = $2
	`

	_, err := p.db.ExecContext(ctx, query, locationJSON, artifactID)
	return err
}

// Health and monitoring implementation
func (p *PostgreSQLAdapter) GetDatabaseHealth(ctx context.Context) (*DatabaseHealth, error) {
	health := &DatabaseHealth{}

	// Check connection
	start := time.Now()
	err := p.db.PingContext(ctx)
	if err != nil {
		health.Status = "unhealthy"
		return health, err
	}
	health.ResponseTime = time.Since(start)
	health.Status = "healthy"

	return health, nil
}

func (p *PostgreSQLAdapter) GetStorageStats(ctx context.Context) (*StorageStats, error) {
	stats := &StorageStats{}

	// Total artifacts
	query := `SELECT COUNT(*) FROM artifacts`
	p.db.QueryRowContext(ctx, query).Scan(&stats.TotalArtifacts)

	// Total size
	query = `SELECT COALESCE(SUM(size), 0) FROM artifacts`
	p.db.QueryRowContext(ctx, query).Scan(&stats.TotalSize)

	return stats, nil
}
