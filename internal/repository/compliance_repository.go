package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/securestor/securestor/internal/models"
)

type ComplianceRepository struct {
	db *sql.DB
}

func NewComplianceRepository(db *sql.DB) *ComplianceRepository {
	return &ComplianceRepository{db: db}
}

func (r *ComplianceRepository) Create(audit *models.ComplianceAudit) error {
	query := `
        INSERT INTO compliance_audits (tenant_id, artifact_id, status, score, auditor, 
                                       license_compliance, security_scan, 
                                       code_quality, data_privacy, audited_at, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
        RETURNING audit_id, created_at, updated_at
    `

	if audit.AuditedAt.IsZero() {
		audit.AuditedAt = time.Now()
	}

	err := r.db.QueryRow(query,
		audit.TenantID,
		audit.ArtifactID,
		audit.Status,
		audit.Score,
		audit.Auditor,
		audit.LicenseCompliance,
		audit.SecurityScan,
		audit.CodeQuality,
		audit.DataPrivacy,
		audit.AuditedAt,
	).Scan(&audit.AuditID, &audit.CreatedAt, &audit.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create compliance audit: %w", err)
	}

	return nil
}

func (r *ComplianceRepository) GetByArtifactID(artifactID uuid.UUID) (*models.ComplianceAudit, error) {
	query := `
        SELECT audit_id, tenant_id, artifact_id, status, score, auditor, license_compliance,
               security_scan, code_quality, data_privacy, audited_at, created_at, updated_at
        FROM compliance_audits
        WHERE artifact_id = $1
        ORDER BY audited_at DESC
        LIMIT 1
    `

	audit := &models.ComplianceAudit{}
	err := r.db.QueryRow(query, artifactID).Scan(
		&audit.AuditID,
		&audit.TenantID,
		&audit.ArtifactID,
		&audit.Status,
		&audit.Score,
		&audit.Auditor,
		&audit.LicenseCompliance,
		&audit.SecurityScan,
		&audit.CodeQuality,
		&audit.DataPrivacy,
		&audit.AuditedAt,
		&audit.CreatedAt,
		&audit.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("compliance audit not found")
		}
		return nil, fmt.Errorf("failed to get compliance audit: %w", err)
	}

	audit.LastAudit = audit.AuditedAt.Format(time.RFC3339)
	audit.Checks = map[string]string{
		"licenseCompliance": audit.LicenseCompliance,
		"securityScan":      audit.SecurityScan,
		"codeQuality":       audit.CodeQuality,
		"dataPrivacy":       audit.DataPrivacy,
	}

	return audit, nil
}

func (r *ComplianceRepository) GetHistory(artifactID uuid.UUID) ([]models.ComplianceAudit, error) {
	query := `
        SELECT audit_id, tenant_id, artifact_id, status, score, auditor, license_compliance,
               security_scan, code_quality, data_privacy, audited_at, created_at, updated_at
        FROM compliance_audits
        WHERE artifact_id = $1
        ORDER BY audited_at DESC
    `

	rows, err := r.db.Query(query, artifactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance history: %w", err)
	}
	defer rows.Close()

	var audits []models.ComplianceAudit
	for rows.Next() {
		var audit models.ComplianceAudit
		err := rows.Scan(
			&audit.AuditID,
			&audit.TenantID,
			&audit.ArtifactID,
			&audit.Status,
			&audit.Score,
			&audit.Auditor,
			&audit.LicenseCompliance,
			&audit.SecurityScan,
			&audit.CodeQuality,
			&audit.DataPrivacy,
			&audit.AuditedAt,
			&audit.CreatedAt,
			&audit.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan compliance audit: %w", err)
		}

		audit.LastAudit = audit.AuditedAt.Format(time.RFC3339)
		audits = append(audits, audit)
	}

	return audits, nil
}
